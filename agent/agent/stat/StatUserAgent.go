package stat

import (
	//	"log"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/agent/data"
	"sync"
	//	"github.com/whatap/golib/util/hmap"
)

type StatUserAgent struct {
	table *pack.StatUserAgentPack
	timer *TimingSender
}

var userAgentLock = sync.Mutex{}
var statUserAgent *StatUserAgent

// Singleton
func GetInstanceStatUserAgent() *StatUserAgent {
	userAgentLock.Lock()
	defer userAgentLock.Unlock()
	if statUserAgent != nil {
		return statUserAgent
	}
	statUserAgent = new(StatUserAgent)
	statUserAgent.table = pack.NewStatUserAgentPack()
	statUserAgent.timer = GetInstanceTimingSender()

	return statUserAgent
}

func (this *StatUserAgent) IncUserAgent(userAgent int32) {
	if userAgent != 0 {
		this.table.UserAgents.AddNoOver(userAgent, 1)
	}
}

func (this *StatUserAgent) Send(now int64) {
	if this.table.UserAgents.Size() == 0 {
		return
	}

	p := this.table
	this.table = pack.NewStatUserAgentPack()

	p.Time = now

	data.Send(p)

}
func (this *StatUserAgent) Clear() {
	this.table.UserAgents.Clear()
}
