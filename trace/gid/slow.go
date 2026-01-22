//go:build !goid_noslow

package gid

import "runtime"

// slowProvider uses runtime.Stack() to parse goroutine ID.
// This is the fallback method, always available but slower.
type slowProvider struct{}

func (p *slowProvider) Get() int64 {
	return getGIDSlow()
}

func (p *slowProvider) Name() string {
	return "slow (runtime.Stack)"
}

func (p *slowProvider) Available() bool {
	return true
}

func getSlowProvider() Provider {
	return &slowProvider{}
}

// getGIDSlow extracts goroutine ID from runtime.Stack output.
// Format: "goroutine 123 [running]:\n..."
func getGIDSlow() int64 {
	var buf [32]byte
	runtime.Stack(buf[:], false)

	// Skip "goroutine " prefix (10 bytes)
	var gid int64
	for i := 10; i < 32; i++ {
		if buf[i] == ' ' {
			break
		}
		gid = gid*10 + int64(buf[i]-'0')
	}
	return gid
}
