package whatapsarama

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/trace"
	"golang.org/x/net/context"
)

func TestOnSendWithoutMetadata(t *testing.T) {
	assert := assert.New(t)

	brokers := []string{"1.1.1.1:9092"}
	interceptor := Interceptor{Brokers: brokers}
	msg := &sarama.ProducerMessage{
		Topic:    "Topic",
		Key:      sarama.StringEncoder("Key"),
		Value:    sarama.StringEncoder("Value"),
		Metadata: nil,
	}

	interceptor.OnSend(msg)

	check := false
	for _, header := range msg.Headers {
		key := string(header.Key)
		if key == saramaTraceCtx {
			check = true
		}
	}

	assert.True(check)

	assert.Contains(msg.Topic, "Topic")
	assert.Contains(msg.Key, "Key")
	assert.Contains(msg.Value, "Value")

}

func TestOnSendWithContext(t *testing.T) {
	assert := assert.New(t)

	brokers := []string{"1.1.1.1:9092"}
	interceptor := Interceptor{Brokers: brokers}
	msg := &sarama.ProducerMessage{
		Topic:    "Topic",
		Key:      sarama.StringEncoder("Key"),
		Value:    sarama.StringEncoder("Value"),
		Metadata: context.Background(),
	}

	interceptor.OnSend(msg)

	check := false
	for _, header := range msg.Headers {
		key := string(header.Key)
		if key == saramaTraceCtx {
			check = true
		}
	}

	assert.True(check)

	assert.Contains(msg.Topic, "Topic")
	assert.Contains(msg.Key, "Key")
	assert.Contains(msg.Value, "Value")

}

func TestOnSendWithHeader(t *testing.T) {
	assert := assert.New(t)

	ctx, _ := trace.Start(context.Background(), "TEST")
	defer trace.End(ctx, nil)

	brokers := []string{"1.1.1.1:9092"}
	interceptor := Interceptor{Brokers: brokers}
	msg := &sarama.ProducerMessage{
		Topic:    "Topic",
		Key:      sarama.StringEncoder("Key"),
		Value:    sarama.StringEncoder("Value"),
		Metadata: trace.GetMTrace(ctx),
	}

	interceptor.OnSend(msg)

	check := false
	for _, header := range msg.Headers {
		key := string(header.Key)
		if key == saramaTraceCtx {
			check = true
		}
	}

	assert.True(check)
	assert.Contains(msg.Topic, "Topic")
	assert.Contains(msg.Key, "Key")
	assert.Contains(msg.Value, "Value")

}

func TestOnSendWithError(t *testing.T) {
	assert := assert.New(t)

	meta := "inputError"

	ctx, _ := trace.Start(context.Background(), "TEST")
	defer trace.End(ctx, nil)

	brokers := []string{"1.1.1.1:9092"}
	interceptor := Interceptor{Brokers: brokers}
	msg := &sarama.ProducerMessage{
		Topic:    "Topic",
		Key:      sarama.StringEncoder("Key"),
		Value:    sarama.StringEncoder("Value"),
		Metadata: meta,
	}

	interceptor.OnSend(msg)

	check := false
	for _, header := range msg.Headers {
		key := string(header.Key)
		if key == saramaTraceCtx {
			check = true
		}
	}

	assert.True(check)
	assert.Contains(msg.Topic, "Topic")
	assert.Contains(msg.Key, "Key")
	assert.Contains(msg.Value, "Value")

}

func TestOnConsumer(t *testing.T) {
	assert := assert.New(t)

	msg := &sarama.ConsumerMessage{
		Topic:     "Topic",
		Key:       []byte("Key"),
		Value:     []byte("Value"),
		Partition: 1,
		Offset:    2,
	}

	brokers := []string{"1.1.1.1:9092"}
	interceptor := Interceptor{Brokers: brokers}

	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(http.Header{})
	record := &sarama.RecordHeader{Key: []byte(saramaTraceCtx), Value: buf.Bytes()}
	msg.Headers = append(msg.Headers, record)

	interceptor.OnConsume(msg)

	assert.Contains(msg.Topic, "Topic")
	assert.Contains(string(msg.Key), "Key")
	assert.Contains(string(msg.Value), "Value")
	assert.Equal(msg.Partition, int32(1))
	assert.Equal(msg.Offset, int64(2))
}
