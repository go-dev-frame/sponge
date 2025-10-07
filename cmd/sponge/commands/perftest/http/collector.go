package http

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/go-dev-frame/sponge/cmd/sponge/commands/perftest/common"
	"github.com/go-dev-frame/sponge/pkg/gin/frontend"
)

// PerfTestCollectorCMD is the command for running collector performance test for HTTP API
func PerfTestCollectorCMD() *cobra.Command {
	var (
		port          int
		collectorHost string
		agents        int
	)

	cmd := &cobra.Command{
		Use:   "collector",
		Short: "Run the collector service to manage and coordinate agents. Required for distributed cluster mode",
		Long:  "Run the collector service to manage and coordinate multiple agents in a distributed performance testing cluster. Real time display of aggregated metrics on the testing UI interface.",
		Example: color.HiBlackString(`  # Running the collector service, default listen port is 8888
  %s collector

  # Running the collector service and specify host address
  %s collector --port=8888 --collector-address=http://<ip or domain name>:8888`,
			common.CommandPrefix, common.CommandPrefix),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := NewCollectorServer(port, collectorHost)
			if err != nil {
				return err
			}
			var printHelp func()
			if agents > 0 {
				session := server.createTest(agents)
				printHelp = func() {
					printRegisterHelp(session.TestID, session.ExpectedAgents)
				}
			}

			u := fmt.Sprintf("http://localhost:%d", port)
			if collectorHost != "" {
				u = collectorHost
			}
			go func() {
				_ = openBrowser(u)
			}()

			return server.Run(printHelp)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8888, "collector server port")
	cmd.Flags().StringVarP(&collectorHost, "collector-address", "a", "", "the address where the collector service can be accessed in your browser, e.g. http://<ip or domain name>[:port]")
	cmd.Flags().IntVarP(&agents, "agent_num", "n", 0, "number of agents to test")

	return cmd
}

// PerfTestData test statistics data
type PerfTestData struct {
	ID     string `json:"id"`     // test id
	URL    string `json:"url"`    // test url
	Method string `json:"method"` // test method

	TotalDuration float64 `json:"total_duration"` // unit: s
	TotalRequests int64   `json:"total_requests"`
	SuccessCount  int64   `json:"success_count"`
	ErrorCount    int64   `json:"error_count"`
	QPS           float64 `json:"qps"` // unit: req/sec

	AvgLatency float64 `json:"avg_latency"` // unit: ms
	P25Latency float64 `json:"p25_latency"` // unit: ms
	P50Latency float64 `json:"p50_latency"` // unit: ms
	P95Latency float64 `json:"p95_latency"` // unit: ms
	P99Latency float64 `json:"p99_latency"` // unit: ms
	MaxLatency float64 `json:"max_latency"` // unit: ms
	MinLatency float64 `json:"min_latency"` // unit: ms

	TotalSent     int64 `json:"total_sent"`     // unit: bytes
	TotalReceived int64 `json:"total_received"` // unit: bytes

	StatusCodes map[int]int64 `json:"status_codes"`
	CreatedAt   string        `json:"created_at"`
	Status      string        `json:"status"`   // running, finished, stopped
	AgentID     string        `json:"agent_id"` // agent identify

	Errors []string `json:"errors"` // error details
}

