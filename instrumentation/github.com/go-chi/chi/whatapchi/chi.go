package whatapchi

import (
	"net/http"

	"github.com/whatap/go-api/trace"
)

// useMiddleware is satisfied by chi.Router (v4/v5) or any router with a Use method.
type useMiddleware interface {
	Use(middlewares ...func(http.Handler) http.Handler)
}

// WrapRouter adds WhaTap middleware to a chi.Router (or compatible router) and returns it.
// Uses generics to avoid importing chi directly, supporting both chi/v4 and chi/v5.
// The router must have a Use method accepting func(http.Handler) http.Handler variadic args.
func WrapRouter[T any](r T) T {
	if u, ok := any(r).(useMiddleware); ok {
		u.Use(Middleware)
	}
	return r
}

func Middleware(next http.Handler) http.Handler {
	return trace.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
