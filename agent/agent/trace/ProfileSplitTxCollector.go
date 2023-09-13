package trace

import (
	"fmt"
	"sync"

	"github.com/whatap/golib/lang"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/keygen"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	// "github.com/whatap/go-api/agent/util/logutil"
)

type ProfileSplitTxCollector struct {
	BUFFER_MAX      int
	bufferParent    []step.Step
	bufferChild     []step.Step
	bufferParentPos int

	bufferChildPos       int
	parentStepSplitCount int
	childSplitTxNum      int

	parentIndex int32
	childIndex  int32

	parent        *TraceContext
	parentProfile *ProfileStepThread
	profile       *ProfileVirtualTxThread

	childStart    int
	virtualTxHash int32

	lock sync.Mutex
	conf *config.Config
}

func NewProfileSplitTxCollector(parent *TraceContext) *ProfileSplitTxCollector {
	//logutil.Infoln(">>>>", "New ProfileSplitTxCollector")

	p := new(ProfileSplitTxCollector)
	p.parent = parent
	p.conf = config.GetConfig()
	p.BUFFER_MAX = p.conf.TraceStepMaxCount
	p.bufferParent = make([]step.Step, p.BUFFER_MAX)
	p.bufferChild = make([]step.Step, p.BUFFER_MAX)

	p.childSplitTxNum = 1
	p.parentIndex = -1

	p.parentProfile = GetInstanceProfileStepThread()
	p.profile = GetInstanceProfileVirtualTxThread()
	p.childStart = 0
	p.virtualTxHash = 0

	return p
}

func (this *ProfileSplitTxCollector) HasStep() bool {
	return this.bufferChildPos > 0 || this.bufferParentPos > 0
}

func (this *ProfileSplitTxCollector) Push(step step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	step.SetParent(this.parentIndex)
	step.SetIndex(this.childIndex)
	this.parentIndex = step.GetIndex()
	this.childIndex++
}

func (this *ProfileSplitTxCollector) Add(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferChildPos >= this.BUFFER_MAX {
		this.send(this.childSplitTxNum, this.bufferChild)
		this.bufferChild = make([]step.Step, this.BUFFER_MAX)
		this.bufferChildPos = 0
		this.childSplitTxNum += 1
	}

	st.SetParent(this.parentIndex)
	st.SetIndex(this.childIndex)
	this.childIndex += 1

	this.bufferChild[this.bufferChildPos] = st
	this.bufferChildPos += 1
}

func (this *ProfileSplitTxCollector) AddHeavy(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferChildPos >= this.BUFFER_MAX {
		this.send(this.childSplitTxNum, this.bufferChild)
		this.bufferChild = make([]step.Step, this.BUFFER_MAX)
		this.bufferChildPos = 0
		this.childSplitTxNum += 1
	}

	st.SetParent(this.parentIndex)
	st.SetIndex(this.childIndex)
	this.childIndex += 1

	this.bufferChild[this.bufferChildPos] = st
	this.bufferChildPos += 1
}

func (this *ProfileSplitTxCollector) JustAdd(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferChildPos >= this.BUFFER_MAX {
		this.send(this.childSplitTxNum, this.bufferChild)
		this.bufferChild = make([]step.Step, this.BUFFER_MAX)
		this.bufferChildPos = 0
		this.childSplitTxNum += 1
	}

	this.bufferChild[this.bufferChildPos] = st
	this.bufferChildPos += 1
}

func (this *ProfileSplitTxCollector) AppendStep(st step.Step) {
	this.Add(st)
}
func (this *ProfileSplitTxCollector) AddTail(st step.Step) {
	this.Add(st)
}

func (this *ProfileSplitTxCollector) Pop(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferChildPos >= this.BUFFER_MAX {
		this.send(this.childSplitTxNum, this.bufferChild)
		this.bufferChild = make([]step.Step, this.BUFFER_MAX)
		this.bufferChildPos = 0
		this.childSplitTxNum += 1
	}

	this.bufferChild[this.bufferChildPos] = st
	this.bufferChildPos += 1

	this.parentIndex = st.GetParent()
}

