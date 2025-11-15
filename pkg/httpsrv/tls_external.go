package httpsrv

import (
	"errors"
	"fmt"
	"net/http"
)

var _ TLSer = (*TLSExternalConfig)(nil)

type TLSExternalConfig struct {
	certFile string
	keyFile  string
}

func NewTLSExternalConfig(certFile, keyFile string) *TLSExternalConfig {
	return &TLSExternalConfig{
		certFile: certFile,
		keyFile:  keyFile,
	}
}

func (c *TLSExternalConfig) Validate() error {
	if c.certFile == "" {
		return errors.New("cert file must be specified in external mode")
	}
	if c.keyFile == "" {
		return errors.New("key file must be specified in external mode")
	}
	return nil
}

func (c *TLSExternalConfig) Run(server *http.Server) error {
	if err := server.ListenAndServeTLS(c.certFile, c.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[https server] listen and serve TLS error: %v", err)
	}
	return nil
}
