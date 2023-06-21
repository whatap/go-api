package sys

import (
	"syscall"
)

func GetPid() int {
	return syscall.Getpid()
}
