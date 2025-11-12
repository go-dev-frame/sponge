package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CoresConfig = cors.Config

// CoresOption set coresOptions.
type CoresOption func(*coresOptions)

type coresOptions struct {
	newCoresConfig *CoresConfig // if nil, use default config under fields.

	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	maxAge           time.Duration
	allowWildcard    bool
	allowCredentials bool
}

func (o *coresOptions) apply(opts ...CoresOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultCoreOptions() *coresOptions {
	return &coresOptions{
		allowOrigins:     []string{"*"},
		allowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		allowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept", "X-Requested-With", "X-CSRF-Token"},
		exposeHeaders:    []string{"Content-Length", "text/plain", "Authorization", "Content-Type"},
		allowCredentials: true,
		allowWildcard:    true,
		maxAge:           12 * time.Hour,
	}
}

// WithNewConfig set cors config, if nil, use default config under fields.
func WithNewConfig(config *CoresConfig) CoresOption {
	return func(o *coresOptions) {
		o.newCoresConfig = config
	}
}

// WithAllowOrigins set allowOrigins, e.g. "https://yourdomain.com", "https://*.subdomain.com"
func WithAllowOrigins(allowOrigins ...string) CoresOption {
	return func(o *coresOptions) {
		o.allowOrigins = allowOrigins
	}
}

// WithAllowMethods set allowMethods, e.g. "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"
func WithAllowMethods(allowMethods ...string) CoresOption {
	return func(o *coresOptions) {
		o.allowMethods = allowMethods
	}
}

// WithAllowHeaders set allowHeaders, e.g. "Origin", "Authorization", "Content-Type", "Accept"
func WithAllowHeaders(allowHeaders ...string) CoresOption {
	return func(o *coresOptions) {
		o.allowHeaders = allowHeaders
	}
}

// WithExposeHeaders set exposeHeaders
func WithExposeHeaders(exposeHeaders ...string) CoresOption {
	return func(o *coresOptions) {
		o.exposeHeaders = exposeHeaders
	}
}

// WithMaxAge set maxAge
func WithMaxAge(maxAge time.Duration) CoresOption {
	return func(o *coresOptions) {
		o.maxAge = maxAge
	}
}

// WithAllowCredentials set allowCredentials
func WithAllowCredentials(allowCredentials bool) CoresOption {
	return func(o *coresOptions) {
		o.allowCredentials = allowCredentials
	}
}

// WithAllowWildcard set allowWildcard
func WithAllowWildcard(allowWildcard bool) CoresOption {
	return func(o *coresOptions) {
		o.allowWildcard = allowWildcard
	}
}

// Cors cross domain
func Cors(opts ...CoresOption) gin.HandlerFunc {
	o := defaultCoreOptions()
	o.apply(opts...)

	var corsConfig cors.Config
	if o.newCoresConfig != nil {
		corsConfig = *o.newCoresConfig
	} else {
		corsConfig = cors.Config{}
		corsConfig.AllowOrigins = o.allowOrigins
		corsConfig.AllowMethods = o.allowMethods
		corsConfig.AllowHeaders = o.allowHeaders
		corsConfig.ExposeHeaders = o.exposeHeaders
		corsConfig.AllowCredentials = o.allowCredentials
		corsConfig.AllowWildcard = o.allowWildcard
		corsConfig.MaxAge = o.maxAge
	}

	return cors.New(corsConfig)
}
