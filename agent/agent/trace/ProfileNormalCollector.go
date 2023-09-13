package trace

import (
	"sort"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
	// "github.com/whatap/go-api/agent/util/logutil"
	"github.com/whatap/golib/lang/step"
)

type ProfileNormalCollector struct {
	buffer       []step.Step
	position     int32
	parent_index int32

	buffer_len int32
	normal_len int32
	heavy_len  int32
	heavy_time int32
	mutex      sync.Mutex
}

func NewProfileNormalCollector() *ProfileNormalCollector {
	//logutil.Infoln(">>>>", "New ProfileNormalCollector")
	p := new(ProfileNormalCollector)

	conf := config.GetConfig()
	p.buffer = make([]step.Step, 0, conf.ProfileStepMaxCount)
	p.position = 0
	p.parent_index = -1
	p.buffer_len = conf.ProfileStepMaxCount
	p.normal_len = conf.ProfileStepNormalCount
	p.heavy_len = conf.ProfileStepHeavyCount
	p.heavy_time = conf.ProfileStepHeavyTime

	return p
}

func (this *ProfileNormalCollector) HasStep() bool {
	return (this.position > 0)
}

func (this *ProfileNormalCollector) Add(st step.Step) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.position < this.normal_len {
		st.SetIndex(this.position)
		st.SetParent(this.parent_index)
		//this.buffer[this.position] = st
		this.buffer = append(this.buffer, st)
		this.position++
		//	}
		// Add 에 heavy_len 로직 추가
		// Java에서 스텝 시작, 스텝 종료로 나누어서 push, pop 처리하는 부분이 없고, 모두 스텝 종료시에 Add로만 처리하기 때문에 heavy_len 처리 로직 추가.
	} else if this.position < this.heavy_len {
		if st.GetElapsed() >= this.heavy_time {
			st.SetIndex(this.position)
			st.SetParent(this.parent_index)
			//this.buffer[this.position] = st
			this.buffer = append(this.buffer, st)
			this.position++
		}
	}
}

func (this *ProfileNormalCollector) AddHeavy(st step.Step) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.position < this.buffer_len {
		st.SetIndex(this.position)
		st.SetParent(this.parent_index)
		//this.buffer[this.position] = st
		this.buffer = append(this.buffer, st)
		this.position++
	}
}

func (this *ProfileNormalCollector) GetSteps__() []step.Step {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	sort.Slice(this.buffer, func(i, j int) bool {
		return this.buffer[i].GetStartTime() < this.buffer[j].GetStartTime()
	})
	// index 다시 정렬
	for i, it := range this.buffer {
		it.SetIndex(int32(i))
	}

	if this.position >= this.buffer_len {
		return this.buffer
	} else {
		return this.buffer[0:this.position]
	}
}

func (this *ProfileNormalCollector) GetSteps() []step.Step {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	tmp := make([]step.Step, 0)
	if this.position >= this.buffer_len {
		tmp = append(tmp, this.buffer...)
	} else {
		tmp = append(tmp, this.buffer[0:this.position]...)
	}

	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].GetStartTime() < tmp[j].GetStartTime()
	})
	// index 다시 정렬
	for i, it := range tmp {
		it.SetIndex(int32(i))
	}

	return tmp
}

func (this *ProfileNormalCollector) GetLastSteps(n int) []step.Step {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	cnt := int(this.position)
	if int(this.position) > n {
		cnt = n
	}

	buff := make([]step.Step, cnt)
	x := int(this.position) - cnt
	for i := 0; i < cnt; i++ {
		buff[i] = this.buffer[x+i]
	}
	return buff
}

func (this *ProfileNormalCollector) GetStep4Error() []step.Step {
	return this.GetLastSteps(5)
}

// TODO st.SetDrop 필요
func (this *ProfileNormalCollector) Push(st step.Step) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.position < this.normal_len {
		st.SetIndex(this.position)
		st.SetParent(this.parent_index)
		this.parent_index = this.position
		this.buffer[this.position] = st
		this.position++
	} else {
		st.SetDrop(true)
	}
}

// TODO st.GetElapsed , st.GetDrop 필요
func (this *ProfileNormalCollector) Pop(st step.Step) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if st.GetDrop() {
		if this.position < this.heavy_len {
			if st.GetElapsed() >= this.heavy_time {
				st.SetIndex(this.position)
				st.SetParent(this.parent_index)
				//this.buffer[this.position] = st
				this.buffer = append(this.buffer, st)
				this.position++
			}
		}
	} else {
		this.parent_index = st.GetParent()
	}
}

func (this *ProfileNormalCollector) AddTail(st step.Step) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if int(this.position) < cap(this.buffer) {
		st.SetIndex(this.position)
		st.SetParent(this.parent_index)
		//this.buffer[this.position] = st
		this.buffer = append(this.buffer, st)
		this.position++
	}
}

func (this *ProfileNormalCollector) IsReal() bool {
	return true
}

func (this *ProfileNormalCollector) Append(steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		this.Add(st)
	}
}

func (this *ProfileNormalCollector) AppendParentOffsetTime(parentOffsetTime int, steps []step.Step) {
	if steps == nil {
		return
	}
	for _, st := range steps {
		st.SetStartTime(int32(parentOffsetTime))
		this.Add(st)
	}
}

func (this *ProfileNormalCollector) GetSplitCount() int {
	return 0
}
func (this *ProfileNormalCollector) Length() int {
	return int(this.position)
}
