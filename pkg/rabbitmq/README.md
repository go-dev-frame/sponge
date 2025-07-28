## rabbitmq

`rabbitmq` is a rabbitmq client library based on [amqp091-go](https://github.com/rabbitmq/amqp091-go), supports automatic reconnection and customized setting parameters, includes `direct`, `topic`, `fanout`, `headers`, `delayed message`, `publisher subscriber` a total of six message types, and dead letter is supported.

### Example of use

#### Code Example

The following code example is including `direct`, `topic`, `fanout`, `headers`, `delayed message`, `publisher subscriber` six message types.

> Tip: the wrapped `Consume` function uses manual acknowledgement mode by default and does not need to call the ack function again.

```go
package main

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/rabbitmq"
)

var (
	producerCount int32
	consumerCount int32
)

func main() {
	url := "amqp://guest:guest@127.0.0.1:5672/"

	directExample(url)

	//topicExample(url)

	//fanoutExample(url)

	//headersExample(url)

	//delayedMessageExample(url)

	//publisherSubscriberExample(url)
}

func directExample(url string) {
	exchangeName := "direct-exchange-demo"
	queueName := "direct-queue-1"
	routeKey := "direct-key-1"
	exchange := rabbitmq.NewDirectExchange(exchangeName, routeKey)
	var queueArgs map[string]interface{}
	fmt.Printf("\n\n-------------------- direct --------------------\n")

	// producer-side direct message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		for i := 1; i <= 100; i++ {
			err = p.PublishDirect(context.Background(), []byte("[direct] message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side direct message
	{
		c := runConsume(url, exchange, queueName, queueArgs)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c.Count()))
	}

	printStat()
}

func topicExample(url string) {
	exchangeName := "topic-exchange-demo"
	queueName := "topic-queue-1"
	routingKey := "key1.key2.*"
	exchange := rabbitmq.NewTopicExchange(exchangeName, routingKey)
	var queueArgs map[string]interface{}
	fmt.Printf("\n\n-------------------- topic --------------------\n")

	// producer-side topic message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		for i := 1; i <= 100; i++ {
			key := "key1.key2.key" + strconv.Itoa(i)
			err = p.PublishTopic(context.Background(), key, []byte("[topic] "+key+" message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side topic message
	{
		c := runConsume(url, exchange, queueName, queueArgs)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c.Count()))
	}

	printStat()
}

func fanoutExample(url string) {
	exchangeName := "fanout-exchange-demo"
	queueName := "fanout-queue-1"
	exchange := rabbitmq.NewFanoutExchange(exchangeName)
	var queueArgs map[string]interface{}
	fmt.Printf("\n\n-------------------- fanout --------------------\n")

	// producer-side fanout message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		for i := 1; i <= 100; i++ {
			err = p.PublishFanout(context.Background(), []byte("[fanout] message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side fanout message
	{
		queueName = "fanout-queue-1"
		c1 := runConsume(url, exchange, queueName, queueArgs)
		queueName = "fanout-queue-2"
		c2 := runConsume(url, exchange, queueName, queueArgs)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		fmt.Println("\n\nconsumer 2 count:", c2.Count())
	}

	printStat()
}

func headersExample(url string) {
	exchangeName := "headers-exchange-demo"
	queueName := "headers-queue-1"
	headersKeys := map[string]interface{}{"hello": "world", "foo": "bar"}
	exchange := rabbitmq.NewHeadersExchange(exchangeName, rabbitmq.HeadersTypeAll, headersKeys) // all, you can set HeadersTypeAny type
	var queueArgs map[string]interface{}
	fmt.Printf("\n\n-------------------- headers --------------------\n")

	// producer-side headers message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		ctx := context.Background()
		for i := 1; i <= 100; i++ {
			headersKeys1 := headersKeys
			err = p.PublishHeaders(ctx, headersKeys1, []byte("[headers] key1 message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)

			// because of x-match: all, headersKeys2 will not match the same queue, so drop it
			headersKeys2 := map[string]interface{}{"foo": "bar"}
			err = p.PublishHeaders(ctx, headersKeys2, []byte("[headers] key2 message "+strconv.Itoa(i)))
			checkErr(err)
		}
	}

	// consumer-side headers message
	{
		c := runConsume(url, exchange, queueName, queueArgs)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c.Count()))
	}

	printStat()
}

func delayedMessageExample(url string) {
	exchangeName := "delayed-message-exchange-demo"
	queueName := "delayed-message-queue"
	routingKey := "delayed-key"
	exchange := rabbitmq.NewDelayedMessageExchange(exchangeName, rabbitmq.NewDirectExchange("", routingKey))
	var queueArgs map[string]interface{}
	fmt.Printf("\n\n-------------------- delayed message --------------------\n")

	// producer-side delayed message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		ctx := context.Background()
		datetimeLayout := "2006-01-02 15:04:05.000"
		for i := 1; i <= 100; i++ {
			err = p.PublishDelayedMessage(ctx, time.Second*3, []byte("[delayed] message "+strconv.Itoa(i)+" at "+time.Now().Format(datetimeLayout)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side delayed message
	{
		c := runConsume(url, exchange, queueName, queueArgs)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c.Count()))
	}

	printStat()
}

func publisherSubscriberExample(url string) {
	channelName := "pub-sub"
	fmt.Printf("\n\n-------------------- publisher subscriber --------------------\n")

	// publisher-side message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewPublisher(channelName, connection)
		checkErr(err)
		defer p.Close()

		for i := 1; i <= 100; i++ {
			err = p.Publish(context.Background(), []byte("[pub-sub] message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// subscriber-side message
	{
		identifier := "pub-sub-queue-1"
		s1 := runSubscriber(url, channelName, identifier)
		identifier = "pub-sub-queue-2"
		s2 := runSubscriber(url, channelName, identifier)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(s1.Count()))
		fmt.Println("\n\nsubscriber 2 count:", s2.Count())
	}

	printStat()
}

func runConsume(url string, exchange *rabbitmq.Exchange, queueName string, queueArgs map[string]interface{}) *rabbitmq.Consumer {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	checkErr(err)

	c, err := rabbitmq.NewConsumer(exchange, queueName, connection,
		rabbitmq.WithConsumerAutoAck(false),
		rabbitmq.WithConsumerQueueDeclareOptions(
			rabbitmq.WithQueueDeclareArgs(queueArgs),
		),
	)
	checkErr(err)

	c.Consume(context.Background(), handler)
	return c
}

func runSubscriber(url string, channelName string, identifier string) *rabbitmq.Subscriber {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	checkErr(err)

	s, err := rabbitmq.NewSubscriber(channelName, identifier, connection, rabbitmq.WithConsumerAutoAck(false))
	checkErr(err)

	s.Subscribe(context.Background(), handler)

	return s
}

var handler = func(ctx context.Context, data []byte, tagID string) error {
	logger.Info("received message", logger.String("tagID", tagID), logger.String("data", string(data)))
	return nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func printStat() {
	fmt.Println("\n\n-------------------- stat --------------------")
	fmt.Println("producer count:", atomic.LoadInt32(&producerCount))
	fmt.Println("consumer count:", atomic.LoadInt32(&consumerCount))
	fmt.Println("----------------------------------------------\n")
	atomic.StoreInt32(&producerCount, 0)
	atomic.StoreInt32(&consumerCount, 0)
}
```

<br>

#### Example of Dead Letter

The following example code is in the `direct`, `topic`, `fanout`, `headers`, `delayed message` five message types to add a queue of dead letters, dead letter queue is fixed to `direct` type.

> Tip: the wrapped `Consume` function uses manual acknowledgement mode by default.

```go
package main

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/rabbitmq"
)

var (
	producerCount int32
	consumerCount int32

	deadLetterConsumerCount int32
)

func main() {
	url := "amqp://guest:guest@127.0.0.1:5672/"

	directExample(url)

	//topicExample(url)

	//fanoutExample(url)

	//headersExample(url)

	//delayedMessageExample(url)
}

func directExample(url string) {
	exchangeName := "direct-exchange-demo-2"
	queueName := "direct-queue-2"
	routingKey := "direct-key-2"
	exchange := rabbitmq.NewDirectExchange(exchangeName, routingKey)
	queueArgs := map[string]interface{}{
		"x-max-length":  60,
		"x-message-ttl": 3000, // milliseconds
	}

	deadLetterQueueName := "dl-" + queueName
	deadLetterExchange := rabbitmq.NewDirectExchange("dl-"+exchangeName, "dl-"+routingKey)

	fmt.Printf("\n\n-------------------- direct --------------------\n")

	// producer-side direct message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection,
			// set queue args
			rabbitmq.WithProducerQueueDeclareOptions(
				rabbitmq.WithQueueDeclareArgs(queueArgs),
			),
			// add dead letter
			rabbitmq.WithDeadLetterOptions(
				rabbitmq.WithDeadLetter(deadLetterExchange.Name(), deadLetterQueueName, deadLetterExchange.RoutingKey()),
			),
		)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs() // get producer queue args

		for i := 1; i <= 100; i++ {
			err = p.PublishDirect(context.Background(), []byte("[direct] say hello"+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side direct message
	{
		c1 := runConsume(url, exchange, queueName, queueArgs)
		c2 := runConsumeForDeadLetter(url, deadLetterExchange, deadLetterQueueName)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		atomic.AddInt32(&deadLetterConsumerCount, int32(c2.Count()))
	}

	printStat()
}

func topicExample(url string) {
	exchangeName := "topic-exchange-demo-2"
	queueName := "topic-queue-2"
	routingKey := "dl-key1.key2.*"
	exchange := rabbitmq.NewTopicExchange(exchangeName, routingKey)
	queueArgs := map[string]interface{}{
		"x-max-length":  60,
		"x-message-ttl": 3000, // milliseconds
	}

	deadLetterQueueName := "dl-" + queueName
	deadLetterExchange := rabbitmq.NewDirectExchange("dl-"+exchangeName, "dl-"+routingKey)

	fmt.Printf("\n\n-------------------- topic --------------------\n")

	// producer-side topic message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection,
			// set queue args
			rabbitmq.WithProducerQueueDeclareOptions(
				rabbitmq.WithQueueDeclareArgs(queueArgs),
			),
			// add dead letter
			rabbitmq.WithDeadLetterOptions(
				rabbitmq.WithDeadLetter(deadLetterExchange.Name(), deadLetterQueueName, deadLetterExchange.RoutingKey()),
			),
		)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		for i := 1; i <= 100; i++ {
			key := "dl-key1.key2.key" + strconv.Itoa(i)
			err = p.PublishTopic(context.Background(), key, []byte("[topic] "+key+" message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side topic message
	{
		c1 := runConsume(url, exchange, queueName, queueArgs)
		c2 := runConsumeForDeadLetter(url, deadLetterExchange, deadLetterQueueName)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		atomic.AddInt32(&deadLetterConsumerCount, int32(c2.Count()))
	}

	printStat()
}

func fanoutExample(url string) {
	exchangeName := "fanout-exchange-demo-2"
	queueName := "fanout-queue-3"
	exchange := rabbitmq.NewFanoutExchange(exchangeName)
	queueArgs := map[string]interface{}{
		"x-max-length":  60,
		"x-message-ttl": 3000, // milliseconds
	}

	deadLetterQueueName := "dl-" + queueName
	deadLetterExchange := rabbitmq.NewDirectExchange("dl-"+exchangeName, "dl-direct-key")
	fmt.Printf("\n\n-------------------- fanout --------------------\n")

	// producer-side fanout message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection,
			// set queue args
			rabbitmq.WithProducerQueueDeclareOptions(
				rabbitmq.WithQueueDeclareArgs(queueArgs),
			),
			// add dead letter
			rabbitmq.WithDeadLetterOptions(
				rabbitmq.WithDeadLetter(deadLetterExchange.Name(), deadLetterQueueName, deadLetterExchange.RoutingKey()),
			),
		)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		for i := 1; i <= 100; i++ {
			err = p.PublishFanout(context.Background(), []byte("[fanout] message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side fanout message
	{
		queueName = "fanout-queue-3"
		c1 := runConsume(url, exchange, queueName, queueArgs)
		queueName = "fanout-queue-4"
		c2 := runConsume(url, exchange, queueName, queueArgs)
		c3 := runConsumeForDeadLetter(url, deadLetterExchange, deadLetterQueueName)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		atomic.AddInt32(&consumerCount, int32(c2.Count()))
		atomic.AddInt32(&deadLetterConsumerCount, int32(c3.Count()))
	}

	printStat()
}

func headersExample(url string) {
	exchangeName := "headers-exchange-demo-2"
	queueName := "headers-queue-2"
	headersKeys := map[string]interface{}{"hello": "world", "foo": "bar"}
	exchange := rabbitmq.NewHeadersExchange(exchangeName, rabbitmq.HeadersTypeAll, headersKeys) // all, you can set HeadersTypeAny type
	queueArgs := map[string]interface{}{
		"x-max-length":  60,
		"x-message-ttl": 3000, // milliseconds
	}

	deadLetterQueueName := "dl-" + queueName
	deadLetterExchange := rabbitmq.NewDirectExchange("dl-"+exchangeName, "dl-headers-key")
	fmt.Printf("\n\n-------------------- headers --------------------\n")

	// producer-side headers message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection,
			// set queue args
			rabbitmq.WithProducerQueueDeclareOptions(
				rabbitmq.WithQueueDeclareArgs(queueArgs),
			),
			// add dead letter
			rabbitmq.WithDeadLetterOptions(
				rabbitmq.WithDeadLetter(deadLetterExchange.Name(), deadLetterQueueName, deadLetterExchange.RoutingKey()),
			),
		)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		ctx := context.Background()
		for i := 1; i <= 100; i++ {
			headersKeys1 := headersKeys
			err = p.PublishHeaders(ctx, headersKeys1, []byte("[headers] message "+strconv.Itoa(i)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)

			// because of x-match: all, headersKeys2 will not match the same queue, so drop it
			headersKeys2 := map[string]interface{}{"foo": "bar"}
			err = p.PublishHeaders(ctx, headersKeys2, []byte("[headers] key2 message"))
			checkErr(err)
		}
	}

	// consumer-side headers message
	{
		c1 := runConsume(url, exchange, queueName, queueArgs)
		c2 := runConsumeForDeadLetter(url, deadLetterExchange, deadLetterQueueName)

		<-time.After(time.Second * 5)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		atomic.AddInt32(&deadLetterConsumerCount, int32(c2.Count()))
	}

	printStat()
}

func delayedMessageExample(url string) {
	exchangeName := "delayed-message-exchange-demo-2"
	queueName := "delayed-message-queue-2"
	routingKey := "delayed-key-2"
	exchange := rabbitmq.NewDelayedMessageExchange(exchangeName, rabbitmq.NewDirectExchange("", routingKey))
	queueArgs := map[string]interface{}{
		"x-max-length":  60,
		"x-message-ttl": 3000, // milliseconds
	}

	deadLetterQueueName := "dl-" + queueName
	deadLetterExchange := rabbitmq.NewDirectExchange("dl-"+exchangeName, "dl-"+routingKey)

	fmt.Printf("\n\n-------------------- delayed message --------------------\n")

	// producer-side delayed message
	{
		connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
		checkErr(err)
		defer connection.Close()

		p, err := rabbitmq.NewProducer(exchange, queueName, connection,
			// set queue args
			rabbitmq.WithProducerQueueDeclareOptions(
				rabbitmq.WithQueueDeclareArgs(queueArgs),
			),
			// add dead letter
			rabbitmq.WithDeadLetterOptions(
				rabbitmq.WithDeadLetter(deadLetterExchange.Name(), deadLetterQueueName, deadLetterExchange.RoutingKey()),
			),
		)
		checkErr(err)
		defer p.Close()
		queueArgs = p.QueueArgs()

		ctx := context.Background()
		datetimeLayout := "2006-01-02 15:04:05.000"
		for i := 1; i <= 200; i++ {
			delayTime := time.Second
			if i > 100 {
				delayTime = time.Second * 2
			}

			err = p.PublishDelayedMessage(ctx, delayTime, []byte("[delayed] message "+strconv.Itoa(i)+" at "+time.Now().Format(datetimeLayout)))
			checkErr(err)
			atomic.AddInt32(&producerCount, 1)
		}
	}

	// consumer-side delayed message
	{
		time.Sleep(time.Second * 3) // wait for all messages to be sent
		c1 := runConsume(url, exchange, queueName, queueArgs)
		c2 := runConsumeForDeadLetter(url, deadLetterExchange, deadLetterQueueName)

		<-time.After(time.Second * 10)
		atomic.AddInt32(&consumerCount, int32(c1.Count()))
		atomic.AddInt32(&deadLetterConsumerCount, int32(c2.Count()))
	}

	printStat()
}

func runConsume(url string, exchange *rabbitmq.Exchange, queueName string, queueArgs map[string]interface{}) *rabbitmq.Consumer {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	checkErr(err)

	c, err := rabbitmq.NewConsumer(exchange, queueName, connection,
		rabbitmq.WithConsumerAutoAck(false),
		rabbitmq.WithConsumerQueueDeclareOptions(
			rabbitmq.WithQueueDeclareArgs(queueArgs),
		),
	)
	checkErr(err)

	c.Consume(context.Background(), handler)
	return c
}

func runConsumeForDeadLetter(url string, exchange *rabbitmq.Exchange, queueName string) *rabbitmq.Consumer {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	checkErr(err)

	c, err := rabbitmq.NewConsumer(exchange, queueName, connection, rabbitmq.WithConsumerAutoAck(false))
	checkErr(err)

	c.Consume(context.Background(), handler)
	return c
}

var handler = func(ctx context.Context, data []byte, tagID string) error {
	logger.Info("received message", logger.String("tagID", tagID), logger.String("data", string(data)))
	return nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func printStat() {
	fmt.Println("\n\n-------------------- stat --------------------")
	fmt.Println("producer count:", producerCount)
	fmt.Println("consumer count:", consumerCount)
	fmt.Println("dead letter consumer count:", deadLetterConsumerCount)
	fmt.Println("----------------------------------------------\n")
	atomic.StoreInt32(&producerCount, 0)
	atomic.StoreInt32(&consumerCount, 0)
	atomic.StoreInt32(&deadLetterConsumerCount, 0)
}
```

<br>

#### Example of Automatic Resumption of Publish

If the error of publish is caused by the network, you can check if the reconnection is successful and publish it again.

```go
package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/rabbitmq"
)

var url = "amqp://guest:guest@127.0.0.1:5672/"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	exchangeName := "direct-exchange-demo"
	queueName := "direct-queue"
	routeKey := "info"
	exchange := rabbitmq.NewDirectExchange(exchangeName, routeKey)

	err := runConsume(ctx, exchange, queueName)
	if err != nil {
		logger.Error("runConsume failed", logger.Err(err))
		return
	}

	err = runProduce(ctx, exchange, queueName)
	if err != nil {
		logger.Error("runProduce failed", logger.Err(err))
		return
	}
}

func runProduce(ctx context.Context, exchange *rabbitmq.Exchange, queueName string) error {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	if err != nil {
		return err
	}
	defer connection.Close()

	p, err := rabbitmq.NewProducer(exchange, queueName, connection)
	if err != nil {
		return err
	}
	defer p.Close()

	count := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			count++
			data := []byte("direct say hello" + strconv.Itoa(count))
			err = p.PublishDirect(ctx, data)
			if err != nil {
				if errors.Is(err, rabbitmq.ErrClosed) {
					for {
						if !connection.CheckConnected() { // check connection
							time.Sleep(time.Second * 2)
							continue
						}
						p, err = rabbitmq.NewProducer(exchange, queueName, connection)
						if err != nil {
							logger.Warn("reconnect failed", logger.Err(err))
							time.Sleep(time.Second * 2)
							continue
						}
						break
					}
				} else {
					logger.Warn("publish failed", logger.Err(err))
				}
			}
			logger.Info("publish message", logger.String("data", string(data)))
			time.Sleep(time.Second * 5)
		}
	}
}

func runConsume(ctx context.Context, exchange *rabbitmq.Exchange, queueName string) error {
	connection, err := rabbitmq.NewConnection(url, rabbitmq.WithLogger(logger.Get()))
	if err != nil {
		return err
	}

	c, err := rabbitmq.NewConsumer(exchange, queueName, connection, rabbitmq.WithConsumerAutoAck(false))
	if err != nil {
		return err
	}

	c.Consume(ctx, handler)

	return nil
}

var handler = func(ctx context.Context, data []byte, tagID string) error {
	logger.Info("received message", logger.String("tagID", tagID), logger.String("data", string(data)))
	return nil
}
```
