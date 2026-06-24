package llm

import (
	"strings"
	"sync"
)

// URLMatch describes a successful auto-detection of an LLM API call.
type URLMatch struct {
	Provider      string // host name, e.g. "api.openai.com"
	OperationType string // "chat" / "messages" / "embeddings" / ...
	URL           string // original URL passed in
}

// hostPattern — single host's known path → operation_type table.
type hostPattern struct {
	host    string
	opPaths []pathOp
}

type pathOp struct {
	pathPrefix string // matched if the request path starts with this
	op         string
}

// builtinHosts — well-known LLM provider URL patterns. host comparison is
// case-insensitive; path matching is prefix-based so versioned paths
// (`/v1/chat/completions/...`) still resolve.
var builtinHosts = []hostPattern{
	{host: "api.openai.com", opPaths: []pathOp{
		{"/v1/chat/completions", "chat"},
		{"/v1/responses", "responses"},
		{"/v1/embeddings", "embeddings"},
		{"/v1/completions", "completions"},
	}},
	{host: "api.anthropic.com", opPaths: []pathOp{
		{"/v1/messages", "messages"},
		{"/v1/complete", "completions"},
	}},
	{host: "generativelanguage.googleapis.com", opPaths: []pathOp{
		{"/v1beta/models", "chat"},
		{"/v1/models", "chat"},
	}},
	{host: "api.x.ai", opPaths: []pathOp{
		{"/v1/chat/completions", "chat"},
	}},
	{host: "api.cohere.com", opPaths: []pathOp{
		{"/v1/chat", "chat"},
		{"/v1/embed", "embeddings"},
		{"/v1/generate", "completions"},
	}},
	{host: "api.together.xyz", opPaths: []pathOp{
		{"/v1/chat/completions", "chat"},
		{"/v1/completions", "completions"},
		{"/v1/embeddings", "embeddings"},
	}},
	{host: "api.mistral.ai", opPaths: []pathOp{
		{"/v1/chat/completions", "chat"},
		{"/v1/embeddings", "embeddings"},
	}},
	{host: "api.groq.com", opPaths: []pathOp{
		{"/openai/v1/chat/completions", "chat"},
	}},
	{host: "api.deepseek.com", opPaths: []pathOp{
		{"/v1/chat/completions", "chat"},
	}},
	{host: "api.perplexity.ai", opPaths: []pathOp{
		{"/chat/completions", "chat"},
	}},
}

var (
	customHosts   []hostPattern
	customHostsMu sync.RWMutex
)

// AddHostPattern registers an additional host (e.g. self-hosted Ollama at
// "ollama.internal") with default operation_type "chat" if no path map is
// supplied. Concurrency-safe.
func AddHostPattern(host string, opByPath map[string]string) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return
	}
	hp := hostPattern{host: host}
	if len(opByPath) == 0 {
		hp.opPaths = []pathOp{{"", "chat"}}
	} else {
		for p, op := range opByPath {
			hp.opPaths = append(hp.opPaths, pathOp{p, op})
		}
	}
	customHostsMu.Lock()
	customHosts = append(customHosts, hp)
	customHostsMu.Unlock()
}

// MatchURL looks up url against builtin + custom host tables. Returns
// (URLMatch, true) on hit, otherwise (zero, false).
func MatchURL(url string) (URLMatch, bool) {
	if url == "" {
		return URLMatch{}, false
	}
	host, path := splitHostPath(url)
	if host == "" {
		return URLMatch{}, false
	}
	hostLower := strings.ToLower(host)
	if m, ok := matchInTable(builtinHosts, hostLower, path); ok {
		m.URL = url
		return m, true
	}
	customHostsMu.RLock()
	defer customHostsMu.RUnlock()
	if m, ok := matchInTable(customHosts, hostLower, path); ok {
		m.URL = url
		return m, true
	}
	return URLMatch{}, false
}

func matchInTable(tbl []hostPattern, host, path string) (URLMatch, bool) {
	for _, hp := range tbl {
		if hp.host != host {
			continue
		}
		for _, po := range hp.opPaths {
			if po.pathPrefix == "" || strings.HasPrefix(path, po.pathPrefix) {
				return URLMatch{Provider: hp.host, OperationType: po.op}, true
			}
		}
		// host matched but no path matched — still flag as LLM with default op
		return URLMatch{Provider: hp.host, OperationType: "unknown"}, true
	}
	return URLMatch{}, false
}

// splitHostPath — "https://api.openai.com/v1/chat" → ("api.openai.com", "/v1/chat")
func splitHostPath(url string) (host, path string) {
	s := url
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i], s[i:]
	}
	return s, ""
}
