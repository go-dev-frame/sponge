package proxykit

import (
	"net/http"
	"testing"
)

func TestChain(t *testing.T) {
	t.Parallel()

	// Handler that tracks call order
	var order []string
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "base")
	})

	// Middlewares that track call order
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1-in")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-out")
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2-in")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-out")
		})
	}

	t.Run("No Middlewares", func(t *testing.T) {
		order = []string{} // reset
		handler := Chain(baseHandler)
		handler.ServeHTTP(nil, nil)

		if len(order) != 1 || order[0] != "base" {
			t.Errorf("expected just 'base' to be called, got: %v", order)
		}
	})

	t.Run("Multiple Middlewares", func(t *testing.T) {
		order = []string{} // reset
		handler := Chain(baseHandler, mw1, mw2)
		handler.ServeHTTP(nil, nil)

		expectedOrder := []string{"mw1-in", "mw2-in", "base", "mw2-out", "mw1-out"}
		if len(order) != len(expectedOrder) {
			t.Fatalf("expected %d calls, got %d. Order: %v", len(expectedOrder), len(order), order)
		}
		for i, v := range expectedOrder {
			if order[i] != v {
				t.Errorf("expected order %v, got %v", expectedOrder, order)
				break
			}
		}
	})
}
