package httpsrv

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestTLSSelfSignedConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		opts      []TLSSelfSignedOption
		wantError bool
	}{
		{
			name:      "default configuration",
			opts:      nil,
			wantError: false,
		},
		{
			name: "custom cache directory",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedCacheDir("/tmp/certs"),
			},
			wantError: false,
		},
		{
			name: "custom expiration days",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedExpirationDays(100),
			},
			wantError: false,
		},
		{
			name: "with WAN IPs",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedWanIPs("192.168.1.1", "10.0.0.1"),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewTLSSelfSignedConfig(tt.opts...)
			err := config.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTLSSelfSignedOptions(t *testing.T) {
	tests := []struct {
		name               string
		opts               []TLSSelfSignedOption
		expectedDir        string
		expectedExpiration int
		expectedWanIPs     []string
	}{
		{
			name:               "default options",
			opts:               nil,
			expectedDir:        "configs/self_signed_certs",
			expectedExpiration: 3650,
			expectedWanIPs:     nil,
		},
		{
			name: "custom cache directory",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedCacheDir("/tmp/certs"),
			},
			expectedDir:        "/tmp/certs",
			expectedExpiration: 3650,
			expectedWanIPs:     nil,
		},
		{
			name: "custom expiration",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedExpirationDays(100),
			},
			expectedDir:        "configs/self_signed_certs",
			expectedExpiration: 100,
			expectedWanIPs:     nil,
		},
		{
			name: "with WAN IPs",
			opts: []TLSSelfSignedOption{
				WithTLSSelfSignedWanIPs("192.168.1.1", "10.0.0.1"),
			},
			expectedDir:        "configs/self_signed_certs",
			expectedExpiration: 3650,
			expectedWanIPs:     []string{"192.168.1.1", "10.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultTLSSelfSignedOptions()
			o.apply(tt.opts...)

			if o.cacheDir != tt.expectedDir {
				t.Errorf("cacheDir = %v, want %v", o.cacheDir, tt.expectedDir)
			}
			if o.expirationDays != tt.expectedExpiration {
				t.Errorf("expirationDays = %v, want %v", o.expirationDays, tt.expectedExpiration)
			}
			if len(o.wanIPs) != len(tt.expectedWanIPs) {
				t.Errorf("wanIPs length = %v, want %v", len(o.wanIPs), len(tt.expectedWanIPs))
			}
		})
	}
}

func TestTLSSelfSignedConfig_CertificateGeneration(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	config := NewTLSSelfSignedConfig(
		WithTLSSelfSignedCacheDir(tempDir),
		WithTLSSelfSignedExpirationDays(30),
	)

	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Test certificate generation
	err := config.generateCert()
	if err != nil {
		t.Fatalf("generateCert() failed: %v", err)
	}

	// Check if certificate files were created
	if _, err := os.Stat(config.certFile); os.IsNotExist(err) {
		t.Error("Certificate file was not created")
	}
	if _, err := os.Stat(config.keyFile); os.IsNotExist(err) {
		t.Error("Key file was not created")
	}

	// Test regeneration (should not error)
	err = config.generateCert()
	if err != nil {
		t.Errorf("Regenerate cert failed: %v", err)
	}
}

func TestTLSSelfSignedConfig_Run(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	config := NewTLSSelfSignedConfig(
		WithTLSSelfSignedCacheDir(tempDir),
		WithTLSSelfSignedExpirationDays(1), // Short expiration for testing
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

	// Give server time to start and generate certificates
	time.Sleep(200 * time.Millisecond)

	// Verify certificate files were created
	if _, err := os.Stat(config.certFile); os.IsNotExist(err) {
		t.Error("Certificate file was not created by Run()")
	}
	if _, err := os.Stat(config.keyFile); os.IsNotExist(err) {
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
			t.Errorf("Run() returned unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Run() timed out")
	}
}

func TestTLSSelfSignedConfig_Interface(t *testing.T) {
	var _ TLSer = (*TLSSelfSignedConfig)(nil)

	config := NewTLSSelfSignedConfig()

	// Test interface implementation
	if err := config.Validate(); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestGetLANIP(t *testing.T) {
	ip := getLANIP()
	// We can't reliably test the exact IP, but we can test that it doesn't panic
	// and returns either a valid IP or empty string
	t.Logf("Detected LAN IP: %s", ip)
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", false}, // loopback
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"::1", false}, // IPv6 loopback
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := isValidIP(net.ParseIP(tt.ip))
			if got != tt.want {
				t.Errorf("isValidIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
