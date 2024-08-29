package data

import (
	"bytes"
	"sync"
	"time"

	wio "github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/ansi"
	"github.com/whatap/golib/util/compressutil"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/queue"
	"github.com/whatap/golib/util/stringutil"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	wnet "github.com/whatap/go-api/agent/net"
	"github.com/whatap/go-api/agent/util/logutil"
)

type ZipPackThread struct {
	Queue      *queue.RequestQueue
	conf       *config.Config
	secuMaster *secure.SecurityMaster
	lastLog    int64
	zipSent    int64
	noZipSent  int64

	buffer    bytes.Buffer
	packCount int
	firstTime int64
}

var zipPackThread *ZipPackThread
var zipPackThreadLock = sync.Mutex{}

func GetInstanceZipPackThread() *ZipPackThread {
	if zipPackThread != nil {
		return zipPackThread
	}
	zipPackThread = newZipPackThread()
	langconf.AddConfObserver("ZipPackThread", zipPackThread)

	go zipPackThread.run()

	return zipPackThread
}

func newZipPackThread() *ZipPackThread {
	p := &ZipPackThread{}
	p.conf = config.GetConfig()
	p.Queue = queue.NewRequestQueue(p.conf.TraceZipQueueSize)
	p.secuMaster = secure.GetSecurityMaster()

	return p
}

// implements Runnable of ConfObserver  (lang/conf)
func (this *ZipPackThread) Run() {
	this.Queue.SetCapacity(this.conf.TraceZipQueueSize)
}

func (this *ZipPackThread) Add(p pack.Pack) {
	ok := this.Queue.Put(p)
	if ok == false {
		// 큐가 차면 직접 압축없이 보낸다.
		Send(p)
		this.noZipSent += 1
	}
}

func (this *ZipPackThread) AddWait(p pack.Pack, waitTimeForFull int64) {
	ok := this.Queue.Put(p)
	if ok == false {
		if waitTimeForFull > 0 {
			for this.Queue.Put(p) == false {
				time.Sleep(time.Duration(waitTimeForFull) * time.Millisecond)
			}
		}
	}
}

func (this *ZipPackThread) run() {
	for {
		// shutdown
		if config.GetConfig().Shutdown {
			logutil.Infoln("WA211D-01", "Shutdown ZipPackThread")
			break
		}

		tmp := this.Queue.GetTimeout(this.conf.TraceZipMaxWait2Time)
		func() {
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA111")
				}
			}()
			if tmp != nil {
				if log, ok := tmp.(pack.Pack); ok {
					this.append(log)
				}
			} else {
				this.sendAndClear()
			}
		}()
	}
}

func (this *ZipPackThread) append(p pack.Pack) {
	out := pack.WritePack(wio.NewDataOutputX(), p)
	this.buffer.Write(out.ToByteArray())
	this.packCount++
	if this.firstTime == 0 {
		this.firstTime = p.GetTime()
		if this.buffer.Len() >= this.conf.TraceZipMaxBufferSize {
			this.sendAndClear()
		}
	} else {
		if this.buffer.Len() >= this.conf.TraceZipMaxBufferSize || int(p.GetTime()-this.firstTime) >= this.conf.TraceZipMaxWaitTime {
			this.sendAndClear()
		}
	}
}

func (this *ZipPackThread) sendAndClear() {
	if this.buffer.Len() == 0 {
		return
	}
	p := pack.NewZipPack()
	p.Time = dateutil.SystemNow()
	p.RecordCount = this.packCount
	p.Records = this.buffer.Bytes()

	this.doZip(p)
	if this.conf.DebugTraceZipEnabled {
		if this.conf.DebugTraceZipInterval <= 0 {
			this.log(p)
		} else {
			this.zipSent += 1
			now := dateutil.SystemNow()
			if now > this.lastLog+int64(this.conf.DebugTraceZipInterval) {
				this.lastLog = now
				this.log(p)
				this.zipSent = 0
			}
		}

	}

	p.Pcode = this.secuMaster.PCODE
	p.Oid = this.secuMaster.OID
	p.Okind = this.conf.OKIND
	p.Onode = this.conf.ONODE

	wnet.SendProfile(0, p, false)

	this.buffer.Reset()
	this.firstTime = 0
	this.packCount = 0

}

func (this *ZipPackThread) log(p *pack.ZipPack) {
	sb := stringutil.NewStringBuffer()
	sb.Append("ZipPackTrhead ").Append(ansi.ANSI_GREEN)
	sb.Append("zip_sent=").AppendFormat("%d", this.zipSent)
	sb.Append(" records=").AppendFormat("%d", p.RecordCount)
	sb.Append(" | ").AppendFormat("%d", this.buffer.Len()).Append("=>").AppendFormat("%d", len(p.Records))
	sb.Append(" queue=").AppendFormat("%d", this.Queue.Size())
	if this.noZipSent > 0 {
		sb.Append(" no_zip_sent=").AppendFormat("%d", this.noZipSent)
	}
	sb.Append(ansi.ANSI_RESET)
	logutil.Infoln(sb.ToString())

}

func (this *ZipPackThread) doZip(p *pack.ZipPack) {
	if p.Status != 0 {
		return
	}
	if len(p.Records) < this.conf.TraceZipMinSize {
		return
	}
	p.Status = 1 // gzip 알고리즘 사용
	var err error
	// sz := len(p.Records)

	p.Records, err = compressutil.DoZip(p.Records)

	// logutil.Infoln(">>>>", "compresutil.DoZip", ",sz=", sz, ", len=", len(p.Records), ",error=", err)

	if err != nil {
		logutil.Println("WA11111", "Error dozip ", err)
	}
}

func (this *ZipPackThread) flush() {
	for this.Queue.Size() > 0 {
		time.Sleep(10 * time.Millisecond)
		this.sendAndClear()
	}
}
