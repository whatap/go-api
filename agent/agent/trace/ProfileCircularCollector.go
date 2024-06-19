package trace

import (
	"sync"

	"github.com/whatap/golib/lang/step"

	"github.com/whatap/go-api/agent/agent/config"
	// "github.com/whatap/go-api/agent/util/logutil"
)

type ProfileCircularCollector struct {
	BUFFER_LEN int
	buffer     []step.Step

	loop        int
	position    int
	thisIndex   int32
	parentIndex int32

	conf *config.Config
	lock sync.Mutex
}

func NewProfileCircularCollector() *ProfileCircularCollector {
	//logutil.Infoln(">>>>", "New CircularCollector")
	p := new(ProfileCircularCollector)
	p.conf = config.GetConfig()
	p.BUFFER_LEN = p.conf.TraceStepMaxCount

	p.buffer = make([]step.Step, p.BUFFER_LEN)

	p.loop = 0
	p.position = 0
	p.thisIndex = 0
	p.parentIndex = -1

	return p
}

func (this *ProfileCircularCollector) HasStep() bool {
	return this.position > 0
}

func (this *ProfileCircularCollector) Push(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.position >= this.BUFFER_LEN {
		this.loop++
		this.position = 0
	}

	st.SetIndex(this.thisIndex)
	st.SetParent(this.parentIndex)
	this.parentIndex = this.thisIndex
	this.buffer[this.position] = st
	this.position++
	this.thisIndex++
}

func (this *ProfileCircularCollector) Add(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.position >= this.BUFFER_LEN {
		this.loop++
		this.position = 0
	}

	st.SetIndex(this.thisIndex)
	st.SetParent(this.parentIndex)
	this.buffer[this.position] = st
	this.position++
	this.thisIndex++
}

func (this *ProfileCircularCollector) AddHeavy(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.position >= this.BUFFER_LEN {
		this.loop++
		this.position = 0
	}

	st.SetIndex(this.thisIndex)
	st.SetParent(this.parentIndex)
	this.buffer[this.position] = st
	this.position++
	this.thisIndex++
}

func (this *ProfileCircularCollector) JustAdd(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.position >= this.BUFFER_LEN {
		this.loop++
		this.position = 0
	}
	this.buffer[this.position] = st
	this.position++
	this.thisIndex++
}

func (this *ProfileCircularCollector) AddTail(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.position >= this.BUFFER_LEN {
		this.loop++
		this.position = 0
	}

	st.SetIndex(this.thisIndex)
	st.SetParent(this.parentIndex)
	this.buffer[this.position] = st
	this.position++
	this.thisIndex++

}

func (this *ProfileCircularCollector) Pop(st step.Step) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.parentIndex = st.GetParent()
}

func (this *ProfileCircularCollector) GetLastSteps(n int) []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.loop == 0 {
		cnt := this.position
		if this.position > n {
			cnt = n
		}
		buff := make([]step.Step, cnt)
		x := this.position - cnt
		for i := 0; i < cnt; i++ {
			buff[i] = this.buffer[x+i]
		}
		return buff
	}

	x := this.position - 1
	y := n - 1
	buff := make([]step.Step, n)
	for y >= 0 {
		if x < 0 {
			x = this.BUFFER_LEN - 1
		}
		buff[y] = this.buffer[x]
		y--
		x--
	}
	return buff
}

func (this *ProfileCircularCollector) GetSteps() []step.Step {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.loop == 0 {
		if this.position >= len(this.buffer) {
			return this.buffer
		}
		buff := make([]step.Step, 0)
		buff = append(buff, this.buffer[:this.position]...)
		return buff
	}
	if this.position == 0 {
		return this.buffer
	}

	buff := make([]step.Step, 0)
	buff = append(buff, this.buffer[this.position:this.position+this.BUFFER_LEN-this.position]...)
	buff = append(buff, this.buffer[:this.position]...)
	return buff
}

func (this *ProfileCircularCollector) ToBytes() []byte {
	return step.ToBytesStep(this.GetSteps())
}

func (this *ProfileCircularCollector) GetStep4Error() []step.Step {
	return this.GetLastSteps(5)
}

func (this *ProfileCircularCollector) IsReal() bool {
	return true
}

func (this *ProfileCircularCollector) Append(steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		this.Add(st)
	}
}

func (this *ProfileCircularCollector) AppendParentOffsetTime(parentOffsetTime int, steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		st.SetStartTime(st.GetStartTime() + int32(parentOffsetTime))
		this.Add(st)
	}
}

func (this *ProfileCircularCollector) GetSplitCount() int {
	return 0
}

func (this *ProfileCircularCollector) Length() int {
	return this.loop*this.BUFFER_LEN + this.position
}