func (d *PerfTestData) printReport() {
	var builder Builder
	builder.WriteStringf("\n====== %s TestID: %s =====\n", time.Now().Format(time.DateTime), d.ID)
	builder.WriteString("====== Distributed Cluster Performance Test Report ======\n\n")

	agentStatusMap := make(map[string][]string)
	isStatusRunning := false
	if err := json.Unmarshal([]byte(d.Status), &agentStatusMap); err == nil {
		builder.WriteString("[Agents]\n")
		for status, agentIDs := range agentStatusMap {
			if !isStatusRunning && status == string(AgentStatusRunning) {
				isStatusRunning = true
			}
			sort.Strings(agentIDs)
			for _, agentID := range agentIDs {
				builder.WriteStringf("  • %-19s%s\n", agentID, status)
			}
		}
		builder.WriteString("\n")
	}

	builder.WriteString("[Requests]\n")
	builder.WriteStringf("  • %-19s%d\n", "Total Requests:", d.TotalRequests)
	if isStatusRunning {
		builder.WriteStringf("  • %-19s%d\n", "Successful:", d.SuccessCount)
		builder.WriteStringf("  • %-19s%d\n", "Failed:", d.ErrorCount)
	} else {
		successStr := fmt.Sprintf("  • %-19s%d", "Successful:", d.SuccessCount)
		failureStr := fmt.Sprintf("  • %-19s%d", "Failed:", d.ErrorCount)
		if d.TotalRequests > 0 {
			if d.ErrorCount > 0 {
				failureStr += " ✗ "
			}
			if d.TotalRequests == d.SuccessCount {
				successStr += " (100%) "
			} else {
				percentage := float64ToString(float64(d.SuccessCount)/float64(d.TotalRequests)*100, 1)
				successStr += fmt.Sprintf(" (%s%%) ", percentage)
			}
		}
		builder.WriteString(successStr + "\n")
		builder.WriteString(failureStr + "\n")
	}
	builder.WriteStringf("  • %-19s%s s\n", "Total Duration:", float64ToStringNoRound(d.TotalDuration))
	builder.WriteStringf("  • %-19s%s req/sec\n\n", "Throughput (QPS):", float64ToStringNoRound(d.QPS))

	builder.WriteString("[Latency]\n")
	builder.WriteStringf("  • %-19s%s ms\n", "Average:", float64ToStringNoRound(d.AvgLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "Minimum:", float64ToStringNoRound(d.MinLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "Maximum:", float64ToStringNoRound(d.MaxLatency))
	builder.WriteStringf("  • %-19s%s ms\n", "P25:", float64ToStringNoRound(d.P25Latency))
	builder.WriteStringf("  • %-19s%s ms\n", "P50:", float64ToStringNoRound(d.P50Latency))
	builder.WriteStringf("  • %-19s%s ms\n\n", "P95:", float64ToStringNoRound(d.P95Latency))
	builder.WriteStringf("  • %-19s%s ms\n\n", "P99:", float64ToStringNoRound(d.P99Latency))

	builder.WriteString("[Data Transfer]\n")
	builder.WriteStringf("  • %-19s%d Bytes\n", "Sent:", d.TotalSent)
	builder.WriteStringf("  • %-19s%d Bytes\n\n", "Received:", d.TotalReceived)

	if len(d.StatusCodes) > 0 {
		builder.WriteString("[Status Codes]\n")
		codes := make([]int, 0, len(d.StatusCodes))
		for code := range d.StatusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			builder.WriteStringf("  • %d: %d\n", code, d.StatusCodes[code])
		}
		builder.WriteString("\n")
	}

	if len(d.Errors) > 0 {
		builder.WriteString("[Error Details]\n")
		for _, errStr := range d.Errors {
			if !isStatusRunning {
				errStr = color.RedString(errStr)
			}
			builder.WriteStringf("  • %s\n", errStr)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("=========================================================\n\n")

	if isStatusRunning {
		restoreCursorPositionAndClear()
		fmt.Println(color.HiBlackString(builder.String()))
	} else {
		fmt.Println(color.HiCyanString(builder.String()))
		log.Printf("test session %s finished.\n\n", d.ID)
	}
}

// AgentInfo store registered agent information
type AgentInfo struct {
	ID       string      `json:"id"`
	Callback string      `json:"callback"`
	URL      string      `json:"url"`
	Method   string      `json:"method"`
	Status   AgentStatus `json:"status"`
}

type TestStatus string

const (
	StatusPending   TestStatus = "pending"
	StatusRunning   TestStatus = "running"
	StatusStopped   TestStatus = "stopped"
	StatusCompleted TestStatus = "completed"
	StatusAborted   TestStatus = "aborted"
)

// TestSession manage all states of a single test
type TestSession struct {
	sync.Mutex
	TestID           string
	Status           TestStatus
	ExpectedAgents   int
	Agents           map[string]*AgentInfo
	TestingReports   map[string]PerfTestData // agentID -> PerfTestData
	FinalReports     map[string]PerfTestData // agentID -> PerfTestData
	AggregatedReport *PerfTestData
	CreatedAt        time.Time
}

func NewTestSession(expectedAgents int) *TestSession {
	return &TestSession{
		TestID:         common.NewStringID(),
		Status:         StatusPending,
		ExpectedAgents: expectedAgents,
		Agents:         make(map[string]*AgentInfo),
		TestingReports: make(map[string]PerfTestData),
		FinalReports:   make(map[string]PerfTestData),
		CreatedAt:      time.Now(),
	}
}

// CollectorServer manage all test sessions
type CollectorServer struct {
	sync.RWMutex
	port          int
	collectorHost string
	tests         map[string]*TestSession // testID -> TestSession
}

func NewCollectorServer(port int, collectorHost string) (*CollectorServer, error) {
	if collectorHost != "" {
		_, err := url.Parse(collectorHost)
		if err != nil {
			return nil, err
		}
	}

	return &CollectorServer{
		port:          port,
		collectorHost: collectorHost,
		tests:         make(map[string]*TestSession),
	}, nil
}

// handleCreateTest create test session and return testID and current number of registered agents
func (s *CollectorServer) handleCreateTest(c *gin.Context) {
	hasPendingSession := false
	agentNum := ""
	testID := ""

	s.RLock()
	// check if there is a pending test session
	for _, session := range s.tests {
		if session.Status == StatusPending {
			hasPendingSession = true
			testID = session.TestID
			agentNum = fmt.Sprintf("%d/%d", len(session.Agents), session.ExpectedAgents)
			break
		}
	}
	s.RUnlock()

	if hasPendingSession {
		c.JSON(http.StatusOK, gin.H{"test_id": testID, "agent_num": agentNum})
		return
	}

	agentsStr := c.Query("agent_num")
	expectedAgents, err := strconv.Atoi(agentsStr)
	if err != nil || expectedAgents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'agents' query parameter, must be a positive integer."})
		return
	}

	session := s.createTest(expectedAgents)
	printRegisterHelp(session.TestID, session.ExpectedAgents)

	c.JSON(http.StatusOK, gin.H{"test_id": session.TestID, "agent_num": fmt.Sprintf("0/%d", expectedAgents)})
}

func (s *CollectorServer) pingAgent(session *TestSession, testID string) {
	if session == nil {
		return
	}

	for {
		time.Sleep(5 * time.Second)

		var agents []*AgentInfo
		session.Lock()
		for _, agent := range session.Agents {
			agents = append(agents, agent)
		}
		session.Unlock()

		if len(agents) == 0 {
			continue
		}

		// check session status
		session.Lock()
		if session.Status != StatusPending {
			session.Unlock()
			return
		}
		session.Unlock()

		if len(agents) == 0 {
			continue
		}
		var wg sync.WaitGroup
		for _, agent := range agents {
			wg.Add(1)
			pingURL := agent.Callback + "/ping" + fmt.Sprintf("?test_id=%s&agent_id=%s", testID, agent.ID)
			go func(agent *AgentInfo, pingURL string) {
				defer wg.Done()
				client := http.Client{Timeout: 3 * time.Second}
				resp, err := client.Post(pingURL, "application/json", nil)
				if err != nil {
					deleteAgent(session, agent)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					deleteAgent(session, agent)
				}
			}(agent, pingURL)
		}
		wg.Wait()
	}
}

func deleteAgent(session *TestSession, agent *AgentInfo) {
	var currentAgents int
	var expectedAgents int
	var testID string
	agentID := agent.ID
	session.Lock()
	delete(session.Agents, agent.ID)
	currentAgents = len(session.Agents)
	expectedAgents = session.ExpectedAgents
	testID = session.TestID
	session.Unlock()
	log.Printf("[testID: %s] agent '%s' unregistered (%d/%d), waiting for other agents to register...\n", testID, agentID, currentAgents, expectedAgents)
}

func (s *CollectorServer) createTest(expectedAgents int) *TestSession {
	session := NewTestSession(expectedAgents)
	s.Lock()
	s.tests[session.TestID] = session
	s.Unlock()

	go s.pingAgent(session, session.TestID) // monitor agent availability

	return session
}

// getSession from request safely get session
func (s *CollectorServer) getSession(c *gin.Context) *TestSession {
	testID := c.Param("testID")
	if testID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing testID in URL path"})
		return nil
	}

	s.RLock()
	session, ok := s.tests[testID]
	s.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "test session not found for ID: " + testID})
		return nil
	}
	return session
}

