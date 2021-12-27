package whataphttp

import (
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

	}
}

type WrapRoundTrip struct {
	transport http.RoundTripper
}

func (this *WrapRoundTrip) RoundTrip(req *http.Request) (res *http.Response, err error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return this.transport.RoundTrip(req)
	}
	ctx := req.Context()
	httpcCtx, _ := httpc.Start(ctx, req.URL.String())

	if conf.MtraceEnabled {
		headers := trace.GetMTrace(ctx)
		for key, _ := range headers {
			req.Header.Add(key, headers.Get(key))
		}
	}
	res, err = this.transport.RoundTrip(req)
	httpc.End(httpcCtx, res.StatusCode, "", err)

	return res, err
}

func NewRoundTrip(t http.RoundTripper) http.RoundTripper {
	return &WrapRoundTrip{t}
}
