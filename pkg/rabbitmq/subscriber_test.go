package rabbitmq

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

func TestSubscriber(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := runPublisher(ctx, testChannelName)
	if err != nil {
		t.Log(err)
		return
	}

	err = runSubscriber(ctx, testChannelName, "fanout-queue-1")
	if err != nil {
		t.Log(err)
		return
	}

	err = runSubscriber(ctx, testChannelName, "fanout-queue-2")
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func runSubscriber(ctx context.Context, channelName string, identifier string) error {
	var subscriberErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			subscriberErr = err
			return
		}

		s, err := NewSubscriber(channelName, identifier, connection, WithConsumerAutoAck(false))
		if err != nil {
			subscriberErr = err
			return
		}

		s.Subscribe(ctx, handler)
	})
	return subscriberErr
}

func TestSubscriberErr(t *testing.T) {
	utils.SafeRunWithTimeout(time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		_, err := NewSubscriber(testChannelName, "fanout-queue-1", &Connection{conn: &amqp.Connection{}})
		if err != nil {
			t.Log(err)
			return
		}
	})

	s := &Subscriber{&Consumer{connection: &Connection{conn: &amqp.Connection{}}, ch: &amqp.Channel{}}}
	utils.SafeRun(context.Background(), func(ctx context.Context) {
		s.Subscribe(context.Background(), handler)
	})
	utils.SafeRun(context.Background(), func(ctx context.Context) {
		s.Close()
	})
}
