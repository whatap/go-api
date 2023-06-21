package api

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/golib/lang/step"
)

func TestStartMethod(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	method := "method---1"

	st := StartMethod(ctx, startTime, method)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)
}

func TestStartMethodValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	method := "method---1"

	startTime = 0
	method = ""

	st := StartMethod(ctx, startTime, method)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)
}

func TestStartMethodNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "method---1"

	st := StartMethod(nil, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)
}

func TestEndMethod(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "method---1"

	st := StartMethod(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)

	methodStack := ""
	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	EndMethod(ctx, st, methodStack, elapsed, cpu, mem, err)
}

func TestEndMethodValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	method := "method---1"

	startTime = 0
	method = ""

	st := StartMethod(ctx, startTime, method)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)

	methodStack := ""
	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	elapsed = 0
	cpu = 0
	mem = 0
	err = nil

	EndMethod(ctx, st, methodStack, elapsed, cpu, mem, err)
}

func TestEndMethodError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	method := "method---1"

	startTime = 0
	method = ""

	st := StartMethod(ctx, startTime, method)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)

	methodStack := ""
	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	err = fmt.Errorf("Error throw error")

	EndMethod(ctx, st, methodStack, elapsed, cpu, mem, err)
}

func TestEndMethodNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "method---1"

	st := StartMethod(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)

	methodStack := ""
	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	EndMethod(nil, st, methodStack, elapsed, cpu, mem, err)
}

func TestEndMethodNilStep(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, debug.Stack())
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	url := "method---1"

	st := StartMethod(ctx, startTime, url)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewMethodStepX(), st)

	methodStack := ""
	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	EndMethod(ctx, nil, methodStack, elapsed, cpu, mem, err)
}
