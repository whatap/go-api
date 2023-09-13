package trace

import (
	// "fmt"
	"sync"

	// "github.com/whatap/golib/lang"
	// "github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/step"
	"github.com/whatap/golib/util/dateutil"

	// "github.com/whatap/golib/util/hash"
	// "github.com/whatap/golib/util/keygen"

	"github.com/whatap/go-api/agent/agent/config"
	// "gitlab.whatap.io/go/agent/agent/data"
	// "github.com/whatap/go-api/agent/util/logutil"
)

type ProfileLargeCollector struct {
	BUFFER_MAX int
	buffer     []step.Step
	bufferPos  int

	splitCount  int
	position    int32
	parentIndex int32

	parent *TraceContext

	profile *ProfileStepThread
	lock    sync.Mutex
	conf    *config.Config
}

func NewProfileLargeCollector(parent *TraceContext) *ProfileLargeCollector {
	//logutil.Infoln(">>>>", "New ProfileLargeCollector")

	p := new(ProfileLargeCollector)
	p.parent = parent
	p.conf = config.GetConfig()
	p.BUFFER_MAX = p.conf.TraceStepMaxCount
	p.buffer = make([]step.Step, p.BUFFER_MAX)

	p.splitCount = 0
	p.position = 0
	p.parentIndex = -1

	p.profile = GetInstanceProfileStepThread()

	return p
}

func (this *ProfileLargeCollector) HasStep() bool {
	return this.position > 0
}

func (this *ProfileLargeCollector) Push(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	st.SetIndex(this.position)
	st.SetParent(this.parentIndex)
	this.parentIndex = this.position
	this.position++
}

func (this *ProfileLargeCollector) Add(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.bufferPos >= this.BUFFER_MAX {
		this.send(this.splitCount, this.buffer)
		this.buffer = make([]step.Step, this.BUFFER_MAX)
		this.bufferPos = 0
		this.splitCount++
	}

	st.SetIndex(this.position)
	st.SetParent(this.parentIndex)
	this.buffer[this.bufferPos] = st
	this.bufferPos++
	this.position++
}

func (this *ProfileLargeCollector) AddHeavy(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.bufferPos >= this.BUFFER_MAX {
		this.send(this.splitCount, this.buffer)
		this.buffer = make([]step.Step, this.BUFFER_MAX)
		this.bufferPos = 0
		this.splitCount++
	}

	st.SetIndex(this.position)
	st.SetParent(this.parentIndex)
	this.buffer[this.bufferPos] = st
	this.bufferPos++
	this.position++
}

func (this *ProfileLargeCollector) JustAdd(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.bufferPos >= this.BUFFER_MAX {
		this.send(this.splitCount, this.buffer)
		this.buffer = make([]step.Step, this.BUFFER_MAX)
		this.bufferPos = 0
		this.splitCount++
	}
	this.buffer[this.bufferPos] = st
	this.bufferPos++
	this.position++
}

func (this *ProfileLargeCollector) AppendStep(st step.Step) {
	this.Add(st)
}

func (this *ProfileLargeCollector) AddTail(st step.Step) {
	this.Add(st)
}

func (this *ProfileLargeCollector) Pop(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferPos >= this.BUFFER_MAX {
		this.send(this.splitCount, this.buffer)
		this.buffer = make([]step.Step, this.BUFFER_MAX)
		this.bufferPos = 0
		this.splitCount++
	}

	this.buffer[this.bufferPos] = st
	this.bufferPos++
	this.parentIndex = st.GetParent()
}

func (this *ProfileLargeCollector) send(inx int, buff []step.Step) {
	this.profile.Add(dateutil.Now(), this.parent.Txid, inx, buff)
}

func (this *ProfileLargeCollector) GetLastSteps(n int) []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()

	cnt := this.bufferPos
	if this.bufferPos > n {
		cnt = n
	}
	buff := make([]step.Step, cnt)
	x := this.bufferPos - cnt
	for i := 0; i < cnt; i++ {
		buff[i] = this.buffer[x+i]
	}
	return buff
}
func (this *ProfileLargeCollector) GetSteps() []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.bufferPos > len(this.buffer) {
		return this.buffer
	}
	buff := make([]step.Step, 0)
	if this.bufferPos > 0 {
		buff = append(buff, this.buffer[:this.bufferPos]...)
	}
	return buff
}

func (this *ProfileLargeCollector) ToBytes() []byte {
	return step.ToBytesStep(this.GetSteps())
}

func (this *ProfileLargeCollector) GetStep4Error() []step.Step {
	return this.GetLastSteps(5)
}

func (this *ProfileLargeCollector) IsReal() bool {
	return true
}

func (this *ProfileLargeCollector) Append(steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		this.Add(st)
	}
}

func (this *ProfileLargeCollector) AppendParentOffsetTime(parentOffsetTime int, steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		st.SetStartTime(int32(parentOffsetTime))
		this.Add(st)
	}
}

func (this *ProfileLargeCollector) GetSplitCount() int {
	return this.splitCount
}
func (this *ProfileLargeCollector) Length() int {
	return this.splitCount*this.BUFFER_MAX + int(this.position)
}
