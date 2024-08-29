package logsink

import (
	"fmt"
	"strings"

	"github.com/whatap/go-api/agent/agent/config"
	"github.com/whatap/go-api/agent/agent/data"
	"github.com/whatap/go-api/agent/agent/secure"
	"github.com/whatap/go-api/agent/logsink/zip"
	"github.com/whatap/go-api/agent/util/logutil"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/ansi"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/stringutil"
)

func Send(alog *LineLog) {
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

	p.Content = alog.Content

	if conf.HasLogSinkTags_ {
		tags := conf.LogSinkTags
		for _, v := range tags {
			if len(v) > 1 {
				p.Tags.PutString(v[0], v[1])
			}
		}
	}

	// 2024.07.31 java plugin WrLogSinkPack 에서 drop 설정이 있음. 다만 아직 쓰이지 않음.
	// if p.Dropped {
	// 	return
	// }

	if alog.Truncated {
		p.Tags.Put("@truncated", value.NewBoolValue(true))
		// LineLogUtil.log(alog)
		Log(alog)
	}

	if conf.LogSinkZipEnabled {
		zip.GetInstance().Add(p)
	} else {
		data.Send(p)
	}

	if conf.DebugLogSinkEnabled {
		sb := stringutil.NewStringBuffer()
		sb.Append("[")
		sb.Append(ansi.Green(alog.Category))
		sb.Append("|" + ansi.Yellow(fmt.Sprintf("%d", p.Line)))
		sb.Append("]")
		sb.Append(ansi.Yellow(p.Tags.String())).Append(" ")
		if conf.DebugLogSinkLineLimit > 0 && len(alog.Content) > int(conf.DebugLogSinkLineLimit) {
			sb.Append(alog.Content[0 : int(conf.DebugLogSinkLineLimit)-2])
		} else {
			sb.Append(alog.Content)
		}
		if !strings.HasSuffix(alog.Content, "\n") {
			sb.Append("\n")
		}
		logutil.Infoln(sb.ToString())
	}
}
