package proxykit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// mockBalancer is a mock implementation of the Balancer interface for testing.
type mockBalancer struct {
	backend *Backend
	err     error
}

func (m *mockBalancer) Next(r *http.Request) (*Backend, error) {
	return m.backend, m.err
}
func (m *mockBalancer) GetBackends() []*Backend {
	if m.backend == nil {
		return []*Backend{}
	}
	return []*Backend{m.backend}
}
func (m *mockBalancer) AddBackend(b *Backend)    {}
func (m *mockBalancer) RemoveBackend(b *Backend) {}

func TestNewProxy(t *testing.T) {
	t.Parallel()
	t.Run("Nil Balancer", func(t *testing.T) {
		proxy, err := NewProxy(nil)
		if err == nil {
			t.Error("expected an error when balancer is nil, but got nil")
		}
		if proxy != nil {
			t.Error("expected proxy to be nil when an error occurs")
		}
		if err.Error() != "balancer cannot be nil" {
			t.Errorf("expected error message 'balancer cannot be nil', got '%s'", err.Error())
		}
	})

	t.Run("Valid Balancer", func(t *testing.T) {
		mb := &mockBalancer{}
		proxy, err := NewProxy(mb)
		if err != nil {
			t.Errorf("expected no error, but got: %v", err)
		}
		if proxy == nil {
			t.Fatal("expected proxy to be non-nil")
		}
		if proxy.balancer != mb {
			t.Error("proxy's balancer was not set correctly")
		}
	})
}

func TestProxy_ServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("Balancer Error", func(t *testing.T) {
		// 1. Setup
		mb := &mockBalancer{
			err: errors.New("no backends"),
		}
		proxy, _ := NewProxy(mb)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/foo", nil)
		rr := httptest.NewRecorder()

		// 2. Execute
		proxy.ServeHTTP(rr, req)

		// 3. Assert
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "service not available") {
			t.Errorf("expected body 'service not available', got '%s'", rr.Body.String())
		}
	})

	t.Run("Successful Proxy", func(t *testing.T) {
		// 1. Setup
		// Create a test backend server
		backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Backend-Header", "backend-value")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello from backend"))
		}))
		defer backendServer.Close()

		// Create a real Backend pointing to the test server
		backendURL, _ := url.Parse(backendServer.URL)
		backend := NewBackend("", backendURL)

		// Create a mock balancer that returns this backend
		mb := &mockBalancer{
			backend: backend,
			err:     nil,
		}
		proxy, _ := NewProxy(mb)

		req := httptest.NewRequest(http.MethodGet, "http://proxy.com/foo", nil)
		rr := httptest.NewRecorder()

		// Check active connections before
		if conns := backend.GetActiveConns(); conns != 0 {
			t.Fatalf("expected 0 active connections before request, got %d", conns)
		}

		// 2. Execute
		proxy.ServeHTTP(rr, req)

		// 3. Assert
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if body := rr.Body.String(); body != "hello from backend" {
			t.Errorf("expected body 'hello from backend', got '%s'", body)
		}
		if header := rr.Header().Get("X-Backend-Header"); header != "backend-value" {
			t.Errorf("expected header 'X-Backend-Header' to be 'backend-value', got '%s'", header)
		}

		// Check active connections after
		if conns := backend.GetActiveConns(); conns != 0 {
			t.Fatalf("expected 0 active connections after request, got %d", conns)
		}
	})
}
