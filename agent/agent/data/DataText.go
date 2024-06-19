package data

import (
	//"log"
	"sync"
	// "time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/net"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/queue"
)

const (
	BUFFERED_MAX = 100 * 1024
)

type DataText struct {
	textQueue      *queue.RequestQueue
	bufferPack     *pack.TextPack
	bufferedLength int
	lastDate       int64
	lastSent       int64
	textReset      int32
	conf           *config.Config
}

var dataText *DataText
var lock = sync.Mutex{}

func initial() *DataText {
	lock.Lock()
	defer lock.Unlock()

	if dataText != nil {
		return dataText
	}
	dataText = new(DataText)
	dataText.conf = config.GetConfig()

	dataText.textQueue = queue.NewRequestQueue(int(dataText.conf.QueueTextSize))
	if dataText.conf.QueueLogEnabled {
		logutil.Println("WA10701-01", "textQueue=", dataText.textQueue.GetCapacity())
	}

	// 00시 기준으로 Hash Reset을 위한 설정 추가
	dataText.lastDate = dataText.getDate()
	dataText.textReset = dataText.conf.TextReset

	dataText.bufferPack = pack.NewTextPack()
	dataText.bufferedLength = 0

	// 기본 1개 실행.
	go func() {
		for {
			// shutdown
			if config.GetConfig().Shutdown {
				logutil.Infoln("WA211-05", "Shutdown DataText")
				dataText.reset()
				break
			}

			dataText.process()
		}
	}()
	return dataText
}

var textCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)

// Div 로 해시 중복 확인 분기 , localhost httpc 연결시 ServiceHash와 HttpcUrlHash 가 충돌
var dbcCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var methodCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(5000)

var httpcUrlCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var httpcHostCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var errorCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var serviceCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var sqlCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var stackCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(20000)
var messageCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var useragentCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(2000)
var refererCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var loginCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var sqlParamCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var httpDomainCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)

var mtraceSpecCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)
var mtraceCallerUrlCache *hmap.IntLinkedSet = hmap.NewIntLinkedSet().SetMax(1000)

type TextKey struct {
	div  byte
	hash int32
}

func (this *TextKey) Hash() uint {
	return uint(this.hash ^ int32(this.div<<32))
}

func (this *TextKey) Equals(o hmap.LinkedKey) bool {
	other := o.(*TextKey)
	return this.div == other.div && this.hash == other.hash
}

func SendText(div byte, text string) {
	h := hash.HashStr(text)
	SendHashText(div, h, text)
}
func SendHashText(div byte, h int32, text string) {
	// Div 로 해시 중복 확인 분기 , localhost httpc 연결시 ServiceHash와 HttpcUrlHash 가 충돌
	// Error를 Message 로 출력할 때도 Hash 중복으로 각 Div 전달 안됨.
	// 20170927 Java 소스를 기준으로 각 Text Type 별 해시테이블  추가.
	// 20201130 해시 추가 실패(non block channel put)하면 캐시 삭제, 다시 전송할 수 있도록
	switch div {
	case pack.TEXT_DB_URL:
		if dbcCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			dbcCache.Remove(h)
		}
	case pack.TEXT_METHOD:
		if methodCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			methodCache.Remove(h)
		}
	case pack.TEXT_HTTPC_URL:
		if httpcUrlCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			httpcUrlCache.Remove(h)
		}

	case pack.TEXT_HTTPC_HOST:
		if httpcHostCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			httpcHostCache.Remove(h)
		}

	case pack.TEXT_ERROR:
		if errorCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			errorCache.Remove(h)
		}

	case pack.TEXT_SERVICE:
		if serviceCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			serviceCache.Remove(h)
		}

	case pack.TEXT_SQL:
		if sqlCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			sqlCache.Remove(h)
		}

	case pack.TEXT_STACK_ELEMENTS:
		if stackCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			stackCache.Remove(h)
		}

	case pack.TEXT_MESSAGE:
		if messageCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			messageCache.Remove(h)
		}

	case pack.TEXT_USER_AGENT:
		if useragentCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			useragentCache.Remove(h)
		}

	case pack.TEXT_REFERER:
		if refererCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			refererCache.Remove(h)
		}

	case pack.TEXT_LOGIN:
		if loginCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			loginCache.Remove(h)
		}

	case pack.TEXT_SQL_PARAM:
		if sqlParamCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			sqlParamCache.Remove(h)
		}

	case pack.TEXT_HTTP_DOMAIN:
		if httpDomainCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			httpDomainCache.Remove(h)
		}

	case pack.TEXT_MTRACE_SPEC:
		if mtraceSpecCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			mtraceSpecCache.Remove(h)
		}

	case pack.TEXT_MTRACE_CALLER_URL:
		if mtraceCallerUrlCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			mtraceCallerUrlCache.Remove(h)
		}

	default:
		if textCache.Put(h) != nil {
			return
		}
		if rt := AddHashText(div, h, text); !rt {
			textCache.Remove(h)
		}
	}

	//AddHashText(div, h, text)
}

func AddHashText(div byte, h int32, text string) bool {
	var this *DataText
	if dataText != nil {
		this = dataText
	} else {
		this = initial()
	}

	return this.textQueue.Put(pack.TextRec{Div: div, Hash: h, Text: text})

}

func (this *DataText) process() {
	lock.Lock()
	defer func() {
		lock.Unlock()
		if r := recover(); r != nil {
			logutil.Println("WA10701", " Recover:", r)
		}
	}()

	tmp := this.textQueue.GetTimeout(1000)
	if tmp != nil {
		r := tmp.(pack.TextRec)
		this.bufferPack.AddText(r)
		this.bufferedLength += len(r.Text)
		if this.bufferedLength >= BUFFERED_MAX {
			this.send()
		}
	}

	now := dateutil.Now()
	if net.GetTcpSession().LastConnectedTime > 0 && this.bufferedLength > 0 {
		if now-this.lastSent >= 5000 {
			this.send()
		}
	} else {
		// 현재 데이터가 없다면 last_sent를 초기화.
		this.lastSent = now
	}

	// UTC 00시 기준으로 해시 초기화
	today := this.getDate()
	if this.lastDate != today || this.textReset != this.conf.TextReset {
		this.lastDate = today
		this.textReset = this.conf.TextReset

		if this.bufferedLength > 0 {
			this.send()
		}
		this.reset()
	}
}

func (this *DataText) send() {
	this.lastSent = dateutil.Now()
	Sent(this.bufferPack)

	this.bufferPack = pack.NewTextPack()
	this.bufferedLength = 0
}
func (this *DataText) reset() {
	logutil.Println("WA10702", " Text Map Reset")

	textCache.Clear()

	dbcCache.Clear()
	methodCache.Clear()

	httpcUrlCache.Clear()
	httpcHostCache.Clear()
	errorCache.Clear()
	serviceCache.Clear()
	sqlCache.Clear()

	stackCache.Clear()
	messageCache.Clear()
	useragentCache.Clear()

	refererCache.Clear()
	loginCache.Clear()
	sqlParamCache.Clear()
	httpDomainCache.Clear()

	mtraceSpecCache.Clear()
	mtraceCallerUrlCache.Clear()
}

func ResetHash() {
	var this *DataText
	if dataText != nil {
		this = dataText
	} else {
		this = initial()
	}

	this.reset()
}
func (this *DataText) getDate() int64 {
	return dateutil.Now() / dateutil.MILLIS_PER_HOUR
}
