package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
	"github.com/go-dev-frame/sponge/pkg/conf"
	"github.com/go-dev-frame/sponge/pkg/krand"
)

// PerfTestAgentCMD is the command for performance testing agent
func PerfTestAgentCMD() *cobra.Command {
	var yamlFile string
	var podIP string

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Run an agent to test APIs (supports HTTP/1.1, HTTP/2, and HTTP/3), used in distributed cluster testing",
		Long:  "Run an agent to test APIs (supports HTTP/1.1, HTTP/2, and HTTP/3, used in distributed cluster testing.",
		Example: color.HiBlackString(`  # Running agent
  sponge perftest agent --config=/path/to/agent.yml`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			quitChan := make(chan os.Signal, 1)
			signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

			config := &agentConfig{}

			// define the core logic for restarting services
			restartService := func(cfg *agentConfig) {
				if err := cfg.validate(); err != nil {
					log.Printf("new configuration is invalid, service will not be started: %v", err)
					return
				}

				// create independent context and signal processors for each service instance
				ctx, cancel := context.WithCancel(context.Background())
				currentCfg := *cfg

				manager.mu.Lock()
				manager.currentCfg = &currentCfg
				manager.cancel = cancel
				if podIP != "" {
					manager.currentCfg.AgentHost = fmt.Sprintf("http://%s:%d", podIP, 6601) // default port 6601
				}
				manager.mu.Unlock()

				configBytes, _ := json.Marshal(cfg)
				log.Printf("starting agent service with configuration: %s\n", string(configBytes))

				// runAgent will block until the service is canceled or completed
				err := runAgent(ctx, cfg)
				if err != nil && err != context.Canceled {
					log.Printf("agent service exited with an error: %v", err)
				}
			}

			// callback function for configuration file changes
			onConfigChange := func() {
				log.Println("configuration file change detected.")

				manager.mu.Lock()
				newCfg := *config

				// compare with the current running configuration
				if manager.currentCfg != nil && reflect.DeepEqual(*manager.currentCfg, newCfg) {
					log.Println("configuration content is identical, no restart needed.")
					manager.mu.Unlock()
					return
				}

				// if a service is running, cancel it
				if manager.cancel != nil {
					log.Println("shutting down the previous agent service...")
					manager.cancel()
				}
				manager.mu.Unlock()

				// start the service in the new goroutine
				go restartService(&newCfg)
			}

			// parse YAML file and set listening callbacks
			err := conf.Parse(yamlFile, config, onConfigChange)
			if err != nil {
				return err
			}

			go restartService(config)

			// block until an exit signal is received
			sig := <-quitChan
			log.Printf("received signal: %s. Initiating graceful shutdown...", sig)

			// perform cleaning work
			manager.mu.Lock()
			if manager.cancel != nil {
				log.Println("stopping the current agent service...")
				manager.cancel()
			}
			manager.mu.Unlock()

			// wait for a moment so that the service has time to shut down
			time.Sleep(250 * time.Millisecond)

			log.Println("agent has been shut down.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&yamlFile, "config", "c", "", "yaml config file path")
	_ = cmd.MarkFlagRequired("config")
	cmd.Flags().StringVarP(&podIP, "pod-ip", "p", "", "pod IP address, use in deployment in kubernetes")

	return cmd
}

// global Service Manager, used to hold the current running service status
var manager struct {
	mu         sync.Mutex
	currentCfg *agentConfig
	cancel     context.CancelFunc
}

const (
	protocolHTTP  = "http"
	protocolHTTP2 = "http2"
	protocolHTTP3 = "http3"
	protocolHTTPS = "https"
)

type agentConfig struct {
	// test protocol, support http, http2, http3
	Protocol string `yaml:"protocol"` // default http

	// test target
	TestURL string   `yaml:"testURL"`
	Method  string   `yaml:"method"`
	Body    string   `yaml:"body"`
	Headers []string `yaml:"headers"`

	// test parameters
	Worker   *int          `yaml:"worker"` // default 3 * CPU
	Total    uint64        `yaml:"total"`  // default 5000
	Duration time.Duration `yaml:"duration"`

	// push to target
	PushURL           string        `yaml:"pushURL"`
	AgentPushInterval time.Duration `yaml:"agentPushInterval"` // default 1s
	PrometheusJobName string        `yaml:"prometheusJobName"`

	// cluster mode
	ClusterEnabled  *bool   `yaml:"clusterEnabled"` // default true
	CollectorHost   string  `yaml:"collectorHost"`
	AgentHost       string  `yaml:"agentHost"`
	AgentID         *string `yaml:"agentID"`         // default random string
	LoopTestSession *bool   `yaml:"loopTestSession"` // default true
}

func (a *agentConfig) validate() error { //nolint
	if a.Protocol != protocolHTTP && a.Protocol != protocolHTTP2 && a.Protocol != protocolHTTP3 {
		return fmt.Errorf("invalid 'protocol', only http, http2, http3 are supported")
	}
	if a.TestURL == "" {
		return fmt.Errorf("invalid 'url', required")
	}
	_, err := url.Parse(a.TestURL)
	if err != nil {
		return fmt.Errorf("invalid 'url', %v", err)
	}

	method := strings.ToUpper(a.Method)
	if method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return fmt.Errorf("invalid'method', only GET, POST, PUT, DELETE are supported")
	}
	if a.Worker == nil || *a.Worker <= 0 {
		worker := 3 * runtime.NumCPU()
		a.Worker = &worker
	}
	if a.Total <= 0 && a.Duration <= 0 {
		a.Total = 5000
	}

	if a.AgentID == nil || *a.AgentID == "" {
		agentID := "aid_" + krand.String(krand.R_All, 10)
		a.AgentID = &agentID
	}

	if a.LoopTestSession == nil {
		loopTestSession := true
		a.LoopTestSession = &loopTestSession
	}

	if a.ClusterEnabled == nil {
		clusterEnabled := true
		a.ClusterEnabled = &clusterEnabled
	}
	if *a.ClusterEnabled {
		if a.CollectorHost == "" {
			return fmt.Errorf("invalid 'clusterEnabled', 'collectorHost' is required")
		}
		if a.AgentHost == "" {
			return fmt.Errorf("invalid 'clusterEnabled', 'agentHost' is required")
		}
		if *a.AgentID == "" {
			return fmt.Errorf("invalid 'clusterEnabled', 'agentID' is required")
		}
	}
	if a.AgentPushInterval <= 0 {
		a.AgentPushInterval = 1 * time.Second
	}
	return nil
}

// runAgent initiate the core logic of the agent
func runAgent(ctx context.Context, a *agentConfig) error {
	bodyBytes, headerMap, err := common.ParseHTTPParams(a.Method, a.Headers, a.Body, "")
	if err != nil {
		return err
	}

	params := &HTTPReqParams{
		URL:     a.TestURL,
		Method:  a.Method,
		Headers: headerMap,
		Body:    bodyBytes,
	}

	var httpClient *http.Client
	switch a.Protocol {
	case protocolHTTP:
		params.version = "HTTP/1.1"
		httpClient = newHTTPClient(*a.Worker)
	case protocolHTTP2:
		params.version = "HTTP/2"
		httpClient = newHTTP2Client(*a.Worker)
	case protocolHTTP3:
		params.version = "HTTP/3"
		httpClient = newHTTP3Client(*a.Worker)
	}

	p := &PerfTestHTTP{
		ID:                common.NewStringID(),
		Client:            httpClient,
		Params:            params,
		Worker:            *a.Worker,
		TotalRequests:     a.Total,
		Duration:          a.Duration,
		PushURL:           a.PushURL,
		pushInterval:      a.AgentPushInterval,
		PrometheusJobName: a.PrometheusJobName,

		clusterEnable: *a.ClusterEnabled,
		agentID:       *a.AgentID,
	}
	if err = p.checkParams(); err != nil {
		return err
	}

	if *a.ClusterEnabled {
		var agent *Agent
		agent, err = NewAgent(*a.AgentID, a.CollectorHost, a.AgentHost, a.TestURL, a.Method)
		if err != nil {
			return err
		}
		agent.runPerformanceTestFn = func(testCtx context.Context, testID string) error {
			p.pushToCollectorURL = fmt.Sprintf("%s/tests/%s/report", a.CollectorHost, testID)
			if a.PrometheusJobName == "" {
				p.PushURL = p.pushToCollectorURL
			}
			return p.Run(testCtx, a.Duration, "")
		}
		err = agent.Run(ctx, *a.LoopTestSession)
	} else {
		err = p.Run(ctx, a.Duration, "")
	}

	if ctx.Err() != nil {
		time.Sleep(500 * time.Millisecond) // waiting for all goroutines to exit
	}

	return err
}

// AgentStatus defined agent status
type AgentStatus string

const (
	AgentStatusIdle       AgentStatus = "idle"
	AgentStatusRegistered AgentStatus = "registered"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusFinished   AgentStatus = "finished"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusCanceled   AgentStatus = "canceled"
)

// Agent is the core structure of performance testing services
type Agent struct {
	// config info
	ID            string
	CollectorHost string
	AgentHost     string
	TestURL       string
	TestMethod    string

	// statistics management
	mu         sync.Mutex
	status     AgentStatus
	testID     string
	testCtx    context.Context
	testCancel context.CancelFunc

	startSignal          chan string                                        // receive testID
	runPerformanceTestFn func(testCtx context.Context, testID string) error // performance test function

	listenerPort string
	httpServer   *http.Server
}

// NewAgent creates a new Agent instance
func NewAgent(id, collectorHost, agentHost, testURL, testMethod string) (*Agent, error) {
	if id == "" {
		return nil, fmt.Errorf("invalid agent configuration, 'agent-id' is required")
	}

	if collectorHost == "" {
		return nil, fmt.Errorf("invalid agent configuration, 'collector-host' is required")
	}
	u, err := url.Parse(collectorHost)
	if err != nil {
		return nil, fmt.Errorf("invalid 'collector-host' URL: %v", err)
	}
	if u.Scheme != protocolHTTP && u.Scheme != protocolHTTPS {
		return nil, fmt.Errorf("invalid 'collector-host' URL scheme, only http and https are supported")
	}

	var listenerPort string
	if agentHost == "" {
		return nil, fmt.Errorf("invalid agent configuration, 'agent-host' is required")
	}
	u, err = url.Parse(agentHost)
	if err != nil {
		return nil, fmt.Errorf("invalid 'agent-host' URL: %v, e.g. http://192.168.1.20:9999", err)
	}
	if u.Scheme != protocolHTTP && u.Scheme != protocolHTTPS {
		return nil, fmt.Errorf("invalid 'agent-host' URL scheme, only http and https are supported")
	}
	newAgentHost := strings.TrimSuffix(agentHost, u.Port())
	listenerPort = common.CheckPortInUse(u.Port())
	if listenerPort == "" {
		switch u.Scheme {
		case protocolHTTP:
			listenerPort = "80"
		case protocolHTTPS:
			listenerPort = "443"
		}
	} else {
		newAgentHost += listenerPort
	}
	host := strings.TrimSuffix(u.Host, ":"+u.Port())
	switch host {
	case "localhost", "127.0.0.1", "::1", "[::1]", "0.0.0.0":
		fmt.Printf("\n%s\n\n", color.HiYellowString("[WARNING]: 'agent-host' URL is using a loopback address '%s', "+
			"please ensure that the 'agent-host' URL can be accessed by the collector (master).", host))
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		ID:            id,
		CollectorHost: collectorHost,
		AgentHost:     newAgentHost,
		TestURL:       testURL,
		TestMethod:    testMethod,
		status:        AgentStatusIdle,
		startSignal:   make(chan string, 1),
		testCtx:       ctx,
		testCancel:    cancel,
		listenerPort:  listenerPort,
	}, nil
}

// Run agent main loop
func (a *Agent) Run(ctx context.Context, loop bool) error { //nolint
	// 1. start HTTP listener in the background
	isExit := make(chan bool)
	go func() {
		log.Printf("agent '%s' HTTP service starting on port %s\n\n", a.ID, a.listenerPort)
		err := a.startListener()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("agent listener failed: %v\n", err)
		}
		isExit <- true
	}()

	select {
	case <-isExit:
		return nil
	case <-time.After(100 * time.Millisecond): // waiting for the listener to start
	}

	defer func() {
		// use context with timeout to shut down the server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutting down agent HTTP server error: %v\n", err)
		}
	}()

	// if an interrupt signal is received, cancel the test
	go func() {
		<-ctx.Done()
		a.mu.Lock()
		if a.testCtx.Err() == nil {
			a.testCancel()
		}
		a.mu.Unlock()
	}()

	// 2. register to collector until successful
