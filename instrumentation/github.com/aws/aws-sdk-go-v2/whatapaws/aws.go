package whatapaws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	"github.com/aws/smithy-go/transport/http"
	"github.com/whatap/go-api/httpc"
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
		middleware.Before,
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

	// startCtx := func(ctx context.Context) context.Context {
	// 	if traceCtxExist(ctx) {
	// 		return ctx
	// 	}
	// 	traceCtx, err := makeTraceCtx(ctx)
	// 	if err != nil {
	// 		return ctx
	// 	}
	// 	return middleware.WithStackValue(ctx, TraceKey{}, traceCtx)
	// }(ctx)

	// trace.SetParameter(startCtx, map[string][]string{
	// 	"region":    []string{awsmiddleware.GetRegion(ctx)},
	// 	"service":   []string{awsmiddleware.GetServiceID(ctx)},
	// 	"operation": []string{awsmiddleware.GetOperationName(ctx)},
	// })

	fmt.Println("Middleware name=", getTxName(ctx))
	httpcCtx, _ := httpc.Start(ctx, getTxName(ctx))
	startCtx := middleware.WithStackValue(ctx, TraceKey{}, httpcCtx)

	out, metadata, err := next.HandleInitialize(startCtx, in)
	fmt.Println("Middleware error=", err)
	if err != nil {
		trace.Step(ctx, "error", err.Error(), 0, 0)
	}
	return out, metadata, err
}

func EndTrace(ctx context.Context,
	in middleware.DeserializeInput,
	next middleware.DeserializeHandler) (middleware.DeserializeOutput, middleware.Metadata, error) {

	// endCtx, wasStartedInStack := func(ctx context.Context) (context.Context, bool) {
	// 	traceCtxRaw := middleware.GetStackValue(ctx, TraceKey{})
	// 	if traceCtxRaw == nil {
	// 		return ctx, false
	// 	}
	// 	traceCtx, typeMatched := traceCtxRaw.(*trace.TraceCtx)
	// 	if !typeMatched {
	// 		return ctx, false
	// 	}
	// 	return context.WithValue(ctx, "whatap", traceCtx), true
	// }(ctx)

	// out, metadata, err := next.HandleDeserialize(endCtx, in)
	// httpc.Trace(ctx, "", 0, getTxName(ctx), 0, 200, "", err)

	// if wasStartedInStack {
	// 	trace.End(endCtx, err)
	// }

	// Get values out of the request.

	httpcURL := ""
	if req, ok := in.Request.(*http.Request); ok {
		// span.SetTag(ext.HTTPMethod, req.Method)
		// span.SetTag(ext.HTTPURL, req.URL.String())
		// span.SetTag(tagAWSAgent, req.Header.Get("User-Agent"))
		httpcURL = req.URL.String()
		fmt.Println("Middleware deserialize url=", httpcURL)
	}

	out, metadata, err := next.HandleDeserialize(ctx, in)
	// Get values out of the response.
	statusCode := 0
	if res, ok := out.RawResponse.(*http.Response); ok {
		statusCode = res.StatusCode
		// span.SetTag(ext.HTTPCode, res.StatusCode)
	}

	fmt.Println("Middleware end ========================")
	fmt.Println("out=", out)
	fmt.Println("metadata=", metadata)
	fmt.Println("err=", err)

	if traceCtx := middleware.GetStackValue(ctx, TraceKey{}); traceCtx != nil {
		if httpcCtx, ok := traceCtx.(*httpc.HttpcCtx); ok {
			httpc.End(httpcCtx, statusCode, "", err)
		}
	}

	return out, metadata, err
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
