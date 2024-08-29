package logsink

import (
	"fmt"

	"github.com/whatap/golib/lang/value"
)

type LineLog struct {
	Truncated        bool
	OrgContentLength int
	Time             int64
	Category         string
	Tags             *value.MapValue
	Fields           *value.MapValue
	Content          string
}

func NewLineLog() *LineLog {
	p := new(LineLog)
	p.Tags = value.NewMapValue()
	p.Fields = value.NewMapValue()
	return p
}

func (this *LineLog) String() string {
	return fmt.Sprint("LineLog [time=%d, category=%s, tags=%s, fields=%s, content=%s]", this.Time, this.Category, this.Tags, this.Fields, this.Content)
}
