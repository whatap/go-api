//github.com/whatap/go-api/sql
package sql

import (
	"github.com/whatap/go-api/common/lang/pack/udp"
	"github.com/whatap/go-api/trace"
)

type SqlCtx struct {
	ctx  *trace.TraceCtx
	step udp.UdpPack
	// step     *udp.UdpTxSqlPack
	// stepConn *udp.UdpTxDbcPack
}

func NewSqlCtx() *SqlCtx {
	p := new(SqlCtx)
	return p
}
