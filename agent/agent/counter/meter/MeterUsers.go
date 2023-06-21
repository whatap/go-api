package meter

import (
	//"log"
	"sync"
	//	"time"

	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hll"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
)

const (
	METER_USERS_MAX_USERS = 7000
)

type MeterUsers struct {
	users *hmap.LongLongLinkedMap
	lock  sync.Mutex
}

func NewMeterUsers() *MeterUsers {
	p := new(MeterUsers)
	p.users = hmap.NewLongLongLinkedMapDefault().SetMax(METER_USERS_MAX_USERS)

	return p
}

var meterUsers *MeterUsers = NewMeterUsers()

func AddMeterUsers(wClientId int64) {
	if wClientId != 0 && config.GetConfig().RealtimeUserEnabled {
		meterUsers.users.PutLast(wClientId, dateutil.Now())
		//fmt.Println("AddMeterUser===================", wClientId)
	}
}

func AddActiveMeterUsers(wClientId int64) {
	if wClientId != 0 && config.GetConfig().RealtimeUserEnabled {
		meterUsers.users.Put(wClientId, dateutil.Now())
		//fmt.Println("AddActiveMeterUser=========================", wClientId)
	}
}

func GetRealtimeUsers() *hll.HyperLogLog {
	meterUsers.lock.Lock()
	defer meterUsers.lock.Unlock()

	max_think_time := config.GetConfig().RealtimeUserThinktimeMax
	now := dateutil.Now()

	loglog := hll.NewHyperLogLogDefault()

	if meterUsers.users.Size() == 0 {
		return loglog
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("WA411", " MeterUsers Recover", r)
			}
		}()

		en := meterUsers.users.Entries()
		//fmt.Println("MeerUsers GetRealtimeUsers ====================> size=", meterUsers.users.Size())
		for en.HasMoreElements() {
			e := en.NextElement().(*hmap.LongLongLinkedEntry)
			//fmt.Println("MeerUsers GetRealtimeUsers ====================> e=", e)
			if now-e.GetValue() > max_think_time {
				meterUsers.users.Remove(e.GetKey())
				//fmt.Println("MeerUsers GetRealtimeUsers Remove ====================> e=", e.GetKey())

			} else {
				//fmt.Println("MeerUsers GetRealtimeUsers ====================> e=", e)
				loglog.OfferLong(uint64(e.GetKey()))
				//fmt.Println("MeerUsers GetRealtimeUsers loglog.OfferLong====================> e=", uint64(e.GetKey()))
			}
		}
	}()

	return loglog
}

// DEBUG
type fakeHash32 uint32

func (f fakeHash32) Sum32() uint32 { return uint32(f) }

func MeterUsersMain() {

	//	for i := 1; i <= 100000; i++ {
	//		AddMeterUsers(keygen.Next())
	//	}
	//	//long cpu1 = SysJMX.getCurrentThreadCPUnano();
	//	cpu1 := 1234124124
	//
	//	for i := 1; i <= 100; i++ {
	//		AddMeterUsers(keygen.Next())
	//	}
	//	//long cpu2 = SysJMX.getCurrentThreadCPUnano();
	//	cpu2 := 12315123234234
	//	loglog := GetRealtimeUsers()
	//
	//	//long cpu3 = SysJMX.getCurrentThreadCPUnano();
	//	cpu3 := 23525234235423423
	//
	//	time.Sleep(1000)
	//
	//	logutil.Println((cpu2 - cpu1) / 1000000)
	//	logutil.Println((cpu3 - cpu2) / 1000000)
	//	logutil.Println("user : ", loglog.Cardinality())
}
