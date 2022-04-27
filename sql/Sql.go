//github.com/whatap/go-api/sql
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

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/common/util/stringutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

const (
	PACKET_DB_MAX_SIZE           = 4 * 1024  // max size of sql
	PACKET_SQL_MAX_SIZE          = 32 * 1024 // max size of sql
	PACKET_HTTPC_MAX_SIZE        = 32 * 1024 // max size of sql
	PACKET_MESSAGE_MAX_SIZE      = 32 * 1024 // max size of message
	PACKET_METHOD_STACK_MAX_SIZE = 32 * 1024 // max size of message

	COMPILE_FILE_MAX_SIZE = 2 * 1024 // max size of filename

	HTTP_HOST_MAX_SIZE   = 2 * 1024 // max size of host
	HTTP_URI_MAX_SIZE    = 2 * 1024 // max size of uri
	HTTP_METHOD_MAX_SIZE = 256      // max size of method
	HTTP_IP_MAX_SIZE     = 256      // max size of ip(request_addr)
	HTTP_UA_MAX_SIZE     = 2 * 1024 // max size of user agent
	HTTP_REF_MAX_SIZE    = 2 * 1024 // max size of referer
	HTTP_USERID_MAX_SIZE = 2 * 1024 // max size of userid

	HTTP_PARAM_MAX_COUNT      = 20
	HTTP_PARAM_KEY_MAX_SIZE   = 255 // = 을 빼고 255 byte
	HTTP_PARAM_VALUE_MAX_SIZE = 256

	HTTP_HEADER_MAX_COUNT      = 20
	HTTP_HEADER_KEY_MAX_SIZE   = 255 // = 을 빼고 255 byte
	HTTP_HEADER_VALUE_MAX_SIZE = 256

	SQL_PARAM_MAX_COUNT      = 20
	SQL_PARAM_VALUE_MAX_SIZE = 256

	STEP_ERROR_MESSAGE_MAX_SIZE = 4 * 1024
)

func Start(ctx context.Context, dbhost, sql string) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		traceCtx.ActiveSQL = true
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = traceCtx
		if pack := udp.CreatePack(udp.TX_SQL, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = stringutil.Truncate(hidePwd(dbhost), PACKET_DB_MAX_SIZE)
			p.Sql = stringutil.Truncate(sql, PACKET_SQL_MAX_SIZE)
			sqlCtx.step = p
			// if conf.Debug {
			// 	log.Println("[WA-SQL-01001] Start: ", traceCtx.Txid, ", ", traceCtx.Name, "\n", dbhost, "\n", sql)
			// }
		}
		return sqlCtx, nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-01002] Start: Not found Txid ",
			"\n dbhost: ", dbhost, "\n sql: ", sql)
	}
	return nil, fmt.Errorf("Not found Txid ")
}

func StartOpen(ctx context.Context, dbhost string) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		traceCtx.ActiveDBC = true
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = traceCtx
		if pack := udp.CreatePack(udp.TX_DB_CONN, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxDbcPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = stringutil.Truncate(hidePwd(dbhost), PACKET_DB_MAX_SIZE)
			sqlCtx.step = p
			// if conf.Debug {
			// 	log.Println("[WA-SQL-02001] Start Connection: ", traceCtx.Txid, ", ", traceCtx.Name, "\n", dbhost)
			// }
		}
		return sqlCtx, nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-02001] StartOpen: Not found Txid ", "\n dbhost: ", dbhost)
	}
	return nil, fmt.Errorf("Not found Txid ")
}

func StartWithParam(ctx context.Context, dbhost, sql string, param ...interface{}) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		traceCtx.ActiveSQL = true
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = traceCtx
		if pack := udp.CreatePack(udp.TX_SQL_PARAM, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlParamPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = stringutil.Truncate(hidePwd(dbhost), PACKET_DB_MAX_SIZE)
			p.Sql = stringutil.Truncate(sql, PACKET_SQL_MAX_SIZE)
			p.Param = paramsToString(param...)
			sqlCtx.step = p

			// if conf.Debug {
			// 	log.Println("[WA-SQL-03001] StartWithParam: ", traceCtx.Txid, ", ", traceCtx.Name, "\n", dbhost, "\n", sql, "\n", param)
			// }
		}
		return sqlCtx, nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-03002] StartWithParam: Not found Txid ", ", \n dbhost: ", dbhost, ", \n sql: ", sql, ", \n args: ", param)
	}
	return nil, fmt.Errorf("Not found Txid ")
}

func StartWithParamArray(ctx context.Context, dbhost, sql string, param []interface{}) (*SqlCtx, error) {
	return StartWithParam(ctx, dbhost, sql, param...)
}

