// Package herdr talks to the local herdr server over its unix-socket JSON API.
//
// Only the agent methods remote-herdr needs are wrapped. Each call opens a
// fresh connection, writes one JSON request line and reads one response line.
package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Agent is the subset of herdr's agent info that the UI uses.
type Agent struct {
	Host        string `json:"host"` // set by remote-herdr, not herdr
	PaneID      string `json:"paneId"`
	WorkspaceID string `json:"workspaceId"`
	TabID       string `json:"tabId"`
	Agent       string `json:"agent"`
	SessionID   string `json:"sessionId"`
	Status      string `json:"status"`
	Title       string `json:"title"`
	Cwd         string `json:"cwd"`
	Focused     bool   `json:"focused"`
}

// Error is an error response from the herdr server.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("herdr: %s: %s", e.Code, e.Message) }

// Client calls the herdr socket API.
type Client struct {
	SocketPath string
	Timeout    time.Duration
}

// DefaultSocketPath returns $HERDR_SOCKET_PATH or ~/.config/herdr/herdr.sock.
func DefaultSocketPath() string {
	if p := os.Getenv("HERDR_SOCKET_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "herdr", "herdr.sock")
}

// maxResponseBytes bounds a single response line (screen reads can be large).
const maxResponseBytes = 8 << 20

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.SocketPath)
	if err != nil {
		return fmt.Errorf("herdr: connect: %w", err)
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	req, err := json.Marshal(map[string]any{"id": "remote-herdr", "method": method, "params": params})
	if err != nil {
		return err
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return fmt.Errorf("herdr: write: %w", err)
	}

	r := bufio.NewReaderSize(conn, 64<<10)
	line, err := readLine(r, maxResponseBytes)
	if err != nil {
		return fmt.Errorf("herdr: read: %w", err)
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *Error          `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("herdr: decode: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(resp.Result, result)
}

func readLine(r *bufio.Reader, limit int) ([]byte, error) {
	var buf []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return nil, err
		}
		buf = append(buf, chunk...)
		if len(buf) > limit {
			return nil, errors.New("response too large")
		}
		if !isPrefix {
			return buf, nil
		}
	}
}

type rawAgent struct {
	PaneID       string  `json:"pane_id"`
	WorkspaceID  string  `json:"workspace_id"`
	TabID        string  `json:"tab_id"`
	Agent        *string `json:"agent"`
	AgentStatus  string  `json:"agent_status"`
	TitleStrip   string  `json:"terminal_title_stripped"`
	Cwd          *string `json:"cwd"`
	Focused      bool    `json:"focused"`
	AgentSession *struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	} `json:"agent_session"`
}

func (a rawAgent) toAgent() Agent {
	out := Agent{
		PaneID:      a.PaneID,
		WorkspaceID: a.WorkspaceID,
		TabID:       a.TabID,
		Status:      a.AgentStatus,
		Title:       a.TitleStrip,
		Focused:     a.Focused,
	}
	if a.Agent != nil {
		out.Agent = *a.Agent
	}
	if a.Cwd != nil {
		out.Cwd = *a.Cwd
	}
	if a.AgentSession != nil && a.AgentSession.Kind == "id" {
		out.SessionID = a.AgentSession.Value
	}
	return out
}

// List returns every pane herdr recognises as an agent.
func (c *Client) List(ctx context.Context) ([]Agent, error) {
	var res struct {
		Agents []rawAgent `json:"agents"`
	}
	if err := c.call(ctx, "agent.list", map[string]any{}, &res); err != nil {
		return nil, err
	}
	agents := make([]Agent, 0, len(res.Agents))
	for _, a := range res.Agents {
		agents = append(agents, a.toAgent())
	}
	return agents, nil
}

// Read returns the pane's terminal text with ANSI sequences stripped.
// source is one of visible, recent, recent_unwrapped.
func (c *Client) Read(ctx context.Context, pane, source string, lines int) (string, error) {
	params := map[string]any{"target": pane, "source": source, "format": "text", "strip_ansi": true}
	if lines > 0 {
		params["lines"] = lines
	}
	var res struct {
		Read struct {
			Text string `json:"text"`
		} `json:"read"`
	}
	if err := c.call(ctx, "agent.read", params, &res); err != nil {
		return "", err
	}
	return res.Read.Text, nil
}

// Prompt submits text followed by Enter. herdr rejects it with
// agent_blocked when the agent is waiting at an approval dialog.
func (c *Client) Prompt(ctx context.Context, pane, text string) error {
	return c.call(ctx, "agent.prompt", map[string]any{"target": pane, "text": text}, nil)
}

// SendKeys sends named key presses (for example "enter", "esc").
func (c *Client) SendKeys(ctx context.Context, pane string, keys ...string) error {
	return c.call(ctx, "agent.send_keys", map[string]any{"target": pane, "keys": keys}, nil)
}
