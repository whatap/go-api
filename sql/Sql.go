// github.com/whatap/go-api/sql
package sql

import (
	"bytes"
	"context"
	"database/sql/driver"
	"fmt"

	"log"

	// "reflect"
	//"runtime/debug"
	"strings"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	agentapi "github.com/whatap/go-api/agent/agent/trace/api"
	"github.com/whatap/go-api/trace"

	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/stringutil"
)

const (
	SQL_TYPE_DBC       = 1
	SQL_TYPE_SQL       = 2
	SQL_TYPE_SQL_PARAM = 3

	SQL_PARAM_MAX_COUNT      = 20
	SQL_PARAM_VALUE_MAX_SIZE = 256
)

func StartOpen(ctx context.Context, dbhost string) (*SqlCtx, error) {
	if trace.DISABLE() {
		return PoolSqlContext(), nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return PoolSqlContext(), nil
	}
	sqlCtx := PoolSqlContext()
	var wCtx *agenttrace.TraceContext
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		sqlCtx.ctx = traceCtx
		sqlCtx.Txid = traceCtx.Txid
		sqlCtx.ServiceName = traceCtx.Name
		wCtx = traceCtx.Ctx
	}
	sqlCtx.StartTime = dateutil.SystemNow()
	sqlCtx.Dbc = hidePwd(dbhost)
	sqlCtx.Type = SQL_TYPE_DBC

	sqlCtx.step = agentapi.StartDBC(wCtx, sqlCtx.StartTime, sqlCtx.Dbc)
	return sqlCtx, nil
}

func Start(ctx context.Context, dbhost, sql string) (*SqlCtx, error) {
	if trace.DISABLE() {
		return PoolSqlContext(), nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return PoolSqlContext(), nil
	}
	sqlCtx := PoolSqlContext()
	var wCtx *agenttrace.TraceContext
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		sqlCtx.ctx = traceCtx
		sqlCtx.Txid = traceCtx.Txid
		sqlCtx.ServiceName = traceCtx.Name
		wCtx = traceCtx.Ctx
	}
	sqlCtx.StartTime = dateutil.SystemNow()
	sqlCtx.Dbc = hidePwd(dbhost)
	sqlCtx.Sql = sql
	sqlCtx.Type = SQL_TYPE_SQL

	sqlCtx.step = agentapi.StartSql(wCtx, sqlCtx.StartTime, sqlCtx.Dbc, sqlCtx.Sql, "")

	return sqlCtx, nil
}

func StartWithParam(ctx context.Context, dbhost, sql string, param ...interface{}) (*SqlCtx, error) {
	if trace.DISABLE() {
		return PoolSqlContext(), nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return PoolSqlContext(), nil
	}
	sqlCtx := PoolSqlContext()
	var wCtx *agenttrace.TraceContext
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		sqlCtx.ctx = traceCtx
		sqlCtx.Txid = traceCtx.Txid
		sqlCtx.ServiceName = traceCtx.Name
		wCtx = traceCtx.Ctx
	}
	sqlCtx.StartTime = dateutil.SystemNow()
	sqlCtx.Dbc = hidePwd(dbhost)
	sqlCtx.Sql = sql
	if conf.ProfileSqlParamEnabled {
		sqlCtx.Type = SQL_TYPE_SQL
	} else {
		sqlCtx.Type = SQL_TYPE_SQL_PARAM
		sqlCtx.Param = paramsToString(param...)
	}

	sqlCtx.step = agentapi.StartSql(wCtx, sqlCtx.StartTime, sqlCtx.Dbc, sqlCtx.Sql, sqlCtx.Param)
	return sqlCtx, nil
}

func StartWithParamArray(ctx context.Context, dbhost, sql string, param []interface{}) (*SqlCtx, error) {
	return StartWithParam(ctx, dbhost, sql, param...)
}

