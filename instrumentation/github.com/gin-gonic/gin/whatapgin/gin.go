package whatapgin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			c.Next()
			return
		}
		ctx, _ := trace.StartWithRequest(c.Request)
		c.Request = c.Request.WithContext(ctx)

		defer func() {
			x := recover()
			var err error = nil

			if len(c.Errors) > 0 {
				err = fmt.Errorf("Errors: %s", c.Errors.String())
				trace.Error(ctx, err)
				err = nil
			}
			if x != nil {
				err = fmt.Errorf("Panic: %v", x)
				trace.Error(ctx, err)
				err = nil
			}
			status := c.Writer.Status()
			if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
				traceCtx.Status = int32(status)
			}
			if status >= 400 {
				err = fmt.Errorf("Status: %d,%s", status, http.StatusText(status))
			}

			// trace http parameter
			if conf.ProfileHttpParameterEnabled && strings.HasPrefix(c.Request.RequestURI, conf.ProfileHttpParameterUrlPrefix) {
				if c.Request.Form != nil {
					trace.SetParameter(ctx, c.Request.Form)
				}
			}
			trace.End(ctx, err)
			if x != nil {
				panic(x)
			}
		}()
		c.Next()
	}
}
