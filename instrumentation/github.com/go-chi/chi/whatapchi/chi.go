package whatapchi

import (
	"net/http"

	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Middleware(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		} else {
			return trace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}

	}(next)

}
