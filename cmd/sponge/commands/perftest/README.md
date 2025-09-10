## English | [简体中文](readme-cn.md)

## PerfTest

`perftest` is a lightweight and high-performance testing tool that supports **HTTP/1.1, HTTP/2, HTTP/3, and WebSocket** protocols.
It can execute high-concurrency requests efficiently and push real-time statistics to a custom HTTP server or **Prometheus** for monitoring and analysis.

<br>

### ✨ Features

* ✅ Support for **HTTP/1.1, HTTP/2, HTTP/3, WebSocket**
* ✅ Two modes: **fixed total requests** or **fixed duration**
* ✅ Configurable **workers (concurrency)**
* ✅ Support for **GET, POST, custom request bodies**
* ✅ **Real-time statistics push** (HTTP server / Prometheus)
* ✅ **Detailed performance reports** (QPS, latency distribution, data transfer, status codes, etc.)
* ✅ **WebSocket message performance test** with custom message payload and send interval

<br>

### 📦 Installation

```bash
go install github.com/go-dev-frame/sponge/cmd/sponge@latest
```

After installation, run `sponge perftest -h` to see usage.

<br>

### 🚀 Usage Examples

#### 1. HTTP/1.1 Performance Test

```bash
# Default mode: worker=CPU*3, 5000 requests, GET request
sponge perftest http --url=http://localhost:8080/user/1

# Fixed number of requests: 50 workers, 500k requests
sponge perftest http --worker=50 --total=500000 --url=http://localhost:8080/user/1

# Fixed number of requests: POST with JSON body
sponge perftest http --worker=50 --total=500000 --method=POST \
  --url=http://localhost:8080/user \
  --body='{"name":"Alice","age":25}'

# Fixed duration: 50 workers, run for 10s
sponge perftest http --worker=50 --duration=10s --url=http://localhost:8080/user/1

# Push statistics to custom HTTP server every 1s
sponge perftest http --worker=50 --total=500000 \
  --url=http://localhost:8080/user/1 \
  --push-url=http://localhost:9090/report

# Push statistics to Prometheus (job=xxx)
sponge perftest http --worker=50 --duration=10s \
  --url=http://localhost:8080/user/1 \
  --push-url=http://localhost:9091/metrics \
  --prometheus-job-name=perftest-http
```

**Report Example:**

```
500000 / 500000   [==================================================] 100.00% 8.85s

========== HTTP/1.1 Performance Test Report ==========

[Requests]
  • Total Requests:    500000
  • Successful:        500000 (100%)
  • Failed:            0
  • Total Duration:    8.85s
  • Throughput (QPS):  56489.26 req/sec

[Latency]
  • Average:           0.88 ms
  • Minimum:           0.00 ms
  • Maximum:           21.56 ms
  • P25:               0.00 ms
  • P50:               1.01 ms
  • P95:               2.34 ms

[Data Transfer]
  • Sent:              12.5 MB
  • Received:          24.5 MB

[Status Codes]
  • 200:               500000
```

<br>

#### 2. HTTP/2 Performance Test

Usage is the same as HTTP/1.1, just replace `http` with `http2`:

```bash
sponge perftest http2 --worker=50 --total=500000 --url=http2://localhost:8080/user/1
```

<br>

#### 3. HTTP/3 Performance Test

Usage is the same as HTTP/1.1, just replace `http` with `http3`:

```bash
sponge perftest http3 --worker=50 --total=500000 --url=http3://localhost:8080/user/1
```

<br>

#### 4. WebSocket Performance Test

```bash
# Default: 10 workers, 10s duration, random(10) string message
sponge perftest websocket --url=ws://localhost:8080/ws

# Send fixed string messages, interval=10ms
sponge perftest websocket --worker=100 --duration=1m \
  --send-interval=10ms \
  --body-string=abcdefghijklmnopqrstuvwxyz \
  --url=ws://localhost:8080/ws

# Send JSON messages, default no interval
sponge perftest websocket --worker=10 --duration=10s \
  --body='{"name":"Alice","age":25}' \
  --url=ws://localhost:8080/ws

# Send JSON messages, interval=10ms
sponge perftest websocket --worker=100 --duration=1m \
  --send-interval=10ms \
  --body='{"name":"Alice","age":25}' \
  --url=ws://localhost:8080/ws
```

**Report Example:**

```
5.0s / 5.0s [==================================================] 100.00%

========== WebSocket Performance Test Report ==========

[Connections]
  • Total:              10
  • Successful:         10 (100%)
  • Failed:             0
  • Latency:            min: 14.80 ms, avg: 14.80 ms, max: 14.80 ms

[Messages Sent]
  • Total Messages:     2954089
  • Total Bytes:        295408900
  • Throughput (QPS):   590817.80 msgs/sec

[Messages Received]
  • Total Messages:     2954089
  • Total Bytes:        295408900
  • Throughput (QPS):   590817.80 msgs/sec
```

<br>

### ⚙️ Common Parameters

| Parameter               | Description                                                                   | Example                                   |
| ----------------------- |-------------------------------------------------------------------------------|-------------------------------------------|
| `--url`, `-u`                 | Request URL (http/https/ws)                                                   | `--url=http://localhost:8080/user/1`      |
| `--worker`, `-w`              | Number of concurrent workers (default = CPU\*3)                               | `--worker=50`                             |
| `--total`, `-t`               | Total number of requests (mutually exclusive with `--duration`, `--duration` higher priority) | `--total=500000`                          |
| `--duration`, `-d`              | Duration of the test (mutually exclusive with `--total`)                      | `--duration=10s`                          |
| `--method`, `-m`              | HTTP method                                                                   | `--method=POST`                           |
| `--body`, `-b`                 | Request body (JSON supported)                                                 | `--body='{"name":"Alice"}'`               |
| `--body-string`,`-s`         | WebSocket message string body                                                 | `--body-string=hello`                     |
| `--send-interval`, `-i`       | Interval between WebSocket messages                                           | `--send-interval=10ms`                    |
| `--push-url`, `-p`            | URL to push statistics                                                        | `--push-url=http://localhost:9090/report` |
| `--prometheus-job-name`, `-j` | Job name for Prometheus metrics                                               | `--prometheus-job-name=perftest-http`     |

<br>

### 📊 Typical Use Cases

* ✅  Performance testing **Web APIs (HTTP/1.1/2/3)**
* ✅ Comparing performance differences across HTTP versions
* ✅ Stress-testing **real-time WebSocket services**
* ✅ **CI/CD integration** with Prometheus monitoring

<br>

### 📝 Summary

`perftest` is a simple yet powerful performance testing tool.
It’s suitable for quick performance verification as well as integration into **CI/CD pipelines** or **real-time monitoring systems** with Prometheus and Grafana.
