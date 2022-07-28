package whatapchi

import (
	"net/http"

	"github.com/whatap/go-api/trace"
)

func Middleware(next http.Handler) http.Handler {
	return trace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
