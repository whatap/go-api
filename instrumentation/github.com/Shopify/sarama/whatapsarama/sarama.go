package whatapsarama

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/whatap/go-api/httpc"
	"github.com/whatap/go-api/trace"
)

const (
	saramaTraceCtx = "whatapTraceCtx"
)

// WrapConfig adds WhaTap interceptors to a sarama.Config and returns it.
// Use this to instrument sarama.Config created in any context (struct fields, return values, etc).
func WrapConfig(config *sarama.Config) *sarama.Config {
	if config != nil {
		interceptor := &Interceptor{}
		config.Producer.Interceptors = []sarama.ProducerInterceptor{interceptor}
		config.Consumer.Interceptors = []sarama.ConsumerInterceptor{interceptor}
	}
	return config
}

type Interceptor struct {
	Brokers []string
}

func (in *Interceptor) OnSend(msg *sarama.ProducerMessage) {
	if trace.DISABLE() {
		return
	}

	name := fmt.Sprintf("produceTransaction/%s", msg.Topic)

	var produceCtx context.Context
	metadata := msg.Metadata

	switch v := metadata.(type) {
	case http.Header:
		produceCtx, _ = trace.Start(context.Background(), name)
		defer trace.End(produceCtx, nil)
		trace.UpdateMtraceWithContext(produceCtx, v)
	case context.Context:
		produceCtx = v
	default:
		produceCtx, _ = trace.Start(context.Background(), name)
		defer trace.End(produceCtx, nil)
	}

	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(trace.GetMTrace(produceCtx))
	msg.Metadata = trace.GetMTrace(produceCtx)

	record := sarama.RecordHeader{Key: []byte(saramaTraceCtx), Value: buf.Bytes()}
	msg.Headers = append(msg.Headers, record)

	text := fmt.Sprintf("Topic : %s, Key : %s, Value : %s, Brokers : %s", msg.Topic, msg.Key, msg.Value, strings.Join(in.Brokers, " "))

	trace.Step(produceCtx, "Producer Interceptor Message", text, 1, 1)

	// Record as external call for httpc statistics
	host := strings.Join(in.Brokers, ",")
	if len(in.Brokers) > 0 {
		host = in.Brokers[0]
	}
	url := fmt.Sprintf("kafka://%s/%s", host, msg.Topic)
	httpc.Trace(produceCtx, host, 9092, url, 0, 0, "", nil)
}

func (in *Interceptor) OnConsume(msg *sarama.ConsumerMessage) {
	if trace.DISABLE() {
		return
	}

	name := fmt.Sprintf("consumeTransaction/%s", msg.Topic)
	ctx, _ := trace.Start(context.Background(), name)
	defer trace.End(ctx, nil)
	text := fmt.Sprintf("Topic : %s, Key : %s, Value : %s, Offset : %d, Time : %s", msg.Topic, string(msg.Key), string(msg.Value), msg.Offset, msg.Timestamp.String())

	h := make(http.Header)

	for _, header := range msg.Headers {
		key := string(header.Key)
		if key == saramaTraceCtx {
			buf := bytes.NewBuffer(header.Value)
			gob.NewDecoder(buf).Decode(&h)
			trace.UpdateMtraceWithContext(ctx, h)
			break
		}
	}

	trace.Step(ctx, "Consumer Interceptor Message", text, 1, 1)
}
