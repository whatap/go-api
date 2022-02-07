package whataphttp

import (
	"context"
	"net/http"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/httpc"
	"github.com/whatap/go-api/trace"
)

// wrapping type of http.HanderFunc, example : http.Handle(pattern, http.HandlerFunc)
func HandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			handler(w, r)
			return
		}
		ctx, _ := trace.StartWithRequest(r)
		defer trace.End(ctx, nil)
		handler(w, r.WithContext(ctx))
	})
}

// wrapping handler function, example : http.HandleFunc(func(http.ResponseWriter, *http.Request))
func Func(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			handler(w, r)
			return
		}
		ctx, _ := trace.StartWithRequest(r)
		defer trace.End(ctx, nil)
		handler(w, r.WithContext(ctx))
	}
}

type WrapRoundTrip struct {
	transport http.RoundTripper
	ctx       context.Context
}

func (this *WrapRoundTrip) RoundTrip(req *http.Request) (res *http.Response, err error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return this.transport.RoundTrip(req)
	}
	ctx := req.Context()
	wCtx := selectContext(ctx, this.ctx)
	httpcCtx, _ := httpc.Start(wCtx, req.URL.String())
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(wCtx)
		for key, _ := range headers {
			req.Header.Add(key, headers.Get(key))
		}
	}
	res, err = this.transport.RoundTrip(req)
	httpc.End(httpcCtx, res.StatusCode, "", err)

	return res, err
}

func NewRoundTrip(ctx context.Context, t http.RoundTripper) http.RoundTripper {
	return &WrapRoundTrip{t, ctx}
}

func selectContext(contexts ...context.Context) (ctx context.Context) {
	var first context.Context
	for i, it := range contexts {
		if i == 0 {
			first = it
		}
		if _, traceCtx := trace.GetTraceContext(it); traceCtx != nil {
			return it
		}
	}
	return first
}