func End(sqlCtx *SqlCtx, err error) error {
	if trace.DISABLE() {
		return nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}
	// driver.ErrSkip is not collected.
	if err == driver.ErrSkip {
		if conf.Debug {
			log.Println("[WA-SQL-04001] End: Error Skip ", err)
		}
		//return nil
		err = nil
	}

	if sqlCtx != nil && sqlCtx.step != nil {
		elapsed := int32(dateutil.SystemNow() - sqlCtx.StartTime)
		wCtx := trace.GetAgentTraceContext(sqlCtx.ctx)

		switch sqlCtx.Type {
		case SQL_TYPE_DBC:
			//agentapi.ProfileDBC(wCtx, sqlCtx.StartTime, sqlCtx.Dbc, elapsed, sqlCtx.Cpu, sqlCtx.Mem, err)
			if st, ok := sqlCtx.step.(*step.DBCStep); ok {
				agentapi.EndDBC(wCtx, st, elapsed, sqlCtx.Cpu, sqlCtx.Mem, err)
			}
			if conf.Debug {
				log.Println("[WA-SQL-04002] Open DB txid: ", sqlCtx.Txid, ", uri: ", sqlCtx.ServiceName, "\n dbhost: ", sqlCtx.Dbc, "\n time: ", elapsed, "ms ", "\n error: ", err)
			}
		case SQL_TYPE_SQL, SQL_TYPE_SQL_PARAM:
			//agentapi.ProfileSql(wCtx, sqlCtx.StartTime, sqlCtx.Dbc, sqlCtx.Sql, elapsed, sqlCtx.Cpu, sqlCtx.Mem, err)
			if st, ok := sqlCtx.step.(*step.SqlStepX); ok {
				agentapi.EndSql(wCtx, st, elapsed, sqlCtx.Cpu, sqlCtx.Mem, err)
			}
			if conf.Debug {
				log.Println("[WA-SQL-04003] Sql txid: ", sqlCtx.Txid, ", uri: ", sqlCtx.ServiceName, "\n dbhost: ", sqlCtx.Dbc, "\n sql: ", sqlCtx.Sql, "\n time: ", elapsed, "ms ", "\n error: ", err)
			}
		}

		CloseSqlContext(sqlCtx)
		return nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-04005] End SqlCtx is nil: ", "\n error: ", err)
	}
	return fmt.Errorf("SqlCtx is nil")
}

func Trace(ctx context.Context, dbhost, sql string, param []interface{}, elapsed int, err error) error {
	if trace.DISABLE() {
		return nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}
	var txid int64
	var serviceName string
	var wCtx *agenttrace.TraceContext
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		wCtx = traceCtx.Ctx
		txid = traceCtx.Txid
		serviceName = traceCtx.Name
	}
	sqlParam := paramsToString(param...)
	dbhost = hidePwd(dbhost)
	//udpClient := whatapnet.GetUdpClient()
	if conf.ProfileSqlParamEnabled && (param != nil && len(param) > 0) {
		if conf.Debug {
			log.Println("[WA-SQL-05001] txid: ", txid, ", uri: ", serviceName, "\n dbhost: ", dbhost, "\n sql: ", sql, "\n args: ", sqlParam, "\n time: ", elapsed, "ms ", "\n error: ", err)
		}
		agentapi.ProfileSql(wCtx, dateutil.SystemNow(), dbhost, sql, sqlParam, int32(elapsed), 0, 0, err)
	} else {
		if conf.Debug {
			log.Println("[WA-SQL-05002] txid: ", txid, ", uri: ", serviceName, "\n dbhost: ", dbhost, "\n sql: ", sql, "\n time: ", int32(elapsed), "ms ", "\n error: ", err)
		}
		agentapi.ProfileSql(wCtx, dateutil.SystemNow(), dbhost, sql, "", int32(elapsed), 0, 0, err)
	}
	return nil
}

func paramsToString(params ...interface{}) string {
	var buffer bytes.Buffer
	var val interface{}
	for i, v := range params {
		p, ok := v.(driver.NamedValue)
		if ok {
			val = p.Value
		} else {
			val = v
		}
		if i < SQL_PARAM_MAX_COUNT {
			if i < len(params)-1 || i < SQL_PARAM_MAX_COUNT-1 {
				switch t := val.(type) {
				case string:
					buffer.WriteString(fmt.Sprintf("'%s',", stringutil.Truncate(t, SQL_PARAM_VALUE_MAX_SIZE)))
				default:
					str := fmt.Sprintf("%v,", val)
					buffer.WriteString(stringutil.Truncate(str, SQL_PARAM_VALUE_MAX_SIZE))
				}

			} else {
				switch t := val.(type) {
				case string:
					buffer.WriteString(fmt.Sprintf("'%s'", stringutil.Truncate(t, SQL_PARAM_VALUE_MAX_SIZE)))
				default:
					str := fmt.Sprintf("%v", val)
					buffer.WriteString(stringutil.Truncate(str, SQL_PARAM_VALUE_MAX_SIZE))
				}
			}
		}
	}
	return string(buffer.Bytes())
}

func hidePwd(connStr string) string {
	if first := strings.Index(connStr, ":"); first > -1 {
		last := strings.Index(connStr[first:], "@")
		if last > -1 && first < last {
			return fmt.Sprintf("%s:#%s", connStr[0:first], (connStr[first:])[last:])
		}
	}
	if first := strings.Index(connStr, "password="); first > -1 {
		last := strings.Index(connStr[first:], " ")
		if last > -1 && first < last {
			return fmt.Sprintf("%spassword=#%s", connStr[0:first], (connStr[first:])[last:])
		}
	}

	return connStr
}
