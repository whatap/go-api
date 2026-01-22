// trace/wrap.go - 범용 Wrap 함수 (Go 1.18+ Generics)
// 특정 도메인에 속하지 않는 범용 추적용
package trace

import (
	"context"
)

// WrapError - 범용 error 반환 함수
//
// 사용 예시:
//
//	if err := trace.WrapError(ctx, "FileProcessor.Process", func() error {
//	    return processFile(path)
//	}); err != nil { ... }
func WrapError(ctx context.Context, name string, fn func() error) error {
	ctx, _ = Start(ctx, name)
	err := fn()
	End(ctx, err)
	return err
}

// Wrap - 범용 (T, error) 반환 함수
//
// 사용 예시:
//
//	result, err := trace.Wrap(ctx, "ThirdParty.Calculate", func() (int, error) {
//	    return thirdPartyLib.Calculate(input)
//	})
func Wrap[T any](ctx context.Context, name string, fn func() (T, error)) (T, error) {
	ctx, _ = Start(ctx, name)
	result, err := fn()
	End(ctx, err)
	return result, err
}

// WrapVoid - 범용 void 함수
//
// 사용 예시:
//
//	trace.WrapVoid(ctx, "Cleanup.Execute", func() {
//	    cleanup()
//	})
func WrapVoid(ctx context.Context, name string, fn func()) {
	ctx, _ = Start(ctx, name)
	fn()
	End(ctx, nil)
}
