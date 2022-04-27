package whatapecho

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			conf := config.GetConfig()
			if !conf.TransactionEnabled {
				return next(c)
			}
			r := c.Request()
			ctx, _ := trace.StartWithRequest(r)
			c.SetRequest(r.WithContext(ctx))
			var err error = nil
			defer func() {
				x := recover()
				if x != nil {
					err = fmt.Errorf("Panic: %v", x)
					trace.Error(ctx, err)
					err = nil
				}
				status := 200
				if err != nil {
					// reference echo.DefaultHttpErrorHandler
					he, ok := err.(*echo.HTTPError)
					if ok {
						if he.Internal != nil {
							if herr, ok := he.Internal.(*echo.HTTPError); ok {
								he = herr
							}
						}
					} else {
						he = &echo.HTTPError{
							Code:    http.StatusInternalServerError,
							Message: http.StatusText(http.StatusInternalServerError),
						}
					}
					status = he.Code
					if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
						traceCtx.Status = int32(status)
					}
					trace.Error(ctx, err)
					err = nil
				}
				if c.Response().Committed {
					status = c.Response().Status
					if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
						traceCtx.Status = int32(status)
					}
					if status >= 400 {
						err = fmt.Errorf("Status: %d,%s", status, http.StatusText(status))
					}
				}

				trace.End(ctx, err)
				if x != nil {
					panic(x)
				}
			}()
			err = next(c)
			return err
		}
	}
}
func WrapHTTPErrorHandler(handler echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		trace.Error(c.Request().Context(), err)
		if handler != nil {
			handler(err, c)
		}
	}
}
