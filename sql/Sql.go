//github.com/whatap/go-api/sql
package sql

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
)

func Start(ctx context.Context, dbhost, sql string) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = wCtx
		if pack := udp.CreatePack(udp.TX_SQL, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = hidePwd(dbhost)
			p.Sql = sql
			sqlCtx.step = p
		}
		return sqlCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}

func StartOpen(ctx context.Context, dbhost string) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = wCtx
		if pack := udp.CreatePack(udp.TX_DB_CONN, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxDbcPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = hidePwd(dbhost)
			sqlCtx.step = p
		}
		return sqlCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}

func StartWithParam(ctx context.Context, dbhost, sql string, param ...interface{}) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = wCtx
		if pack := udp.CreatePack(udp.TX_SQL_PARAM, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlParamPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Dbc = hidePwd(dbhost)
			p.Sql = sql
			p.Param = paramsToString(param...)
			sqlCtx.step = p
		}
		return sqlCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}

func StartWithParamArray(ctx context.Context, dbhost, sql string, param []interface{}) (*SqlCtx, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return NewSqlCtx(), nil
	}
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		sqlCtx := NewSqlCtx()
		sqlCtx.ctx = wCtx
		if pack := udp.CreatePack(udp.TX_SQL_PARAM, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlParamPack)
			p.Time = dateutil.SystemNow()
			p.Dbc = hidePwd(dbhost)
			p.Sql = sql
			p.Param = paramsToString(param...)
			sqlCtx.step = p
		}
		return sqlCtx, nil
	}

	return nil, fmt.Errorf("Not found Txid ")
}

func End(sqlCtx *SqlCtx, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if sqlCtx != nil && sqlCtx.step != nil {
		up := sqlCtx.step
		switch up.GetPackType() {
		case udp.TX_DB_CONN:
			p := up.(*udp.UdpTxDbcPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = err.Error()
				p.ErrorType = err.Error()
			}
			udpClient.Send(p)

		case udp.TX_SQL:
			p := up.(*udp.UdpTxSqlPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = err.Error()
				p.ErrorType = err.Error()
			}
			udpClient.Send(p)

		case udp.TX_SQL_PARAM:
			p := up.(*udp.UdpTxSqlParamPack)
			p.Elapsed = int32(dateutil.SystemNow() - p.Time)
			if err != nil {
				p.ErrorMessage = err.Error()
				p.ErrorType = err.Error()
			}
			udpClient.Send(p)

		}

		return nil
	}

	return fmt.Errorf("SqlCtx is nil")
}

func Trace(ctx context.Context, dbhost, sql, param string, elapsed int, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*trace.TraceCtx)
		if pack := udp.CreatePack(udp.TX_SQL, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxSqlPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Elapsed = int32(elapsed)
			p.Dbc = hidePwd(dbhost)
			p.Sql = sql
			//TO-DO
			//p.Param = param
			udpClient.Send(p)
		}
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func paramsToString(params ...interface{}) string {
	var buffer bytes.Buffer
	for i, v := range params {
		if i < len(params)-1 {
			buffer.WriteString(fmt.Sprintf("%v,", v))
		} else {
			buffer.WriteString(fmt.Sprintf("%v", v))
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
