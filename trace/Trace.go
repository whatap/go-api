// github.com/whatap/go-api/trace
package trace

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	whatapboot "github.com/whatap/go-api/agent/agent/boot"
	agentconfig "github.com/whatap/go-api/agent/agent/config"
	agenttrace "github.com/whatap/go-api/agent/agent/trace"
	agentapi "github.com/whatap/go-api/agent/agent/trace/api"

	"github.com/whatap/golib/io"
	langvalue "github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hexa32"
	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/keygen"
	"github.com/whatap/golib/util/logo"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/golib/util/urlutil"
)

var (
	WHATAP_COOKIE_NAME = "WHATAP"
	traceLock          sync.Mutex
	disable            bool = true
)

func DISABLE() bool {
	return disable
}

type WrapResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (l *WrapResponseWriter) WriteHeader(status int) {
	l.Status = status
	l.ResponseWriter.WriteHeader(status)
}

func Init(m map[string]string) {
	logo.Print2("golang", whatapboot.AGENT_VERSION)
	// DISABLE = false
	disable = false

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if m != nil {
		agentconfig.GetConfig()
		agentconfig.SetValues(&m)
	}
	keygen.AddSeed(os.Getpid())
	// embeded
	go whatapboot.Boot()
}

func Shutdown() {
	// whatapnet.UdpShutdown()
}

func GetTraceContext(ctx context.Context) (context.Context, *TraceCtx) {
	if ctx == nil {
		return ctx, nil
	}
	if v := ctx.Value("whatap"); v != nil {
		return ctx, v.(*TraceCtx)
	}

	// TO-DO goroutine id
	if v := GetGIDTraceCtx(GetGID()); v != nil {
		return ctx, v
	}

	return ctx, nil
}

func GetAgentTraceContext(tCtx *TraceCtx) *agenttrace.TraceContext {
	if tCtx != nil {
		return tCtx.Ctx
	}
	return nil
}

func NewTraceContext(ctx context.Context) (context.Context, *TraceCtx) {
	if ctx == nil {
		ctx = context.Background()
	}
	var traceCtx *TraceCtx
	traceCtx = PoolTraceContext()
	traceCtx.GID = GetGID()
	traceCtx.Ctx = agenttrace.PoolTraceContext()

	wCtx := traceCtx.Ctx
	wCtx.Txid = keygen.Next()
	traceCtx.Txid = wCtx.Txid

	ctx = context.WithValue(ctx, "whatap", traceCtx)
	// TO-DO goroutine id
	AddGIDTraceCtx(traceCtx.GID, traceCtx)
	return ctx, traceCtx
}

func Start(ctx context.Context, name string) (context.Context, error) {
	if DISABLE() {
		return ctx, nil
	}
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return ctx, nil
	}

	ctx, traceCtx := NewTraceContext(ctx)
	traceCtx.Name = name
	traceCtx.StartTime = dateutil.SystemNow()
	// update multi trace info
	UpdateMtrace(traceCtx, http.Header{})

	wCtx := traceCtx.Ctx
	wCtx.StartTime = traceCtx.StartTime
	wCtx.ServiceURL = urlutil.NewURL(name)
	agentapi.StartTx(wCtx)

	return ctx, nil
}

func StartWithRequest(r *http.Request) (context.Context, error) {
	if DISABLE() {
		return r.Context(), nil
	}
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return r.Context(), nil
	}

	ctx, traceCtx := NewTraceContext(r.Context())
	traceCtx.Name = r.RequestURI
	traceCtx.StartTime = dateutil.SystemNow()
	// update multi trace info
	UpdateMtrace(traceCtx, r.Header)

	wCtx := traceCtx.Ctx
	wCtx.StartTime = traceCtx.StartTime
	wCtx.ServiceURL = urlutil.NewURL(filepath.Join(r.Host, "/", r.RequestURI))
	ipaddr := GetRemoteIP(r.RemoteAddr, r.Header)
	wCtx.RemoteIp = io.ToInt(iputil.ToBytes(ipaddr), 0)
	wCtx.HttpMethod = r.Method
	wCtx.RefererURL = urlutil.NewURL(r.Referer())
	wCtx.UserAgentString = r.UserAgent()
	wCtx.WClientId = int64(hash.HashStr(GetClientId(r, ipaddr)))
	if conf.Debug {
		log.Println("[WA-TX-02001] StartWithRequest: ", traceCtx.Txid, ", ", traceCtx.Name)
	}
	agentapi.StartTx(wCtx)

	//http.Header -> map[string][]string
	SetHeader(ctx, r.Header)

	return ctx, nil
}

