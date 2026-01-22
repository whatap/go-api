// httpc/wrap.go - 범용 Wrap 함수 (Go 1.18+ Generics)
// 외부 HTTP API 호출 계측용
package httpc

import (
	"context"
)

// WrapError - error만 반환하는 HTTP 호출
//
// 사용 예시:
//
//	if err := httpc.WrapError(ctx, "https://api.example.com/webhook", func() error {
//	    return sendWebhook(payload)
//	}); err != nil { ... }
func WrapError(ctx context.Context, url string, fn func() error) error {
	httpcCtx, _ := Start(ctx, url)
	err := fn()
	End(httpcCtx, 0, "", err)
	return err
}

// Wrap - (T, error) 반환하는 HTTP 호출
//
// 사용 예시:
//
//	resp, err := httpc.Wrap(ctx, "https://api.payment.com/charge", func() (*PaymentResp, error) {
//	    return paymentClient.Charge(amount)
//	})
func Wrap[T any](ctx context.Context, url string, fn func() (T, error)) (T, error) {
	httpcCtx, _ := Start(ctx, url)
	result, err := fn()
	End(httpcCtx, 0, "", err)
	return result, err
}

// WrapWithStatus - HTTP 상태 코드도 기록
//
// 사용 예시:
//
//	resp, status, err := httpc.WrapWithStatus(ctx, "https://api.example.com/data",
//	    func() (*Response, int, error) {
//	        resp, err := client.Get(url)
//	        if resp != nil {
//	            return resp, resp.StatusCode, err
//	        }
//	        return nil, 0, err
//	    },
//	)
func WrapWithStatus[T any](ctx context.Context, url string, fn func() (T, int, error)) (T, int, error) {
	httpcCtx, _ := Start(ctx, url)
	result, status, err := fn()
	End(httpcCtx, status, "", err)
	return result, status, err
}
