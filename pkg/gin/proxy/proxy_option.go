package proxy

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Option set options.
type Option func(*options)

type options struct {
	managerPrefixPath  string // default "/endpoints"
	managerMiddlewares []gin.HandlerFunc
	zapLogger          *zap.Logger
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultOptions() *options {
	return &options{
		managerPrefixPath: "/endpoints",
	}
}

// WithManagerEndpoints sets manager prefix path and middlewares, managerPrefixPath default "/endpoints".
func WithManagerEndpoints(managerPrefixPath string, middlewares ...gin.HandlerFunc) Option {
	return func(o *options) {
		if managerPrefixPath != "" {
			if !strings.HasPrefix(managerPrefixPath, "/") {
				managerPrefixPath = "/" + managerPrefixPath
			}
			managerPrefixPath = strings.TrimSuffix(managerPrefixPath, "/")
			o.managerPrefixPath = managerPrefixPath
		}
		o.managerMiddlewares = middlewares
	}
}

// WithLogger sets logger.
func WithLogger(logger *zap.Logger) Option {
	return func(o *options) {
		o.zapLogger = logger
	}
}

// -------------------------------------------------------------------------------------------

var (
	BalancerRoundRobin = "round_robin"
	BalancerLeastConn  = "least_conn"
	BalancerIPHash     = "ip_hash"
)

// PassOption set passOptions.
type PassOption func(*passOptions)

type passOptions struct {
	healthCheckInterval time.Duration // default 5s
	healthCheckTimeout  time.Duration // default 3s
	balancerType        string        // supported values: "round_robin", "least_conn", "ip_hash", default "round_robin"
	passMiddlewares     []gin.HandlerFunc
}

func (o *passOptions) apply(opts ...PassOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultPassOptions() *passOptions {
	return &passOptions{
		healthCheckInterval: 5 * time.Second,
		healthCheckTimeout:  3 * time.Second,
		balancerType:        BalancerRoundRobin,
	}
}

// WithPassBalancer sets balancer type.
func WithPassBalancer(balancerType string) PassOption {
	return func(o *passOptions) {
		o.balancerType = balancerType
	}
}

// WithPassHealthCheck sets health check interval and timeout.
func WithPassHealthCheck(interval time.Duration, timeout time.Duration) PassOption {
	return func(o *passOptions) {
		if interval >= time.Second {
			o.healthCheckInterval = interval
		}
		if timeout >= time.Millisecond*100 {
			o.healthCheckTimeout = timeout
		}
	}
}

// WithPassMiddlewares sets proxy middlewares.
func WithPassMiddlewares(middlewares ...gin.HandlerFunc) PassOption {
	return func(o *passOptions) {
		o.passMiddlewares = middlewares
	}
}
