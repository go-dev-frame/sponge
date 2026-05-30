package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// TLSEncryptOption set tlsEncryptOptions.
type TLSEncryptOption func(*tlsEncryptOptions)

type tlsEncryptOptions struct {
	cacheDir          string
	domains           []string
	httpAddr          string
	enableRedirect    bool
	redirectHTTPSPort int
}

func (o *tlsEncryptOptions) apply(opts ...TLSEncryptOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultTLSEncryptOptions() *tlsEncryptOptions {
	return &tlsEncryptOptions{
		cacheDir:          "configs/encrypt_certs",
		enableRedirect:    false,
		httpAddr:          ":80",
		redirectHTTPSPort: 443,
	}
}

// WithTLSEncryptCacheDir sets the directory to store Let's Encrypt certificates.
func WithTLSEncryptCacheDir(cacheDir string) TLSEncryptOption {
	return func(o *tlsEncryptOptions) {
		o.cacheDir = cacheDir
	}
}

// WithTLSEncryptDomains adds domains to the Let's Encrypt host whitelist.
func WithTLSEncryptDomains(domains ...string) TLSEncryptOption {
	return func(o *tlsEncryptOptions) {
		o.domains = append(o.domains, domains...)
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

// WithTLSEncryptRedirectHTTPSPort sets the HTTPS port used in HTTP-to-HTTPS redirects.
// When httpsPort is 443 or less than 1, redirects use the default HTTPS port without
// appending a port to the target host.
func WithTLSEncryptRedirectHTTPSPort(httpsPort int) TLSEncryptOption {
	return func(o *tlsEncryptOptions) {
		o.redirectHTTPSPort = httpsPort
	}
}

// ------------------------------------------------------------------------------------------

var _ TLSer = (*TLSAutoEncryptConfig)(nil)

type TLSAutoEncryptConfig struct {
	domain            string   // The primary domain to request a certificate for in production mode.
	domains           []string // Domain whitelist for Let's Encrypt certificates.
	email             string   // Used for Let's Encrypt account registration and important notices.
	cacheDir          string   // Directory to store Let's Encrypt certificates.
	httpAddr          string   // Listen address for the HTTP redirect service (defaults to :80).
	enableRedirect    bool     // Enable HTTP-to-HTTPS redirect service (default: false).
	redirectHTTPSPort int      // HTTPS port used in redirect targets (defaults to 443).

	m              *autocert.Manager // Manages certificates automatically.
	redirectServer *http.Server      // The HTTP redirect server.
}

func NewTLSEAutoEncryptConfig(domain string, email string, opts ...TLSEncryptOption) *TLSAutoEncryptConfig {
	o := defaultTLSEncryptOptions()
	o.apply(opts...)

	return &TLSAutoEncryptConfig{
		domain:            strings.TrimSpace(domain),
		domains:           normalizeDomains(append([]string{domain}, o.domains...)),
		email:             email,
		cacheDir:          o.cacheDir,
		httpAddr:          o.httpAddr,
		enableRedirect:    o.enableRedirect,
		redirectHTTPSPort: o.redirectHTTPSPort,
	}
}

func (c *TLSAutoEncryptConfig) Validate() error {
	c.domains = normalizeDomains(append([]string{c.domain}, c.domains...))
	if len(c.domains) == 0 {
		return errors.New("domain must be specified in encrypt mode")
	}
	c.domain = c.domains[0]
	if c.email == "" {
		return errors.New("email must be specified in encrypt mode")
	}
	if c.cacheDir == "" {
		c.cacheDir = "configs/encrypt_certs"
	}
	if c.httpAddr == "" {
		c.httpAddr = ":80"
	}
	if c.redirectHTTPSPort < 1 {
		c.redirectHTTPSPort = 443
	}
	return nil
}

func (c *TLSAutoEncryptConfig) Run(server *http.Server) error {
	m := &autocert.Manager{
		Cache:      autocert.DirCache(c.cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(c.domains...),
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
		Handler: c.m.HTTPHandler(c.redirectHandler()), // Handles ACME challenges and redirection.
	}
	c.redirectServer = server

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[redirect http server] listen and serve HTTP error: %v", err)
	}
	return nil
}

func (c *TLSAutoEncryptConfig) redirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Use HTTPS", http.StatusBadRequest)
			return
		}

		host := strings.TrimSpace(r.Host)
		if parsedHost, _, err := net.SplitHostPort(host); err == nil && parsedHost != "" {
			host = parsedHost
		}
		if host == "" && r.URL != nil {
			host = r.URL.Host
		}

		targetHost := host
		if host != "" && c.redirectHTTPSPort > 0 && c.redirectHTTPSPort != 443 {
			targetHost = net.JoinHostPort(host, strconv.Itoa(c.redirectHTTPSPort))
		}

		http.Redirect(w, r, "https://"+targetHost+r.URL.RequestURI(), http.StatusFound)
	})
}

func (c *TLSAutoEncryptConfig) shutDownRedirectHTTP() error {
	if c.redirectServer == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.redirectServer.Shutdown(ctx)
}

func normalizeDomains(domains []string) []string {
	seen := map[string]struct{}{}
	filtered := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		filtered = append(filtered, domain)
	}
	return filtered
}
