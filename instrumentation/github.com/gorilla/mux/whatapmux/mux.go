package whatapmux

import (
	"net/http"

	"github.com/gorilla/mux"

	whataptrace "github.com/whatap/go-api/trace"
)

func Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return whataptrace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
