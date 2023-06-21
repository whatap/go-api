package countertag

import (
	"github.com/whatap/golib/lang/pack"
)

type Task interface {
	process(p *pack.TagCountPack)
}
