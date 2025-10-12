package sse

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// HubOption hub option
type HubOption func(*hubOptions)

type hubOptions struct {
	store Store
	// enable resend events from store specified by event ID, default is false, dependency store
	enableResendEvents bool

	ctx                context.Context
	cancel             context.CancelFunc
	logger             *zap.Logger
	pushBufferSize     int
	pushFailedHandleFn func(uid string, event *Event)
	workerNum          int
}

func defaultHubOptions() *hubOptions {
	zapLogger, _ := zap.NewProduction()
	ctx, cancel := context.WithCancel(context.Background())
	return &hubOptions{
		store:          nil,
		ctx:            ctx,
		cancel:         cancel,
		logger:         zapLogger,
		pushBufferSize: 1000,
		workerNum:      10,
	}
}

func (h *hubOptions) apply(opts ...HubOption) {
	for _, opt := range opts {
		opt(h)
	}
}

// WithContext set context and cancel
func WithContext(ctx context.Context, cancel context.CancelFunc) HubOption {
	return func(h *hubOptions) {
		if ctx != nil {
			h.ctx = ctx
		}
		if cancel != nil {
			h.cancel = cancel
		}
	}
}

// WithStore set store
func WithStore(store Store) HubOption {
	return func(h *hubOptions) {
		h.store = store
	}
}

// WithEnableResendEvents enable resend events
func WithEnableResendEvents() HubOption {
	return func(h *hubOptions) {
		h.enableResendEvents = true
	}
}

// WithLogger set logger
func WithLogger(logger *zap.Logger) HubOption {
	return func(h *hubOptions) {
		if logger == nil {
			return
		}
		h.logger = logger
	}
}

// WithPushBufferSize set push events buffer size
func WithPushBufferSize(size int) HubOption {
	return func(h *hubOptions) {
		if size <= 0 {
			return
		}
		h.pushBufferSize = size
	}
}

// WithPushFailedHandleFn set push failed handle function
func WithPushFailedHandleFn(fn func(uid string, event *Event)) HubOption {
	return func(h *hubOptions) {
		h.pushFailedHandleFn = fn
	}
}

// WithWorkerNum set worker num
func WithWorkerNum(num int) HubOption {
	return func(h *hubOptions) {
		if num <= 0 {
			return
		}
		h.workerNum = num
	}
}

// ------------------------------------------------------------------------------------------

// UserEvent user event
type UserEvent struct {
	UID   string `json:"uid"`
	Event *Event `json:"event"`
}

// Hub event center, manage client connections, receive user events, and broadcast them to online users
type Hub struct {
	store Store
	// default is false, if it is enabled and store is not nil, send event messages starting
	// from the specified event ID after disconnecting and reconnecting.
	enableResendEvents bool

	clients            *SafeMap // userID -> *Client
	register           chan *UserClient
	unregister         chan *UserClient
	broadcast          chan *UserEvent
	asyncTaskPool      *AsyncTaskPool
	PushStats          *PushStats
	maxRetry           int
	pushBufferSize     int
	pushFailedHandleFn func(uid string, event *Event)

	ctx       context.Context
	cancel    context.CancelFunc
	zapLogger *zap.Logger
}

