package proxy

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()
	if opts.managerPrefixPath != "/endpoints" {
		t.Errorf("expected default prefix /endpoints, got %s", opts.managerPrefixPath)
	}
	if opts.managerMiddlewares != nil {
		t.Errorf("expected nil manager middlewares by default")
	}
}

func TestWithManagerEndpoints(t *testing.T) {
	mw := func(c *gin.Context) {}
	opts := defaultOptions()
	opts.apply(WithManagerEndpoints("api", mw))

	if opts.managerPrefixPath != "/api" {
		t.Errorf("expected /api, got %s", opts.managerPrefixPath)
	}
	if len(opts.managerMiddlewares) != 1 {
		t.Errorf("expected one middleware")
	}
}

func TestWithManagerEndpointsWithSlash(t *testing.T) {
	opts := defaultOptions()
	opts.apply(WithManagerEndpoints("/api/"))

	if opts.managerPrefixPath != "/api" {
		t.Errorf("expected /api (trimmed), got %s", opts.managerPrefixPath)
	}
}

func TestWithManagerEndpointsEmpty(t *testing.T) {
	opts := defaultOptions()
	opts.apply(WithManagerEndpoints(""))

	// should keep default
	if opts.managerPrefixPath != "/endpoints" {
		t.Errorf("expected default /endpoints, got %s", opts.managerPrefixPath)
	}
}

func TestWithLogger(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	opts := defaultOptions()
	opts.apply(WithLogger(logger))

	if opts.zapLogger == nil {
		t.Errorf("expected logger to be set")
	}
}

func TestDefaultPassOptions(t *testing.T) {
	opts := defaultPassOptions()
	if opts.healthCheckInterval != 5*time.Second {
		t.Errorf("expected default 5s, got %v", opts.healthCheckInterval)
	}
	if opts.healthCheckTimeout != 3*time.Second {
		t.Errorf("expected default 3s, got %v", opts.healthCheckTimeout)
	}
	if opts.balancerType != "round_robin" {
		t.Errorf("expected default round_robin, got %s", opts.balancerType)
	}
}

func TestWithPassBalancer(t *testing.T) {
	opts := defaultPassOptions()
	opts.apply(WithPassBalancer("least_conn"))

	if opts.balancerType != "least_conn" {
		t.Errorf("expected least_conn, got %s", opts.balancerType)
	}
}

func TestWithPassHealthCheckValid(t *testing.T) {
	opts := defaultPassOptions()
	opts.apply(WithPassHealthCheck(2*time.Second, 2*time.Second))

	if opts.healthCheckInterval != 2*time.Second {
		t.Errorf("expected 2s, got %v", opts.healthCheckInterval)
	}
	if opts.healthCheckTimeout != 2*time.Second {
		t.Errorf("expected 2s, got %v", opts.healthCheckTimeout)
	}
}

func TestWithPassHealthCheckInvalid(t *testing.T) {
	opts := defaultPassOptions()
	opts.apply(WithPassHealthCheck(time.Millisecond*500, time.Millisecond*50))
	if opts.healthCheckInterval != 5*time.Second {
		t.Errorf("invalid interval should not be applied")
	}
	if opts.healthCheckTimeout != 3*time.Second {
		t.Errorf("invalid timeout should not be applied")
	}
}

func TestWithPassMiddlewares(t *testing.T) {
	m1 := func(c *gin.Context) {}
	m2 := func(c *gin.Context) {}
	opts := defaultPassOptions()
	opts.apply(WithPassMiddlewares(m1, m2))

	if len(opts.passMiddlewares) != 2 {
		t.Errorf("expected 2 middlewares, got %d", len(opts.passMiddlewares))
	}
}
