package staticfs

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Option sets staticFS Options.
type Option func(*options)

type options struct {
	indexFile       string        // The default file returned when accessing a directory, e.g., "index.html"
	cacheExpiration time.Duration // File cache expiration time, default is 5 minute
	cacheSize       int           // Maximum number of entries in the file existence cache, default is 1000
	cacheMaxAge     time.Duration // Cache control max-age in seconds, default is 0 (no cache)
	middlewares     []gin.HandlerFunc
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultOptions() *options {
	return &options{
		indexFile:       "index.html",
		cacheExpiration: 5 * time.Minute,
		cacheSize:       1000,
		cacheMaxAge:     0,
	}
}

// WithIndexFile sets the default index file name.
func WithIndexFile(indexFile string) Option {
	return func(o *options) {
		o.indexFile = indexFile
	}
}

// WithCacheExpiration sets the cache expiration time.
func WithCacheExpiration(duration time.Duration) Option {
	return func(o *options) {
		o.cacheExpiration = duration
	}
}

// WithCacheMaxAge sets the Cache-Control max-age header value.
func WithCacheMaxAge(duration time.Duration) Option {
	return func(o *options) {
		o.cacheMaxAge = duration
	}
}

// WithCacheSize sets the maximum number of entries in the file existence cache.
func WithCacheSize(size int) Option {
	return func(o *options) {
		o.cacheSize = size
	}
}

// WithMiddlewares sets middlewares for staticFS.
func WithMiddlewares(middlewares ...gin.HandlerFunc) Option {
	return func(o *options) {
		o.middlewares = middlewares
	}
}

// ------------------------------------------------------------------------------------------

func notFondData(data any) gin.H {
	return gin.H{"code": 404, "msg": "not found", "data": data}
}

// cacheEntry represents a cached file system entry
type cacheEntry struct {
	exists    bool
	isDir     bool
	timestamp time.Time
}

type staticFS struct {
	urlPrefix string // URL path prefix, such as "/assets/"
	diskRoot  string // The root directory on the disk, such as "/var/web/static"
	indexFile string // The default file returned when accessing a directory, e.g., "index.html"

	cacheMaxAge     time.Duration // Cache control max-age in seconds
	fileCache       sync.Map      // Cache for file existence and type
	cacheSize       int           // Maximum cache size
	cacheExpiration time.Duration // Cache entry expiration time
	cacheCount      int           // Current count of cache entries
	cacheMutex      sync.Mutex    // Mutex for cache count operations
}

// checkFileExistence checks if a file exists and caches the result
func (s *staticFS) checkFileExistence(filePath string) (exists bool, isDir bool) {
	// Check cache first
	if value, found := s.fileCache.Load(filePath); found {
		entry := value.(cacheEntry)
		// Check if the cache entry is still valid
		if time.Since(entry.timestamp) < s.cacheExpiration {
			return entry.exists, entry.isDir
		}
	}

	// Not in cache or expired, check file system
	fi, err := os.Stat(filePath)

	// Create new cache entry
	newEntry := cacheEntry{
		exists:    err == nil,
		isDir:     err == nil && fi.IsDir(),
		timestamp: time.Now(),
	}

	// Update cache
	s.fileCache.Store(filePath, newEntry)

	// Check if we need to clean up the cache
	s.cacheMutex.Lock()
	s.cacheCount++
	if s.cacheCount >= s.cacheSize {
		// Reset the cache when it gets too large
		s.fileCache = sync.Map{}
		s.cacheCount = 0
	}
	s.cacheMutex.Unlock()

	return newEntry.exists, newEntry.isDir
}

func (s *staticFS) handler(c *gin.Context) {
	reqPath := c.Request.URL.Path

	// Extract the relative path and build the absolute path in the file system.
	// filepath.FromSlash ensures the path separator is compatible with the current OS.
	relPath := strings.TrimPrefix(reqPath, s.urlPrefix)
	filePath := filepath.Join(s.diskRoot, filepath.FromSlash(relPath))

	// Set cache headers if enabled
	if s.cacheMaxAge > 0 {
		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(s.cacheMaxAge.Seconds())))
	}

	// Check if the file exists using our cached method
	exists, isDir := s.checkFileExistence(filePath)
	if !exists {
		c.JSON(http.StatusNotFound, notFondData(filePath))
		return
	}
	// Case 1: The path exists.
	if exists {
		if isDir {
			// If directory listing is not allowed, try to serve the index file in the directory.
			indexPath := filepath.Join(filePath, s.indexFile)
			// Check if the index file exists using our cached method
			indexExists, _ := s.checkFileExistence(indexPath)
			if indexExists {
				c.File(indexPath)
				return
			}
			c.JSON(http.StatusNotFound, notFondData(indexPath))
		}
		// If it's a file, serve the file directly.
		c.File(filePath)
		return
	}

	// Case 2: The path does not exist.
	// Check if the request path might be a directory with a missing /index.html.
	// For example, a request for /static/about could correspond to /static/about/index.html.
	// A simple check is done by seeing if the path base contains a "." (likely no extension).
	var indexPath string
	if !strings.Contains(path.Base(relPath), ".") {
		indexPath = filepath.Join(filePath, s.indexFile)
		indexExists, _ := s.checkFileExistence(indexPath)
		if indexExists {
			c.File(indexPath)
			return
		}
	}
	c.JSON(http.StatusNotFound, notFondData(indexPath))
}

// StaticFS sets static file server for gin engine.
func StaticFS(r *gin.Engine, urlPrefix string, diskRoot string, opts ...Option) {
	// Ensure URLPrefix starts and ends with a '/', to simplify subsequent path handling.
	if !strings.HasPrefix(urlPrefix, "/") {
		urlPrefix = "/" + urlPrefix
	}
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix = urlPrefix + "/"
	}
	absDiskRoot, err := filepath.Abs(diskRoot)
	if err != nil {
		absDiskRoot = diskRoot
	}

	o := defaultOptions()
	o.apply(opts...)
	sfs := &staticFS{
		urlPrefix: urlPrefix,
		diskRoot:  absDiskRoot,
		indexFile: o.indexFile,

		fileCache:       sync.Map{},
		cacheExpiration: o.cacheExpiration, // Default cache expiration time 5 minute
		cacheSize:       o.cacheSize,       // Default cache size 1000
		cacheMaxAge:     o.cacheMaxAge,     // Default cache max-age 0 (no cache)
		cacheCount:      0,
	}

	if urlPrefix != "/" {
		// When reqPath is missing the trailing '/', redirect to keep the URL consistent.
		// For example, redirect /static to /static/.
		r.GET(strings.TrimSuffix(urlPrefix, "/"), func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, urlPrefix)
		})
	}

	if len(o.middlewares) > 0 {
		handlers := append(o.middlewares, sfs.handler)
		r.GET(urlPrefix+"*path", handlers...)
	} else {
		r.GET(urlPrefix+"*path", sfs.handler)
	}
}
