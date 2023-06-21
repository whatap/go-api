package trace

import (
	"time"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"
)

var (
	connpools [][]int32
	timer     *time.Timer
)

func AddDBConnPool(pid int32, url string, actCnt int32, inactCnt int32) {
	SendConnPoolTagCount(pid, url, actCnt, inactCnt)

	urlHash := hash.HashStr(url)

	data.SendHashText(pack.TEXT_DB_URL, urlHash, url)

	found := false
	for i, _ := range connpools {
		if urlHash == connpools[i][0] {
			connpools[i][1] += actCnt
			connpools[i][2] += inactCnt
			found = true
		}
	}
	if !found {
		connpools = append(connpools, []int32{urlHash, actCnt, inactCnt})
	}

	if timer == nil {
		timer := time.NewTimer(1 * time.Second)
		go func() {
			<-timer.C
			if connpools != nil {
				for _, connpool := range connpools {
					meter.GetInstanceConnPool().AddDBConnPool(connpool[0], connpool[1], connpool[2])
				}
			}
			connpools = nil
			timer = nil
		}()
	}
}

func SendConnPoolTagCount(pid int32, url string, actCnt int32, inactCnt int32) {
	p := pack.NewTagCountPack()
	p.Category = "db_pool_detail"
	p.Time = dateutil.Now()
	p.Tags.PutString("pool", url)
	p.Tags.PutLong("pid", int64(pid))
	p.Data.PutLong("act", int64(actCnt))
	p.Data.PutLong("idle", int64(inactCnt))

	data.Send(p)
}
