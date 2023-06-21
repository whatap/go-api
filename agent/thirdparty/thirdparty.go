package thirdparty

import (
	"github.com/whatap/go-api/agent/thirdparty/nvidiasmi"
)

func StartAll() {
	nvidiasmi.InitNvidiaWatch()
}
