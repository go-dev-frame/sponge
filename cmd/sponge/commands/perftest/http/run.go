package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
)

// PerfTestHTTP performance test parameters for HTTP
type PerfTestHTTP struct {
	ID int64 // performance test ID

	Client *http.Client
	Params *HTTPReqParams

	Worker        int
	TotalRequests uint64
	Duration      time.Duration

	PushURL           string
	PrometheusJobName string
}

func (p *PerfTestHTTP) checkParams() error {
	if p.Worker == 0 {
		return fmt.Errorf("'--worker' number must be greater than 0")
	}

	if p.TotalRequests == 0 && p.Duration == 0 {
		return errors.New("'--duration' and '--total' must be set one of them")
	}

	if p.PrometheusJobName != "" && p.PushURL == "" {
		return errors.New("'--prometheus-job-name' has already been set, '--push-url' must be set")
	}

	return nil
}

// RunWithFixedRequestsNum implements performance with a fixed number of requests.
func (p *PerfTestHTTP) RunWithFixedRequestsNum() (*Statistics, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	var wg sync.WaitGroup

	jobs := make(chan struct{}, p.Worker)
	resultCh := make(chan Result, p.Worker*3)
	statsDone := make(chan struct{})
	bar := &common.Bar{}

	collector := &statsCollector{
		durations: make([]float64, 0, p.TotalRequests),
	}
	var spc *statsPrometheusCollector
	var start time.Time

	// The collector counts the results of each request, closes the statsDone
	// channel, and notifies the main thread of the end
	if p.PushURL == "" {
		go collector.collect(resultCh, statsDone)
	} else {
		if p.PrometheusJobName == "" {
			spc = &statsPrometheusCollector{}
		} else {
			spc = newStatsPrometheusCollector()
		}
		//go collector.collectAndPush(ctx, resultCh, statsDone, spc, b.PushURL, b.PrometheusJobName, b.Params, start)
		go collector.collectAndPush(ctx, resultCh, statsDone, spc, p, start)
	}

	for i := 0; i < p.Worker; i++ {
		wg.Go(func() {
			for range jobs {
				requestOnce(p.Client, p.Params, resultCh)
				bar.Increment()
			}
		})
	}

	start = time.Now()
	bar = common.NewBar(int64(p.TotalRequests), start)
	// Distribute tasks and listen for context cancellation events
loop:
	for i := uint64(0); i < p.TotalRequests; i++ {
		select {
		case jobs <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
	}

	close(jobs)

	wg.Wait()
	close(resultCh)

	<-statsDone

	totalTime := time.Since(start)
	if ctx.Err() == nil {
		bar.Finish()
	} else {
		fmt.Println()
	}

	statistics, err := collector.printReport(totalTime, p.TotalRequests, p.Params)

	if p.PushURL != "" {
		spc.copyStatsCollector(collector)
		pushStatistics(spc, p, totalTime)
	}

	return statistics, err
}

// RunWithFixedDuration implements performance with a fixed duration.
func (p *PerfTestHTTP) RunWithFixedDuration() (*Statistics, error) {
	// Create a context that will be canceled when the duration is over
	ctx, cancel := context.WithTimeout(context.Background(), p.Duration)
	defer cancel()

	// Handle manual interruption (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	var wg sync.WaitGroup
	resultCh := make(chan Result, p.Worker*3)
	statsDone := make(chan struct{})

	// Since the total number of requests is unknown, we initialize with a reasonable capacity to reduce reallocation's.
	collector := &statsCollector{
		durations: make([]float64, 0, 100000),
	}
	var spc *statsPrometheusCollector
	var start time.Time

	if p.PushURL == "" {
		go collector.collect(resultCh, statsDone)
	} else {
		if p.PrometheusJobName == "" {
			spc = &statsPrometheusCollector{}
		} else {
			spc = newStatsPrometheusCollector()
		}
		go collector.collectAndPush(ctx, resultCh, statsDone, spc, p, start)
	}

	// Start workers
	for i := 0; i < p.Worker; i++ {
		wg.Go(func() {
			// Keep sending requests until the context is canceled
			for {
				select {
				case <-ctx.Done():
					return
				default:
					requestOnce(p.Client, p.Params, resultCh)
				}
			}
		})
	}

	start = time.Now()
	bar := common.NewTimeBar(p.Duration)
	bar.Start()

	<-ctx.Done() // Wait for the timeout or a signal

	totalTime := time.Since(start)

	// Wait for all workers to finish their current request
	wg.Wait()
	// Close the result channel to signal the collector that no more results will be sent
	close(resultCh)
	// Wait for the collector to process all the results in the channel
	<-statsDone

	if errors.Is(ctx.Err(), context.Canceled) {
		fmt.Println()
	} else {
		bar.Finish()
	}

	// The total number of requests is the count of collected results
	totalRequests := collector.successCount + collector.errorCount
	statistics, err := collector.printReport(totalTime, totalRequests, p.Params)

	if p.PushURL != "" {
		spc.copyStatsCollector(collector)
		pushStatistics(spc, p, totalTime)
	}

	return statistics, err
}

func pushStatistics(spc *statsPrometheusCollector, b *PerfTestHTTP, totalTime time.Duration) {
	var err error
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3) //nolint
	if b.PrometheusJobName == "" {
		err = spc.PushToServer(ctx, b.PushURL, totalTime, b.Params, b.ID)
	} else {
		err = spc.PushToPrometheus(ctx, b.PushURL, b.PrometheusJobName, totalTime)
	}
	_, _ = color.New(color.Bold).Println("[Push Statistics]")
	var result = color.GreenString("ok")
	if err != nil {
		result = color.RedString("%v", err)
	}
	fmt.Printf("  • %s\n", result)
}

