//github.com/whatap/go-api/trace
package trace

import (
	"github.com/whatap/go-api/common/util/dateutil"
)

type TraceCtx struct {
	Txid      int64
	Host      string
	Name      string
	StartTime int64

	MTid        int64
	MDepth      int32
	MCallerTxid int64

	MCallee     int64
	MCallerSpec string
	MCallerUrl  string

	MCallerPoidKey string

	TraceMtraceCallerValue string
	TraceMtracePoidValue   string
	TraceMtraceSpecValue   string
}

func (this *TraceCtx) GetElapsedTime() int {
	return int(this.StartTime - dateutil.SystemNow())
}
