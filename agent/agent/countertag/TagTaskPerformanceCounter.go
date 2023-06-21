//go:build !windows
// +build !windows

package countertag

import (
	//"log"

	"github.com/whatap/golib/lang/pack"
)

type TagTaskPerformanceCounter struct {
}

func NewTagTaskPerformanceCounter() *TagTaskPerformanceCounter {
	p := new(TagTaskPerformanceCounter)
	return p
}

func (this *TagTaskPerformanceCounter) process(p *pack.TagCountPack) {
}
