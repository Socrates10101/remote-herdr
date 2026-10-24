package server

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"remote-herdr/internal/herdr"
)

type fakeHerdr struct {
	agents  []herdr.Agent
	screen  string
	prompts []string
	keys    []string
}

func (f *fakeHerdr) List(context.Context) ([]herdr.Agent, error) { return f.agents, nil }
func (f *fakeHerdr) Read(context.Context, string, string, int) (string, error) {
	return f.screen, nil
}
func (f *fakeHerdr) Prompt(_ context.Context, _ string, text string) error {
	f.prompts = append(f.prompts, text)
	return nil
}
func (f *fakeHerdr) SendKeys(_ context.Context, _ string, keys ...string) error {
	f.keys = append(f.keys, keys...)
	return nil
}

const testOrigin = "https://box.tail.ts.net:8450"

func newTest(t *testing.T, status string) (*Server, *fakeHerdr) {
	t.Helper()
	f := &fakeHerdr{agents: []herdr.Agent{{PaneID: "w1:p1", Status: status, Cwd: t.TempDir()}}}
	s, err := New(Config{
		Herdr:        f,
		ProjectsDir:  t.TempDir(),
		AllowedUsers: map[string]bool{"me@example.com": true},
		Origin:       testOrigin,
		Logger:       log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, f
}

func req(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "box.tail.ts.net:8450"
	r.Header.Set("Tailscale-User-Login", "me@example.com")
	if method == http.MethodPost {
		r.Header.Set("Origin", testOrigin)
		r.Header.Set("X-Remote-Herdr", "1")
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func do(s *Server, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

func TestNewRejectsUnsafeConfig(t *testing.T) {
	f := &fakeHerdr{}
	if _, err := New(Config{Herdr: f, Origin: testOrigin}); err == nil {
		t.Error("expected error without allowed users")
	}
	if _, err := New(Config{Herdr: f, Origin: "http://x", AllowedUsers: map[string]bool{"a": true}}); err == nil {
		t.Error("expected error for non-https origin")
	}
}

func TestAuthAndHost(t *testing.T) {
	s, _ := newTest(t, "idle")
	if w := do(s, req("GET", "/api/agents", "")); w.Code != 200 {
		t.Fatalf("allowed user: got %d", w.Code)
	}
	r := req("GET", "/api/agents", "")
	r.Header.Set("Tailscale-User-Login", "other@example.com")
	if w := do(s, r); w.Code != 403 {
		t.Errorf("other user: got %d", w.Code)
	}
	r = req("GET", "/api/agents", "")
	r.Header.Del("Tailscale-User-Login")
	if w := do(s, r); w.Code != 403 {
		t.Errorf("missing user: got %d", w.Code)
	}
	r = req("GET", "/api/agents", "")
	r.Host = "evil.example"
	if w := do(s, r); w.Code != http.StatusMisdirectedRequest {
		t.Errorf("bad host: got %d", w.Code)
	}
	r = req("GET", "/api/agents", "")
	r.Host = "localhost" // tailscale serve -> unix socket
	if w := do(s, r); w.Code != 200 {
		t.Errorf("localhost host: got %d", w.Code)
	}
	r = req("GET", "/api/agents", "")
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	if w := do(s, r); w.Code != 403 {
		t.Errorf("cross-site fetch: got %d", w.Code)
	}
}

func TestPostCSRF(t *testing.T) {
	s, f := newTest(t, "idle")
	body := `{"text":"hello"}`
	for name, mutate := range map[string]func(*http.Request){
		"wrong origin":   func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
		"no origin":      func(r *http.Request) { r.Header.Del("Origin") },
		"no csrf header": func(r *http.Request) { r.Header.Del("X-Remote-Herdr") },
		"form post":      func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") },
	} {
		r := req("POST", "/api/agents/w1:p1/prompt", body)
		mutate(r)
		if w := do(s, r); w.Code != 403 {
			t.Errorf("%s: got %d", name, w.Code)
		}
	}
	if len(f.prompts) != 0 {
		t.Fatalf("prompt sent despite csrf failure: %v", f.prompts)
	}
	if w := do(s, req("POST", "/api/agents/w1:p1/prompt", body)); w.Code != 200 {
		t.Fatalf("valid prompt: got %d %s", w.Code, w.Body)
	}
	if len(f.prompts) != 1 || f.prompts[0] != "hello" {
		t.Fatalf("prompts = %v", f.prompts)
	}
}

func TestPromptRejectsShellAndSlash(t *testing.T) {
	s, f := newTest(t, "idle")
	for _, text := range []string{"!rm -rf ~", "  !ls", "/clear", "a\x1b[201~b", ""} {
		b := `{"text":` + jsonString(text) + `}`
		if w := do(s, req("POST", "/api/agents/w1:p1/prompt", b)); w.Code != http.StatusUnprocessableEntity {
			t.Errorf("%q: got %d", text, w.Code)
		}
	}
	if len(f.prompts) != 0 {
		t.Fatalf("unexpected prompts %v", f.prompts)
	}
}

func TestPromptRefusedWhileBlocked(t *testing.T) {
	s, f := newTest(t, "blocked")
	if w := do(s, req("POST", "/api/agents/w1:p1/prompt", `{"text":"hi"}`)); w.Code != http.StatusConflict {
		t.Fatalf("got %d", w.Code)
	}
	if len(f.prompts) != 0 {
		t.Fatal("prompt sent")
	}
}

func TestDecision(t *testing.T) {
	s, f := newTest(t, "blocked")
	f.screen = "Do you want to proceed?\n ❯ 1. Yes\n   2. Yes, and don't ask again\n   3. No (esc)\n"
	if w := do(s, req("POST", "/api/agents/w1:p1/decision", `{"decision":"approve"}`)); w.Code != 200 {
		t.Fatalf("approve: %d %s", w.Code, w.Body)
	}
	if w := do(s, req("POST", "/api/agents/w1:p1/decision", `{"decision":"deny"}`)); w.Code != 200 {
		t.Fatalf("deny: %d", w.Code)
	}
	if strings.Join(f.keys, ",") != "enter,esc" {
		t.Fatalf("keys = %v", f.keys)
	}

	f.keys = nil
	f.screen = "Which option?\n ❯ 1. Option A\n   2. Option B\n"
	if w := do(s, req("POST", "/api/agents/w1:p1/decision", `{"decision":"approve"}`)); w.Code != http.StatusConflict {
		t.Fatalf("unknown dialog: got %d", w.Code)
	}
	if len(f.keys) != 0 {
		t.Fatalf("keys sent for unknown dialog: %v", f.keys)
	}
}

func TestDecisionRequiresBlocked(t *testing.T) {
	s, f := newTest(t, "idle")
	if w := do(s, req("POST", "/api/agents/w1:p1/decision", `{"decision":"deny"}`)); w.Code != http.StatusConflict {
		t.Fatalf("got %d", w.Code)
	}
	if len(f.keys) != 0 {
		t.Fatal("keys sent")
	}
}

func TestUnknownPaneAndBadID(t *testing.T) {
	s, _ := newTest(t, "idle")
	if w := do(s, req("GET", "/api/agents/w9:p9/screen", "")); w.Code != 404 {
		t.Errorf("unknown pane: %d", w.Code)
	}
	if w := do(s, req("GET", "/api/agents/..%2fx/screen", "")); w.Code == 200 {
		t.Errorf("bad pane id accepted")
	}
}

func TestNoWriteEndpoints(t *testing.T) {
	s, _ := newTest(t, "idle")
	for _, p := range []string{"/api/agents/w1:p1/file", "/api/agents/w1:p1/files", "/api/agents/w1:p1/exec", "/api/agents/w1:p1/run"} {
		if w := do(s, req("POST", p, `{}`)); w.Code != http.StatusMethodNotAllowed && w.Code != 404 {
			t.Errorf("POST %s: got %d", p, w.Code)
		}
	}
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte("0123456789abcdef"[r>>4])
			b.WriteByte("0123456789abcdef"[r&0xf])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func TestAgentsCarryHost(t *testing.T) {
	s, _ := newTest(t, "idle")
	w := do(s, req("GET", "/api/agents", ""))
	if !strings.Contains(w.Body.String(), `"host":"`+s.cfg.HostName+`"`) || s.cfg.HostName == "" {
		t.Fatalf("host missing: %s", w.Body)
	}
}
