package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
)

// PerfTestHTTPCMD creates a new cobra.Command for HTTP/1.1 performance test.
func PerfTestHTTPCMD() *cobra.Command {
	var (
		targetURL string
		method    string
		body      string
		bodyFile  string
		headers   []string

		worker   int
		total    uint64
		duration time.Duration

		out               string
		pushURL           string
		pushInterval      time.Duration
		prometheusJobName string

		// Cluster mode parameters
		clusterEnable   bool
		collectorHost   string
		agentHost       string
		agentID         string
		loopTestSession bool
	)

	//nolint:lll
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run a performance test against HTTP/1.1 APIs",
		Long:  "Run a performance test against HTTP/1.1 APIs.",
		Example: color.HiBlackString(`  # Standalone Mode

    # Default parameters: 3*CPU workers, 5000 requests, GET method
    %s http --url=http://192.168.1.200:8080/user/1

    # Fixed number of requests: 50 workers, 500k requests, GET method
    %s http --worker=50 --total=500000 --url=http://192.168.1.200:8080/user/1

    # Fixed number of requests: 3*CPU workers, 500k requests, POST method with JSON body
    %s http --total=500000 --url=http://192.168.1.200:8080/user --method=POST --body={\"name\":\"Alice\",\"age\":25}

    # Fixed duration: 3*CPU workers, duration 10s, GET method
    %s http --duration=10s --url=http://192.168.1.200:8080/user/1

    # Fixed duration: 3*CPU workers, duration 10s, POST method with JSON body
    %s http --duration=10s --url=http://192.168.1.200:8080/user --method=POST --body={\"name\":\"Alice\",\"age\":25}

    # Fixed number of requests: 3*CPU workers, 500k requests, GET method, push statistics to custom HTTP endpoints every second by default
    %s http --total=500000 --url=http://192.168.1.200:8080/user/1 --push-url=http://localhost:7070/report

    # Fixed duration: 3*CPU workers, duration 10s, GET method, push statistics to prometheus (job=xxx) every second by default
    %s http --duration=10s --url=http://192.168.1.200:8080/user/1 --push-url=http://localhost:9090 --prometheus-job-name=perftest-http


  # Cluster Mode, add parameter '--cluster-enable', '--collector-host, --agent-host', '--agent-id' on the basis of standalone mode

    # Fixed number of requests: 3*CPU workers, 500k requests, GET method, push statistics to collector (master) every second by default
    %s http --total=500000 --url=http://192.168.1.200:8080/user/1 --cluster-enable=true --collector-host=http://192.168.1.10:8888 --agent-host=http://192.168.1.60:6601 --agent-id=agent-1

    # Fixed duration: 3*CPU workers, duration 10s, GET method, push statistics to collector (master) every second by default
    %s http --duration=10s --url=http://192.168.1.200:8080/user/1 --cluster-enable=true --collector-host=http://192.168.1.10:8888 --agent-host=http://192.168.1.60:6601 --agent-id=agent-1`,
			common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix, common.CommandPrefix),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyBytes, headerMap, err := common.ParseHTTPParams(method, headers, body, bodyFile)
			if err != nil {
				return err
			}

			params := &HTTPReqParams{
				URL:     targetURL,
				Method:  method,
				Headers: headerMap,
				Body:    bodyBytes,
				version: "HTTP/1.1",
			}

			p := &PerfTestHTTP{
				ID:                common.NewStringID(),
				Client:            newHTTPClient(worker),
				Params:            params,
				Worker:            worker,
				TotalRequests:     total,
				Duration:          duration,
				PushURL:           pushURL,
				pushInterval:      pushInterval,
				PrometheusJobName: prometheusJobName,

				clusterEnable: clusterEnable,
				agentID:       agentID,
			}
			if err = p.checkParams(); err != nil {
				return err
			}

			ctx := captureSignal()

			if clusterEnable {
				var agent *Agent
				agent, err = NewAgent(agentID, collectorHost, agentHost, targetURL, method)
				if err != nil {
					return err
				}
				agent.runPerformanceTestFn = func(testCtx context.Context, testID string) error {
					p.pushToCollectorURL = fmt.Sprintf("%s/tests/%s/report", collectorHost, testID)
					if prometheusJobName == "" {
						p.PushURL = p.pushToCollectorURL // force push to collector host
					}
					return p.Run(testCtx, duration, out)
				}
				err = agent.Run(ctx, loopTestSession)
			} else {
				err = p.Run(ctx, duration, out)
			}

			if ctx.Err() != nil {
				time.Sleep(500 * time.Millisecond) // wait for all goroutines to exit
			}

			return err
		},
	}

	cmd.Flags().StringVarP(&targetURL, "url", "u", "", "request URL")
	_ = cmd.MarkFlagRequired("url")
	cmd.Flags().StringVarP(&method, "method", "m", "GET", "request method")
	cmd.Flags().StringSliceVarP(&headers, "header", "e", nil, "request headers")
	cmd.Flags().StringVarP(&body, "body", "b", "", "request body (priority higher than --body-file)")
	cmd.Flags().StringVarP(&bodyFile, "body-file", "f", "", "request body file")

	cmd.Flags().IntVarP(&worker, "worker", "w", runtime.NumCPU()*3, "number of workers concurrently processing requests")
	cmd.Flags().Uint64VarP(&total, "total", "t", 5000, "total requests")
	cmd.Flags().DurationVarP(&duration, "duration", "d", 0, "duration of the test, e.g., 10s, 1m (priority higher than --total)")

	cmd.Flags().StringVarP(&out, "out", "o", "", "save statistics to JSON file")
	cmd.Flags().StringVarP(&pushURL, "push-url", "p", "", "push statistics to target URL ")
	cmd.Flags().DurationVarP(&pushInterval, "push-interval", "i", time.Second, "push statistics interval, ranging from 100ms to 10s")
	cmd.Flags().StringVarP(&prometheusJobName, "prometheus-job-name", "j", "", "if not empty, the --push-url parameter value indicates prometheus url")

	// Cluster mode parameters
	cmd.Flags().BoolVar(&clusterEnable, "cluster-enable", false, "enable cluster mode")
	cmd.Flags().StringVar(&collectorHost, "collector-host", "", "collector host, also known as cluster master (e.g. http://192.168.1.10:8888)")
	cmd.Flags().StringVar(&agentHost, "agent-host", "", "callback host for this agent (e.g. http://192.168.1.60:6601)")
	cmd.Flags().StringVar(&agentID, "agent-id", "", "unique id for this agent (e.g. agent-1)")
	cmd.Flags().BoolVar(&loopTestSession, "loop-test-session", false, "if set to true, the agent runs indefinitely until the service is terminated. If false, it terminates after the test completes")

	return cmd
}

func newHTTPClient(worker int) *http.Client {
	if worker <= 0 {
		worker = runtime.NumCPU() * 3
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 15 * time.Second,
			}).DialContext,
			MaxIdleConns:          worker + 10,
			MaxIdleConnsPerHost:   worker,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,                       // default 1 second
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // skip certificate validation
		},
		Timeout: 15 * time.Second,
	}
}
