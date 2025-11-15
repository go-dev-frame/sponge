package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/httpsrv"
	"github.com/go-dev-frame/sponge/pkg/servicerd/registry"

	"github.com/go-dev-frame/sponge/internal/config"
	"github.com/go-dev-frame/sponge/internal/routers"
)

var _ app.IServer = (*httpServer)(nil)

type httpServer struct {
	addr   string
	server *httpsrv.Server

	instance  *registry.ServiceInstance
	iRegistry registry.Registry
}

// Start http service
func (s *httpServer) Start() error {
	if s.iRegistry != nil {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second) //nolint
		if err := s.iRegistry.Register(ctx, s.instance); err != nil {
			return err
		}
	}

	if err := s.server.Run(); err != nil {
		return fmt.Errorf("run %s service error: %v", s.server.Scheme(), err)
	}
	return nil
}

// Stop http service
func (s *httpServer) Stop() error {
	if s.iRegistry != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		go func() {
			_ = s.iRegistry.Deregister(ctx, s.instance)
			cancel()
		}()
		<-ctx.Done()
	}

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second) //nolint
	return s.server.Shutdown(ctx)
}

// String comment
func (s *httpServer) String() string {
	return s.server.Scheme() + " service address is " + s.addr
}

func newServer(server *http.Server, tls config.TLS) *httpsrv.Server {
	var c *httpsrv.Server
	switch httpsrv.Mode(tls.EnableMode) {
	case httpsrv.ModeTLSSelfSigned:
		c = httpsrv.New(server, httpsrv.NewTLSSelfSignedConfig())
	case httpsrv.ModeTLSEncrypt:
		c = httpsrv.New(server,
			httpsrv.NewTLSEAutoEncryptConfig(
				tls.Domain,
				tls.Email,
				// enable http redirect to https, port 80 to 443, default is false
				//httpsrv.WithTLSEncryptEnableRedirect(),
			),
		)
	case httpsrv.ModeTLSExternal:
		c = httpsrv.New(server, httpsrv.NewTLSExternalConfig(tls.CertFile, tls.KeyFile))
	default:
		c = httpsrv.New(server) // default is http, no tls
	}
	return c
}

// NewHTTPServer creates a new http server
func NewHTTPServer(addr string, opts ...HTTPOption) app.IServer {
	o := defaultHTTPOptions()
	o.apply(opts...)

	if o.isProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := routers.NewRouter()
	server := &http.Server{
		Addr:    addr,
		Handler: router,
		//ReadTimeout:    time.Second*30,
		//WriteTimeout:   time.Second*60,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20,
	}

	return &httpServer{
		addr:      addr,
		server:    newServer(server, o.tls),
		iRegistry: o.iRegistry,
		instance:  o.instance,
	}
}

// delete the templates code start

// NewHTTPServer_pbExample creates a new web server
func NewHTTPServer_pbExample(addr string, opts ...HTTPOption) app.IServer { //nolint
	o := defaultHTTPOptions()
	o.apply(opts...)

	if o.isProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := routers.NewRouter_pbExample()
	server := &http.Server{
		Addr:    addr,
		Handler: router,
		//ReadTimeout:    time.Second*30,
		//WriteTimeout:   time.Second*60,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 << 20,
	}

	return &httpServer{
		addr:      addr,
		server:    newServer(server, o.tls),
		iRegistry: o.iRegistry,
		instance:  o.instance,
	}
}

// delete the templates code end
