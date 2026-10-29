package transcript

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// maxSessionFileBytes bounds one <sessionsDir>/<pid>.json registry file.
const maxSessionFileBytes = 64 << 10

// Background is a running background Claude Code session (a fork or a job
// started from `claude agents`). herdr cannot see its session id because the
// pane only runs a client (`claude attach` or `claude agents`), so the
// session is matched by the name the client shows as the terminal title.
type Background struct {
	ID   string
	Name string
	Cwd  string
}

// LiveBackground reads the registry Claude Code keeps in
// <sessionsDir>/<pid>.json and returns background sessions whose process is
// still alive. Unreadable or malformed files are skipped.
func LiveBackground(sessionsDir string) []Background {
	matches, _ := filepath.Glob(filepath.Join(sessionsDir, "*.json"))
	var out []Background
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxSessionFileBytes {
			continue
		}
		b, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		var s struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
			Kind      string `json:"kind"`
			Name      string `json:"name"`
		}
		if json.Unmarshal(b, &s) != nil || s.Kind != "bg" || s.Name == "" ||
			!sessionIDPattern.MatchString(s.SessionID) || !alive(s.PID) {
			continue
		}
		out = append(out, Background{ID: s.SessionID, Name: s.Name, Cwd: s.Cwd})
	}
	return out
}

// MatchBackground returns the id of the only session named title in cwd, or
// "" when none or several match (guessing could show the wrong conversation).
func MatchBackground(sessions []Background, title, cwd string) string {
	id := ""
	for _, s := range sessions {
		if s.Name != title || s.Cwd != cwd {
			continue
		}
		if id != "" {
			return ""
		}
		id = s.ID
	}
	return id
}

// alive reports whether pid exists. Signal 0 only checks; nothing is sent.
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
