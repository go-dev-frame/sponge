package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// TLSEncryptOption set tlsEncryptOptions.
type TLSEncryptOption func(*tlsEncryptOptions)

type tlsEncryptOptions struct {
	cacheDir       string
	httpAddr       string
	enableRedirect bool
}

func (o *tlsEncryptOptions) apply(opts ...TLSEncryptOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultTLSEncryptOptions() *tlsEncryptOptions {
	return &tlsEncryptOptions{
		cacheDir:       "configs/encrypt_certs",
		enableRedirect: false,
		httpAddr:       ":80",
	}
}

// WithTLSEncryptCacheDir sets the directory to store Let's Encrypt certificates.
func WithTLSEncryptCacheDir(cacheDir string) TLSEncryptOption {
	return func(o *tlsEncryptOptions) {
		o.cacheDir = cacheDir
	}
}

// WithTLSEncryptEnableRedirect enables the HTTP-to-HTTPS redirect service.
// By default, it listens on ":80".
// An optional httpAddr can be provided to specify a different address.
func WithTLSEncryptEnableRedirect(httpAddr ...string) TLSEncryptOption {
	return func(o *tlsEncryptOptions) {
		o.enableRedirect = true
		if len(httpAddr) > 0 && httpAddr[0] != "" {
			o.httpAddr = httpAddr[0]
		}
	}
}

// ------------------------------------------------------------------------------------------

var _ TLSer = (*TLSAutoEncryptConfig)(nil)

type TLSAutoEncryptConfig struct {
	domain         string // The domain to request a certificate for in production mode.
	email          string // Used for Let's Encrypt account registration and important notices.
	cacheDir       string // Directory to store Let's Encrypt certificates.
	httpAddr       string // Listen address for the HTTP redirect service (defaults to :80).
	enableRedirect bool   // Enable HTTP-to-HTTPS redirect service (default: false).

	m              *autocert.Manager // Manages certificates automatically.
	redirectServer *http.Server      // The HTTP redirect server.
}

func NewTLSEAutoEncryptConfig(domain string, email string, opts ...TLSEncryptOption) *TLSAutoEncryptConfig {
	o := defaultTLSEncryptOptions()
	o.apply(opts...)

	return &TLSAutoEncryptConfig{
		domain:         domain,
		email:          email,
		cacheDir:       o.cacheDir,
		httpAddr:       o.httpAddr,
		enableRedirect: o.enableRedirect,
	}
}

func (c *TLSAutoEncryptConfig) Validate() error {
	if c.domain == "" {
		return errors.New("domain must be specified in encrypt mode")
	}
	if c.email == "" {
		return errors.New("email must be specified in encrypt mode")
	}
	if c.cacheDir == "" {
		c.cacheDir = "configs/encrypt_certs"
	}
	if c.httpAddr == "" {
		c.httpAddr = ":80"
	}
	return nil
}

func (c *TLSAutoEncryptConfig) Run(server *http.Server) error {
	m := &autocert.Manager{
		Cache:      autocert.DirCache(c.cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(c.domain),
		Email:      c.email,
	}
	c.m = m
	server.TLSConfig = m.TLSConfig()

	if c.enableRedirect {
		go func() {
			if err := c.redirectHTTP(); err != nil {
				panic(fmt.Sprintf("[redirect http server] %v\n", err))
			}
		}()
	}

	if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[https server] listen and serve TLS error: %v", err)
	}

	if c.enableRedirect {
		_ = c.shutDownRedirectHTTP()
	}

	return nil
}

func (c *TLSAutoEncryptConfig) redirectHTTP() error {
	server := &http.Server{
		Addr:    c.httpAddr,
		Handler: c.m.HTTPHandler(nil), // Handles ACME challenges and redirection.
	}
	c.redirectServer = server

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[redirect http server] listen and serve HTTP error: %v", err)
	}
	return nil
}

func (c *TLSAutoEncryptConfig) shutDownRedirectHTTP() error {
	if c.redirectServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.redirectServer.Shutdown(ctx)
}