func StartWithContext(ctx context.Context, name string) (context.Context, error) {
	if DISABLE() {
		return ctx, nil
	}
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return ctx, nil
	}
	if ctx, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		traceCtx.Name = name
		traceCtx.StartTime = dateutil.SystemNow()
		// update multi trace info
		UpdateMtrace(traceCtx, http.Header{})

		wCtx := traceCtx.Ctx
		if wCtx == nil {
			wCtx = agenttrace.PoolTraceContext()
			traceCtx.Ctx = wCtx
			wCtx.Txid = traceCtx.Txid
		}

		wCtx.StartTime = traceCtx.StartTime
		wCtx.ServiceURL = urlutil.NewURL(name)
		if conf.Debug {
			log.Println("[WA-TX-03001] StartWithContext: ", traceCtx.Txid, ", ", traceCtx.Name)
		}
		agentapi.StartTx(wCtx)
	} else {
		if conf.Debug {
			log.Println("[WA-TX-03002] StartWithContext: Not found trace context ", name)
		}
		return ctx, fmt.Errorf("Not found trace context ")
	}
	return ctx, nil
}

func SetHeader(ctx context.Context, m map[string][]string) {
	if DISABLE() {
		return
	}
	conf := agentconfig.GetConfig()
	if !conf.ProfileHttpHeaderEnabled {
		return
	}
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		// http.Header -> map[string][]string
		if strings.HasPrefix(traceCtx.Name, conf.ProfileHttpHeaderUrlPrefix) {
			parsedHeader := ParseHeader(m)
			agentapi.ProfileMsg(traceCtx.Ctx, "HTTP_HEADERS", parsedHeader, 0, 0)
			if conf.Debug {
				log.Println("[WA-TX-06001] txid:", traceCtx.Txid, ", uri: ", traceCtx.Name, "\n headers: ", parsedHeader)
			}
		}
	}
}
func SetParameter(ctx context.Context, m map[string][]string) {
	if DISABLE() {
		return
	}
	conf := agentconfig.GetConfig()
	if !conf.ProfileHttpParameterEnabled {
		return
	}
	if m == nil && len(m) <= 0 {
		return
	}
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		if strings.HasPrefix(traceCtx.Name, conf.ProfileHttpParameterUrlPrefix) {
			parsedParam := ParseParameter(m)
			agentapi.ProfileSecureMsg(traceCtx.Ctx, "HTTP-PARAMS", parsedParam, 0, 0)
			if conf.Debug {
				log.Println("[WA-TX-07001] HTTP-PARAMS txid:", traceCtx.Txid, ", uri: ", traceCtx.Name, "\n params: ", parsedParam)
			}
		}
	}
}
func GetClientId(r *http.Request, remoteIP string) string {
	if DISABLE() {
		return strings.TrimSpace(remoteIP)
	}
	conf := agentconfig.GetConfig()

	if !conf.Enabled || !conf.TraceUserEnabled {
		return strings.TrimSpace(remoteIP)
	}
	if conf.TraceUserUsingIp {
		return strings.TrimSpace(remoteIP)
	}
	if conf.TraceUserHeaderTicketEnabled {
		for k, v := range r.Header {
			if strings.ToLower(strings.TrimSpace(k)) == strings.ToLower(strings.TrimSpace(conf.TraceUserHeaderTicket)) && len(v) > 0 {
				return strings.TrimSpace(v[0])
			}
		}
	}

	for _, cookie := range r.Cookies() {
		for _, v := range conf.TraceUserCookieKeys {
			if strings.ToLower(strings.TrimSpace(cookie.Name)) == strings.ToLower(strings.TrimSpace(v)) {
				return strings.TrimSpace(cookie.Value)
			}
		}
	}

	// WhaTap Cookie name is constant WHATAP_COOKIE_NAME(WHATAP)
	for _, cookie := range r.Cookies() {
		if strings.ToUpper(strings.TrimSpace(cookie.Name)) == WHATAP_COOKIE_NAME {
			return strings.TrimSpace(cookie.Value)
		}
	}

	return strings.TrimSpace(remoteIP)
}
func GetWhatapCookie(r *http.Request) (cookie *http.Cookie, exists bool) {
	for _, c := range r.Cookies() {
		if c.Name == WHATAP_COOKIE_NAME {
			return c, true
		}
	}
	if cookie == nil {
		cookie = &http.Cookie{
			Name:  WHATAP_COOKIE_NAME,
			Value: fmt.Sprintf("%d", keygen.Next()),
		}
	}
	return cookie, false
}

func SetWhatapCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if w != nil && cookie != nil {
		w.Header().Add("Set-Cookie", cookie.String())
	}
}

func Step(ctx context.Context, title, message string, elapsed, value int) error {
	if DISABLE() {
		return nil
	}
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		agentapi.ProfileMsg(traceCtx.Ctx, title, message, int32(elapsed), int32(value))
		return nil
	}

	return fmt.Errorf("Not found Txid ")
}

func Error(ctx context.Context, err error) error {
	if DISABLE() {
		return nil
	}
	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}
	if err != nil {
		var txid int64
		var serviceName string

		if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
			agentapi.ProfileError(traceCtx.Ctx, err)
			txid = traceCtx.Txid
			serviceName = traceCtx.Name
		} else {
			agentapi.ProfileError(nil, err)
		}

		if conf.Debug {
			log.Println("[WA-TX-04001] txid:", txid, ", uri: ", serviceName, "\n error: ", err)
		}
	}
	return nil
}

func End(ctx context.Context, err error) error {
	if DISABLE() {
		return nil
	}

	conf := agentconfig.GetConfig()
	if !conf.Enabled {
		return nil
	}
	Error(ctx, err)
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		wCtx := traceCtx.Ctx
		wCtx.Mtid = traceCtx.MTid
		wCtx.Mdepth = traceCtx.MDepth
		wCtx.McallerTxid = traceCtx.MCallerTxid
		wCtx.McallerPoidKey = traceCtx.MCallerPoidKey
		wCtx.McallerSpec = traceCtx.MCallerSpec
		wCtx.McallerUrl = traceCtx.MCallerUrl
		wCtx.McallerStepId = traceCtx.MCallerStepId
		wCtx.Status = traceCtx.Status

		if conf.Debug {
			log.Println("[WA-TX-05001] txid: ", traceCtx.Txid, ", uri: ", traceCtx.Name,
				"\n time: ", (dateutil.SystemNow() - traceCtx.StartTime), "ms ", "\n error: ", err)
		}

		// tracecontext traceparent
		wCtx.SetExtraFieldString("x-trace-id", traceCtx.MCallerTraceId)
		if wCtx.McallerTxid == 0 && wCtx.McallerStepId != 0 {
			wCtx.SetExtraField("x-parent-id", langvalue.NewDecimalValue(wCtx.McallerStepId))
		}

		agentapi.EndTx(wCtx)
		// TO-DO goroutine id
		RemoveGIDTraceCtx(traceCtx.GID)
		CloseTraceContext(traceCtx)
		return nil
	}
	if conf.Debug {
		log.Println("[WA-TX-05002] End: Not found Txid ", "\n error: ", err)
	}
	return fmt.Errorf("Not found Txid ")
}

