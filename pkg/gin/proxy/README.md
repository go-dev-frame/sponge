## Gin Reverse Proxy

A production-grade reverse proxy library implemented in Go. It not only provides high performance and multiple load balancing strategies, but also supports **dynamic route management**, allowing you to add or remove backend servers at runtime via API — without restarting the service.

### Core Features

*   **Dynamic Service Discovery**: Add or remove backend nodes in real-time through HTTP APIs.
*   **High Performance Core**: Built on `net/http/httputil` with deeply optimized connection pooling for effortless high-concurrency handling.
*   **Rich Load Balancing Strategies**: Includes Round Robin, The Least Connections, and IP Hash.
*   **Active Health Checks**: Automatically detects and isolates unhealthy nodes, and brings them back online once they recover.
*   **Multi-route Support**: Distribute traffic to different backend groups based on path prefixes.

### Example of Usage

```go
package main

import (
    "fmt"
    "github.com/go-dev-frame/sponge/pkg/gin/proxy"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()

    p := proxy.New(r) // default configuration, managerPrefixPath = "/endpoints"
    pass1(p)
    pass2(p)

    // Other normal routes
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    fmt.Println("Gin server with dynamic proxy started on http://localhost:8080")

    r.Run(":8080")
}

func pass1(p *proxy.Proxy) {
    prefixPath := "/proxy/"
    initialTargets := []string{"http://localhost:8081", "http://localhost:8082"}
    err := p.Pass(prefixPath, initialTargets) // default configuration, balancer = RoundRobin, healthCheckInterval = 5s, healthCheckTimeout = 3s
    if err != nil {
        panic(err)
    }
}

func pass2(p *proxy.Proxy) {
    prefixPath := "/personal/"
    initialTargets := []string{"http://localhost:8083", "http://localhost:8084"}
    err := p.Pass(prefixPath, initialTargets)
    if err != nil {
        panic(err)
    }
}
```

<br>

### Advanced Settings

1. Configure the management endpoints' route prefix, middleware, and logger through function parameters of `proxy.New`.
    ```go
    p := proxy.New(r, proxy.Config{
        proxy.WithManagerEndpoints("/admin", Middlewares...),
        proxy.WithLogger(zap.NewExample()),
    })
    ```

2. Configure the load balancing type, health check interval, and middleware for the endpoints through function parameters of `p.Pass`.
    ```go
    err := p.Pass("/proxy/", []string{"http://localhost:8081", "http://localhost:8082"}, proxy.Config{
        proxy.WithPassBalancer(proxy.BalancerIPHash),
        proxy.WithPassHealthCheck(time.Second * 5, time.Second * 3),
        proxy.WithPassMiddlewares(Middlewares...),
    })
    ```

<br>

### Management API Guide

After the proxy is started, you can manage backend services dynamically via the following APIs.

#### 1. List all backends

Retrieve all backend nodes and their health status for the given route.

* **GET** `/endpoints/list?prefixPath=/proxy/`

```json
{
  "prefixPath": "/proxy/",
  "targets": [
    {"target": "http://localhost:8081", "healthy": true},
    {"target": "http://localhost:8082", "healthy": true}
  ]
}
```

<br>

#### 2. Add backend nodes

Dynamically scale out. New nodes will automatically enter the health check loop and start receiving traffic.

* **POST** `/endpoints/add`
* **Body**:

  ```json
  {
    "prefixPath": "/proxy/",
    "targets": ["http://localhost:8085"]
  }
  ```

<br>

#### 3. Remove backend nodes

Dynamically scale in. Health checks for removed nodes will stop automatically.

* **POST** `/endpoints/remove`
* **Body**:

  ```json
  {
    "prefixPath": "/proxy/",
    "targets": ["http://localhost:8085"]
  }
  ```

<br>

#### 4. Check the status of a single backend node

* **GET** `/endpoints?prefixPath=/proxy/&target=http://localhost:8082`

   ```json
   {
     "target": "http://localhost:8082",
     "healthy": true
   }
   ```
