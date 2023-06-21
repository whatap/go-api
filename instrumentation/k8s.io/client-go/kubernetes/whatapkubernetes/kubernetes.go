package whatapkubernetes

import (
	"context"
	"net/http"

	"github.com/whatap/go-api/instrumentation/net/http/whataphttp"
)

func WrapRoundTripper() func(t http.RoundTripper) http.RoundTripper {
	return func(t http.RoundTripper) http.RoundTripper {
		wRT := whataphttp.NewWrapRoundTrip(context.Background(), t)
		return wRT
	}
}

func WrapRoundTripperWithContext(ctx context.Context) func(t http.RoundTripper) http.RoundTripper {
	return func(t http.RoundTripper) http.RoundTripper {
		wRT := whataphttp.NewWrapRoundTrip(ctx, t)
		return wRT
	}
}
