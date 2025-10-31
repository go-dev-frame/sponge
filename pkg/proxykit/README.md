## Proxy Kit

A production-grade reverse proxy library implemented in Go. It not only provides high performance and multiple load balancing strategies, but also supports **dynamic route management**, allowing you to add or remove backend servers at runtime via API — without restarting the service.

<br>

### Core Features

*   **Dynamic Service Discovery**: Add or remove backend nodes in real-time through HTTP APIs.
*   **High Performance Core**: Built on `net/http/httputil` with deeply optimized connection pooling for effortless high-concurrency handling.
*   **Rich Load Balancing Strategies**: Includes Round Robin, The Least Connections, and IP Hash.
*   **Active Health Checks**: Automatically detects and isolates unhealthy nodes, and brings them back online once they recover.
*   **Multi-route Support**: Distribute traffic to different backend groups based on path prefixes.

<br>

### Example of Usage

```go
package main

import (
    "log"
    "net/http"
    "time"
    "github.com/go-dev-frame/sponge/pkg/proxykit"
)

func main() {
    prefixPath := "/proxy/"

    // 1. Create the route manager
    manager := proxykit.NewRouteManager()

    // 2. Initialize backend targets
    initialTargets := []string{"http://localhost:8081", "http://localhost:8082"}
    backends, _ := proxykit.ParseBackends(prefixPath, initialTargets)
    proxykit.StartHealthChecks(backends, proxykit.HealthCheckConfig{Interval: 5 * time.Second})
    balancer := proxykit.NewRoundRobin(backends)

    // Register route
    apiRoute, err := manager.AddRoute(prefixPath, balancer)
    if err != nil {
        log.Fatalf("Could not add initial route: %v", err)
    }

    // 3. Build standard library mux
    mux := http.NewServeMux()

    // 4. Register proxy handler (same as gin.Any)
    mux.Handle(prefixPath, apiRoute.Proxy)

    // 5. Management API (corresponds to /endpoints/...)
    mux.HandleFunc("/endpoints/add", manager.HandleAddBackends)
    mux.HandleFunc("/endpoints/remove", manager.HandleRemoveBackends)
    mux.HandleFunc("/endpoints/list", manager.HandleListBackends)
    mux.HandleFunc("/endpoints", manager.HandleGetBackend)

    // 6. Other normal routes
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message": "pong"}`))
    })

    // 7. Start HTTP server
    log.Println("HTTP server with dynamic proxy started on http://localhost:8080")
    log.Printf("Proxying requests from %s*\n", prefixPath)
    log.Println("Management API available at /endpoints/*")

    err = http.ListenAndServe(":8080", mux)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
```

<br>

### Management API Guide

After the proxy is started, you can manage backend services dynamically via the following APIs.

#### 1. List all backends

Retrieve all backend nodes and their health status for the given route.

* **GET** `/endpoints/list?prefixPath=/api/`

```json
{
  "prefixPath": "/api/",
  "targets": [
    {"target": "http://localhost:8081", "healthy": true}
  ]
}
```

#### 2. Add backend nodes

Dynamically scale out. New nodes will automatically enter the health check loop and start receiving traffic.

* **POST** `/endpoints/add`
* **Body**:

  ```json
  {
    "prefixPath": "/api/",
    "targets": ["http://localhost:8083", "http://localhost:8084"]
  }
  ```

#### 3. Remove backend nodes

Dynamically scale in. Health checks for removed nodes will stop automatically.

* **POST** `/endpoints/remove`
* **Body**:

  ```json
  {
    "prefixPath": "/api/",
    "targets": ["http://localhost:8081"]
  }
  ```

#### 4. Inspect a single backend node

* **GET** `/endpoints?prefixPath=/api/&target=http://localhost:8082`

```json
{
  "target": "http://localhost:8082",
  "healthy": true
}
```
