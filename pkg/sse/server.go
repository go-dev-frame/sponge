package sse

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
)

// ServeOption is an option for Serve
type ServeOption func(*serveOptions)

type serveOptions struct {
	extraHeaders map[string]string
}

func defaultServeOptions() *serveOptions {
	return &serveOptions{}
}

func (o *serveOptions) apply(opts ...ServeOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithServeExtraHeaders sets extra headers to be sent with the response.
func WithServeExtraHeaders(headers map[string]string) ServeOption {
	return func(o *serveOptions) {
		o.extraHeaders = headers
	}
}

// -------------------------------------------------------------------------------------------

// Serve serves a client connection
func (h *Hub) Serve(c *gin.Context, uid string, opts ...ServeOption) {
	if uid == "" {
		responseCode400(c, "uid is empty, not allow connection")
		return
	}

	o := defaultServeOptions()
	o.apply(opts...)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	for k, v := range o.extraHeaders {
		c.Writer.Header().Set(k, v)
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		_ = c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("streaming unsupported"))
		return
	}

	client := &UserClient{
		UID:     uid,
		Send:    make(chan *Event, h.pushBufferSize),
		writer:  c.Writer,
		flusher: flusher,
	}

	h.register <- client
	defer func() {
		h.unregister <- client
	}()

	eventType := DefaultEventType
	if et := c.Query("event_type"); et != "" {
		eventType = et
	} else if et = c.Request.Header.Get("Event-Type"); et != "" {
		eventType = et
	}
	lastEventID := ""
	if id := c.Query("last_event_id"); id != "" {
		lastEventID = id
	} else if id = c.Request.Header.Get("Last-Event-ID"); id != "" {
		lastEventID = id
	}

	if h.enableResendEvents && lastEventID != "" {
		h.resendEvents(client, eventType, lastEventID)
	}

	once := sync.Once{}

	c.Stream(func(w io.Writer) bool {
		once.Do(func() {
			_ = client.write([]byte(": heartbeat\n\n")) // client connected, response once means connection is ok
		})

		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()

		select {
		case e, ok := <-client.Send:
			if !ok {
				return false
			}
			if err := client.sendEvent(e); err != nil {
				h.PushStats.IncFailed()
				return false
			}
			h.PushStats.IncSuccess()
			return true

		case <-timer.C:
			err := client.write([]byte(": heartbeat\n\n"))
			if err != nil {
				return false
			}
			return true

		case <-c.Request.Context().Done():
			return false
		}
	})
}

// ServeHandler gin handler for sse server
func (h *Hub) ServeHandler(opts ...ServeOption) func(c *gin.Context) {
	return func(c *gin.Context) {
		var uid string
		v, exists := c.Get("uid") // set uid in auth middleware
		if exists {
			uid, _ = v.(string)
		}
		if uid == "" {
			responseCode400(c, "invalid uid")
			return
		}
		h.Serve(c, uid, opts...)
	}
}

// PushRequest push request
type PushRequest struct {
	UIDs   []string `json:"uids"`
	Events []*Event `json:"events"`
}

// PushEventHandler gin handler for push event request
func (h *Hub) PushEventHandler() func(c *gin.Context) {
	return func(c *gin.Context) {
		req := PushRequest{}
		if err := c.ShouldBindJSON(&req); err != nil {
			responseCode400(c, err.Error())
			return
		}

		if err := h.Push(req.UIDs, req.Events...); err != nil {
			responseCode400(c, err.Error())
			return
		}

		responseCode200(c)
	}
}

// UserClient information
type UserClient struct {
	UID     string // user id
	Send    chan *Event
	writer  http.ResponseWriter
	flusher http.Flusher

	isSendClosedEvent bool
}

func (c *UserClient) sendEvent(e *Event) error {
	var buf bytes.Buffer
	buf.WriteString("id: ")
	buf.WriteString(e.ID)
	buf.WriteString("\nevent: ")
	buf.WriteString(e.Event)
	buf.WriteString("\ndata: ")

	data, err := json.Marshal(e.Data)
	if err != nil {
		return fmt.Errorf("json.Marshal error: %v", err)
	}
	buf.Write(data)
	buf.WriteString("\n\n")

	if e.Event == "close" {
		c.isSendClosedEvent = true
	}

	return c.write(buf.Bytes())
}

func (c *UserClient) write(data []byte) error {
	_, err := c.writer.Write(data)
	if err != nil {
		return err
	}
	c.flusher.Flush()
	return nil
}

func responseCode400(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"code": http.StatusBadRequest, "msg": msg, "data": struct{}{}})
}

func responseCode200(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": struct{}{}})
}
