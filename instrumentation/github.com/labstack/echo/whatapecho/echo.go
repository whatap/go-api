package whatapecho

import (
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
			err := next(c)
			trace.End(ctx, err)
			return err
		}
	}
}
