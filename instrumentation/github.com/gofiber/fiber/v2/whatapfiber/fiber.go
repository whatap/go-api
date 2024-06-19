package whatapfiber

import (
	"context"
	"fmt"
	"net/http"

	// "net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/whatap/go-api/agent/agent/config"

	"github.com/whatap/go-api/instrumentation/github.com/valyala/fasthttp/whatapfasthttp"
	"github.com/whatap/go-api/trace"
)

func traceParams(fiberCtx *fiber.Ctx, ctx context.Context) error {

	conf := config.GetConfig()
	// trace http parameter
	if !conf.ProfileHttpParameterEnabled {
		return nil
	}
	if !strings.HasPrefix(string(fiberCtx.Context().RequestURI()), conf.ProfileHttpParameterUrlPrefix) {
		return nil
	}

	// get parameter from fasthttp
	fasthttpRequest := fiberCtx.Context()
	form := whatapfasthttp.TraceHttpParameter(fasthttpRequest)

	// get path parameter from router of fiber
	params := fiberCtx.AllParams()
	for k, v := range params {
		nK := "router." + k
		form[nK] = append(form[nK], v)
	}

	if form != nil {
		trace.SetParameter(ctx, form)
	}

	return nil
}

func Middleware() func(c *fiber.Ctx) error {
	return func(fiberCtx *fiber.Ctx) error {
		if trace.DISABLE() {
			return fiberCtx.Next()
		}

		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			return fiberCtx.Next()
		}

		ctx, _ := whatapfasthttp.StartWithFastHttpRequest(fiberCtx.Context())
		var err error = nil
		defer func() {
			x := recover()

			if x != nil {
				err = fmt.Errorf("Panic: %v", x)
				trace.Error(ctx, err)
			}

			status := fiberCtx.Response().StatusCode()
			if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
				traceCtx.Status = int32(status)
			}

			if status >= 400 {
				err = fmt.Errorf("Status: %d,%s", status, http.StatusText(status))
			}

			traceParams(fiberCtx, ctx)

			// Set Whatap Cookie
			if conf.TraceUserSetCookie {
				if cookie, exists := whatapfasthttp.GetWhatapCookie(fiberCtx.Context()); !exists {
					whatapfasthttp.SetWhatapCookie(fiberCtx.Context(), cookie)
				}
			}

			trace.End(ctx, err)
			if x != nil {
				if !conf.GoRecoverEnabled {
					panic(x)
				}
			}
		}()

		err = fiberCtx.Next()
		return err
	}
}
