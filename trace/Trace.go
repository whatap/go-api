//github.com/whatap/go-api/trace
package trace

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/whatap/go-api/common/lang/pack/udp"
	whatapnet "github.com/whatap/go-api/common/net"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/common/util/hash"
	"github.com/whatap/go-api/common/util/hexa32"
	"github.com/whatap/go-api/common/util/keygen"
	"github.com/whatap/go-api/common/util/stringutil"
	"github.com/whatap/go-api/config"
)

func Init(m map[string]string) {
	// TO-DO
	if m != nil {
		config.GetConfig().ApplyConfig(m)
	}
	udpClient := whatapnet.GetUdpClient()
	p := udp.NewUdpConfigPack()
	p.Data = config.GetConfig().ToString()

	udpClient.Send(p)
}

func Start(ctx context.Context, name string) (context.Context, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return ctx, nil
	}

	udpClient := whatapnet.GetUdpClient()
	var wCtx *TraceCtx
	if v := ctx.Value("whatap"); v != nil {
		wCtx = v.(*TraceCtx)
	} else {
		wCtx = new(TraceCtx)
		wCtx.Txid = keygen.Next()
		ctx = context.WithValue(ctx, "whatap", wCtx)
	}
	wCtx.Name = name
	wCtx.StartTime = dateutil.SystemNow()

	if pack := udp.CreatePack(udp.TX_START, udp.UDP_PACK_VERSION); pack != nil {
		p := pack.(*udp.UdpTxStartPack)
		p.Txid = wCtx.Txid
		p.Time = wCtx.StartTime
		p.Host = ""
		p.Uri = name
		p.Ipaddr = ""
		p.HttpMethod = ""
		p.Ref = ""
		p.UAgent = ""
		udpClient.Send(p)
	}
	return ctx, nil
}

func StartWithRequest(r *http.Request) (context.Context, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return r.Context(), nil
	}

	udpClient := whatapnet.GetUdpClient()
	ctx := r.Context()
	var wCtx *TraceCtx
	if v := ctx.Value("whatap"); v != nil {
		wCtx = v.(*TraceCtx)
	} else {
		wCtx = new(TraceCtx)
		wCtx.Txid = keygen.Next()
		ctx = context.WithValue(ctx, "whatap", wCtx)
	}

	wCtx.Name = r.RequestURI
	wCtx.Host = r.Host
	wCtx.StartTime = dateutil.SystemNow()

	// update multi trace info
	UpdateMtrace(wCtx, r.Header)

	if pack := udp.CreatePack(udp.TX_START, udp.UDP_PACK_VERSION); pack != nil {
		p := pack.(*udp.UdpTxStartPack)

		p.Txid = wCtx.Txid
		p.Time = wCtx.StartTime
		p.Host = r.Host
		p.Uri = r.RequestURI
		p.Ipaddr = r.RemoteAddr
		p.HttpMethod = r.Method
		p.Ref = r.Referer()
		p.UAgent = r.UserAgent()
		udpClient.Send(p)
	}
	// Parse form
	if conf.ProfileHttpParameterEnabled && strings.HasPrefix(wCtx.Name, conf.ProfileHttpParameterUrlPrefix) {
		r.ParseForm()
		sb := stringutil.NewStringBuffer()
		if len(r.Form) > 0 {
			for k, v := range r.Form {
				sb.Append(k).Append("=")
				if len(v) > 0 {
					sb.AppendLine(v[0])
				}
			}
			if pack := udp.CreatePack(udp.TX_SECURE_MSG, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxSecureMessagePack)
				p.Time = dateutil.SystemNow()
				p.Hash = "HTTP-PARAMS"
				p.Desc = sb.ToString()
				udpClient.Send(p)
			}
			sb.Clear()
		}

	}

	// r.Form -> url.Values -> map[string][]string
	if conf.ProfileHttpHeaderEnabled && strings.HasPrefix(wCtx.Name, conf.ProfileHttpHeaderUrlPrefix) {
		sb := stringutil.NewStringBuffer()
		if len(r.Header) > 0 {
			for k, v := range r.Header {
				sb.Append(k).Append("=")
				if len(v) > 0 {
					sb.AppendLine(v[0])
				}
			}
			if pack := udp.CreatePack(udp.TX_MSG, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxMessagePack)
				p.Txid = wCtx.Txid
				p.Time = dateutil.SystemNow()
				p.Hash = "HTTP-HEADERS"
				p.Desc = sb.ToString()
				udpClient.Send(p)
			}
			sb.Clear()
		}
	}
	return ctx, nil
}

