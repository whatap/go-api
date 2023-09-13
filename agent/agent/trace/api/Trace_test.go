package api

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"

	// "github.com/whatap/golib/lang/step"

	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/urlutil"
)

func TestStartTx(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.StartTime = int64(123456789)
	ctx.Txid = 12345
	ctx.ServiceURL = urlutil.NewURL("http://aaa.bbb.com/index")
	assert.NotNil(t, ctx.ServiceURL)
	conf := agentconfig.GetConfig()
	assert.NotNil(t, conf.TraceIgnoreUrlSet)
	StartTx(ctx)
	assert.Equal(t, hash.HashStr(ctx.ServiceURL.Path), ctx.ServiceHash)

	// assert.IsType(t, *step.HttpcStepX, st)
}

func TestStartTxValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.StartTime = int64(123456789)
	ctx.Txid = 12345
	ctx.ServiceURL = urlutil.NewURL("http://aaa.bbb.com/index")

	StartTx(ctx)

	// assert.IsType(t, *step.HttpcStepX, st)
}

func TestStartTxNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.StartTime = int64(123456789)
	ctx.Txid = 12345
	ctx.ServiceURL = urlutil.NewURL("http://aaa.bbb.com/index")

	StartTx(nil)

	// assert.IsType(t, *step.HttpcStepX, st)
}

func TestEndTx(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.Txid = 12345
	ctx.ServiceURL = urlutil.NewURL("http://aaa.bbb.com/index")

	StartTx(ctx)

	EndTx(ctx)

	// assert.IsType(t, *step.HttpcStepX, st)
}

func TestEndTxNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.Txid = 12345
	ctx.ServiceURL = urlutil.NewURL("http://aaa.bbb.com/index")

	StartTx(ctx)

	EndTx(nil)

	// assert.IsType(t, *step.HttpcStepX, st)
}

func TestProfileMsg(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
}

func TestProfileSecureMsg(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
}

func TestProfileError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.Txid = 12345

	StartTx(ctx)
	ProfileError(ctx, fmt.Errorf("Error thorw error"))
	EndTx(ctx)

	assert.NotEqual(t, 0, ctx.Error)
}

func TestProfileErrorNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := agenttrace.PoolTraceContext()
	assert.NotNil(t, ctx)
	ctx.Txid = 12345

	StartTx(ctx)
	ProfileError(nil, fmt.Errorf("Error thorw error"))
	//h := hash.HashStr("Error throw error")
	EndTx(ctx)

	//assert.Equal(t, h, ctx.Error)
}