// handleRegister register agent to test session
func (s *CollectorServer) handleRegister(c *gin.Context) { //nolint
	var agentInfo AgentInfo
	if err := c.ShouldBindJSON(&agentInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse agent JSON: " + err.Error()})
		return
	}
	if agentInfo.ID == "" || agentInfo.Callback == "" || agentInfo.URL == "" || agentInfo.Method == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fields 'id', 'callback', 'url', and 'method' are required"})
		return
	}

	var assignedSession *TestSession

	// 1. find all pending sessions to avoid long time lock on the server.
	s.RLock()
	pendingSessions := make([]*TestSession, 0)
	for _, session := range s.tests {
		if session.Status == StatusPending {
			pendingSessions = append(pendingSessions, session)
		}
	}
	s.RUnlock()

	if len(pendingSessions) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no new test sessions available in collector"})
		return
	}

	// 2. try to find a pending session that matches the agent's URL and Method, and has available slots.
	for _, session := range pendingSessions {
		session.Lock()
		// check if the session is appropriate: pending status, not fully staffed, and at least one agent available for matching
		if session.Status == StatusPending && len(session.Agents) > 0 && len(session.Agents) < session.ExpectedAgents {
			if _, ok := session.Agents[agentInfo.ID]; ok {
				session.Unlock()
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("agent-id (%s) already registered for test session (%s)",
					agentInfo.ID, session.TestID)})
				return
			}

			var firstAgent *AgentInfo
			for _, ag := range session.Agents {
				firstAgent = ag
				break
			}
			if firstAgent != nil && firstAgent.URL == agentInfo.URL && firstAgent.Method == agentInfo.Method {
				session.Unlock()
				assignedSession = session
				break
			}
			session.Unlock()
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("agent '%s' test target (%s %s) doesn't match existing target (%s %s)",
				agentInfo.ID, agentInfo.Method, agentInfo.URL, firstAgent.Method, firstAgent.URL)})
			return
		}
		session.Unlock()
	}

	// 3. if not found, try to find an empty pending session.
	if assignedSession == nil {
		for _, session := range pendingSessions {
			session.Lock()
			if session.Status == StatusPending && len(session.Agents) == 0 {
				assignedSession = session
				session.Unlock()
				break
			}
			session.Unlock()
		}
	}

	// 4. if found, register agent to the session.
	if assignedSession != nil {
		assignedSession.Lock()
		defer assignedSession.Unlock()

		// check if the selected session is still available
		if assignedSession.Status != StatusPending {
			c.JSON(http.StatusConflict, gin.H{"error": "the selected session is no longer available"})
			return
		}

		if len(assignedSession.Agents) >= assignedSession.ExpectedAgents {
			c.JSON(http.StatusConflict, gin.H{"error": "the selected session is now full"})
			return
		}

		if _, ok := assignedSession.Agents[agentInfo.ID]; ok {
			// if agent already registered, return success.
			c.JSON(http.StatusOK, gin.H{"agentID": agentInfo.ID, "testID": assignedSession.TestID, "message": "Agent already registered"})
			return
		}

		assignedSession.Agents[agentInfo.ID] = &agentInfo

		currentAgents := len(assignedSession.Agents)
		if currentAgents < assignedSession.ExpectedAgents {
			log.Printf("[testID: %s] agent '%s' registered (%d/%d), waiting for other agents to register...\n", assignedSession.TestID, agentInfo.ID, currentAgents, assignedSession.ExpectedAgents)
		} else {
			log.Printf("[testID: %s] agent '%s' registered (%d/%d)\n", assignedSession.TestID, agentInfo.ID, currentAgents, assignedSession.ExpectedAgents)
		}

		c.JSON(http.StatusOK, gin.H{"agentID": agentInfo.ID, "testID": assignedSession.TestID})

		// 5. check if all agents have been registered.
		if currentAgents == assignedSession.ExpectedAgents {
			log.Printf("[testID: %s] all %d agents have been registered, start checking if all agents are ready for testing.", assignedSession.TestID, assignedSession.ExpectedAgents)
			go s.coordinateTestStart(assignedSession)
		}
		return
	}

	// 6. if no available session found, return no available session error.
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no suitable test session available at the moment, need to create new test session first."})
}

