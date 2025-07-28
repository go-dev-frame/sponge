// Package server is a package that holds the http or grpc service.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/errcode"
	"github.com/go-dev-frame/sponge/pkg/grpc/gtls"
	"github.com/go-dev-frame/sponge/pkg/grpc/interceptor"
	"github.com/go-dev-frame/sponge/pkg/grpc/metrics"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/prof"
	"github.com/go-dev-frame/sponge/pkg/servicerd/registry"

	"github.com/go-dev-frame/sponge/internal/config"
	"github.com/go-dev-frame/sponge/internal/ecode"
	"github.com/go-dev-frame/sponge/internal/service"
)

var _ app.IServer = (*grpcServer)(nil)

var (
	defaultTokenAppID  = "grpc"
	defaultTokenAppKey = "mko09ijn"
)

type grpcServer struct {
	addr   string
	server *grpc.Server
	listen net.Listener

	mux                             *http.ServeMux
	httpServer                      *http.Server
	registerMetricsMuxAndMethodFunc func() error

	iRegistry registry.Registry
	instance  *registry.ServiceInstance
}

// Start grpc service
func (s *grpcServer) Start() error {
	// registration Services
	if s.iRegistry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.iRegistry.Register(ctx, s.instance); err != nil {
			return err
		}
	}

	if s.registerMetricsMuxAndMethodFunc != nil {
		if err := s.registerMetricsMuxAndMethodFunc(); err != nil {
			return err
		}
	}

	// if either pprof or metrics is enabled, the http service will be started
	if s.mux != nil {
		addr := fmt.Sprintf(":%d", config.Get().Grpc.HTTPPort)
		s.httpServer = &http.Server{
			Addr:    addr,
			Handler: s.mux,
		}
		go func() {
			fmt.Printf("http address of pprof and metrics %s\n", addr)
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				panic("listen and serve error: " + err.Error())
			}
		}()
	}

	listen := metrics.NewCustomListener(s.listen, metrics.WithConnectionsLogger(logger.Get()), metrics.WithConnectionsGauge())
	if err := s.server.Serve(listen); err != nil { // block
		return err
	}

	return nil
}

// Stop grpc service
func (s *grpcServer) Stop() error {
	if s.iRegistry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		go func() {
			_ = s.iRegistry.Deregister(ctx, s.instance)
			cancel()
		}()
		<-ctx.Done()
	}

	s.server.GracefulStop()

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

// String comment
func (s *grpcServer) String() string {
	return "grpc service address " + s.addr
}

// secure option
func (s *grpcServer) secureServerOption() grpc.ServerOption {
	switch config.Get().Grpc.ServerSecure.Type {
	case "one-way": // server side certification
		credentials, err := gtls.GetServerTLSCredentials(
			config.Get().Grpc.ServerSecure.CertFile,
			config.Get().Grpc.ServerSecure.KeyFile,
		)
		if err != nil {
			panic(err)
		}
		logger.Info("grpc security type: sever-side certification")
		return grpc.Creds(credentials)

	case "two-way": // both client and server side certification
		credentials, err := gtls.GetServerTLSCredentialsByCA(
			config.Get().Grpc.ServerSecure.CaFile,
			config.Get().Grpc.ServerSecure.CertFile,
			config.Get().Grpc.ServerSecure.KeyFile,
		)
		if err != nil {
			panic(err)
		}
		logger.Info("grpc security type: both client-side and server-side certification")
		return grpc.Creds(credentials)
	}

	logger.Info("grpc security type: insecure")
	return nil
}

// setting up unary server interceptors
func (s *grpcServer) unaryServerOptions() grpc.ServerOption {
	unaryServerInterceptors := []grpc.UnaryServerInterceptor{
		interceptor.UnaryServerRecovery(),
		interceptor.UnaryServerRequestID(),
	}

	// logger interceptor, to print simple messages, replace interceptor.UnaryServerLog with interceptor.UnaryServerSimpleLog
	unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerLog(
		logger.Get(),
		interceptor.WithReplaceGRPCLogger(),
	))

	// token interceptor
	if config.Get().Grpc.EnableToken {
		checkToken := func(appID string, appKey string) error {
			// todo the defaultTokenAppID and defaultTokenAppKey are usually retrieved from the cache or database
			if appID != defaultTokenAppID || appKey != defaultTokenAppKey {
				return status.Errorf(codes.Unauthenticated, "app id or app key checksum failure")
			}
			return nil
		}
		unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerToken(checkToken))
	}

	// jwt token interceptor
	//unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerJwtAuth(
	// // choose a verification method as needed
	//interceptor.WithStandardVerify(standardVerifyFn), // standard verify (default), you can set standardVerifyFn to nil if you don't need it
	//interceptor.WithCustomVerify(customVerifyFn), // custom verify
	// // specify the grpc API to ignore token verification(full path)
	//interceptor.WithAuthIgnoreMethods("/api.user.v1.User/Register", "/api.user.v1.User/Login"),
	//))

	// metrics interceptor
	if config.Get().App.EnableMetrics {
		unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerMetrics())
		s.registerMetricsMuxAndMethodFunc = s.registerMetricsMuxAndMethod()
	}

	// limit interceptor
	if config.Get().App.EnableLimit {
		unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerRateLimit(
		//interceptor.WithWindow(time.Second*5), // default 10s
		//interceptor.WithBucket(200),           // default 100
		//interceptor.WithCPUThreshold(900),     // default 800
		))
	}

	// circuit breaker interceptor
	if config.Get().App.EnableCircuitBreaker {
		unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerCircuitBreaker(
			//interceptor.WithBreakerOption(
			//circuitbreaker.WithSuccess(75),           // default 60
			//circuitbreaker.WithRequest(100),          // default 100
			//circuitbreaker.WithBucket(10),            // default 10
			//circuitbreaker.WithWindow(time.Second*3), // default 3s
			//),
			//interceptor.WithUnaryServerDegradeHandler(handler), // Add degradation processing
			interceptor.WithValidCode( // Add timeout error codes can also trigger circuit breaking
				ecode.StatusInternalServerError.Code(),
				ecode.StatusServiceUnavailable.Code(),
			),
		))
	}

	// trace interceptor
	if config.Get().App.EnableTrace {
		unaryServerInterceptors = append(unaryServerInterceptors, interceptor.UnaryServerTracing())
	}

	return grpc.ChainUnaryInterceptor(unaryServerInterceptors...)
}

