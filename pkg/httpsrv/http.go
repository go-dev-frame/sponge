package httpsrv

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// TLSer abstract different TLS operation schemes
type TLSer interface {
	Validate() error
	Run(server *http.Server) error
}

// Server TLS server.
type Server struct {
	scheme string       // e.g. http or https.
	server *http.Server // server is the HTTP server instance.

	tlser TLSer
}

// New returns a new Server with TLSer injected.
func New(server *http.Server, tlser ...TLSer) *Server {
	var tlsMode TLSer
	if len(tlser) > 0 {
		tlsMode = tlser[0]
	}

	var scheme = "https"
	if tlsMode == nil {
		scheme = "http"
	}

	return &Server{
		scheme: scheme,
		server: server,
		tlser:  tlsMode,
	}
}

func (s *Server) validate() error {
	if s.server == nil {
		return errors.New("server must be specified")
	}
	if s.tlser != nil {
		return s.tlser.Validate()
	}
	return nil
}

// Run starts the server according to the provided configuration.
func (s *Server) Run() error {
	if err := s.validate(); err != nil {
		return err
	}

	// no TLS mode specified, run in http mode.
	if s.tlser == nil {
		return s.runHTTP()
	}

	return s.tlser.Run(s.server)
}

// Shutdown gracefully shuts down the server and releases resources.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// runHTTP starts the server in http mode, without TLS.
func (s *Server) runHTTP() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[http server] listen and serve error: %v", err)
	}
	return nil
}

// Scheme returns the scheme of the server, e.g. http or https.
func (s *Server) Scheme() string {
	return s.scheme
}
