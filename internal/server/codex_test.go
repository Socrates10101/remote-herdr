package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"remote-herdr/internal/herdr"
)

const codexSID = "01a1b0bd-ace3-7b12-bc7e-1124f9cd5906"

// codexItemLine is one item_completed rollout line.
func codexItemLine(t *testing.T, ts string, item map[string]any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"timestamp": ts, "type": "event_msg",
		"payload": map[string]any{"type": "item_completed", "item": item},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}

func codexRollout(t *testing.T, dir, sid string, lines ...string) {
	t.Helper()
	write(t, filepath.Join(dir, "2026", "11", "06", "rollout-2026-11-06T12-03-03-"+sid+".jsonl"), strings.Join(lines, ""))
}

func TestTranscriptCodex(t *testing.T) {
	s, f := newTest(t, "idle")
	s.cfg.CodexSessionsDir = t.TempDir()
	codexRollout(t, s.cfg.CodexSessionsDir, codexSID,
		codexItemLine(t, "t1", map[string]any{"type": "UserMessage", "id": "u1",
			"content": []any{map[string]any{"type": "text", "text": "codex hello"}}}),
		codexItemLine(t, "t2", map[string]any{"type": "AgentMessage", "id": "a1",
			"content": []any{map[string]any{"type": "Text", "text": "codex answer"}}}))
	write(t, filepath.Join(s.cfg.ProjectsDir, "p", codexSID+".jsonl"), jsonl("user", "claude hello", "t1"))

	get := func(agent string) string {
		f.agents[0].Agent, f.agents[0].SessionID = agent, codexSID
		w := do(s, req(http.MethodGet, "/api/agents/w1:p1/transcript", ""))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", agent, w.Code, w.Body)
		}
		return w.Body.String()
	}

	body := get("codex")
	if !strings.Contains(body, "codex hello") || !strings.Contains(body, "codex answer") || strings.Contains(body, "claude hello") {
		t.Errorf("codex agent body = %s", body)
	}
	body = get("claude")
	if !strings.Contains(body, "claude hello") || strings.Contains(body, "codex") {
		t.Errorf("claude agent body = %s", body)
	}
	if body = get("gemini"); !strings.Contains(body, `"size":-1`) {
		t.Errorf("other agent body = %s", body)
	}
	s.cfg.CodexSessionsDir = ""
	if body = get("codex"); !strings.Contains(body, `"size":-1`) {
		t.Errorf("codex without sessions dir body = %s", body)
	}
}

