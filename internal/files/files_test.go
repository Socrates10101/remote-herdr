package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "proj")
	outside = filepath.Join(base, "secret")
	for _, d := range []string{root, outside, filepath.Join(root, "docs"), filepath.Join(root, ".git")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "README.md"), "# hi")
	write(filepath.Join(root, ".env"), "TOKEN=x")
	write(filepath.Join(root, "docs", "a.tex"), `\section{x}`)
	write(filepath.Join(root, "bin.dat"), "\x00\x01\x02")
	write(filepath.Join(outside, "key"), "secret")
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	return root, outside
}

func TestResolve(t *testing.T) {
	root, _ := setup(t)
	ok := []string{"", "README.md", "docs/a.tex", "docs/../README.md"}
	for _, p := range ok {
		if _, err := Resolve(root, p); err != nil {
			t.Errorf("%q: %v", p, err)
		}
	}
	bad := []string{"../secret/key", "/etc/passwd", ".env", ".git", "docs/../../secret/key", "link/key", "a\x00b"}
	for _, p := range bad {
		if _, err := Resolve(root, p); err == nil {
			t.Errorf("%q: expected error", p)
		}
	}
	if _, err := Resolve("relative", "x"); !errors.Is(err, ErrForbidden) {
		t.Errorf("relative root accepted")
	}
}

func TestListHidesDotfiles(t *testing.T) {
	root, _ := setup(t)
	entries, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name[0] == '.' {
			t.Errorf("dotfile listed: %s", e.Name)
		}
	}
	if len(entries) == 0 || !entries[0].Dir {
		t.Errorf("directories should come first: %+v", entries)
	}
}

func TestOpen(t *testing.T) {
	root, _ := setup(t)
	c, err := Open(filepath.Join(root, "README.md"))
	if err != nil || !c.IsText {
		t.Fatalf("README: %v %+v", err, c)
	}
	c.File.Close()
	if _, err := Open(filepath.Join(root, "bin.dat")); !errors.Is(err, ErrUnsupported) {
		t.Errorf("binary served: %v", err)
	}
	if _, err := Open(root); !errors.Is(err, ErrUnsupported) {
		t.Errorf("directory served: %v", err)
	}
}
