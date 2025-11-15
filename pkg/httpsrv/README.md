## English | [中文](readme-cn.md)

## HTTP Server

`httpsrv` is a convenient wrapper library for Go's `net/http`, designed to simplify and standardize the process of starting HTTP and HTTPS servers. It supports multiple TLS certificate management modes, including self-signed certificates, Let's Encrypt, external files, and remote APIs, allowing you to quickly launch a robust web server with simple configuration.

<br>

### Features

- **Multiple Operating Modes**:
    - **HTTP**: Quickly start a standard HTTP service.
    - **Self-Signed**: Automatically generate and manage self-signed TLS certificates for local development environments.
    - **Let's Encrypt**: Integrates with `autocert` to automatically obtain and renew Let's Encrypt certificates.
    - **External**: Use your existing certificate and private key files.
    - **Remote API**: Dynamically fetch certificates from a specified API endpoint.
- **Graceful Shutdown**: Built-in `Shutdown` method for easy implementation of a graceful server shutdown.
- **Simple Configuration**: Provides a clear and flexible configuration method through chain calls and the option pattern.
- **High Extensibility**: The `TLSer` interface allows you to easily implement custom certificate management strategies, such as fetching certificates from Etcd, Consul, etc.

<br>

### Examples of Usage

Below are usage examples for different modes.

#### 1. Start a standard HTTP server

This is the simplest mode, requiring no TLS configuration.

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // Create an HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    // Configure http.Server
    httpServer := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // Create and run the service
    fmt.Println("HTTP server listening on :8080")
    server := httpsrv.New(httpServer)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 2. HTTPS - Self-Signed Certificate (for development)

This mode automatically generates `cert.pem` and `key.pem` files, making it ideal for local development and testing.

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // Create an HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    // Configure http.Server
    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // Configure self-signed mode
    tlsConfig := httpsrv.NewTLSSelfSignedConfig(
        // Optional: Custom certificate storage directory
        //httpsrv.WithTLSSelfSignedCacheDir("certs/self-signed"),
        // Optional: Custom certificate validity period (in days)
        //httpsrv.WithTLSSelfSignedExpirationDays(365),
        // Optional: Add other IPs to the certificate
        //httpsrv.WithTLSSelfSignedWanIPs("192.168.1.100"),
    )
    
    // Create and run the service
    fmt.Println("HTTP server listening on :8443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 3. HTTPS - Let's Encrypt (for production)

This mode automatically obtains certificates from Let's Encrypt and handles renewal.

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // Create an HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":443",
        Handler: mux,
    }

    // Configure Let's Encrypt mode
    tlsConfig := httpsrv.NewTLSEAutoEncryptConfig(
        "your-domain.com",        // Your domain
        "your-email@example.com", // Your email for Let's Encrypt notifications
        // Optional: Enable HTTP -> HTTPS automatic redirection (listens on :80 by default)
        //httpsrv.WithTLSEncryptEnableRedirect(),
        // Optional: Custom certificate cache directory
        //httpsrv.WithTLSEncryptCacheDir("certs/encrypt"),
    )

    fmt.Println("HTTP server listening on :443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 4. HTTPS - Using External Certificate Files

If you already have your own certificate and private key files, you can use this mode.

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // Create an HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // Configure external file mode
    tlsConfig := httpsrv.NewTLSExternalConfig(
        "/path/to/your/cert.pem",
        "/path/to/your/key.pem",
    )

    fmt.Println("HTTP server listening on :8443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 5. HTTPS - Fetching Certificates from a Remote API

This mode allows you to dynamically pull the certificate and private key from a URL. The API should return a JSON object containing `cert_file` and `key_file` fields, with their values being Base64-encoded PEM data, as shown below:

```json
{
  "cert_file": "-----BEGIN CERTIFICATE-----\nCERTIFICATE_DATA\n-----END CERTIFICATE-----",
  "key_file": "-----BEGIN PRIVATE KEY-----\nPRIVATE_KEY_DATA\n-----END PRIVATE KEY-----"
}
```

```go
package main

import (
    "fmt"
    "github.com/go-dev-frame/sponge/pkg/httpsrv"
    "net/http"
)

func main() {
    // Create an HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // Configure remote API mode
    tlsConfig := httpsrv.NewTLSRemoteAPIConfig(
        "https://your-api-endpoint.com/certs",
        // Optional: Set request headers for authentication, etc.
        //httpsrv.WithTLSRemoteAPIHeaders(map[string]string{"Authorization": "Bearer your-token"}),
        // Optional: Set http.Client request timeout
        //httpsrv.WithTLSRemoteAPITimeout(10*time.Second),
    )

    fmt.Println("HTTP server listening on :8443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```
