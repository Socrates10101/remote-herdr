package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func codexScreen(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func labels(d *Dialog) []string {
	var out []string
	for _, it := range d.Items {
		out = append(out, it.Label)
	}
	return out
}

func checkExecApproval(t *testing.T, name string) {
	t.Helper()
	d := ParseCodexDialog(codexScreen(t, name))
	if d == nil || d.Cursor != 0 {
		t.Fatalf("%s: dialog = %+v", name, d)
	}
	wantLabels := []string{
		"Yes, proceed (y)",
		"Yes, and don't ask again for commands that start with `echo hi > hello.txt` (p)",
		"No, and tell Codex what to do differently (esc)",
	}
	if !reflect.DeepEqual(labels(d), wantLabels) {
		t.Errorf("%s: labels = %q", name, labels(d))
	}
	for _, it := range d.Items {
		if it.Description != "" || it.FreeText {
			t.Errorf("%s: item = %+v", name, it)
		}
	}
	wantCtx := []string{
		"Would you like to run the following command?",
		"Environment: local",
		"Reason: The read-only sandbox blocked writing hello.txt. Allow running the command outside the sandbox?",
		"$ echo hi > hello.txt",
	}
	if !reflect.DeepEqual(d.Context, wantCtx) {
		t.Errorf("%s: context = %q", name, d.Context)
	}
}

func TestParseCodexExecApproval(t *testing.T) {
	checkExecApproval(t, "codex-exec-approval.txt")
	// The history prompt "› 1. my numbered prompt" is not an option.
	checkExecApproval(t, "codex-numbered-prompt.txt")
}

func TestParseCodexPatchApproval(t *testing.T) {
	d := ParseCodexDialog(codexScreen(t, "codex-patch-approval.txt"))
	if d == nil || len(d.Items) != 3 || d.Cursor != 0 {
		t.Fatalf("dialog = %+v", d)
	}
	if d.Items[1].Label != "Yes, and don't ask again for these files (a)" {
		t.Errorf("labels = %q", labels(d))
	}
	want := []string{
		"Would you like to make the following edits?",
		"Description: Apply proposed file edits",
		"Destination: /tmp/rh-codex-demo/README.md",
	}
	if !reflect.DeepEqual(d.Context, want) {
		t.Errorf("context = %q", d.Context)
	}
}

func TestParseCodexQuestion(t *testing.T) {
	d := ParseCodexDialog(codexScreen(t, "codex-question.txt"))
	if d == nil || d.Cursor != 0 {
		t.Fatalf("dialog = %+v", d)
	}
	if want := []string{"Red", "Green", "Blue", "None of the above"}; !reflect.DeepEqual(labels(d), want) {
		t.Errorf("labels = %q", labels(d))
	}
	descs := []string{"Choose red.", "Choose green.", "Choose blue.", "Optionally, add details in notes (tab)"}
	for i, it := range d.Items {
		if it.Description != descs[i] {
			t.Errorf("item %d description = %q, want %q", i, it.Description, descs[i])
		}
	}
	if want := []string{"Question 1/2 (2 unanswered)", "Pick a color"}; !reflect.DeepEqual(d.Context, want) {
		t.Errorf("context = %q", d.Context)
	}
	if got := strings.Join(d.KeysToChoose(2), ","); got != "down,down,enter" {
		t.Errorf("keys = %q", got)
	}
}

func TestParseCodexTrust(t *testing.T) {
	d := ParseCodexDialog(codexScreen(t, "codex-trust.txt"))
	if d == nil || d.Cursor != 0 {
		t.Fatalf("dialog = %+v", d)
	}
	if want := []string{"Trust and continue", "Back to Agent Command Center"}; !reflect.DeepEqual(labels(d), want) {
		t.Errorf("labels = %q", labels(d))
	}
	if len(d.Context) == 0 || d.Context[0] != "Folder access" || !strings.Contains(strings.Join(d.Context, "\n"), "/tmp/rh-codex-demo") {
		t.Errorf("context = %q", d.Context)
	}
}

func TestParseCodexNoDialog(t *testing.T) {
	for _, name := range []string{"codex-idle.txt", "codex-idle-numbered-prompt.txt"} {
		if d := ParseCodexDialog(codexScreen(t, name)); d != nil {
			t.Errorf("%s: dialog = %+v", name, d)
		}
	}
	noCursor := strings.Replace(codexScreen(t, "codex-exec-approval.txt"), "› 1.", "  1.", 1)
	if d := ParseCodexDialog(noCursor); d != nil {
		t.Errorf("no cursor: dialog = %+v", d)
	}
	// "center" and "description" contain "enter" and "esc" but are not a key hint.
	notHint := strings.Replace(codexScreen(t, "codex-trust.txt"), "enter continue · esc back", "Agent Command Center · see the description", 1)
	if d := ParseCodexDialog(notHint); d != nil {
		t.Errorf("substring hint: dialog = %+v", d)
	}
}

func TestLooksLikeCodexApproval(t *testing.T) {
	for name, want := range map[string]bool{
		"codex-exec-approval.txt":  true,
		"codex-patch-approval.txt": true,
		"codex-question.txt":       false,
		"codex-trust.txt":          false,
		"codex-idle.txt":           false,
	} {
		if got := LooksLikeCodexApproval(codexScreen(t, name)); got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}
