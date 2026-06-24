package logsink

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/logsink/zip"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/ansi"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/stringutil"
)

// §229: originalStdout 은 logsink 패키지 init 시점에 os.Stdout 값을 복사해 둔다.
// 이후 agent/logsink/std.StdOutSenderThread 가 os.Stdout 을 ProxyStream 으로 교체해도
// originalStdout 은 원본 *os.File(fd 1) 을 계속 가리키므로, DebugLogSinkEnabled 디버그
// 출력이 ProxyStream → logsink.Send 를 거쳐 자기 자신을 재캡처하는 무한 재귀(§229) 를 차단한다.
var originalStdout = os.Stdout

func Send(alog *LineLog) {
	p := buildLogSinkPack(alog)
	conf := config.GetConfig()

	if alog.Truncated {
		// LineLogUtil.log(alog)
		Log(alog)
	}

	// 2024.07.31 java plugin WrLogSinkPack 에서 drop 설정이 있음. 다만 아직 쓰이지 않음.
	// if p.Dropped {
	// 	return
	// }

	if conf.LogSinkZipEnabled {
		zip.GetInstance().Add(p)
	} else {
		data.Send(p)
	}

	if conf.DebugLogSinkEnabled {
		writeDebugLogSink(alog, p.Line, p.Tags.String(), conf)
	}
}

// buildLogSinkPack — LineLog → LogSinkPack 변환만 수행 (data.Send / zip 미호출).
// 단위 테스트가 외부 글로벌 의존성 (data, zip) 없이 Tags/Fields/Content 전파를 검증할 수 있도록 분리.
func buildLogSinkPack(alog *LineLog) *pack.LogSinkPack {
	p := pack.NewLogSinkPack()

	p.Time = alog.Time
	if p.Time <= 0 {
		p.Time = dateutil.SystemNow()
	}
	p.Category = alog.Category

	secu := secure.GetSecurityMaster()
	conf := config.GetConfig()

	if secu.ONAME != "" {
		p.Tags.PutString("oname", secu.ONAME)
	}
	if conf.OKIND != 0 {
		p.Tags.PutString("okindName", conf.OKIND_NAME)
	}
	if conf.ONODE != 0 {
		p.Tags.PutString("onodeName", conf.ONODE_NAME)
	}

	if alog.Fields != nil && alog.Fields.Size() > 0 {
		if p.Fields == nil {
			p.Fields = alog.Fields
		} else {
			p.Fields.PutAll(alog.Fields)
		}
	}

	// §260: LineLog.Tags 도 LogSinkPack.Tags 로 전파.
	// 누락 시 LLM 의 @txid / @step_id / provider / url / operation_type / model /
	// finish_reason / llm_log_type 등 모든 메타 태그가 서버에 도달하지 않음.
	if alog.Tags != nil && alog.Tags.Size() > 0 {
		p.Tags.PutAll(alog.Tags)
	}

	p.Content = alog.Content

	if conf.HasLogSinkTags_ {
		tags := conf.LogSinkTags
		for _, v := range tags {
			if len(v) > 1 {
				p.Tags.PutString(v[0], v[1])
			}
		}
	}

	if alog.Truncated {
		p.Tags.Put("@truncated", value.NewBoolValue(true))
	}

	return p
}

// writeDebugLogSink 는 §229 무한 재귀를 막기 위해 debug 출력을 caller 측에서 분리.
// - Stdout/Stderr 캡처 카테고리 (ProxyStream → Send → 여기) 는 건너뛴다 (self-category 필터).
// - logutil 을 거치지 않고 originalStdout(fd 1 원본 *os.File) 에 직접 쓴다.
//   logutil.Infoln 은 내부 log.Logger 가 교체된 os.Stdout(ProxyStream) 을 참조할 수 있어 위험.
func writeDebugLogSink(alog *LineLog, pLine int64, pTags string, conf *config.Config) {
	if alog.Category == conf.LogSinkCategoryStdOut || alog.Category == conf.LogSinkCategoryStdErr {
		return
	}
	sb := stringutil.NewStringBuffer()
	sb.Append("[")
	sb.Append(ansi.Green(alog.Category))
	sb.Append("|" + ansi.Yellow(fmt.Sprintf("%d", pLine)))
	sb.Append("]")
	sb.Append(ansi.Yellow(pTags)).Append(" ")
	if conf.DebugLogSinkLineLimit > 0 && len(alog.Content) > int(conf.DebugLogSinkLineLimit) {
		sb.Append(alog.Content[0 : int(conf.DebugLogSinkLineLimit)-2])
	} else {
		sb.Append(alog.Content)
	}
	if !strings.HasSuffix(alog.Content, "\n") {
		sb.Append("\n")
	}
	fmt.Fprintf(originalStdout, "%s [WA-LOGSINK] %s", time.Now().Format("2006/01/02 15:04:05"), sb.ToString())
}
