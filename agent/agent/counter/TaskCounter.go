package counter

import (
	"github.com/whatap/golib/lang/pack"
)

// 나중에 제거해야함...
//

type TaskCounter struct {
}

func (this *TaskCounter) process(p *pack.CounterPack1) {

	p.Cpu = 20.5
	//	p.Tps = 100.1
	//	p.RespTime = 100
	//	p.ActSvcSlice = []int16{10, 20, 30}

}
