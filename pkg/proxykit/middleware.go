package proxykit

import (
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// Middleware is a function that takes a http.Handler and returns a http.Handler,
// used to build a middleware chain.
type Middleware func(http.Handler) http.Handler

// Chain links multiple middlewares together to form a single http.Handler.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Start from the last middleware and wrap backwards,
	// so that the first middleware is the outermost layer.
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

type logger struct {
	*zap.Logger
}

func newLogger() *logger {
	var l, _ = zap.NewProduction()
	return &logger{l}
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.Sugar().Infof(format, v...)
}

func (l *logger) Println(v ...interface{}) {
	l.Sugar().Info(v...)
}

var log = newLogger()

var doOnce sync.Once

func SetLogger(l *zap.Logger) {
	doOnce.Do(func() {
		log = &logger{l}
	})
}
