package logsink

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/golib/lang/value"
)

// §229 회귀 테스트 — WA-LOGSINK 디버그 출력이 교체된 os.Stdout 을 거치지 않고
// originalStdout 으로 직접 나가는지, Stdout/Stderr 카테고리는 self-filter 로 걸러지는지 검증.
//
// 배경: debug_logsink_enabled=true 설정 시 SendfUtil 디버그 블록이 logutil.Infoln 을 호출했는데,
// logutil 은 내부 log.Logger 가 StdOutSenderThread.reset() 이 설치한 ProxyStream 기반 os.Stdout 을
// 참조하는 경로가 생겨, ProxyStream → logsink.Send → 디버그 블록 → logutil.Infoln → ProxyStream 으로
// 무한 재귀가 발생했다 (§229, 60초 만에 58GB 로그). 아래 테스트가 그 재발을 막는다.

// withSwappedStdout 은 테스트 중 originalStdout 과 os.Stdout 을 임시로 교체하고
// 종료 시 복원하는 헬퍼. originalStdout 은 패키지 변수이므로 같은 패키지에서 접근 가능.
func withSwappedStdout(t *testing.T, realPipe *os.File, proxyPipe *os.File, fn func()) {
	t.Helper()
	origOriginal := originalStdout
	origStdout := os.Stdout
	originalStdout = realPipe
	os.Stdout = proxyPipe
	defer func() {
		originalStdout = origOriginal
		os.Stdout = origStdout
	}()
	fn()
}

func newTestConf() *config.Config {
	conf := &config.Config{}
	conf.LogSinkCategoryStdOut = "AppStdOut"
	conf.LogSinkCategoryStdErr = "AppStdErr"
	conf.DebugLogSinkEnabled = true
	conf.DebugLogSinkLineLimit = 0
	return conf
}

func newTestLineLog(category, content string) *LineLog {
	return &LineLog{
		Category: category,
		Tags:     value.NewMapValue(),
		Fields:   value.NewMapValue(),
		Content:  content,
	}
}

// §229: 일반 카테고리 로그는 originalStdout 으로 출력되고, 교체된 os.Stdout(ProxyStream 역할)
// 에는 아무 것도 쓰여선 안 된다 — 재귀 차단의 핵심.
func TestWriteDebugLogSink_BypassProxyStdout(t *testing.T) {
	realR, realW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe real: %v", err)
	}
	defer realR.Close()
	defer realW.Close()

	proxyR, proxyW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe proxy: %v", err)
	}
	defer proxyR.Close()
	defer proxyW.Close()

	conf := newTestConf()
	alog := newTestLineLog("MyApp", "hello world\n")

	withSwappedStdout(t, realW, proxyW, func() {
		writeDebugLogSink(alog, 1, "{}", conf)
	})

	// 실제 원본 stdout 쪽에는 WA-LOGSINK 출력이 있어야 한다.
	realW.Close()
	realOut, _ := io.ReadAll(realR)
	if !strings.Contains(string(realOut), "[WA-LOGSINK]") {
		t.Fatalf("originalStdout 에 WA-LOGSINK 출력 없음: %q", string(realOut))
	}
	if !strings.Contains(string(realOut), "hello world") {
		t.Fatalf("originalStdout 에 content 없음: %q", string(realOut))
	}

	// 교체된 os.Stdout(ProxyStream 대역) 쪽에는 아무 것도 쓰이면 안 된다 — 재귀 차단 근거.
	proxyW.Close()
	proxyOut, _ := io.ReadAll(proxyR)
	if len(proxyOut) != 0 {
		t.Fatalf("proxy stdout 에 출력이 흘러나옴 (§229 재귀 위험): %q", string(proxyOut))
	}
}

// §229: AppStdOut 카테고리(=logsink_category_stdout) 로그는 debug 재출력 대상에서 제외.
// ProxyStream 이 캡처한 내용을 여기서 다시 찍으면 화면 2중 출력.
func TestWriteDebugLogSink_SkipsStdoutCategory(t *testing.T) {
	realR, realW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer realR.Close()
	defer realW.Close()

	conf := newTestConf()
	alog := newTestLineLog(conf.LogSinkCategoryStdOut, "captured from stdout\n")

	withSwappedStdout(t, realW, os.Stdout, func() {
		writeDebugLogSink(alog, 1, "{}", conf)
	})

	realW.Close()
	out, _ := io.ReadAll(realR)
	if len(out) != 0 {
		t.Fatalf("Stdout 카테고리는 skip 되어야 하지만 출력됨: %q", string(out))
	}
}

