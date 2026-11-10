package server

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

// MaxPromptChars bounds prompt size.
const MaxPromptChars = 20_000

var paneIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}:[A-Za-z0-9_-]{1,32}$`)

// ValidPaneID reports whether s looks like a herdr pane id (e.g. "wN:p2").
func ValidPaneID(s string) bool { return paneIDPattern.MatchString(s) }

// ValidatePrompt enforces what a remote prompt may contain.
//
//   - Leading "!" is refused unless allowShell is set: it is the shell mode
//     of Claude Code and Codex, which run it as a shell command without a
//     permission prompt.
//   - Leading "/" is refused unless the command is allow-listed.
//   - Control characters other than newline and tab are refused so text
//     cannot inject terminal escape or bracketed-paste sequences.
func ValidatePrompt(text string, allowedSlash map[string]bool, allowShell bool) error {
	if !utf8.ValidString(text) {
		return errors.New("prompt is not valid UTF-8")
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return errors.New("prompt is empty")
	}
	if utf8.RuneCountInString(text) > MaxPromptChars {
		return errors.New("prompt is too long")
	}
	for _, r := range text {
		if r == '\n' || r == '\t' {
			continue
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return errors.New("prompt contains control characters")
		}
	}
	if strings.HasPrefix(trimmed, "!") && !allowShell {
		return errors.New(`prompts starting with "!" run shell commands and are not allowed`)
	}
	if strings.HasPrefix(trimmed, "/") {
		cmd := strings.Fields(trimmed)[0]
		if !allowedSlash[cmd] {
			return errors.New("this slash command is not allowed remotely")
		}
	}
	return nil
}

// approvePattern matches Claude Code's permission dialog with the cursor on
// "1. Yes", so that Enter approves exactly once.
var approvePattern = regexp.MustCompile(`(?m)^\s*❯\s*1\.\s*Yes\b`)

// LooksLikeApprovalDialog reports whether the visible screen shows a
// permission dialog that Enter would approve once.
func LooksLikeApprovalDialog(screen string) bool { return approvePattern.MatchString(screen) }
