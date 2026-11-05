package server

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"remote-herdr/internal/herdr"
	"remote-herdr/internal/previews"
	"remote-herdr/internal/transcript"
)

// previewOwner is the session a preview folder belongs to. Host, PaneID and
// Title are set while that session is open in a herdr pane.
type previewOwner struct {
	SessionID string `json:"sessionId"`
	Source    string `json:"source"` // marker | transcript
	Host      string `json:"host,omitempty"`
	PaneID    string `json:"paneId,omitempty"`
	Title     string `json:"title,omitempty"`
}

type previewItem struct {
	previews.Entry
	Owner *previewOwner `json:"owner,omitempty"`
}

type previewApp struct {
	Port      int    `json:"port"`
	URL       string `json:"url"`
	Listening bool   `json:"listening"`
}

// mentionCache remembers which preview folders each transcript mentions,
// keyed by path and invalidated when the file size changes.
type mentionCache struct {
	mu    sync.Mutex
	files map[string]mentionFile
}

type mentionFile struct {
	size int64
	keys map[string]string // "<project>/<name>" -> timestamp of the last mention
}

func (s *Server) previewsEnabled() bool { return s.cfg.PreviewDir != "" }

// mentions returns the preview folders a session's transcript points to:
// URLs in assistant text, and URLs printed alone on a line of tool output
// (what a publishing script prints). User text is ignored, so a pasted link
// does not make a session the owner; neither do URLs inside search results
// or documents the session read.
func (s *Server) mentions(sessionID string) map[string]string {
	path, err := transcript.Locate(s.cfg.ProjectsDir, sessionID)
	if err != nil {
		return nil
	}
	size := transcript.Size(path)
	s.mcache.mu.Lock()
	if f, ok := s.mcache.files[path]; ok && f.size == size {
		s.mcache.mu.Unlock()
		return f.keys
	}
	s.mcache.mu.Unlock()

	msgs, size, err := transcript.Load(path, 0)
	if err != nil {
		return nil
	}
	keys := map[string]string{}
	for _, m := range msgs {
		for _, b := range m.Blocks {
			var found []string
			switch {
			case b.Kind == "text" && m.Role == "assistant":
				found = previews.Mentions(s.cfg.PreviewOrigin, b.Text)
			case b.Kind == "tool_result" && !b.IsError:
				found = previews.Published(s.cfg.PreviewOrigin, b.Text)
			}
			for _, k := range found {
				keys[k] = m.Timestamp
			}
		}
	}
	s.mcache.mu.Lock()
	if s.mcache.files == nil {
		s.mcache.files = map[string]mentionFile{}
	}
	s.mcache.files[path] = mentionFile{size: size, keys: keys}
	s.mcache.mu.Unlock()
	return keys
}

// resolvePreviews lists preview folders with their owners. A .session marker
// wins; otherwise the live session that mentioned the folder last owns it.
func (s *Server) resolvePreviews(agents []herdr.Agent) ([]previewItem, error) {
	entries, err := previews.Scan(s.cfg.PreviewDir, s.cfg.PreviewOrigin)
	if err != nil {
		return nil, err
	}
	live := map[string]herdr.Agent{}
	type mention struct{ sid, at string }
	mentioned := map[string]mention{}
	for _, a := range agents {
		if a.SessionID == "" {
			continue
		}
		if _, seen := live[a.SessionID]; seen {
			continue
		}
		live[a.SessionID] = a
		for k, at := range s.mentions(a.SessionID) {
			if cur, ok := mentioned[k]; !ok || at > cur.at {
				mentioned[k] = mention{a.SessionID, at}
			}
		}
	}
	out := make([]previewItem, 0, len(entries))
	for _, e := range entries {
		it := previewItem{Entry: e}
		switch {
		case e.Marker != "":
			it.Owner = &previewOwner{SessionID: e.Marker, Source: "marker"}
		default:
			if m, ok := mentioned[e.Key()]; ok {
				it.Owner = &previewOwner{SessionID: m.sid, Source: "transcript"}
			}
		}
		if it.Owner != nil {
			if a, ok := live[it.Owner.SessionID]; ok {
				it.Owner.Host, it.Owner.PaneID, it.Owner.Title = a.Host, a.PaneID, a.Title
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// previewFrameOrigins are the only origins the UI may show in an iframe:
// the preview origin and the app ports on its host. All differ from the
// app's origin (checked in New), so framed pages cannot call this API.
func (s *Server) previewFrameOrigins() []string {
	u, _ := url.Parse(s.cfg.PreviewOrigin)
	out := []string{s.cfg.PreviewOrigin}
	for _, p := range s.cfg.PreviewAppPorts {
		out = append(out, u.Scheme+"://"+net.JoinHostPort(u.Hostname(), strconv.Itoa(p)))
	}
	return out
}

// previewApps reports whether each configured app port has a listener on
// loopback. It only opens and closes a TCP connection; nothing is sent.
func (s *Server) previewApps(ctx context.Context) []previewApp {
	u, _ := url.Parse(s.cfg.PreviewOrigin)
	out := make([]previewApp, len(s.cfg.PreviewAppPorts))
	var wg sync.WaitGroup
	for i, p := range s.cfg.PreviewAppPorts {
		out[i] = previewApp{Port: p, URL: u.Scheme + "://" + net.JoinHostPort(u.Hostname(), strconv.Itoa(p)) + "/"}
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := net.Dialer{Timeout: 300 * time.Millisecond}
			c, err := d.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p)))
			if err == nil {
				c.Close()
				out[i].Listening = true
			}
		}()
	}
	wg.Wait()
	return out
}

// handlePreviews lists every preview folder and the app ports. herdr being
// down only loses the owners' pane links.
func (s *Server) handlePreviews(w http.ResponseWriter, r *http.Request) {
	agents, _ := s.listAgents(r.Context())
	items, err := s.resolvePreviews(agents)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "cannot read previews")
		return
	}
	writeJSON(w, map[string]any{"entries": items, "apps": s.previewApps(r.Context())})
}

// handleAgentPreviews lists the preview folders owned by one session.
func (s *Server) handleAgentPreviews(w http.ResponseWriter, r *http.Request, a herdr.Agent) {
	items := []previewItem{}
	if a.SessionID != "" {
		agents, _ := s.listAgents(r.Context())
		all, err := s.resolvePreviews(agents)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "cannot read previews")
			return
		}
		for _, it := range all {
			if it.Owner != nil && it.Owner.SessionID == a.SessionID {
				items = append(items, it)
			}
		}
	}
	writeJSON(w, map[string]any{"entries": items})
}
