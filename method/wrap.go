// method/wrap.go - 범용 Wrap 함수 (Go 1.18+ Generics)
// 사용자 정의 메서드/함수 추적용
package method

import (
	"context"
)

// WrapError - error만 반환하는 메서드
//
// 사용 예시:
//
//	if err := method.WrapError(ctx, "OrderService.ValidateOrder", func() error {
//	    return s.validateOrder(order)
//	}); err != nil { ... }
func WrapError(ctx context.Context, methodName string, fn func() error) error {
	methodCtx, _ := Start(ctx, methodName)
	err := fn()
	End(methodCtx, err)
	return err
}

// Wrap - (T, error) 반환하는 메서드
//
// 사용 예시:
//
//	order, err := method.Wrap(ctx, "OrderService.ProcessOrder", func() (*Order, error) {
//	    return s.processOrderInternal(orderID)
//	})
func Wrap[T any](ctx context.Context, methodName string, fn func() (T, error)) (T, error) {
	methodCtx, _ := Start(ctx, methodName)
	result, err := fn()
	End(methodCtx, err)
	return result, err
}

// WrapVoid - 반환값 없는 메서드
//
// 사용 예시:
//
//	method.WrapVoid(ctx, "CacheService.Invalidate", func() {
//	    s.cache.Delete(key)
//	})
func WrapVoid(ctx context.Context, methodName string, fn func()) {
	methodCtx, _ := Start(ctx, methodName)
	fn()
	End(methodCtx, nil)
}