// setting up stream server interceptors
func (s *grpcServer) streamServerOptions() grpc.ServerOption {
	streamServerInterceptors := []grpc.StreamServerInterceptor{
		interceptor.StreamServerRecovery(),
		//interceptor.StreamServerRequestID(),
	}

	// logger interceptor, to print simple messages, replace interceptor.StreamServerLog with interceptor.StreamServerSimpleLog
	streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerLog(
		logger.Get(),
		interceptor.WithReplaceGRPCLogger(),
	))

	// token interceptor
	if config.Get().Grpc.EnableToken {
		checkToken := func(appID string, appKey string) error {
			// todo the defaultTokenAppID and defaultTokenAppKey are usually retrieved from the cache or database
			if appID != defaultTokenAppID || appKey != defaultTokenAppKey {
				return status.Errorf(codes.Unauthenticated, "app id or app key checksum failure")
			}
			return nil
		}
		streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerToken(checkToken))
	}

	// jwt token interceptor
	//streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerJwtAuth(
	// // choose a verification method as needed
	//interceptor.WithStandardVerify(standardVerifyFn), // standard verify (default), you can set standardVerifyFn to nil if you don't need it
	//interceptor.WithCustomVerify(customVerifyFn), // custom verify
	// // specify the grpc API to ignore token verification(full path)
	//	interceptor.WithAuthIgnoreMethods("/api.user.v1.User/Register", "/api.user.v1.User/Login"),
	//))

	// metrics interceptor
	if config.Get().App.EnableMetrics {
		streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerMetrics())
	}

	// limit interceptor
	if config.Get().App.EnableLimit {
		streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerRateLimit())
	}

	// circuit breaker interceptor
	if config.Get().App.EnableCircuitBreaker {
		streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerCircuitBreaker(
			// set rpc code for circuit breaker, default already includes codes.Internal and codes.Unavailable
			interceptor.WithValidCode(ecode.StatusInternalServerError.Code()),
			interceptor.WithValidCode(ecode.StatusServiceUnavailable.Code()),
		))
	}

	// trace interceptor
	if config.Get().App.EnableTrace {
		streamServerInterceptors = append(streamServerInterceptors, interceptor.StreamServerTracing())
	}

	return grpc.ChainStreamInterceptor(streamServerInterceptors...)
}

func (s *grpcServer) setOptions() []grpc.ServerOption {
	var options []grpc.ServerOption

	secureOption := s.secureServerOption()
	if secureOption != nil {
		options = append(options, secureOption)
	}

	options = append(options, s.unaryServerOptions())
	options = append(options, s.streamServerOptions())

	return options
}

func (s *grpcServer) registerMetricsMuxAndMethod() func() error {
	return func() error {
		if s.mux == nil {
			s.mux = http.NewServeMux()
		}
		metrics.Register(s.mux, s.server)
		return nil
	}
}

func (s *grpcServer) registerProfMux() {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}
	prof.Register(s.mux, prof.WithIOWaitTime())
}

func (s *grpcServer) addHTTPRouter() {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}
	s.mux.HandleFunc("/codes", errcode.ListGRPCErrCodes) // error codes router

	cfgStr := config.Show()
	s.mux.HandleFunc("/config", errcode.ShowConfig([]byte(cfgStr))) // config router
}

// NewGRPCServer creates a new grpc server
func NewGRPCServer(addr string, opts ...GrpcOption) app.IServer {
	var err error
	o := defaultGrpcOptions()
	o.apply(opts...)
	s := &grpcServer{
		addr:      addr,
		iRegistry: o.iRegistry,
		instance:  o.instance,
	}
	s.addHTTPRouter()
	if config.Get().App.EnableHTTPProfile {
		s.registerProfMux()
	}

	s.listen, err = net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	s.server = grpc.NewServer(s.setOptions()...)
	service.RegisterAllService(s.server) // register for all services
	return s
}
