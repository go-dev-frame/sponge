package httpsrv

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestServer_New(t *testing.T) {
	tests := []struct {
		name      string
		server    *http.Server
		tlser     []TLSer
		wantError bool
	}{
		{
			name:      "create server without TLSer",
			server:    &http.Server{Addr: ":8080"},
			tlser:     nil,
			wantError: false,
		},
		{
			name:      "create server with TLSer",
			server:    &http.Server{Addr: ":8080"},
			tlser:     []TLSer{&mockTLSer{}},
			wantError: false,
		},
		{
			name:      "create server with nil server",
			server:    nil,
			tlser:     nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(tt.server, tt.tlser...)
			err := srv.validate()

			if (err != nil) != tt.wantError {
				t.Errorf("validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestServer_Run_HTTP(t *testing.T) {
	server := &http.Server{
		Addr: "localhost:0", // Use port 0 for automatic port assignment
	}

	srv := New(server)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Run()
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Test shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 2*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestServer_Run_WithTLSer(t *testing.T) {
	mockTLS := &mockTLSer{}
	server := &http.Server{
		Addr: "localhost:0",
	}

	srv := New(server, mockTLS)

	// This should use the mock TLSer
	err := srv.validate()
	if err != nil {
		t.Errorf("validate() with TLSer error = %v", err)
	}

	if srv.Scheme() != "https" {
		t.Errorf("Scheme() error = %v, want https", srv.Scheme())
	}
}

func TestServer_Shutdown(t *testing.T) {
	tests := []struct {
		name    string
		server  *Server
		wantErr bool
	}{
		{
			name:    "shutdown nil server",
			server:  &Server{server: nil},
			wantErr: false,
		},
		{
			name:    "shutdown running server",
			server:  &Server{server: &http.Server{Addr: ":0"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := tt.server.Shutdown(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Shutdown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// mockTLSer is a mock implementation of TLSer for testing
type mockTLSer struct {
	validateError error
	runError      error
}

func (m *mockTLSer) Validate() error {
	return m.validateError
}

func (m *mockTLSer) Run(server *http.Server) error {
	return m.runError
}
