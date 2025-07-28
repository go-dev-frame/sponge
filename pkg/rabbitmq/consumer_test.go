package rabbitmq

import (
	"context"
	"fmt"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/go-dev-frame/sponge/pkg/utils"
)

func TestConsumerOptions(t *testing.T) {
	opts := []ConsumerOption{
		WithConsumerExchangeDeclareOptions(
			WithExchangeDeclareAutoDelete(true),
			WithExchangeDeclareInternal(true),
			WithExchangeDeclareNoWait(true),
			WithExchangeDeclareArgs(map[string]interface{}{"foo1": "bar1"}),
		),
		WithConsumerQueueDeclareOptions(
			WithQueueDeclareAutoDelete(true),
			WithQueueDeclareExclusive(true),
			WithQueueDeclareNoWait(true),
			WithQueueDeclareArgs(map[string]interface{}{"foo": "bar"}),
		),
		WithConsumerQueueBindOptions(
			WithQueueBindNoWait(true),
			WithQueueBindArgs(map[string]interface{}{"foo2": "bar2"}),
		),
		WithConsumerConsumeOptions(
			WithConsumeConsumer("test"),
			WithConsumeExclusive(true),
			WithConsumeNoLocal(true),
			WithConsumeNoWait(true),
			WithConsumeArgs(map[string]interface{}{"foo": "bar"}),
		),
		WithConsumerQosOptions(
			WithQosEnable(),
			WithQosPrefetchCount(1),
			WithQosPrefetchSize(4096),
			WithQosPrefetchGlobal(true),
		),
		WithConsumerAutoAck(true),
		WithConsumerPersistent(true),
	}

	o := defaultConsumerOptions()
	o.apply(opts...)

	assert.True(t, o.queueDeclare.autoDelete)
	assert.True(t, o.queueDeclare.exclusive)
	assert.True(t, o.queueDeclare.noWait)
	assert.Equal(t, "bar", o.queueDeclare.args["foo"])

	assert.True(t, o.exchangeDeclare.autoDelete)
	assert.True(t, o.exchangeDeclare.internal)
	assert.True(t, o.exchangeDeclare.noWait)
	assert.Equal(t, "bar1", o.exchangeDeclare.args["foo1"])

	assert.True(t, o.queueBind.noWait)
	assert.Equal(t, "bar2", o.queueBind.args["foo2"])

	assert.True(t, o.isPersistent)
	assert.True(t, o.isAutoAck)
}

var handler = func(ctx context.Context, data []byte, tagID string) error {
	fmt.Printf("[received]: tagID=%s, data=%s\n", tagID, data)
	return nil
}

func consume(ctx context.Context, queueName string, exchange *Exchange) error {
	var consumeErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			consumeErr = err
			return
		}

		c, err := NewConsumer(exchange, queueName, connection, WithConsumerAutoAck(false))
		if err != nil {
			consumeErr = err
			return
		}
		c.Consume(ctx, handler)
	})
	return consumeErr
}

