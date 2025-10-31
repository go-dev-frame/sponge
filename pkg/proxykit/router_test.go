package proxykit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Helper to create a mock balancer for router tests
type mockRouterBalancer struct {
	backends []*Backend
	mu       sync.RWMutex
}

func newMockRouterBalancer() *mockRouterBalancer {
	return &mockRouterBalancer{
		backends: []*Backend{},
	}
}
func (m *mockRouterBalancer) Next(r *http.Request) (*Backend, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.backends) == 0 {
		return nil, ErrNoHealthyBackends
	}
	return m.backends[0], nil
}
func (m *mockRouterBalancer) GetBackends() []*Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := make([]*Backend, len(m.backends))
	copy(copied, m.backends)
	return copied
}
func (m *mockRouterBalancer) AddBackend(b *Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backends = append(m.backends, b)
}
func (m *mockRouterBalancer) RemoveBackend(b *Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var updated []*Backend
	for _, backend := range m.backends {
		if backend.URL.String() != b.URL.String() {
			updated = append(updated, backend)
		}
	}
	m.backends = updated
}

func TestNewRouteManager(t *testing.T) {
	t.Parallel()
	m := NewRouteManager()
	if m == nil {
		t.Fatal("NewRouteManager returned nil")
	}
	if m.routes == nil {
		t.Error("routeManager routes map is nil")
	}
}

func TestRouteManager_AddRoute(t *testing.T) {
	t.Parallel()
	m := NewRouteManager()
	b := newMockRouterBalancer()

	t.Run("Valid Route", func(t *testing.T) {
		route, err := m.AddRoute("/api", b)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if route.PrefixPath != "/api/" {
			t.Errorf("expected prefix path '/api/', got '%s'", route.PrefixPath)
		}
		if _, exists := m.routes["/api/"]; !exists {
			t.Error("route was not added to the map")
		}
	})

	t.Run("Prefix Normalization", func(t *testing.T) {
		m := NewRouteManager()
		b := newMockRouterBalancer()

		_, err := m.AddRoute("no-slash", b)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := m.routes["/no-slash/"]; !exists {
			t.Error("expected route '/no-slash/'")
		}

		_, err = m.AddRoute("end-slash/", b)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := m.routes["/end-slash/"]; !exists {
			t.Error("expected route '/end-slash/'")
		}
	})

	t.Run("Duplicate Route", func(t *testing.T) {
		m := NewRouteManager()
		b := newMockRouterBalancer()
		_, err := m.AddRoute("/dup", b)
		if err != nil {
			t.Fatal(err)
		}
		_, err = m.AddRoute("/dup/", b)
		if err == nil {
			t.Fatal("expected an error for duplicate route, got nil")
		}
	})

	t.Run("Nil Balancer", func(t *testing.T) {
		m := NewRouteManager()
		_, err := m.AddRoute("/nil-b", nil)
		if err == nil {
			t.Fatal("expected an error for nil balancer, got nil")
		}
	})
}

func TestRouteManager_GetRoute(t *testing.T) {
	t.Parallel()
	m := NewRouteManager()
	b := newMockRouterBalancer()
	m.AddRoute("/api", b)

	route, exists := m.GetRoute("/api/")
	if !exists {
		t.Fatal("expected route to exist")
	}
	if route == nil {
		t.Fatal("expected route to be non-nil")
	}
	if route.PrefixPath != "/api/" {
		t.Error("got wrong route")
	}

	_, exists = m.GetRoute("/non-existent/")
	if exists {
		t.Fatal("expected route to not exist")
	}
}