// coordinateTestStart handle test start signal, check if all agents are ready, broadcast start signal to all agents.
func (s *CollectorServer) coordinateTestStart(session *TestSession) {
	log.Printf("[testID: %s] starting readiness check for all agents...\n", session.TestID)

	if s.performReadinessCheck(session) {
		return
	}

	for {
		time.Sleep(5 * time.Second)
		if s.performReadinessCheck(session) {
			return
		}
	}
}

func (s *CollectorServer) performReadinessCheck(session *TestSession) bool {
	allReady, err := s.checkAllAgentsReady(session)
	if err != nil {
		log.Printf("[testID: %s] failed to check all agents readiness: %s", session.TestID, err.Error())
		return true
	}

	if allReady {
		log.Printf("[testID: %s] all agents are ready. Broadcasting start signal.", session.TestID)
		s.broadcastSignal(session, "/start")
		return true
	}

	log.Printf("[testID: %s] not all agents are ready, retrying in 5 seconds...", session.TestID)
	return false // return false to continue the loop
}

// handleReport receive report from agents, update test session and aggregate reports.
func (s *CollectorServer) handleReport(c *gin.Context) {
	session := s.getSession(c)
	if session == nil {
		return
	}

	var report PerfTestData
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse report JSON: " + err.Error()})
		return
	}

	session.Lock()
	defer session.Unlock()

	// handle differently based on the report status
	switch AgentStatus(report.Status) {
	case AgentStatusFinished, AgentStatusStopped:
		if _, ok := session.FinalReports[report.AgentID]; !ok {
			session.FinalReports[report.AgentID] = report
		}
		// check if all agents have finished testing and final reports have been received.
		if len(session.FinalReports) == session.ExpectedAgents && session.Status != StatusCompleted {
			// aggregate the final report
			session.AggregatedReport = s.aggregateReports(session.TestID, session.FinalReports)
			if AgentStatus(report.Status) == AgentStatusFinished {
				session.Status = StatusCompleted
			} else {
				session.Status = StatusStopped
			}
		}
	case AgentStatusRunning:
		session.TestingReports[report.AgentID] = report
		// aggregate all current 'testing' reports and update aggregated data in real-time
		session.AggregatedReport = s.aggregateReports(session.TestID, session.TestingReports)
	}

	if session.AggregatedReport != nil {
		session.AggregatedReport.printReport()
		if session.Status == StatusCompleted || session.Status == StatusStopped {
			printCreateTestHelp()
		}
	}

	c.Status(http.StatusOK)
}

