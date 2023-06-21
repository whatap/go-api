package stat

import (
	//	"log"
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/agent/data"
)

//protected final int TABLE_MAX_SIZE = 10000;

type StatRemoteIp struct {
	table *pack.StatRemoteIpPack
	timer *TimingSender
}

var remoteIpLock = sync.Mutex{}
var statRemoteIp *StatRemoteIp

// Singleton
func GetInstanceStatRemoteIp() *StatRemoteIp {
	remoteIpLock.Lock()
	defer remoteIpLock.Unlock()
	if statRemoteIp != nil {
		return statRemoteIp
	}
	statRemoteIp = new(StatRemoteIp)
	statRemoteIp.table = pack.NewStatRemoteIpPack()
	statRemoteIp.timer = GetInstanceTimingSender()

	return statRemoteIp
}

func (this *StatRemoteIp) IncRemoteIp(ip int32) {
	if ip != 0 {
		this.table.IpTable.AddNoOver(ip, 1)
	}
	//fmt.Println("====================================")
	//fmt.Println("IpTable = ", this.table.IpTable.ToString())
}

func (this *StatRemoteIp) Send(now int64) {
	if this.table.IpTable.Size() == 0 {
		return
	}

	p := this.table
	this.table = pack.NewStatRemoteIpPack()

	p.Time = now

	data.Send(p)

}

func (this *StatRemoteIp) Clear() {
	this.table.IpTable.Clear()
}
