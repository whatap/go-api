package zip

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/ansi"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/queue"
)

type ZipSendProxyThread struct {
	Queue     *queue.RequestQueue
	buffer    bytes.Buffer
	packCount int
	firstTime int64
}

var zipSendProxyThread *ZipSendProxyThread
var zipSendProxyThreadMutex = sync.Mutex{}

func GetInstance() *ZipSendProxyThread {
	zipSendProxyThreadMutex.Lock()
	defer zipSendProxyThreadMutex.Unlock()
	if zipSendProxyThread != nil {
		return zipSendProxyThread
	}
	ConfLogSink := config.GetConfig().ConfLogSink
	p := new(ZipSendProxyThread)
	p.Queue = queue.NewRequestQueue(int(ConfLogSink.LogSinkQueueSize))
	zipSendProxyThread = p
	go zipSendProxyThread.run()

	return zipSendProxyThread
}

func (this *ZipSendProxyThread) Add(p *pack.LogSinkPack) {
	this.Queue.Put(p)
}

func (this *ZipSendProxyThread) run() {
	ConfLogSink := config.GetConfig().ConfLogSink
	for true {
		if tmp := this.Queue.GetTimeout(int(ConfLogSink.MaxWaitTime)); tmp != nil {
			log := tmp.(*pack.LogSinkPack)
			if log != nil {
				this.Append(log)
			} else {
				this.sendAndClear()
			}
		}
	}
}

func (this *ZipSendProxyThread) Append(p *pack.LogSinkPack) {
	defer func() {
		if r := recover(); r != nil {
			// Recover
			logutil.Println("WA-LOGS-101", "Recover Append ", r)
		}
	}()

	ConfLogSink := config.GetConfig().ConfLogSink
	dout := io.NewDataOutputX()
	pack.WritePack(dout, p)
	this.buffer.Write(dout.ToByteArray())
	this.packCount += 1

	if this.firstTime == 0 {
		this.firstTime = p.Time
		if this.buffer.Len() >= int(ConfLogSink.MaxBufferSize) {
			this.sendAndClear()
		}
	} else {
		if this.buffer.Len() >= int(ConfLogSink.MaxBufferSize) || p.Time-this.firstTime >= int64(ConfLogSink.MaxWaitTime) {
			this.sendAndClear()
		}
	}
}

func (this *ZipSendProxyThread) sendAndClear() {
	if this.buffer.Len() == 0 {
		return
	}
	ConfLogSink := config.GetConfig().ConfLogSink

	p := pack.NewZipPack()
	p.Time = dateutil.SystemNow()
	p.RecordCount = this.packCount
	p.Records = this.buffer.Bytes()

	this.doZip(p)
	if ConfLogSink.DebugLogSinkZipEnabled {
		logutil.Println("WA-LOGS-102", fmt.Sprintln("LogSink ",
			ansi.Green(fmt.Sprintln("Zip status=", p.Status, " records=", p.RecordCount, " | ",
				this.buffer.Len(), "=>", len(p.Records)))))
	}

	data.Send(p)
	this.buffer.Reset()
	this.firstTime = 0
	this.packCount = 0
}

func (this *ZipSendProxyThread) doZip(p *pack.ZipPack) {
	ConfLogSink := config.GetConfig().ConfLogSink
	if p.Status != 0 {
		return
	}
	if len(p.Records) < int(ConfLogSink.LogSinkZipMinSize) {
		return
	}
	// TODO
	//	z := ZipModLoader.zipImpl
	//
	//	p.Status = z.ID()
	//	p.Records = z.Compress(p.Records)

	z := NewDefaultZipMod()
	p.Status = z.ID()
	var err error
	//logutil.Infoln(">>>>", "before len=", len(p.Records), "-", string(p.Records))
	if p.Records, err = z.Compress(p.Records); err != nil {
		logutil.Println("WA-LOGS-103", "Compress Error ", err)
	}
}