LOOP:
	ticker := time.NewTicker(5 * time.Second)
	var lastErrMsg string
	var testID string
	var err error
	for {
		// check if the external context has been canceled
		select {
		case <-ctx.Done():
			ticker.Stop()
			return context.Canceled
		default:
		}

		testID, err = a.registerWithCollector()
		if err == nil {
			a.mu.Lock()
			a.testID = testID
			a.status = AgentStatusRegistered
			a.testCtx, a.testCancel = context.WithCancel(context.Background())
			a.mu.Unlock()
			log.Printf("agent '%s' successfully registered for test session '%s', waiting for start signal...\n", a.ID, a.testID)
			break
		}

		errMsg := strings.TrimSuffix(err.Error(), ".")
		if errMsg != lastErrMsg {
			log.Printf("%s, retrying every 5 seconds...\n", errMsg)
			lastErrMsg = errMsg
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			ticker.Stop()
			return context.Canceled
		}
	}
	ticker.Stop()

	var runStatus int32
	pingCtx, pingCancel := context.WithCancel(context.Background())
	pingCancelFn := func() {
		pingCancel()
	}
	go func(runCtx context.Context, agentCancel context.CancelFunc) {
		a.mu.Lock()
		pingURL := fmt.Sprintf("%s/ping/%s?agent_id=%s", a.CollectorHost, testID, a.ID)
		a.mu.Unlock()
		pingTicker := time.NewTicker(5 * time.Second)
		defer pingTicker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-pingCtx.Done():
				return
			case <-pingTicker.C:
				if atomic.LoadInt32(&runStatus) == 1 { // agent is running, no need to ping
					return
				}
				client := &http.Client{Timeout: 3 * time.Second}
				resp, err := client.Post(pingURL, "application/json", nil) //nolint
				pingSuccess := true
				var status int
				if err != nil {
					pingSuccess = false
				} else {
					status = resp.StatusCode
					if resp.StatusCode != http.StatusOK {
						pingSuccess = false
					}
					_ = resp.Body.Close()
				}
				if atomic.LoadInt32(&runStatus) == 1 { // agent is running, no need to ping
					return
				}
				if !pingSuccess {
					agentCancel()
					log.Printf("[testID: %s] collector ping failed, status code: %d\n", testID, status)
					return
				}
			}
		}
	}(a.testCtx, a.testCancel)

	// 3. waiting for start signal or context cancellation
	select {
	case newTestID := <-a.startSignal:
		atomic.StoreInt32(&runStatus, 1)
		if newTestID != testID {
			log.Printf("received start signal for test session (%s), but agent is already running test session (%s), dropping signal\n", newTestID, testID)
			break
		}
		pingCancelFn()
		log.Printf("[testID: %s] beginning performance test\n", testID)
		err = a.runPerformanceTestFn(a.testCtx, testID)
		if err != nil {
			log.Printf("running performance test error: %v\n", err)
			break
		}
		log.Printf("[testID: %s] performance test finished\n\n", testID)
	case <-a.testCtx.Done(): // internal testing context canceled
		atomic.StoreInt32(&runStatus, 1)
	case <-ctx.Done(): // external (primary) context canceled
		atomic.StoreInt32(&runStatus, 1)
		return context.Canceled
	}

	if loop {
		a.mu.Lock()
		a.startSignal = make(chan string, 1)
		a.mu.Unlock()
		if ctx.Err() != nil {
			return context.Canceled
		}
		goto LOOP
	}

	return nil
}

