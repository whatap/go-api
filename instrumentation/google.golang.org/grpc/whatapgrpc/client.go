package whatapgrpc

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/httpc"
	"github.com/whatap/go-api/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		conf := config.GetConfig()
		if !conf.GrpcProfileEnabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		wCtx, _ := httpc.Start(ctx, fmt.Sprintf("grpc://%s%s", strings.TrimSpace(cc.Target()), strings.TrimSpace(method)))
		mt := httpc.GetMTrace(wCtx)
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			for k, v := range mt {
				md.Append(k, v...)
			}
		} else {
			newMd := metadata.New(make(map[string]string))
			for k, v := range mt {
				newMd.Append(k, v...)
			}
			ctx = metadata.NewOutgoingContext(ctx, newMd)
		}

		err := invoker(ctx, method, req, reply, cc, opts...)

		httpc.End(wCtx, 0, "", err)
		return err
	}
}

type wrapClientStream struct {
	grpc.ClientStream
	ctx    context.Context
	Method string
	Target string
	conf   *config.Config
}

func (w *wrapClientStream) Header() (metadata.MD, error) {
	md, err := w.ClientStream.Header()
	return md, err
}

func (w *wrapClientStream) CloseSend() error {
	err := w.ClientStream.CloseSend()
	return err
}

func (w *wrapClientStream) RecvMsg(m interface{}) (err error) {
	return w.TraceStream("/RecvMsg", func() error {
		return w.ClientStream.RecvMsg(m)
	})
}

func (w *wrapClientStream) SendMsg(m interface{}) (err error) {
	return w.TraceStream("/SendMsg", func() error {
		return w.ClientStream.SendMsg(m)
	})
}

func (w *wrapClientStream) TraceStream(div string, callFunc func() error) (err error) {
	if !w.conf.GrpcProfileStreamClientEnabled {
		return callFunc()
	}
	if w.conf.GrpcProfileStreamIdentify {
		div = fmt.Sprintf("/%s%s", "StreamClient", div)
	}

	if w.conf.InArray(w.Method, w.conf.GrpcProfileStreamMethod) {
		wCtx, _ := trace.Start(w.ClientStream.Context(), path.Join(div, w.Target, w.Method))
		err = callFunc()
		trace.End(wCtx, err)
	} else {
		if _, traceCtx := trace.GetTraceContext(w.ctx); traceCtx != nil {
			st := dateutil.SystemNow()
			err = callFunc()
			trace.Step(w.ctx, path.Join(div, w.Target, w.Method), "", int(dateutil.SystemNow()-st), 0)
			if err != nil {
				trace.Step(w.ctx, path.Join(div, w.Target, w.Method), fmt.Sprintf("Error %s", err.Error()), 0, 0)
			}
		} else {
			err = callFunc()
		}
	}
	return err
}
func newWrapClientStream(s grpc.ClientStream, ctx context.Context, method, target string, conf *config.Config) grpc.ClientStream {
	return &wrapClientStream{s, ctx, method, target, conf}
}

func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
		streamer grpc.Streamer, opts ...grpc.CallOption) (s grpc.ClientStream, err error) {

		conf := config.GetConfig()

		div := "/Start"
		if conf.GrpcProfileStreamIdentify {
			div = fmt.Sprintf("/%s%s", "StreamClient", div)
		}
		if conf.InArray(method, conf.GrpcProfileStreamMethod) {
			ctx, _ := trace.Start(ctx, path.Join(div, cc.Target(), method))
			// stream
			s, err = streamer(ctx, desc, cc, method, opts...)
			trace.End(ctx, err)
		} else {
			st := dateutil.SystemNow()
			ctx, traceCtx := trace.GetTraceContext(ctx)
			if traceCtx != nil {
				mt := trace.GetMTrace(ctx)
				md, ok := metadata.FromOutgoingContext(ctx)
				if ok {
					for k, v := range mt {
						md.Append(k, v...)
					}
				} else {
					newMd := metadata.New(make(map[string]string))
					for k, v := range mt {
						newMd.Append(k, v...)
					}
					ctx = metadata.NewOutgoingContext(ctx, newMd)
				}
			}
			s, err = streamer(ctx, desc, cc, method, opts...)

			if ctx, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
				trace.Step(ctx, path.Join(div, cc.Target(), method), "", int(dateutil.SystemNow()-st), 0)
				if err != nil {
					trace.Step(ctx, path.Join(div, cc.Target(), method), fmt.Sprintf("Error %s", err.Error()), 0, 0)
				}
			}
		}
		return newWrapClientStream(s, ctx, method, cc.Target(), conf), err
	}
}
