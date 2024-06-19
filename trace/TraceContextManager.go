package trace

import (
	"sync"

	agentconfig "github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/util/hmap"
)

const (
	STAT_METHOD = 0
	STAT_SQL    = 1
	STAT_HTTPC  = 2
	STAT_DBC    = 3
	STAT_SOCKET = 4
)

var tcm *TraceContextManager
var tcmLock sync.Mutex

type TraceContextManager struct {
	ctxTable *hmap.LongKeyLinkedMap
}

func GetTraceContextManager() *TraceContextManager {
	if tcm != nil {
		return tcm
	}
	tcmLock.Lock()
	defer tcmLock.Unlock()
	if tcm != nil {
		return tcm
	}
	tcm = newTraceContextManager()
	return tcm
}

func newTraceContextManager() *TraceContextManager {
	p := new(TraceContextManager)
	conf := agentconfig.GetConfig()
	p.ctxTable = hmap.NewLongKeyLinkedMapDefault().SetMax(int(conf.TxMaxCount))
	return p
}

func (tcm *TraceContextManager) Clear() {
	tcm.ctxTable.Clear()
}

// var conf *agentconfig.Config = agentconfig.GetConfig()

// var ctxTable *hmap.LongKeyLinkedMap =
// var ctxTable *hmap.LongKeyLinkedMap = hmap.NewLongKeyLinkedMapDefault().SetMax(int(5000))

func AddGIDTraceCtx(GID int64, traceCtx *TraceCtx) {
	conf := agentconfig.GetConfig()
	if !conf.GoUseGoroutineIDEnabled {
		return
	}
	ctxm := GetTraceContextManager()
	ctxm.ctxTable.Put(GID, traceCtx)
	// ctxTable.Put(GID, traceCtx)
}
func GetGIDTraceCtx(GID int64) *TraceCtx {
	conf := agentconfig.GetConfig()
	if !conf.GoUseGoroutineIDEnabled {
		return nil
	}
	ctxm := GetTraceContextManager()
	// if obj := ctxTable.Get(GID); obj != nil {
	if obj := ctxm.ctxTable.Get(GID); obj != nil {
		if v, ok := obj.(*TraceCtx); ok {
			return v
		}
	}
	return nil
}

func RemoveGIDTraceCtx(GID int64) {
	conf := agentconfig.GetConfig()
	if !conf.GoUseGoroutineIDEnabled {
		return
	}
	ctxm := GetTraceContextManager()
	ctxm.ctxTable.Remove(GID)
	// ctxTable.Remove(GID)
}
