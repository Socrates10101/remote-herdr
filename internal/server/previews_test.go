package server

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"remote-herdr/internal/herdr"
	"remote-herdr/internal/previews"
)

const (
	previewOrigin = "https://box.tail.ts.net:8460"
	sidLive       = "11111111-1111-4111-8111-111111111111"
	sidOther      = "22222222-2222-4222-8222-222222222222"
	sidGone       = "33333333-3333-4333-8333-333333333333"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func jsonl(role, text, ts string) string {
	b, _ := json.Marshal(map[string]any{
		"type": role, "uuid": ts, "timestamp": ts,
		"message": map[string]any{"role": role, "content": text},
	})
	return string(b) + "\n"
}

func toolResult(text, ts string) string {
	b, _ := json.Marshal(map[string]any{
		"type": "user", "uuid": ts + "r", "timestamp": ts,
		"message": map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "tool_result", "content": text},
		}},
	})
	return string(b) + "\n"
}

type previewFixture struct {
	s    *Server
	dir  string
	proj string
}

func newPreviewTest(t *testing.T, ports ...int) previewFixture {
	t.Helper()
	dir, proj := t.TempDir(), t.TempDir()
	f := &fakeHerdr{agents: []herdr.Agent{
		{PaneID: "w1:p1", Agent: "claude", SessionID: sidLive, Title: "live", Cwd: t.TempDir()},
		{PaneID: "w1:p2", Agent: "claude", SessionID: sidOther, Title: "other", Cwd: t.TempDir()},
	}}
	s, err := New(Config{
		Herdr:           f,
		ProjectsDir:     proj,
		AllowedUsers:    map[string]bool{"me@example.com": true},
		Origin:          testOrigin,
		Logger:          log.New(io.Discard, "", 0),
		PreviewDir:      dir,
		PreviewOrigin:   previewOrigin + "/",
		PreviewAppPorts: ports,
	})
	if err != nil {
		t.Fatal(err)
	}
	return previewFixture{s, dir, proj}
}

type previewResp struct {
	Entries []struct {
		Project string        `json:"project"`
		Name    string        `json:"name"`
		URL     string        `json:"url"`
		Owner   *previewOwner `json:"owner"`
	} `json:"entries"`
	Apps []previewApp `json:"apps"`
}