// handleStopTest stop current test session, broadcast /stop signal to all agents and wait for a response.
func (s *CollectorServer) handleStopTest(c *gin.Context) {
	session := s.getSession(c)
	if session == nil {
		return
	}

	session.Lock()
	defer session.Unlock()

	// allow stopping test in Running or Pending state
	if session.Status != StatusRunning && session.Status != StatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "test is not in a running or pending state, cannot stop."})
		return
	}

	agentIDs := make([]string, 0, len(session.Agents))
	for agentID := range session.Agents {
		agentIDs = append(agentIDs, agentID)
	}

	if session.Status == StatusPending {
		log.Printf("[testID: %s] received stop request for pending session, aborting test.", session.TestID)
		go s.broadcastSignal(session, "/cancel") // broadcast cancel signal to all agents
	} else {
		log.Printf("[testID: %s] received stop request, broadcasting stop signal.", session.TestID)
		go s.broadcastSignal(session, "/stop") // broadcast stop signal to all agents
	}

	if len(agentIDs) > 0 {
		c.String(http.StatusOK, fmt.Sprintf("stop test session done, signal broadcasted to agents [%s] .", strings.Join(agentIDs, ", ")))
	} else {
		c.String(http.StatusOK, "stop test done.")
	}
}

// handleGetReport get test report
func (s *CollectorServer) handleGetReport(c *gin.Context) {
	session := s.getSession(c)
	if session == nil {
		return
	}

	session.Lock()
	data := gin.H{
		"status":            session.Status,
		"report":            struct{}{},
		"registered_agents": len(session.Agents),
		"expected_agents":   session.ExpectedAgents,
	}
	if session.AggregatedReport != nil {
		data["report"] = session.AggregatedReport
	}
	session.Unlock()

	c.JSON(http.StatusOK, data)
}

