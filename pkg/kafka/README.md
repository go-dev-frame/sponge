## kafka

`kafka` is a kafka client library based on [sarama](https://github.com/IBM/sarama) encapsulation, producer supports synchronous and asynchronous production messages, consumer supports group and partition consumption messages, fully compatible with the usage of sarama.

<br>

## Example of use

### Producer

#### Synchronous Produce

```go
package main

import (
	"fmt"
	"github.com/IBM/sarama"
	"github.com/go-dev-frame/sponge/pkg/kafka"
)

func main() {
	testTopic := "my-topic"
	addrs := []string{"localhost:9092"}
	// default config are requiredAcks=WaitForAll, partitionerConstructor=NewHashPartitioner, returnSuccesses=true
	p, err := kafka.InitSyncProducer(addrs, kafka.SyncProducerWithVersion(sarama.V3_6_0_0))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer p.Close()

	// Case 1: send sarama.ProducerMessage type message
	msg := testData[0].(*sarama.ProducerMessage) // testData is https://github.com/go-dev-frame/sponge/blob/main/pkg/kafka/producer_test.go#L19
	partition, offset, err := p.SendMessage(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("partition:", partition, "offset:", offset)

	// Case 2: send multiple types  message
	for _, data := range testData {
		partition, offset, err := p.SendData(testTopic, data)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("partition:", partition, "offset:", offset)
	}
}
```

<br>

### Asynchronous Produce

```go
package main

import (
	"fmt"
	"time"
	"github.com/IBM/sarama"
	"github.com/go-dev-frame/sponge/pkg/kafka"
)

func main() {
	testTopic := "my-topic"
	addrs := []string{"localhost:9092"}

	p, err := kafka.InitAsyncProducer(addrs,
		kafka.AsyncProducerWithVersion(sarama.V3_6_0_0),
		kafka.AsyncProducerWithRequiredAcks(sarama.WaitForLocal),
		kafka.AsyncProducerWithFlushMessages(50),
		kafka.AsyncProducerWithFlushFrequency(time.Millisecond*500),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer p.Close()

	// Case 1: send sarama.ProducerMessage type message, supports multiple messages
	msg := testData[0].(*sarama.ProducerMessage) // testData is https://github.com/go-dev-frame/sponge/blob/main/pkg/kafka/producer_test.go#L19
	err = p.SendMessage(msg, msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Case 2: send multiple types  message, supports multiple messages
	err = p.SendData(testTopic, testData...)
	if err != nil {
		fmt.Println(err)
		return
	}

	<-time.After(time.Second) // wait for all messages to be sent
}
```

<br>

### Consumer

#### Consume Group

```go
package main

import (
	"fmt"
	"time"
	"context"
	"github.com/IBM/sarama"
	"github.com/go-dev-frame/sponge/pkg/kafka"
)

func main() {
	testTopic := "my-topic"
	groupID := "my-group"
	addrs := []string{"localhost:9092"}

	// default config are offsetsInitial=OffsetOldest, autoCommitEnable=true, autoCommitInterval=time.Second
	cg, err := kafka.InitConsumerGroup(addrs, groupID, kafka.ConsumerWithVersion(sarama.V3_6_0_0))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cg.Close()

	// Case 1: consume default handle message
	go cg.ConsumeLoop(context.Background(), []string{testTopic}, handleMsgFn) // handleMsgFn is https://github.com/go-dev-frame/sponge/blob/main/pkg/kafka/consumer_test.go#L19

	// Case 2: consume custom handle message
	go cg.ConsumeCustomLoop(context.Background(), []string{testTopic}, &myConsumerGroupHandler{ // myConsumerGroupHandler is https://github.com/go-dev-frame/sponge/blob/main/pkg/kafka/consumer_test.go#L26
		autoCommitEnable: true,
	})

	<-time.After(time.Minute) // wait exit
}
```

<br>

#### Consume Partition

```go
package main

import (
	"fmt"
	"time"
	"context"
	"github.com/IBM/sarama"
	"github.com/go-dev-frame/sponge/pkg/kafka"
)

func main() {
	testTopic := "my-topic"
	addrs := []string{"localhost:9092"}

	c, err := kafka.InitConsumer(addrs, kafka.ConsumerWithVersion(sarama.V3_6_0_0))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	// Case 1: consume one partition
	go c.ConsumePartition(context.Background(), testTopic, 0, sarama.OffsetNewest, handleMsgFn) // handleMsgFn is https://github.com/go-dev-frame/sponge/blob/main/pkg/kafka/consumer_test.go#L19

	// Case 2: consume all partition
	c.ConsumeAllPartition(context.Background(), testTopic, sarama.OffsetNewest, handleMsgFn)

	<-time.After(time.Minute) // wait exit
}
```

<br>

### Topic Backlog

Obtain the total backlog of the topic and the backlog of each partition.

```go
package main

import (    
	"fmt"
	"github.com/go-dev-frame/sponge/pkg/kafka"    
)

func main() {
	m, err := kafka.InitClientManager(brokerList, groupID)
	if err != nil {
		panic(err)
	}
	defer m.Close()

	total, backlogs, err := m.GetBacklog(topic)
	if err != nil {
		panic(err)
	}

	fmt.Println("total backlog:", total)
	for _, backlog := range backlogs {
		fmt.Printf("partation=%d, backlog=%d, next_consume_offset=%d\n", backlog.Partition, backlog.Backlog, backlog.NextConsumeOffset)
	}
}
```