func getPreviews(t *testing.T, s *Server, path string) previewResp {
	t.Helper()
	w := do(s, req(http.MethodGet, path, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: %d %s", path, w.Code, w.Body)
	}
	var r previewResp
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPreviewsDisabledByDefault(t *testing.T) {
	s, _ := newTest(t, "idle")
	for _, p := range []string{"/api/previews", "/api/agents/w1:p1/previews"} {
		if w := do(s, req(http.MethodGet, p, "")); w.Code == http.StatusOK {
			t.Errorf("%s: preview API must not exist when disabled", p)
		}
	}
	w := do(s, req(http.MethodGet, "/api/me", ""))
	if !strings.Contains(w.Body.String(), `"previews":false`) {
		t.Errorf("me = %s", w.Body)
	}
}

func TestPreviewConfigRejected(t *testing.T) {
	base := func() Config {
		return Config{Herdr: &fakeHerdr{}, AllowedUsers: map[string]bool{"a": true}, Origin: testOrigin}
	}
	cases := map[string]func(*Config){
		"dir without origin": func(c *Config) { c.PreviewDir = "/tmp" },
		"origin without dir": func(c *Config) { c.PreviewOrigin = previewOrigin },
		"http origin":        func(c *Config) { c.PreviewDir, c.PreviewOrigin = "/tmp", "http://box.tail.ts.net:8460" },
		"origin with path":   func(c *Config) { c.PreviewDir, c.PreviewOrigin = "/tmp", previewOrigin+"/x" },
		"same as app origin": func(c *Config) { c.PreviewDir, c.PreviewOrigin = "/tmp", testOrigin },
		"ports without dir":  func(c *Config) { c.PreviewAppPorts = []int{8461} },
		"port out of range": func(c *Config) {
			c.PreviewDir, c.PreviewOrigin, c.PreviewAppPorts = "/tmp", previewOrigin, []int{70000}
		},
	}
	for name, mod := range cases {
		c := base()
		mod(&c)
		if _, err := New(c); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestPreviewOwners(t *testing.T) {
	fx := newPreviewTest(t)
	// Marked folders: one by the live session, one by a session that ended.
	write(t, filepath.Join(fx.dir, "proj", "marked", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "marked", previews.MarkerName), sidLive+"\n")
	write(t, filepath.Join(fx.dir, "proj", "gone", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "gone", previews.MarkerName), sidGone+"\n")
	// Unmarked folders found through transcripts.
	write(t, filepath.Join(fx.dir, "proj", "said", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "both", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "pasted", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "orphan", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "put", "index.html"), "x")
	write(t, filepath.Join(fx.dir, "proj", "grepped", "index.html"), "x")
	// The marker wins over a later mention by another session.
	write(t, filepath.Join(fx.proj, "p", sidLive+".jsonl"),
		jsonl("assistant", previewOrigin+"/proj/said/ を見て", "2026-11-04T01:00:00Z")+
			jsonl("assistant", previewOrigin+"/proj/both/", "2026-11-04T01:00:00Z")+
			jsonl("user", previewOrigin+"/proj/pasted/", "2026-11-04T01:00:00Z")+
			toolResult(previewOrigin+"/proj/put/\n", "2026-11-04T01:00:00Z")+
			toolResult("  12 "+previewOrigin+"/proj/grepped/\n", "2026-11-04T01:00:00Z"))
	write(t, filepath.Join(fx.proj, "p", sidOther+".jsonl"),
		jsonl("assistant", previewOrigin+"/proj/both/ と "+previewOrigin+"/proj/marked/", "2026-11-04T02:00:00Z"))

	r := getPreviews(t, fx.s, "/api/previews")
	owners := map[string]*previewOwner{}
	for _, e := range r.Entries {
		owners[e.Name] = e.Owner
		if !strings.HasPrefix(e.URL, previewOrigin+"/proj/") {
			t.Errorf("url = %s", e.URL)
		}
	}
	check := func(name, sid, source, pane string) {
		t.Helper()
		o := owners[name]
		if sid == "" {
			if o != nil {
				t.Errorf("%s: owner = %+v, want none", name, o)
			}
			return
		}
		if o == nil || o.SessionID != sid || o.Source != source || o.PaneID != pane {
			t.Errorf("%s: owner = %+v, want %s/%s/%s", name, o, sid, source, pane)
		}
	}
	check("marked", sidLive, "marker", "w1:p1")
	check("gone", sidGone, "marker", "")
	check("said", sidLive, "transcript", "w1:p1")
	check("both", sidOther, "transcript", "w1:p2") // last mention wins
	check("pasted", "", "", "")                    // user text does not count
	check("orphan", "", "", "")
	check("put", sidLive, "transcript", "w1:p1") // printed alone by a script
	check("grepped", "", "", "")                 // inside other output

	mine := getPreviews(t, fx.s, "/api/agents/w1:p1/previews")
	var names []string
	for _, e := range mine.Entries {
		names = append(names, e.Name)
	}
	if len(names) != 3 || !(contains(names, "marked") && contains(names, "said") && contains(names, "put")) {
		t.Errorf("session previews = %v", names)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func TestPreviewApps(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	used := ln.Addr().(*net.TCPAddr).Port
	free, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	freePort := free.Addr().(*net.TCPAddr).Port
	free.Close()

	fx := newPreviewTest(t, used, freePort)
	r := getPreviews(t, fx.s, "/api/previews")
	if len(r.Apps) != 2 || !r.Apps[0].Listening || r.Apps[1].Listening {
		t.Fatalf("apps = %+v", r.Apps)
	}
	if r.Apps[0].URL != "https://box.tail.ts.net:"+strconv.Itoa(used)+"/" {
		t.Errorf("url = %s", r.Apps[0].URL)
	}
}
