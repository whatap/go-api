package whatapaws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/util/dateutil"
)

func TestUnit(t *testing.T) {
	assert := assert.New(t)
	whatapConfig := map[string]string{
		"net_udp_port": "6600",
		"debug":        "true",
	}
	trace.Init(whatapConfig)
	defer trace.Shutdown()
	ctx := context.TODO()

	t.Run("Trace start", func(t *testing.T) {
		in := middleware.InitializeInput{}
		outRaw, _, err := StartTrace(ctx, in, MockHandler{})
		assert.Nil(err)

		outCtx, typeMatched := outRaw.Result.(context.Context)
		assert.True(typeMatched)
		assert.NotNil(outCtx)

		traceCtxRaw := middleware.GetStackValue(outCtx, TraceKey{})
		assert.NotNil(traceCtxRaw)
		traceCtx, typeMatched := traceCtxRaw.(*trace.TraceCtx)
		if assert.True(typeMatched) {
			assert.NotNil(traceCtx)
			assert.Greater(len(traceCtx.Name), 0)
		}
		ctx = outCtx
	},
	)

	t.Run("End trace", func(t *testing.T) {
		in := middleware.DeserializeInput{}
		outRaw, _, err := EndTrace(ctx, in, MockHandler{})
		assert.Nil(err)

		outCtx, typeMatched := outRaw.Result.(context.Context)
		assert.True(typeMatched)
		assert.NotNil(outCtx)

		traceCtxRaw := middleware.GetStackValue(outCtx, TraceKey{})
		assert.NotNil(traceCtxRaw)
		traceCtx, typeMatched := traceCtxRaw.(*trace.TraceCtx)
		if assert.True(typeMatched) {

			assert.NotNil(traceCtx)
			//clear 됐는지 테스트
			assert.Equal(int64(0), traceCtx.Txid)
			assert.Equal("", traceCtx.Name)
		}

	},
	)

	t.Run("don't overwrite existing traceCtx", func(t *testing.T) {
		traceCtx := &trace.TraceCtx{
			Txid:      int64(-1),
			Name:      "skynet",
			StartTime: dateutil.SystemNow(),
		}
		ctx := context.WithValue(context.TODO(), "whatap", traceCtx)

		in := middleware.InitializeInput{}
		outRaw, _, err := StartTrace(ctx, in, MockHandler{})
		assert.Nil(err)
		outCtx := outRaw.Result.(context.Context)
		if traced, ok := outCtx.Value("whatap").(*trace.TraceCtx); ok {
			assert.Equal(int64(-1), traced.Txid)
			assert.Equal("skynet", traced.Name)
		}
	},
	)

	t.Run("Middleware Stack", func(t *testing.T) {
		awsCfg := AppendMiddleware(aws.Config{})
		assert.NotNil(awsCfg)
		assert.Equal(2, len(awsCfg.APIOptions))
		stack := &middleware.Stack{
			Initialize:  middleware.NewInitializeStep(),
			Deserialize: middleware.NewDeserializeStep(),
		}
		err := AddStartTrace(stack)
		assert.Nil(err)
		_, idFound := stack.Initialize.Get(TraceStartFuncName)
		assert.True(idFound)

		err = AddEndTrace(stack)
		assert.Nil(err)
		_, idFound = stack.Deserialize.Get(TraceEndFuncName)
		assert.True(idFound)
	},
	)

	t.Run("Integration", func(t *testing.T) {
		// Load the Shared AWS Configuration (~/.aws/config)
		cfg, err := config.LoadDefaultConfig(context.TODO())
		assert.Nil(err)
		cfg = AppendMiddleware(cfg)
		assert.Equal(2, len(cfg.APIOptions))

		// Create an Amazon S3 service client
		client := s3.NewFromConfig(cfg)

		if !assert.NotNil(client) {
			return
		}

		output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
			Bucket: aws.String("dev-default-region"),
		})
		if !assert.Nil(err) {
			return
		}

		t.Log("first page results:")
		for _, object := range output.Contents {
			t.Logf("key=%s size=%d", aws.ToString(object.Key), object.Size)
		}
	},
	)
}
