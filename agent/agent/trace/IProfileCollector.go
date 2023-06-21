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
	AddTail(step step.Step)
	IsReal() bool
	Append(steps []step.Step)
	AppendParent(parentOffsetTime int, steps []step.Step)
	GetSplitCount() int
}
