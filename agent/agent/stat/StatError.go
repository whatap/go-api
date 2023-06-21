package stat

import (
	//	"fmt"
	_ "fmt"
	//"log"
	"strings"
	"sync"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/bitutil"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/util/logutil"
)

type StatError struct {
	// table *LongKeyLinkedMap
	table *hmap.LongKeyLinkedMap

	// ErrorSnapSeq int64
	ErrorSnapSeq int64

	// timer *TimingSender
	timer *TimingSender
}

const (
	STAT_ERROR_TABLE_MAX_SIZE = 500
)

var statErroLock = sync.Mutex{}
var statError *StatError

// Singleton func GetInstanceStatError() *StatError {
func GetInstanceStatError() *StatError {
	statErroLock.Lock()
	defer statErroLock.Unlock()
	if statError != nil {
		return statError
	}
	statError = new(StatError)
	statError.table = hmap.NewLongKeyLinkedMapDefault().SetMax(STAT_ERROR_TABLE_MAX_SIZE)
	statError.ErrorSnapSeq = dateutil.Now()
	statError.timer = GetInstanceTimingSender()

	return statError
}

func (this *StatError) NextErrorSnapSeq() int64 {
	statErroLock.Lock()
	defer statErroLock.Unlock()
	this.ErrorSnapSeq++
	return this.ErrorSnapSeq
}

// Java Throwable struct
type ErrorThrowable struct {
	//ErrorType int32
	ErrorType      int32
	ErrorClassName string
	ErrorCode      int64
	ErrorMessage   string
	ErrorStack     []int32
}

func NewErrorThrowable() *ErrorThrowable {
	p := new(ErrorThrowable)
	return p
}

func (this *StatError) AddErrorHashOnly(thr *ErrorThrowable, msg string) (ret int64) {
	defer func() {
		if r := recover(); r != nil {
			// Continue
			ret = 0
		}
	}()
	classHash := hash.HashStr(thr.ErrorClassName)
	data.SendHashText(pack.TEXT_ERROR, classHash, thr.ErrorClassName)
	msg = stringutil.TrimEmpty(msg)
	//msg = stringutil.Truncate(msg, 200)
	msgHash := hash.HashStr(msg)
	data.SendHashText(pack.TEXT_ERROR, msgHash, msg)
	// return
	ret = bitutil.Composite64(classHash, msgHash)
	return
}

// public long addError(Throwable thr, String msg, int txUrlHash, ProfileCollector profile, byte type, int hash) {
// TODO impot cycle 확인  ProfileCollector
func (this *StatError) AddError(thr *ErrorThrowable, msg string, txUrlHash int32, errorSnapEnabled bool, steps []step.Step, Type byte, Hash int32) int64 {
	//fmt.Println("StatError, AddError", ",msg=", msg, ",msg==\"\"", (msg == ""))
	if msg == "" {
		return this.addError(thr, 0, txUrlHash, errorSnapEnabled, steps, Type, Hash)
	} else {
		msg = stringutil.TrimEmpty(msg)
		//msg = stringutil.Truncate(msg, 200)
		msgHash := hash.HashStr(msg)

		data.SendHashText(pack.TEXT_ERROR, msgHash, msg)

		return this.addError(thr, msgHash, txUrlHash, errorSnapEnabled, steps, Type, Hash)
	}
}