func TestConsumer_direct(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	exchangeName := "direct-exchange-demo"
	queueName := "direct-queue-1"
	routeKey := "direct-key-1"
	exchange := NewDirectExchange(exchangeName, routeKey)

	err := producerDirect(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	err = consume(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func TestConsumer_topic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	exchangeName := "topic-exchange-demo"
	queueName := "topic-queue-1"
	routingKey := "key1.key2.*"
	exchange := NewTopicExchange(exchangeName, routingKey)

	err := producerTopic(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	err = consume(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func TestConsumer_fanout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	exchangeName := "fanout-exchange-demo"
	queueName := "fanout-queue-1"
	exchange := NewFanoutExchange(exchangeName)

	err := producerFanout(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	err = consume(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func TestConsumer_headers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	exchangeName := "headers-exchange-demo"
	queueName := "headers-queue-1"
	kv1 := map[string]interface{}{"hello1": "world1", "foo1": "bar1"}
	exchange := NewHeadersExchange(exchangeName, HeadersTypeAll, kv1) // all

	err := producerHeaders(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	err = consume(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func TestConsumer_delayedMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*7)
	defer cancel()
	exchangeName := "delayed-message-exchange-demo"
	queueName := "delayed-message-queue"
	routingKey := "delayed-key"
	exchange := NewDelayedMessageExchange(exchangeName, NewDirectExchange("", routingKey))

	err := producerDelayedMessage(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	err = consume(ctx, queueName, exchange)
	if err != nil {
		t.Log(err)
		return
	}

	<-ctx.Done()
	time.Sleep(time.Millisecond * 100)
}

func producerDirect(ctx context.Context, queueName string, exchange *Exchange) error {
	var producerErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			producerErr = err
			return
		}
		defer connection.Close()

		p, err := NewProducer(exchange, queueName, connection)
		if err != nil {
			producerErr = err
			return
		}
		defer p.Close()

		_ = p.PublishDirect(ctx, []byte("say hello 1"))
		_ = p.PublishDirect(ctx, []byte("say hello 2"))
		producerErr = p.PublishDirect(ctx, []byte("say hello 3"))
	})

	return producerErr
}

func producerTopic(ctx context.Context, queueName string, exchange *Exchange) error {
	var producerErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			producerErr = err
			return
		}
		defer connection.Close()

		p, err := NewProducer(exchange, queueName, connection)
		if err != nil {
			producerErr = err
			return
		}
		defer p.Close()

		key := "key1.key2.key3"
		producerErr = p.PublishTopic(ctx, key, []byte(key+" say hello"))
	})

	return producerErr
}

func producerFanout(ctx context.Context, queueName string, exchange *Exchange) error {
	var producerErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			producerErr = err
			return
		}
		defer connection.Close()

		p, err := NewProducer(exchange, queueName, connection)
		if err != nil {
			producerErr = err
			return
		}
		defer p.Close()

		producerErr = p.PublishFanout(ctx, []byte(" say hello"))
	})
	return producerErr
}

func producerHeaders(ctx context.Context, queueName string, exchange *Exchange) error {
	var producerErr error
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			producerErr = err
			return
		}
		defer connection.Close()

		p, err := NewProducer(exchange, queueName, connection)
		if err != nil {
			producerErr = err
			return
		}
		defer p.Close()

		headersKey1 := exchange.headersKeys
		err = p.PublishHeaders(ctx, headersKey1, []byte("say hello 1"))
		if err != nil {
			producerErr = err
			return
		}
		headersKey1 = map[string]interface{}{"foo": "bar"}
		producerErr = p.PublishHeaders(ctx, headersKey1, []byte("say hello 2"))
	})
	return producerErr
}

func producerDelayedMessage(ctx context.Context, queueName string, exchange *Exchange) error {
	var producerErr error
	utils.SafeRunWithTimeout(time.Second*6, func(cancel context.CancelFunc) {
		defer cancel()
		connection, err := NewConnection(url)
		if err != nil {
			producerErr = err
			return
		}
		defer connection.Close()

		p, err := NewProducer(exchange, queueName, connection)
		if err != nil {
			producerErr = err
			return
		}
		defer p.Close()

		producerErr = p.PublishDelayedMessage(ctx, time.Second*5, []byte("say hello "+time.Now().Format(datetimeLayout)))
		time.Sleep(time.Second)
		producerErr = p.PublishDelayedMessage(ctx, time.Second*5, []byte("say hello "+time.Now().Format(datetimeLayout)))
	})
	return producerErr
}

func TestConsumerErr(t *testing.T) {
	connection := &Connection{
		exit:        make(chan struct{}),
		zapLog:      zap.NewNop(),
		conn:        &amqp.Connection{},
		isConnected: true,
	}

	exchange := NewDirectExchange("foo", "bar")
	c, err := NewConsumer(exchange, "test", connection, WithConsumerQosOptions(
		WithQosEnable(),
		WithQosPrefetchCount(1)),
	)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(c.Count())
	c.ch = &amqp.Channel{}

	utils.SafeRunWithTimeout(time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		err := c.initialize()
		if err != nil {
			t.Log(err)
			return
		}
	})
	utils.SafeRunWithTimeout(time.Second, func(cancel context.CancelFunc) {
		defer cancel()
		_, err := c.consumeWithContext(context.Background())
		if err != nil {
			t.Log(err)
			return
		}
	})
	utils.SafeRunWithTimeout(time.Second*3, func(cancel context.CancelFunc) {
		defer cancel()
		c.Consume(context.Background(), handler)
	})
	utils.SafeRun(context.Background(), func(ctx context.Context) {
		c.Close()
	})
	time.Sleep(time.Millisecond * 2500)
	close(c.connection.exit)
}
