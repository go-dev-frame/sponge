package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backendServer.Close()

	validEndpoints := []string{backendServer.URL}

	t.Run("SuccessDefaultOptions", func(t *testing.T) {
		r := gin.New()
		p := New(r)
		err := p.Pass("/proxy", validEndpoints)
		require.NoError(t, err)

		routes := r.Routes()
		assert.True(t, routeExists(routes, "POST", "/endpoints/add"))
		assert.True(t, routeExists(routes, "POST", "/endpoints/remove"))
		assert.True(t, routeExists(routes, "GET", "/endpoints/list"))
		assert.True(t, routeExists(routes, "GET", "/endpoints"))
		assert.True(t, routeExists(routes, "GET", "/proxy/*path"))
	})

	t.Run("SuccessWithAllBalancers", func(t *testing.T) {
		balancers := []string{"round_robin", "least_conn", "ip_hash"}
		for _, b := range balancers {
			t.Run(b, func(t *testing.T) {
				r := gin.New()
				p := New(r)
				err := p.Pass("/proxy", validEndpoints, WithPassBalancer(b))
				require.NoError(t, err)
			})
		}
	})

	t.Run("SuccessWithMiddlewares", func(t *testing.T) {
		r := gin.New()
		proxyMw := dummyMiddleware
		managerMw := dummyMiddleware
		p := New(r, WithManagerEndpoints("/admin", managerMw))
		err := p.Pass("/proxy", validEndpoints,
			WithPassMiddlewares(proxyMw),
		)
		require.NoError(t, err)

		routes := r.Routes()
		assert.True(t, routeExists(routes, "POST", "/admin/add"))
		assert.True(t, routeExists(routes, "GET", "/admin/list"))
		assert.True(t, routeExists(routes, "GET", "/admin"))
		assert.True(t, routeExists(routes, "GET", "/proxy/*path"))
		assert.False(t, routeExists(routes, "GET", "/endpoints")) // The default path should not exist
	})

	t.Run("ErrorParseBackends", func(t *testing.T) {
		r := gin.New()
		invalidEndpoints := []string{"http://:invalid"}
		p := New(r)
		err := p.Pass("/proxy", invalidEndpoints)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse backends error")
	})

	t.Run("ErrorUnsupportedBalancer", func(t *testing.T) {
		r := gin.New()
		p := New(r)
		err := p.Pass("/proxy", validEndpoints, WithPassBalancer("unknown_balancer"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported balancer type: unknown_balancer")
	})

	t.Run("SuccessWithEmptyEndpoints", func(t *testing.T) {
		r := gin.New()
		p := New(r)
		err := p.Pass("/proxy", []string{})
		require.NoError(t, err)
	})
}

func dummyMiddleware(c *gin.Context) {
	c.Next()
}

func routeExists(routes gin.RoutesInfo, method, path string) bool {
	for _, r := range routes {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}
