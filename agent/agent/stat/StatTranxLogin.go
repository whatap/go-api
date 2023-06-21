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
	STAT_TRANX_LOGIN_TABLE_MAX_SIZE = 700
)

type StatTranxLogin struct {
	table *hmap.LinkedMap
	timer *TimingSender
}

var tranxLoginLock = sync.Mutex{}
var tranxLogin *StatTranxLogin

// Singleton  func GetInstanceStatTranx() *StatTranx {
func GetInstanceStatTranxLogin() *StatTranxLogin {
	tranxLoginLock.Lock()
	defer tranxLoginLock.Unlock()
	if tranxLogin != nil {
		return tranxLogin
	}
	tranxLogin = new(StatTranxLogin)

	//tranxLogin.table = hmap.NewLinkedMap().SetMax(STAT_TRANX_REFERER_TABLE_MAX_SIZE)
	tranxLogin.table = hmap.NewLinkedMapDefault().SetMax(int(config.GetConfig().StatLoginMaxCount))
	tranxLogin.timer = GetInstanceTimingSender()

	return tranxLogin
}

// func (this * StatTranx) GetService(hash int32) *pack.ServiceRec {
func (this *StatTranxLogin) GetService(login int32, urlhash int32) *pack.TimeCount {

	k := variable.NewI2(login, urlhash)
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
func (this *StatTranxLogin) Send(now int64) {
	if this.table.Size() == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10400", " Recover", r) //, string(debug.Stack()))
		}
	}()

	login := list.NewIntList(this.table.Size())
	url := list.NewIntList(this.table.Size())
	count := list.NewIntList(this.table.Size())
	err := list.NewIntList(this.table.Size())
	time := list.NewLongList(this.table.Size())

	en := this.table.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LinkedEntry)
		k := ent.GetKey().(*variable.I2)
		v := ent.GetValue().(*pack.TimeCount)

		login.AddInt(int(k.V1))
		url.AddInt(int(k.V2))
		count.AddInt(int(v.Count))
		err.AddInt(int(v.Error))
		time.AddLong(v.Time)
	}

	this.table.Clear()

	out := pack.NewStatGeneralPack()
	out.Put("login", login)
	out.Put("url", url)
	out.Put("count", count)
	out.Put("error", err)
	out.Put("time", time)

	out.Id = "login"
	out.Time = now

	data.Send(out)
}

func (this *StatTranxLogin) Clear() {
	this.table.Clear()
}
