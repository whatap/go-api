package whatapecho

import (
	"github.com/labstack/echo"

	whataptrace "github.com/whatap/go-api/trace"
)

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, _ := whataptrace.StartWithRequest(c.Request())
			c.SetRequest(r.WithContext(ctx))
			err := next(c)
			whataptrace.End(ctx, err)
			return err
		}
	}
}
