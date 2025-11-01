package proxy

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/proxykit"
)

// Proxy is a proxy server.
type Proxy struct {
	r       *gin.Engine
	manager *proxykit.RouteManager
}

// New creates a new Proxy instance.
func New(r *gin.Engine, opts ...Option) *Proxy {
	o := defaultOptions()
	o.apply(opts...)

	if o.zapLogger != nil {
		proxykit.SetLogger(o.zapLogger)
	}

	manager := proxykit.NewRouteManager()

	// setup manager endpoints routes
	managerRelativePath := o.managerPrefixPath
	var managerGroup *gin.RouterGroup
	if len(o.managerMiddlewares) > 0 {
		managerGroup = r.Group(managerRelativePath, o.managerMiddlewares...)
	} else {
		managerGroup = r.Group(managerRelativePath)
	}
	{
		managerGroup.POST("/add", gin.WrapF(manager.HandleAddBackends))
		managerGroup.POST("/remove", gin.WrapF(manager.HandleRemoveBackends))
		managerGroup.GET("/list", gin.WrapF(manager.HandleListBackends))
		managerGroup.GET("", gin.WrapF(manager.HandleGetBackend))
	}

	return &Proxy{
		r:       r,
		manager: manager,
	}
}

// Pass registers proxy endpoints to gin engine.
func (p *Proxy) Pass(prefixPath string, endpoints []string, opts ...PassOption) error {
	o := defaultPassOptions()
	o.apply(opts...)

	backends, err := proxykit.ParseBackends(prefixPath, endpoints)
	if err != nil {
		return fmt.Errorf("parse backends error: %v", err)
	}
	proxykit.StartHealthChecks(backends, proxykit.HealthCheckConfig{
		Interval: o.healthCheckInterval,
		Timeout:  o.healthCheckTimeout,
	})

	var balancer proxykit.Balancer
	switch o.balancerType {
	case BalancerRoundRobin:
		balancer = proxykit.NewRoundRobin(backends)
	case BalancerLeastConn:
		balancer = proxykit.NewLeastConnections(backends)
	case BalancerIPHash:
		balancer = proxykit.NewIPHash(backends)
	default:
		return fmt.Errorf("unsupported balancer type: %s", o.balancerType)
	}

	apiRoute, err := p.manager.AddRoute(prefixPath, balancer)
	if err != nil {
		return fmt.Errorf("could not add initial route: %v", err)
	}

	// setup proxy endpoints routes
	proxyRelativePath := proxykit.AnyRelativePath(prefixPath) // /prefixPath/*path
	proxyHandlerFuncs := append(o.passMiddlewares, gin.WrapH(apiRoute.Proxy))
	p.r.Any(proxyRelativePath, proxyHandlerFuncs...)

	return nil
}
