// Package previews lists the preview folders that sessions publish under one
// directory (for example ~/previews/<project>/<name>/, served elsewhere at
// <origin>/<project>/<name>/) and works out which session each belongs to.
//
// It only reads: two directory levels and a small ".session" marker file per
// folder. Preview contents are never served from here; the UI links to the
// separate preview origin so preview scripts never run with remote-herdr's
// origin.
package previews

import (
	"bufio"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// MarkerName is the file a publisher writes into a preview folder. Its first
// line is the session id that published the folder: Claude Code's
// $CLAUDE_CODE_SESSION_ID or Codex's $CODEX_THREAD_ID.
const MarkerName = ".session"

const (
	maxMarkerBytes = 256
	maxEntries     = 2000
)

var sessionIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Entry is one preview folder.
type Entry struct {
	Project  string    `json:"project"`
	Name     string    `json:"name"`
	URL      string    `json:"url"`
	Modified time.Time `json:"modified"`
	// Marker is the session id from the folder's .session file, if any.
	Marker string `json:"-"`
}

// Key identifies an entry as "<project>/<name>", or "<project>" for a site
// published directly at the project level.
func (e Entry) Key() string {
	if e.Name == "" {
		return e.Project
	}
	return e.Project + "/" + e.Name
}

// Scan lists <root>/<project>/<name>/ folders, newest first. A project
// folder with its own index.html is one site and listed as a single entry
// with an empty Name. Names starting with "." and symbolic links are
// skipped. A missing root yields no entries.
func Scan(root, origin string) ([]Entry, error) {
	projects, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	for _, p := range projects {
		if !plainDir(p) {
			continue
		}
		pdir := filepath.Join(root, p.Name())
		if fi, err := os.Lstat(filepath.Join(pdir, "index.html")); err == nil && fi.Mode().IsRegular() {
			if len(out) < maxEntries {
				out = append(out, Entry{
					Project:  p.Name(),
					URL:      origin + "/" + url.PathEscape(p.Name()) + "/",
					Modified: modified(pdir, p),
					Marker:   readMarker(filepath.Join(pdir, MarkerName)),
				})
			}
			continue
		}
		names, err := os.ReadDir(pdir)
		if err != nil {
			continue
		}
		for _, n := range names {
			if !plainDir(n) || len(out) >= maxEntries {
				continue
			}
			dir := filepath.Join(root, p.Name(), n.Name())
			out = append(out, Entry{
				Project:  p.Name(),
				Name:     n.Name(),
				URL:      origin + "/" + url.PathEscape(p.Name()) + "/" + url.PathEscape(n.Name()) + "/",
				Modified: modified(dir, n),
				Marker:   readMarker(filepath.Join(dir, MarkerName)),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

// plainDir is a real directory (not a symlink) whose name does not start with ".".
func plainDir(d os.DirEntry) bool {
	return d.Type().IsDir() && !strings.HasPrefix(d.Name(), ".")
}

// modified is the newest mtime of the folder and its direct children, so
// overwriting index.html in place counts as an update.
func modified(dir string, d os.DirEntry) time.Time {
	var t time.Time
	if fi, err := d.Info(); err == nil {
		t = fi.ModTime()
	}
	children, _ := os.ReadDir(dir)
	for _, c := range children {
		if fi, err := c.Info(); err == nil && fi.ModTime().After(t) {
			t = fi.ModTime()
		}
	}
	return t
}

// readMarker returns the session id in a .session file, or "" when the file
// is missing, not a regular file, too large, or not a session id.
func readMarker(path string) string {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxMarkerBytes {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	line, _ := bufio.NewReader(io.LimitReader(f, maxMarkerBytes)).ReadString('\n')
	line = strings.TrimSpace(line)
	if !sessionIDPattern.MatchString(line) {
		return ""
	}
	return strings.ToLower(line)
}

// Mentions finds preview URLs under origin in text and returns their keys:
// "<project>/<name>" for two or more segments (deeper file paths are reduced
// to the folder) and "<project>" for a one-segment URL ending in "/". The
// bare origin is ignored.
func Mentions(origin, text string) []string {
	if origin == "" || !strings.Contains(text, origin) {
		return nil
	}
	var out []string
	rest := text
	for {
		i := strings.Index(rest, origin+"/")
		if i < 0 {
			return out
		}
		rest = rest[i+len(origin)+1:]
		end := strings.IndexFunc(rest, func(r rune) bool {
			return r <= ' ' || strings.ContainsRune("\"'<>()[]{}`\\|", r) || r > 0x7e
		})
		u := rest
		if end >= 0 {
			u = rest[:end]
		}
		u, _, _ = strings.Cut(u, "?")
		u, _, _ = strings.Cut(u, "#")
		segs := strings.SplitN(u, "/", 3)
		if len(segs) == 2 && segs[0] != "" && segs[1] == "" {
			if p, err := url.PathUnescape(segs[0]); err == nil && !strings.Contains(p, "/") {
				out = append(out, p)
			}
			continue
		}
		if len(segs) < 2 || segs[0] == "" || segs[1] == "" {
			continue
		}
		p, err1 := url.PathUnescape(segs[0])
		n, err2 := url.PathUnescape(segs[1])
		if err1 != nil || err2 != nil || strings.Contains(p, "/") || strings.Contains(n, "/") {
			continue
		}
		if len(segs) == 2 {
			// Without the trailing slash, sentence punctuation may follow the name.
			n = strings.TrimRight(n, ".,;:")
		}
		if n == "" {
			continue
		}
		out = append(out, p+"/"+n)
	}
}

// Published returns the keys of preview URLs that stand alone on a line of
// tool output, the way a publishing script prints them. URLs inside other
// output (search results, file listings, documentation) are not counted.
func Published(origin, text string) []string {
	if origin == "" || !strings.Contains(text, origin) {
		return nil
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, origin+"/") && !strings.ContainsAny(line, " \t") {
			out = append(out, Mentions(origin, line)...)
		}
	}
	return out
}
