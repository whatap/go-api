package trace

import (
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/queue"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/secure"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	"github.com/whatap/go-api/agent/util/logutil"
)

type StepSplit struct {
	Time  int64
	Txid  int64
	Inx   int
	Steps []step.Step
}

func NewStepSplit(t, txid int64, inx int, steps []step.Step) *StepSplit {
	return &StepSplit{
		Time:  t,
		Txid:  txid,
		Inx:   inx,
		Steps: steps,
	}
}

type ProfileStepThread struct {
	Queue      *queue.RequestQueue
	drop       int64
	conf       *config.Config
	secuMaster *secure.SecurityMaster
}

var profileStepThread *ProfileStepThread
var profileStepThreadLock = sync.Mutex{}

func GetInstanceProfileStepThread() *ProfileStepThread {
	profileStepThreadLock.Lock()
	defer profileStepThreadLock.Unlock()

	if profileStepThread != nil {
		return profileStepThread
	}
	profileStepThread = newProfileStepThread()
	langconf.AddConfObserver("ProfileStepThread", profileStepThread)

	go profileStepThread.run()

	return profileStepThread
}

func newProfileStepThread() *ProfileStepThread {
	p := &ProfileStepThread{}
	p.conf = config.GetConfig()
	p.Queue = queue.NewRequestQueue(p.conf.TraceTxSplitQueueSize)
	p.secuMaster = secure.GetSecurityMaster()
	return p
}

// implements Runnable of ConfObserver  (lang/conf)
func (this *ProfileStepThread) Run() {
	this.Queue.SetCapacity(this.conf.TraceTxSplitQueueSize)
}

func (this *ProfileStepThread) Add(t, txid int64, inx int, steps []step.Step) {
	this.Queue.Put(NewStepSplit(t, txid, inx, steps))
}

func (this *ProfileStepThread) run() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA1300", "Recover ", r)
				}
			}()

			tmp := this.Queue.GetTimeout(1000)
			if tmp != nil {
				if log, ok := tmp.(*StepSplit); ok {
					if log != nil {
						p := pack.NewProfileStepSplitPack()

						p.Pcode = this.secuMaster.PCODE
						p.Oid = this.secuMaster.OID
						p.Time = log.Time
						p.Txid = log.Txid
						p.Inx = log.Inx
						p.Steps = step.ToBytesStep(log.Steps)

						GetInstanceZipProfileThread().Add(p)
					}
				}
			}
		}()
	}
}