func End(sqlCtx *SqlCtx, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	// driver.ErrSkip is not collected.
	if err == driver.ErrSkip {
		if conf.Debug {
			log.Println("[WA-SQL-04001] End: Error Skip ", err)
		}
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if sqlCtx != nil && sqlCtx.step != nil {
		up := sqlCtx.step
		switch up.GetPackType() {
		case udp.TX_DB_CONN:
			sqlCtx.ctx.ActiveDBC = false
			p := up.(*udp.UdpTxDbcPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
				p.ErrorType = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
			}
			if conf.Debug {
				log.Println("[WA-SQL-04002] txid: ", p.Txid, ", uri: ", sqlCtx.ctx.Name, "\n dbhost: ", p.Dbc, "\n time: ", p.Elapsed, "ms ", "\n error: ", err)
			}
			udpClient.Send(p)

		case udp.TX_SQL:
			sqlCtx.ctx.ActiveSQL = false
			p := up.(*udp.UdpTxSqlPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
				p.ErrorType = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
			}
			if conf.Debug {
				log.Println("[WA-SQL-04003] txid: ", p.Txid, ", uri: ", sqlCtx.ctx.Name, "\n dbhost: ", p.Dbc, "\n sql: ", p.Sql, "\n time: ", p.Elapsed, "ms ", "\n error: ", err)
			}
			udpClient.Send(p)

		case udp.TX_SQL_PARAM:
			sqlCtx.ctx.ActiveSQL = false
			p := up.(*udp.UdpTxSqlParamPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
				p.ErrorType = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
			}
			if conf.Debug {
				log.Println("[WA-SQL-04004] txid: ", p.Txid, ", uri: ", sqlCtx.ctx.Name, "\n dbhost: ", p.Dbc, "\n sql: ", p.Sql, "\n args: ", p.Param, "\n time: ", p.Elapsed, "ms ", "\n error: ", err)
			}
			udpClient.Send(p)

		}
		return nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-04005] End SqlCtx is nil: ", "\n error: ", err)
	}
	return fmt.Errorf("SqlCtx is nil")
}

func Trace(ctx context.Context, dbhost, sql string, param []interface{}, elapsed int, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		if param != nil && len(param) > 0 {
			if pack := udp.CreatePack(udp.TX_SQL_PARAM, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxSqlParamPack)
				p.Txid = traceCtx.Txid
				p.Time = dateutil.SystemNow()
				p.Elapsed = int32(elapsed)
				p.Dbc = stringutil.Truncate(hidePwd(dbhost), PACKET_DB_MAX_SIZE)
				p.Sql = stringutil.Truncate(sql, PACKET_SQL_MAX_SIZE)
				if err != nil {
					p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
					p.ErrorType = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
				}
				p.Param = paramsToString(param...)

				if conf.Debug {
					log.Println("[WA-SQL-05001] txid: ", p.Txid, ", uri: ", traceCtx.Name, "\n dbhost: ", p.Dbc, "\n sql: ", p.Sql, "\n args: ", p.Param, "\n time: ", p.Elapsed, "ms ", "\n error: ", err)
				}
				udpClient.Send(p)
			}
		}
		if pack := udp.CreatePack(udp.TX_SQL, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlPack)
			p.Txid = traceCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Elapsed = int32(elapsed)
			p.Dbc = stringutil.Truncate(hidePwd(dbhost), PACKET_DB_MAX_SIZE)
			p.Sql = stringutil.Truncate(sql, PACKET_SQL_MAX_SIZE)
			if err != nil {
				p.ErrorMessage = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
				p.ErrorType = stringutil.Truncate(err.Error(), STEP_ERROR_MESSAGE_MAX_SIZE)
			}
			if conf.Debug {
				log.Println("[WA-SQL-05002] txid: ", p.Txid, ", uri: ", traceCtx.Name, "\n dbhost: ", p.Dbc, "\n sql: ", p.Sql, "\n time: ", p.Elapsed, "ms ", "\n error: ", err)
			}
			udpClient.Send(p)
		}
		return nil
	}
	if conf.Debug {
		log.Println("[WA-SQL-05003] Trace: Not found Txid, ", "\n dbhost: ", dbhost, "\n sql: ", sql, "\n error: ", err)
	}
	return fmt.Errorf("Not found Txid ")
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
					buffer.WriteString(fmt.Sprintf("%v,", stringutil.Truncate(t, SQL_PARAM_VALUE_MAX_SIZE)))
				default:
					buffer.WriteString(fmt.Sprintf("%v,", val))
				}

			} else {
				switch t := val.(type) {
				case string:
					buffer.WriteString(fmt.Sprintf("%v", stringutil.Truncate(t, SQL_PARAM_VALUE_MAX_SIZE)))
				default:
					buffer.WriteString(fmt.Sprintf("%v", val))
				}
			}
		}
	}
	return string(buffer.Bytes())
}

func hidePwd(connStr string) string {
	first := strings.Index(connStr, ".")
	last := strings.Index(connStr, "@")
	if first > -1 && last > -1 && first < last {
		return fmt.Sprintf("%s:#@%s", connStr[0:first], connStr[last+1:])
	}
	return connStr
}
