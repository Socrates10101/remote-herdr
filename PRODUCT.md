# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users
The owner of the host machine (each person runs their own instance for their own herdr). They run several Claude Code sessions in parallel inside herdr panes and use remote-herdr mainly from an iPhone (Safari, added to the home screen) to follow those sessions away from the desk. If it reads well, it may become the everyday way to read sessions, including on a PC browser.

## Product Purpose
Make the output of running Claude Code sessions comfortable to read from a phone, and let the owner answer approvals or send a short instruction without opening a terminal. Success means the owner prefers reading a session here over squinting at a terminal screen.

## Positioning
Not an agent platform: it adds no accounts, relays, or new execution paths. It attaches to the sessions already running in herdr, shows them with proper markdown, math, code and images, and can only do what a person at the terminal would do with a prompt, Enter, or Esc.

## Operating Context
- Reached only through `tailscale serve` inside the owner's tailnet; identity comes from the Tailscale login.
- Sessions are herdr panes running the Claude Code TUI; the same session may be typed into from the terminal at any time.
- Content is mostly long assistant answers in Japanese and English with markdown, tables, LaTeX math, code, tool calls and occasional images; also project files (`.md`, `.tex`, images, PDFs).

## Capabilities and Constraints
- Read: agent list with status (blocked / working / idle / done), conversation transcript, terminal screen text, project files (read-only, no dotfiles).
- Act: send a prompt; answer a selection dialog (permission prompts, AskUserQuestion single/multi select) by moving the cursor and pressing Enter; cancel or interrupt (Esc).
- Never: run shell commands, write files, start agents. Prompts starting with `!` are refused.
- UI copy is Japanese.

## Brand Commitments
- Name: remote-herdr.
- The owner asked for the conventions of a typical AI chat app (familiar conversation layout) rather than a bespoke dashboard look, with carefully tuned typography for Japanese and English reading.
- Layout and sizes follow the internal Spark Chat design; colours stay remote-herdr's own (Claude-app lineage). Typography: IBM Plex Sans JP / Mono.
- Phones: the header hides while reading and the composer is easy to bring back; wide screens keep both visible with the session list in a left sidebar.

## Evidence on Hand
Real sessions on the host are the only content. No screenshots, logos, or other assets exist yet.

## Product Principles
1. Reading comes first: long answers, math and code must be comfortable on a phone.
2. Stay out of the way: chrome hides while reading and returns on demand.
3. Familiar over novel: behave like the chat apps the owner already uses.
4. Never more power than the terminal: every action maps to a prompt, Enter, or Esc.
