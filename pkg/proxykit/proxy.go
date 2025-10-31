package proxykit

import (
	"errors"
	"net/http"
)

// Proxy is a reverse proxy that implements the http.Handler interface.
type Proxy struct {
	balancer Balancer
}

// NewProxy creates a new reverse proxy instance.
func NewProxy(balancer Balancer) (*Proxy, error) {
	if balancer == nil {
		return nil, errors.New("balancer cannot be nil")
	}
	return &Proxy{
		balancer: balancer,
	}, nil
}

// ServeHTTP handles incoming HTTP requests and forwards them to the backend
// selected by the load balancer.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Select a healthy backend according to the load balancing strategy.
	backend, err := p.balancer.Next(r)
	if err != nil {
		log.Printf("[Proxy] error selecting backend: %v", err)
		http.Error(w, "service not available", http.StatusServiceUnavailable)
		return
	}

	// Increase the active connection count, and ensure it is decremented
	// when the request completes.
	backend.IncrementActiveConns()
	defer backend.DecrementActiveConns()

	backend.proxy.ServeHTTP(w, r)
}
