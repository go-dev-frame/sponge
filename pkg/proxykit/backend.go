package proxykit

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultTransport returns a http.Transport optimized for reverse proxy usage.
func DefaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// MaxIdleConnsPerHost controls the maximum number of idle (keep-alive)
		// connections to keep per-host. Increasing this value can reduce the cost
		// of creating new TCP connections under high concurrency.
		MaxIdleConnsPerHost: 100,
		// MaxIdleConns is the total maximum number of idle connections maintained by the client.
		MaxIdleConns: 200,
		// IdleConnTimeout is the maximum amount of time an idle connection remains open before closing.
		IdleConnTimeout: 90 * time.Second,
		// TLSHandshakeTimeout is the maximum time allowed for the TLS handshake.
		TLSHandshakeTimeout: 10 * time.Second,
		// ExpectContinueTimeout is the maximum time to wait for the server's first response header.
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// Backend encapsulates the backend server information.
type Backend struct {
	URL             *url.URL
	isHealthy       atomic.Bool
	activeConns     atomic.Int64
	proxy           *httputil.ReverseProxy
	stopHealthCheck chan struct{} // Used to stop the health check goroutine
	stopOnce        sync.Once     // Ensures stop is called only once
}

// ParseBackends helper function: converts a list of URL strings into []*Backend
func ParseBackends(prefixPath string, targets []string) ([]*Backend, error) {
	var backends []*Backend
	for _, t := range targets {
		u, err := url.Parse(t)
		if err != nil {
			return nil, err
		}
		backends = append(backends, NewBackend(prefixPath, u))
	}
	return backends, nil
}

// NewBackend creates a new Backend instance.
func NewBackend(prefixPath string, u *url.URL) *Backend {
	proxy := httputil.NewSingleHostReverseProxy(u)

	// Use the optimized Transport
	proxy.Transport = DefaultTransport()

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, strings.TrimSuffix(prefixPath, "/"))
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", u.Host)
	}

	b := &Backend{
		URL:             u,
		proxy:           proxy,
		stopHealthCheck: make(chan struct{}),
	}

	b.isHealthy.Store(true) // initialize as healthy by default
	return b
}

// StopHealthCheck stops the health check goroutine associated with this backend.
func (b *Backend) StopHealthCheck() {
	b.stopOnce.Do(func() {
		close(b.stopHealthCheck)
	})
}

func (b *Backend) SetHealthy(healthy bool) {
	b.isHealthy.Store(healthy)
}

func (b *Backend) IsHealthy() bool {
	return b.isHealthy.Load()
}

func (b *Backend) GetActiveConns() int64 {
	return b.activeConns.Load()
}

func (b *Backend) IncrementActiveConns() {
	b.activeConns.Add(1)
}

func (b *Backend) DecrementActiveConns() {
	b.activeConns.Add(-1)
}
