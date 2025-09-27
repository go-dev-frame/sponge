package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// collector of statistical results
type statsCollector struct {
	durations      []float64
	totalReqBytes  int64
	totalRespBytes int64
	successCount   uint64
	errorCount     uint64
	errSet         map[string]struct{}
	statusCodeSet  map[int]int64
}

func (c *statsCollector) collect(results <-chan Result, done chan<- struct{}) {
	errSet := make(map[string]struct{})
	statusCodes := make(map[int]int64)

	for r := range results {
		if r.Err == nil {
			c.successCount++
			c.durations = append(c.durations, float64(r.Duration))
		} else {
			c.errorCount++
			if _, ok := errSet[r.Err.Error()]; !ok {
				errSet[r.Err.Error()] = struct{}{}
			}
		}
		if _, ok := statusCodes[r.StatusCode]; !ok {
			statusCodes[r.StatusCode] = 1
		} else {
			statusCodes[r.StatusCode]++
		}

		c.totalReqBytes += r.ReqSize
		c.totalRespBytes += r.RespSize
	}
	c.errSet = errSet
	c.statusCodeSet = statusCodes

	close(done)
}

// nolint
func (c *statsCollector) collectAndPush(ctx context.Context, results <-chan Result, done chan<- struct{},
	spc *statsPrometheusCollector, p *PerfTestHTTP, start time.Time) {
	errSet := make(map[string]struct{})
	statusCodes := make(map[int]int64)

	pushTicker := time.NewTicker(p.pushInterval)
	defer pushTicker.Stop()
	start = time.Now()

	for r := range results {
		if r.Err == nil {
			c.successCount++
			c.durations = append(c.durations, float64(r.Duration))
		} else {
			c.errorCount++
			if _, ok := errSet[r.Err.Error()]; !ok {
				errSet[r.Err.Error()] = struct{}{}
			}
		}
		if _, ok := statusCodes[r.StatusCode]; !ok {
			statusCodes[r.StatusCode] = 1
		} else {
			statusCodes[r.StatusCode]++
		}

		c.totalReqBytes += r.ReqSize
		c.totalRespBytes += r.RespSize
		c.errSet = errSet
		c.statusCodeSet = statusCodes
		select {
		case <-pushTicker.C:
			spc.copyStatsCollector(c)
			if p.PrometheusJobName == "" {
				spc.PushToServerAsync(ctx, p.PushURL, time.Since(start), p.Params, p.ID, p.agentID, AgentStatusRunning)
			} else {
				spc.PushToPrometheusAsync(ctx, p.PushURL, p.PrometheusJobName, time.Since(start))
				if p.clusterEnable {
					spc.PushToServerAsync(ctx, p.pushToCollectorURL, time.Since(start), p.Params, p.ID, p.agentID, AgentStatusRunning)
				}
			}
		default:
			continue
		}
	}

	close(done)
}

func (c *statsCollector) toStatistics(totalTime time.Duration, totalRequests uint64, params *HTTPReqParams) *Statistics {
	sort.Float64s(c.durations)

	var totalDuration float64
	for _, d := range c.durations {
		totalDuration += d
	}

	var avg, minLatency, maxLatency float64
	var p25, p50, p95, p99 float64

	if c.successCount > 0 {
		avg = totalDuration / float64(c.successCount)
		minLatency = c.durations[0]
		maxLatency = c.durations[c.successCount-1]
		percentile := func(p float64) float64 {
			index := int(float64(c.successCount-1) * p)
			return c.durations[index]
		}
		p25 = percentile(0.25)
		p50 = percentile(0.50)
		p95 = percentile(0.95)
		p99 = percentile(0.99)
	}

	errors := []string{}
	for errStr := range c.errSet {
		errors = append(errors, errStr)
	}

	body := params.Body
	if len(body) > 300 {
		body = append(body[:293], []byte(" ......")...)
	}

	return &Statistics{
		URL:    params.URL,
		Method: params.Method,
		Body:   string(body),

		TotalRequests: totalRequests,
		Errors:        errors,
		SuccessCount:  c.successCount,
		ErrorCount:    c.errorCount,
		TotalDuration: math.Round(totalTime.Seconds()*100) / 100,
		QPS:           math.Round(float64(c.successCount)/totalTime.Seconds()*10) / 10,

		AvgLatency: convertToMilliseconds(avg),
		P25Latency: convertToMilliseconds(p25),
		P50Latency: convertToMilliseconds(p50),
		P95Latency: convertToMilliseconds(p95),
		P99Latency: convertToMilliseconds(p99),
		MinLatency: convertToMilliseconds(minLatency),
		MaxLatency: convertToMilliseconds(maxLatency),

		TotalSent:     c.totalReqBytes,
		TotalReceived: c.totalRespBytes,
		StatusCodes:   c.statusCodeSet,
	}
}

