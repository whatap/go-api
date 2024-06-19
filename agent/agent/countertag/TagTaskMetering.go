package countertag

import (
	//"log"
	"fmt"
	"os"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/go-api/agent/util/sys"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

type TagTaskMetering struct {
	nextTime int64
}

func NewTagTaskMetering() *TagTaskMetering {
	p := new(TagTaskMetering)
	return p
}

/*
*
미터링시 필요한 TagCounterPack 데이터

category: “metering”

pcode (long): 프로젝트 코드
oid (int): agent 고유 id 값
okine(int): 에이전트(어플리케이션) 종류 값
onode (int): 에이전트(어플리케이션) 노드 값
time (long): 수집된 시간 (ms)

# Tag
otype (int): agent 타입 (ap, sm, db ...)
subtype (int): 에이전트 타입의 추가 정보 (AP: JAVA, NODE, PYTHON, PHP…, SM: LINUX, Windows, OSX …)
ip (int): IP address 의 Hash값
host_uuid (string): 호스트의 uuid 값
csp (string): cloud 공급자 정보 (optional)

# Field
mcore (float): 미터링 되는 코어수 또는 수량
*
*/
func (this *TagTaskMetering) process(p *pack.TagCountPack) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA140001", " Task Telegraf Process Recover ", r)
		}
	}()

	conf := config.GetConfig()
	secu := secure.GetSecurityMaster()
	if this.nextTime == 0 || p.Time >= this.nextTime {
		p.Category = "metering"

		p.PutTag("otype", fmt.Sprintf("%d", 1))
		p.PutTag("subtype", fmt.Sprintf("%d", conf.AppType))
		p.PutTag("ip", fmt.Sprintf("%d", secu.IP))
		p.PutTag("host_uuid", secu.MeterIP)
		csp := os.Getenv("CLOUD_PLATFORM")
		p.PutTag("csp", csp)

		p.Put("mcore", sys.GetCPUNum())

		data.Send(p)

		now := dateutil.Now() / dateutil.MILLIS_PER_FIVE_MINUTE * dateutil.MILLIS_PER_FIVE_MINUTE
		this.nextTime = now + dateutil.MILLIS_PER_FIVE_MINUTE
	}
}