func UpdateMtraceWithContext(ctx context.Context, header http.Header) {
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		UpdateMtrace(traceCtx, header)
	}
}
func GetMTrace(ctx context.Context) http.Header {
	if DISABLE() {
		return make(http.Header)
	}
	rt := make(http.Header)
	conf := agentconfig.GetConfig()
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		// create distribute trace header
		traceCtx.MStepId = keygen.Next()
		if traceCtx.MCallerTraceId != "" {
			traceCtx.TraceMtraceTraceparentValue = fmt.Sprintf("00-%s-%016x-01", traceCtx.MCallerTraceId, uint64(traceCtx.MStepId))
		} else {
			traceCtx.TraceMtraceTraceparentValue = fmt.Sprintf("00-0000000000000000%016x-%016x-01", uint64(traceCtx.MTid), uint64(traceCtx.MStepId))
		}
		traceCtx.TraceMtraceCallerValue = fmt.Sprintf("%s,%s,%s,%s", hexa32.ToString32(traceCtx.MTid), strconv.Itoa(int(traceCtx.MDepth)+1), hexa32.ToString32(traceCtx.Txid), hexa32.ToString32(traceCtx.MStepId))

		rt.Set(conf.TraceMtraceTraceparentKey, traceCtx.TraceMtraceTraceparentValue)
		rt.Set(conf.TraceMtraceCallerKey, traceCtx.TraceMtraceCallerValue)
		rt.Set(conf.TraceMtracePoidKey, traceCtx.TraceMtracePoidValue)
		rt.Set(conf.TraceMtraceSpecKey1, traceCtx.TraceMtraceSpecValue)

		// 2023.11.07 deprcated
		// Mcallee
		// if conf.MtraceCalleeTxidEnabled {
		// 	traceCtx.TraceMtraceMcallee = keygen.Next()
		// 	rt.Set(conf.TraceMtraceCalleeKey, fmt.Sprintf("%d", traceCtx.TraceMtraceMcallee))
		// }
	}

	return rt
}
func UpdateMtrace(traceCtx *TraceCtx, header http.Header) {
	if DISABLE() {
		return
	}

	conf := agentconfig.GetConfig()
	if !conf.MtraceEnabled {
		return
	}
	// convert new header, header key syntax is canoncail convention
	h := make(http.Header)
	for k, v := range header {
		for _, it := range v {
			h.Add(k, it)
		}
	}

	isTraceparent := false
	useWhatap := true
	// W3C Trace Context traceparent
	if val := h.Get(conf.TraceMtraceTraceparentKey); val != "" {
		isTraceparent = true
		v := strings.TrimSpace(val)
		arr := stringutil.Split(v, "-")
		if len(arr) >= 4 {
			traceCtx.MCallerTraceId = arr[1]
			if val, err := strconv.ParseUint(traceCtx.MCallerTraceId[16:], 16, 64); err == nil {
				traceCtx.MTid = int64(val)
			} else {
				traceCtx.MTid = 0
			}

			if val, err := strconv.ParseUint(arr[2], 16, 64); err == nil {
				traceCtx.MCallerStepId = int64(val)
			} else {
				traceCtx.MCallerStepId = 0
			}

			if conf.Debug {
				log.Println("[WA-TX-08001] update mtrace traceparent ", v, ", mtid=", traceCtx.MTid, ", mcaller_step=", traceCtx.MCallerStepId)
			}
		}
	}
	// x-wtap-mst
	if val := h.Get(conf.TraceMtraceCallerKey); val != "" {
		v := strings.TrimSpace(val)
		arr := stringutil.Split(v, ",")
		var mtid, stepId, mcallerTxid int64
		if len(arr) >= 3 {
			mtid = hexa32.ToLong32(arr[0])
			if val, err := strconv.Atoi(arr[1]); err == nil {
				traceCtx.MDepth = int32(val)
			}
			mcallerTxid = hexa32.ToLong32(arr[2])
		}
		if len(arr) >= 4 {
			stepId = hexa32.ToLong32(arr[3])
		}

		// traceparent , whatap header 모두 있을 때, 가능한 caller txid를 설정.
		// gateway에서 받은 header를 그대로 전달해 줄 경우 callertxid가 다르게 설정될 수 있음.
		if isTraceparent {
			if traceCtx.MCallerStepId == stepId {
				traceCtx.MCallerTxid = mcallerTxid
			} else {
				// parent id != whatap.stepid . don't use whatap header
				if conf.Debug {
					log.Printf("[WA-TX-08002] stepid(%d) is not equal traceparent stepid(%s), mtid=(%d), traceparent mtid=(%d)", traceCtx.MCallerStepId, stepId, mtid, traceCtx.MTid)
				}
				useWhatap = false
			}
		} else {
			traceCtx.MTid = mtid
			traceCtx.MCallerTxid = mcallerTxid
			traceCtx.MCallerStepId = stepId
		}

		if conf.Debug {
			log.Println("[WA-TX-08003] update mtrace x-wtap-mst ", v, ", mtid=", traceCtx.MTid, ", mcaller=", traceCtx.MCallerTxid, ", mcaller_step=", traceCtx.MCallerStepId)
		}
	}

	if useWhatap {
		// 2023.11.07 deprcated
		// if val := header.Get(conf.TraceMtraceCalleeKey); val != "" {
		// 	traceCtx.MCallee = hexa32.ToLong32(v)
		// 	if traceCtx.MCallee != 0 {
		// 		traceCtx.Txid = traceCtx.MCallee
		// 		if traceCtx.Ctx != nil {
		// 			traceCtx.Ctx.Txid = traceCtx.MCallee
		// 		}
		// 	}
		// }

		// x-wtap-spec1
		if val := h.Get(conf.TraceMtraceSpecKey1); val != "" {
			v := strings.TrimSpace(val)
			arr := stringutil.Split(v, ",")
			if len(arr) >= 2 {
				traceCtx.MCallerSpec = arr[0]
				traceCtx.MCallerUrl = arr[1]
			}
			if conf.Debug {
				log.Println("[WA-TX-08004] update mtrace x-wtap-spec1 ", v, ", mcaller_spec=", traceCtx.MCallerSpec, ", mcaller_url=", traceCtx.MCallerUrl)
			}
		}
		// x-wtap-poid
		if val := h.Get(conf.TraceMtracePoidKey); val != "" {
			v := strings.TrimSpace(val)
			traceCtx.MCallerPoidKey = v
			if conf.Debug {
				log.Println("[WA-TX-08004] update mtrace poid ", v)
			}
		}

	}

	if traceCtx.MTid == 0 {
		checkSeq := keygen.Next()
		if int32(math.Abs(float64(checkSeq/100%100))) < conf.MtraceRate {
			traceCtx.MTid = checkSeq
		}
	}

	traceCtx.MStepId = keygen.Next()
	// traceCtx.TraceMtraceTraceparentValue = fmt.Sprintf()

	if traceCtx.MCallerTraceId != "" {
		traceCtx.TraceMtraceTraceparentValue = fmt.Sprintf("00-%s-%016x-01", traceCtx.MCallerTraceId, uint64(traceCtx.MStepId))
	} else {
		traceCtx.TraceMtraceTraceparentValue = fmt.Sprintf("00-0000000000000000%016x-%016x-01", uint64(traceCtx.MTid), uint64(traceCtx.MStepId))
	}
	traceCtx.TraceMtraceCallerValue = fmt.Sprintf("%s,%s,%s,%s", hexa32.ToString32(traceCtx.MTid), strconv.Itoa(int(traceCtx.MDepth)+1), hexa32.ToString32(traceCtx.Txid), hexa32.ToString32(traceCtx.MStepId))
	traceCtx.TraceMtraceSpecValue = fmt.Sprintf("%s, %s", conf.MtraceSpec, strconv.Itoa(int(hash.HashStr(traceCtx.Name))))
	traceCtx.TraceMtracePoidValue = fmt.Sprintf("%s, %s, %s", hexa32.ToString32(conf.PCODE), hexa32.ToString32(int64(conf.OKIND)), hexa32.ToString32(conf.OID))
}

