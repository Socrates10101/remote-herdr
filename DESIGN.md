# Design

remote-herdr follows the layout and sizing conventions of a typical AI chat app
(sidebar + single thread column) and keeps its own colours (warm neutrals + clay
accent, Claude-app lineage). Source of truth: `web/src/app.css`.

## Principles

- The agent's answer is prose without a bubble; the owner's message is a pill bubble on the right.
- Chrome gets out of the way: the top bar and the composer slide away on downward scroll and return on upward scroll, at the top, and at the bottom. The approval sheet never hides.
- One authored motion: the eight-ray spark turns while an agent works. Everything else is a short ease-out.
- Light and dark follow the system (`prefers-color-scheme`).

## Colour

| Token | Light | Dark | Use |
|---|---|---|---|
| `--bg` | `#faf9f5` | `#262624` | page |
| `--surface` | `#f0eee6` | `#30302e` | grouped lists, segmented track, inline code |
| `--surface-2` | `#e8e6dc` | `#3a3a37` | user bubble, pressed rows |
| `--raised` | `#ffffff` | `#4a4944` | composer, active segment, approval sheet |
| `--text` / `--text-2` / `--text-3` | `#1f1e1d` / `#5e5d59` / `#8a887f` | `#f5f4ef` / `#b0aea5` / `#85837b` | primary / secondary / tertiary text |
| `--accent` | `#c6613f` | `#d97757` | send, approve, spark, folder icons, focus ring, selection |
| `--attention` | `#b8502f` | `#e8916f` | approval needed, errors |

Restrained strategy: warm neutrals plus the one clay accent. Terminal and approval excerpts sit on a near-black `--term-bg` in both themes.

## Type

- IBM Plex Sans JP (400/500/600) and IBM Plex Mono, self-hosted via @fontsource; `palt` on, antialiased.
- Base 14px/1.7. Prose (`.md`) 15px/1.8, h2 18px, inline code 13px mono, tables 13.5px.
- Top-bar title 17px/600; tabs and sidebar rows 13.5px; group labels and meta 12–12.5px.

## Components

- Layout: 268px sidebar (>820px) + main column; 56px top bar; thread max 880px; dock max 1000px.
- Top bar: title (and cwd on wide screens) on the left, 会話 / ファイル tabs on the right edge with a 2px accent underline; a back chevron only on phones.
- Sidebar: brand row (56px), status groups (承認が必要 / 作業中 / そのほか) as 12px labels, 13.5px rows with 8px radius, owner login in the footer. The row's meta line names the agent (Claude / Codex) only when both kinds are listed.
- Messages: owner's message as an 18px-radius bubble (max 70%); the agent's turn as a 26px mark column plus an "<agent> · time" header (Claude or Codex), then prose.
- Tool calls: one-line rows between hairlines (chevron, tool name, mono argument), packed into one run; expand to a code block.
- Code blocks: bordered 10px-radius box with a header row (language, コピー).
- Composer: 24px-radius box (20px on phones), textarea 15px, a lower row with the shortcut hint and a 36px round send button (stop while working).
- Answer sheet (agent blocked): question text, options as 8px-radius rows (checkboxes for multi-select, accent Submit row), free text disabled, キャンセル（Esc）.
- Phones: the header hides on any scroll; the composer hides only after 24px of downward scroll and returns on 8px upward. Both return at the page edges or on a tap. Wide screens keep both visible.
- Icons: one authored stroke set in `Icon.svelte` (24px grid, 1.75 stroke). No emoji.
