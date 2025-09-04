package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a single WebSocket client worker.
type Client struct {
	id    int
	conn  *websocket.Conn
	stats *statsCollector
	url   string

	isJSON bool
	//sendPayloadTemplate map[string]any // isJSON=true, for JSON data
	sendPayloadBytes []byte // if isJSON = true, sendPayloadBytes is the JSON data, otherwise, it is the binary data
	sendTicker       *time.Ticker
}

// NewClient creates a new WebSocket client worker.
func NewClient(id int, url string, stats *statsCollector, sendInterval time.Duration, payloadData []byte, isJSON bool) *Client {
	var ticker *time.Ticker
	if sendInterval > 0 {
		ticker = time.NewTicker(sendInterval)
	}

	return &Client{
		id:               id,
		stats:            stats,
		url:              url,
		isJSON:           isJSON,
		sendPayloadBytes: payloadData,
		sendTicker:       ticker,
	}
}

// Dial establishes a WebSocket connection to the server.
func (c *Client) Dial(ctx context.Context) error {
	customDialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}

	dialStartTime := time.Now()
	conn, _, err := customDialer.DialContext(ctx, c.url, nil)
	connectTime := time.Since(dialStartTime)
	if err != nil {
		c.stats.AddConnectFailure()
		c.stats.errSet.Add(err.Error())
		return err
	}
	c.stats.RecordConnectTime(connectTime)
	c.stats.AddConnectSuccess()
	c.conn = conn
	return nil
}

// Run starts the client worker.
func (c *Client) Run(ctx context.Context) {

	err := c.Dial(ctx)
	if err != nil {
		return
	}
	if c.conn == nil {
		return
	}
	defer c.conn.Close()
	defer c.stats.AddDisconnect()

	var loopWg sync.WaitGroup
	loopWg.Add(2)

	go c.writeLoop(ctx, &loopWg)
	go c.readLoop(ctx, &loopWg)

	<-ctx.Done()

	// Wait for loops to finish with a timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		loopWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-shutdownCtx.Done():
	}
}

func (c *Client) sendData(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, c.sendPayloadBytes); err != nil {
		c.stats.AddError()
		return err
	}
	c.stats.AddMessageSent()
	c.stats.AddSentBytes(uint64(len(c.sendPayloadBytes)))

	return nil
}

// writeLoop handles sending messages with sequence numbers.
func (c *Client) writeLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	if c.sendTicker != nil {
		defer c.sendTicker.Stop()
	}

	if c.sendTicker != nil {
		for {
			select {
			case <-c.sendTicker.C:
				if err := c.sendData(ctx); err != nil {
					c.stats.AddError()
					c.stats.errSet.Add(err.Error())
					return
				}
			case <-ctx.Done():
				return
			}
		}
	} else {
		for {
			if ctx.Err() != nil {
				return
			}
			if err := c.sendData(ctx); err != nil {
				c.stats.AddError()
				c.stats.errSet.Add(err.Error())
				return
			}
		}
	}
}

// readLoop handles receiving messages and checking for latency and order.
func (c *Client) readLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		//_ = c.conn.SetReadDeadline(time.Now().Add(time.Second))
		msgType, p, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.stats.errSet.Add(err.Error())
				c.stats.AddError()
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}

		if msgType == websocket.TextMessage {
			c.stats.AddMessageRecv()
			c.stats.AddRecvBytes(uint64(len(p)))
		}
	}
}
