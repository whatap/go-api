//github.com/whatap/go-api/trace
package trace

import (
	"context"
	"fmt"
	"math"
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

func Shutdown() {
	whatapnet.UdpShutdown()
}

func GetTraceContext(ctx context.Context) (context.Context, *TraceCtx) {
	if ctx == nil {
		return ctx, nil
	}
	var wCtx *TraceCtx
	if v := ctx.Value("whatap"); v != nil {
		wCtx = v.(*TraceCtx)
	} else {
		wCtx = nil
	}
	return ctx, wCtx
}

func NewTraceContext(ctx context.Context) (context.Context, *TraceCtx) {
	if ctx == nil {
		ctx = context.Background()
	}
	var wCtx *TraceCtx
	wCtx = new(TraceCtx)
	wCtx.Txid = keygen.Next()
	ctx = context.WithValue(ctx, "whatap", wCtx)
	return ctx, wCtx
}

func Start(ctx context.Context, name string) (context.Context, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return ctx, nil
	}

	udpClient := whatapnet.GetUdpClient()
	ctx, wCtx := NewTraceContext(ctx)
	wCtx.Name = name
	wCtx.StartTime = dateutil.SystemNow()
	// update multi trace info
	UpdateMtrace(wCtx, http.Header{})

	if pack := udp.CreatePack(udp.TX_START, udp.UDP_PACK_VERSION); pack != nil {
		p := pack.(*udp.UdpTxStartPack)
		p.Txid = wCtx.Txid
		p.Time = wCtx.StartTime
		p.Host = wCtx.Host
		p.Uri = name
		p.Ipaddr = wCtx.Ipaddr
		p.HttpMethod = wCtx.HttpMethod
		p.Ref = wCtx.Ref
		p.UAgent = wCtx.UAgent
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
	ctx, wCtx := NewTraceContext(r.Context())

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
	// r.Form -> url.Values -> map[string][]string
	r.ParseForm()
	SetParameter(ctx, r.Form)
	// http.Header -> map[string][]string
	SetHeader(ctx, r.Header)

	return ctx, nil
}

func StartWithContext(ctx context.Context, name string) (context.Context, error) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return ctx, nil
	}
	udpClient := whatapnet.GetUdpClient()
	if ctx, wCtx := GetTraceContext(ctx); wCtx != nil {
		wCtx.Name = name
		wCtx.StartTime = dateutil.SystemNow()
		// update multi trace info
		UpdateMtrace(wCtx, http.Header{})

		if pack := udp.CreatePack(udp.TX_START, udp.UDP_PACK_VERSION); pack != nil {
			p := pack.(*udp.UdpTxStartPack)
			p.Txid = wCtx.Txid
			p.Time = wCtx.StartTime
			p.Host = wCtx.Host
			p.Uri = name
			p.Ipaddr = wCtx.Ipaddr
			p.HttpMethod = wCtx.HttpMethod
			p.Ref = wCtx.Ref
			p.UAgent = wCtx.UAgent
			udpClient.Send(p)
		}
	} else {
		return ctx, fmt.Errorf("Not found trace context ")
	}
	return ctx, nil
}

func SetHeader(ctx context.Context, m map[string][]string) {
	conf := config.GetConfig()
	if !conf.ProfileHttpHeaderEnabled {
		return
	}
	udpClient := whatapnet.GetUdpClient()
	if _, wCtx := GetTraceContext(ctx); wCtx != nil {
		// http.Header -> map[string][]string
		if strings.HasPrefix(wCtx.Name, conf.ProfileHttpHeaderUrlPrefix) {
			if pack := udp.CreatePack(udp.TX_MSG, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxMessagePack)
				p.Txid = wCtx.Txid
				p.Time = dateutil.SystemNow()
				p.Hash = "HTTP-HEADERS"
				p.SetHeader(map[string][]string(m))
				udpClient.Send(p)
			}
		}
	}
}

