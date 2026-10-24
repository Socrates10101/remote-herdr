// Package files resolves read-only file access inside an agent's working
// directory.
//
// Rules:
//   - paths are relative to the root and must stay inside it after resolving symlinks;
//   - any path component starting with "." is refused (.env, .git, .claude, ...);
//   - only listed media types and small UTF-8 text files are served.
package files

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// ErrForbidden is returned for paths outside the root or hidden paths.
var ErrForbidden = errors.New("forbidden path")

// ErrUnsupported is returned for binary files that are not an allowed media type.
var ErrUnsupported = errors.New("unsupported file type")

// MaxTextBytes limits text files; MaxMediaBytes limits images and PDFs.
const (
	MaxTextBytes  = 2 << 20
	MaxMediaBytes = 50 << 20
)

// mediaTypes are served with their real type. SVG is only ever shown via <img>,
// and the server adds a sandbox CSP so scripts inside it cannot run.
var mediaTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".pdf":  "application/pdf",
}

// Entry is a directory listing item.
type Entry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
	Kind string `json:"kind"` // dir | image | pdf | text | other
}

// Resolve returns the absolute path for rel inside root.
func Resolve(root, rel string) (string, error) {
	if root == "" || !filepath.IsAbs(root) {
		return "", ErrForbidden
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if strings.HasPrefix(rel, "/") || strings.ContainsRune(rel, 0) {
		return "", ErrForbidden
	}
	clean := filepath.Clean("/" + rel)[1:]
	if clean != "" {
		for _, part := range strings.Split(clean, "/") {
			if part == ".." || strings.HasPrefix(part, ".") {
				return "", ErrForbidden
			}
		}
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(realRoot, clean))
	if err != nil {
		return "", err
	}
	if target != realRoot && !strings.HasPrefix(target, realRoot+string(filepath.Separator)) {
		return "", ErrForbidden
	}
	return target, nil
}

// KindOf classifies a file name for the UI.
func KindOf(name string) string {
	switch mt := mediaTypes[strings.ToLower(filepath.Ext(name))]; {
	case mt == "application/pdf":
		return "pdf"
	case strings.HasPrefix(mt, "image/"):
		return "image"
	}
	return "text"
}

// List returns visible entries of a directory, directories first.
func List(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		e := Entry{Name: name, Dir: info.IsDir(), Size: info.Size()}
		if e.Dir {
			e.Kind = "dir"
		} else if info.Mode().IsRegular() {
			e.Kind = KindOf(name)
		} else {
			e.Kind = "other"
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Content is an opened file ready to serve.
type Content struct {
	File        *os.File
	Size        int64
	ContentType string
	IsText      bool
}

// Open opens a regular file and decides how it may be served.
// The caller must close Content.File.
func Open(path string) (*Content, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		f.Close()
		return nil, ErrUnsupported
	}
	if mt, ok := mediaTypes[strings.ToLower(filepath.Ext(path))]; ok {
		if fi.Size() > MaxMediaBytes {
			f.Close()
			return nil, ErrUnsupported
		}
		return &Content{File: f, Size: fi.Size(), ContentType: mt}, nil
	}
	if fi.Size() > MaxTextBytes {
		f.Close()
		return nil, ErrUnsupported
	}
	head := make([]byte, 8192)
	n, _ := io.ReadFull(f, head)
	if !looksLikeText(head[:n]) {
		f.Close()
		return nil, ErrUnsupported
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}
	return &Content{File: f, Size: fi.Size(), ContentType: "text/plain; charset=utf-8", IsText: true}, nil
}

func looksLikeText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	// Tolerate a multi-byte rune cut at the buffer edge.
	for i := 0; i < utf8.UTFMax-1 && len(b) > 0 && !utf8.Valid(b); i++ {
		b = b[:len(b)-1]
	}
	return utf8.Valid(b) && !strings.ContainsRune(string(b), 0)
}
