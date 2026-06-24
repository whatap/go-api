package llm

import "testing"

func TestMatchURL_BuiltinHosts(t *testing.T) {
	cases := []struct {
		url      string
		wantProv string
		wantOp   string
	}{
		{"https://api.openai.com/v1/chat/completions", "api.openai.com", "chat"},
		{"https://api.openai.com/v1/embeddings", "api.openai.com", "embeddings"},
		{"https://api.openai.com/v1/responses", "api.openai.com", "responses"},
		{"https://api.anthropic.com/v1/messages", "api.anthropic.com", "messages"},
		{"https://api.x.ai/v1/chat/completions", "api.x.ai", "chat"},
		{"https://api.cohere.com/v1/chat", "api.cohere.com", "chat"},
		{"https://api.mistral.ai/v1/chat/completions", "api.mistral.ai", "chat"},
		{"https://api.groq.com/openai/v1/chat/completions", "api.groq.com", "chat"},
	}
	for _, c := range cases {
		m, ok := MatchURL(c.url)
		if !ok {
			t.Fatalf("MatchURL(%q) should hit", c.url)
		}
		if m.Provider != c.wantProv {
			t.Fatalf("MatchURL(%q).Provider: want %q got %q", c.url, c.wantProv, m.Provider)
		}
		if m.OperationType != c.wantOp {
			t.Fatalf("MatchURL(%q).OperationType: want %q got %q", c.url, c.wantOp, m.OperationType)
		}
	}
}

func TestMatchURL_HostMatchedPathUnknown(t *testing.T) {
	// host known, path unknown — should still flag as LLM with op="unknown"
	m, ok := MatchURL("https://api.openai.com/v999/something/new")
	if !ok {
		t.Fatalf("known host should hit even with unknown path")
	}
	if m.Provider != "api.openai.com" {
		t.Fatalf("Provider: %q", m.Provider)
	}
	if m.OperationType != "unknown" {
		t.Fatalf("OperationType: want 'unknown' got %q", m.OperationType)
	}
}

func TestMatchURL_UnknownHost(t *testing.T) {
	// arbitrary host — must not false-positive
	cases := []string{
		"https://example.com/v1/chat/completions",
		"http://localhost:11434/api/chat",
		"https://internal.svc/api/v1/embed",
		"",
	}
	for _, url := range cases {
		if _, ok := MatchURL(url); ok {
			t.Fatalf("MatchURL(%q) should not match", url)
		}
	}
}

func TestAddHostPattern_Default(t *testing.T) {
	t.Cleanup(resetCustomHosts)
	AddHostPattern("ollama.internal", nil)

	m, ok := MatchURL("http://ollama.internal/api/chat")
	if !ok {
		t.Fatalf("custom host should match")
	}
	if m.Provider != "ollama.internal" || m.OperationType != "chat" {
		t.Fatalf("custom host result: %+v", m)
	}
}

func TestAddHostPattern_Custom(t *testing.T) {
	t.Cleanup(resetCustomHosts)
	AddHostPattern("my-proxy.example.com", map[string]string{
		"/v1/embed": "embeddings",
		"/v1/chat":  "chat",
	})

	m, ok := MatchURL("https://my-proxy.example.com/v1/embed")
	if !ok || m.OperationType != "embeddings" {
		t.Fatalf("embed path: %+v ok=%v", m, ok)
	}
	m, ok = MatchURL("https://my-proxy.example.com/v1/chat")
	if !ok || m.OperationType != "chat" {
		t.Fatalf("chat path: %+v ok=%v", m, ok)
	}
}

func TestAddHostPattern_CaseInsensitive(t *testing.T) {
	t.Cleanup(resetCustomHosts)
	AddHostPattern("Custom.LLM.Host", nil)
	if _, ok := MatchURL("https://custom.llm.host/api"); !ok {
		t.Fatalf("host comparison must be case-insensitive")
	}
}

func TestSplitHostPath(t *testing.T) {
	cases := []struct {
		in   string
		host string
		path string
	}{
		{"https://api.openai.com/v1/chat", "api.openai.com", "/v1/chat"},
		{"http://localhost:8080/api", "localhost:8080", "/api"},
		{"api.openai.com/v1/chat", "api.openai.com", "/v1/chat"},
		{"api.openai.com", "api.openai.com", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		host, path := splitHostPath(c.in)
		if host != c.host || path != c.path {
			t.Fatalf("splitHostPath(%q): want (%q, %q), got (%q, %q)", c.in, c.host, c.path, host, path)
		}
	}
}

func resetCustomHosts() {
	customHostsMu.Lock()
	customHosts = nil
	customHostsMu.Unlock()
}
