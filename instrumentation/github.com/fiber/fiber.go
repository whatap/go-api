package whatapfiber

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/whatap/go-api/config"

	whatapfasthttp "github.com/whatap/go-api/instrumentation/github.com/fasthttp"
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

	form, err := url.ParseQuery(string(fiberCtx.Body()))
	if err != nil {
		return err
	}
	if form != nil {
		trace.SetParameter(ctx, form)
	}

	return nil
}

func Middleware() func(c *fiber.Ctx) error {
	return func(fiberCtx *fiber.Ctx) error {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			fiberCtx.Next()
			return nil
		}

		ctx, _ := whatapfasthttp.StartWithFastHttpRequest(fiberCtx.Context())

		defer func() {
			x := recover()

			if x != nil {
				err := fmt.Errorf("Panic: %v", x)
				trace.Error(ctx, err)
			}
			traceParams(fiberCtx, ctx)

			status := fiberCtx.Response().StatusCode()
			if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
				traceCtx.Status = int32(status)
			}

			err := func() error {
				if status >= 400 {
					return fmt.Errorf("Status: %d,%s", status, http.StatusText(status))
				}
				return nil
			}()

			trace.End(ctx, err)
			if x != nil {
				panic(x)
			}
		}()

		fiberCtx.Next()
		return nil
	}
}