func (this *ProfileSplitTxCollector) send(inx int, buff []step.Step) {
	txElapsed := this.parent.GetElapsedTime()
	childElapsed := txElapsed - this.childStart
	childName := fmt.Sprintf("%s-%d", this.parent.ServiceName, inx)
	childTxid := keygen.Next()
	this.profile.Add(dateutil.SystemNow(), childName, childTxid, this.childStart, childElapsed, this.parent, buff)

	if this.bufferParentPos >= this.BUFFER_MAX {
		this.sendParent(this.parentStepSplitCount, this.bufferParent)
		this.bufferParent = make([]step.Step, this.BUFFER_MAX)
		this.bufferParentPos = 0
		this.parentStepSplitCount += 1
	}

	if this.virtualTxHash == 0 {
		this.virtualTxHash = hash.HashStr(lang.MESSAGE_VIRTUAL_TX)
		data.SendHashText(pack.TEXT_MESSAGE, this.virtualTxHash, lang.MESSAGE_VIRTUAL_TX)
	}

	st := step.NewMessageStep()
	st.StartTime = int32(this.childStart)
	st.Hash = this.virtualTxHash
	st.Desc = fmt.Sprintf("%s <%d>", childName, childTxid)
	st.Time = int32(childElapsed)

	this.bufferParent[this.bufferParentPos] = st
	this.bufferParentPos += 1
	this.childStart = txElapsed
}

func (this *ProfileSplitTxCollector) sendParent(inx int, buff []step.Step) {
	this.parentProfile.Add(dateutil.SystemNow(), this.parent.Txid, inx, buff)
}

func (this *ProfileSplitTxCollector) GetLastSteps(n int) []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()

	cnt := int(this.bufferChildPos)
	if int(this.bufferChildPos) > n {
		cnt = n
	}

	buff := make([]step.Step, cnt)
	x := this.bufferChildPos - cnt
	for i := 0; i < cnt; i++ {
		buff[i] = this.bufferChild[x+i]
	}
	return buff
}

func (this *ProfileSplitTxCollector) GetSteps() []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferParentPos == 0 && this.bufferChildPos == 0 {
		return make([]step.Step, 0)
	}

	if this.bufferParentPos == 0 {
		if this.bufferChildPos >= len(this.bufferChild) {
			return this.bufferChild
		}

		buff := make([]step.Step, 0)
		if this.bufferChildPos > 0 {
			buff = append(buff, this.bufferChild[:this.bufferChildPos]...)
		}
		return buff
	}

	if this.bufferChildPos == 0 {
		buff := make([]step.Step, 0)
		if this.bufferParentPos > 0 {
			buff = append(buff, this.bufferParent[:this.bufferParentPos]...)
		}
		return buff
	}

	buff := make([]step.Step, 0)
	buff = append(buff, this.bufferParent[:this.bufferParentPos]...)
	buff = append(buff, this.bufferChild[:this.bufferChildPos]...)
	return buff
}

func (this *ProfileSplitTxCollector) ToBytes() []byte {
	return step.ToBytesStep(this.GetSteps())
}

func (this *ProfileSplitTxCollector) GetStep4Error() []step.Step {
	return this.GetLastSteps(5)
}

func (this *ProfileSplitTxCollector) IsReal() bool {
	return true
}

func (this *ProfileSplitTxCollector) Append(steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		this.Add(st)
	}
}

func (this *ProfileSplitTxCollector) AppendParentOffsetTime(parentOffsetTime int, steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		st.SetStartTime(st.GetStartTime() + int32(parentOffsetTime))
		this.Add(st)
	}
}

func (this *ProfileSplitTxCollector) GetSplitCount() int {
	return this.parentStepSplitCount
}

func (this *ProfileSplitTxCollector) Length() int {
	return this.parentStepSplitCount*this.BUFFER_MAX + this.bufferChildPos
}