func Step(ctx context.Context, title, message string, elapsed, value int) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*TraceCtx)
		if pack := udp.CreatePack(udp.TX_MSG, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxMessagePack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Hash = title
			p.Desc = message
			//p.Value = value
			udpClient.Send(p)
		}
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func End(ctx context.Context, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*TraceCtx)
		if pack := udp.CreatePack(udp.TX_END, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxEndPack)
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()

			p.Host = wCtx.Host
			p.Uri = wCtx.Name

			p.Mtid = wCtx.MTid
			p.Mdepth = wCtx.MDepth
			p.McallerTxid = wCtx.MCallerTxid
			p.McallerPoidKey = wCtx.MCallerPoidKey
			p.McallerSpec = wCtx.MCallerSpec
			p.McallerUrl = wCtx.MCallerUrl

			udpClient.Send(p)
		}
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func UpdateMtrace(wCtx *TraceCtx, header http.Header) {
	conf := config.GetConfig()
	if !conf.Enabled || !conf.MtraceEnabled {
		return
	}
	for k, _ := range header {
		v := strings.TrimSpace(header.Get(k))
		switch strings.ToLower(strings.TrimSpace(k)) {
		case conf.TraceMtraceCallerKey:
			arr := stringutil.Split(v, ",")
			if len(arr) >= 3 {
				wCtx.MTid = hexa32.ToLong32(arr[0])

				if val, err := strconv.Atoi(arr[1]); err == nil {
					wCtx.MDepth = int32(val)
				}
				wCtx.MCallerTxid = hexa32.ToLong32(arr[2])
			}
		case conf.TraceMtraceCalleeKey:
			wCtx.MCallee = hexa32.ToLong32(v)
			if wCtx.MCallee != 0 {
				wCtx.Txid = wCtx.MCallee
			}

		case conf.TraceMtraceSpecKey1:
			arr := stringutil.Split(v, ",")
			if len(arr) >= 2 {
				wCtx.MCallerSpec = arr[0]
				wCtx.MCallerUrl = arr[1]
			}
		case conf.TraceMtracePoidKey:
			wCtx.MCallerPoidKey = v
		}
	}

	if wCtx.MTid == 0 {
		checkSeq := keygen.Next()
		checkFlag := false
		x := conf.MtraceRate / 10

		switch x {
		case 10:
			checkFlag = true
		case 9:
			if checkSeq%10 != 0 {
				checkFlag = true
			}
		case 8:
			if checkSeq%5 != 0 {
				checkFlag = true
			}
		case 7:
			if checkSeq%4 != 0 {
				checkFlag = true
			}
		case 6:
			if checkSeq%3 != 0 {
				checkFlag = true
			}
		case 5:
			if checkSeq%2 != 0 {
				checkFlag = true
			}
		case 4:
			if checkSeq%3 == 0 || checkSeq%5 == 0 {
				checkFlag = true
			}
		case 3:
			if checkSeq%4 == 0 || checkSeq%5 == 0 {
				checkFlag = true
			}
		case 2:
			if checkSeq%5 == 0 {
				checkFlag = true
			}
		case 1:
			if checkSeq%10 == 0 {
				checkFlag = true
			}
		}

		if checkFlag {
			wCtx.MTid = keygen.Next()
		}
	}

	wCtx.TraceMtraceCallerValue = fmt.Sprintf("%s,%s,%s", hexa32.ToString32(wCtx.MTid), strconv.Itoa(int(wCtx.MDepth)+1), hexa32.ToString32(wCtx.Txid))
	wCtx.TraceMtraceSpecValue = fmt.Sprintf("%s, %s", conf.MtraceSpec, strconv.Itoa(int(hash.HashStr(wCtx.Name))))
	wCtx.TraceMtracePoidValue = fmt.Sprintf("%s, %s, %s", hexa32.ToString32(conf.PCODE), hexa32.ToString32(int64(conf.OKIND)), hexa32.ToString32(conf.OID))
}
func Shutdown() {
	whatapnet.UdpShutdown()
}

// wrapping type of http.HanderFunc, example : http.Handle(pattern, http.HandlerFunc)
func HandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wCtx, _ := StartWithRequest(r)
		defer End(wCtx, nil)
		handler(w, r.WithContext(wCtx))
	})
}

// wrapping handler function, example : http.HandleFunc(func(http.ResponseWriter, *http.Request))
func Func(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		wCtx, _ := StartWithRequest(r)
		defer End(wCtx, nil)
		handler(w, r.WithContext(wCtx))
	}
}
