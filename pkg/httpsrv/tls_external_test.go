package httpsrv

import (
	"net/http"
	"testing"
)

func TestTLSExternalConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		certFile  string
		keyFile   string
		wantError bool
	}{
		{
			name:      "valid configuration",
			certFile:  "cert.pem",
			keyFile:   "key.pem",
			wantError: false,
		},
		{
			name:      "missing certificate file",
			certFile:  "",
			keyFile:   "key.pem",
			wantError: true,
		},
		{
			name:      "missing key file",
			certFile:  "cert.pem",
			keyFile:   "",
			wantError: true,
		},
		{
			name:      "both files missing",
			certFile:  "",
			keyFile:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewTLSExternalConfig(tt.certFile, tt.keyFile)
			err := config.Validate()

			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTLSExternalConfig_Interface(t *testing.T) {
	var _ TLSer = (*TLSExternalConfig)(nil)

	config := NewTLSExternalConfig("cert.pem", "key.pem")

	// Test that it implements the interface by calling methods
	if err := config.Validate(); err != nil {
		t.Logf("Validate() returned: %v", err)
	}

	// Run will fail in test environment due to missing files, but should not panic
	server := &http.Server{Addr: ":0"}
	err := config.Run(server)
	if err != nil {
		t.Logf("Run() returned expected error in test environment: %v", err)
	}
}
