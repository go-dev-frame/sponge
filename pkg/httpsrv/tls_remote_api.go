package httpsrv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// TLSRemoteAPIOption set tlsRemoteAPIOptions.
type TLSRemoteAPIOption func(*tlsRemoteAPIOptions)

type tlsRemoteAPIOptions struct {
	headers  map[string]string
	timeout  time.Duration
	cacheDir string
}

func (o *tlsRemoteAPIOptions) apply(opts ...TLSRemoteAPIOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultTLSRemoteAPIOptions() *tlsRemoteAPIOptions {
	return &tlsRemoteAPIOptions{
		timeout:  5 * time.Second,
		cacheDir: "configs/remote_api_certs",
	}
}

// WithTLSRemoteAPIHeaders set headers for tlsRemoteAPI.
func WithTLSRemoteAPIHeaders(headers map[string]string) TLSRemoteAPIOption {
	return func(o *tlsRemoteAPIOptions) {
		o.headers = headers
	}
}

// WithTLSRemoteAPITimeout set timeout for tlsRemoteAPI.
func WithTLSRemoteAPITimeout(timeout time.Duration) TLSRemoteAPIOption {
	return func(o *tlsRemoteAPIOptions) {
		o.timeout = timeout
	}
}

// WithTLSRemoteAPICacheDir set cacheDir for tlsRemoteAPI.
func WithTLSRemoteAPICacheDir(cacheDir string) TLSRemoteAPIOption {
	return func(o *tlsRemoteAPIOptions) {
		o.cacheDir = cacheDir
	}
}

// -------------------------------------------------------------------------------------------

var _ TLSer = (*TLSRemoteAPIConfig)(nil)

// TLSRemoteAPIConfig implements certificate retrieval from other service API
type TLSRemoteAPIConfig struct {
	url      string            // Certificate PEM download URL
	headers  map[string]string // Optional: request headers
	timeout  time.Duration     // Optional: HTTP timeout
	cacheDir string            // Cache directory

	certFile string // Cached certificate file path
	keyFile  string // Cached private key file path

	httpClient *http.Client // Internal HTTP client
}

func NewTLSRemoteAPIConfig(url string, opts ...TLSRemoteAPIOption) *TLSRemoteAPIConfig {
	o := defaultTLSRemoteAPIOptions()
	o.apply(opts...)
	return &TLSRemoteAPIConfig{
		url:      url,
		headers:  o.headers,
		timeout:  o.timeout,
		cacheDir: o.cacheDir,
	}
}

func (c *TLSRemoteAPIConfig) Validate() error {
	if c.url == "" {
		return errors.New("certURL and keyURL must be specified for remote API mode")
	}
	if c.cacheDir == "" {
		c.cacheDir = "configs/remote_api_certs"
	}
	if c.timeout <= time.Millisecond*100 {
		c.timeout = 5 * time.Second
	}
	if c.certFile == "" {
		c.certFile = filepath.Join(c.cacheDir, "cert.pem")
	}
	if c.keyFile == "" {
		c.keyFile = filepath.Join(c.cacheDir, "key.pem")
	}
	c.httpClient = &http.Client{Timeout: c.timeout}
	return nil
}

func (c *TLSRemoteAPIConfig) Run(server *http.Server) error {
	var certData, keyData []byte
	var err error

	// retry 3 times to download cert from API
	for i := 0; i < 3; i++ {
		certData, keyData, err = c.downloadFile()
		if err == nil {
			break
		}
		err = fmt.Errorf("download cert from API failed: %v", err)
		time.Sleep(3 * time.Second)
	}
	if err == nil {
		// write cert and key to file
		_ = os.MkdirAll(c.cacheDir, 0760)
		if err = os.WriteFile(c.certFile, certData, 0640); err != nil {
			return fmt.Errorf("failed to write cert file: %v", err)
		}
		if err = os.WriteFile(c.keyFile, keyData, 0640); err != nil {
			return fmt.Errorf("failed to write key file: %v", err)
		}
	}

	if err = server.ListenAndServeTLS(c.certFile, c.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[https server] listen and serve TLS error: %v", err)
	}

	return nil
}

type TLSRemoteAPIResponse struct {
	CertFile []byte `json:"cert_file"`
	KeyFile  []byte `json:"key_file"`
}

func (c *TLSRemoteAPIConfig) downloadFile() (certData []byte, keyData []byte, err error) {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var apiResponse TLSRemoteAPIResponse
	if err = json.Unmarshal(b, &apiResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return apiResponse.CertFile, apiResponse.KeyFile, nil
}
