package whatapsarama

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/whatap/go-api/trace"
)

const (
	saramaTraceCtx = "whatapTraceCtx"
)

type Interceptor struct {
	Brokers []string
}

func (in *Interceptor) OnSend(msg *sarama.ProducerMessage) {
	if trace.DISABLE() {
		return
	}

	name := fmt.Sprintf("produceTransaction/%s", msg.Topic)

	var produceCtx context.Context
	var ok bool
	metadata := msg.Metadata

	if metadata != nil {
		switch reflect.TypeOf(metadata).String() {
		case "http.Header":
			header, ok := metadata.(http.Header)
			if ok != true {
				fmt.Println("http.Header Covert Error!")
			}

			produceCtx, _ = trace.Start(context.Background(), name)
			defer trace.End(produceCtx, nil)

			trace.UpdateMtraceWithContext(produceCtx, header)

		case "*context.emptyCtx":
			produceCtx, ok = metadata.(context.Context)
			if ok != true {
				fmt.Println("context Covert Error!")
			}
		default:
			produceCtx, _ = trace.Start(context.Background(), name)
			defer trace.End(produceCtx, nil)
		}
	} else {
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
}

func (in *Interceptor) OnConsume(msg *sarama.ConsumerMessage) {
	if trace.DISABLE() {
		return
	}

	name := fmt.Sprintf("consumeTransaction/%s", msg.Topic)
	ctx, _ := trace.Start(context.Background(), name)
	defer trace.End(ctx, nil)
	text := fmt.Sprintf("Topic : %s, Key : %s, Vaslue : %s, Offset : %d, Time : %s", msg.Topic, string(msg.Key), string(msg.Value), msg.Offset, msg.Timestamp.String())

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
