package whatapfasthttp

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
	"github.com/whatap/go-api/config"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/lang/pack/udp"
	whatapnet "github.com/whatap/golib/net"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/stringutil"
)

func HeaderToMap(header *fasthttp.RequestHeader) map[string][]string {
	ret := map[string][]string{}

	header.VisitAll(func(keyRaw []byte, valueRaw []byte) {
		key, val := string(keyRaw), string(valueRaw)
		ret[key] = []string{val}
	},
	)
	return ret
}

func StartWithFastHttpRequest(r *fasthttp.RequestCtx) (context.Context, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return r, nil
	}

	udpClient := whatapnet.GetUdpClient()
	_, traceCtx := trace.NewTraceContext(r)
	r.SetUserValue("whatap", traceCtx)

	traceCtx.Name = string(r.RequestURI())
	traceCtx.Host = string(r.Host())
	traceCtx.StartTime = dateutil.SystemNow()
	// update multi trace info
	UpdateFastHttpMtrace(traceCtx, r.Request.Header)

	if pack := udp.CreatePack(udp.TX_START, udp.UDP_PACK_VERSION); pack != nil {
		p := pack.(*udp.UdpTxStartPack)

		p.Txid = traceCtx.Txid
		p.Time = traceCtx.StartTime
		p.Host = string(r.Host())
		p.Uri = string(r.RequestURI())
		p.Ipaddr = r.RemoteAddr().String()
		p.HttpMethod = string(r.Method())
		p.Ref = string(r.Referer())
		p.UAgent = string(r.UserAgent())

		udpClient.Send(p)
	}
	SetFastHttpHeader(r, &r.Request.Header)

	return r, nil
}

func SetFastHttpHeader(ctx context.Context, header *fasthttp.RequestHeader) {
	conf := config.GetConfig()
	if !conf.ProfileHttpHeaderEnabled {
		return
	}
	udpClient := whatapnet.GetUdpClient()
	if _, traceCtx := trace.GetTraceContext(ctx); traceCtx != nil {
		if strings.HasPrefix(traceCtx.Name, conf.ProfileHttpHeaderUrlPrefix) {
			if pack := udp.CreatePack(udp.TX_MSG, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxMessagePack)
				p.Txid = traceCtx.Txid
				p.Time = dateutil.SystemNow()
				p.Hash = "HTTP-HEADERS"
				p.SetHeader(HeaderToMap(header))
				if conf.Debug {
					log.Println("[WA-TX-06001] txid:", traceCtx.Txid, ", uri: ", traceCtx.Name, "\n headers: ", p.Desc)
				}
				udpClient.Send(p)
			}
		}
	}
}

func UpdateFastHttpMtrace(traceCtx *trace.TraceCtx, header fasthttp.RequestHeader) {
	conf := config.GetConfig()
	if !conf.MtraceEnabled {
		return
	}

	header.VisitAll(func(keyRaw, valueRaw []byte) {
		if len(valueRaw) <= 0 {
			return
		}

		key, val := string(keyRaw), string(valueRaw)

		v := strings.TrimSpace(val)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case conf.TraceMtraceCallerKey:
			arr := stringutil.Split(v, ",")
			if len(arr) >= 3 {
				traceCtx.MTid = hexa32.ToLong32(arr[0])

				if val, err := strconv.Atoi(arr[1]); err == nil {
					traceCtx.MDepth = int32(val)
				}
				traceCtx.MCallerTxid = hexa32.ToLong32(arr[2])
			}
		case conf.TraceMtraceCalleeKey:
			traceCtx.MCallee = hexa32.ToLong32(v)
			if traceCtx.MCallee != 0 {
				traceCtx.Txid = traceCtx.MCallee
			}

		case conf.TraceMtraceSpecKey1:
			arr := stringutil.Split(v, ",")
			if len(arr) >= 2 {
				traceCtx.MCallerSpec = arr[0]
				traceCtx.MCallerUrl = arr[1]
			}
		case conf.TraceMtracePoidKey:
			traceCtx.MCallerPoidKey = v
		}
	},
	)

	if traceCtx.MTid == 0 {
		checkSeq := keygen.Next()
		if int32(math.Abs(float64(checkSeq/100%100))) < conf.MtraceRate {
			traceCtx.MTid = checkSeq
		}
	}
	traceCtx.TraceMtraceCallerValue = fmt.Sprintf("%s,%s,%s", hexa32.ToString32(traceCtx.MTid), strconv.Itoa(int(traceCtx.MDepth)+1), hexa32.ToString32(traceCtx.Txid))
	traceCtx.TraceMtraceSpecValue = fmt.Sprintf("%s, %s", conf.MtraceSpec, strconv.Itoa(int(hash.HashStr(traceCtx.Name))))
	traceCtx.TraceMtracePoidValue = fmt.Sprintf("%s, %s, %s", hexa32.ToString32(conf.PCODE), hexa32.ToString32(int64(conf.OKIND)), hexa32.ToString32(conf.OID))
}
