package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/servicerd/registry"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"github.com/go-dev-frame/sponge/configs"
	"github.com/go-dev-frame/sponge/internal/config"
)

// need real database to test
func TestHTTPServer(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}
	config.Get().App.EnableMetrics = true
	config.Get().App.EnableTrace = true
	config.Get().App.EnableHTTPProfile = true
	config.Get().App.EnableLimit = true
	config.Get().App.EnableCircuitBreaker = true

	port, _ := utils.GetAvailablePort()
	addr := fmt.Sprintf(":%d", port)
	gin.SetMode(gin.ReleaseMode)

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		server := NewHTTPServer(addr,
			WithHTTPIsProd(true),
			WithHTTPRegistry(&iRegistry{}, &registry.ServiceInstance{}),
		)
		assert.NotNil(t, server)
		cancel()
	})
	utils.SafeRunWithTimeout(time.Second, func(cancel context.CancelFunc) {
		server := NewHTTPServer(addr)
		assert.NotNil(t, server)
		cancel()
	})

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		server := NewHTTPServer_pbExample(addr,
			WithHTTPIsProd(true),
			WithHTTPRegistry(&iRegistry{}, &registry.ServiceInstance{}),
		)
		assert.NotNil(t, server)
		cancel()
	})
	utils.SafeRunWithTimeout(time.Second, func(cancel context.CancelFunc) {
		server := NewHTTPServer_pbExample(addr)
		assert.NotNil(t, server)
		cancel()
	})
}

func TestHTTPServerMock(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		t.Fatal(err)
	}
	config.Get().App.EnableMetrics = true
	config.Get().App.EnableTrace = true
	config.Get().App.EnableHTTPProfile = true
	config.Get().App.EnableLimit = true
	config.Get().App.EnableCircuitBreaker = true

	port, _ := utils.GetAvailablePort()
	addr := fmt.Sprintf(":%d", port)

	o := defaultHTTPOptions()
	if o.isProd {
		gin.SetMode(gin.ReleaseMode)
	}
	s := &httpServer{
		addr:      addr,
		instance:  &registry.ServiceInstance{},
		iRegistry: &iRegistry{},
	}
	s.server = &http.Server{
		Addr:           addr,
		Handler:        http.NewServeMux(),
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		time.Sleep(time.Second * 3)
		_ = s.server.Shutdown(context.Background())
	}()

	str := s.String()
	assert.NotEmpty(t, str)
	err = s.Start()
	assert.NoError(t, err)
	err = s.Stop()
	assert.NoError(t, err)
}

type iRegistry struct{}

func (i *iRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	return nil
}

func (i *iRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	return nil
}