func (s *CollectorServer) handlePing(c *gin.Context) {
	session := s.getSession(c)
	if session == nil {
		return
	}

	if session.Status != StatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("test session (%s) is not in a pending state, current state is %s",
			session.TestID, session.Status)})
		return
	}

	agentID := c.Query("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing agent_id query parameter"})
		return
	}

	var isExist bool
	var testID string
	session.Lock()
	testID = session.TestID
	_, isExist = session.Agents[agentID]
	session.Unlock()

	if !isExist {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("not found agent '%s' in test session '%s'", agentID, testID)})
		return
	}
	c.JSON(http.StatusOK, nil)
}

// runAgentTaskPool creates a work pool to concurrently process tasks for all agents.
// It will dynamically adjust the number of workers based on the number of agents and CPU cores.
//
// Parameters:
//
// Agents: List of agents that need to be processed.
// Task: A function that defines specific operations to be performed on a single agent.
func runAgentTaskPool(agents []*AgentInfo, task func(agent *AgentInfo)) {
	if len(agents) == 0 {
		return
	}

	// calculate the number of workers
	maxWorkers := 3 * runtime.NumCPU()
	numWorkers := len(agents)
	if numWorkers > maxWorkers {
		numWorkers = maxWorkers
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	jobs := make(chan *AgentInfo, numWorkers)

	// run worker
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[worker %d] panic recovered: %v", workerID, r)
				}
			}()

			for agent := range jobs {
				task(agent) // run task
			}
		}(w)
	}

	// distribute tasks
	for _, agent := range agents {
		jobs <- agent
	}
	close(jobs)

	wg.Wait()
}

// checkAllAgentsReady send /ready signal to all agents and wait for a response.
func (s *CollectorServer) checkAllAgentsReady(session *TestSession) (bool, error) {
	session.Lock()
	testID := session.TestID
	expectedAgents := session.ExpectedAgents
	agentsToCheck := make([]*AgentInfo, 0, len(session.Agents))
	for _, agent := range session.Agents {
		agentsToCheck = append(agentsToCheck, agent)
	}
	session.Unlock()

	if len(agentsToCheck) == 0 {
		return false, errors.New("no agent to check")
	} else if len(agentsToCheck) != expectedAgents {
		return false, fmt.Errorf("expected %d agents, but only %d registered", expectedAgents, len(agentsToCheck))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := make(chan bool, len(agentsToCheck))

	// define specific tasks to be executed for each agent
	checkTask := func(a *AgentInfo) {
		callBackURL := a.Callback + "/ready" + fmt.Sprintf("?test_id=%s&agent_id=%s", testID, a.ID)
		req, _ := http.NewRequestWithContext(ctx, "POST", callBackURL, nil)
		req.Header.Set("Content-Type", "application/json")
		client := http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[testID: %s] agent '%s' readiness check failed: %v", testID, a.ID, err)
			results <- false
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			results <- true
		} else {
			log.Printf("[testID: %s] agent '%s' is not ready (status: %s)", testID, a.ID, resp.Status)
			results <- false
		}
	}

	// execute tasks using a work pool
	go func() {
		runAgentTaskPool(agentsToCheck, checkTask)
		close(results)
	}()

	// collect all results
	allReady := true
	for i := 0; i < len(agentsToCheck); i++ {
		select {
		case ok := <-results:
			allReady = allReady && ok
		case <-ctx.Done():
			log.Printf("[testID: %s] check all agents ready timed out, not all agents responded.", testID)
		}
	}

	return allReady, nil
}