// wrapping type of http.HanderFunc, example : http.Handle(pattern, http.HandlerFunc)
func HandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(Func(handler))
}

// wrapping handler function, example : http.HandleFunc(func(http.ResponseWriter, *http.Request))
func Func(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if DISABLE() {
			handler(w, r)
			return
		}

		conf := agentconfig.GetConfig()
		if !conf.TransactionEnabled {
			handler(w, r)
			return
		}
		wrw := &WrapResponseWriter{ResponseWriter: w}
		ctx, _ := StartWithRequest(r)
		wRequest := r.WithContext(ctx)
		defer func() {
			x := recover()
			var err error = nil
			if x != nil {
				err = fmt.Errorf("%v", x)
				Error(ctx, err)
				err = nil
			}
			status := wrw.Status
			if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
				traceCtx.Status = int32(status)
			}
			if status >= 400 {
				err = fmt.Errorf("Status %d:%s", status, http.StatusText(status))
			}
			// trace http parameter
			if conf.ProfileHttpParameterEnabled && strings.HasPrefix(r.RequestURI, conf.ProfileHttpParameterUrlPrefix) {
				if wRequest.Form != nil {
					SetParameter(ctx, wRequest.Form)
				}
			}

			// Set Whatap Cookie
			if conf.TraceUserSetCookie {
				if cookie, exists := GetWhatapCookie(r); !exists {
					SetWhatapCookie(w, cookie)
				}
			}
			End(ctx, err)
			if x != nil {
				if !conf.GoRecoverEnabled {
					panic(x)
				}
			}
		}()
		handler(wrw, wRequest)

	}
}

