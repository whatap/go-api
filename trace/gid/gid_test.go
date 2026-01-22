package gid

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/petermattis/goid"
)

// ============ Provider Implementations for Benchmarking ============

// Old implementation (64 bytes + string conversion)
func getGIDOld() int64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return int64(n)
}

// sync.Pool version
var gidBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64)
		return &buf
	},
}

func getGIDPooled() int64 {
	bp := gidBufPool.Get().(*[]byte)
	b := *bp
	b = b[:runtime.Stack(b, false)]
	b = b[10:] // skip "goroutine "

	var n int64
	for _, c := range b {
		if c == ' ' {
			break
		}
		n = n*10 + int64(c-'0')
	}

	gidBufPool.Put(bp)
	return n
}

// Direct parsing version (no string conversion)
func getGIDDirect() int64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = b[10:]

	var n int64
	for _, c := range b {
		if c == ' ' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// petermattis/goid library (direct call for comparison)
func getGIDPeter() int64 {
	return goid.Get()
}

// ============ Single-Thread Benchmarks ============

func BenchmarkGetGID_Exported(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GetGID()
	}
}

func BenchmarkGetGID_Peter(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getGIDPeter()
	}
}

func BenchmarkGetGID_Slow(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getGIDSlow()
	}
}

func BenchmarkGetGID_Old(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getGIDOld()
	}
}

func BenchmarkGetGID_Pooled(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getGIDPooled()
	}
}

func BenchmarkGetGID_Direct(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getGIDDirect()
	}
}

// ============ Parallel Benchmarks ============

func BenchmarkGetGID_Exported_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GetGID()
		}
	})
}

func BenchmarkGetGID_Peter_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = getGIDPeter()
		}
	})
}

func BenchmarkGetGID_Slow_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = getGIDSlow()
		}
	})
}

func BenchmarkGetGID_Old_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = getGIDOld()
		}
	})
}

// ============ Correctness Tests ============

func TestGetGID_AllVersions(t *testing.T) {
	exported := GetGID()
	peter := getGIDPeter()
	slow := getGIDSlow()
	old := getGIDOld()
	pooled := getGIDPooled()
	direct := getGIDDirect()

	if exported != peter {
		t.Errorf("Peter mismatch: exported=%d, peter=%d", exported, peter)
	}
	if exported != slow {
		t.Errorf("Slow mismatch: exported=%d, slow=%d", exported, slow)
	}
	if exported != old {
		t.Errorf("Old mismatch: exported=%d, old=%d", exported, old)
	}
	if exported != pooled {
		t.Errorf("Pooled mismatch: exported=%d, pooled=%d", exported, pooled)
	}
	if exported != direct {
		t.Errorf("Direct mismatch: exported=%d, direct=%d", exported, direct)
	}

	t.Logf("GID: %d (all versions match)", exported)
	t.Logf("Active provider: %s", GetProviderName())
}

func TestGetGID_MultipleGoroutines(t *testing.T) {
	const numGoroutines = 100
	results := make(chan struct {
		exported int64
		peter    int64
		slow     int64
	}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			results <- struct {
				exported int64
				peter    int64
				slow     int64
			}{
				exported: GetGID(),
				peter:    getGIDPeter(),
				slow:     getGIDSlow(),
			}
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		r := <-results
		if r.exported != r.peter || r.exported != r.slow {
			t.Errorf("Mismatch in goroutine: exported=%d, peter=%d, slow=%d",
				r.exported, r.peter, r.slow)
		}
	}
}

func TestGetProviderName(t *testing.T) {
	name := GetProviderName()
	t.Logf("Active GID provider: %s", name)

	// Should be one of the known providers
	validNames := []string{"petermattis/goid", "slow (runtime.Stack)", "unsafe (getg)"}
	found := false
	for _, valid := range validNames {
		if name == valid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Unknown provider name: %s", name)
	}
}