// broadcastSignal broadcast signal to all agents in the session, wait for a response.
func (s *CollectorServer) broadcastSignal(session *TestSession, path string) {
	session.Lock()
	switch path {
	case "/start":
		session.Status = StatusRunning
	case "/stop":
		session.Status = StatusStopped
	case "/cancel":
		session.Status = StatusAborted
	}
	agentsToSignal := make([]*AgentInfo, 0, len(session.Agents))
	for _, agent := range session.Agents {
		// in stopping state, only signal agents that haven't submitted final reports
		if path == "/stop" {
			if _, finished := session.FinalReports[agent.ID]; !finished {
				agentsToSignal = append(agentsToSignal, agent)
			}
		} else {
			agentsToSignal = append(agentsToSignal, agent)
		}
	}
	session.Unlock()

	// define specific tasks to be executed for each agent
	signalTask := func(a *AgentInfo) {
		agentURL := a.Callback + path + fmt.Sprintf("?test_id=%s&agent_id=%s", session.TestID, a.ID) // path is /start, /stop or /cancel
		log.Printf("[testID: %s] sending signal '%s' to agent '%s'\n", session.TestID, path, a.ID)
		client := &http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := client.Post(agentURL, "application/json", nil)
		if err != nil {
			log.Printf("[testID: %s] sending signal to agent '%s' error: %v", session.TestID, a.ID, err)
			return
		}
		defer resp.Body.Close()
	}

	// execute tasks using a work pool
	runAgentTaskPool(agentsToSignal, signalTask)

	if len(agentsToSignal) > 0 {
		setCursorPosition()
	}

	if path == "/stop" || path == "/cancel" {
		fmt.Println()
		printCreateTestHelp()
	}
}

// aggregateReports aggregate reports from all agents in the given map and return the aggregated report.
func (s *CollectorServer) aggregateReports(id string, reports map[string]PerfTestData) *PerfTestData {
	if len(reports) == 0 {
		return nil
	}

	var (
		aggReport = &PerfTestData{
			StatusCodes: make(map[int]int64),
			Errors:      make([]string, 0),
		}

		totalWeightedLatency                                   float64
		p25Latencies, p50Latencies, p95Latencies, p99Latencies = []float64{}, []float64{}, []float64{}, []float64{}
		maxDuration                                            float64

		errMap    = make(map[string][]string) // error message --> agent IDs
		isFirst   = true
		agentIDs  = make([]string, 0, len(reports))
		statusMap = make(map[string][]string) // status --> agent IDs
	)

	for agentID, report := range reports {
		if isFirst {
			aggReport.URL = report.URL
			aggReport.Method = report.Method
			aggReport.MinLatency = report.MinLatency
			isFirst = false
		}

		aggReport.TotalRequests += report.TotalRequests
		aggReport.SuccessCount += report.SuccessCount
		aggReport.ErrorCount += report.ErrorCount
		aggReport.TotalSent += report.TotalSent
		aggReport.TotalReceived += report.TotalReceived

		aggReport.QPS += report.QPS

		if report.TotalDuration > maxDuration {
			maxDuration = report.TotalDuration
		}
		totalWeightedLatency += report.AvgLatency * float64(report.TotalRequests)

		for code, count := range report.StatusCodes {
			aggReport.StatusCodes[code] += count
		}

		if report.MaxLatency > aggReport.MaxLatency {
			aggReport.MaxLatency = report.MaxLatency
		}

		if report.MinLatency < aggReport.MinLatency {
			aggReport.MinLatency = report.MinLatency
		}

		p25Latencies = append(p25Latencies, report.P25Latency)
		p50Latencies = append(p50Latencies, report.P50Latency)
		p95Latencies = append(p95Latencies, report.P95Latency)
		p99Latencies = append(p99Latencies, report.P99Latency)

		for _, errs := range report.Errors {
			if _, ok := errMap[errs]; !ok {
				errMap[errs] = []string{agentID}
			} else {
				errMap[errs] = append(errMap[errs], agentID)
			}
		}

		agentIDs = append(agentIDs, agentID)

		if v, ok := statusMap[report.Status]; !ok {
			statusMap[report.Status] = []string{agentID}
		} else {
			statusMap[report.Status] = append(v, agentID)
		}
	}

	// to simplify the operation, the average value is used here to calculate p25, p50, p95, and p99
	aggReport.P25Latency = averageLatency(p25Latencies)
	aggReport.P50Latency = averageLatency(p50Latencies)
	aggReport.P95Latency = averageLatency(p95Latencies)
	aggReport.P99Latency = averageLatency(p99Latencies)

	aggReport.TotalDuration = maxDuration
	if aggReport.TotalRequests > 0 {
		aggReport.AvgLatency = math.Round(totalWeightedLatency/float64(aggReport.TotalRequests)*100) / 100
	}

	for errStr, aids := range errMap {
		aggReport.Errors = append(aggReport.Errors, fmt.Sprintf("%s (from agents %s)", errStr, strings.Join(aids, ", ")))
	}

	aggReport.ID = id
	aggReport.QPS = math.Round(aggReport.QPS*10) / 10
	aggReport.CreatedAt = time.Now().Format(time.RFC3339)
	aggReport.AgentID = strings.Join(agentIDs, ", ")
	status, _ := json.Marshal(statusMap)
	aggReport.Status = string(status)

	return aggReport
}

