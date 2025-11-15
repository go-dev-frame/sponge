## HTTP/S Server

`httpsrv` 是一个 Go `net/http` 的便捷封装库，旨在简化和标准化启动 HTTP 和 HTTPS 服务器的流程。它支持多种 TLS 证书管理模式，包括自签名证书、Let's Encrypt、外部文件以及远程 API，让你可以通过简单的配置快速启动一个健壮的 Web 服务器。

<br>

### 特性

- **多种运行模式**:
    - **HTTP**: 快速启动一个标准的 HTTP 服务。
    - **自签名 (Self-Signed)**: 自动为本地开发环境生成和管理自签名 TLS 证书。
    - **Let's Encrypt**: 与 `autocert` 集成，自动获取和续订 Let's Encrypt 证书。
    - **外部文件 (External)**: 使用你提供的现有证书和私钥文件。
    - **远程 API (Remote API)**: 从一个指定的 API 端点动态获取证书。
- **平滑关闭 (Graceful Shutdown)**: 内置 `Shutdown` 方法，轻松实现服务的平滑关闭。
- **配置简单**: 通过链式调用和选项模式，提供清晰、灵活的配置方式。
- **高可扩展性**: `TLSer` 接口允许你轻松实现自定义的证书管理策略，例如从 Etcd、Consul 等获取证书。

<br>

### 使用示例

下面是不同模式下的使用示例。

#### 1. 启动一个标准的 HTTP 服务器

这是最简单的模式，无需任何 TLS 配置。

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // 创建一个 HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    // 配置 http.Server
    httpServer := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // 创建并运行服务
    fmt.Println("HTTP server listening on :8080")
    server := httpsrv.New(httpServer)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 2. HTTPS - 自签名证书 (用于开发)

此模式会自动生成 `cert.pem` 和 `key.pem` 文件，非常适合本地开发和测试。

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // 创建一个 HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    // 配置 http.Server
    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // 配置自签名模式
    tlsConfig := httpsrv.NewTLSSelfSignedConfig(
        // 可选：自定义证书存储目录
        //httpsrv.WithTLSSelfSignedCacheDir("certs/self-signed"),
        // 可选：自定义证书有效期（天）
        //httpsrv.WithTLSSelfSignedExpirationDays(365),
        // 可选：添加其他IP到证书中
        //httpsrv.WithTLSSelfSignedWanIPs("192.168.1.100"),
    )
    
    // 创建并运行服务
    fmt.Println("HTTP server listening on :8443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

<br>

#### 3. HTTPS - Let's Encrypt (用于生产)

此模式会自动从 Let's Encrypt 获取证书，并自动处理续期。

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // 创建一个 HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":443",
        Handler: mux,
    }

    // 配置 Let's Encrypt 模式
    tlsConfig := httpsrv.NewTLSEAutoEncryptConfig(
        "your-domain.com",        // 你的域名
        "your-email@example.com", // 你的邮箱，用于接收 Let's Encrypt 通知
        // 可选：开启 HTTP -> HTTPS 自动重定向 (默认监听 :80)
        //httpsrv.WithTLSEncryptEnableRedirect(),
        // 可选：自定义证书缓存目录
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

#### 4. HTTPS - 使用外部证书文件

如果你已经有自己的证书和私钥文件，可以使用此模式。

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/go-dev-frame/sponge/pkg/httpsrv"
)

func main() {
    // 创建一个 HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // 配置外部文件模式
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

#### 5. HTTPS - 从远程 API 获取证书

此模式允许你从一个 URL 动态拉取证书和私钥。API 应返回一个包含 `cert_file` 和 `key_file` 字段的 JSON 对象，值为 Base64 编码的 PEM 数据，如下所示：

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
    // 创建一个 HTTP Mux
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, HTTP World!")
    })

    httpServer := &http.Server{
        Addr:    ":8443",
        Handler: mux,
    }

    // 配置远程 API 模式
    tlsConfig := httpsrv.NewTLSRemoteAPIConfig(
        "https://your-api-endpoint.com/certs",
        // 可选：设置请求头，用于认证等
        //httpsrv.WithTLSRemoteAPIHeaders(map[string]string{"Authorization": "Bearer your-token"}),
        // 可选：设置 http.Client 请求超时
        //httpsrv.WithTLSRemoteAPITimeout(10*time.Second),
    )

    fmt.Println("HTTP server listening on :8443")
    server := httpsrv.New(httpServer, tlsConfig)
    if err := server.Run(); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}
```