// convert float64 to string with specified precision, automatically process the last 0
func float64ToString(f float64, precision int) string {
	if precision == 0 {
		return strconv.FormatInt(int64(f), 10)
	}
	if precision >= 1 && precision <= 6 {
		factor := math.Pow10(precision)
		rounded := math.Round(f*factor) / factor
		return strconv.FormatFloat(rounded, 'f', precision, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func float64ToStringNoRound(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func (c *statsCollector) printReport(totalDuration time.Duration, totalRequests uint64, params *HTTPReqParams, id string) (*Statistics, error) {
	var builder Builder

	builder.WriteStringf("\n========== %s Performance Test Report ==========\n\n", params.version)
	if c.successCount == 0 {
		builder.WriteString(color.New(color.Bold).Sprint("[Requests]\n"))
		builder.WriteStringf("  • %-19s%d\n", "Total Requests:", totalRequests)
		builder.WriteStringf("  • %-19s%d%s\n", "Successful:", 0, color.RedString(" (0%)"))
		builder.WriteStringf("  • %-19s%d%s\n", "Failed:", c.errorCount, color.RedString(" ✗"))
		builder.WriteStringf("  • %-19s%s s\n\n", "Total Duration:", float64ToString(totalDuration.Seconds(), 2))

		if len(c.statusCodeSet) > 0 {
			printStatusCodeSet(&builder, c.statusCodeSet)
		}

		if len(c.errSet) > 0 {
			printErrorSet(&builder, c.errSet)
		}

		fmt.Println(builder.String())
		return nil, nil
	}

	st := c.toStatistics(totalDuration, totalRequests, params)
	st.ID = id

	builder.WriteString(color.New(color.Bold).Sprint("[Requests]\n"))
	builder.WriteStringf("  • %-19s%d\n", "Total Requests:", st.TotalRequests)
	successStr := fmt.Sprintf("  • %-19s%d", "Successful:", st.SuccessCount)
	failureStr := fmt.Sprintf("  • %-19s%d", "Failed:", st.ErrorCount)
	if st.TotalRequests > 0 {
		if st.ErrorCount > 0 {
			failureStr += color.RedString(" ✗ ")
		}
		if totalRequests == st.SuccessCount {
			successStr += color.GreenString(" (100%) ")
		} else {
			percentage := float64ToString(float64(st.SuccessCount)/float64(st.TotalRequests)*100, 1)
			successStr += color.YellowString(" (%s%%) ", percentage)
		}
	}
	builder.WriteString(successStr + "\n")
	builder.WriteString(failureStr + "\n")
	builder.WriteStringf("  • %-19s%s s\n", "Total Duration:", float64ToStringNoRound(st.TotalDuration))
	builder.WriteStringf("  • %-19s%s req/sec\n\n", "Throughput (QPS):", float64ToStringNoRound(st.QPS))

	builder.WriteString(color.New(color.Bold).Sprint("[Latency]\n"))
	builder.WriteStringf("  • %-19s%s ms\n", "Average:", float64ToStringNoRound(st.AvgLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "Minimum:", float64ToStringNoRound(st.MinLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "Maximum:", float64ToStringNoRound(st.MaxLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "P25:", float64ToStringNoRound(st.P25Latency))
	builder.WriteStringf("  • %-19s%s ms\n", "P50:", float64ToStringNoRound(st.P50Latency))
	builder.WriteStringf("  • %-19s%s ms\n", "P95:", float64ToStringNoRound(st.P95Latency))
	builder.WriteStringf("  • %-19s%s ms\n\n", "P99:", float64ToStringNoRound(st.P99Latency))

	builder.WriteString(color.New(color.Bold).Sprint("[Data Transfer]\n"))
	builder.WriteStringf("  • %-19s%d Bytes\n", "Sent:", st.TotalSent)
	builder.WriteStringf("  • %-19s%d Bytes\n\n", "Received:", st.TotalReceived)

	if len(c.statusCodeSet) > 0 {
		printStatusCodeSet(&builder, st.StatusCodes)
	}

	if len(c.errSet) > 0 {
		printErrorSet(&builder, c.errSet)
	}

	fmt.Printf("%s", builder.String())

	return st, nil
}

func printStatusCodeSet(builder *Builder, statusCodeSet map[int]int64) {
	codes := make([]int, 0, len(statusCodeSet))
	for code := range statusCodeSet {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	builder.WriteString(color.New(color.Bold).Sprint("[Status Codes]\n"))
	for _, code := range codes {
		builder.WriteStringf("  • %-19s%d\n", fmt.Sprintf("%d:", code), statusCodeSet[code])
	}
	builder.WriteString("\n")
}

func printErrorSet(builder *Builder, errSet map[string]struct{}) {
	builder.WriteString(color.New(color.Bold).Sprint("[Error Details]\n"))
	for errStr := range errSet {
		builder.WriteStringf("  • %s\n", color.RedString(errStr))
	}
	builder.WriteString("\n")
}

// --------------------------------------------------------------------------------

// Statistics performance test statistical data
type Statistics struct {
	ID string `json:"id"` // Performance Test ID

	URL    string `json:"url"`    // performed request URL
	Method string `json:"method"` // request method
	Body   string `json:"body"`   // request body (JSON)

	TotalRequests uint64   `json:"total_requests"` // total requests
	TotalDuration float64  `json:"total_duration"` //  total duration (seconds)
	SuccessCount  uint64   `json:"success_count"`  // successful requests (status code 2xx)
	ErrorCount    uint64   `json:"error_count"`    // failed requests (status code not 2xx)
	Errors        []string `json:"errors"`         // error details

	QPS        float64 `json:"qps"`         // requests per second (Throughput)
	AvgLatency float64 `json:"avg_latency"` // average latency (ms)
	P25Latency float64 `json:"p25_latency"` // 25th percentile latency (ms)
	P50Latency float64 `json:"p50_latency"` // 50th percentile latency (ms)
	P95Latency float64 `json:"p95_latency"` // 95th percentile latency (ms)
	P99Latency float64 `json:"p99_latency"` // 95th percentile latency (ms)
	MinLatency float64 `json:"min_latency"` // minimum latency (ms)
	MaxLatency float64 `json:"max_latency"` // maximum latency (ms)

	TotalSent     int64 `json:"total_sent"`     // total sent (bytes)
	TotalReceived int64 `json:"total_received"` // total received (bytes)

	StatusCodes map[int]int64 `json:"status_codes"` // status code distribution (count)

	CreatedAt time.Time `json:"created_at"` // created time

	Status  string `json:"status"`   // running, finished
	AgentID string `json:"agent_id"` // identify agent
}

// Save saves the statistics data to a JSON file.
func (s *Statistics) Save(filePath string) error {
	err := ensureFileExists(filePath)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ensureFileExists(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()
	}
	return nil
}

func convertToMilliseconds(f float64) float64 {
	if f <= 0.0 {
		return 0
	}
	return math.Round((f/1e6)*100) / 100
}

// --------------------------------------------------------------------------

type statsPrometheusCollector struct {
	statsCollector *statsCollector

	// prometheus metrics
	totalRequestsGauge prometheus.Gauge
	successGauge       prometheus.Gauge
	errorGauge         prometheus.Gauge
	totalTimeGauge     prometheus.Gauge
	qpsGauge           prometheus.Gauge
	avgLatencyGauge    prometheus.Gauge
	p25LatencyGauge    prometheus.Gauge
	p50LatencyGauge    prometheus.Gauge
	p95LatencyGauge    prometheus.Gauge
	minLatencyGauge    prometheus.Gauge
	maxLatencyGauge    prometheus.Gauge
	totalSentGauge     prometheus.Gauge
	totalRecvGauge     prometheus.Gauge
	statusCodeGaugeVec *prometheus.GaugeVec
}

func newStatsPrometheusCollector() *statsPrometheusCollector {
	return &statsPrometheusCollector{
		totalRequestsGauge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_total_requests", Help: "Total requests"}),
		successGauge:       prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_success_count", Help: "Successful requests"}),
		errorGauge:         prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_error_count", Help: "Failed requests"}),
		totalTimeGauge:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_total_time_seconds", Help: "Total time elapsed"}),
		qpsGauge:           prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_qps", Help: "Queries per second"}),
		avgLatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_avg_latency_ms", Help: "Average latency (ms)"}),
		p25LatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_p25_latency_ms", Help: "P25 latency (ms)"}),
		p50LatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_p50_latency_ms", Help: "P50 latency (ms)"}),
		p95LatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_p95_latency_ms", Help: "P95 latency (ms)"}),
		minLatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_min_latency_ms", Help: "Minimum latency (ms)"}),
		maxLatencyGauge:    prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_max_latency_ms", Help: "Maximum latency (ms)"}),
		totalSentGauge:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_total_sent_bytes", Help: "Total bytes sent"}),
		totalRecvGauge:     prometheus.NewGauge(prometheus.GaugeOpts{Name: "performance_test_total_received_bytes", Help: "Total bytes received"}),
		statusCodeGaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "performance_test_status_code_count",
				Help: "Count of responses by HTTP status code",
			},
			[]string{"status_code"},
		),
	}
}

func (spc *statsPrometheusCollector) copyStatsCollector(s *statsCollector) {
	var durations = make([]float64, 0, 100000)
	if s.durations != nil {
		durations = make([]float64, len(s.durations))
		copy(durations, s.durations)
	}

	var errSet = make(map[string]struct{})
	if s.errSet != nil {
		errSet = make(map[string]struct{}, len(s.errSet))
		for k, v := range s.errSet {
			errSet[k] = v
		}
	}

	var statusCodeSet = make(map[int]int64)
	if s.statusCodeSet != nil {
		statusCodeSet = make(map[int]int64, len(s.statusCodeSet))
		for k, v := range s.statusCodeSet {
			if k == 0 {
				k = -1
			}
			statusCodeSet[k] = v
		}
	}
	spc.statsCollector = &statsCollector{
		durations:      durations,
		errSet:         errSet,
		statusCodeSet:  statusCodeSet,
		totalReqBytes:  s.totalReqBytes,
		totalRespBytes: s.totalRespBytes,
		successCount:   s.successCount,
		errorCount:     s.errorCount,
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Round((float64(len(sorted)) - 1) * p))
	if idx < 0 {
		idx = 0
	} else if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// PushToPrometheus pushes the statistics to a Prometheus.
func (spc *statsPrometheusCollector) PushToPrometheus(ctx context.Context, pushGatewayURL, jobName string, elapsed time.Duration) error {
	totalReq := spc.statsCollector.successCount + spc.statsCollector.errorCount
	qps := 0.0
	if elapsed.Seconds() > 0 {
		qps = float64(spc.statsCollector.successCount) / elapsed.Seconds()
	}

	d := append([]float64(nil), spc.statsCollector.durations...)
	sort.Float64s(d)
	avg, minVal, maxVal := 0.0, 0.0, 0.0
	if len(d) > 0 {
		sum := 0.0
		minVal = d[0]
		maxVal = d[len(d)-1]
		for _, v := range d {
			sum += v
		}
		avg = sum / float64(len(d))
	}

	// set gauges
	spc.totalRequestsGauge.Set(float64(totalReq))
	spc.successGauge.Set(float64(spc.statsCollector.successCount))
	spc.errorGauge.Set(float64(spc.statsCollector.errorCount))
	spc.totalTimeGauge.Set(elapsed.Seconds())
	spc.qpsGauge.Set(qps)
	spc.avgLatencyGauge.Set(avg * 1000)
	spc.p25LatencyGauge.Set(percentile(d, 0.25) * 1000)
	spc.p50LatencyGauge.Set(percentile(d, 0.50) * 1000)
	spc.p95LatencyGauge.Set(percentile(d, 0.95) * 1000)
	spc.minLatencyGauge.Set(minVal * 1000)
	spc.maxLatencyGauge.Set(maxVal * 1000)
	spc.totalSentGauge.Set(float64(spc.statsCollector.totalReqBytes))
	spc.totalRecvGauge.Set(float64(spc.statsCollector.totalRespBytes))

	for code, count := range spc.statsCollector.statusCodeSet {
		spc.statusCodeGaugeVec.WithLabelValues(fmt.Sprintf("%d", code)).Set(float64(count))
	}

	pusher := push.New(pushGatewayURL, jobName).
		Collector(spc.totalRequestsGauge).
		Collector(spc.successGauge).
		Collector(spc.errorGauge).
		Collector(spc.totalTimeGauge).
		Collector(spc.qpsGauge).
		Collector(spc.avgLatencyGauge).
		Collector(spc.p25LatencyGauge).
		Collector(spc.p50LatencyGauge).
		Collector(spc.p95LatencyGauge).
		Collector(spc.minLatencyGauge).
		Collector(spc.maxLatencyGauge).
		Collector(spc.totalSentGauge).
		Collector(spc.totalRecvGauge).
		Collector(spc.statusCodeGaugeVec)

	return pusher.PushContext(ctx)
}

// PushToPrometheusAsync pushes the statistics to a Prometheus asynchronously
func (spc *statsPrometheusCollector) PushToPrometheusAsync(ctx context.Context, pushGatewayURL, jobName string, elapsed time.Duration) {
	go func() {
		_ = spc.PushToPrometheus(ctx, pushGatewayURL, jobName, elapsed)
	}()
}

// PushToServer pushes the statistics data to a custom server
// body is the JSON data of Statistics struct
func (spc *statsPrometheusCollector) PushToServer(ctx context.Context, pushURL string, elapsed time.Duration, httpReqParams *HTTPReqParams, id string, agentID string, status AgentStatus) error {
	statistics := spc.statsCollector.toStatistics(elapsed, spc.statsCollector.successCount+spc.statsCollector.errorCount, httpReqParams)
	statistics.ID = id
	statistics.CreatedAt = time.Now()
	statistics.AgentID = agentID
	statistics.Status = string(status)

	_, err := postWithContext(ctx, pushURL, statistics)
	return err
}

// PushToServerAsync pushes the statistics data to a custom server asynchronously
func (spc *statsPrometheusCollector) PushToServerAsync(ctx context.Context, pushURL string, elapsed time.Duration, httpReqParams *HTTPReqParams, id string, agentID string, status AgentStatus) {
	go func() {
		_ = spc.PushToServer(ctx, pushURL, elapsed, httpReqParams, id, agentID, status)
	}()
}

func postWithContext(ctx context.Context, url string, data *Statistics) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip certificate validation
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return resp, nil
	}

	return resp, fmt.Errorf(`post "%s" failed with status code %d`, url, resp.StatusCode)
}