func averageLatency(latencies []float64) float64 {
	if len(latencies) == 0 {
		return 0
	}

	var latencyCount float64
	for _, latency := range latencies {
		latencyCount += latency
	}
	avg := latencyCount / float64(len(latencies))

	return math.Round(avg*100) / 100
}

//go:embed perftest
var staticFS embed.FS

// Run collector server.
func (s *CollectorServer) Run(printHelp func()) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	//prof.Register(router, prof.WithIOWaitTime())

	router.POST("/tests", s.handleCreateTest)
	router.POST("/register", s.handleRegister)
	testGroup := router.Group("/tests/:testID")
	{
		testGroup.POST("/report", s.handleReport)
		testGroup.GET("/report", s.handleGetReport)
		testGroup.POST("/stop", s.handleStopTest)
	}
	router.POST("/ping/:testID", s.handlePing)

	host := s.collectorHost
	f := frontend.New("perftest",
		frontend.WithEmbedFS(staticFS),
		frontend.WithHandleContent(func(content []byte) []byte {
			if host != "" {
				return bytes.ReplaceAll(content, []byte("http://localhost:8888"), []byte(host))
			}
			return content
		}, "appConfig.js"),
	)
	err := f.SetRouter(router)
	if err != nil {
		panic(err)
	}

	// "/" and "/index.html" -> "/perftest/index.html"
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/perftest/index.html")
	})
	router.GET("/index.html", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/perftest/index.html")
	})

	server := &http.Server{
		Addr:         ":" + strconv.Itoa(s.port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil { //nolint
			log.Printf("could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Printf("collector server starting on port %d\n", s.port)
	u := fmt.Sprintf("http://<IP or domain>:%d", s.port)
	if s.collectorHost != "" {
		u = s.collectorHost
	}
	fmt.Println(color.HiBlackString("[Tip]: access '%s' in the browser to enter the testing interface", u))
	fmt.Println()

	if printHelp == nil {
		printCreateTestHelp()
	} else {
		printHelp()
	}

	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %v", err)
	}

	<-done
	log.Println("server stopped")
	return nil
}

func setCursorPosition() {
	fmt.Print("\033[s")
}

func restoreCursorPositionAndClear() {
	fmt.Print("\033[u\033[J")
}

func printCreateTestHelp() {
	log.Printf("waiting for create a new test session...\n\n")
}

func printRegisterHelp(testID string, agentNum int) {
	log.Printf("created new test session: testID=%s, expectedAgents=%d\n", testID, agentNum)
	log.Printf("waiting for agents to register...\n%s\n\n",
		color.HiBlackString("[Tip]: run command 'sponge perftest agent -c agent.yml' to register agents, "+
			"agent.yml file reference: https://github.com/go-dev-frame/sponge/blob/main/cmd/sponge/commands/perftest/agent.yml"))
}

type Builder struct {
	strings.Builder
}

func (b *Builder) WriteString(s string) {
	b.Builder.WriteString(s)
}

func (b *Builder) WriteStringf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(b, format, args...)
}

func openBrowser(visitURL string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}

	args = append(args, visitURL)
	return exec.Command(cmd, args...).Start()
}