func TestCodexDialogAndChoose(t *testing.T) {
	s, f := newTest(t, "blocked")
	f.agents[0].Agent = "codex"
	f.screen = codexScreen(t, "codex-question.txt")

	w := do(s, req(http.MethodGet, "/api/agents/w1:p1/dialog", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("dialog: %d %s", w.Code, w.Body)
	}
	for _, l := range []string{"Red", "Green", "Blue", "None of the above", `"cursor":0`} {
		if !strings.Contains(w.Body.String(), l) {
			t.Errorf("dialog body lacks %q: %s", l, w.Body)
		}
	}

	w = do(s, req(http.MethodPost, "/api/agents/w1:p1/choose", `{"index":2,"label":"Blue"}`))
	if w.Code != http.StatusOK || strings.Join(f.keys, ",") != "down,down,enter" {
		t.Errorf("choose: %d %s keys=%v", w.Code, w.Body, f.keys)
	}

	f.keys = nil
	w = do(s, req(http.MethodPost, "/api/agents/w1:p1/choose", `{"index":1,"label":"Red"}`))
	if w.Code != http.StatusConflict || len(f.keys) != 0 {
		t.Errorf("stale choose: %d %s keys=%v", w.Code, w.Body, f.keys)
	}
}

func TestCodexDecision(t *testing.T) {
	s, f := newTest(t, "blocked")
	approve := func(agent, screen string) int {
		f.agents[0].Agent = agent
		f.screen = screen
		f.keys = nil
		return do(s, req(http.MethodPost, "/api/agents/w1:p1/decision", `{"decision":"approve"}`)).Code
	}
	if c := approve("codex", codexScreen(t, "codex-exec-approval.txt")); c != http.StatusOK || strings.Join(f.keys, ",") != "enter" {
		t.Errorf("codex exec approve: %d keys=%v", c, f.keys)
	}
	if c := approve("codex", codexScreen(t, "codex-question.txt")); c != http.StatusConflict || len(f.keys) != 0 {
		t.Errorf("codex question approve: %d keys=%v", c, f.keys)
	}
	if c := approve("codex", permission); c != http.StatusConflict || len(f.keys) != 0 {
		t.Errorf("codex with claude screen: %d keys=%v", c, f.keys)
	}
	if c := approve("claude", codexScreen(t, "codex-exec-approval.txt")); c != http.StatusConflict || len(f.keys) != 0 {
		t.Errorf("claude with codex screen: %d keys=%v", c, f.keys)
	}
}

func TestCodexPreviewOwner(t *testing.T) {
	fx := newPreviewTest(t)
	fx.s.cfg.CodexSessionsDir = t.TempDir()
	fx.s.cfg.Herdr.(*fakeHerdr).agents[1].Agent = "codex"
	codexRollout(t, fx.s.cfg.CodexSessionsDir, sidOther,
		codexItemLine(t, "2026-11-06T12:00:00Z", map[string]any{
			"type": "CommandExecution", "id": "c1",
			"command":           []string{"/bin/bash", "-lc", "publish"},
			"aggregated_output": previewOrigin + "/proj/codexed/\n",
			"status":            "completed", "exit_code": 0,
		}))
	write(t, filepath.Join(fx.dir, "proj", "codexed", "index.html"), "<p>x</p>")

	r := getPreviews(t, fx.s, "/api/previews")
	for _, e := range r.Entries {
		if e.Project != "proj" || e.Name != "codexed" {
			continue
		}
		o := e.Owner
		if o == nil || o.SessionID != sidOther || o.Source != "transcript" || o.PaneID != "w1:p2" {
			t.Errorf("owner = %+v", o)
		}
		return
	}
	t.Errorf("entry not found: %+v", r.Entries)
}

// steppedHerdr lets a test advance the SSE loop one agent listing at a time.
type steppedHerdr struct {
	Herdr
	step chan struct{}
}

func (h steppedHerdr) List(ctx context.Context) ([]herdr.Agent, error) {
	select {
	case <-h.step:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return h.Herdr.List(ctx)
}

// The event stream reports a Codex pane's transcript size, also when the
// rollout appears only after herdr has reported the session id.
func TestCodexEvents(t *testing.T) {
	s, f := newTest(t, "working")
	s.cfg.CodexSessionsDir = t.TempDir()
	s.cfg.PollInterval = time.Millisecond
	f.agents[0].Agent, f.agents[0].SessionID = "codex", codexSID
	step := make(chan struct{})
	s.cfg.Herdr = steppedHerdr{Herdr: f, step: step}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.ServeHTTP(w, req(http.MethodGet, "/api/events", "").WithContext(ctx))
	}()
	next := func() {
		t.Helper()
		select {
		case step <- struct{}{}:
		case <-done:
			t.Fatalf("event stream ended early: %s", w.Body)
		case <-time.After(5 * time.Second):
			t.Fatal("event stream stopped listing agents")
		}
	}

	next() // first state: herdr reports the id, the rollout does not exist yet
	codexRollout(t, s.cfg.CodexSessionsDir, codexSID,
		codexItemLine(t, "t1", map[string]any{"type": "AgentMessage", "id": "a1",
			"content": []any{map[string]any{"type": "Text", "text": "late"}}}))
	next() // second state: the rollout is found
	next() // the third listing starts only after the second state was written
	cancel()
	<-done

	var sizes []map[string]int64
	for _, ev := range strings.Split(w.Body.String(), "\n\n") {
		data, ok := strings.CutPrefix(ev, "event: state\ndata: ")
		if !ok {
			continue
		}
		var st struct {
			Sizes map[string]int64 `json:"sizes"`
		}
		if err := json.Unmarshal([]byte(data), &st); err != nil {
			t.Fatalf("bad state %q: %v", data, err)
		}
		sizes = append(sizes, st.Sizes)
	}
	fi, err := os.Stat(filepath.Join(s.cfg.CodexSessionsDir, "2026", "11", "06", "rollout-2026-11-06T12-03-03-"+codexSID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	key := s.cfg.HostName + "/w1:p1"
	if len(sizes) != 2 {
		t.Fatalf("got %d state events, want 2: %s", len(sizes), w.Body)
	}
	if _, ok := sizes[0][key]; ok {
		t.Errorf("size reported before the rollout existed: %v", sizes[0])
	}
	if got := sizes[1][key]; got != fi.Size() {
		t.Errorf("size = %d, want %d (%v)", got, fi.Size(), sizes[1])
	}
}
