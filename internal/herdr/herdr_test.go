package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// serve answers each connection with respond(request).
func serve(t *testing.T, respond func(req map[string]any) string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "herdr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			line, _ := bufio.NewReader(c).ReadBytes('\n')
			var req map[string]any
			_ = json.Unmarshal(line, &req)
			_, _ = c.Write([]byte(respond(req) + "\n"))
			c.Close()
		}
	}()
	return path
}

func TestListAndPrompt(t *testing.T) {
	var got []map[string]any
	path := serve(t, func(req map[string]any) string {
		got = append(got, req)
		switch req["method"] {
		case "agent.list":
			return `{"id":"x","result":{"type":"agent_list","agents":[{"pane_id":"w1:p1","agent":"claude","agent_status":"idle","cwd":"/tmp","terminal_title_stripped":"T","agent_session":{"kind":"id","value":"abc"}}]}}`
		case "agent.prompt":
			return `{"id":"x","error":{"code":"agent_blocked","message":"blocked"}}`
		}
		return `{"id":"x","result":{}}`
	})
	c := &Client{SocketPath: path}
	agents, err := c.List(context.Background())
	if err != nil || len(agents) != 1 || agents[0].SessionID != "abc" || agents[0].Status != "idle" {
		t.Fatalf("List: %v %+v", err, agents)
	}
	err = c.Prompt(context.Background(), "w1:p1", "--looks-like-a-flag")
	var he *Error
	if !errors.As(err, &he) || he.Code != "agent_blocked" {
		t.Fatalf("Prompt error = %v", err)
	}
	params := got[1]["params"].(map[string]any)
	if params["text"] != "--looks-like-a-flag" || params["target"] != "w1:p1" {
		t.Fatalf("prompt params = %v", params)
	}
}