// public long addError(Throwable thr, int msgHash, int txUrlHash, ProfileCollector profile, byte type, int hash) {
// import cycle 을 위해서 profile *trace.ProfileCollector 을 errorSnapEnabled, []Steps 로 변경
func (this *StatError) addError(thr *ErrorThrowable, msgHash, txUrlHash int32, errorSnapEnabled bool, steps []step.Step, Type byte, Hash int32) int64 {
	//fmt.Println("StatError, addError", "errorSnapEnabled=", errorSnapEnabled, ",steps=", steps, ",Type=", Type, ",hash=", Hash)
	var ret int64

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA511", "StatError Recover:", r)
			// 에러 발생할 경우 0 반환
			ret = 0
		}
	}()

	// TODO Error ClassName, Error Type 처리
	//		String class1 = thr.getClass().getName();
	//		if (class1.equals(SQLException.class.getName())) {
	//			class1 = new StringBuffer().append(class1).append("(").append(((SQLException) thr).getErrorCode()).append(")").toString();
	//		}

	classHash := hash.HashStr(thr.ErrorClassName)
	//fmt.Println("StatError, classHasht=", classHash, ",ErrorClassName=", thr.ErrorClassName)

	// import cycle whatap/data 조심
	data.SendHashText(pack.TEXT_ERROR, classHash, thr.ErrorClassName)

	key := bitutil.Composite64(classHash, txUrlHash)
	//fmt.Println("StatError, txUrlHash=", txUrlHash, ",key=", key)

	recIf := this.table.Get(key)

	var rec *pack.ErrorRec
	if recIf == nil {
		rec = pack.NewErrorRec()

		rec.ClassHash = classHash
		rec.Service = txUrlHash
		rec.Msg = msgHash

		this.table.Put(key, rec)
	} else {
		rec = recIf.(*pack.ErrorRec)
	}

	if rec.SnapSeq == 0 && errorSnapEnabled {
		//try catch
		func() {
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA512", "StatError.addError Stap Recover:", r)
				}
			}()
			p := pack.NewErrorSnapPack1()
			p.Seq = this.NextErrorSnapSeq()

			rec.SnapSeq = p.Seq
			rec.Msg = msgHash
			p.SetProfile(steps)
			p.SetStack(thr.ErrorStack)

			// whatap 내부에서 정의된 Exception들은 스택을 생성 전달하지 않는다.
			// 객체가 singleton이기 때문에 스택이 맞지도 않는다. @paul
			if !strings.HasPrefix(thr.ErrorClassName, "whatap.agent.error") {
				//p.setStack(StackDumpUtil.getStack(thr, Configure.getInstance().trace_error_callstack_depth));
			}

			p.AppendType = Type
			p.AppendHash = Hash
			//
			p.Time = dateutil.Now()
			data.Send(p)
			//fmt.Println("StatError, data.send", p)
		}()
	}

	rec.Count++
	ret = bitutil.Composite64(classHash, msgHash)
	//fmt.Println("StatError, addError, ret=", ret, ",classHash=", classHash, ",msgHash=", msgHash)
	return ret
	//return bitutil.Composite64(classHash, msgHash)
}

func (this *StatError) Send(now int64) {
	if this.table.Size() == 0 {
		return
	}

	p := pack.NewStatErrorPack().SetRecords(this.table.Size(), this.table.Values())
	p.Time = now
	this.table.Clear()
	data.Send(p)
}

func (this *StatError) Clear() {
	this.table.Clear()
}

func (this *StatError) AddErrorSql(thr *ErrorThrowable, dbc, service int32, steps []step.Step, stackTrace bool) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA513", "StatError.AddErrorSql Recover:", r)
			// 에러 발생할 경우 0 반환
		}
	}()
	//			String class1 = thr.getClass().getName();
	//			if (class1.equals(SQLException.class.getName())) {
	//				class1 = new StringBuffer().append(class1).append("(").append(((SQLException) thr).getErrorCode()).append(")").toString();
	//			}
	//
	classHash := hash.HashStr(thr.ErrorClassName)
	msgHash := dbc
	data.SendHashText(pack.TEXT_ERROR, classHash, thr.ErrorClassName)

	key := bitutil.Composite64(classHash, service)

	recIf := this.table.Get(key)

	var rec *pack.ErrorRec
	if recIf == nil {
		rec = pack.NewErrorRec()

		rec.ClassHash = classHash
		rec.Service = service
		rec.Msg = msgHash

		this.table.Put(key, rec)
	} else {
		rec = recIf.(*pack.ErrorRec)
	}

	if rec.SnapSeq == 0 && len(steps) > 0 {
		//try catch
		func() {
			defer func() {
				if r := recover(); r != nil {
					logutil.Println("WA514", "StatError.AddErrorSql Stap Recover:", r)
				}
			}()
			p := pack.NewErrorSnapPack1()
			p.Seq = this.NextErrorSnapSeq()
			rec.SnapSeq = p.Seq
			rec.Msg = msgHash

			p.SetProfile(steps)

			if stackTrace {
				// stackTrace
				// Java stack 배열에서 설정 trace_error_callstack_depth 만큼 추출
				// StackDumpUtil.stack 에서 각 스택별로 해시를 추출 int[] 을 setStack 에 전달.
				//						StackTraceElement[] stack = thr.getStackTrace();
				//						if (stack.length > 2) {
				//							StackTraceElement[] newStack = new StackTraceElement[stack.length - 2];
				//							System.arraycopy(stack, 2, newStack, 0, newStack.length);
				//							stack = newStack;
				//						}
				//						p.setStack(StackDumpUtil.stack(stack, Configure.getInstance().trace_error_callstack_depth));
			}

			p.Time = dateutil.Now()
			data.Send(p)
			//fmt.Println("StatError.AddErrorSql, data.send", p)
		}()
	}

	rec.Count++

}
