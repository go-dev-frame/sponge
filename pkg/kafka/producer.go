// Package kafka is a kafka client package.
package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
	json "github.com/bytedance/sonic"
	"go.uber.org/zap"
)

// ProducerMessage is sarama ProducerMessage
type ProducerMessage = sarama.ProducerMessage

// ---------------------------------- sync producer ---------------------------------------

// SyncProducer is a sync producer.
type SyncProducer struct {
	Producer sarama.SyncProducer
}

// InitSyncProducer init sync producer.
func InitSyncProducer(addrs []string, opts ...SyncProducerOption) (*SyncProducer, error) {
	o := defaultSyncProducerOptions()
	o.apply(opts...)

	var config *sarama.Config
	if o.config != nil {
		config = o.config
	} else {
		config = sarama.NewConfig()
		config.Version = o.version
		config.Producer.RequiredAcks = o.requiredAcks
		config.Producer.Partitioner = o.partitioner
		config.Producer.Return.Successes = o.returnSuccesses
		config.ClientID = o.clientID
		if o.tlsConfig != nil {
			config.Net.TLS.Config = o.tlsConfig
			config.Net.TLS.Enable = true
		}
	}

	producer, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}

	return &SyncProducer{Producer: producer}, nil
}

// SendMessage sends a message to a topic.
func (p *SyncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	return p.Producer.SendMessage(msg)
}

// SendData sends a message to a topic with multiple types of data.
func (p *SyncProducer) SendData(topic string, data interface{}) (int32, int64, error) {
	var msg *sarama.ProducerMessage
	switch val := data.(type) {
	case *sarama.ProducerMessage:
		msg = val
	case []byte:
		msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(val)}
	case string:
		msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(val)}
	case *Message:
		msg = &sarama.ProducerMessage{Topic: val.Topic, Value: sarama.ByteEncoder(val.Data), Key: sarama.ByteEncoder(val.Key)}
	default:
		buf, err := json.Marshal(data)
		if err != nil {
			return 0, 0, err
		}
		msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(buf)}
	}

	return p.Producer.SendMessage(msg)
}

// Close closes the producer.
func (p *SyncProducer) Close() error {
	if p.Producer != nil {
		return p.Producer.Close()
	}
	return nil
}

// Message is a message to be sent to a topic.
type Message struct {
	Topic string `json:"topic"`
	Data  []byte `json:"data"`
	Key   []byte `json:"key"`
}

// ---------------------------------- async producer ---------------------------------------

// AsyncProducer is async producer.
type AsyncProducer struct {
	Producer  sarama.AsyncProducer
	zapLogger *zap.Logger
	exit      chan struct{}
}

// InitAsyncProducer init async producer.
func InitAsyncProducer(addrs []string, opts ...AsyncProducerOption) (*AsyncProducer, error) {
	o := defaultAsyncProducerOptions()
	o.apply(opts...)

	var config *sarama.Config
	if o.config != nil {
		config = o.config
	} else {
		config = sarama.NewConfig()
		config.Version = o.version
		config.Producer.RequiredAcks = o.requiredAcks
		config.Producer.Partitioner = o.partitioner
		config.Producer.Return.Successes = o.returnSuccesses
		config.ClientID = o.clientID
		config.Producer.Flush.Messages = o.flushMessages
		config.Producer.Flush.Frequency = o.flushFrequency
		config.Producer.Flush.Bytes = o.flushBytes
		if o.tlsConfig != nil {
			config.Net.TLS.Config = o.tlsConfig
			config.Net.TLS.Enable = true
		}
	}

	producer, err := sarama.NewAsyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}

	p := &AsyncProducer{
		Producer:  producer,
		zapLogger: o.zapLogger,
		exit:      make(chan struct{}),
	}

	go p.handleResponse(o.handleFailedFn)

	return p, nil
}

// SendMessage sends messages to a topic.
func (p *AsyncProducer) SendMessage(messages ...*sarama.ProducerMessage) error {
	for _, msg := range messages {
		select {
		case p.Producer.Input() <- msg:
		case <-p.exit:
			return fmt.Errorf("async produce message had exited")
		}
	}

	return nil
}

// SendData sends messages to a topic with multiple types of data.
func (p *AsyncProducer) SendData(topic string, multiData ...interface{}) error {
	var messages []*sarama.ProducerMessage

	for _, data := range multiData {
		var msg *sarama.ProducerMessage
		switch val := data.(type) {
		case *sarama.ProducerMessage:
			msg = val
		case []byte:
			msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(val)}
		case string:
			msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(val)}
		case *Message:
			msg = &sarama.ProducerMessage{Topic: val.Topic, Value: sarama.ByteEncoder(val.Data), Key: sarama.ByteEncoder(val.Key)}
		default:
			buf, err := json.Marshal(data)
			if err != nil {
				return err
			}
			msg = &sarama.ProducerMessage{Topic: topic, Value: sarama.ByteEncoder(buf)}
		}
		messages = append(messages, msg)
	}

	return p.SendMessage(messages...)
}

// handleResponse handles the response of async producer, if producer message failed, you can handle it, e.g. add to other queue to handle later.
func (p *AsyncProducer) handleResponse(handleFn AsyncSendFailedHandlerFn) {
	defer func() {
		if e := recover(); e != nil {
			p.zapLogger.Error("panic occurred while processing async message", zap.Any("error", e))
			p.handleResponse(handleFn)
		}
	}()

	for {
		select {
		case pm := <-p.Producer.Successes():
			p.zapLogger.Info("async send successfully",
				zap.String("topic", pm.Topic),
				zap.Int32("partition", pm.Partition),
				zap.Int64("offset", pm.Offset))
		case err := <-p.Producer.Errors():
			p.zapLogger.Error("async send failed", zap.Error(err.Err), zap.Any("msg", err.Msg))
			if handleFn != nil {
				e := handleFn(err.Msg)
				if e != nil {
					p.zapLogger.Error("handle failed msg failed", zap.Error(e))
				}
			}
		case <-p.exit:
			return
		}
	}
}

// Close closes the producer.
func (p *AsyncProducer) Close() error {
	defer func() { _ = recover() }() // ignore error
	close(p.exit)
	if p.Producer != nil {
		return p.Producer.Close()
	}
	return nil
}
