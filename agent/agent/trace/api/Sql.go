package api

import (
	"runtime/debug"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/stat"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/stringutil"
)

func StartDBC(ctx *agenttrace.TraceContext, startTime int64, dbhost string) *step.DBCStep {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11110", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	st := step.NewDBCStep()
	st.Hash = hash.HashStr(dbhost)
	data.SendHashText(pack.TEXT_DB_URL, st.Hash, dbhost)
	data.SendHashText(pack.TEXT_METHOD, st.Hash, dbhost)

	// Active status
	if ctx != nil {
		st.StartTime = int32(startTime - ctx.StartTime)
		ctx.ActiveDbc = st.Hash
	}
	return st
}

func EndDBC(ctx *agenttrace.TraceContext, st *step.DBCStep, elapsed int32, cpu, mem int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11120", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()
	if ctx == nil || st == nil {
		return
	}

	st.Elapsed = elapsed

	thr := ErrorToThr(err)

	if thr != nil {
		ProfileErrorStep(thr, ctx)

		msg := stringutil.TrimEmpty(thr.ErrorMessage)
		msg = stringutil.Truncate(msg, 200)
		msgHash := hash.HashStr(msg)
		data.SendHashText(pack.TEXT_ERROR, msgHash, msg)
		st.Error = msgHash
	}

	ctx.DbcTime += st.Elapsed
	ctx.ActiveDbc = 0

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}
}

func StartSql(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql, sqlParam string) *step.SqlStepX {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11130", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	conf := agentconfig.GetConfig()
	st := step.NewSqlStepX()
	st.Dbc = hash.HashStr(dbhost)
	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, dbhost)

	psql := agenttrace.EscapeLiteral(sql)
	if psql == nil {
		st.Hash = hash.HashStr(sql)
		psql = agenttrace.NewParsedSql('*', st.Hash, "")
	} else {
		st.Hash = psql.Sql
	}

	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
	if psql != nil {
		switch psql.Type {
		case 'S':
			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
		case 'U':
			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
		default:
			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
		}
	}

	// SQL Param Encrypt.
	if conf.ProfileSqlParamEnabled && psql != nil {
		crc := ref.NewBYTE()
		st.P1 = agenttrace.ToParamBytes(psql.Param, crc)
		// bind
		if sqlParam != "" {
			st.P2 = agenttrace.ToParamBytes(sqlParam, crc)
		}
		st.Pcrc = crc.Value
	}

	if ctx != nil {
		st.StartTime = int32(startTime - ctx.StartTime)
		ctx.ActiveSqlhash = st.Hash
	}
	return st
}

func EndSql(ctx *agenttrace.TraceContext, st *step.SqlStepX, elapsed int32, cpu, mem int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11140", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	if st == nil {
		return
	}

	conf := agentconfig.GetConfig()
	st.Elapsed = elapsed

	thr := ErrorToThr(err)
	if ctx == nil {
		// 통계만 추가
		if thr != nil {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)
			//thr.ErrorStack = stackToArray(p.Stack)
			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
		}
		return
	}

	if conf.ProfileSqlResourceEnabled {
		st.StartCpu = int32(cpu)
		st.StartMem = mem
	}

	if thr != nil {
		ProfileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = thr.ErrorStack
	}

	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
	ctx.ActiveSqlhash = 0
	ctx.ActiveDbc = 0

	ctx.SqlCount++
	ctx.SqlTime += st.Elapsed

	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}
}
func ProfileSql(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql, sqlParam string, elapsed int32, cpu, mem int64, err error) {
	st := StartSql(ctx, startTime, dbhost, sql, sqlParam)
	EndSql(ctx, st, elapsed, cpu, mem, err)
}

// SQL End
func ProfileSql1(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql, sqlParam string, elapsed int32, cpu, mem int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA-API11150", " Recover ", r, "/n", string(debug.Stack()))
		}
	}()

	conf := agentconfig.GetConfig()
	st := step.NewSqlStepX()
	st.Dbc = hash.HashStr(dbhost)
	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, dbhost)
	st.Elapsed = elapsed

	psql := agenttrace.EscapeLiteral(sql)
	if psql == nil {
		st.Hash = hash.HashStr(sql)
		psql = agenttrace.NewParsedSql('*', st.Hash, "")
	} else {
		st.Hash = psql.Sql
	}

	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
	if psql != nil {
		switch psql.Type {
		case 'S':
			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
		case 'U':
			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
		default:
			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
		}
	}

	thr := ErrorToThr(err)
	if ctx == nil {
		// 통계만 추가
		if thr != nil {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)
			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
		} else {
			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
		}
		return
	}

	st.StartTime = int32(startTime - ctx.StartTime)

	// SQL Param Encrypt 추가.
	if conf.ProfileSqlParamEnabled && psql != nil {
		crc := ref.NewBYTE()
		st.P1 = agenttrace.ToParamBytes(psql.Param, crc)
		// bind
		if sqlParam != "" {
			st.P2 = agenttrace.ToParamBytes(sqlParam, crc)
		}
		st.Pcrc = crc.Value
	}

	if conf.ProfileSqlResourceEnabled {
		st.StartCpu = int32(cpu)
		st.StartMem = mem
	}

	if thr != nil {
		ProfileErrorStep(thr, ctx)
		st.Error = ctx.Error
		st.Stack = thr.ErrorStack
	}

	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
	ctx.ActiveSqlhash = 0
	ctx.ActiveDbc = 0

	ctx.SqlCount++
	ctx.SqlTime += st.Elapsed

	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

	if st.Error == 0 {
		ctx.Profile.Add(st)
	} else {
		ctx.Profile.AddHeavy(st)
	}
}
