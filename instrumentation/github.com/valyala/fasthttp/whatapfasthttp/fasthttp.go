package whatapfasthttp

import (
	"context"
	"fmt"
	"log"

	"net/http"
	"path/filepath"

	"strings"

	"github.com/valyala/fasthttp"
	"github.com/whatap/go-api/agent/agent/config"
	agentapi "github.com/whatap/go-api/agent/agent/trace/api"
	"github.com/whatap/go-api/trace"
	"github.com/whatap/golib/io"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"

	"github.com/whatap/golib/util/iputil"
	"github.com/whatap/golib/util/keygen"

	// "github.com/whatap/golib/util/stringutil"
	"github.com/whatap/golib/util/urlutil"
)

func Func(handler func(ctx *fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		conf := config.GetConfig()
		if !conf.TransactionEnabled {
			handler(ctx)
			return
		}
		wCtx, _ := StartWithFastHttpRequest(ctx)
		var err error = nil
		defer func() {
			x := recover()
			if x != nil {
				err = fmt.Errorf("Panic: %v", x)
				trace.Error(wCtx, err)
			}

			status := ctx.Response.StatusCode()
			if _, traceCtx := trace.GetTraceContext(wCtx); traceCtx != nil {
				traceCtx.Status = int32(status)
			}
			if status >= 400 {
				err = fmt.Errorf("Status %d:%s", status, http.StatusText(status))
			}

			reqeustURI := string(ctx.Request.RequestURI())

			// trace http parameter
			if conf.ProfileHttpParameterEnabled && strings.HasPrefix(reqeustURI, conf.ProfileHttpParameterUrlPrefix) {
				form := TraceHttpParameter(ctx)
				if form != nil {
					trace.SetParameter(wCtx, form)
				}
			}

			// Set WhaTap Cookie
			if conf.TraceUserSetCookie {
				if cookie, exists := GetWhatapCookie(ctx); !exists {
					SetWhatapCookie(ctx, cookie)
				}
			}
			trace.End(wCtx, err)
			if x != nil {
				if !conf.GoRecoverEnabled {
					panic(x)
				}
			}
		}()

		handler(ctx)
	}
}

func TraceHttpParameter(ctx *fasthttp.RequestCtx) map[string][]string {
	query_args := ctx.QueryArgs()
	form_args := ctx.PostArgs()
	form := make(map[string][]string)
	visit_func := func(key, value []byte) {
		// fmt.Println("visit_func key=", string(key), ",v=", string(value))
		form[string(key)] = append(form[string(key)], string(value))
	}
	if query_args != nil {
		query_args.VisitAll(visit_func)
	}
	if form_args != nil {
		form_args.VisitAll(visit_func)
	}
	return form
}

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

	_, traceCtx := trace.NewTraceContext(r)
	r.SetUserValue("whatap", traceCtx)

	traceCtx.Name = string(r.RequestURI())
	traceCtx.Host = string(r.Host())
	traceCtx.StartTime = dateutil.SystemNow()
	// update multi trace info
	UpdateFastHttpMtrace(traceCtx, r.Request.Header)

	wCtx := traceCtx.Ctx
	wCtx.StartTime = traceCtx.StartTime
	wCtx.ServiceURL = urlutil.NewURL(filepath.Join(string(r.Host()), "/", string(r.RequestURI())))

	ipaddr := trace.GetRemoteIP(r.RemoteAddr().String(), HeaderToMap(&r.Request.Header))
	wCtx.RemoteIp = io.ToInt(iputil.ToBytes(ipaddr), 0)
	wCtx.HttpMethod = string(r.Method())
	wCtx.RefererURL = urlutil.NewURL(string(r.Referer()))
	wCtx.UserAgentString = string(r.UserAgent())
	wCtx.WClientId = int64(hash.HashStr(GetClientId(r, ipaddr)))
	if conf.Debug {
		log.Println("[WA-TX-02001] StartWithFastHttpRequest: ", traceCtx.Txid, ", ", traceCtx.Name)
	}
	agentapi.StartTx(wCtx)

	SetFastHttpHeader(r, &r.Request.Header)

	return r, nil
}

func SetFastHttpHeader(ctx context.Context, header *fasthttp.RequestHeader) {
	trace.SetHeader(ctx, HeaderToMap(header))
}

func UpdateFastHttpMtrace(traceCtx *trace.TraceCtx, header fasthttp.RequestHeader) {
	conf := config.GetConfig()
	if !conf.MtraceEnabled {
		return
	}
	// convert fasthttp header to http.Header(map[string][]string)
	h := make(map[string][]string)
	header.VisitAll(func(keyRaw, valueRaw []byte) {
		if len(valueRaw) <= 0 {
			return
		}
		key, val := string(keyRaw), string(valueRaw)
		h[key] = []string{val}
	})
	trace.UpdateMtrace(traceCtx, h)
}

func GetClientId(ctx *fasthttp.RequestCtx, remoteIP string) string {
	r := ctx.Request
	clientID := remoteIP
	conf := config.GetConfig()
	if !conf.Enabled || !conf.TraceUserEnabled {
		return clientID
	}
	if conf.TraceUserUsingIp {
		return clientID
	}
	header := r.Header
	if conf.TraceUserHeaderTicketEnabled {
		header.VisitAll(func(keyRaw, valueRaw []byte) {
			if len(valueRaw) <= 0 {
				return
			}
			k := string(keyRaw)
			v := string(valueRaw)

			if strings.ToLower(strings.TrimSpace(k)) == strings.ToLower(strings.TrimSpace(conf.TraceUserHeaderTicket)) {
				clientID = v
				return
			}
		})
	}

	header.VisitAllCookie(func(key, value []byte) {
		for _, v := range conf.TraceUserCookieKeys {
			if strings.ToLower(strings.TrimSpace(string(key))) == strings.ToLower(strings.TrimSpace(string(v))) {
				clientID = string(v)
				return
			}
		}
	})

	// WhaTap Cookie name is constant WHATAP_COOKIE_NAME(WHATAP)
	header.VisitAllCookie(func(key, value []byte) {
		for _, v := range conf.TraceUserCookieKeys {
			if strings.ToUpper(strings.TrimSpace(string(key))) == trace.WHATAP_COOKIE_NAME {
				clientID = string(v)
				return
			}
		}
	})

	return clientID
}

func GetWhatapCookie(ctx *fasthttp.RequestCtx) (cookie *fasthttp.Cookie, exists bool) {
	ctx.Request.Header.VisitAllCookie(func(key, value []byte) {
		if string(key) == trace.WHATAP_COOKIE_NAME {
			cookie = new(fasthttp.Cookie)
			cookie.SetKey(string(key))
			cookie.SetValue(string(value))
			return
		}
	})
	if cookie == nil {
		cookie := new(fasthttp.Cookie)
		cookie.SetKey(trace.WHATAP_COOKIE_NAME)
		cookie.SetValue(fmt.Sprintf("%d", keygen.Next()))
		return cookie, false
	}
	return cookie, true
}

func SetWhatapCookie(ctx *fasthttp.RequestCtx, cookie *fasthttp.Cookie) {
	if ctx != nil && cookie != nil {
		ctx.Response.Header.SetCookie(cookie)
	}
}
