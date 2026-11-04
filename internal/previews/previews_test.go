package previews

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sid = "0123abcd-0000-4000-8000-00000000000a"

func mk(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	mk(t, filepath.Join(root, "proj", "20261104-a b", "index.html"), "x")
	mk(t, filepath.Join(root, "proj", "20261104-a b", MarkerName), sid+"\n")
	mk(t, filepath.Join(root, "proj", "20261103-old", "index.html"), "x")
	mk(t, filepath.Join(root, "proj", "20261103-old", MarkerName), "not-a-session\n")
	mk(t, filepath.Join(root, ".trash", "x", "index.html"), "x")     // dot project
	mk(t, filepath.Join(root, "proj", ".hidden", "index.html"), "x") // dot entry
	mk(t, filepath.Join(root, "loose.html"), "x")                    // file at the root
	mk(t, filepath.Join(root, "site", "index.html"), "x")            // project-level site
	mk(t, filepath.Join(root, "site", "assets", "a.png"), "x")       // part of that site
	outside := t.TempDir()
	mk(t, filepath.Join(outside, "secret", "index.html"), "x")
	if err := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "proj", "link")); err != nil {
		t.Fatal(err)
	}
	// A symlinked marker is not trusted.
	mk(t, filepath.Join(outside, "m"), sid)
	mk(t, filepath.Join(root, "proj2", "e", "index.html"), "x")
	if err := os.Symlink(filepath.Join(outside, "m"), filepath.Join(root, "proj2", "e", MarkerName)); err != nil {
		t.Fatal(err)
	}

	got, err := Scan(root, "https://h:8460")
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Entry{}
	for _, e := range got {
		byKey[e.Key()] = e
	}
	if s := byKey["site"]; s.Name != "" || s.URL != "https://h:8460/site/" {
		t.Errorf("site = %+v", s)
	}
	if _, ok := byKey["site/assets"]; ok {
		t.Error("subfolder of a site listed separately")
	}
	if len(got) != 4 {
		t.Fatalf("entries = %v", got)
	}
	a := byKey["proj/20261104-a b"]
	if a.Marker != sid || a.URL != "https://h:8460/proj/20261104-a%20b/" {
		t.Errorf("a = %+v", a)
	}
	if byKey["proj/20261103-old"].Marker != "" {
		t.Error("invalid marker accepted")
	}
	if byKey["proj2/e"].Marker != "" {
		t.Error("symlinked marker accepted")
	}

	if got, err := Scan(filepath.Join(root, "missing"), "https://h"); err != nil || got != nil {
		t.Errorf("missing root: %v %v", got, err)
	}
}

func TestMentions(t *testing.T) {
	o := "https://h.ts.net:8460"
	text := strings.Join([]string{
		o + "/proj/20261104-fig/",
		"(" + o + "/proj/20261104-two/index.html)",
		"見て: " + o + "/p%20x/n. 次へ",
		o + "/",                         // bare origin
		o + "/site/",                    // one segment: a project-level site
		"https://h.ts.net:8461/proj/x/", // other port
		"`" + o + "/proj/q/?v=2`",
	}, "\n")
	want := []string{"proj/20261104-fig", "proj/20261104-two", "p x/n", "site", "proj/q"}
	if got := Mentions(o, text); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
	if Mentions(o, "nothing here") != nil {
		t.Error("expected nil")
	}
}

func TestPublished(t *testing.T) {
	o := "https://h.ts.net:8460"
	out := "copied\n" + o + "/proj/fig/\n  59 " + o + "/proj/counted/\nsee " + o + "/proj/doc/ for details\n"
	if got := Published(o, out); !reflect.DeepEqual(got, []string{"proj/fig"}) {
		t.Errorf("got %q", got)
	}
}
