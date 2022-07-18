package whatapchi

import (
	"fmt"
	"net/http"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Middleware(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conf := config.GetConfig()
			if !conf.TransactionEnabled {
				next.ServeHTTP(w, r)
				return
			}

			ctx, _ := trace.StartWithRequest(r)
			wrw := &trace.WrapResponseWriter{ResponseWriter: w}

			defer func() {
				x := recover()
				var err error = nil

				if x != nil {
					err = fmt.Errorf("Panic: %v", x)
					trace.Error(ctx, err)
					err = nil
				}
				status := wrw.Status

				if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
					traceCtx.Status = int32(status)
				}

				if status >= 400 {
					err = fmt.Errorf("Status : %d, %s", status, http.StatusText(status))
				}
				trace.End(ctx, err)

				if x != nil {
					panic(x)
				}
			}()

			r = r.WithContext(ctx)
			next.ServeHTTP(wrw, r)
		})
	}(next)
}
