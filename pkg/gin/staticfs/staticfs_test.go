package staticfs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Create test directory structure
func setupTestDir(t *testing.T) (string, func()) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "staticfs-test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}

	// Create test file structure
	// /
	// ├── index.html
	// ├── style.css
	// ├── js/
	// │   └── app.js
	// └── subdir/
	//     ├── index.html
	//     └── page.html
	// └── custom-index/
	//     └── home.html
	// └── empty-dir/

	// Create files
	files := map[string]string{
		"index.html":             "<html><body>home</body></html>",
		"style.css":              "body { color: red; }",
		"js/app.js":              "console.log('Hello');",
		"subdir/index.html":      "<html><body>sub dir home</body></html>",
		"subdir/page.html":       "<html><body>sub dir page</body></html>",
		"custom-index/home.html": "<html><body>custom dir home</body></html>",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		// Ensure directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	// Create empty directory
	emptyDir := filepath.Join(tempDir, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create empty directory: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func dummyMiddleware(c *gin.Context) {
	c.Header("X-Dummy", "hit")
	c.Next()
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOptions(t *testing.T) {
	t.Run("DefaultOptions", func(t *testing.T) {
		o := defaultOptions()
		assert.Equal(t, "index.html", o.indexFile)
		assert.Equal(t, 5*time.Minute, o.cacheExpiration)
		assert.Equal(t, 1000, o.cacheSize)
		assert.Equal(t, time.Duration(0), o.cacheMaxAge)
		assert.Empty(t, o.middlewares)
	})

	t.Run("ApplyOptions", func(t *testing.T) {
		o := defaultOptions()
		mw := dummyMiddleware
		o.apply(
			WithIndexFile("home.html"),
			WithCacheExpiration(10*time.Second),
			WithCacheMaxAge(30*time.Second),
			WithCacheSize(50),
			WithMiddlewares(mw),
		)

		assert.Equal(t, "home.html", o.indexFile)
		assert.Equal(t, 10*time.Second, o.cacheExpiration)
		assert.Equal(t, 30*time.Second, o.cacheMaxAge)
		assert.Equal(t, 50, o.cacheSize)
		assert.Len(t, o.middlewares, 1)
	})
}

func TestStaticFS_CheckFileExistence(t *testing.T) {
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()
	absRoot, _ := filepath.Abs(tempDir)

	sfs := &staticFS{
		diskRoot:        absRoot,
		fileCache:       sync.Map{},
		cacheExpiration: 5 * time.Minute,
		cacheSize:       100,
		cacheCount:      0,
	}

	filePath := filepath.Join(absRoot, "style.css")
	dirPath := filepath.Join(absRoot, "js")
	nonExistentPath := filepath.Join(absRoot, "not-exist.txt")

	// 1. Cache miss (file)
	exists, isDir := sfs.checkFileExistence(filePath)
	assert.True(t, exists)
	assert.False(t, isDir)
	assert.Equal(t, 1, sfs.cacheCount)
	// Check if the cache has stored the entry
	val, found := sfs.fileCache.Load(filePath)
	assert.True(t, found)
	assert.True(t, val.(cacheEntry).exists)
	assert.False(t, val.(cacheEntry).isDir)

	// 2. Cache hit (file)
	exists, isDir = sfs.checkFileExistence(filePath)
	assert.True(t, exists)
	assert.False(t, isDir)
	assert.Equal(t, 1, sfs.cacheCount) // count should not increase

	// 3. Cache miss (directory)
	exists, isDir = sfs.checkFileExistence(dirPath)
	assert.True(t, exists)
	assert.True(t, isDir)
	assert.Equal(t, 2, sfs.cacheCount)

	// 4. Cache miss (non-existent)
	exists, isDir = sfs.checkFileExistence(nonExistentPath)
	assert.False(t, exists)
	assert.False(t, isDir)
	assert.Equal(t, 3, sfs.cacheCount)

	// 5. Cache hit (non-existent)
	exists, isDir = sfs.checkFileExistence(nonExistentPath)
	assert.False(t, exists)
	assert.False(t, isDir)
	assert.Equal(t, 3, sfs.cacheCount)

	t.Run("CacheExpiration", func(t *testing.T) {
		sfs = &staticFS{
			diskRoot:        absRoot,
			fileCache:       sync.Map{},
			cacheExpiration: 10 * time.Millisecond,
			cacheSize:       100,
			cacheCount:      0,
		}
		path := filepath.Join(absRoot, "index.html")

		// First call
		sfs.checkFileExistence(path)
		assert.Equal(t, 1, sfs.cacheCount)
		val1, _ := sfs.fileCache.Load(path)

		// Wait for cache expiration
		time.Sleep(15 * time.Millisecond)

		// Second call (should be a miss)
		sfs.checkFileExistence(path)
		assert.Equal(t, 2, sfs.cacheCount) // Counter increases
		val2, _ := sfs.fileCache.Load(path)

		// Timestamps should be different
		assert.NotEqual(t, val1.(cacheEntry).timestamp, val2.(cacheEntry).timestamp)
	})

	t.Run("CacheSizeLimit", func(t *testing.T) {
		sfs = &staticFS{
			diskRoot:        absRoot,
			fileCache:       sync.Map{},
			cacheExpiration: 5 * time.Minute,
			cacheSize:       2, // limit size to 2
			cacheCount:      0,
		}
		path1 := filepath.Join(absRoot, "index.html")
		path2 := filepath.Join(absRoot, "style.css")
		path3 := filepath.Join(absRoot, "js/app.js")

		// Add the first one
		sfs.checkFileExistence(path1)
		assert.Equal(t, 1, sfs.cacheCount)
		_, found1 := sfs.fileCache.Load(path1)
		assert.True(t, found1)

		// Add the second one (triggers reset)
		sfs.checkFileExistence(path2)
		// Cache is reset, counter is zeroed
		assert.Equal(t, 0, sfs.cacheCount)
		// Check if the cache is cleared
		_, found1AfterReset := sfs.fileCache.Load(path1)
		_, found2AfterReset := sfs.fileCache.Load(path2)
		assert.False(t, found1AfterReset)
		assert.False(t, found2AfterReset) // Note: The code logic checks the size and resets *after* storage, so the newly added entry will also be cleared

		// Add the third one (the cache is empty again now)
		sfs.checkFileExistence(path3)
		assert.Equal(t, 1, sfs.cacheCount)
		_, found3 := sfs.fileCache.Load(path3)
		assert.True(t, found3)
	})
}

func TestStaticFS_Handler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Helper function to read the response body
	readBody := func(w *httptest.ResponseRecorder) string {
		body, _ := io.ReadAll(w.Body)
		return string(body)
	}

	t.Run("DefaultBehavior", func(t *testing.T) {
		r := gin.New()
		StaticFS(r, "/static", tempDir)

		// 1. Access file
		w := performRequest(r, "GET", "/static/style.css")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "body { color: red; }", readBody(w))

		// 2. Access nested file
		w = performRequest(r, "GET", "/static/js/app.js")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "console.log('Hello');", readBody(w))

		// 3. Access directory (with trailing slash), should return index.html
		w = performRequest(r, "GET", "/static/subdir/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html><body>sub dir home</body></html>", readBody(w))

		// 4. Access root directory (with trailing slash), should return index.html
		w = performRequest(r, "GET", "/static/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html><body>home</body></html>", readBody(w))

		// 5. Access directory (without trailing slash), should return index.html (implicit index)
		w = performRequest(r, "GET", "/static/subdir")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html><body>sub dir home</body></html>", readBody(w))

		// 6. Access non-existent file
		w = performRequest(r, "GET", "/static/not-found.js")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, readBody(w), "not found")

		// 7. Access empty directory (no index.html, directory listing forbidden)
		w = performRequest(r, "GET", "/static/empty-dir/")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("CustomIndexFile", func(t *testing.T) {
		r := gin.New()
		StaticFS(r, "/static", tempDir, WithIndexFile("home.html"))

		// 1. Access directory that should use custom index
		w := performRequest(r, "GET", "/static/custom-index/")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html><body>custom dir home</body></html>", readBody(w))

		// 2. Access directory that still uses default index (because home.html does not exist)
		w = performRequest(r, "GET", "/static/subdir/")
		assert.Equal(t, http.StatusNotFound, w.Code) // home.html not found, and directory listing is forbidden
	})

	t.Run("CacheMaxAge", func(t *testing.T) {
		r := gin.New()
		StaticFS(r, "/static", tempDir, WithCacheMaxAge(time.Hour))

		// 1. Access existent file
		w := performRequest(r, "GET", "/static/style.css")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "max-age=3600", w.Header().Get("Cache-Control"))

		// 2. Access non-existent file (header should also be set, because the header is set before the check)
		w = performRequest(r, "GET", "/static/not-found.js")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "max-age=3600", w.Header().Get("Cache-Control"))
	})

	t.Run("Middlewares", func(t *testing.T) {
		r := gin.New()
		StaticFS(r, "/static", tempDir, WithMiddlewares(dummyMiddleware))

		w := performRequest(r, "GET", "/static/style.css")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "hit", w.Header().Get("X-Dummy"))
	})

	t.Run("URLPrefixHandling", func(t *testing.T) {
		// 1. Test redirect
		r := gin.New()
		StaticFS(r, "/static", tempDir)
		w := performRequest(r, "GET", "/static")
		assert.Equal(t, http.StatusMovedPermanently, w.Code)
		assert.Equal(t, "/static/", w.Header().Get("Location"))

		// 2. Test root URL ("/"), should not have a redirect
		rRoot := gin.New()
		StaticFS(rRoot, "/", tempDir)
		// Access root directory
		wRoot := performRequest(rRoot, "GET", "/")
		assert.Equal(t, http.StatusOK, wRoot.Code)
		assert.Equal(t, "<html><body>home</body></html>", readBody(wRoot))
		// Access file
		wFile := performRequest(rRoot, "GET", "/style.css")
		assert.Equal(t, http.StatusOK, wFile.Code)
		assert.Equal(t, "body { color: red; }", readBody(wFile))
		// Access subdirectory
		wSub := performRequest(rRoot, "GET", "/subdir/")
		assert.Equal(t, http.StatusOK, wSub.Code)
		assert.Equal(t, "<html><body>sub dir home</body></html>", readBody(wSub))

		// 3. Test prefix (no slash), should be added automatically
		rNoSlash := gin.New()
		StaticFS(rNoSlash, "assets", tempDir)
		// Check redirect
		wRedirect := performRequest(rNoSlash, "GET", "/assets")
		assert.Equal(t, http.StatusMovedPermanently, wRedirect.Code)
		assert.Equal(t, "/assets/", wRedirect.Header().Get("Location"))
		// Check file serving
		wAccess := performRequest(rNoSlash, "GET", "/assets/style.css")
		assert.Equal(t, http.StatusOK, wAccess.Code)
		assert.Equal(t, "body { color: red; }", readBody(wAccess))
	})
}
