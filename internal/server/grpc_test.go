package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/go-dev-frame/sponge/pkg/grpc/gtls/certfile"
	"github.com/go-dev-frame/sponge/pkg/servicerd/registry"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"github.com/go-dev-frame/sponge/configs"
	"github.com/go-dev-frame/sponge/internal/config"
)

func TestGRPCServer(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}

	config.Get().App.EnableMetrics = true
	config.Get().App.EnableTrace = true
	config.Get().App.EnableHTTPProfile = true
	config.Get().App.EnableLimit = true
	config.Get().App.EnableCircuitBreaker = true
	config.Get().Grpc.EnableToken = true

	port, _ := utils.GetAvailablePort()
	addr := fmt.Sprintf(":%d", port)
	instance := registry.NewServiceInstance("foo", "bar", []string{"grpc://127.0.0.1:8282"})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		server := NewGRPCServer(addr,
			WithGrpcRegistry(nil, instance),
		)
		assert.NotNil(t, server)
		cancel()
	})
}

func TestGRPCServerMock(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}
	config.Get().App.EnableMetrics = true
	config.Get().App.EnableTrace = true
	config.Get().App.EnableHTTPProfile = true
	config.Get().App.EnableLimit = true
	config.Get().App.EnableCircuitBreaker = true
	config.Get().Grpc.EnableToken = true

	port, _ := utils.GetAvailablePort()
	addr := fmt.Sprintf(":%d", port)
	instance := registry.NewServiceInstance("foo", "bar", []string{"grpc://127.0.0.1:8282"})

	o := defaultGrpcOptions()
	o.apply(WithGrpcRegistry(&gRegistry{}, instance))

	s := &grpcServer{
		addr:      addr,
		iRegistry: o.iRegistry,
		instance:  o.instance,
	}

	s.listen, err = net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	s.server = grpc.NewServer(s.unaryServerOptions(), s.streamServerOptions())

	go func() {
		time.Sleep(time.Second * 3)
		s.server.Stop()
	}()

	str := s.String()
	assert.NotEmpty(t, str)
	err = s.Start()
	assert.NoError(t, err)
	err = s.Stop()
	assert.NoError(t, err)
}

type gRegistry struct{}

func (g gRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	return nil
}

func (g gRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	return nil
}

func Test_grpcServer_setOptions(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := &grpcServer{}

	defer func() {
		recover()
	}()

	config.Get().Grpc.ServerSecure.Type = ""
	opt := s.secureServerOption()
	assert.Equal(t, nil, opt)

	config.Get().Grpc.ServerSecure.Type = "one-way"
	config.Get().Grpc.ServerSecure.CertFile = certfile.Path("one-way/server.crt")
	config.Get().Grpc.ServerSecure.KeyFile = certfile.Path("one-way/server.key")
	opt = s.secureServerOption()
	assert.NotNil(t, opt)

	config.Get().Grpc.ServerSecure.Type = "two-way"
	config.Get().Grpc.ServerSecure.CaFile = certfile.Path("two-way/ca.pem")
	config.Get().Grpc.ServerSecure.CertFile = certfile.Path("two-way/server/server.pem")
	config.Get().Grpc.ServerSecure.KeyFile = certfile.Path("two-way/server/server.key")
	opt = s.secureServerOption()
	assert.NotNil(t, opt)

	fmt.Println(certfile.Path("one-way/server.crt"), certfile.Path("one-way/server.key"))
}
