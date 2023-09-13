package trace

import (
	"github.com/whatap/golib/lang/step"
)

type IProfileCollector interface {
	GetLastSteps(n int) []step.Step
	GetSteps() []step.Step
	Add(step step.Step)
	Push(step step.Step)
	HasStep() bool
	Pop(step step.Step)
	GetStep4Error() []step.Step
	AddHeavy(step step.Step)
	AddTail(step step.Step)
	IsReal() bool
	Append(steps []step.Step)
	AppendParentOffsetTime(parentOffsetTime int, steps []step.Step)
	GetSplitCount() int
}

func NewProfileCollector(mode int, ctx *TraceContext) IProfileCollector {
	var p IProfileCollector
	switch mode {
	// case 0:
	// 	this.profile = FakeCollector.instance;
	// 	break;
	case 1:
		p = NewProfileNormalCollector()
	case 2:
		p = NewProfileCircularCollector()
		break
	case 3:
		p = NewProfileLargeCollector(ctx)
	case 4:
		p = NewProfileSplitTxCollector(ctx)
	default:
		p = NewProfileNormalCollector()
	}
	return p
}