// NewHub create a new event center
func NewHub(opts ...HubOption) *Hub {
	o := defaultHubOptions()
	o.apply(opts...)

	h := &Hub{
		store:              o.store,
		clients:            NewSafeMap(),
		register:           make(chan *UserClient),
		unregister:         make(chan *UserClient),
		pushBufferSize:     o.pushBufferSize,
		broadcast:          make(chan *UserEvent, o.pushBufferSize), // default buffer size is 1000
		asyncTaskPool:      NewAsyncTaskPool(o.workerNum),           // default worker num is 10
		pushFailedHandleFn: o.pushFailedHandleFn,
		PushStats:          &PushStats{},
		maxRetry:           3,

		ctx:                o.ctx,
		cancel:             o.cancel,
		zapLogger:          o.logger,
		enableResendEvents: o.enableResendEvents,
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case cli := <-h.register:
			h.clients.Set(cli.UID, cli)
			h.zapLogger.Info("[sse] user connected", zap.String("uid", cli.UID))

		case cli := <-h.unregister:
			if h.clients.Has(cli.UID) {
				h.clients.Delete(cli.UID)
				close(cli.Send)
				h.zapLogger.Info("[sse] user disconnected", zap.String("uid", cli.UID))
			}

		case ue := <-h.broadcast:
			cli, ok := h.clients.Get(ue.UID)
			if ok {
				select {
				case cli.Send <- ue.Event:
				//h.zapLogger.Info("[sse] pushed event to client", zap.String("uid", ue.UID), zap.String("event_id", ue.Event.ID))
				default:
					trySendWithTimeout(h, cli, ue, 5*time.Second)
				}
			}

		case <-h.ctx.Done():
			h.zapLogger.Info("[sse] event center stopped")
			return
		}
	}
}

// Push to specified users or all online users
func (h *Hub) Push(uids []string, events ...*Event) error {
	if len(events) == 0 {
		return errors.New("events can not be empty")
	}
	for _, e := range events {
		if err := e.CheckValid(); err != nil {
			return fmt.Errorf("check event valid failed: %v", err)
		}
		if e.ID == "" {
			e.ID = newStringID()
		}

		if h.store != nil {
			err := h.store.Save(h.ctx, e)
			if err != nil {
				return fmt.Errorf("save event failed: %v", err)
			}
		}

		if len(uids) > 0 {
			// push to specified users
			for _, uid := range uids {
				if !h.clients.Has(uid) {
					continue
				}
				pushOne(h, &UserEvent{
					UID:   uid,
					Event: e,
				})
			}
		} else {
			// push to all online users
			pushAll(h, e)
		}
	}

	return nil
}

func pushOne(h *Hub, ue *UserEvent) {
	h.PushStats.IncTotal()

	// try to send without blocking, fast path
	select {
	case h.broadcast <- ue:

	default:
		// use async task pool to submit push task with timeout retry
		h.asyncTaskPool.Submit(func() {
			tryPushWithTimeout(h, ue, 5*time.Second)
		})
	}
}

func pushAll(h *Hub, e *Event) {
	h.clients.Range(func(uidKey, _ interface{}) bool {
		uid := uidKey.(string)
		ue := &UserEvent{
			UID:   uid,
			Event: e,
		}
		pushOne(h, ue)
		return true
	})
}

// Asynchronous retry push with timeout logic
func tryPushWithTimeout(h *Hub, ue *UserEvent, timeout time.Duration) {
	for i := 0; i < h.maxRetry; i++ {
		timer := time.NewTimer(timeout)
		select {
		case h.broadcast <- ue:
			timer.Stop()
			return
		case <-timer.C:
			h.PushStats.IncTimeout()
			h.zapLogger.Warn("push timeout", zap.String("uid", ue.UID), zap.String("event_id", ue.Event.ID), zap.Int("retry", i+1))
		}
	}

	// all retry failed
	h.PushStats.IncFailed()
	if h.pushFailedHandleFn != nil {
		h.pushFailedHandleFn(ue.UID, ue.Event)
	}
}

// Asynchronous retry send with timeout logic
func trySendWithTimeout(h *Hub, cli *UserClient, ue *UserEvent, timeout time.Duration) {
	for i := 0; i < h.maxRetry; i++ {
		timer := time.NewTimer(timeout)
		select {
		case cli.Send <- ue.Event:
			timer.Stop()
			return
		case <-timer.C:
			h.PushStats.IncTimeout()
			h.zapLogger.Warn("send timeout", zap.String("uid", ue.UID), zap.String("event_id", ue.Event.ID), zap.Int("retry", i+1))
		}
	}

	// all retry failed
	h.PushStats.IncFailed()
	if h.pushFailedHandleFn != nil {
		h.pushFailedHandleFn(ue.UID, ue.Event)
	}
}

