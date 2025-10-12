package sse

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
	"go.uber.org/zap"
)

type ClientOption func(*clientOptions)

type clientOptions struct {
	headers               map[string]string
	zapLogger             *zap.Logger
	reconnectTimeInterval time.Duration
}

func defaultClientOptions() *clientOptions {
	logger, _ := zap.NewProduction()
	return &clientOptions{
		reconnectTimeInterval: 2 * time.Second,
		zapLogger:             logger,
	}
}

func (o *clientOptions) apply(opts ...ClientOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithClientHeaders set HTTP headers
func WithClientHeaders(headers map[string]string) ClientOption {
	return func(o *clientOptions) {
		o.headers = headers
	}
}

// WithClientLogger set logger
func WithClientLogger(logger *zap.Logger) ClientOption {
	return func(o *clientOptions) {
		if logger != nil {
			o.zapLogger = logger
		}
	}
}

// WithClientReconnectTimeInterval set reconnect time interval
func WithClientReconnectTimeInterval(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.reconnectTimeInterval = d
	}
}

// -------------------------------------------------------------------------------------------

// EventCallback event callback
type EventCallback func(event *Event)

// SSEClient sse client
type SSEClient struct {
	url         string
	client      *http.Client
	callbacks   map[string]EventCallback
	lastEventID string
	mu          sync.RWMutex
	connected   bool
	stopCh      chan struct{}

	headers   map[string]string
	zapLogger *zap.Logger
	// default is 2 seconds, backoff time interval will double after each retry
	reconnectTimeInterval time.Duration
}

// NewClient create a new sse client
func NewClient(url string, opts ...ClientOption) *SSEClient {
	o := defaultClientOptions()
	o.apply(opts...)

	return &SSEClient{
		url: url,
		client: &http.Client{
			Timeout: 0, // no timeout
		},
		callbacks: make(map[string]EventCallback),
		stopCh:    make(chan struct{}),

		headers:               o.headers,
		reconnectTimeInterval: o.reconnectTimeInterval,
		zapLogger:             o.zapLogger,
	}
}

// OnEvent register event callback
func (c *SSEClient) OnEvent(eventType string, callback EventCallback) {
	c.mu.Lock()
	c.callbacks[eventType] = callback
	c.mu.Unlock()
}

// Connect to server
func (c *SSEClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected {
		return fmt.Errorf("already connected")
	}

	go c.connectServer()
	return nil
}

// Disconnect to server
func (c *SSEClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	c.zapLogger.Info("exit connect SSE service")
	close(c.stopCh)
}

// Wait returns a channel that will be closed when the client is disconnected.
func (c *SSEClient) Wait() <-chan struct{} {
	return c.stopCh
}

// GetConnectStatus get connect status
func (c *SSEClient) GetConnectStatus() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *SSEClient) setConnectStatus(st bool) {
	c.mu.Lock()
	c.connected = st
	c.mu.Unlock()
}

// support auto reconnect, retry strategy, exponential backoff algorithm, max retries is 5 times, max backoff is 30 seconds
func (c *SSEClient) connectServer() {
	s := &retryStrategy{
		backoff:    c.reconnectTimeInterval,
		maxBackoff: 30 * time.Second,
		retryCount: 0,
		maxRetries: 5,
	}
	c.setConnectStatus(true)

	for {
		err := c.stream(s)
		if err != nil {
			if strings.Contains(err.Error(), noConnectionErrText) {
				c.setConnectStatus(false)
				isExits := s.retry()
				c.zapLogger.Warn("connect SSE service failed", zap.Error(err), zap.String("retry_strategy",
					fmt.Sprintf("reconnect in %v (try %d/%d)", s.backoff, s.retryCount, s.maxRetries)))
				if isExits {
					c.Disconnect()
					return
				}
			} else {
				s.reset(c.reconnectTimeInterval)
			}
		}

		select {
		case <-c.stopCh:
			return
		case <-time.After(s.backoff): // wait for next retry
		}
	}
}

func (c *SSEClient) stream(s *retryStrategy) error {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return err
	}

	// add request headers
	c.mu.RLock()
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}
	c.mu.RUnlock()

	// send request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.setConnectStatus(true)
	s.reset(c.reconnectTimeInterval)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var eventType, eventID, data string
	var eventData []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			// blank lines indicate the end of the event, process the event
			if len(eventData) > 0 {
				err = c.processEvent(eventType, eventID, strings.Join(eventData, "\n"))
				if err != nil {
					c.zapLogger.Warn("process event error", zap.Error(err))
					continue
				}
				eventType, eventID, data = "", "", ""
				eventData = nil
			}
			continue
		}

		// parse event fields
		if strings.HasPrefix(line, ":") {
			continue
		} else if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "id:") {
			eventID = strings.TrimSpace(line[3:])
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(line[5:])
			eventData = append(eventData, data)
		}

		if eventType == "close" {
			c.Disconnect()
			return fmt.Errorf("close event received")
		}

		select {
		case <-c.stopCh:
			return nil
		default:
		}
	}
}

func (c *SSEClient) processEvent(eventType, eventID, data string) error {
	if eventID != "" {
		c.mu.Lock()
		c.lastEventID = eventID
		c.mu.Unlock()
	}

	if eventType == "" {
		eventType = DefaultEventType // default event type
	}

	c.mu.RLock()
	callback, ok := c.callbacks[eventType]
	c.mu.RUnlock()

	if ok {
		var eventData interface{}
		err := json.Unmarshal([]byte(data), &eventData)
		if err != nil {
			return err
		}

		event := &Event{
			ID:    eventID,
			Event: eventType,
			Data:  eventData,
		}

		callback(event)
	}
	return nil
}

// -------------------------------------------------------------------------------------------

var noConnectionErrText = "No connection could be made because the target machine actively refused it"

func SetNoConnectionErrText(errText string) {
	noConnectionErrText = errText
}

type retryStrategy struct {
	backoff    time.Duration // reconnect time
	maxBackoff time.Duration // max reconnect time

	retryCount int // retry count
	maxRetries int // max retry count, -1 means unlimited retries
}

// retry strategy, exponential backoff algorithm, returns true when the number of retries reaches the maximum
func (s *retryStrategy) retry() bool {
	s.retryCount++

	// calculate the next reconnect time (exponential backoff)
	if s.retryCount > 1 {
		nextBackoff := s.backoff * 2
		if nextBackoff > s.maxBackoff {
			nextBackoff = s.maxBackoff
		}
		s.backoff = nextBackoff
	}

	// check if maximum number of retries is reached
	if s.maxRetries > 0 && s.retryCount >= s.maxRetries {
		return true
	}

	return false
}

func (s *retryStrategy) reset(d time.Duration) {
	s.backoff = d
	s.retryCount = 0
}
