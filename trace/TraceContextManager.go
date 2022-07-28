package trace

import (
	//	"log"
	"sync"

	"github.com/whatap/go-api/config"
	"github.com/whatap/golib/lang/pack/udp"
	whatapnet "github.com/whatap/golib/net"
	"github.com/whatap/golib/util/hmap"
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

//var goidTable *hmap.LongLongLinkedMap = hmap.NewLongLongLinkedMapDefault().SetMax(int(conf.TxMaxCount))

var ctxLock sync.Mutex

func AddTraceCtx(traceCtx *TraceCtx) {
	ctxTable.Put(traceCtx.Txid, traceCtx)
	// goidTable.Put(GetGID(), traceCtx.Txid)
}

// func RemoveTraceCtx(txid int64) *TraceCtx {
// 	if v := ctxTable.Remove(txid); v != nil {
// 		if tCtx, ok := v.(*TraceCtx); ok {
// 			return tCtx
// 		}
// 	}
// 	return nil
// }
func RemoveTraceCtx(traceCtx *TraceCtx) {
	ctxTable.Remove(traceCtx.Txid)
	//goidTable.Put(GetGID())
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
