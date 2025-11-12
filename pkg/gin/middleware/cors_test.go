package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCoreOptions(t *testing.T) {
	opts := defaultCoreOptions()

	assert.NotNil(t, opts)
	assert.Equal(t, []string{"*"}, opts.allowOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}, opts.allowMethods)
	assert.Equal(t, []string{"Origin", "Authorization", "Content-Type", "Accept", "X-Requested-With", "X-CSRF-Token"}, opts.allowHeaders)
	assert.Equal(t, []string{"Content-Length", "text/plain", "Authorization", "Content-Type"}, opts.exposeHeaders)
	assert.True(t, opts.allowCredentials)
	assert.True(t, opts.allowWildcard)
	assert.Equal(t, 12*time.Hour, opts.maxAge)
}

func TestCorsWithDefaultOptions(t *testing.T) {
	router := gin.New()
	router.Use(Cors())
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "test", w.Body.String())
}

func TestCorsWithCustomConfig(t *testing.T) {
	customConfig := &CoresConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           1 * time.Hour,
		AllowWildcard:    true,
	}

	router := gin.New()
	router.Use(Cors(WithNewConfig(customConfig)))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	router.ServeHTTP(w, req)

	// CORS middleware returns 204 for preflight requests
	assert.Equal(t, 204, w.Code)
}

func TestCorsWithAllowOrigins(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowOrigins("https://example.com", "https://test.com")))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	router.ServeHTTP(w, req)

	// CORS middleware returns 204 for preflight requests
	assert.Equal(t, 204, w.Code)
}

func TestCorsWithAllowMethods(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowMethods("GET", "POST")))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithAllowHeaders(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowHeaders("X-Custom-Header", "X-Another-Header")))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithExposeHeaders(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithExposeHeaders("X-Custom-Header", "X-Another-Header")))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithMaxAge(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithMaxAge(30 * time.Minute)))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithAllowCredentials(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowCredentials(false)))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithAllowWildcard(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowCredentials(false)))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsWithMultipleOptions(t *testing.T) {
	router := gin.New()
	router.Use(Cors(
		WithAllowOrigins("https://example.com"),
		WithAllowMethods("GET", "POST"),
		WithAllowHeaders("X-Custom-Header"),
		WithExposeHeaders("X-Exposed-Header"),
		WithMaxAge(30*time.Minute),
		WithAllowCredentials(false),
	))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	router.ServeHTTP(w, req)

	// CORS middleware returns 204 for preflight requests
	assert.Equal(t, 204, w.Code)
}

func TestCorsWithNilConfig(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithNewConfig(nil)))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestCorsOptionsApply(t *testing.T) {
	opts := &coresOptions{}
	opt1 := WithAllowOrigins("https://example.com")
	opt2 := WithAllowMethods("GET")

	opts.apply(opt1, opt2)

	assert.Equal(t, []string{"https://example.com"}, opts.allowOrigins)
	assert.Equal(t, []string{"GET"}, opts.allowMethods)
}

func TestCorsPreflightRequest(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowOrigins("https://example.com")))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	router.ServeHTTP(w, req)

	// CORS middleware returns 204 for preflight requests
	assert.Equal(t, 204, w.Code)
}

func TestCorsActualRequest(t *testing.T) {
	router := gin.New()
	router.Use(Cors(WithAllowOrigins("https://example.com")))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "test", w.Body.String())
}
