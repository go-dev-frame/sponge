## 简体中文 | [English](README.md)

## PerfTest

`perftest` 是一个轻量级、高性能的压测工具，支持 **HTTP/1.1、HTTP/2、HTTP/3 和 WebSocket** 协议。
它不仅能快速执行高并发请求，还支持将统计数据实时推送到自定义 HTTP 服务或 **Prometheus**，便于监控和分析。

<br>

### ✨ 特性

* ✅ 支持 **HTTP/1.1、HTTP/2、HTTP/3、WebSocket** 压测
* ✅ 支持 **固定请求数** 或 **固定持续时间** 两种模式
* ✅ 支持 **并发数 (worker) 配置**
* ✅ 支持 **GET/POST/自定义请求体**
* ✅ 支持 **实时统计数据推送**（HTTP Server / Prometheus）
* ✅ 支持 **详细性能报告**（QPS、延迟分布、数据传输、状态码统计等）
* ✅ 支持 **WebSocket 消息压力测试**（自定义消息内容与发送间隔）

<br>

### 📦 安装

```bash
go install github.com/go-dev-frame/sponge/cmd/sponge@latest
```

安装完成后，执行 `sponge perftest -h` 查看帮助。

<br>

### 🚀 使用示例

#### 1. HTTP/1.1 压测

```bash
# 默认模式：5000 次请求，worker=CPU*3，并发执行 GET 请求
sponge perftest http --url=http://localhost:8080/user/1

# 固定请求数模式：50 并发，发送 50w 个请求
sponge perftest http --worker=50 --total=500000 --url=http://localhost:8080/user/1

# 固定请求数模式：POST 请求，带 JSON 请求体
sponge perftest http --worker=50 --total=500000 --method=POST \
  --url=http://localhost:8080/user \
  --body='{"name":"Alice","age":25}'

# 固定持续时间模式：50 并发，持续 10s
sponge perftest http --worker=50 --duration=10s --url=http://localhost:8080/user/1

# 推送统计数据到自定义 HTTP 服务（每秒一次）
sponge perftest http --worker=50 --total=500000 \
  --url=http://localhost:8080/user/1 \
  --push-url=http://localhost:7070/report

# 推送统计数据到 Prometheus（job=xxx）
sponge perftest http --worker=50 --duration=10s \
  --url=http://localhost:8080/user/1 \
  --push-url=http://localhost:9091/metrics \
  --prometheus-job-name=perftest-http
```

**报告示例：**

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

#### 2. HTTP/2 压测

与 HTTP/1.1 使用方式相同，只需将 `http` 替换为 `http2`：

```bash
sponge perftest http2 --worker=50 --total=500000 --url=https://localhost:6443/user/1
```

<br>

#### 3. HTTP/3 压测

与 HTTP/1.1 使用方式相同，只需将 `http` 替换为 `http3`：

```bash
sponge perftest http3 --worker=50 --total=500000 --url=https://localhost:8443/user/1
```

<br>

#### 4. WebSocket 压测

```bash
# 默认模式：10 并发，持续 10s，随机消息（100 字符）
sponge perftest websocket --worker=10 --duration=10s --url=ws://localhost:8080/ws

# 发送固定字符串消息，每个worker发送消息间隔 10ms
sponge perftest websocket --worker=100 --duration=1m \
  --send-interval=10ms \
  --body-string=abcdefghijklmnopqrstuvwxyz \
  --url=ws://localhost:8080/ws

# 发送 JSON 消息，默认无间隔
sponge perftest websocket --worker=10 --duration=10s \
  --body='{"name":"Alice","age":25}' \
  --url=ws://localhost:8080/ws

# 发送 JSON 消息，每个worker发送消息间隔 10ms
sponge perftest websocket --worker=100 --duration=1m \
  --send-interval=10ms \
  --body='{"name":"Alice","age":25}' \
  --url=ws://localhost:8080/ws
```

**报告示例：**

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

## ⚙️ 常用参数说明

| 参数                            | 说明                                                                                 | 示例                                        |
|-------------------------------|------------------------------------------------------------------------------------|-------------------------------------------|
| `--url`, `-u`                 | 请求地址（支持 http/https/ws）                                                             | `--url=http://localhost:8080/user/1`      |
| `--worker`, `-w`              | 并发数（默认=CPU\*3）                                                                     | `--worker=50`                             |
| `--total`, `-t`               | 总请求数（与 `--duration` 互斥，`--duration` 优先级更高）                                         | `--total=500000`                          |
| `--duration`, `-d`            | 测试持续时间（与 `--total` 互斥）                                                             | 持续时间                                                                     | `--duration=10s`                          |
| `--method`, `-m`              | HTTP 方法                                                                            | `--method=POST`                           |
| `--body`, `-b`                | 请求体（支持 JSON 格式）                                                                    | `--body='{"name":"Alice"}'`               |
| `--body-string`,`-s`          | WebSocket 消息内容（字符串）                                                                | `--body-string=hello`                     |
| `--send-interval`, `-i`       | WebSocket 消息发送间隔                                                                   | `--send-interval=10ms`                    |
| `--push-url`, `-p`            | 推送统计数据的地址                                                                          | `--push-url=http://localhost:7070/report` |
| `--prometheus-job-name`, `-j` | Prometheus Job 名称，如果值为空，参数`--push-url`指向的 url 将被认为是自定义 HTTP 服务地址，否则是 Prometheus 地址 | `--prometheus-job-name=xxx`               |

<br>

## 📊 典型使用场景

* ✅ **压测 Web API (HTTP/1.1/2/3)**
* ✅ **对比 HTTP 协议版本性能差异**
* ✅ **压测 WebSocket 实时推送服务**
* ✅ **结合 Prometheus 监控实时压测数据**

<br>

## 📝 总结

`perftest` 是一个简单易用但功能强大的压测工具，既能用于快速验证接口性能，也能集成到 **CI/CD 测试环境** 或 **性能监控平台**。
