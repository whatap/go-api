package whataphttp

import (
	"context"
	//	"fmt"
	"io"
	//	"log"
	"net/http"
	"net/url"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/httpc"
	"github.com/whatap/go-api/trace"
)

// wrapping type of http.HanderFunc, example : http.Handle(pattern, http.HandlerFunc)
func HandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(trace.Func(handler))
}

// wrapping handler function, example : http.HandleFunc(func(http.ResponseWriter, *http.Request))
func Func(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return trace.Func(handler)
}

type WrapRoundTrip struct {
	ctx       context.Context
	transport http.RoundTripper
}

func (this *WrapRoundTrip) RoundTrip(req *http.Request) (res *http.Response, err error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return this.transport.RoundTrip(req)
	}
	if this.transport == nil {
		this.transport = http.DefaultTransport
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
	if res != nil {
		httpc.End(httpcCtx, res.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return res, err
}
func NewWrapRoundTrip(ctx context.Context, t http.RoundTripper) *WrapRoundTrip {
	return &WrapRoundTrip{ctx, t}
}

func NewRoundTrip(ctx context.Context, t http.RoundTripper) http.RoundTripper {
	return &WrapRoundTrip{ctx, t}
}

func HttpGet(ctx context.Context, url string) (*http.Response, error) {
	httpcCtx, _ := httpc.Start(ctx, url)
	resp, err := http.Get(url)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

func HttpPost(ctx context.Context, url string, contentType string, body io.Reader) (*http.Response, error) {
	httpcCtx, _ := httpc.Start(ctx, url)
	resp, err := http.Post(url, contentType, body)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

func HttpPostForm(ctx context.Context, url string, data url.Values) (*http.Response, error) {
	httpcCtx, _ := httpc.Start(ctx, url)
	resp, err := http.PostForm(url, data)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
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
