package sse

import (
	"context"
	"errors"
)

// DefaultEventType is the default event type if not provided in the event
var DefaultEventType = "message"

// Event defines a Server-Sent Event
type Event struct {
	ID    string      `json:"id"`    // event id, unique, if not provided, server will generate one
	Event string      `json:"event"` // event type
	Data  interface{} `json:"data"`  // event data
}

// CheckValid checks if the event is valid
func (e *Event) CheckValid() error {
	if e.Event == "" {
		return errors.New("invalid event")
	}
	if e.Data == nil {
		return errors.New("invalid data")
	}
	return nil
}

// CloseEvent returns a close event
func CloseEvent() *Event {
	return &Event{
		Event: "close",
		Data:  "server closed connection, do not retry",
	}
}

// Store defines the interface for storing and retrieving events
type Store interface {
	Save(ctx context.Context, e *Event) error
	// ListByLastID list events by event type and last id, return events and last id, if no more events, return empty slice and last id ""
	ListByLastID(ctx context.Context, eventType string, lastID string, pageSize int) ([]*Event, string, error)
}
