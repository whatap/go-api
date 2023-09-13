package trace

import (
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/service"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/queue"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/data"

	"github.com/whatap/go-api/agent/agent/secure"
	langconf "github.com/whatap/go-api/agent/lang/conf"
	// "github.com/whatap/go-api/agent/util/logutil"
)

type ChildTx struct {
	etime        int64
	childStart   int32
	parentTxid   int64
	childTxid    int64
	steps        []step.Step
	service      string
	childElapsed int32
	sqlCount     int32
	sqlTime      int32
	rsCount      int32
	rsTime       int32
	httpcCount   int32
	httpcTime    int32
	dbcTime      int32
}

type ProfileVirtualTxThread struct {
	Queue      *queue.RequestQueue
	secuMaster *secure.SecurityMaster
	conf       *config.Config
}

var profileVirtualTxThread *ProfileVirtualTxThread
var profileVirtualTxThreadLock sync.Mutex

func GetInstanceProfileVirtualTxThread() *ProfileVirtualTxThread {
	profileVirtualTxThreadLock.Lock()
	defer profileVirtualTxThreadLock.Unlock()

	if profileVirtualTxThread != nil {
		return profileVirtualTxThread
	}
	profileVirtualTxThread := newProfileVirtualTxThread()
	go profileVirtualTxThread.run()
	langconf.AddConfObserver("ProfileVirtualTxThread", profileVirtualTxThread)

	return profileVirtualTxThread
}

func newProfileVirtualTxThread() *ProfileVirtualTxThread {
	p := &ProfileVirtualTxThread{}
	p.conf = config.GetConfig()
	p.Queue = queue.NewRequestQueue(p.conf.TraceTxSplitQueueSize)
	p.secuMaster = secure.GetSecurityMaster()

	return p
}

// implements Runnable of ConfObserver  (lang/conf)
func (this *ProfileVirtualTxThread) Run() {
	this.Queue.SetCapacity(this.conf.TraceTxSplitQueueSize)
}

func (this *ProfileVirtualTxThread) Add(curTime int64, childName string, childTxid int64, childStart, childElapsed int,
	p *TraceContext, steps []step.Step) {

	x := &ChildTx{}
	x.etime = curTime
	x.childStart = int32(childStart)
	x.parentTxid = p.Txid
	x.childTxid = childTxid
	x.steps = steps
	x.service = childName
	x.childElapsed = int32(childElapsed)

	x.sqlCount = p.SqlCount
	x.sqlTime = p.SqlTime
	x.rsCount = p.RsCount
	x.rsTime = int32(p.RsTime)
	x.httpcCount = p.HttpcCount
	x.httpcTime = p.HttpcTime
	x.dbcTime = p.DbcTime

	this.Queue.Put(x)
}

func (this *ProfileVirtualTxThread) run() {
	for {

		func() {
			defer func() {
				if r := recover(); r != nil {
				}
			}()

			tmp := this.Queue.Get()
			if log, ok := tmp.(*ChildTx); ok {

				p := pack.NewProfilePack()
				p.Pcode = this.secuMaster.PCODE
				p.Oid = this.secuMaster.OID
				p.Time = log.etime

				serviceHash := hash.HashStr(log.service)
				data.SendHashText(pack.TEXT_SERVICE, serviceHash, log.service)

				tx := service.NewTxRecord()
				tx.Txid = log.childTxid
				tx.Mcaller = log.parentTxid
				tx.EndTime = log.etime
				tx.Service = serviceHash
				tx.Elapsed = log.childElapsed

				tx.Fields = value.NewMapValue()
				tx.Fields.PutLong("ParentTxid", log.parentTxid)
				tx.Fields.PutString("TxType", "Virtual")
				tx.Fields.PutLong("FirstStepIdx", int64(log.steps[0].GetIndex()))

				if log.childStart >= 0 {
					sqlCount := int32(0)
					sqlTime := int32(0)
					httpcCount := int32(0)
					httpcTime := int32(0)
					rsCount := int32(0)
					rsTime := int32(0)
					dbcTime := int32(0)
					for i := 0; i < len(log.steps); i++ {
						sp := log.steps[i]
						sp.SetStartTime(sp.GetStartTime() - int32(log.childStart))

						switch sp.GetStepType() {
						case step.STEP_SQL_X:
							sqlCount++
							sqlTime += sp.GetElapsed()
						case step.STEP_DBC:
							dbcTime += sp.GetElapsed()
						case step.STEP_HTTPCALL_X:
							httpcCount++
							httpcTime += sp.GetElapsed()
						case step.STEP_RESULTSET:
							if st, ok := sp.(*step.ResultSetStep); ok {
								rsCount += st.Fetch
							}
							rsTime += sp.GetElapsed()
						}
					}
					tx.SqlCount = sqlCount
					tx.SqlTime = sqlTime
					tx.SqlFetchCount = rsCount
					tx.SqlFetchTime = rsTime
					tx.HttpcCount = httpcCount
					tx.HttpcTime = httpcTime
					tx.DbcTime = dbcTime

					tx.Fields.PutLong("ParentElapsed", int64(log.childStart+log.childElapsed))
					tx.Fields.PutLong("ParentElapsed", int64(log.childStart+log.childElapsed))

					if log.sqlCount > 0 {
						tx.Fields.PutLong("ParentSqlCount", int64(log.sqlCount))
						tx.Fields.PutLong("ParentSqlTime", int64(log.sqlTime))
					}
					if log.httpcCount > 0 {
						tx.Fields.PutLong("ParentHttpCallCount", int64(log.httpcCount))
						tx.Fields.PutLong("ParentHttpCallTime", int64(log.httpcTime))
					}
					if log.rsCount > 0 {
						tx.Fields.PutLong("ParentFetchCount", int64(log.rsCount))
						tx.Fields.PutLong("ParentFetchTime", int64(log.rsTime))
					}
					if log.dbcTime > 0 {
						tx.Fields.PutLong("ParentDbcTime", int64(log.dbcTime))
					}

				}

				// logutil.Infof(">>>>", "service=%s, sql: %d, %d, httpc: %d, %d fetch: %d, %d",
				// 	log.service, tx.SqlCount, tx.SqlTime, tx.HttpcCount, tx.HttpcTime, tx.SqlFetchCount, tx.SqlFetchTime)

				meter.GetInstanceMeterService().AddVirtualTx(tx)

				p.Transaction = tx
				p.Steps = step.ToBytesStep(log.steps)

				GetInstanceZipProfileThread().Add(p)

			}
		}()

	}

}
