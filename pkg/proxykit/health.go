package proxykit

import (
	"net"
	"time"
)

// HealthCheckConfig defined the configuration for health check.
type HealthCheckConfig struct {
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
}

// StartHealthChecks initiate backend health check for the backend server pool.
func StartHealthChecks(backends []*Backend, config HealthCheckConfig) {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 2 * time.Second
	}

	for _, b := range backends {
		go runHealthCheck(b, config)
	}
}

func runHealthCheck(backend *Backend, config HealthCheckConfig) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-backend.stopHealthCheck:
			return

		case <-ticker.C:
			host := backend.URL.Hostname()
			port := backend.URL.Port()

			if port == "" {
				switch backend.URL.Scheme {
				case "https":
					port = "443"
				case "http":
					port = "80"
				default:
					log.Printf("[Health Check] Unsupported scheme '%s' for backend %s, marking as UNHEALTHY", backend.URL.Scheme, backend.URL)
					if backend.IsHealthy() {
						backend.SetHealthy(false)
					}
					continue
				}
			}

			addressToDial := net.JoinHostPort(host, port)
			conn, err := net.DialTimeout("tcp", addressToDial, config.Timeout)
			if err != nil {
				if backend.IsHealthy() {
					log.Printf("[Health Check] %s is now UNHEALTHY: failed to connect to %s - %v", backend.URL, addressToDial, err)
					backend.SetHealthy(false)
				}
				continue
			}
			_ = conn.Close()

			if !backend.IsHealthy() {
				log.Printf("[Health Check] %s is now HEALTHY", backend.URL)
				backend.SetHealthy(true)
			}
		}
	}
}
