package server

import (
	"net/http"
	"strings"
	"testing"
)

// Screens captured from Claude Code 2.1.310 (trimmed).
const singleSelect = `❯ Use the AskUserQuestion tool exactly once with two questions: (1) single-select 'Pick a color' with options Red,
  Green, Blue; (2) multiSelect true 'Pick fruits' with options Apple, Banana, Cherry. Do nothing else.
────────────────────────────────────────────────────────────────────────────────
←  ☐ Color  ☐ Fruits  ✔ Submit  →
Pick a color
❯ 1. Red
     Red
  2. Green
     Green
  3. Blue
     Blue
  4. Type something.
────────────────────────────────────────────────────────────────────────────────
  5. Chat about this
Enter to select · Tab/Arrow keys to navigate · Esc to cancel
`

const multiSelect = `❯ Use the AskUserQuestion tool exactly once
────────────────────────────────────────────────────────────────────────────────
←  ☒ Color  ☐ Fruits  ✔ Submit  →
Pick fruits
  1. [✔] Apple
         Apple
  2. [ ] Banana
         Banana
❯ 3. [ ] Cherry
         Cherry
  4. [ ] Type something
     Submit
────────────────────────────────────────────────────────────────────────────────
  5. Chat about this
Enter to select · Tab/Arrow keys to navigate · Esc to cancel
`

const permission = `❯ Create the file /tmp/rh-tui/hello.txt containing hi
  ⎿  $ echo hi > /tmp/rh-tui/hello.txt
────────────────────────────────────────────────────────────────────────────────
 Bash command
 Create hello.txt containing hi and print it
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
 echo hi > /tmp/rh-tui/hello.txt
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
 Do you want to proceed?
 ❯ 1. Yes
   2. Yes, and always allow access to /tmp/rh-tui from this project
   3. Yes, and switch to auto mode · auto mode handles these prompts for you
   4. No
 Esc to cancel · Tab to amend
`

func TestParseSingleSelect(t *testing.T) {
	d := ParseDialog(singleSelect)
	if d == nil || len(d.Items) != 5 || d.Cursor != 0 {
		t.Fatalf("dialog = %+v", d)
	}
	if d.Items[1].Label != "Green" || d.Items[1].Description != "" || !d.Items[3].FreeText {
		t.Errorf("items = %+v", d.Items)
	}
	if strings.Join(d.Context, "|") != "←  ☐ Color  ☐ Fruits  ✔ Submit  →|Pick a color" {
		t.Errorf("context = %q", d.Context)
	}
	if got := strings.Join(d.KeysToChoose(2), ","); got != "down,down,enter" {
		t.Errorf("keys = %s", got)
	}
}

func TestParseMultiSelect(t *testing.T) {
	d := ParseDialog(multiSelect)
	if d == nil || len(d.Items) != 6 || d.Cursor != 2 {
		t.Fatalf("dialog = %+v", d)
	}
	if d.Items[0].Checked == nil || !*d.Items[0].Checked || *d.Items[1].Checked {
		t.Errorf("checkboxes = %+v %+v", d.Items[0], d.Items[1])
	}
	if d.Items[4].Label != "Submit" || d.Items[4].Number != "" {
		t.Errorf("submit = %+v", d.Items[4])
	}
	if got := strings.Join(d.KeysToChoose(0), ","); got != "up,up,enter" {
		t.Errorf("keys = %s", got)
	}
}

func TestParsePermission(t *testing.T) {
	d := ParseDialog(permission)
	if d == nil || len(d.Items) != 4 || d.Cursor != 0 || d.Items[3].Label != "No" {
		t.Fatalf("dialog = %+v", d)
	}
	if d.Context[0] != "Bash command" || d.Context[len(d.Context)-1] != "Do you want to proceed?" {
		t.Errorf("context = %q", d.Context)
	}
}

func TestParseIgnoresPromptHistory(t *testing.T) {
	screen := "❯ 1. my numbered prompt\n  ⎿ output\n" + singleSelect
	d := ParseDialog(screen)
	if d == nil || len(d.Items) != 5 || d.Items[0].Label != "Red" || d.Cursor != 0 {
		t.Fatalf("prompt history leaked into dialog: %+v", d)
	}
	if d := ParseDialog("just some output\n❯ hello\n"); d != nil {
		t.Errorf("unexpected dialog %+v", d)
	}
}

func TestChoose(t *testing.T) {
	s, f := newTest(t, "blocked")
	f.screen = singleSelect
	if w := do(s, req("POST", "/api/agents/w1:p1/choose", `{"index":2,"label":"Blue"}`)); w.Code != 200 {
		t.Fatalf("choose: %d %s", w.Code, w.Body)
	}
	if strings.Join(f.keys, ",") != "down,down,enter" {
		t.Fatalf("keys = %v", f.keys)
	}
	f.keys = nil
	for name, body := range map[string]string{
		"stale label":  `{"index":2,"label":"Green"}`,
		"free text":    `{"index":3,"label":"Type something."}`,
		"out of range": `{"index":9,"label":"x"}`,
	} {
		if w := do(s, req("POST", "/api/agents/w1:p1/choose", body)); w.Code != http.StatusConflict {
			t.Errorf("%s: got %d", name, w.Code)
		}
	}
	if len(f.keys) != 0 {
		t.Fatalf("keys sent: %v", f.keys)
	}
}

func TestChooseRequiresBlocked(t *testing.T) {
	s, f := newTest(t, "idle")
	f.screen = singleSelect
	if w := do(s, req("POST", "/api/agents/w1:p1/choose", `{"index":0,"label":"Red"}`)); w.Code != http.StatusConflict {
		t.Fatalf("got %d", w.Code)
	}
}