// registerWithCollector register an agent with the Collector
func (a *Agent) registerWithCollector() (string, error) {
	registerURL := a.CollectorHost + "/register"
	agentInfo := AgentInfo{
		ID:       a.ID,
		Callback: a.AgentHost,
		URL:      a.TestURL,
		Method:   a.TestMethod,
	}

	body, err := json.Marshal(agentInfo)
	if err != nil {
		return "", fmt.Errorf("registration failed, marshal agent error: %v", err)
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Post(registerURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("registration failed, %v", err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errorResp struct {
			Error string `json:"error"`
		}
		errStr := string(respBody)
		err = json.Unmarshal(respBody, &errorResp)
		if err == nil {
			errStr = errorResp.Error
		}
		return "", fmt.Errorf("registration failed, %s", errStr)
	}

	var result map[string]string
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("registration failed, %v", err)
	}

	testID, ok := result["testID"]
	if !ok {
		return "", fmt.Errorf("registration failed, testID not found in registration response")
	}

	return testID, nil
}

// startListener start an HTTP server to listen for instructions from the Collector
func (a *Agent) startListener() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", a.handleReady)
	mux.HandleFunc("/start", a.handleStart)
	mux.HandleFunc("/stop", a.handleStop)
	mux.HandleFunc("/cancel", a.handleCancel)
	mux.HandleFunc("/ping", a.handlePing)

	a.httpServer = &http.Server{Addr: ":" + a.listenerPort, Handler: mux}
	err := a.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleReady respond with 200 OK to indicate agent is ready to receive instructions from collector
func (a *Agent) handleReady(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	status := a.status
	currentTestID := a.testID
	currentAgentID := a.ID
	a.mu.Unlock()

	err := getAndCheckID(r, currentTestID, currentAgentID)
	if err != nil {
		log.Printf("readiness check failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status != AgentStatusRegistered {
		err = fmt.Errorf("agent not in registered state, current status: %s", status)
		log.Printf("readiness check failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("[testID: %s] readiness check OK", currentTestID)
	w.WriteHeader(http.StatusOK)
}

// handleStart handles the start signal from the collector. It sets the agent's status to running and
// sends a start signal to the agent's runPerformanceTestFn function.
func (a *Agent) handleStart(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	status := a.status
	currentTestID := a.testID
	currentAgentID := a.ID
	a.mu.Unlock()

	err := getAndCheckID(r, currentTestID, currentAgentID)
	if err != nil {
		log.Printf("readiness check failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status == AgentStatusRegistered {
		a.mu.Lock()
		a.status = AgentStatusRunning
		a.mu.Unlock()
		w.WriteHeader(http.StatusOK)

		defer func() {
			if recover() != nil {
				log.Printf("repeated start signal, dropping signal for test session (%s)\n", currentTestID)
			}
		}()
		select {
		case a.startSignal <- currentTestID:
			close(a.startSignal)
		case <-time.After(2 * time.Second):
			log.Printf("start signal channel full, dropping signal for test session (%s)\n", currentTestID)
		}
	} else {
		log.Printf("received start signal but agent is in unexpected state: %s\n", a.status)
		http.Error(w, "agent not in registered state", http.StatusConflict)
	}
}

// handleStop handles the stop signal from the collector.
func (a *Agent) handleStop(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	status := a.status
	currentTestID := a.testID
	currentAgentID := a.ID
	a.mu.Unlock()

	err := getAndCheckID(r, currentTestID, currentAgentID)
	if err != nil {
		log.Printf("stop test failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//nolint
	if status == AgentStatusRunning || status == AgentStatusRegistered || status == AgentStatusIdle {
		a.mu.Lock()
		a.status = AgentStatusStopped
		a.testCancel()
		a.mu.Unlock()
	} else {
		err = fmt.Errorf("agent not in %s, %s, %s state, current status: %s", AgentStatusRunning, AgentStatusRegistered, AgentStatusIdle, status)
		log.Printf("stop test session (%s) failed: %v\n", currentTestID, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	//log.Printf("[testID: %s] stop test session OK.", currentTestID)
}

// handleCancel handles the cancel signal from the collector.
func (a *Agent) handleCancel(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	status := a.status
	currentTestID := a.testID
	currentAgentID := a.ID
	a.mu.Unlock()

	err := getAndCheckID(r, currentTestID, currentAgentID)
	if err != nil {
		log.Printf("readiness check failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//nolint
	if status == AgentStatusRegistered || status == AgentStatusIdle {
		a.mu.Lock()
		a.status = AgentStatusCanceled
		a.testCancel()
		a.mu.Unlock()
	} else {
		err = fmt.Errorf("agent not in %s, %s state, current status: %s", AgentStatusRegistered, AgentStatusIdle, status)
		log.Printf("cancel test session (%s) failed: %v\n", currentTestID, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	log.Printf("[testID: %s] stop test session OK.", currentTestID)
	w.WriteHeader(http.StatusOK)
}

func (a *Agent) handlePing(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	status := a.status
	currentTestID := a.testID
	currentAgentID := a.ID
	a.mu.Unlock()

	err := getAndCheckID(r, currentTestID, currentAgentID)
	if err != nil {
		log.Printf("ping failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if status != AgentStatusRegistered {
		pingErr := fmt.Sprintf("ping failed, agent (%s) not in registered to test session (%s), current status: %s", currentAgentID, currentTestID, status)
		log.Println(pingErr)
		http.Error(w, pingErr, http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getAndCheckID(r *http.Request, currentTestID string, currentAgentID string) error {
	testID := r.URL.Query().Get("test_id")
	agentID := r.URL.Query().Get("agent_id")
	if currentTestID != testID {
		return fmt.Errorf("invalid test_id, expected: '%s', actual: '%s'", currentTestID, testID)
	}
	if currentAgentID != agentID {
		return fmt.Errorf("invalid agent_id, expected: '%s', actual: '%s'", currentAgentID, agentID)
	}
	return nil
}
