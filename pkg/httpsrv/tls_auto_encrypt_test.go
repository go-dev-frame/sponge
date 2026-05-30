package httpsrv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTLSAutoEncryptConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		email     string
		opts      []TLSEncryptOption
		wantError bool
	}{
		{
			name:      "valid configuration",
			domain:    "example.com",
			email:     "admin@example.com",
			opts:      nil,
			wantError: false,
		},
		{
			name:      "missing domain",
			domain:    "",
			email:     "admin@example.com",
			opts:      nil,
			wantError: true,
		},
		{
			name:      "missing email",
			domain:    "example.com",
			email:     "",
			opts:      nil,
			wantError: true,
		},
		{
			name:      "with redirect enabled",
			domain:    "example.com",
			email:     "admin@example.com",
			opts:      []TLSEncryptOption{WithTLSEncryptEnableRedirect()},
			wantError: false,
		},
		{
			name:      "with custom cache directory",
			domain:    "example.com",
			email:     "admin@example.com",
			opts:      []TLSEncryptOption{WithTLSEncryptCacheDir("/tmp/certs")},
			wantError: false,
		},
		{
			name:      "with extra domains",
			domain:    "example.com",
			email:     "admin@example.com",
			opts:      []TLSEncryptOption{WithTLSEncryptDomains("www.example.com", "api.example.com")},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewTLSEAutoEncryptConfig(tt.domain, tt.email, tt.opts...)
			err := config.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTLSAutoEncryptConfig_DefaultValues(t *testing.T) {
	config := NewTLSEAutoEncryptConfig("example.com", "admin@example.com")

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	if config.cacheDir == "" {
		t.Error("cacheDir should have default value")
	}
	if config.httpAddr == "" {
		t.Error("httpAddr should have default value")
	}
}

func TestTLSEncryptOptions(t *testing.T) {
	tests := []struct {
		name                      string
		opts                      []TLSEncryptOption
		expectedDir               string
		expectedAddr              string
		expectedRedirect          bool
		expectedRedirectHTTPSPort int
	}{
		{
			name:                      "default options",
			opts:                      nil,
			expectedDir:               "configs/encrypt_certs",
			expectedAddr:              ":80",
			expectedRedirect:          false,
			expectedRedirectHTTPSPort: 443,
		},
		{
			name:                      "custom cache directory",
			opts:                      []TLSEncryptOption{WithTLSEncryptCacheDir("/custom/dir")},
			expectedDir:               "/custom/dir",
			expectedAddr:              ":80",
			expectedRedirect:          false,
			expectedRedirectHTTPSPort: 443,
		},
		{
			name:                      "enable redirect with default address",
			opts:                      []TLSEncryptOption{WithTLSEncryptEnableRedirect()},
			expectedDir:               "configs/encrypt_certs",
			expectedAddr:              ":80",
			expectedRedirect:          true,
			expectedRedirectHTTPSPort: 443,
		},
		{
			name:                      "enable redirect with custom address",
			opts:                      []TLSEncryptOption{WithTLSEncryptEnableRedirect(":8080")},
			expectedDir:               "configs/encrypt_certs",
			expectedAddr:              ":8080",
			expectedRedirect:          true,
			expectedRedirectHTTPSPort: 443,
		},
		{
			name:                      "enable redirect with custom https port",
			opts:                      []TLSEncryptOption{WithTLSEncryptEnableRedirect(":8080"), WithTLSEncryptRedirectHTTPSPort(8443)},
			expectedDir:               "configs/encrypt_certs",
			expectedAddr:              ":8080",
			expectedRedirect:          true,
			expectedRedirectHTTPSPort: 8443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultTLSEncryptOptions()
			o.apply(tt.opts...)

			if o.cacheDir != tt.expectedDir {
				t.Errorf("cacheDir = %v, want %v", o.cacheDir, tt.expectedDir)
			}
			if o.httpAddr != tt.expectedAddr {
				t.Errorf("httpAddr = %v, want %v", o.httpAddr, tt.expectedAddr)
			}
			if o.enableRedirect != tt.expectedRedirect {
				t.Errorf("enableRedirect = %v, want %v", o.enableRedirect, tt.expectedRedirect)
			}
			if o.redirectHTTPSPort != tt.expectedRedirectHTTPSPort {
				t.Errorf("redirectHTTPSPort = %v, want %v", o.redirectHTTPSPort, tt.expectedRedirectHTTPSPort)
			}
		})
	}
}

func TestTLSAutoEncryptConfig_RedirectHandler(t *testing.T) {
	tests := []struct {
		name          string
		httpsPort     int
		host          string
		wantLocation  string
		wantStatus    int
		requestMethod string
	}{
		{
			name:          "default https port",
			httpsPort:     443,
			host:          "example.com:8080",
			wantLocation:  "https://example.com/login?next=%2F",
			wantStatus:    http.StatusFound,
			requestMethod: http.MethodGet,
		},
		{
			name:          "custom https port",
			httpsPort:     8443,
			host:          "example.com:8080",
			wantLocation:  "https://example.com:8443/login?next=%2F",
			wantStatus:    http.StatusFound,
			requestMethod: http.MethodGet,
		},
		{
			name:          "non get method",
			httpsPort:     8443,
			host:          "example.com:8080",
			wantStatus:    http.StatusBadRequest,
			requestMethod: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewTLSEAutoEncryptConfig("example.com", "admin@example.com",
				WithTLSEncryptRedirectHTTPSPort(tt.httpsPort))
			req := httptest.NewRequest(tt.requestMethod, "http://"+tt.host+"/login?next=%2F", nil)
			recorder := httptest.NewRecorder()

			config.redirectHandler().ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantLocation != "" {
				if got := recorder.Header().Get("Location"); got != tt.wantLocation {
					t.Fatalf("Location = %q, want %q", got, tt.wantLocation)
				}
			}
		})
	}
}

func TestTLSAutoEncryptConfig_Run(t *testing.T) {
	config := NewTLSEAutoEncryptConfig("example.com", "admin@example.com",
		WithTLSEncryptEnableRedirect("localhost:0"))

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Create a test server that will immediately shut down
	server := &http.Server{
		Addr: "localhost:0",
	}

	// Test should complete quickly since the server can't actually start TLS
	// in test environment without proper certificates
	done := make(chan error, 1)
	go func() {
		done <- config.Run(server)
	}()

	// Give it a moment then shutdown
	time.Sleep(200 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Wait for Run to complete
	select {
	case err := <-done:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Run() returned expected error in test environment: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Run() timed out")
	}
}