// resend events to specified client after disconnecting and reconnecting
func (h *Hub) resendEvents(client *UserClient, eventType string, lastEventID string) {
	if h.store == nil || client == nil {
		return
	}

	for {
		events, nextID, err := h.store.ListByLastID(h.ctx, eventType, lastEventID, 100)
		if err != nil {
			h.zapLogger.Warn("get events error:", zap.Error(err))
			return
		}

		if len(events) == 0 {
			return
		}

		for _, e := range events {
			if err = client.sendEvent(e); err != nil {
				if h.pushFailedHandleFn != nil {
					h.pushFailedHandleFn(client.UID, e)
				} else {
					h.zapLogger.Warn("[sse] send event failed",
						zap.Error(err),
						zap.String("event_id", e.ID),
						zap.String("event_type", e.Event))
				}
			}
		}

		// update lastID to nextID
		lastEventID = nextID

		// if nextID is empty, it means all events have been sent, exit loop
		if nextID == "" {
			return
		}
	}
}

// PrintPushStats print push stats
func (h *Hub) PrintPushStats() {
	total, success, failed, timeout := h.PushStats.Snapshot()
	h.zapLogger.Info("push stats",
		zap.Int64("total", total),
		zap.Int64("success", success),
		zap.Int64("failed", failed),
		zap.Int64("timeout", timeout))
}

// PushHeartBeat push heart beat event to specified user
func (h *Hub) PushHeartBeat(uid string) {
	e := &Event{Event: "heartbeat"}
	pushOne(h, &UserEvent{
		UID:   uid,
		Event: e,
	})
}

// OnlineClientsNum get online clients num
func (h *Hub) OnlineClientsNum() int {
	return h.clients.Len()
}

// Close event center and stop all worker,
// By default, send a shutdown event to the client. If you want the client
// to automatically reconnect, please set the tryToReconnect parameter to false.
func (h *Hub) Close(tryToReconnect ...bool) {
	if h.OnlineClientsNum() > 0 {
		h.zapLogger.Info("[sse] closing clients", zap.Int("client_num", h.OnlineClientsNum()))

		noTryToReconnect := false
		if len(tryToReconnect) == 0 || tryToReconnect[0] {
			_ = h.Push(nil, CloseEvent())
			noTryToReconnect = true
		}

		if noTryToReconnect {
			for i := 0; i < 50; i++ {
				time.Sleep(100 * time.Millisecond)
				h.clients.Range(func(uidKey, cliVal interface{}) bool {
					cli := cliVal.(*UserClient)
					if cli.isSendClosedEvent && h.clients.Has(cli.UID) {
						h.clients.Delete(cli.UID)
						close(cli.Send)
					}
					return true
				})
				if h.OnlineClientsNum() == 0 {
					break
				}
			}
		}
	}

	h.cancel()
	h.asyncTaskPool.Wait()
	h.asyncTaskPool.Stop()
}

// ------------------------------------------------------------------------------------------

// PushStats push stats
type PushStats struct {
	total   int64 // total push count
	success int64 // success push count
	failed  int64 // failed push count
	timeout int64 // timeout push count
}

// IncTotal increment total push count
func (s *PushStats) IncTotal() {
	atomic.AddInt64(&s.total, 1)
}

// IncSuccess increment success push count
func (s *PushStats) IncSuccess() {
	atomic.AddInt64(&s.success, 1)
}

// IncFailed increment failed push count
func (s *PushStats) IncFailed() {
	atomic.AddInt64(&s.failed, 1)
}

// IncTimeout increment timeout push count
func (s *PushStats) IncTimeout() {
	atomic.AddInt64(&s.timeout, 1)
}

// Snapshot get push stats snapshot
func (s *PushStats) Snapshot() (total, success, failed, timeout int64) { //nolint
	return atomic.LoadInt64(&s.total),
		atomic.LoadInt64(&s.success),
		atomic.LoadInt64(&s.failed),
		atomic.LoadInt64(&s.timeout)
}

func newStringID() string {
	ns := time.Now().UnixMicro() * 1000
	return strconv.FormatInt(ns+rand.Int63n(1000), 16)
}
