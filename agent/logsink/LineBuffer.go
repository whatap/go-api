package logsink

import (
	"bytes"
	"strings"
	"sync"

	"github.com/whatap/go-api/agent/agent/config"
)

type LineBuffer struct {
	isMulti bool
	out     bytes.Buffer
	empty   []string
	lock    sync.Mutex
}

func NewLineBuffer() *LineBuffer {
	p := new(LineBuffer)
	// public StringBuilder out = new StringBuilder(ConfLogSink.logsink_line_size + 128);
	p.empty = make([]string, 0)
	return p
}

func (this *LineBuffer) Append(line string) []string {
	this.lock.Lock()
	defer this.lock.Unlock()

	if line == "" {
		return this.empty
	}

	conf := config.GetConfig()

	if strings.HasSuffix(line, "\n") {
		if strings.HasPrefix(line, "\t") {
			if this.out.Len() > 0 && this.isMulti == false {
				old := this.out.String()
				this.out.WriteString(line)
				this.isMulti = true
				return []string{old}
			} else {
				this.out.WriteString(line)
				this.isMulti = true
				return this.empty
			}
		}
		this.isMulti = false
		if this.out.Len() > 0 {
			old := this.out.String()
			this.out.Reset()
			return []string{old, line}
		} else {
			return []string{line}

		}
	} else if this.out.Len() >= int(conf.LogSinkLineSize) {
		old := this.out.String()
		this.out.Reset()
		this.out.WriteString(line)
		return []string{old}
	} else {
		this.out.WriteString(line)
		return this.empty
	}
}

func (this *LineBuffer) AppendLine(line string) []string {
	this.lock.Lock()
	defer this.lock.Unlock()

	if line == "" {
		if this.out.Len() > 0 {
			old := this.out.String()
			this.out.Reset()
			return []string{old}
		}
		return this.empty
	}

	if strings.HasPrefix(line, "\t") {
		if this.out.Len() > 0 && this.isMulti == false {
			// old := this.out.String()
			this.out.WriteString(line)
			this.out.WriteString("\n")
			this.isMulti = true
			return []string{this.out.String()}
		} else {
			this.out.WriteString(line)
			this.out.WriteString("\n")
			this.isMulti = true
			return this.empty
		}
	}
	this.isMulti = false
	if this.out.Len() > 0 {
		old := this.out.String()
		this.out.Reset()
		return []string{old, line}
	} else {
		return []string{line}
	}
}

func (this *LineBuffer) Flush() string {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.out.Len() > 0 {
		old := this.out.String()
		this.out.Reset()
		return old
	}
	return ""
}
