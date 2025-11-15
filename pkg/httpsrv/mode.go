package httpsrv

// Mode defines the server's running mode.
type Mode string

const (
	// ModeTLSSelfSigned runs the server using a self-signed TLS certificate (localhost or testing use).
	ModeTLSSelfSigned Mode = "self-signed"
	// ModeTLSEncrypt runs the server using a Let's Encrypt certificate (automatic certificate management).
	ModeTLSEncrypt Mode = "encrypt"
	// ModeTLSExternal runs the server using user-provided TLS certificate and key files.
	ModeTLSExternal Mode = "external"
	// ModeRemoteAPI runs the server using remote API to manage TLS certificate.
	ModeRemoteAPI = "remote-api"
)
