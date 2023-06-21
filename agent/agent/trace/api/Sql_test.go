package api

import (
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/golib/lang/step"
)

func TestStartDBC(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)
}

func TestStartDBCValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	startTime = 0
	dbhost = ""

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)
}

func TestEndDBC(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)

	elapsed := int32(33)
	cpu := int64(0)
	mem := int64(0)
	var err error = nil

	EndDBC(ctx, st, elapsed, cpu, mem, err)
}

func TestEndDBCValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	startTime = 0
	dbhost = ""

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)

	elapsed := int32(33)
	cpu := int64(0)
	mem := int64(0)
	var err error = nil

	EndDBC(ctx, st, elapsed, cpu, mem, err)
}

func TestEndDBCError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)

	elapsed := int32(33)
	cpu := int64(0)
	mem := int64(0)
	var err error = nil
	err = fmt.Errorf("Error throw error")

	EndDBC(ctx, st, elapsed, cpu, mem, err)
}

func TestEndDBCNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)

	elapsed := int32(33)
	cpu := int64(0)
	mem := int64(0)
	var err error = nil

	EndDBC(nil, st, elapsed, cpu, mem, err)
}

func TestEndDBCNilStep(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()

	ctx := trace.PoolTraceContext()
	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"

	st := StartDBC(ctx, startTime, dbhost)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewDBCStep(), st)

	elapsed := int32(33)
	cpu := int64(0)
	mem := int64(0)
	var err error = nil

	EndDBC(ctx, nil, elapsed, cpu, mem, err)
}

func TestStartSql(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)
}

func TestStartSqlValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	startTime = 0
	dbhost = ""
	sql = ""
	sqlParam = ""

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)
}

func TestStartSqlNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	st := StartSql(nil, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)
}

func TestEndSql(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)

	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil
	EndSql(ctx, st, elapsed, cpu, mem, err)
}

func TestEndSqlValidate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	startTime = 0
	dbhost = ""
	sql = ""
	sqlParam = ""

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)

	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil

	elapsed = 0
	cpu = 0
	mem = 0

	EndSql(ctx, st, elapsed, cpu, mem, err)
}

func TestEndSqlNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)

	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil
	EndSql(nil, st, elapsed, cpu, mem, err)
}

func TestEndSqlNilStep(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code is panic, %v\n stack=%s", r, string(debug.Stack()))
		}
	}()
	ctx := trace.PoolTraceContext()

	ctx.Txid = 12345

	startTime := int64(123456789)
	dbhost := "dbhost---1"
	sql := "select * from aaa where a=3 and t like '%aaa%'"
	sqlParam := "aaa, bbb, ccc"

	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	assert.NotNil(t, st)
	assert.IsType(t, step.NewSqlStepX(), st)

	elapsed := int32(123)
	cpu := int64(0)
	mem := int64(1)
	var err error = nil
	EndSql(ctx, nil, elapsed, cpu, mem, err)
}
