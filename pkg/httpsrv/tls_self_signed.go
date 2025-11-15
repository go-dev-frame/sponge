package httpsrv

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// TLSSelfSignedOption set tlsSelfSignedOptions.
type TLSSelfSignedOption func(*tlsSelfSignedOptions)

type tlsSelfSignedOptions struct {
	cacheDir       string
	expirationDays int
	wanIPs         []string // IP addresses to include in the certificate.
}

func (o *tlsSelfSignedOptions) apply(opts ...TLSSelfSignedOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func defaultTLSSelfSignedOptions() *tlsSelfSignedOptions {
	return &tlsSelfSignedOptions{
		cacheDir:       "configs/self_signed_certs",
		expirationDays: 3650,
	}
}

// WithTLSSelfSignedCacheDir sets the cache directory for self-signed certificates.
func WithTLSSelfSignedCacheDir(cacheDir string) TLSSelfSignedOption {
	return func(o *tlsSelfSignedOptions) {
		o.cacheDir = cacheDir
	}
}

// WithTLSSelfSignedExpirationDays sets the expiration days for self-signed certificates.
func WithTLSSelfSignedExpirationDays(expirationDays int) TLSSelfSignedOption {
	return func(o *tlsSelfSignedOptions) {
		o.expirationDays = expirationDays
	}
}

// WithTLSSelfSignedWanIPs sets the IP addresses to include in the certificate.
func WithTLSSelfSignedWanIPs(wanIPs ...string) TLSSelfSignedOption {
	return func(o *tlsSelfSignedOptions) {
		o.wanIPs = wanIPs
	}
}

// ------------------------------------------------------------------------------------------

var _ TLSer = (*TLSSelfSignedConfig)(nil)

type TLSSelfSignedConfig struct {
	cacheDir       string
	certFile       string
	keyFile        string
	expirationDays int
	wanIPs         []string // IP addresses to include in the certificate.
}

func NewTLSSelfSignedConfig(opts ...TLSSelfSignedOption) *TLSSelfSignedConfig {
	o := defaultTLSSelfSignedOptions()
	o.apply(opts...)
	return &TLSSelfSignedConfig{
		cacheDir:       o.cacheDir,
		certFile:       filepath.Join(o.cacheDir, "cert.pem"),
		keyFile:        filepath.Join(o.cacheDir, "key.pem"),
		expirationDays: o.expirationDays,
		wanIPs:         o.wanIPs,
	}
}

func (c *TLSSelfSignedConfig) Validate() error {
	if c.cacheDir == "" {
		c.cacheDir = "configs/self_signed_certs"
	}
	if c.expirationDays < 1 {
		c.expirationDays = 3650
	}
	if c.certFile == "" {
		c.certFile = filepath.Join(c.cacheDir, "cert.pem")
	}
	if c.keyFile == "" {
		c.keyFile = filepath.Join(c.cacheDir, "key.pem")
	}
	return nil
}

// generateCert checks for and generates a certificate for local development.
// If the certificate doesn't exist or expires in less than 30 days, it generates a new one.
func (c *TLSSelfSignedConfig) generateCert() error {
	// Check if the certificate file exists.
	certBytes, err := os.ReadFile(c.certFile)
	if os.IsNotExist(err) {
		return c.createCert()
	}
	if err != nil {
		return err
	}

	// If the cert exists, decode it and check its expiration.
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return c.createCert()
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return c.createCert()
	}

	// Renewal threshold is 30 days.
	renewalThreshold := 30 * 24 * time.Hour
	remainingValidity := time.Until(cert.NotAfter)

	if remainingValidity < renewalThreshold {
		return c.createCert()
	}

	return nil
}

// createCert is a helper function that creates the certificate and key files.
func (c *TLSSelfSignedConfig) createCert() error {
	_ = os.Remove(c.certFile)
	_ = os.Remove(c.keyFile)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	expiration := time.Duration(c.expirationDays) * 24 * time.Hour
	var netIPs = []net.IP{net.ParseIP("127.0.0.1")}
	if lanIP := getLANIP(); lanIP != "" {
		netIPs = append(netIPs, net.ParseIP(lanIP))
	}
	for _, ip := range c.wanIPs {
		netIPs = append(netIPs, net.ParseIP(ip))
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Dev Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(expiration),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           netIPs,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(c.cacheDir, 0760)

	// Write certificate file.
	certOut, err := os.Create(filepath.Clean(c.certFile))
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	// Write key file.
	keyOut, err := os.Create(filepath.Clean(c.keyFile))
	if err != nil {
		return err
	}
	defer keyOut.Close()
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	return pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func (c *TLSSelfSignedConfig) Run(server *http.Server) error {
	if err := c.generateCert(); err != nil {
		return err
	}

	if err := server.ListenAndServeTLS(c.certFile, c.keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("[https server] listen and serve TLS error: %v", err)
	}
	return nil
}

func getLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if isValidIP(localAddr.IP) {
		return localAddr.IP.String()
	}
	return ""
}

func isValidIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() {
		return false
	}
	return ip.To4() != nil
}
