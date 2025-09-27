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
	ID string // performance test ID

	Client *http.Client
	Params *HTTPReqParams

	Worker        int
	TotalRequests uint64
	Duration      time.Duration

	PushURL           string
	PrometheusJobName string
	pushInterval      time.Duration

	agentID            string
	clusterEnable      bool
	pushToCollectorURL string
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

	if p.pushInterval < time.Millisecond*100 || p.pushInterval > time.Second*10 {
		p.pushInterval = time.Second
	}

	return nil
}

// Run the performance test with fixed number of requests or fixed duration.
func (p *PerfTestHTTP) Run(ctx context.Context, duration time.Duration, out string) error {
	var err error
	var stats *Statistics
	if duration > 0 {
		stats, err = p.RunWithFixedDuration(ctx)
	} else {
		stats, err = p.RunWithFixedRequestsNum(ctx)
	}

	if err != nil {
		return err
	}
	if out != "" && stats != nil {
		err = stats.Save(out)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("failed to save statistics to file: %s", err)
		}
		fmt.Printf("\nsave statistics to '%s' successfully\n", out)
	}
	return nil
}

// RunWithFixedRequestsNum implements performance with a fixed number of requests.
func (p *PerfTestHTTP) RunWithFixedRequestsNum(globalCtx context.Context) (*Statistics, error) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint
	defer cancel()
	go func() {
		select {
		case <-globalCtx.Done():
			if ctx.Err() == nil {
				cancel()
			}
		case <-ctx.Done():
			return
		}
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
		go collector.collectAndPush(ctx, resultCh, statsDone, spc, p, start)
	}

	for i := 0; i < p.Worker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				requestOnce(p.Client, p.Params, resultCh)
				bar.Increment()
			}
		}()
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

	var status AgentStatus
	totalTime := time.Since(start)
	if ctx.Err() == nil {
		bar.Finish()
		status = AgentStatusFinished
	} else {
		bar.Stop()
		status = AgentStatusStopped
	}

	statistics, err := collector.printReport(totalTime, p.TotalRequests, p.Params, p.ID)

	if p.PushURL != "" {
		spc.copyStatsCollector(collector)
		pushStatistics(spc, p, totalTime, status)
	}

	return statistics, err
}

// RunWithFixedDuration implements performance with a fixed duration.
func (p *PerfTestHTTP) RunWithFixedDuration(globalCtx context.Context) (*Statistics, error) {
	// Create a context that will be canceled when the duration is over
	ctx, cancel := context.WithTimeout(context.Background(), p.Duration) //nolint
	defer cancel()
	go func() {
		select {
		case <-globalCtx.Done():
			if ctx.Err() == nil {
				cancel()
			}
		case <-ctx.Done():
			return
		}
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Keep sending requests until the context is canceled
			for {
				select {
				case <-ctx.Done():
					return // Exit goroutine when context is canceled
				default:
					requestOnce(p.Client, p.Params, resultCh)
				}
			}
		}()
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

	var status AgentStatus
	if errors.Is(ctx.Err(), context.Canceled) {
		bar.Stop()
		status = AgentStatusStopped
	} else {
		bar.Finish()
		status = AgentStatusFinished
	}

	// The total number of requests is the count of collected results
	totalRequests := collector.successCount + collector.errorCount
	statistics, err := collector.printReport(totalTime, totalRequests, p.Params, p.ID)

	if p.PushURL != "" {
		spc.copyStatsCollector(collector)
		pushStatistics(spc, p, totalTime, status)
	}

	return statistics, err
}

func pushStatistics(spc *statsPrometheusCollector, p *PerfTestHTTP, totalTime time.Duration, status AgentStatus) {
	var err, err2 error
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5) //nolint
	if p.PrometheusJobName == "" {
		err = spc.PushToServer(ctx, p.PushURL, totalTime, p.Params, p.ID, p.agentID, status)
	} else {
		err = spc.PushToPrometheus(ctx, p.PushURL, p.PrometheusJobName, totalTime)
		if p.clusterEnable {
			err2 = spc.PushToServer(ctx, p.pushToCollectorURL, totalTime, p.Params, p.ID, p.agentID, status)
		}
	}
	_, _ = color.New(color.Bold).Println("[Push Statistics]")
	var result = color.GreenString("ok")
	if err != nil {
		result = color.RedString("%v", err)
	}
	fmt.Printf("  • %s\n\n", result)
	if err2 != nil {
		result = color.RedString("push to collector failed: %v", err2)
		fmt.Printf("  • %s\n\n", result)
	}
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
	Headers map[string]string
	Body    []byte

	version string
}

func buildRequest(params *HTTPReqParams) (*http.Request, error) {
	var req *http.Request
	var err error

	reqMethod := strings.ToUpper(params.Method)
	if reqMethod == "POST" || reqMethod == "PUT" || reqMethod == "PATCH" || reqMethod == "DELETE" {
		body := bytes.NewReader(params.Body)
		req, err = http.NewRequest(reqMethod, params.URL, body)
	} else {
		req, err = http.NewRequest(reqMethod, params.URL, nil)
	}

	for k, v := range params.Headers {
		req.Header.Set(k, v)
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

func captureSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM) // handle manual interruption (Ctrl+C)
	go func() {
		<-sigCh
		if ctx.Err() == nil {
			cancel()
		}
	}()
	return ctx
}
