package active

import (
	"runtime"
	"sync"
	"unsafe"
)

// GoroutineInfo 는 goroutine 정보를 담는 구조체
type GoroutineInfo struct {
	ID    uint64
	State string
	Stack string
}

var stackPool = sync.Pool{
	New: func() interface{} { return make([]byte, 1<<20) },
}

// CollectGoroutineStacksCallback 은 모든 goroutine 스택을 수집하여 콜백으로 전달한다.
// 할당을 최소화하여 정기 수집에 최적화되어 있다.
// 주의: 콜백 내에서 전달받은 state, stack 문자열을 보관하려면 복사해야 한다.
func CollectGoroutineStacksCallback(fn func(id uint64, state, stack string)) {
	buf := captureAllStacks()
	original := buf

	for len(buf) > 0 {
		id, state, block, rest := parseNextBlock(buf)
		buf = rest
		if block != nil {
			fn(id, bytesToString(state), bytesToString(block))
		}
	}

	stackPool.Put(original[:cap(original)])
}

// CollectGoroutineStacks 는 모든 goroutine 스택을 map으로 반환한다.
// key는 goroutine ID이다.
func CollectGoroutineStacks() map[uint64]*GoroutineInfo {
	buf := captureAllStacks()
	original := buf

	est := countBlocks(buf)
	result := make(map[uint64]*GoroutineInfo, est)

	for len(buf) > 0 {
		id, state, block, rest := parseNextBlock(buf)
		buf = rest
		if block != nil {
			result[id] = &GoroutineInfo{
				ID:    id,
				State: string(state),
				Stack: string(block),
			}
		}
	}

	stackPool.Put(original[:cap(original)])
	return result
}

// CollectGoroutineStacksSlice 는 모든 goroutine 스택을 slice로 반환한다.
// map보다 약간 빠르며, 순회만 필요할 때 사용한다.
func CollectGoroutineStacksSlice() []*GoroutineInfo {
	buf := captureAllStacks()
	original := buf

	est := countBlocks(buf)
	result := make([]*GoroutineInfo, 0, est)

	for len(buf) > 0 {
		id, state, block, rest := parseNextBlock(buf)
		buf = rest
		if block != nil {
			result = append(result, &GoroutineInfo{
				ID:    id,
				State: string(state),
				Stack: string(block),
			})
		}
	}

	stackPool.Put(original[:cap(original)])
	return result
}

// captureAllStacks 는 모든 goroutine의 스택을 캡처한다.
func captureAllStacks() []byte {
	buf := stackPool.Get().([]byte)
	buf = buf[:cap(buf)]

	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, len(buf)*2)
	}
}

// countBlocks 는 버퍼 내 goroutine 블록 수를 추정한다.
func countBlocks(buf []byte) int {
	n := 1
	for i := 0; i < len(buf)-1; i++ {
		if buf[i] == '\n' && buf[i+1] == '\n' {
			n++
			i++
		}
	}
	return n
}

// parseNextBlock 는 버퍼에서 다음 goroutine 블록을 파싱한다.
func parseNextBlock(buf []byte) (id uint64, state, block, rest []byte) {
	// 블록 경계 찾기 ("\n\n")
	blockEnd := -1
	for i := 0; i < len(buf)-1; i++ {
		if buf[i] == '\n' && buf[i+1] == '\n' {
			blockEnd = i
			break
		}
	}

	if blockEnd == -1 {
		block = buf
		rest = nil
	} else {
		block = buf[:blockEnd]
		rest = buf[blockEnd+2:]
	}

	// trim leading whitespace
	for len(block) > 0 && block[0] <= ' ' {
		block = block[1:]
	}
	// trim trailing whitespace
	for len(block) > 0 && block[len(block)-1] <= ' ' {
		block = block[:len(block)-1]
	}

	// 최소 길이: "goroutine X [Y]:" = 15+
	if len(block) < 15 {
		return 0, nil, nil, rest
	}

	// "goroutine " 접두사 확인 (bounds check elimination)
	_ = block[9]
	if block[0] != 'g' || block[1] != 'o' || block[2] != 'r' ||
		block[3] != 'o' || block[4] != 'u' || block[5] != 't' ||
		block[6] != 'i' || block[7] != 'n' || block[8] != 'e' || block[9] != ' ' {
		return 0, nil, nil, rest
	}

	// goroutine ID 파싱
	i := 10
	for i < len(block) && block[i] >= '0' && block[i] <= '9' {
		id = id*10 + uint64(block[i]-'0')
		i++
	}
	if i == 10 {
		return 0, nil, nil, rest
	}

	// state 추출: " [running]:" -> "running"
	start := -1
	limit := i + 30
	if limit > len(block) {
		limit = len(block)
	}
	for j := i; j < limit; j++ {
		if block[j] == '[' {
			start = j + 1
			break
		}
	}
	if start != -1 {
		end := start
		for end < len(block) && block[end] != ']' {
			end++
		}
		state = block[start:end]
	}

	return id, state, block, rest
}

// bytesToString 은 []byte를 복사 없이 string으로 변환한다. (Go 1.18 호환)
// 주의: 원본 버퍼가 변경되면 반환된 string도 변경된다.
func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}
