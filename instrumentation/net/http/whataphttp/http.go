package whataphttp

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

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
	if this.transport == nil {
		this.transport = http.DefaultTransport
	}

	if trace.DISABLE() {
		return this.transport.RoundTrip(req)
	}

	conf := config.GetConfig()
	if !conf.Enabled {
		return this.transport.RoundTrip(req)
	}

	ctx := req.Context()
	wCtx := selectContext(ctx, this.ctx)
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(wCtx)
		for key, _ := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}
	httpcCtx, _ := httpc.Start(wCtx, req.URL.String())
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

func HttpGet(ctx context.Context, urlStr string) (*http.Response, error) {
	if trace.DISABLE() {
		return http.Get(urlStr)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

func HttpPost(ctx context.Context, urlStr string, contentType string, body io.Reader) (*http.Response, error) {
	if trace.DISABLE() {
		return http.Post(urlStr, contentType, body)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, body)
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

func HttpPostForm(ctx context.Context, urlStr string, data url.Values) (*http.Response, error) {
	if trace.DISABLE() {
		return http.PostForm(urlStr, data)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

// DefaultClientGet wraps http.DefaultClient.Get() for instrumentation.
// This marker function indicates the original code used http.DefaultClient.Get()
// instead of http.Get(), allowing perfect restoration on removal.
func DefaultClientGet(ctx context.Context, urlStr string) (*http.Response, error) {
	if trace.DISABLE() {
		return http.DefaultClient.Get(urlStr)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

// DefaultClientPost wraps http.DefaultClient.Post() for instrumentation.
// This marker function indicates the original code used http.DefaultClient.Post()
// instead of http.Post(), allowing perfect restoration on removal.
func DefaultClientPost(ctx context.Context, urlStr string, contentType string, body io.Reader) (*http.Response, error) {
	if trace.DISABLE() {
		return http.DefaultClient.Post(urlStr, contentType, body)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, body)
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

// DefaultClientPostForm wraps http.DefaultClient.PostForm() for instrumentation.
// This marker function indicates the original code used http.DefaultClient.PostForm()
// instead of http.PostForm(), allowing perfect restoration on removal.
func DefaultClientPostForm(ctx context.Context, urlStr string, data url.Values) (*http.Response, error) {
	if trace.DISABLE() {
		return http.DefaultClient.PostForm(urlStr, data)
	}

	httpcCtx, _ := httpc.Start(ctx, urlStr)

	// Create request with mtrace headers for distributed tracing
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		httpc.End(httpcCtx, -1, "", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Set mtrace headers (same pattern as WrapRoundTrip)
	conf := config.GetConfig()
	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key := range headers {
			req.Header.Set(key, headers.Get(key))
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		httpc.End(httpcCtx, resp.StatusCode, "", err)
	} else {
		httpc.End(httpcCtx, -1, "", err)
	}
	return resp, err
}

// NewRoundTripWithEmptyTransport creates a RoundTripper wrapper for an http.Client
// that originally had no Transport field (empty http.Client{}).
// This marker function allows perfect restoration on removal by indicating
// the Transport field should be removed entirely rather than restored to http.DefaultTransport.
func NewRoundTripWithEmptyTransport(ctx context.Context) http.RoundTripper {
	return &WrapRoundTrip{ctx, http.DefaultTransport}
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
