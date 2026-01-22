package active

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// 테스트용 goroutine 생성
func spawnGoroutines(n int) func() {
	var wg sync.WaitGroup
	done := make(chan struct{})
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			wg.Done()
			<-done
		}()
	}

	wg.Wait()
	time.Sleep(10 * time.Millisecond) // goroutine 안정화

	return func() { close(done) }
}

// ============ Callback 벤치마크 ============

func BenchmarkCollectGoroutineStacksCallback_10(b *testing.B) {
	cleanup := spawnGoroutines(10)
	defer cleanup()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		CollectGoroutineStacksCallback(func(id uint64, state, stack string) {
			_ = id
		})
	}
}

func BenchmarkCollectGoroutineStacksCallback_100(b *testing.B) {
	cleanup := spawnGoroutines(100)
	defer cleanup()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		CollectGoroutineStacksCallback(func(id uint64, state, stack string) {
			_ = id
		})
	}
}

// ============ Map 벤치마크 ============

func BenchmarkCollectGoroutineStacks_100(b *testing.B) {
	cleanup := spawnGoroutines(100)
	defer cleanup()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := CollectGoroutineStacks()
		_ = m
	}
}

// ============ Slice 벤치마크 ============

func BenchmarkCollectGoroutineStacksSlice_100(b *testing.B) {
	cleanup := spawnGoroutines(100)
	defer cleanup()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := CollectGoroutineStacksSlice()
		_ = s
	}
}

// ============ 메모리 할당 벤치마크 ============

func BenchmarkCollectGoroutineStacksCallback_100_Alloc(b *testing.B) {
	cleanup := spawnGoroutines(100)
	defer cleanup()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		CollectGoroutineStacksCallback(func(id uint64, state, stack string) {
			_ = id
		})
	}
}

// ============ 정확성 테스트 ============

func TestStackCollection_Correctness(t *testing.T) {
	cleanup := spawnGoroutines(50)
	defer cleanup()

	// 현재 goroutine 수 확인
	numGoroutines := runtime.NumGoroutine()
	t.Logf("Expected goroutines: ~%d", numGoroutines)

	// Callback 테스트
	var callbackCount int
	CollectGoroutineStacksCallback(func(id uint64, state, stack string) {
		callbackCount++
	})
	t.Logf("Callback collected: %d", callbackCount)

	// Map 테스트
	mapResult := CollectGoroutineStacks()
	t.Logf("Map collected: %d", len(mapResult))

	// Slice 테스트
	sliceResult := CollectGoroutineStacksSlice()
	t.Logf("Slice collected: %d", len(sliceResult))

	// 결과 비교
	if callbackCount != len(mapResult) || callbackCount != len(sliceResult) {
		t.Errorf("Counts differ: callback=%d, map=%d, slice=%d",
			callbackCount, len(mapResult), len(sliceResult))
	}
}
