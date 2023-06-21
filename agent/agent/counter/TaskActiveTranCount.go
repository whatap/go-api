package counter

import (
	//"log"
	//"github.com/whatap/go-api/agent/agent/active"
	// "github.com/whatap/go-api/agent/agent/active/udp"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/counter/meter"
	"github.com/whatap/go-api/agent/agent/trace"
	"github.com/whatap/go-api/agent/util/logutil"

	// "github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/dateutil"
)

type TaskActiveTranCount struct {
}

func NewTaskActiveTranCount() *TaskActiveTranCount {
	p := new(TaskActiveTranCount)
	return p
}

func (this *TaskActiveTranCount) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA311", "process Recover", r)
		}
	}()
	conf := config.GetConfig()
	if !conf.CounterEnabledAct_ {
		if conf.CounterLogEnabled {
			logutil.Println("WA311-01", "Disable counter, active tran count")
		}
		return
	} else {
		if conf.CounterLogEnabled {
			logutil.Println("WA311-02", "Start counter, active tran count")
		}
	}

	p.ActSvcSlice = make([]int16, 3)

	var activeX *meter.MeterActiveX = nil
	if conf.ActxMeterEnabled {
		activeX = meter.GetInstanceMeterActiveX()
		activeX.ReInit()
	}

	en := trace.GetContextEnumeration()
	var tmp interface{}

	for en.HasMoreElements() {
		tmp = en.NextElement()
		if tmp == nil {
			logutil.Println("WA312", "TaskActiveTranCount TraceConext is nil")
			continue
		}
		ctx := tmp.(*trace.TraceContext)
		//fmt.Println("TraceContext ctx=", ctx.ProfileSeq)

		if ctx == nil {
			continue
		}

		// TODO 패킷 유실관련 해서 redTime 이상의 한 단계를 더 두고 해당 tranx 은 종료 및 삭제 시킴,
		// 패킷을 버리지 말고 Mssage Step 으로 Timeout 추가 후 트랜잭션 종료 필요
		// TODO 현재시간과 어떤 시간을 비교할 지 결정 transaction 시작 시간(PHP Extension 에서 보낸 시간) 또는 Agent에서 StartTx 를 받은 시간
		if int32(dateutil.SystemNow()-ctx.StartTime) > int32(conf.TraceActiveTransactionLostTime) {
			// 정상 적인 종료 처리 진행.
			trace.RemoveLostContext(ctx.ProfileSeq)
		} else if int32(dateutil.SystemNow()-ctx.StartTime) < int32(conf.TraceActiveTransactionSlowTime) {
			p.ActSvcSlice[0]++
		} else if int32(dateutil.SystemNow()-ctx.StartTime) < int32(conf.TraceActiveTransactionVerySlowTime) {
			p.ActSvcSlice[1]++
		} else {
			p.ActSvcSlice[2]++
		}
		p.ActSvcCount++

		if ctx.WClientId != 0 {
			meter.AddActiveMeterUsers(ctx.WClientId)
		}

		// caller별 active tx에 대한 건수를 수집한다.
		//		if activeX != nil {
		//			activeX.AddTx(ctx.McallerPcode, ctx.McallerOkind, ctx.McallerOid)
		//			httpcHostHash := ctx.HttpHostHash
		//			if httpcHostHash != 0 {
		//				activeX.AddHttpc(httpcHostHash);
		//			} else {
		//				sql := ctx.sql
		//				if sql != nil {
		//					activeX.AddSql(sql.dbc)
		//				}
		//			}
		//		}

		this.CountingStatus(ctx, p)
	}
}

func (this *TaskActiveTranCount) CountingStatus(ctx *trace.TraceContext, p *pack.CounterPack1) {
	if ctx.ActiveSqlhash != 0 {
		p.ActiveStat[1]++ // sql
	} else if ctx.ActiveHttpcHash != 0 {
		p.ActiveStat[2]++ // httpc
		// } else if ctx.db_opening {
	} else if ctx.ActiveDbc != 0 {
		p.ActiveStat[3]++ // dbc
		//		} else if ctx.socket_connecting {
		//			p.ActiveStat[4]++ // socket
		//		}
	} else {
		p.ActiveStat[0]++ // method
	}
}
