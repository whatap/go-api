package awsv2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	"github.com/whatap/go-api/trace"
)

//string 같은 builtin 타입을 Key로 사용하면 충돌 가능성 있으므로 커스텀 구조체 사용
type TraceKey struct{}

const (
	TraceStartFuncName = "StartTrace"
	TraceEndFuncName   = "EndTrace"
)

func AppendMiddleware(cfg aws.Config) aws.Config {
	cfg.APIOptions = append(cfg.APIOptions, AddStartTrace, AddEndTrace)
	return cfg
}

func AddStartTrace(stack *middleware.Stack) error {
	return stack.Initialize.Add(
		middleware.InitializeMiddlewareFunc(TraceStartFuncName, StartTrace),
		middleware.After,
	)
}

func AddEndTrace(stack *middleware.Stack) error {
	return stack.Deserialize.Add(
		middleware.DeserializeMiddlewareFunc(TraceEndFuncName, EndTrace),
		middleware.After,
	)
}

func StartTrace(ctx context.Context,
	in middleware.InitializeInput,
	next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {

	startCtx := func(ctx context.Context) context.Context {
		if traceCtxExist(ctx) {
			return ctx
		}
		traceCtx, err := makeTraceCtx(ctx)
		if err != nil {
			return ctx
		}
		return middleware.WithStackValue(ctx, TraceKey{}, traceCtx)
	}(ctx)

	trace.SetParameter(startCtx, map[string][]string{
		"region":    []string{awsmiddleware.GetRegion(ctx)},
		"service":   []string{awsmiddleware.GetServiceID(ctx)},
		"operation": []string{awsmiddleware.GetOperationName(ctx)},
	})

	return next.HandleInitialize(startCtx, in)
}

func EndTrace(ctx context.Context,
	in middleware.DeserializeInput,
	next middleware.DeserializeHandler) (middleware.DeserializeOutput, middleware.Metadata, error) {

	endCtx, wasStartedInStack := func(ctx context.Context) (context.Context, bool) {
		traceCtxRaw := middleware.GetStackValue(ctx, TraceKey{})
		if traceCtxRaw == nil {
			return ctx, false
		}
		traceCtx, typeMatched := traceCtxRaw.(*trace.TraceCtx)
		if !typeMatched {
			return ctx, false
		}
		return context.WithValue(ctx, "whatap", traceCtx), true
	}(ctx)

	if wasStartedInStack {
		trace.End(endCtx, nil)
	}

	return next.HandleDeserialize(endCtx, in)
}

func traceCtxExist(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	valRaw := ctx.Value("whatap")
	if valRaw == nil {
		return false
	}
	_, typeMatched := valRaw.(*trace.TraceCtx)
	return typeMatched
}

//WARN: Initialize.Add 함수의 인자를 middleware.After로 줘야 GetServiceID, GetOperationName이 정상 동작
//StartTrace를 실행하기 전에 다른 미들웨어가 실행되면 실행시간 측정이 정확하지 않게 됨
//aws.Config.APIOptions에 다른 미들웨어를 넣기 전에 whatapaws.AppendMiddleware를 무조건 먼저 실행해야 함
func getTxName(ctx context.Context) string {
	region := awsmiddleware.GetRegion(ctx)
	serviceID := awsmiddleware.GetServiceID(ctx)
	operation := awsmiddleware.GetOperationName(ctx)
	return fmt.Sprintf("%s.%s.%s", region, serviceID, operation)
}

func makeTraceCtx(ctx context.Context) (*trace.TraceCtx, error) {
	startCtx, err := trace.Start(ctx, getTxName(ctx))
	if err != nil {
		return nil, err
	}
	return startCtx.Value("whatap").(*trace.TraceCtx), nil
}
