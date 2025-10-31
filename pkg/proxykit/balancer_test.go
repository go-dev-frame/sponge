package proxykit

import (
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
)

// newTestBackend is a helper to create backends for balancer tests.
// Note: This uses the *real* NewBackend, so we can test state.
func newTestBackend(t *testing.T, urlStr string, healthy bool, conns int64) *Backend {
	t.Helper()
	u, err := url.Parse(urlStr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	b := NewBackend("", u)
	b.SetHealthy(healthy)
	b.activeConns.Store(conns)
	return b
}

func TestRoundRobin(t *testing.T) {
	t.Parallel()

	b1 := newTestBackend(t, "http://b1.com", true, 0)
	b2 := newTestBackend(t, "http://b2.com", true, 0)
	b3 := newTestBackend(t, "http://b3.com", false, 0) // Unhealthy
	b4 := newTestBackend(t, "http://b4.com", true, 0)

	//rr := NewRoundRobin([]*Backend{b1, b2, b3, b4})

	t.Run("Add/Remove/GetBackends", func(t *testing.T) {
		rr := NewRoundRobin(nil)
		if len(rr.GetBackends()) != 0 {
			t.Error("expected 0 backends")
		}
		b1 := newTestBackend(t, "http://b1.com", true, 0)
		rr.AddBackend(b1)
		if len(rr.GetBackends()) != 1 {
			t.Error("expected 1 backend after add")
		}

		b2 := newTestBackend(t, "http://b2.com", true, 0)
		rr.AddBackend(b2)
		if len(rr.GetBackends()) != 2 {
			t.Error("expected 2 backends after add")
		}

		rr.RemoveBackend(b1)
		backends := rr.GetBackends()
		if len(backends) != 1 {
			t.Error("expected 1 backend after remove")
		}
		if backends[0].URL.String() != "http://b2.com" {
			t.Error("removed wrong backend")
		}

		// Test removing non-existent
		rr.RemoveBackend(b1)
		if len(rr.GetBackends()) != 1 {
			t.Error("removing non-existent backend changed slice")
		}
	})

	t.Run("Next - Cycling", func(t *testing.T) {
		rr := NewRoundRobin([]*Backend{b1, b2, b3, b4})
		expected := []*Backend{b1, b2, b4, b1, b2, b4} // b3 is skipped
		for i, exp := range expected {
			next, err := rr.Next(nil)
			if err != nil {
				t.Fatalf("test %d: Expected no error, got %v", i, err)
			}
			if next != exp {
				t.Fatalf("test %d: Expected backend %s, got %s", i, exp.URL, next.URL)
			}
		}
	})

	t.Run("Next - No Healthy Backends", func(t *testing.T) {
		rr := NewRoundRobin([]*Backend{b3}) // Only unhealthy
		_, err := rr.Next(nil)
		if err != ErrNoHealthyBackends {
			t.Errorf("expected ErrNoHealthyBackends, got %v", err)
		}
	})

	t.Run("Next - Nil Backends", func(t *testing.T) {
		rr := NewRoundRobin(nil)
		_, err := rr.Next(nil)
		if err != ErrNoHealthyBackends {
			t.Errorf("expected ErrNoHealthyBackends, got %v", err)
		}
	})
}

func TestLeastConnections(t *testing.T) {
	t.Parallel()

	b1 := newTestBackend(t, "http://b1.com", true, 5)
	b2 := newTestBackend(t, "http://b2.com", true, 2)
	b3 := newTestBackend(t, "http://b3.com", false, 1) // Unhealthy
	b4 := newTestBackend(t, "http://b4.com", true, 8)

	lc := NewLeastConnections([]*Backend{b1, b2, b3, b4})

	t.Run("Add/Remove/GetBackends", func(t *testing.T) {
		lc := NewLeastConnections(nil)
		b1 := newTestBackend(t, "http://b1.com", true, 0)
		lc.AddBackend(b1)
		if len(lc.GetBackends()) != 1 {
			t.Error("expected 1 backend after add")
		}
		lc.RemoveBackend(b1)
		if len(lc.GetBackends()) != 0 {
			t.Error("expected 0 backends after remove")
		}
	})

	t.Run("Next - Finds Least", func(t *testing.T) {
		// b1=5, b2=2, b3=unhealthy, b4=8
		next, err := lc.Next(nil)
		if err != nil {
			t.Fatal(err)
		}
		if next != b2 {
			t.Fatalf("expected b2 (conns=2), got %s (conns=%d)", next.URL, next.GetActiveConns())
		}

		// Simulate connection
		next.IncrementActiveConns() // b2 now has 3 conns

		next, err = lc.Next(nil)
		if err != nil {
			t.Fatal(err)
		}
		if next != b2 {
			t.Fatalf("expected b2 (conns=3), got %s (conns=%d)", next.URL, next.GetActiveConns())
		}

		b2.IncrementActiveConns() // b2 now 4
		b2.IncrementActiveConns() // b2 now 5

		// Now b1=5, b2=5, b4=8
		next, err = lc.Next(nil)
		if err != nil {
			t.Fatal(err)
		}
		if next != b1 && next != b2 {
			t.Fatalf("expected b1 or b2 (conns=5), got %s (conns=%d)", next.URL, next.GetActiveConns())
		}
	})

	t.Run("Next - No Healthy Backends", func(t *testing.T) {
		lc := NewLeastConnections([]*Backend{b3}) // Only unhealthy
		_, err := lc.Next(nil)
		if err != ErrNoHealthyBackends {
			t.Errorf("expected ErrNoHealthyBackends, got %v", err)
		}
	})
}

func TestIPHash(t *testing.T) {
	t.Parallel()

	b1 := newTestBackend(t, "http://b1.com", true, 0)
	b2 := newTestBackend(t, "http://b2.com", true, 0)
	b3 := newTestBackend(t, "http://b3.com", false, 0) // Unhealthy
	b4 := newTestBackend(t, "http://b4.com", true, 0)

	h := NewIPHash([]*Backend{b1, b2, b3, b4})

	t.Run("Add/Remove/GetBackends", func(t *testing.T) {
		h := NewIPHash(nil)
		b1 := newTestBackend(t, "http://b1.com", true, 0)
		h.AddBackend(b1)
		if len(h.GetBackends()) != 1 {
			t.Error("expected 1 backend after add")
		}
		h.RemoveBackend(b1)
		if len(h.GetBackends()) != 0 {
			t.Error("expected 0 backends after remove")
		}
	})

	t.Run("Next - Consistent Hashing", func(t *testing.T) {
		ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "1.1.1.1", "2.2.2.2"}
		healthy := []*Backend{b1, b2, b4} // b3 is skipped
		n := uint32(len(healthy))

		var expectedBackends []*Backend
		for _, ip := range ips {
			hash := crc32.ChecksumIEEE([]byte(ip))
			expectedBackends = append(expectedBackends, healthy[hash%n])
		}

		for i, ip := range ips {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = ip + ":12345" // Use IP as remote addr
			next, err := h.Next(req)
			if err != nil {
				t.Fatalf("test %d: Error: %v", i, err)
			}
			if next != expectedBackends[i] {
				t.Fatalf("test %d: IP %s. Expected %s, got %s", i, ip, expectedBackends[i].URL, next.URL)
			}
		}
	})

	t.Run("Next - No Healthy Backends", func(t *testing.T) {
		h := NewIPHash([]*Backend{b3}) // Only unhealthy
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.1.1.1:12345"
		_, err := h.Next(req)
		if err != ErrNoHealthyBackends {
			t.Errorf("expected ErrNoHealthyBackends, got %v", err)
		}
	})
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For (Single)",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1"},
			remoteAddr: "2.2.2.2:12345",
			expected:   "1.1.1.1",
		},
		{
			name:       "X-Forwarded-For (Multiple)",
			headers:    map[string]string{"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 3.3.3.3"},
			remoteAddr: "4.4.4.4:12345",
			expected:   "1.1.1.1",
		},
		{
			name:       "No Header, RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "5.5.5.5:12345",
			expected:   "5.5.5.5",
		},
		{
			name:       "No Header, RemoteAddr without port (e.g. invalid)",
			headers:    map[string]string{},
			remoteAddr: "6.6.6.6",
			expected:   "6.6.6.6",
		},
		{
			name:       "Empty X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": ""},
			remoteAddr: "7.7.7.7:12345",
			expected:   "7.7.7.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

