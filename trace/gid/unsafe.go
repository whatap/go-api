//go:build goid_unsafe

package gid

import (
	"runtime"
	"sync"
	"unsafe"

	_ "unsafe" // required for go:linkname
)

//go:linkname getg runtime.getg
func getg() unsafe.Pointer

// unsafeProvider uses go:linkname + automatic offset detection.
// This is the fastest method but may break with Go version updates.
type unsafeProvider struct {
	offset   uintptr
	verified bool
}

var (
	unsafeProviderInstance *unsafeProvider
	unsafeProviderOnce     sync.Once
)

func (p *unsafeProvider) Get() int64 {
	g := getg()
	return *(*int64)(unsafe.Pointer(uintptr(g) + p.offset))
}

func (p *unsafeProvider) Name() string {
	return "unsafe (getg)"
}

func (p *unsafeProvider) Available() bool {
	return p.verified
}

func getUnsafeProvider() Provider {
	unsafeProviderOnce.Do(func() {
		p := &unsafeProvider{}
		if p.detectOffset() {
			p.verified = true
			unsafeProviderInstance = p
		}
	})
	return unsafeProviderInstance
}

// detectOffset automatically detects the goid field offset in runtime.g.
// It searches for the goroutine ID by comparing with runtime.Stack output.
func (p *unsafeProvider) detectOffset() bool {
	g := getg()
	if g == nil {
		return false
	}

	// Get expected GID from runtime.Stack (the reference)
	expected := getGIDFromStack()
	if expected <= 0 {
		return false
	}

	// Search for goid field in g struct
	// goid is typically at offset 152 (Go 1.18-1.21) or nearby
	// We search a reasonable range to be version-agnostic
	for offset := uintptr(0); offset < 256; offset += 8 {
		val := *(*int64)(unsafe.Pointer(uintptr(g) + offset))
		if val == expected {
			p.offset = offset
			return true
		}
	}

	return false
}

// getGIDFromStack extracts goroutine ID from runtime.Stack output.
func getGIDFromStack() int64 {
	var buf [32]byte
	runtime.Stack(buf[:], false)
	var gid int64
	for i := 10; i < 32; i++ {
		if buf[i] == ' ' {
			break
		}
		gid = gid*10 + int64(buf[i]-'0')
	}
	return gid
}
