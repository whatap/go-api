package counter

import "os"

// llmPid — process 시작 시 1회 캡처. LLM 메트릭의 5번째 공통 태그 ("pid").
// python-apm `whatap/llm/stats/base_stat.py:9` 의 `currentpid = os.getpid()` 와 동일 패턴.
var llmPid = int64(os.Getpid())
