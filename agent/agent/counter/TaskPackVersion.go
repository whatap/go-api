package counter

import (
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/util/logutil"
)

// 나중에 제거해야함...
//

type TaskPackVersion struct {
}

func NewTaskPackVersion() *TaskPackVersion {
	p := new(TaskPackVersion)
	return p
}
func (this *TaskPackVersion) process(p *pack.CounterPack1) {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA400", "process Recover", r)
		}
	}()

	conf := config.GetConfig()

	// 일단 서버 개발 완료 될때까지.
	p.Version = conf.CounterVersion
	if p.Version == 0 {
		p.HeapPerm /= 1024
		p.HeapUse /= 1024
		p.HeapTot /= 1024
	}

	// v2.0_02 - 2020.03.08
	// p.version==2 이면
	// p.service_satisfied, p.service_tolerated
	// 수집됨
}