func TestRouteManager_Handlers(t *testing.T) {
	// These tests are not parallel because they modify the same manager
	m := NewRouteManager()
	b := newMockRouterBalancer()
	_, err := m.AddRoute("/api", b)
	if err != nil {
		t.Fatalf("failed to add initial route: %v", err)
	}

	t.Run("HandleAddBackends", func(t *testing.T) {
		// 1. Wrong method
		req := httptest.NewRequest(http.MethodGet, "/add", nil)
		rr := httptest.NewRecorder()
		m.HandleAddBackends(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		// 2. Bad JSON
		req = httptest.NewRequest(http.MethodPost, "/add", strings.NewReader(`{invalid`))
		rr = httptest.NewRecorder()
		m.HandleAddBackends(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}

		// 3. Route not found
		body, _ := json.Marshal(ManagementRequest{PrefixPath: "/foo/", Targets: []string{"http://b1.com"}})
		req = httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleAddBackends(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}

		// 4. Success
		body, _ = json.Marshal(ManagementRequest{PrefixPath: "/api/", Targets: []string{"http://b1.com", "http://b2.com"}})
		req = httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleAddBackends(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["addedCount"].(float64) != 2 {
			t.Errorf("expected addedCount=2, got %v", resp["addedCount"])
		}
		route, _ := m.GetRoute("/api/")
		if len(route.Backends) != 2 {
			t.Errorf("expected route to have 2 backends, got %d", len(route.Backends))
		}
		if len(b.GetBackends()) != 2 {
			t.Errorf("expected balancer to have 2 backends, got %d", len(b.GetBackends()))
		}

		// 5. Add duplicate + invalid
		body, _ = json.Marshal(ManagementRequest{PrefixPath: "/api/", Targets: []string{"http://b1.com", "::invalid"}})
		req = httptest.NewRequest(http.MethodPost, "/add", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleAddBackends(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatal(rr.Code)
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["addedCount"].(float64) != 0 {
			t.Errorf("expected addedCount=0 (one duplicate, one invalid), got %v", resp["addedCount"])
		}
		if len(route.Backends) != 2 {
			t.Errorf("expected route to still have 2 backends, got %d", len(route.Backends))
		}
	})

	t.Run("HandleRemoveBackends", func(t *testing.T) {
		// Pre-condition: route /api/ has "http://b1.com" and "http://b2.com"

		// 1. Wrong method
		req := httptest.NewRequest(http.MethodGet, "/remove", nil)
		rr := httptest.NewRecorder()
		m.HandleRemoveBackends(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}

		// 2. Bad JSON
		req = httptest.NewRequest(http.MethodPost, "/remove", strings.NewReader(`{invalid`))
		rr = httptest.NewRecorder()
		m.HandleRemoveBackends(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}

		// 3. Route not found
		body, _ := json.Marshal(ManagementRequest{PrefixPath: "/foo/", Targets: []string{"http://b1.com"}})
		req = httptest.NewRequest(http.MethodPost, "/remove", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleRemoveBackends(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}

		// 4. Remove non-existent
		body, _ = json.Marshal(ManagementRequest{PrefixPath: "/api/", Targets: []string{"http://b3.com"}})
		req = httptest.NewRequest(http.MethodPost, "/remove", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleRemoveBackends(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatal(rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["removedCount"].(float64) != 0 {
			t.Errorf("expected removedCount=0, got %v", resp["removedCount"])
		}
		if len(b.GetBackends()) != 2 {
			t.Errorf("expected balancer to still have 2 backends, got %d", len(b.GetBackends()))
		}

		// 5. Success
		body, _ = json.Marshal(ManagementRequest{PrefixPath: "/api/", Targets: []string{"http://b1.com"}})
		req = httptest.NewRequest(http.MethodPost, "/remove", bytes.NewReader(body))
		rr = httptest.NewRecorder()
		m.HandleRemoveBackends(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["removedCount"].(float64) != 1 {
			t.Errorf("expected removedCount=1, got %v", resp["removedCount"])
		}
		route, _ := m.GetRoute("/api/")
		if len(route.Backends) != 1 {
			t.Errorf("expected route to have 1 backend, got %d", len(route.Backends))
		}
		if len(b.GetBackends()) != 1 {
			t.Errorf("expected balancer to have 1 backend, got %d", len(b.GetBackends()))
		}
		if route.Backends[0].URL.String() != "http://b2.com" {
			t.Error("wrong backend was removed")
		}
	})

	t.Run("HandleListBackends", func(t *testing.T) {
		// Pre-condition: route /api/ has "http://b2.com"

		// 1. Missing param
		req := httptest.NewRequest(http.MethodGet, "/list", nil)
		rr := httptest.NewRecorder()
		m.HandleListBackends(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}

		// 2. Route not found
		req = httptest.NewRequest(http.MethodGet, "/list?prefixPath=/foo/", nil)
		rr = httptest.NewRecorder()
		m.HandleListBackends(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}

		// 3. Success
		req = httptest.NewRequest(http.MethodGet, "/list?prefixPath=/api/", nil)
		rr = httptest.NewRecorder()
		m.HandleListBackends(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		var resp struct {
			PrefixPath string `json:"prefixPath"`
			Targets    []struct {
				Target  string `json:"target"`
				Healthy bool   `json:"healthy"`
			} `json:"targets"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.PrefixPath != "/api/" {
			t.Errorf("expected prefixPath /api/, got %s", resp.PrefixPath)
		}
		if len(resp.Targets) != 1 {
			t.Fatalf("expected 1 target, got %d", len(resp.Targets))
		}
		if resp.Targets[0].Target != "http://b2.com" {
			t.Errorf("expected target 'http://b2.com', got %s", resp.Targets[0].Target)
		}
		if resp.Targets[0].Healthy != true {
			t.Error("expected target to be healthy")
		}
	})

	t.Run("HandleGetBackend", func(t *testing.T) {
		// Pre-condition: route /api/ has "http://b2.com"

		// 1. Missing params
		req := httptest.NewRequest(http.MethodGet, "/get", nil)
		rr := httptest.NewRecorder()
		m.HandleGetBackend(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rr.Code)
		}

		// 2. Route not found
		req = httptest.NewRequest(http.MethodGet, "/get?prefixPath=/foo/&target=http://b2.com", nil)
		rr = httptest.NewRecorder()
		m.HandleGetBackend(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}

		// 3. Target not found
		req = httptest.NewRequest(http.MethodGet, "/get?prefixPath=/api/&target=http://b1.com", nil)
		rr = httptest.NewRecorder()
		m.HandleGetBackend(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rr.Code)
		}

		// 4. Success
		req = httptest.NewRequest(http.MethodGet, "/get?prefixPath=/api/&target=http://b2.com", nil)
		rr = httptest.NewRecorder()
		m.HandleGetBackend(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["target"] != "http://b2.com" {
			t.Errorf("expected target 'http://b2.com', got %s", resp["target"])
		}
		if resp["healthy"] != true {
			t.Error("expected healthy=true")
		}
	})
}

func TestAnyRelativePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in  string
		out string
	}{
		{"api", "/api/*path"},
		{"/api", "/api/*path"},
		{"/api/", "/api/*path"},
		{"", "/*path"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := AnyRelativePath(tt.in); got != tt.out {
				t.Errorf("expected AnyRelativePath('%s') to be '%s', got '%s'", tt.in, tt.out, got)
			}
		})
	}
}
