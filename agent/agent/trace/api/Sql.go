package api

import (
	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/stat"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/ref"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/stringutil"
)

func StartDBC(ctx *agenttrace.TraceContext, startTime int64, dbhost string) *step.DBCStep {
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

// //func ProfileDBC(ctx *agenttrace.TraceContext, dbhost, sql string, param []interface{}, elapsed int, err error) {
// func ProfileDBC(ctx *agenttrace.TraceContext, startTime int64, dbhost string, elapsed int32, cpu, mem int64, err error) {
// 	// ctx interface 변환 전에 먼저 nil 체크, 기존 panic 보완
// 	if ctx == nil {
// 		return
// 	}

// 	st := step.NewDBCStep()
// 	st.Hash = hash.HashStr(dbhost)
// 	data.SendHashText(pack.TEXT_DB_URL, st.Hash, dbhost)
// 	// JAVA 에서 DBCStep 은 TEXT 를 METHOD 로 가져옴.   agent.trace.JdbcUrls -> DataTextAgent.addMethod(dbc_hash, url);
// 	data.SendHashText(pack.TEXT_METHOD, st.Hash, dbhost)

// 	st.StartTime = int32(startTime - ctx.StartTime)
// 	st.Elapsed = elapsed

// 	thr := ErrorToThr(err)

// 	if thr != nil {
// 		ProfileErrorStep(thr, ctx)

// 		msg := stringutil.TrimEmpty(thr.ErrorMessage)
// 		msg = stringutil.Truncate(msg, 200)
// 		msgHash := hash.HashStr(msg)
// 		data.SendHashText(pack.TEXT_ERROR, msgHash, msg)
// 		st.Error = msgHash
// 	}

// 	//log.Println("DBC", st.Hash, "," , p.Dbc, "\n,"\n,step=", st)
// 	ctx.DbcTime += st.Elapsed
// 	if st.Error == 0 {
// 		ctx.Profile.Add(st)
// 	} else {
// 		ctx.Profile.AddHeavy(st)
// 	}
// }

func StartSql(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql, sqlParam string) *step.SqlStepX {
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

// // SQL End
// func ProfileSql1(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql string, elapsed int32, cpu, mem int64, err error) {
// 	conf := agentconfig.GetConfig()
// 	st := step.NewSqlStepX()
// 	st.Dbc = hash.HashStr(dbhost)
// 	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, dbhost)
// 	st.Elapsed = elapsed

// 	// DEBUG 일단 제외
// 	// TODO Configure 설정 추가
// 	//if (st.Elapsed > conf.Profile_error_sql_time_max) {
// 	//	errHash := StatError.getInstance().addError(SLOW_SQL.o, SLOW_SQL.o.getMessage(), ctx.service_hash, ctx.profile, TextTypes.SQL, step.hash)
// 	//	if (ctx.Error == 0) {
// 	//		ctx.Error = hash
// 	//		}
// 	//	st.Error = hash
// 	//	}
// 	//var crud byte
// 	psql := agenttrace.EscapeLiteral(sql)
// 	if psql == nil {
// 		st.Hash = hash.HashStr(sql)
// 		psql = agenttrace.NewParsedSql('*', st.Hash, "")
// 		// SqlStep_3
// 		//crud = ' '
// 	} else {
// 		st.Hash = psql.Sql
// 		// SqlStep_3
// 		//crud = psql.Type
// 	}

// 	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
// 	if psql != nil {
// 		switch psql.Type {
// 		case 'S':
// 			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
// 		case 'U':
// 			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
// 		default:
// 			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
// 		}
// 	}

// 	thr := ErrorToThr(err)
// 	if ctx == nil {
// 		// 통계만 추가
// 		if thr != nil {
// 			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)
// 			//thr.ErrorStack = stackToArray(p.Stack)
// 			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
// 		} else {
// 			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
// 		}
// 		return
// 	}

// 	st.StartTime = int32(startTime - ctx.StartTime)

// 	// SQL Param Encrypt 추가.
// 	if conf.ProfileSqlParamEnabled && psql != nil {
// 		// SqlStep_3
// 		//st.SetTrue(1)
// 		crc := ref.NewBYTE()
// 		//logutil.Println("Before Encrypt ", psql.Param)
// 		st.P1 = agenttrace.ToParamBytes(psql.Param, crc)
// 		//logutil.Println("Encrypt ", string(st.P1), ", Crc=", crc.Value , st.P2 )
// 		st.Pcrc = crc.Value
// 	}

// 	if conf.ProfileSqlResourceEnabled {
// 		// SqlStep_3
// 		//		st.SetTrue(2)
// 		//		st.Cpu = int32(p.Cpu)
// 		//		st.Mem = int32(p.Mem)
// 		st.StartCpu = int32(cpu)
// 		st.StartMem = mem
// 	}

// 	if thr != nil {
// 		//thr.ErrorStack = stackToArray(p.Stack)

// 		ProfileErrorStep(thr, ctx)
// 		st.Error = ctx.Error
// 		//st.Stack = thr.ErrorStack
// 	}

// 	// TODO
// 	//	st.Xtype = (byte) (st.Xtype | xtype);
// 	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
// 	ctx.ActiveSqlhash = 0
// 	ctx.ActiveDbc = 0

// 	ctx.SqlCount++
// 	ctx.SqlTime += st.Elapsed

// 	//ctx.Active_sqlhash = st.Hash;
// 	//ctx.Active_dbc = st.Dbc;

// 	// TODO
// 	//ctx.Active_crud =	psql.Type
// 	// TODO
// 	//	if conf.Profile_sql_param_enabled && psql != null {
// 	//		switch (psql.Type) {
// 	//			case 'S': fallthrough
// 	//			case 'D': fallthrough
// 	//			case 'U':
// 	//				BYTE crc = new BYTE();
// 	//				st.setTrue(1);
// 	//				st.p1 = toParamBytes(psql.param, crc);
// 	//				st.pcrc = crc.value;
// 	//
// 	//		}
// 	//	}
// 	// st.Crud;

// 	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
// 	// import cycle 내부 addSqlTime으로 변경, stat.GetInstanceStatSql().AddSqlTime  -> addSqlTime
// 	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

// 	if st.Error == 0 {
// 		ctx.Profile.Add(st)
// 	} else {
// 		ctx.Profile.AddHeavy(st)
// 	}

// 	//log.Printf("TraceContextManager:ProfileSql: END hsah=%d, DBC %d, %s", st.Hash, st.Dbc, p.Dbc)
// }

// // SQL End
// func ProfileSqlParam1(ctx *agenttrace.TraceContext, startTime int64, dbhost, sql, sqlParam string, elapsed int32, cpu, mem int64, err error) {
// 	conf := agentconfig.GetConfig()
// 	st := step.NewSqlStepX()
// 	st.Dbc = hash.HashStr(dbhost)
// 	st.Elapsed = elapsed
// 	st.Hash = hash.HashStr(sql)
// 	data.SendHashText(pack.TEXT_SQL, st.Hash, sql)

// 	psql := agenttrace.EscapeLiteral(sql)
// 	if psql == nil {
// 		st.Hash = hash.HashStr(sql)
// 		psql = agenttrace.NewParsedSql('*', st.Hash, "")
// 	} else {
// 		st.Hash = psql.Sql
// 	}

// 	st.Xtype = step.SQL_XTYPE_METHOD_QUERY
// 	if psql != nil {
// 		switch psql.Type {
// 		case 'S':
// 			st.Xtype = step.SQL_XTYPE_METHOD_QUERY
// 		case 'U':
// 			st.Xtype = step.SQL_XTYPE_METHOD_UPDATE
// 		default:
// 			st.Xtype = step.SQL_XTYPE_METHOD_EXECUTE
// 		}
// 	}

// 	thr := ErrorToThr(err)
// 	if ctx == nil {
// 		// 통계만 추가
// 		if thr != nil {
// 			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, true)
// 			//thr.ErrorStack = stackToArray(p.Stack)
// 			stat.GetInstanceStatError().AddError(thr, thr.ErrorMessage, 0, false, nil, 0, 0)
// 		} else {
// 			stat.GetInstanceStatSql().AddSqlTime(0, st.Dbc, st.Hash, st.Elapsed, false)
// 		}
// 		return
// 	}

// 	st.StartTime = int32(startTime - ctx.StartTime)

// 	// SQL Param Encrypt 추가.
// 	if conf.ProfileSqlParamEnabled && psql != nil {
// 		//st.SetTrue(1)
// 		crc := ref.NewBYTE()
// 		st.P1 = agenttrace.ToParamBytes(psql.Param, crc)
// 		// bind
// 		if sqlParam != "" {
// 			st.P2 = agenttrace.ToParamBytes(sqlParam, crc)
// 			//logutil.Println("Encrypt ", string(st.P2), ", Crc=", crc.Value, p.Param)
// 		}
// 		st.Pcrc = crc.Value
// 	}

// 	if conf.ProfileSqlResourceEnabled {
// 		// SqlStep_3
// 		//		st.SetTrue(2)
// 		//		st.Cpu = int32(p.Cpu)
// 		//		st.Mem = int32(p.Mem)
// 		st.StartCpu = int32(cpu)
// 		st.StartMem = mem
// 	}

// 	if thr != nil {
// 		//thr.ErrorStack = stackToArray(p.Stack)

// 		ProfileErrorStep(thr, ctx)
// 		st.Error = ctx.Error
// 		st.Stack = thr.ErrorStack
// 	}

// 	// TODO
// 	//	st.Xtype = (byte) (st.Xtype | xtype);
// 	ctx.ExecutedSqlhash = ctx.ActiveSqlhash
// 	ctx.ActiveSqlhash = 0
// 	ctx.ActiveDbc = 0

// 	ctx.SqlCount++
// 	ctx.SqlTime += st.Elapsed

// 	//ctx.Active_sqlhash = st.Hash;
// 	//ctx.Active_dbc = st.Dbc;

// 	// TODO
// 	//ctx.Active_crud =	psql.Type
// 	// TODO
// 	//	if conf.Profile_sql_param_enabled && psql != null {
// 	//		switch (psql.Type) {
// 	//			case 'S': fallthrough
// 	//			case 'D': fallthrough
// 	//			case 'U':
// 	//				BYTE crc = new BYTE();
// 	//				st.setTrue(1);
// 	//				st.p1 = toParamBytes(psql.param, crc);
// 	//				st.pcrc = crc.value;
// 	//
// 	//		}
// 	//	}
// 	// st.Crud;

// 	meter.GetInstanceMeterSQL().Add(st.Dbc, st.Elapsed, (st.Error != 0))
// 	// import cycle 내부 addSqlTime으로 변경, stat.GetInstanceStatSql().AddSqlTime  -> addSqlTime
// 	stat.GetInstanceStatSql().AddSqlTime(ctx.ServiceHash, st.Dbc, st.Hash, st.Elapsed, st.Error != 0)

// 	if st.Error == 0 {
// 		ctx.Profile.Add(st)
// 	} else {
// 		ctx.Profile.AddHeavy(st)
// 	}

// 	data.SendHashText(pack.TEXT_DB_URL, st.Dbc, dbhost)

// 	//log.Printf("TraceContextManager:ProfileSqlParam: END hsah=%d, DBC %d, %s", st.Hash, st.Dbc, p.Dbc)
// }
