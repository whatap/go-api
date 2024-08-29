package logsink

import (
	"fmt"
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestAppendLine(t *testing.T) {
	bf := NewLineBuffer()
	ss := []string{"abc1", "abc2", "\tdef3", "\tdef4", "abc5", "abc6"}
	for i, s := range ss {
		v := bf.AppendLine(s)
		for j, tmp := range v {
			fmt.Println(i, "-", j, ", ", s, "=>", tmp)
		}
	}
}

func TestAppend(t *testing.T) {
	bf := NewLineBuffer()
	ss := []string{"abc1", "abc2", "\tdef3", "\tdef4", "abc5", "abc6\n"}
	for i, s := range ss {
		v := bf.Append(s)
		for j, tmp := range v {
			fmt.Println(i, "-", j, ", ", s, "=>", tmp)
		}
	}

	fmt.Println(bf.Flush())
}
