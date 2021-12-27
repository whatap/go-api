package whatapmux

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Middleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
		return trace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
