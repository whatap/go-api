package whatapgin

import (
	"fmt"
	"net/http"

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

		c.Next()

		var err error
		status := c.Writer.Status()
		if status >= 500 && status < 600 {
			err = fmt.Errorf("%d: %s", status, http.StatusText(status))
			trace.Error(ctx, err)
		}

		if len(c.Errors) > 0 {
			err = fmt.Errorf("Errors: %s", c.Errors.String())
			trace.Error(ctx, err)
		}
		trace.End(ctx, err)
	}
}
