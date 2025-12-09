package stat

import (
	"math"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/bitutil"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/list"
)

const (
	STAT_TRANX_STATUS_TABLE_MAX_SIZE = 700
)

type StatTranxStatus struct {
	table *hmap.LongKeyLinkedMap
	timer *TimingSender
}

var tranxStatusLock = sync.Mutex{}
var statTranxStatus *StatTranxStatus

// Singleton
func GetInstanceStatTranxStatus() *StatTranxStatus {
	tranxStatusLock.Lock()
	defer tranxStatusLock.Unlock()

	if statTranxStatus != nil {
		return statTranxStatus
	}
	statTranxStatus = new(StatTranxStatus)
	statTranxStatus.table = hmap.NewLongKeyLinkedMap(STAT_TRANX_STATUS_TABLE_MAX_SIZE+1, 1).SetMax(STAT_TRANX_STATUS_TABLE_MAX_SIZE)
	statTranxStatus.timer = GetInstanceTimingSender()

	return statTranxStatus
}
func (this *StatTranxStatus) Add(status, urlhash, elapsed int32, err int64) {
	v := this.GetService(status, urlhash)
	if v != nil {
		v.Count++
		v.Time += int64(elapsed)
		if err > 0 {
			v.Error++
		}
	}
}

func (this *StatTranxStatus) GetService(status int32, urlhash int32) *pack.TimeCount {
	conf := config.GetConfig()
	key := bitutil.Composite64(status, urlhash)

	var rt interface{}
	if this.table.Size() < int(conf.StatTxStatusMaxCount) {
		rt = this.table.Get(key)
		if rt == nil {
			v := pack.NewTimeCountDefault()
			this.table.Put(key, v)
			return v
		} else {
			return rt.(*pack.TimeCount)
		}
	} else {
		rt = this.table.Get(key)
		if rt != nil {
			return rt.(*pack.TimeCount)
		} else {
			return nil
		}
	}
}

func (this *StatTranxStatus) Send(now int64) {
	if this.table.Size() == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA545", " Recover", r) //, string(debug.Stack()))
		}
	}()

	conf := config.GetConfig()

	currTable := this.table
	sz := int(math.Max(STAT_TRANX_STATUS_TABLE_MAX_SIZE, float64(conf.StatTxStatusMaxCount)))
	this.table = hmap.NewLongKeyLinkedMap(int(sz)+1, 1)

	url := list.NewIntList(currTable.Size())
	status := list.NewIntList(currTable.Size())
	count := list.NewIntList(currTable.Size())
	err := list.NewIntList(currTable.Size())
	time := list.NewLongList(currTable.Size())

	en := currTable.Entries()
	for en.HasMoreElements() {
		ent := en.NextElement().(*hmap.LongKeyLinkedEntry)
		k := ent.GetKey()
		v := ent.GetValue().(*pack.TimeCount)

		url.AddInt(int(bitutil.GetLow64(k)))
		status.AddInt(int(bitutil.GetHigh64(k)))
		count.AddInt(int(v.Count))
		err.AddInt(int(v.Error))
		time.AddLong(v.Time)
	}

	this.table.Clear()
	currTable.Clear()

	var out *pack.StatGeneralPack

	if conf.Stat1MEnabled {
		out = pack.NewStatGeneralPackType(pack.PACK_STAT_GENERAL_1)
		out.DataStartTime = now - dateutil.MILLIS_PER_MINUTE
	} else {
		out = pack.NewStatGeneralPack()
	}

	out.Put("url", url)
	out.Put("status", status)
	out.Put("count", count)
	out.Put("error", err)
	out.Put("time", time)

	out.Id = "url-status"
	out.Time = now

	if conf.StatZipEnabled {
		data.GetInstanceZipPackThread().Add(out)
	} else {
		data.Send(out)
	}

}

func (this *StatTranxStatus) Clear() {
	this.table.Clear()
}
func (this *StatTranxStatus) Size() int {
	return this.table.Size()
}
