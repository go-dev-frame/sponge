package mysql

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Option set the mysql options.
type Option func(*options)

type options struct {
	isLog         bool
	slowThreshold time.Duration

	maxIdleConns    int
	maxOpenConns    int
	connMaxLifetime time.Duration

	disableForeignKey bool
	enableTrace       bool

	requestIDKey string
	gLog         *zap.Logger
	logLevel     logger.LogLevel

	slavesDsn  []string
	mastersDsn []string

	plugins []gorm.Plugin
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// default settings
func defaultOptions() *options {
	return &options{
		isLog:         false,            // whether to output logs, default off
		slowThreshold: time.Duration(0), // if greater than 0, only print logs that are longer than the threshold, higher priority than isLog

		maxIdleConns:    3,                // set the maximum number of connections in the idle connection pool
		maxOpenConns:    50,               // set the maximum number of open database connections
		connMaxLifetime: 30 * time.Minute, // sets the maximum amount of time a connection can be reused

		disableForeignKey: true,  // disables the use of foreign keys, true is recommended for production environments, enabled by default
		enableTrace:       false, // whether to enable link tracing, default is off

		requestIDKey: "",          // request id key
		gLog:         nil,         // custom logger
		logLevel:     logger.Info, // default logLevel
	}
}

// WithLogging set log sql, If l=nil, the gorm log library will be used
func WithLogging(l *zap.Logger, level ...logger.LogLevel) Option {
	return func(o *options) {
		o.isLog = true
		o.gLog = l
		if len(level) > 0 {
			o.logLevel = level[0]
		} else {
			o.logLevel = logger.Info
		}
	}
}

// WithSlowThreshold Set sql values greater than the threshold
func WithSlowThreshold(d time.Duration) Option {
	return func(o *options) {
		o.slowThreshold = d
	}
}

// WithMaxIdleConns set max idle conns
func WithMaxIdleConns(size int) Option {
	return func(o *options) {
		o.maxIdleConns = size
	}
}

// WithMaxOpenConns set max open conns
func WithMaxOpenConns(size int) Option {
	return func(o *options) {
		o.maxOpenConns = size
	}
}

// WithConnMaxLifetime set conn max lifetime
func WithConnMaxLifetime(t time.Duration) Option {
	return func(o *options) {
		o.connMaxLifetime = t
	}
}

// WithEnableForeignKey use foreign keys
func WithEnableForeignKey() Option {
	return func(o *options) {
		o.disableForeignKey = false
	}
}

// WithEnableTrace use trace
func WithEnableTrace() Option {
	return func(o *options) {
		o.enableTrace = true
	}
}

// WithLogRequestIDKey log request id
func WithLogRequestIDKey(key string) Option {
	return func(o *options) {
		if key == "" {
			key = "request_id"
		}
		o.requestIDKey = key
	}
}

// WithRWSeparation setting read-write separation
func WithRWSeparation(slavesDsn []string, mastersDsn ...string) Option {
	return func(o *options) {
		o.slavesDsn = slavesDsn
		o.mastersDsn = mastersDsn
	}
}

// WithGormPlugin setting gorm plugin
func WithGormPlugin(plugins ...gorm.Plugin) Option {
	return func(o *options) {
		o.plugins = plugins
	}
}
