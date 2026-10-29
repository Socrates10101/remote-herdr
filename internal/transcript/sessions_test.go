package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const deadPID = 1 << 30 // above Linux pid_max

func writeSession(t *testing.T, dir string, pid int, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", pid)), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLiveBackground(t *testing.T) {
	dir := t.TempDir()
	me := os.Getpid()
	writeSession(t, dir, me, fmt.Sprintf(`{"pid":%d,"sessionId":"def2e8ec-c8de-4c6b-aeca-bb60277d3200","cwd":"/w","kind":"bg","name":"fork ⑂"}`, me))
	writeSession(t, dir, me+1, fmt.Sprintf(`{"pid":%d,"sessionId":"11111111-1111-1111-1111-111111111111","cwd":"/w","kind":"interactive","name":"main"}`, me))
	writeSession(t, dir, deadPID, fmt.Sprintf(`{"pid":%d,"sessionId":"22222222-2222-2222-2222-222222222222","cwd":"/w","kind":"bg","name":"gone"}`, deadPID))
	writeSession(t, dir, me+2, fmt.Sprintf(`{"pid":%d,"sessionId":"../../etc/passwd","cwd":"/w","kind":"bg","name":"bad"}`, me))
	writeSession(t, dir, me+3, `not json`)

	got := LiveBackground(dir)
	if len(got) != 1 || got[0].ID != "def2e8ec-c8de-4c6b-aeca-bb60277d3200" || got[0].Name != "fork ⑂" {
		t.Fatalf("got %+v", got)
	}
	if LiveBackground(filepath.Join(dir, "missing")) != nil {
		t.Error("missing dir should yield nothing")
	}
}

func TestMatchBackground(t *testing.T) {
	bg := []Background{
		{ID: "a", Name: "Agent 現状把握", Cwd: "/w"},
		{ID: "b", Name: "Agent 現状把握 ⑂", Cwd: "/w"},
		{ID: "c", Name: "dup", Cwd: "/w"},
		{ID: "d", Name: "dup", Cwd: "/w"},
		{ID: "e", Name: "dup", Cwd: "/other"},
	}
	cases := []struct{ title, cwd, want string }{
		{"Agent 現状把握 ⑂", "/w", "b"},
		{"Agent 現状把握", "/w", "a"},
		{"Agent 現状把握", "/other", ""},
		{"dup", "/w", ""}, // ambiguous
		{"dup", "/other", "e"},
		{"", "/w", ""},
	}
	for _, c := range cases {
		if got := MatchBackground(bg, c.title, c.cwd); got != c.want {
			t.Errorf("MatchBackground(%q, %q) = %q, want %q", c.title, c.cwd, got, c.want)
		}
	}
}