func SetParameter(ctx context.Context, m map[string][]string) {
	conf := config.GetConfig()
	if !conf.Enabled {
		return
	}
	udpClient := whatapnet.GetUdpClient()
	if _, wCtx := GetTraceContext(ctx); wCtx != nil {
		if conf.ProfileHttpParameterEnabled && strings.HasPrefix(wCtx.Name, conf.ProfileHttpParameterUrlPrefix) {
			if pack := udp.CreatePack(udp.TX_SECURE_MSG, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxSecureMessagePack)
				p.Time = dateutil.SystemNow()
				p.Hash = "HTTP-PARAMS"
				p.SetParameter(map[string][]string(m))
				udpClient.Send(p)
			}
		}
	}
}
func Step(ctx context.Context, title, message string, elapsed, value int) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if _, wCtx := GetTraceContext(ctx); wCtx != nil {
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

func Error(ctx context.Context, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if err != nil {
		if _, wCtx := GetTraceContext(ctx); wCtx != nil {
			if pack := udp.CreatePack(udp.TX_ERROR, udp.UDP_PACK_VERSION); pack != nil {
				p := pack.(*udp.UdpTxErrorPack)
				p.Txid = wCtx.Txid
				p.Time = dateutil.SystemNow()
				p.ErrorType = err.Error()
				p.ErrorMessage = err.Error()

				udpClient.Send(p)
			}
			return nil
		}
	}

	return fmt.Errorf("Not found Txid ")
}

func End(ctx context.Context, err error) error {
	conf := config.GetConfig()
	if !conf.Enabled {
		return nil
	}
	udpClient := whatapnet.GetUdpClient()
	if _, wCtx := GetTraceContext(ctx); wCtx != nil {
		Error(ctx, err)
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

func UpdateMtraceWithContext(ctx context.Context, header http.Header) {
	if _, wCtx := GetTraceContext(ctx); wCtx != nil {
		UpdateMtrace(wCtx, header)
	}
}
func GetMTrace(ctx context.Context) http.Header {
	rt := make(http.Header)
	conf := config.GetConfig()
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		rt.Set(conf.TraceMtraceCallerKey, traceCtx.TraceMtraceCallerValue)
		rt.Set(conf.TraceMtracePoidKey, traceCtx.TraceMtracePoidValue)
		rt.Set(conf.TraceMtraceSpecKey1, traceCtx.TraceMtraceSpecValue)
	}
	return rt
}
func UpdateMtrace(wCtx *TraceCtx, header http.Header) {
	conf := config.GetConfig()
	if !conf.MtraceEnabled {
		return
	}
	for k, val := range header {
		if len(val) > 0 {
			v := strings.TrimSpace(val[0])
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
	}

	if wCtx.MTid == 0 {
		checkSeq := keygen.Next()
		if int32(math.Abs(float64(checkSeq/100%100))) < conf.MtraceRate {
			wCtx.MTid = checkSeq
		}
	}
	wCtx.TraceMtraceCallerValue = fmt.Sprintf("%s,%s,%s", hexa32.ToString32(wCtx.MTid), strconv.Itoa(int(wCtx.MDepth)+1), hexa32.ToString32(wCtx.Txid))
	wCtx.TraceMtraceSpecValue = fmt.Sprintf("%s, %s", conf.MtraceSpec, strconv.Itoa(int(hash.HashStr(wCtx.Name))))
	wCtx.TraceMtracePoidValue = fmt.Sprintf("%s, %s, %s", hexa32.ToString32(conf.PCODE), hexa32.ToString32(int64(conf.OKIND)), hexa32.ToString32(conf.OID))
}

// wrapping type of http.HanderFunc, example : http.Handle(pattern, http.HandlerFunc)
func HandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := StartWithRequest(r)
		defer End(ctx, nil)
		handler(w, r.WithContext(ctx))
	})
}

// wrapping handler function, example : http.HandleFunc(func(http.ResponseWriter, *http.Request))
func Func(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := StartWithRequest(r)
		defer End(ctx, nil)
		handler(w, r.WithContext(ctx))
	}
}
