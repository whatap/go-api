package stat

import (
	//"log"
	//"math"
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/list"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
)

const (
	STAT_TRANX_MT_CALLER_TABLE_MAX_SIZE = 700
)

// implements LinkedKey
type MtraceKEY struct {
	CallerPcode int64
	CallerOkind int32
	CallerSpec  int32
	CallerUrl   int32
	Url         int32
	_hash_      int32
}

func NewMtraceKEY() *MtraceKEY {
	p := new(MtraceKEY)
	return p
}

// override LinkedKey Hash() uint
func (this *MtraceKEY) Hash() uint {
	if this._hash_ != 0 {
		return uint(this._hash_)
	}

	prime := int32(31)
	result := int32(1)

	result = prime*result + int32(this.CallerPcode^int64(uint64(this.CallerPcode)>>32))
	result = prime*result + this.CallerOkind
	result = prime*result + this.CallerSpec
	result = prime*result + this.CallerUrl
	result = prime*result + this.Url
	this._hash_ = result
	return uint(result)
}

// override LinkedKey Equals(h LinkedKey) bool
func (this *MtraceKEY) Equals(h hmap.LinkedKey) bool {
	if this == h {
		return true
	}
	if h == nil {
		return false
	}
	//	if (getClass() != obj.getClass())
	//		return false;
	other := h.(*MtraceKEY)
	if this.CallerPcode != other.CallerPcode {
		return false
	}
	if this.CallerOkind != other.CallerOkind {
		return false
	}
	if this.CallerSpec != other.CallerSpec {
		return false
	}
	if this.CallerUrl != other.CallerUrl {
		return false
	}
	if this.Url != other.Url {
		return false
	}
	return true
}

type StatTranxMtCaller struct {
	table *hmap.LinkedMap
	timer *TimingSender
}

var tranxMtCallerLock = sync.Mutex{}
var tranxMtCaller *StatTranxMtCaller

// Singleton  func GetInstanceStatTranx() *StatTranx {
func GetInstanceStatTranxMtCaller() *StatTranxMtCaller {
	tranxMtCallerLock.Lock()
	defer tranxMtCallerLock.Unlock()
	if tranxMtCaller != nil {
		return tranxMtCaller
	}
	tranxMtCaller = new(StatTranxMtCaller)

	//tranxDomain.table = hmap.NewLinkedMap().SetMax(STAT_TRANX_MT_CALLER_TABLE_MAX_SIZE)
	tranxMtCaller.table = hmap.NewLinkedMapDefault().SetMax(int(config.GetConfig().StatMtraceMaxCount))
	tranxMtCaller.timer = GetInstanceTimingSender()

	return tranxMtCaller
}

func (this *StatTranxMtCaller) GetService(key *MtraceKEY) *pack.TimeCount {

	rt := this.table.Get(key)
	if rt == nil {
		v := pack.NewTimeCountDefault()
		this.table.Put(key, v)
		return v

	} else {
		return rt.(*pack.TimeCount)
	}
}

// func (this * StatTranx) Send(now int64) {
func (this *StatTranxMtCaller) Send(now int64) {

	if this.table.Size() == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10500", " Recover", r) //, string(debug.Stack()))
		}
	}()

	conf := config.GetConfig()
	mtraceSpecHash := int32(0)
	if conf.MtraceSpec == "" {
		mtraceSpecHash = 0
	} else {
		mtraceSpecHash = int32(hash.HashStr(conf.MtraceSpec))
		data.SendHashText(pack.TEXT_MTRACE_SPEC, mtraceSpecHash, conf.MtraceSpec)
	}

	callerPcode := list.NewLongList(this.table.Size())
	callerOkind := list.NewLongList(this.table.Size())
	callerSpec := list.NewIntList(this.table.Size())
	callerUrl := list.NewIntList(this.table.Size())
	thisSpec := list.NewIntList(this.table.Size())
	url := list.NewIntList(this.table.Size())
	count := list.NewIntList(this.table.Size())
	err := list.NewIntList(this.table.Size())
	time := list.NewLongList(this.table.Size())

	en := this.table.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		k := ent.GetKey().(*MtraceKEY)
		v := ent.GetValue().(*pack.TimeCount)

		callerPcode.AddLong(k.CallerPcode)
		callerOkind.AddInt(int(k.CallerOkind))
		callerSpec.AddInt(int(k.CallerSpec))
		callerUrl.AddInt(int(k.CallerUrl))
		thisSpec.AddInt(int(mtraceSpecHash))
		url.AddInt(int(k.Url))
		count.AddInt(int(v.Count))
		err.AddInt(int(v.Error))
		time.AddLong(v.Time)

		//logutil.Infoln(" CallerPcode=", k.CallerPcode, ",CallerSpec=", k.CallerSpec )
	}

	this.table.Clear()

	out := pack.NewStatGeneralPack()
	out.Put("caller_pcode", callerPcode)
	out.Put("caller_okind", callerOkind)
	out.Put("caller_spec", callerSpec)
	out.Put("caller_url", callerUrl)
	out.Put("spec", thisSpec)
	out.Put("url", url)
	out.Put("count", count)
	out.Put("error", err)
	out.Put("time", time)

	out.Id = "mt"
	out.Time = now

	data.Send(out)
}

func (this *StatTranxMtCaller) Clear() {
	this.table.Clear()
}