// -------------------------------------------------------------------------------------------

// Result record the results of the request
type Result struct {
	Duration   time.Duration
	ReqSize    int64
	RespSize   int64
	StatusCode int
	Err        error
}

type HTTPReqParams struct {
	URL     string
	Method  string
	Headers []string
	Body    string

	version string
}

func buildRequest(params *HTTPReqParams) (*http.Request, error) {
	var req *http.Request
	var err error

	reqMethod := strings.ToUpper(params.Method)
	if reqMethod == "POST" || reqMethod == "PUT" || reqMethod == "PATCH" || reqMethod == "DELETE" {
		body := bytes.NewReader([]byte(params.Body))
		req, err = http.NewRequest(reqMethod, params.URL, body)
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	} else {
		req, err = http.NewRequest(reqMethod, params.URL, nil)
	}

	for _, h := range params.Headers {
		kvs := strings.SplitN(h, ":", 2)
		if len(kvs) == 2 {
			req.Header.Set(strings.TrimSpace(kvs[0]), strings.TrimSpace(kvs[1]))
		}
	}

	return req, err
}

func requestOnce(client *http.Client, params *HTTPReqParams, ch chan<- Result) {
	req, err := buildRequest(params)
	if err != nil {
		ch <- Result{Err: err}
		return
	}

	var reqSize int64
	if req.Body != nil {
		reqSize = req.ContentLength
		if seeker, ok := req.Body.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
	}

	begin := time.Now()
	resp, err := client.Do(req)
	if err != nil { // Check for request-level errors (e.g. timeout, DNS resolution failure)
		duration := time.Since(begin)
		ch <- Result{
			Duration: duration,
			ReqSize:  reqSize,
			RespSize: 0,
			Err:      err,
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 { // Check if the response status code is not 2xx
		respSize, _ := io.Copy(io.Discard, resp.Body)
		duration := time.Since(begin)
		ch <- Result{
			Duration:   duration,
			ReqSize:    reqSize,
			RespSize:   respSize,
			StatusCode: resp.StatusCode,
			Err:        fmt.Errorf("%s, [%s] %s", http.StatusText(resp.StatusCode), req.Method, req.URL.String()),
		}
		return
	}

	respSize, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		duration := time.Since(begin)
		ch <- Result{
			Duration:   duration,
			ReqSize:    reqSize,
			RespSize:   respSize,
			StatusCode: resp.StatusCode,
			Err:        err,
		}
		return
	}

	duration := time.Since(begin)
	ch <- Result{
		Duration:   duration,
		ReqSize:    reqSize,
		RespSize:   respSize,
		StatusCode: resp.StatusCode,
	}
}
