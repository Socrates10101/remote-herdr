package transcript

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const codexID = "01a1b0bd-ace3-7b12-bc7e-1124f9cd5906"

func loadRollout(t *testing.T) []Message {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "codex-rollout.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return ParseCodex(data)
}

func TestParseCodex(t *testing.T) {
	msgs := loadRollout(t)
	wantIDs := []string{"u1", "msg_1", "exec-1", "exec-2", "exec-3", "exec-4", "ws-1", "ws-2", "ig-1", "iv-1", "mcp-1", "plan-1"}
	if len(msgs) != 13 {
		t.Fatalf("got %d messages: %+v", len(msgs), msgs)
	}
	for i, id := range wantIDs {
		if msgs[i].UUID != id {
			t.Errorf("message %d uuid = %q, want %q", i, msgs[i].UUID, id)
		}
	}
	seen := map[string]bool{}
	for _, m := range msgs {
		if seen[m.UUID] {
			t.Errorf("duplicate uuid %q", m.UUID)
		}
		seen[m.UUID] = true
	}

	// 1: user text only.
	if m := msgs[0]; m.Role != "user" || len(m.Blocks) != 1 || m.Blocks[0].Text != `Explain $e^{i\pi}$ and create hello.txt` {
		t.Errorf("user message = %+v", m)
	}
	// 2: "Text" content type is accepted.
	if m := msgs[1]; m.Role != "assistant" || len(m.Blocks) != 1 || m.Blocks[0].Text != "Sure: \\(e^{i\\pi}=-1\\). Creating the file.\n" {
		t.Errorf("agent message = %+v", m)
	}
	// 3: silent successful command, input not HTML-escaped.
	b := msgs[2].Blocks
	if len(b) != 1 || b[0].Kind != "tool_use" || b[0].Name != "Shell" {
		t.Fatalf("exec-1 blocks = %+v", b)
	}
	if !strings.Contains(b[0].Input, `"command": "echo hi > hello.txt"`) || strings.Contains(b[0].Input, "\\u00"+"3e") {
		t.Errorf("exec-1 input = %q", b[0].Input)
	}
	// 4: failed with output.
	b = msgs[3].Blocks
	if len(b) != 2 || b[0].Name != "Shell" || !strings.Contains(b[0].Input, "cat missing.txt") ||
		b[1].Kind != "tool_result" || b[1].Text != "cat: missing.txt: No such file or directory\n" || !b[1].IsError {
		t.Errorf("exec-2 blocks = %+v", b)
	}
	// 5: argv joined, exit code as the text.
	b = msgs[4].Blocks
	if len(b) != 2 || !strings.Contains(b[0].Input, `"git status"`) || b[1].Text != "exit code 128" || !b[1].IsError {
		t.Errorf("exec-3 blocks = %+v", b)
	}
	// 6: file changes in sorted path order.
	b = msgs[5].Blocks
	if len(b) != 5 {
		t.Fatalf("exec-4 blocks = %+v", b)
	}
	for i, want := range map[int]string{0: "/tmp/demo/README.md", 2: "/tmp/demo/new.txt", 4: "/tmp/demo/old.txt"} {
		if b[i].Kind != "tool_use" || b[i].Name != "Edit" || !strings.Contains(b[i].Input, want) {
			t.Errorf("exec-4 block %d = %+v", i, b[i])
		}
	}
	if !strings.Contains(b[0].Input, `"update"`) || !strings.Contains(b[1].Text, "+from codex") ||
		!strings.Contains(b[2].Input, `"add"`) || b[3].Text != "new file\n" || !strings.Contains(b[4].Input, `"delete"`) {
		t.Errorf("exec-4 blocks = %+v", b)
	}
	// 7, 8: web search.
	if b = msgs[6].Blocks; len(b) != 1 || b[0].Name != "WebSearch" || !strings.Contains(b[0].Input, `"query": "herdr socket api"`) || strings.Contains(b[0].Input, `"url"`) {
		t.Errorf("ws-1 blocks = %+v", b)
	}
	if b = msgs[7].Blocks; len(b) != 1 || b[0].Name != "WebSearch" || !strings.Contains(b[0].Input, `"url": "https://example.com/doc"`) {
		t.Errorf("ws-2 blocks = %+v", b)
	}
	// 9: image generation (result is a string here).
	if b = msgs[8].Blocks; len(b) != 1 || b[0].Name != "ImageGen" ||
		!strings.Contains(b[0].Input, `"prompt": "A small violet star"`) || !strings.Contains(b[0].Input, `"path": "/tmp/demo/star.png"`) {
		t.Errorf("ig-1 blocks = %+v", b)
	}
	// 10: file:// URL turned into a path.
	if b = msgs[9].Blocks; len(b) != 1 || b[0].Name != "ViewImage" || !strings.Contains(b[0].Input, `"path": "/tmp/demo/dot one.png"`) {
		t.Errorf("iv-1 blocks = %+v", b)
	}
	// 11: MCP call.
	b = msgs[10].Blocks
	if len(b) != 2 || b[0].Name != "docs.search" || !strings.Contains(b[0].Input, `"query"`) ||
		b[1].Kind != "tool_result" || b[1].Text != "no results" || b[1].IsError {
		t.Errorf("mcp-1 blocks = %+v", b)
	}
	// 12, 13: plan and final answer.
	if b = msgs[11].Blocks; len(b) != 1 || b[0].Text != "## Plan\n\n1. Do it" {
		t.Errorf("plan blocks = %+v", b)
	}
	last := msgs[12]
	if len(last.Blocks) != 1 || last.Blocks[0].Text != "Done. Created `hello.txt`." || last.UUID == "msg_1" {
		t.Errorf("final message = %+v", last)
	}
}

func TestLoadCodex(t *testing.T) {
	path := filepath.Join("testdata", "codex-rollout.jsonl")
	msgs, size, err := LoadCodex(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	if size != fi.Size() {
		t.Errorf("size = %d, want %d", size, fi.Size())
	}
	if len(msgs) != 2 || msgs[0].UUID != "plan-1" || !strings.HasPrefix(msgs[1].Blocks[0].Text, "Done.") {
		t.Errorf("messages = %+v", msgs)
	}
}

func TestLocateCodex(t *testing.T) {
	dir := t.TempDir()
	day := filepath.Join(dir, "2026", "11", "06")
	if err := os.MkdirAll(day, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "rollout-2026-11-06T12-03-03-" + codexID + ".jsonl"
	want := filepath.Join(day, name)
	if err := os.WriteFile(want, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LocateCodex(dir, codexID)
	if err != nil || got != want {
		t.Fatalf("LocateCodex = %q, %v", got, err)
	}
	if _, err := LocateCodex(dir, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing id error = %v", err)
	}
	if _, err := LocateCodex("", codexID); !errors.Is(err, ErrNotFound) {
		t.Errorf("empty dir error = %v", err)
	}
	// A relative dir must not resolve against the process cwd, even when a
	// matching log exists there.
	t.Chdir(dir)
	if _, err := LocateCodex(".", codexID); !errors.Is(err, ErrNotFound) {
		t.Errorf("relative dir error = %v", err)
	}
	if _, err := LocateCodex(dir, "../../etc/passwd"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("invalid id error = %v", err)
	}

	// Two dated copies: the newest wins.
	other := filepath.Join(dir, "2026", "11", "07", name)
	if err := os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(want, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(other, now, now); err != nil {
		t.Fatal(err)
	}
	if got, _ := LocateCodex(dir, codexID); got != other {
		t.Errorf("newest = %q, want %q", got, other)
	}
}
