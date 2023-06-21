package api

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/golib/lang/step"
)

func TestStartHttpc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)
}

func TestStartHttpcNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(nil, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)
}

func TestStartHttpcValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	startTime = 0
	url = ""
	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)
}

func TestEndHttpc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)

	elapsed := int32(123)
	status := int32(200)
	reason := ""
	cpu := int64(0)
	mem := int64(1)
	mcallee := int64(0)
	var err error = nil

	EndHttpc(ctx, st, elapsed, status, reason, cpu, mem, mcallee, err)

}

func TestEndHttpcValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	startTime = 0
	url = ""

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)

	elapsed := int32(123)
	status := int32(200)
	reason := ""
	cpu := int64(0)
	mem := int64(1)
	mcallee := int64(0)
	var err error = nil

	elapsed = 0
	status = 0
	reason = ""
	cpu = 0
	mem = 0
	err = nil

	EndHttpc(ctx, st, elapsed, status, reason, cpu, mem, mcallee, err)

}

func TestEndHttpcError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)

	elapsed := int32(123)
	status := int32(200)
	reason := ""
	cpu := int64(0)
	mem := int64(1)
	mcallee := int64(0)
	var err error = nil
	err = fmt.Errorf("Error throw error")

	EndHttpc(ctx, st, elapsed, status, reason, cpu, mem, mcallee, err)

}

func TestEndHttpcNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)

	elapsed := int32(123)
	status := int32(200)
	reason := ""
	cpu := int64(0)
	mem := int64(1)
	mcallee := int64(0)
	var err error = nil

	EndHttpc(nil, st, elapsed, status, reason, cpu, mem, mcallee, err)

}

func TestEndHttpcNilStep(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "http://www.url.com/wuuu---1"

	st := StartHttpc(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewHttpcStepX(), st)

	elapsed := int32(123)
	status := int32(200)
	reason := ""
	cpu := int64(0)
	mem := int64(1)
	mcallee := int64(0)
	var err error = nil

	EndHttpc(ctx, nil, elapsed, status, reason, cpu, mem, mcallee, err)

}