func GetTxid(ctx context.Context) int64 {
	if _, traceCtx := GetTraceContext(ctx); traceCtx != nil {
		return traceCtx.Txid
	}
	return 0
}

func GetGID() int64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return int64(n)
}
func ParseParameter(m map[string][]string) string {
	rt := ""
	if m != nil && len(m) > 0 {
		sb := stringutil.NewStringBuffer()
		for k, v := range m {
			sb.Append(k).Append("=")
			if len(v) > 0 {
				sb.AppendLine(v[0])
			}
		}
		rt = sb.ToString()
		sb.Clear()
	}
	return rt
}

func ParseHeader(m map[string][]string) string {
	if DISABLE() {
		return ""
	}

	conf := agentconfig.GetConfig()
	rt := ""
	if m != nil && len(m) > 0 {
		sb := stringutil.NewStringBuffer()
		for k, v := range m {
			sb.Append(k).Append("=")
			if len(v) > 0 {
				key := strings.ReplaceAll(strings.ToLower(k), "-", "_")
				if !conf.ProfileHttpHeaderIgnoreKeys.HasKey(key) {
					sb.AppendLine(v[0])
				} else {
					sb.AppendLine("#")
				}
			}
		}
		rt = sb.ToString()
		sb.Clear()
	}
	return rt
}

func GetRemoteIP(remoteAddr string, header map[string][]string) string {
	if DISABLE() {
		return ""
	}

	conf := agentconfig.GetConfig()
	if conf.TraceHttpClientIpHeaderKeyEnabled && header != nil {
		var val []string
		var ok bool = false
		for k, v := range header {
			if strings.ToLower(strings.TrimSpace(k)) == strings.ToLower(strings.TrimSpace(conf.TraceHttpClientIpHeaderKey)) {
				val = v
				ok = true
				break
			}
		}
		if ok && len(val) > 0 {
			ipaddr := val[0]
			// X-Forwarded-For
			if strings.Index(ipaddr, ",") > -1 {
				ipArray := strings.Split(ipaddr, ",")
				if len(ipArray) > 1 {
					ipaddr = ipArray[0]
					return strings.TrimSpace(ipaddr)
				}
			}
		}
	}

	arr := strings.Split(remoteAddr, ":")
	if len(arr) > 0 {
		return arr[0]
	}
	return strings.TrimSpace(remoteAddr)
}