// Test concurrent read/write operations on balancers
func TestBalancerConcurrency(t *testing.T) {
	t.Parallel()
	b1 := newTestBackend(t, "http://b1.com", true, 0)
	b2 := newTestBackend(t, "http://b2.com", true, 0)

	balancers := map[string]Balancer{
		"RoundRobin":       NewRoundRobin([]*Backend{b1, b2}),
		"LeastConnections": NewLeastConnections([]*Backend{b1, b2}),
		"IPHash":           NewIPHash([]*Backend{b1, b2}),
	}

	for name, bal := range balancers {
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(4) // 2 readers, 2 writers

			// Reader goroutines
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					req.RemoteAddr = fmt.Sprintf("1.1.1.%d:1234", i)
					if _, err := bal.Next(req); err != nil && err != ErrNoHealthyBackends {
						t.Errorf("reader Next() error: %v", err)
					}
					bal.GetBackends()
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					req.RemoteAddr = fmt.Sprintf("2.2.2.%d:1234", i)
					if _, err := bal.Next(req); err != nil && err != ErrNoHealthyBackends {
						t.Errorf("reader Next() error: %v", err)
					}
					bal.GetBackends()
				}
			}()

			// Writer goroutines
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					b := newTestBackend(t, "http://new-a-"+strconv.Itoa(i), true, 0)
					bal.AddBackend(b)
				}
			}()
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					b := newTestBackend(t, "http://new-b-"+strconv.Itoa(i), true, 0)
					bal.AddBackend(b)
					bal.RemoveBackend(b) // Add and remove
				}
			}()

			wg.Wait()
		})
	}
}
