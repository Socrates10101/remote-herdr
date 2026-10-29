// Command remote-herdr serves a small web UI for the Claude Code sessions
// running in herdr panes. Put it behind `tailscale serve`; see README.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"remote-herdr/internal/herdr"
	"remote-herdr/internal/server"
	"remote-herdr/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func defaultSocket() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), fmt.Sprintf("remote-herdr-%d", os.Getuid()))
	}
	return filepath.Join(dir, "remote-herdr", "remote-herdr.sock")
}

func splitSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out[v] = true
		}
	}
	return out
}

func run() error {
	home, _ := os.UserHomeDir()
	listen := flag.String("listen", "unix:"+defaultSocket(), "unix:/path/to.sock or tcp:127.0.0.1:PORT (loopback only)")
	origin := flag.String("origin", "", "public origin served by tailscale serve, e.g. https://host.tailnet.ts.net:8450 (required)")
	allowUsers := flag.String("allow-user", "", "comma-separated Tailscale logins allowed to use the UI (required)")
	allowSlash := flag.String("allow-slash", "", "comma-separated slash commands allowed in remote prompts, e.g. /compact")
	herdrSock := flag.String("herdr-socket", herdr.DefaultSocketPath(), "herdr API socket")
	hostName := flag.String("host-name", "", "name shown for this machine (default: hostname)")
	accessLog := flag.Bool("access-log", false, "log every request (for debugging clients)")
	claudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if claudeDir == "" {
		claudeDir = filepath.Join(home, ".claude")
	}
	projects := flag.String("projects-dir", filepath.Join(claudeDir, "projects"), "Claude Code projects directory ($CLAUDE_CONFIG_DIR/projects)")
	sessions := flag.String("sessions-dir", filepath.Join(claudeDir, "sessions"), "Claude Code running-session registry ($CLAUDE_CONFIG_DIR/sessions), used to find forked sessions")
	flag.Parse()

	srv, err := server.New(server.Config{
		Herdr:        &herdr.Client{SocketPath: *herdrSock},
		ProjectsDir:  *projects,
		SessionsDir:  *sessions,
		AllowedUsers: splitSet(*allowUsers),
		Origin:       *origin,
		AllowedSlash: splitSet(*allowSlash),
		Static:       web.Dist(),
		HostName:     *hostName,
		AccessLog:    *accessLog,
		Logger:       log.New(os.Stderr, "", log.LstdFlags),
	})
	if err != nil {
		return err
	}

	ln, err := listenOn(*listen)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("remote-herdr listening on %s for %s", *listen, *origin)
	if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// listenOn opens a unix socket (dir 0700, socket 0600) or a loopback TCP port.
func listenOn(spec string) (net.Listener, error) {
	switch {
	case strings.HasPrefix(spec, "unix:"):
		path := strings.TrimPrefix(spec, "unix:")
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("unix socket path must be absolute: %s", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		if fi, err := os.Lstat(path); err == nil {
			if fi.Mode()&os.ModeSocket == 0 {
				return nil, fmt.Errorf("refusing to replace non-socket file %s", path)
			}
			_ = os.Remove(path)
		}
		old := syscall.Umask(0o177)
		ln, err := net.Listen("unix", path)
		syscall.Umask(old)
		if err != nil {
			return nil, err
		}
		return ln, os.Chmod(path, 0o600)
	case strings.HasPrefix(spec, "tcp:"):
		addr := strings.TrimPrefix(spec, "tcp:")
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("tcp listen address must be loopback, got %s", host)
		}
		return net.Listen("tcp", addr)
	}
	return nil, fmt.Errorf("listen must start with unix: or tcp:")
}
