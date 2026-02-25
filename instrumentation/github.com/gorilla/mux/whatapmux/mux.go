package whatapmux

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/whatap/go-api/trace"
)

// WrapRouter adds WhaTap middleware to a mux.Router and returns it.
// Use this to instrument mux.Router created in any context (struct fields, return values, etc).
func WrapRouter(r *mux.Router) *mux.Router {
	if r != nil {
		r.Use(Middleware())
	}
	return r
}

// Middleware returns a middleware function for gorilla/mux.
// Returns func(http.Handler) http.Handler instead of mux.MiddlewareFunc
// for compatibility with forks like containous/mux (used by traefik).
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return trace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
