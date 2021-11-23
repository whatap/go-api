package whatapgrpc

import (
	"context"
	"fmt"
	"net"
	"path"

	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		conf := config.GetConfig()
		if !conf.GrpcProfileEnabled {
			// handler
			return handler(ctx, req)
		}

		ctx, _ = StartWithGrpcServerStream(ctx, "", info.FullMethod)
		// handler
		resp, err := handler(ctx, req)

		trace.End(ctx, err)

		return resp, err
	}
}

type wrapServerStream struct {
	grpc.ServerStream
	ctx    context.Context
	Method string
	conf   *config.Config
}

func (w *wrapServerStream) SetHeader(m metadata.MD) error {
	return w.ServerStream.SetHeader(m)
}

func (w *wrapServerStream) SendHeader(m metadata.MD) error {
	return w.ServerStream.SendHeader(m)
}
func (w *wrapServerStream) SetTrailer(m metadata.MD) {
	w.ServerStream.SetTrailer(m)
}
func (w *wrapServerStream) Context() context.Context {
	return w.ctx
}

func (w *wrapServerStream) RecvMsg(m interface{}) (err error) {
	return w.TraceStream("/RecvMsg", func() error {
		return w.ServerStream.RecvMsg(m)
	})
}

func (w *wrapServerStream) SendMsg(m interface{}) (err error) {
	return w.TraceStream("/SendMsg", func() error {
		return w.ServerStream.SendMsg(m)
	})
}

func (w *wrapServerStream) TraceStream(div string, callFunc func() error) (err error) {
	if !w.conf.GrpcProfileStreamServerEnabled {
		return callFunc()
	}
	if w.conf.GrpcProfileStreamIdentify {
		div = fmt.Sprintf("/%s%s", "StreamServer", div)
	}

	if w.conf.InArray(w.Method, w.conf.GrpcProfileStreamMethod) {
		wCtx, _ := trace.Start(w.ServerStream.Context(), path.Join(div, w.Method))
		err = callFunc()
		trace.End(wCtx, err)
	} else {
		if _, traceCtx := trace.GetTraceContext(w.ctx); traceCtx != nil {
			st := dateutil.SystemNow()
			err = callFunc()
			trace.Step(w.ctx, path.Join(div, w.Method), div, int(dateutil.SystemNow()-st), 0)
			if err != nil {
				trace.Step(w.ctx, path.Join(div, w.Method), fmt.Sprintf("Error %s", err.Error()), 0, 0)
			}
		} else {
			err = callFunc()
		}
	}
	return err
}

func newWrapServerStream(s grpc.ServerStream, ctx context.Context, method string, conf *config.Config) grpc.ServerStream {
	return &wrapServerStream{s, ctx, method, conf}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		conf := config.GetConfig()
		if conf.InArray(info.FullMethod, conf.GrpcProfileStreamMethod) {
			ctx, _ := StartWithGrpcServerStream(ss.Context(), "/Start", info.FullMethod)
			trace.End(ctx, nil)

			err = handler(srv, newWrapServerStream(ss, ctx, info.FullMethod, conf))

			ctx, _ = StartWithGrpcServerStream(ss.Context(), "/End", info.FullMethod)
			trace.End(ctx, err)
		} else {
			ctx, _ := StartWithGrpcServerStream(ss.Context(), "", info.FullMethod)

			err = handler(srv, newWrapServerStream(ss, ctx, info.FullMethod, conf))

			trace.End(ctx, err)
		}
		return err
	}
}

func StartWithGrpcServerStream(ctx context.Context, div string, fullMethod string) (context.Context, error) {
	conf := config.GetConfig()
	ctx, traceCtx := trace.NewTraceContext(ctx)
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		traceCtx.Host = GetValueFromMetadata(md, ":authority")
		traceCtx.UAgent = GetValueFromMetadata(md, "user-agent")
	}
	traceCtx.HttpMethod = "grpc"
	if grpcPeer, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := grpcPeer.Addr.(*net.TCPAddr); ok {
			traceCtx.Ipaddr = tcpAddr.IP.String()
		}
	}
	traceCtx.WClientId = traceCtx.Ipaddr

	if conf.GrpcProfileStreamIdentify {
		div = fmt.Sprintf("/%s%s", "StreamServer", div)
	}

	ctx, err := trace.StartWithContext(ctx, path.Join(div, fullMethod))
	trace.UpdateMtraceWithContext(ctx, map[string][]string(md))
	trace.SetHeader(ctx, map[string][]string(md))

	mt := trace.GetMTrace(ctx)
	md, ok = metadata.FromOutgoingContext(ctx)
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

	return ctx, err
}
func GetValueFromMetadata(md metadata.MD, k string) string {
	if md == nil {
		return ""
	}
	if v, ok := md[k]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}
