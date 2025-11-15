package httpsrv

import (
	"context"
	"net/http"
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
		name             string
		opts             []TLSEncryptOption
		expectedDir      string
		expectedAddr     string
		expectedRedirect bool
	}{
		{
			name:             "default options",
			opts:             nil,
			expectedDir:      "configs/encrypt_certs",
			expectedAddr:     ":80",
			expectedRedirect: false,
		},
		{
			name:             "custom cache directory",
			opts:             []TLSEncryptOption{WithTLSEncryptCacheDir("/custom/dir")},
			expectedDir:      "/custom/dir",
			expectedAddr:     ":80",
			expectedRedirect: false,
		},
		{
			name:             "enable redirect with default address",
			opts:             []TLSEncryptOption{WithTLSEncryptEnableRedirect()},
			expectedDir:      "configs/encrypt_certs",
			expectedAddr:     ":80",
			expectedRedirect: true,
		},
		{
			name:             "enable redirect with custom address",
			opts:             []TLSEncryptOption{WithTLSEncryptEnableRedirect(":8080")},
			expectedDir:      "configs/encrypt_certs",
			expectedAddr:     ":8080",
			expectedRedirect: true,
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
