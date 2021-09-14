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

	fmt.Println("trace.Start txid=", wCtx.Txid)
	p := udp.NewUdpTxStartPack()

	p.Txid = wCtx.Txid
	p.Time = wCtx.StartTime
	p.Host = ""
	p.Uri = name
	p.Ipaddr = ""
	p.HttpMethod = ""
	p.Ref = ""
	p.UAgent = ""
	udpClient.Send(p)

	return ctx, nil
}

func StartWithRequest(r *http.Request) (context.Context, error) {
	udpClient := whatapnet.GetUdpClient()
	conf := config.GetConfig()
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

	p := udp.NewUdpTxStartPack()

	p.Txid = wCtx.Txid
	p.Time = wCtx.StartTime
	p.Host = r.Host
	p.Uri = r.RequestURI
	p.Ipaddr = r.RemoteAddr
	p.HttpMethod = r.Method
	p.Ref = r.Referer()
	p.UAgent = r.UserAgent()
	udpClient.Send(p)

	// Parse form
	if conf.ProfileHttpParameterEnabled && strings.HasPrefix(wCtx.Name, conf.ProfileHttpParameterUrlPrefix) {
		r.ParseForm()
		fmt.Println("param - ", r.Form)
		// r.Form -> url.Values -> map[string][]string
		sb := stringutil.NewStringBuffer()
		fmt.Println("len(r.Form)=", len(r.Form))
		if len(r.Form) > 0 {
			for k, v := range r.Form {
				sb.Append(k).Append("=")
				if len(v) > 0 {
					sb.AppendLine(v[0])
				}
			}
			p := udp.NewUdpTxSecureMessagePack()
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Hash = "HTTP-PARAMS"
			p.Desc = sb.ToString()
			fmt.Println("Pack - ", p)
			udpClient.Send(p)
			sb.Clear()
		}

	}
	fmt.Println("Cookie - ", r.Cookies())

	// r.Form -> url.Values -> map[string][]string

	fmt.Println("len(r.Header)=", len(r.Header))
	if conf.ProfileHttpHeaderEnabled && strings.HasPrefix(wCtx.Name, conf.ProfileHttpHeaderUrlPrefix) {
		sb := stringutil.NewStringBuffer()
		if len(r.Header) > 0 {
			for k, v := range r.Header {
				sb.Append(k).Append("=")
				if len(v) > 0 {
					sb.AppendLine(v[0])
				}
			}
			fmt.Println("header - ", r.Header)

			p := udp.NewUdpTxMessagePack()
			p.Txid = wCtx.Txid
			p.Time = dateutil.SystemNow()
			p.Hash = "HTTP-HEADERS"
			p.Desc = sb.ToString()
			fmt.Println("Pack - ", p)
			udpClient.Send(p)
			sb.Clear()
		}
	}
	return ctx, nil
}

func Step(ctx context.Context, title, message string, elapsed, value int) error {
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*TraceCtx)
		p := udp.NewUdpTxMessagePack()
		p.Txid = wCtx.Txid
		p.Time = dateutil.SystemNow()
		p.Hash = title
		p.Desc = message
		//p.Value = value
		udpClient.Send(p)
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func End(ctx context.Context, err error) error {
	fmt.Println("trace.End ", err)
	udpClient := whatapnet.GetUdpClient()
	if v := ctx.Value("whatap"); v != nil {
		wCtx := v.(*TraceCtx)
		fmt.Println("trace.End txid=", wCtx.Txid, ",name=", wCtx.Name)
		p := udp.NewUdpTxEndPack()
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

		fmt.Println("Trace.End mtid=", wCtx.MTid, ",depth=", wCtx.MDepth, ",ctxid=", wCtx.MCallerTxid)
		fmt.Println("Trace.End poid=", wCtx.MCallerPoidKey, ",spec=", wCtx.MCallerSpec, ",url=", wCtx.MCallerUrl)
		udpClient.Send(p)
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func UpdateMtrace(wCtx *TraceCtx, header http.Header) {
	fmt.Println("UpdateMtrace ")
	conf := config.GetConfig()
	if !conf.MtraceEnabled {
		return
	}
	for k, _ := range header {
		fmt.Println("UpdateMtrace k=", k, ",v=", strings.TrimSpace(header.Get(k)))
		v := strings.TrimSpace(header.Get(k))
		switch strings.ToLower(strings.TrimSpace(k)) {
		case conf.TraceMtraceCallerKey:
			arr := stringutil.Split(v, ",")
			if len(arr) >= 3 {
				wCtx.MTid = hexa32.ToLong32(arr[0])
				fmt.Println("mtid=", wCtx.MTid)

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

	fmt.Printf("UpdateMtrace %s, %s, %s, %d, %d, %d", wCtx.TraceMtraceCallerValue, wCtx.TraceMtracePoidValue, wCtx.TraceMtraceSpecValue, conf.PCODE, conf.OKIND, conf.OID)
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
