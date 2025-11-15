package httpsrv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSRemoteAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		opts      []TLSRemoteAPIOption
		wantError bool
	}{
		{
			name:      "valid configuration",
			url:       "https://api.example.com/certs",
			opts:      nil,
			wantError: false,
		},
		{
			name:      "missing URL",
			url:       "",
			opts:      nil,
			wantError: true,
		},
		{
			name: "with custom headers",
			url:  "https://api.example.com/certs",
			opts: []TLSRemoteAPIOption{
				WithTLSRemoteAPIHeaders(map[string]string{
					"Authorization": "Bearer token",
				}),
			},
			wantError: false,
		},
		{
			name: "with custom timeout",
			url:  "https://api.example.com/certs",
			opts: []TLSRemoteAPIOption{
				WithTLSRemoteAPITimeout(10 * time.Second),
			},
			wantError: false,
		},
		{
			name: "with custom cache directory",
			url:  "https://api.example.com/certs",
			opts: []TLSRemoteAPIOption{
				WithTLSRemoteAPICacheDir("/tmp/certs"),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewTLSRemoteAPIConfig(tt.url, tt.opts...)
			err := config.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTLSRemoteAPIOptions(t *testing.T) {
	tests := []struct {
		name            string
		opts            []TLSRemoteAPIOption
		expectedDir     string
		expectedTimeout time.Duration
	}{
		{
			name:            "default options",
			opts:            nil,
			expectedDir:     "configs/remote_api_certs",
			expectedTimeout: 5 * time.Second,
		},
		{
			name: "custom timeout",
			opts: []TLSRemoteAPIOption{
				WithTLSRemoteAPITimeout(10 * time.Second),
			},
			expectedDir:     "configs/remote_api_certs",
			expectedTimeout: 10 * time.Second,
		},
		{
			name: "custom cache directory",
			opts: []TLSRemoteAPIOption{
				WithTLSRemoteAPICacheDir("/tmp/certs"),
			},
			expectedDir:     "/tmp/certs",
			expectedTimeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultTLSRemoteAPIOptions()
			o.apply(tt.opts...)

			if o.cacheDir != tt.expectedDir {
				t.Errorf("cacheDir = %v, want %v", o.cacheDir, tt.expectedDir)
			}
			if o.timeout != tt.expectedTimeout {
				t.Errorf("timeout = %v, want %v", o.timeout, tt.expectedTimeout)
			}
		})
	}
}

func TestTLSRemoteAPIConfig_DownloadFile(t *testing.T) {
	// Create test server that returns mock certificate data
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TLSRemoteAPIResponse{
			CertFile: []byte("mock-cert-data"),
			KeyFile:  []byte("mock-key-data"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	config := NewTLSRemoteAPIConfig(ts.URL)
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	certData, keyData, err := config.downloadFile()
	if err != nil {
		t.Fatalf("downloadFile() failed: %v", err)
	}

	if string(certData) != "mock-cert-data" {
		t.Errorf("certData = %s, want %s", string(certData), "mock-cert-data")
	}
	if string(keyData) != "mock-key-data" {
		t.Errorf("keyData = %s, want %s", string(keyData), "mock-key-data")
	}
}

func TestTLSRemoteAPIConfig_DownloadFile_Error(t *testing.T) {
	// Test server that returns error status
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	config := NewTLSRemoteAPIConfig(ts.URL)
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	_, _, err := config.downloadFile()
	if err == nil {
		t.Error("downloadFile() should fail with server error")
	}
}

func TestTLSRemoteAPIConfig_DownloadFile_InvalidJSON(t *testing.T) {
	// Test server that returns invalid JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json data"))
	}))
	defer ts.Close()

	config := NewTLSRemoteAPIConfig(ts.URL)
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	_, _, err := config.downloadFile()
	if err == nil {
		t.Error("downloadFile() should fail with invalid JSON")
	}
}

func TestTLSRemoteAPIConfig_Run(t *testing.T) {
	// Create test server that returns mock certificate data
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TLSRemoteAPIResponse{
			CertFile: []byte("-----BEGIN CERTIFICATE-----\nMOCK_CERTIFICATE_DATA\n-----END CERTIFICATE-----"),
			KeyFile:  []byte("-----BEGIN PRIVATE KEY-----\nMOCK_PRIVATE_KEY_DATA\n-----END PRIVATE KEY-----"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	config := NewTLSRemoteAPIConfig(
		ts.URL,
		WithTLSRemoteAPICacheDir(tempDir),
		WithTLSRemoteAPITimeout(2*time.Second),
	)

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Create a test server
	server := &http.Server{
		Addr: "localhost:0", // Use port 0 for automatic port assignment
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}),
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- config.Run(server)
	}()

	// Give server time to start and download certificates
	time.Sleep(200 * time.Millisecond)

	// Verify certificate files were created
	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Error("Certificate file was not created by Run()")
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Error("Key file was not created by Run()")
	}

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Server shutdown failed: %v", err)
	}

	// Wait for Run to complete
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Run() returned expected error in test environment: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Run() timed out")
	}
}

func TestTLSRemoteAPIConfig_Interface(t *testing.T) {
	var _ TLSer = (*TLSRemoteAPIConfig)(nil)

	config := NewTLSRemoteAPIConfig("https://api.example.com/certs")

	if err := config.Validate(); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}
