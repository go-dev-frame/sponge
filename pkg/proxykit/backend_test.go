package proxykit

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestDefaultTransport(t *testing.T) {
	t.Parallel()
	transport := DefaultTransport()
	if transport == nil {
		t.Fatal("defaultTransport() returned nil")
	}
	if transport.MaxIdleConnsPerHost != 100 {
		t.Errorf("expected MaxIdleConnsPerHost=100, got %d", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("expected IdleConnTimeout=90s, got %v", transport.IdleConnTimeout)
	}
}

func TestParseBackends(t *testing.T) {
	t.Parallel()
	t.Run("Valid Targets", func(t *testing.T) {
		targets := []string{"http://server1.com", "https://server2.com/path"}
		backends, err := ParseBackends("/api", targets)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(backends) != 2 {
			t.Fatalf("expected 2 backends, got %d", len(backends))
		}
		if backends[0].URL.String() != "http://server1.com" {
			t.Errorf("expected backend[0] URL 'http://server1.com', got %s", backends[0].URL.String())
		}
		if backends[1].URL.String() != "https://server2.com/path" {
			t.Errorf("expected backend[1] URL 'https://server2.com/path', got %s", backends[1].URL.String())
		}
	})

	t.Run("Invalid Target", func(t *testing.T) {
		targets := []string{"http://good.com", "::not-a-url"}
		_, err := ParseBackends("/api", targets)
		if err == nil {
			t.Fatal("expected an error for invalid URL, got nil")
		}
		if _, ok := err.(*url.Error); !ok {
			t.Errorf("expected error to be of type *url.Error, got %T", err)
		}
	})
}

func TestNewBackend_Director(t *testing.T) {
	t.Parallel()
	backendURL, _ := url.Parse("http://backend.internal:8080")
	prefixPath := "/api/v1/"

	b := NewBackend(prefixPath, backendURL)

	if b == nil {
		t.Fatal("newBackend returned nil")
	}
	if b.proxy == nil {
		t.Fatal("backend proxy is nil")
	}
	if !b.IsHealthy() {
		t.Error("backend should be healthy by default")
	}
	if b.stopHealthCheck == nil {
		t.Error("stopHealthCheck channel is nil")
	}

	// Test the director
	req := httptest.NewRequest(http.MethodGet, "https://proxy.public.com/api/v1/users/123", nil)
	req.Host = "proxy.public.com"

	b.proxy.Director(req)

	// 1. Check host and scheme (from originalDirector)
	if req.URL.Scheme != "http" {
		t.Errorf("expected scheme 'http', got '%s'", req.URL.Scheme)
	}
	if req.URL.Host != "backend.internal:8080" {
		t.Errorf("expected host 'backend.internal:8080', got '%s'", req.URL.Host)
	}

	// 2. Check path stripping
	if req.URL.Path != "/users/123" {
		t.Errorf("expected path '/users/123', got '%s'", req.URL.Path)
	}

	// 3. Check headers
	if h := req.Header.Get("X-Forwarded-Host"); h != "proxy.public.com" {
		t.Errorf("expected X-Forwarded-Host 'proxy.public.com', got '%s'", h)
	}
	if h := req.Header.Get("X-Origin-Host"); h != "backend.internal:8080" {
		t.Errorf("expected X-Origin-Host 'backend.internal:8080', got '%s'", h)
	}
}

func TestBackend_Health(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("http://test.com")
	b := NewBackend("", u)

	if !b.IsHealthy() {
		t.Error("expected initial state to be healthy")
	}

	b.SetHealthy(false)
	if b.IsHealthy() {
		t.Error("expected state to be unhealthy after SetHealthy(false)")
	}

	b.SetHealthy(true)
	if !b.IsHealthy() {
		t.Error("expected state to be healthy after SetHealthy(true)")
	}
}

func TestBackend_Health_Concurrent(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("http://test.com")
	b := NewBackend("", u)
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Sets to false
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b.SetHealthy(false)
		}
	}()

	// Goroutine 2: Sets to true
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			b.SetHealthy(true)
		}
	}()

	wg.Wait()
	// We don't know the final state, but we know it shouldn't have panicked.
	// We can read the value to ensure the race detector is happy.
	_ = b.IsHealthy()
}

func TestBackend_Conns(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("http://test.com")
	b := NewBackend("", u)

	if b.GetActiveConns() != 0 {
		t.Error("expected initial connections to be 0")
	}

	b.IncrementActiveConns()
	if b.GetActiveConns() != 1 {
		t.Error("expected connections to be 1 after increment")
	}

	b.IncrementActiveConns()
	if b.GetActiveConns() != 2 {
		t.Error("expected connections to be 2 after increment")
	}

	b.DecrementActiveConns()
	if b.GetActiveConns() != 1 {
		t.Error("expected connections to be 1 after decrement")
	}

	b.DecrementActiveConns()
	if b.GetActiveConns() != 0 {
		t.Error("expected connections to be 0 after decrement")
	}
}

func TestBackend_Conns_Concurrent(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("http://test.com")
	b := NewBackend("", u)

	var wg sync.WaitGroup
	numGoroutines := 100
	numOps := 1000
	wg.Add(numGoroutines * 2) // For increments and decrements

	// Increment
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				b.IncrementActiveConns()
			}
		}()
	}

	// Decrement
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				b.DecrementActiveConns()
			}
		}()
	}

	wg.Wait()

	expected := int64(0)
	if b.GetActiveConns() != expected {
		t.Errorf("expected final connections to be %d, got %d", expected, b.GetActiveConns())
	}
}

func TestBackend_StopHealthCheck(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("http://test.com")
	b := NewBackend("", u)

	// Check that it's open
	select {
	case <-b.stopHealthCheck:
		t.Fatal("channel should be open, but was closed")
	default:
		// all good
	}

	b.StopHealthCheck()

	// Check that it's closed
	select {
	case <-b.stopHealthCheck:
		// all good
	default:
		t.Fatal("channel should be closed, but was open")
	}

	// Calling it again should not panic (thanks to sync.Once)
	b.StopHealthCheck()
}
