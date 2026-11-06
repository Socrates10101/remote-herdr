# remote-herdr

日本語: [README.ja.md](README.ja.md)

A thin web UI for the Claude Code and Codex (Codex CLI) sessions running in your [herdr](https://herdr.dev) panes.
Open it from a phone or PC browser inside your tailnet to read the conversation, send a
prompt, and answer permission prompts or multiple-choice questions.

- Attaches to the sessions that are already running. No wrapper, no restart.
- Renders markdown, math (KaTeX), code, images and PDFs so long answers are comfortable to read.
- Can do only what a person at the terminal can do with a prompt, Up/Down + Enter, or Esc.
  There is no API for running shell commands, writing files, or starting agents.
- No accounts and no relay server. Authentication and transport are left to Tailscale.

The design and the security promises are in [ARCHITECTURE.md](ARCHITECTURE.md) (Japanese);
the visual rules are in [DESIGN.md](DESIGN.md).

## Requirements

| What | Version | Used for |
|---|---|---|
| [herdr](https://herdr.dev) | 0.9 or later | Session list, screen reading, input (socket API) |
| Claude Code | tested with the 2.1 series | Transcript format (`~/.claude/projects`) and the shape of its permission dialogs |
| Codex CLI | tested with 0.166 | Transcript format (`~/.codex/sessions`) and the shape of its approval and question screens. Needs herdr's Codex integration (`herdr integration install codex`) |
| Tailscale | with `tailscale serve` available | HTTPS, tailnet-only exposure, user identity |
| Go / Node.js / pnpm | 1.26+ / 22+ / 11+ | Build time only |

The intended setup is one instance per person, on their own machine, talking to their own herdr.

## Getting started (about 15 minutes)

1. Build

   ```sh
   git clone https://github.com/Socrates10101/remote-herdr.git && cd remote-herdr
   make build          # builds the UI and embeds it into ./remote-herdr
   ```

2. Find your public URL and your Tailscale login name

   ```sh
   tailscale status --self --json | grep DNSName   # e.g. mybox.tail1234.ts.net.
   tailscale whois $(tailscale ip -4)              # the Name under "User:" is your login (e.g. alice@github)
   ```

   Your public URL is `https://<DNSName without the trailing dot>:<a free port>`. The examples below use 8450.

3. Start the server

   ```sh
   ./remote-herdr \
     -origin https://mybox.tail1234.ts.net:8450 \
     -allow-user alice@github
   ```

   The startup log prints the socket it listens on
   (on Linux the default is `$XDG_RUNTIME_DIR/remote-herdr/remote-herdr.sock`).

4. Expose it to your tailnet (needs root)

   ```sh
   sudo tailscale serve --bg --https=8450 unix:/run/user/$(id -u)/remote-herdr/remote-herdr.sock
   ```

5. Open `https://mybox.tail1234.ts.net:8450` in a browser on your phone or PC.

### Run it as a service (Linux)

Edit `-origin` and `-allow-user` in [contrib/remote-herdr.service](contrib/remote-herdr.service),
copy it to `~/.config/systemd/user/`, then run:

```sh
systemctl --user enable --now remote-herdr
sudo loginctl enable-linger $USER     # if it should start without you being logged in
```

To update, run `make build` and then `systemctl --user restart remote-herdr`.
There is no launchd configuration for macOS yet.

## Usage

- **List**: every herdr session, grouped into "needs approval / working / other".
  On wide screens it is the left sidebar; on phones it is the first screen.
- **Conversation**: the agent's (Claude's or Codex's) output plus tool calls (one line each, tap to expand).
  For Codex, commands with their output, file changes and web searches show up as tool calls.
  A prompt typed in the composer at the bottom is sent to the agent in the terminal as is.
  While the agent is working, the send button becomes an interrupt button (Esc).
- **Answer**: when a permission prompt or a multiple-choice question appears, the options show up as buttons.
  For multi-select, pick the items and then Submit. Free-text answers ("Type something") must be given at the terminal.
  With Codex this covers command and file-edit approvals, the folder trust prompt, and questions (request_user_input).
  Cancel (Esc) means "No, and tell Codex what to do differently" there, which stops the current turn.
- **Files**: browse the session's working directory read-only (images, PDFs, md, tex, ...).
  Files and directories starting with `.` are not shown.
- **Previews** (optional): with `-preview-dir`, each session gets a "Previews" tab and the list gets a "Previews" page.
  The tab opens the session's only preview directly, or lets you pick one when there are several, and shows it
  inside the session view (the top bar and composer float over it translucently, so you can keep giving instructions;
  it can also be opened in a new tab). See "Linking previews" below.
- **Phones**: add it to the home screen for a full-screen app. Swipe from the edge to go back to the list.

### Caveats

- If you send from the phone while something is half-typed in the terminal, the two inputs get mixed.
- Input starting with `!` (the shell mode of Claude Code and Codex) is refused by default. `-allow-shell` enables it.
  With it enabled, any device that can reach this UI can run commands on this machine without a permission prompt.
  Slash commands are also refused by default; allow them individually, e.g. `-allow-slash /compact`.

## Linking previews (optional)

If a separate static server publishes the HTML an agent produces (diagrams, UI drafts) from
`<dir>/<project>/<name>/` at `https://<host>:<port>/<project>/<name>/`, remote-herdr can list those
previews per session and show them.

```sh
./remote-herdr ... \
  -preview-dir ~/previews \
  -preview-origin https://mybox.tail1234.ts.net:8460 \
  -preview-app-ports 8461,8462     # ports used for running apps (optional; shows whether each is in use)
```

- If the publisher writes the session id (`$CLAUDE_CODE_SESSION_ID` for Claude Code, `$CODEX_THREAD_ID` for Codex) as one line to `<name>/.session`,
  the preview is linked to that session reliably. Without it, a preview is linked to the open session
  that last posted its URL.
- remote-herdr neither serves nor deletes previews. It embeds the page from the other origin, and the
  embedded page cannot operate remote-herdr.
- While a preview is shown, the top bar and composer hide after a short idle; touch the top or bottom edge
  to bring them back. To hide and show them as the page scrolls, the static server must add a script to
  its HTML that reports the scroll position to the parent
  (`parent.postMessage({type: 'remote-herdr:scroll', y, max}, '*')`).
- If the static server sends headers that forbid embedding (such as `X-Frame-Options`), open the preview in a new tab.

## Safety model

- The server listens only on a unix socket (mode 0600) and is reachable from outside only through `tailscale serve`.
- Any Tailscale user not listed in `-allow-user` gets a 403. Also restrict which devices may reach
  this port in your tailnet ACL.
- **Do not expose it with Funnel (public internet).** Funnel requests carry no Tailscale user identity,
  which breaks the whole model.
- For a session whose working directory is your home directory, everything under home
  (except dot-files) becomes browsable.

## Troubleshooting

| Symptom | Check |
|---|---|
| Blank page | The browser is not blocking scripts (in Brave, turn off "Block scripts" in Shields for this site only) |
| `not allowed` (403) | The login in `-allow-user` matches the Name under User in `tailscale whois` |
| `unexpected host` (421) | `-origin` matches the URL in the browser, including the port |
| Empty list or `herdr unavailable` | Pass the `socket:` path from `herdr status server` as `-herdr-socket` |
| No conversation shown | If you use `CLAUDE_CONFIG_DIR`, pass its `projects` subdirectory as `-projects-dir` |
| No Codex conversation shown | Check that herdr's Codex integration is installed (`herdr integration status`). If you use `CODEX_HOME`, pass its `sessions` subdirectory as `-codex-sessions-dir`. A Codex session has no id until its first prompt, so it stays empty until then |
| No options shown | Claude Code's or Codex's screen layout may have changed. Answer at the terminal and share the screen text in an issue |

For debugging, `-access-log` records the outcome of every request (prompt bodies are never logged).

## Options

| Flag | Default | Description |
|---|---|---|
| `-origin` | (required) | The URL as seen from the browser (`https://host:port`) |
| `-allow-user` | (required) | Tailscale logins allowed to use the UI (comma-separated) |
| `-listen` | `unix:$XDG_RUNTIME_DIR/remote-herdr/remote-herdr.sock` | `unix:/path` or `tcp:127.0.0.1:PORT` (loopback only) |
| `-allow-slash` | none | Slash commands that may be sent remotely (comma-separated) |
| `-allow-shell` | off | Allow input starting with `!` (the shell mode of Claude Code and Codex) |
| `-herdr-socket` | `$HERDR_SOCKET_PATH` or `~/.config/herdr/herdr.sock` | herdr API socket |
| `-projects-dir` | `$CLAUDE_CONFIG_DIR/projects` (or `~/.claude/projects`) | Claude Code transcripts |
| `-sessions-dir` | `$CLAUDE_CONFIG_DIR/sessions` (or `~/.claude/sessions`) | Registry of running sessions, used to find the transcript of forked sessions |
| `-codex-sessions-dir` | `$CODEX_HOME/sessions` (or `~/.codex/sessions`) | Codex transcripts. Must be an absolute path; empty turns Codex transcripts off |
| `-host-name` | hostname | Machine name shown in the UI (for future multi-host support) |
| `-preview-dir` | none | Folder of published previews (`<project>/<name>/`). Enables the preview features |
| `-preview-origin` | none | URL that serves `-preview-dir` (`https://host:port`). Required with `-preview-dir` |
| `-preview-app-ports` | none | Ports used for running apps (comma-separated). The UI shows whether each is in use |
| `-access-log` | off | Access log for debugging |

## Development

```sh
make test     # go vet + go test + svelte-check
make build
```

- Server: Go standard library only (`cmd/`, `internal/`).
- UI: Svelte 5 + Vite + TypeScript with pnpm (`web/`). The build output is embedded in the binary.
- UI copy is currently Japanese.
- Contributor rules are in [AGENTS.md](AGENTS.md). Any change to a security promise needs a test.

## License

MIT ([LICENSE](LICENSE))
