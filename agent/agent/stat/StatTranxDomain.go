package stat

import (
	//"log"
	//"math"
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/variable"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/list"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
)

const (
	STAT_TRANX_DOMAIN_TABLE_MAX_SIZE = 700
)

type StatTranxDomain struct {
	table *hmap.LinkedMap
	timer *TimingSender
}

var tranxDomainLock = sync.Mutex{}
var tranxDomain *StatTranxDomain

// Singleton  func GetInstanceStatTranx() *StatTranx {
func GetInstanceStatTranxDomain() *StatTranxDomain {
	tranxDomainLock.Lock()
	defer tranxDomainLock.Unlock()
	if tranxDomain != nil {
		return tranxDomain
	}
	tranxDomain = new(StatTranxDomain)

	//tranxDomain.table = hmap.NewLinkedMap().SetMax(STAT_TRANX_DOMAIN_TABLE_MAX_SIZE)
	tranxDomain.table = hmap.NewLinkedMapDefault().SetMax(int(config.GetConfig().StatDomainMaxCount))
	tranxDomain.timer = GetInstanceTimingSender()

	return tranxDomain
}

// func (this * StatTranx) GetService(hash int32) *pack.ServiceRec {
func (this *StatTranxDomain) GetService(domain int32, urlhash int32) *pack.TimeCount {

	k := variable.NewI2(domain, urlhash)
	var rt interface{}
	rt = this.table.Get(k)

	if rt == nil {
		v := pack.NewTimeCountDefault()
		this.table.Put(k, v)
		return v

	} else {
		return rt.(*pack.TimeCount)
	}
}

// func (this * StatTranx) Send(now int64) {
func (this *StatTranxDomain) Send(now int64) {
	if this.table.Size() == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10200", " Recover", r) //, string(debug.Stack()))
		}
	}()

	domain := list.NewIntList(this.table.Size())
	url := list.NewIntList(this.table.Size())
	count := list.NewIntList(this.table.Size())
	err := list.NewIntList(this.table.Size())
	time := list.NewLongList(this.table.Size())

	en := this.table.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		k := ent.GetKey().(*variable.I2)
		v := ent.GetValue().(*pack.TimeCount)

		domain.AddInt(int(k.V1))
		url.AddInt(int(k.V2))
		count.AddInt(int(v.Count))
		err.AddInt(int(v.Error))
		time.AddLong(v.Time)
	}

	this.table.Clear()

	out := pack.NewStatGeneralPack()
	out.Put("domain", domain)
	out.Put("url", url)
	out.Put("count", count)
	out.Put("error", err)
	out.Put("time", time)

	out.Id = "dom"
	out.Time = now

	data.Send(out)
}

// func (this * StatTranx) Clear() {
func (this *StatTranxDomain) Clear() {
	this.table.Clear()
}
