package stat

import (
	//"log"
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	langconf "github.com/whatap/go-api/agent/lang/conf"
)

const (
	STAT_TRANX_TABLE_MAX_SIZE = 5000
)

type StatTranx struct {
	table *hmap.IntKeyLinkedMap
	timer *TimingSender
}

var tranxLock = sync.Mutex{}
var tranx *StatTranx

// Singleton  func GetInstanceStatTranx() *StatTranx {
func GetInstanceStatTranx() *StatTranx {
	tranxLock.Lock()
	defer tranxLock.Unlock()
	if tranx != nil {
		return tranx
	}
	tranx = new(StatTranx)

	tranx.table = hmap.NewIntKeyLinkedMap(STAT_TRANX_TABLE_MAX_SIZE+1, 1).SetMax(int(config.GetConfig().StatTxMaxCount))
	tranx.timer = GetInstanceTimingSender()

	// conf 변경시 max 재설정
	langconf.AddConfObserver("StatTranx", tranx)

	return tranx
}

// Implements lang/conf/ConfObserver/Runnable
func (this *StatTranx) Run() {
	conf := config.GetConfig()
	this.table.SetMax(int(conf.StatSqlMaxCount))
}

func (this *StatTranx) GetService(hash int32) *pack.TransactionRec {
	// map.Intern 함수 사용 안함
	//return this.table.Intern(hash)
	//

	// 이미 있으면 있는 걸 반환
	if this.table.ContainsKey(hash) {
		return this.table.Get(hash).(*pack.TransactionRec)
		// size = 0 이면 TimeCount 로 객체 하나 생성해서 반환
	} else {
		if this.table.IsFull() {
			return nil
		}

		p := pack.NewTransactionRec()
		p.Hash = hash
		this.table.Put(hash, p)

		return p
	}
	//retur nil
}

func (this *StatTranx) Send(now int64) {
	if this.table.Size() == 0 {
		return
	}

	//p := pack.NewStatTransactionPack().SetRecords(this.table.Size(), this.table.Values())
	p := pack.NewStatTransactionPack1().SetRecords(this.table.Size(), this.table.Values())
	p.Spec = int(config.GetConfig().MtraceSpecHash)
	p.Time = now
	this.table.Clear()

	data.Send(p)
}

func (this *StatTranx) Clear() {
	this.table.Clear()
}
