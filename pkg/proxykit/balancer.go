package proxykit

import (
	"errors"
	"hash/crc32"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	ErrNoHealthyBackends = errors.New("no healthy backends available")
)

// Balancer is an interface that must be implemented for all load balancing strategies.
type Balancer interface {
	Next(r *http.Request) (*Backend, error)
	GetBackends() []*Backend
	AddBackend(b *Backend)
	RemoveBackend(b *Backend)
}

// --- RoundRobin ---

type RoundRobin struct {
	backends []*Backend
	next     uint32
	mu       sync.RWMutex
}

func NewRoundRobin(backends []*Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

func (r *RoundRobin) Next(_ *http.Request) (*Backend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	healthy := r.getHealthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	n := atomic.AddUint32(&r.next, 1)
	return healthy[(n-1)%uint32(len(healthy))], nil
}

func (r *RoundRobin) getHealthy() []*Backend {
	var healthy []*Backend
	for _, b := range r.backends {
		if b.IsHealthy() {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

// GetBackends implement the Balancer interface
func (r *RoundRobin) GetBackends() []*Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]*Backend, len(r.backends))
	copy(copied, r.backends)
	return copied
}

// AddBackend implement the Balancer interface
func (r *RoundRobin) AddBackend(b *Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = append(r.backends, b)
}

// RemoveBackend implement the Balancer interface
func (r *RoundRobin) RemoveBackend(b *Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, backend := range r.backends {
		if backend.URL.String() == b.URL.String() {
			r.backends = append(r.backends[:i], r.backends[i+1:]...)
			return
		}
	}
}

// --- LeastConnections ---

type LeastConnections struct {
	backends []*Backend
	mu       sync.RWMutex
}

func NewLeastConnections(backends []*Backend) *LeastConnections {
	return &LeastConnections{backends: backends}
}

func (lc *LeastConnections) Next(_ *http.Request) (*Backend, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	var best *Backend
	minSize := int64(math.MaxInt64)
	found := false

	for _, b := range lc.backends {
		if b.IsHealthy() {
			found = true
			conn := b.GetActiveConns()
			if conn < minSize {
				minSize = conn
				best = b
			}
		}
	}
	if !found || best == nil {
		return nil, ErrNoHealthyBackends
	}
	return best, nil
}

func (lc *LeastConnections) GetBackends() []*Backend {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	copied := make([]*Backend, len(lc.backends))
	copy(copied, lc.backends)
	return copied
}

func (lc *LeastConnections) AddBackend(b *Backend) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.backends = append(lc.backends, b)
}

func (lc *LeastConnections) RemoveBackend(b *Backend) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	for i, backend := range lc.backends {
		if backend.URL.String() == b.URL.String() {
			lc.backends = append(lc.backends[:i], lc.backends[i+1:]...)
			return
		}
	}
}

// --- IPHash ---

type IPHash struct {
	backends []*Backend
	mu       sync.RWMutex
}

func NewIPHash(backends []*Backend) *IPHash {
	return &IPHash{backends: backends}
}

func (h *IPHash) Next(r *http.Request) (*Backend, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	healthy := h.getHealthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	clientIP := getClientIP(r)
	hash := crc32.ChecksumIEEE([]byte(clientIP))
	return healthy[hash%uint32(len(healthy))], nil
}

func (h *IPHash) getHealthy() []*Backend {
	var healthy []*Backend
	for _, b := range h.backends {
		if b.IsHealthy() {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

func (h *IPHash) GetBackends() []*Backend {
	h.mu.RLock()
	defer h.mu.RUnlock()
	copied := make([]*Backend, len(h.backends))
	copy(copied, h.backends)
	return copied
}

func (h *IPHash) AddBackend(b *Backend) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.backends = append(h.backends, b)
}

func (h *IPHash) RemoveBackend(b *Backend) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, backend := range h.backends {
		if backend.URL.String() == b.URL.String() {
			h.backends = append(h.backends[:i], h.backends[i+1:]...)
			return
		}
	}
}

func getClientIP(r *http.Request) string {
	f := r.Header.Get("X-Forwarded-For")
	if f != "" {
		return strings.Split(f, ",")[0]
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
