package trace

import (
	//	"log"
	"sync"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/hmap"
	"github.com/whatap/go-api/config"
)

const (
	STAT_METHOD = 0
	STAT_SQL    = 1
	STAT_HTTPC  = 2
	STAT_DBC    = 3
	STAT_SOCKET = 4
)

var conf *config.Config = config.GetConfig()

var ctxTable *hmap.LongKeyLinkedMap = hmap.NewLongKeyLinkedMapDefault().SetMax(int(conf.TxMaxCount))
var ctxLock sync.Mutex

func AddTraceCtx(traceCtx *TraceCtx) {
	ctxTable.Put(traceCtx.Txid, traceCtx)
}
func RemoveTraceCtx(traceCtx *TraceCtx) {
	ctxTable.Remove(traceCtx.Txid)
}
func GetActiveStats() []int16 {
	ctxLock.Lock()
	defer ctxLock.Unlock()

	aStats := make([]int16, 5, 5)
	en := ctxTable.Values()
	var tmp interface{}
	for en.HasMoreElements() {
		tmp = en.NextElement()
		if tmp == nil {
			continue
		}
		ctx := tmp.(*TraceCtx)
		if ctx.ActiveSQL {
			aStats[STAT_SQL]++
		} else if ctx.ActiveHTTPC {
			aStats[STAT_HTTPC]++
		} else if ctx.ActiveDBC {
			aStats[STAT_DBC]++
		} else if ctx.ActiveSocket {
			aStats[STAT_SOCKET]++
		} else {
			aStats[STAT_METHOD]++
		}
	}
	return aStats
}

type TaskActiveStats struct {
}

func (this *TaskActiveStats) Process(now int64) {
	activeStats := GetActiveStats()
	if up := udp.CreatePack(udp.ACTIVE_STATS, udp.UDP_PACK_VERSION); up != nil {
		p := up.(*udp.UdpActiveStatsPack)
		p.Time = now
		p.ActiveStats = activeStats

		udpClient := whatapnet.GetUdpClient()
		udpClient.Send(p)
	}
}
