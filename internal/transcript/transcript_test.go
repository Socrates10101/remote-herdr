package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sample = `{"type":"user","uuid":"u1","timestamp":"t1","message":{"role":"user","content":"Explain $e^{i\\pi}$"}}
{"type":"assistant","uuid":"a1","timestamp":"t2","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hidden"},{"type":"text","text":"Sure: \\(e^{i\\pi}=-1\\)"},{"type":"tool_use","name":"Read","input":{"file_path":"/x.png"}}]}}
{"type":"user","uuid":"u2","timestamp":"t3","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"text","text":"ok"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}]}]}}
{"type":"user","uuid":"u3","isSidechain":true,"message":{"role":"user","content":"subagent"}}
{"type":"user","uuid":"u4","isMeta":true,"message":{"role":"user","content":"meta"}}
{"type":"assistant","uuid":"a2","message":{"role":"assistant","content":[{"type":"image","source":{"type":"base64","media_type":"image/svg+xml","data":"PHN2Zz4="}}]}}
{"type":"attachment","uuid":"x"}
not json
`

func TestParse(t *testing.T) {
	msgs := Parse([]byte(sample))
	if len(msgs) != 3 {
		t.Fatalf("got %d messages: %+v", len(msgs), msgs)
	}
	if msgs[0].Blocks[0].Text != "Explain $e^{i\\pi}$" {
		t.Errorf("user text = %q", msgs[0].Blocks[0].Text)
	}
	a := msgs[1].Blocks
	if len(a) != 2 || a[0].Kind != "text" || a[1].Kind != "tool_use" || a[1].Name != "Read" {
		t.Errorf("assistant blocks = %+v", a)
	}
	r := msgs[2].Blocks
	if len(r) != 2 || r[0].Kind != "tool_result" || r[1].Kind != "image" || r[1].MediaType != "image/png" {
		t.Errorf("tool result blocks = %+v", r)
	}
}

func TestLocateAndLoad(t *testing.T) {
	dir := t.TempDir()
	id := "c68603cb-965a-435c-a868-3f4bacd2ff89"
	proj := filepath.Join(dir, "-home-x")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, id+".jsonl"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Locate(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	msgs, size, err := Load(p, 2)
	if err != nil || len(msgs) != 2 || size != int64(len(sample)) {
		t.Fatalf("Load: %v %d %d", err, len(msgs), size)
	}
	if _, err := Locate(dir, "../../etc/passwd"); err == nil || strings.Contains(err.Error(), "not found") {
		t.Errorf("invalid id accepted: %v", err)
	}
}