func TestWriteDebugLogSink_SkipsStderrCategory(t *testing.T) {
	realR, realW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer realR.Close()
	defer realW.Close()

	conf := newTestConf()
	alog := newTestLineLog(conf.LogSinkCategoryStdErr, "captured from stderr\n")

	withSwappedStdout(t, realW, os.Stdout, func() {
		writeDebugLogSink(alog, 1, "{}", conf)
	})

	realW.Close()
	out, _ := io.ReadAll(realR)
	if len(out) != 0 {
		t.Fatalf("Stderr 카테고리는 skip 되어야 하지만 출력됨: %q", string(out))
	}
}

// §229: originalStdout 은 패키지 init 시점에 os.Stdout 을 복사해 두어야 한다.
// 테스트 실행 시점에 os.Stdout 이 교체되어 있을 수 있으므로 값 동일성을 요구하지는 않지만,
// nil 이 아니어야 하고 init 이후 writeDebugLogSink 가 의존할 수 있어야 한다.
func TestOriginalStdout_Initialized(t *testing.T) {
	if originalStdout == nil {
		t.Fatal("originalStdout 가 nil — 패키지 init 에서 캡처되지 않음")
	}
}

// §260: buildLogSinkPack 이 LineLog.Tags 를 LogSinkPack.Tags 로 PutAll 해야 한다.
// 누락 시 LLM 의 @txid / @step_id / provider / url / operation_type / model /
// finish_reason / llm_log_type 등 모든 메타 태그가 서버에 도달하지 않는다 (§260 본질).
//
// 이전에는 alog.Fields 만 복사하고 alog.Tags 는 무시했음. 회귀 시 Tags 컬럼이 raw dump 에서
// 모두 사라지므로, Fields 는 전달되는데 Tags 만 빠지는 비대칭 회귀 패턴이 된다.
func TestBuildLogSinkPack_PropagatesLineLogTags(t *testing.T) {
	alog := &LineLog{
		Category: "#LlmCallLog",
		Tags:     value.NewMapValue(),
		Fields:   value.NewMapValue(),
		Content:  "Mock answer: 42",
	}
	// LLM putBaseTags + buildStepStatusLineLog 가 set 하는 키들 — 회귀 시 모두 누락
	alog.Tags.PutString("llm_log_type", "step_status")
	alog.Tags.PutString("@txid", "4594636587138222968")
	alog.Tags.PutString("@step_id", "4607758727229381156")
	alog.Tags.PutString("provider", "api.openai.com")
	alog.Tags.PutString("model", "gpt-4o")
	alog.Tags.PutString("operation_type", "chat")
	alog.Tags.PutString("finish_reason", "stop")
	alog.Fields.PutLong("input_tokens.n", 50)
	alog.Fields.PutLong("output_tokens.n", 10)

	p := buildLogSinkPack(alog)

	if p == nil {
		t.Fatal("buildLogSinkPack returned nil")
	}
	if p.Category != "#LlmCallLog" {
		t.Fatalf("Category mismatch: got %q", p.Category)
	}
	if p.Content != "Mock answer: 42" {
		t.Fatalf("Content mismatch: got %q", p.Content)
	}

	expectedTags := map[string]string{
		"llm_log_type":   "step_status",
		"@txid":          "4594636587138222968",
		"@step_id":       "4607758727229381156",
		"provider":       "api.openai.com",
		"model":          "gpt-4o",
		"operation_type": "chat",
		"finish_reason":  "stop",
	}
	for k, want := range expectedTags {
		got := p.Tags.GetString(k)
		if got != want {
			t.Errorf("Tags[%q] = %q, want %q (§260: alog.Tags PutAll 누락 회귀)", k, got, want)
		}
	}

	if p.Fields.GetLong("input_tokens.n") != 50 {
		t.Errorf("Fields[input_tokens.n] mismatch")
	}
	if p.Fields.GetLong("output_tokens.n") != 10 {
		t.Errorf("Fields[output_tokens.n] mismatch")
	}
}

// §260: 빈 Tags / nil Tags 인 경우는 panic 없이 통과해야 한다 (§229 stdout/stderr ProxyStream 사용처 호환).
func TestBuildLogSinkPack_EmptyTagsSafe(t *testing.T) {
	t.Run("empty Tags", func(t *testing.T) {
		alog := &LineLog{
			Category: "AppStdOut",
			Tags:     value.NewMapValue(),
			Fields:   value.NewMapValue(),
			Content:  "captured stdout line",
		}
		p := buildLogSinkPack(alog)
		if p == nil {
			t.Fatal("buildLogSinkPack returned nil for empty Tags")
		}
		if p.Content != "captured stdout line" {
			t.Errorf("Content mismatch")
		}
	})
	t.Run("nil Tags", func(t *testing.T) {
		alog := &LineLog{
			Category: "AppStdErr",
			Tags:     nil,
			Fields:   nil,
			Content:  "captured stderr line",
		}
		p := buildLogSinkPack(alog)
		if p == nil {
			t.Fatal("buildLogSinkPack returned nil for nil Tags")
		}
	})
}
