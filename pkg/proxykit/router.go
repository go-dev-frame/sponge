package proxykit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// ManagementRequest is for the management API.
type ManagementRequest struct {
	PrefixPath  string            `json:"prefixPath"`
	Targets     []string          `json:"targets"`
	HealthCheck HealthCheckConfig `json:"healthCheck"`
}

// Route holds all components for a specific routing rule.
type Route struct {
	PrefixPath string
	Backends   []*Backend
	Balancer   Balancer
	Proxy      *Proxy
	mu         sync.RWMutex
}

// RouteManager manages all routing rules.
type RouteManager struct {
	routes map[string]*Route
	mu     sync.RWMutex
}

// NewRouteManager creates a new manager.
func NewRouteManager() *RouteManager {
	return &RouteManager{
		routes: make(map[string]*Route),
	}
}

// AddRoute adds a new routing rule and configures its proxy to strip the given prefix.
func (m *RouteManager) AddRoute(prefixPath string, balancer Balancer) (*Route, error) {
	if !strings.HasPrefix(prefixPath, "/") {
		prefixPath = "/" + prefixPath
	}
	if !strings.HasSuffix(prefixPath, "/") {
		prefixPath = prefixPath + "/"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.routes[prefixPath]; exists {
		return nil, fmt.Errorf("route for prefix '%s' already exists", prefixPath)
	}

	proxy, err := NewProxy(balancer)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy for '%s': %w", prefixPath, err)
	}

	route := &Route{
		PrefixPath: prefixPath,
		Backends:   balancer.GetBackends(),
		Balancer:   balancer,
		Proxy:      proxy,
	}

	m.routes[prefixPath] = route
	log.Printf("[Manager] added new route for prefix: %s", prefixPath)
	return route, nil
}

// GetRoute safely retrieves a route.
func (m *RouteManager) GetRoute(prefixPath string) (*Route, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	route, exists := m.routes[prefixPath]
	return route, exists
}

// HandleAddBackends handles the HTTP request to add new backends to a route.
func (m *RouteManager) HandleAddBackends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}
	route, exists := m.GetRoute(req.PrefixPath)
	if !exists {
		http.Error(w, "Not Found: Prefix path does not exist", http.StatusNotFound)
		return
	}
	route.mu.Lock()
	defer route.mu.Unlock()
	addedCount := 0
	for _, targetStr := range req.Targets {
		if containsTarget(route.Backends, targetStr) {
			continue
		}
		targetURL, err := url.Parse(targetStr)
		if err != nil {
			log.Printf("[Manager] error parsing target URL '%s': %v", targetStr, err)
			continue
		}
		backend := NewBackend(req.PrefixPath, targetURL)
		route.Backends = append(route.Backends, backend)
		route.Balancer.AddBackend(backend)
		StartHealthChecks([]*Backend{backend}, req.HealthCheck)
		addedCount++
		log.Printf("[Manager] added backend '%s' to route '%s'", targetStr, route.PrefixPath)
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Backends added successfully", "addedCount": addedCount})
}

// HandleRemoveBackends handles the HTTP request to remove backends from a route.
func (m *RouteManager) HandleRemoveBackends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request: Invalid JSON", http.StatusBadRequest)
		return
	}
	route, exists := m.GetRoute(req.PrefixPath)
	if !exists {
		http.Error(w, "Not Found: Prefix path does not exist", http.StatusNotFound)
		return
	}
	route.mu.Lock()
	defer route.mu.Unlock()
	removedCount := 0
	var updatedBackends []*Backend
	for _, backend := range route.Backends {
		if containsString(req.Targets, backend.URL.String()) {
			backend.StopHealthCheck()
			route.Balancer.RemoveBackend(backend)
			removedCount++
			log.Printf("[Manager] removed backend '%s' from route '%s'", backend.URL.String(), route.PrefixPath)
		} else {
			updatedBackends = append(updatedBackends, backend)
		}
	}
	route.Backends = updatedBackends
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Backends removed successfully", "removedCount": removedCount})
}

// HandleGetBackend handles the HTTP request to get a backend in a route.
func (m *RouteManager) HandleGetBackend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	prefixPath := r.URL.Query().Get("prefixPath")
	target := r.URL.Query().Get("target")
	if prefixPath == "" || target == "" {
		http.Error(w, "Bad Request: 'prefixPath' and 'target' query parameters are required", http.StatusBadRequest)
		return
	}
	route, exists := m.GetRoute(prefixPath)
	if !exists {
		http.Error(w, "Not Found: Prefix path does not exist", http.StatusNotFound)
		return
	}
	route.mu.RLock()
	defer route.mu.RUnlock()
	for _, b := range route.Backends {
		if b.URL.String() == target {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"target": target, "healthy": b.IsHealthy()})
			return
		}
	}
	http.Error(w, "Not Found: Target does not exist in this route", http.StatusNotFound)
}

// HandleListBackends handles the HTTP request to list backends in a route.
func (m *RouteManager) HandleListBackends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	prefixPath := r.URL.Query().Get("prefixPath")
	if prefixPath == "" {
		http.Error(w, "Bad Request: 'prefixPath' query parameter is required", http.StatusBadRequest)
		return
	}
	route, exists := m.GetRoute(prefixPath)
	if !exists {
		http.Error(w, "Not Found: Prefix path does not exist", http.StatusNotFound)
		return
	}
	route.mu.RLock()
	defer route.mu.RUnlock()
	type targetStatus struct {
		Target  string `json:"target"`
		Healthy bool   `json:"healthy"`
	}
	var statuses []targetStatus
	for _, b := range route.Backends {
		statuses = append(statuses, targetStatus{Target: b.URL.String(), Healthy: b.IsHealthy()})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"prefixPath": prefixPath, "targets": statuses})
}

func containsTarget(backends []*Backend, targetStr string) bool {
	for _, b := range backends {
		if b.URL.String() == targetStr {
			return true
		}
	}
	return false
}

func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func AnyRelativePath(prefixPath string) string {
	if !strings.HasPrefix(prefixPath, "/") {
		prefixPath = "/" + prefixPath
	}
	if !strings.HasSuffix(prefixPath, "/") {
		prefixPath = prefixPath + "/"
	}
	return prefixPath + "*path"
}
