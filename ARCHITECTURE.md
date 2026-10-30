# remote-herdr アーキテクチャ

herdr のペインで動いている Claude Code セッションを、スマホや PC のブラウザから
見て、プロンプトを送り、承認・拒否するための薄い UI。

## 目的と非目的

- 目的: 既存の herdr セッションを、ラッパーなしで、端末と同じ TUI のまま扱う。
- 目的: markdown・数式（KaTeX）・コード・画像・PDF を読みやすく表示する。
- 非目的: シェル実行、ファイル書き込み、エージェントの新規起動、独自アカウント、外部 Relay。
  これらは「設定で無効」ではなく、API 自体を持たない。

## 構成

```
iPhone/PC ブラウザ
  │ HTTPS（tailnet 内のみ）
  ▼
tailscale serve ──(unix socket 0600)──▶ remote-herdr（Go, 単一バイナリ）
                                          ├─ herdr ソケット API（agent.list/read/prompt/send_keys）
                                          ├─ ~/.claude/projects/*/<session>.jsonl（読み取りのみ）
                                          ├─ ~/.claude/sessions/<pid>.json（読み取りのみ）
                                          └─ 埋め込み UI（Vite + Svelte のビルド成果物）
```

| 層 | 技術 | 位置 |
|---|---|---|
| サーバー | Go 標準ライブラリのみ | `cmd/`, `internal/` |
| herdr 連携 | herdr の unix ソケット JSON API（1 リクエスト 1 行） | `internal/herdr` |
| 会話記録 | Claude Code の JSONL を読み取り専用で正規化 | `internal/transcript` |
| ファイル閲覧 | エージェントの cwd 配下を読み取り専用で配信 | `internal/files` |
| UI | Vite + TypeScript + Svelte 5、pnpm | `web/` |
| 描画 | markdown-it（html:false）+ texmath/KaTeX（trust:false）+ highlight.js + DOMPurify | `web/src/lib/markdown.ts` |

herdr の CLI ではなくソケット API を使う。CLI は `--` 区切りに対応しておらず、
`-` で始まるプロンプトがオプションとして解釈されうるため。

## セキュリティ不変条件

変更時に壊してはいけない約束。テストは `internal/server/server_test.go` などにある。

1. **待ち受け**: 既定は unix socket（ディレクトリ 0700、ソケット 0600）。TCP は loopback のみ許可。
2. **認証**: 全リクエストで `Tailscale-User-Login` を許可リストと照合する（tailscale serve が付与し、
   クライアント送信分は削除される）。許可リストは起動時に必須。
3. **Host**: 公開 origin のホストか `localhost`（serve → unix socket 時）のみ受け付ける。
4. **クロスサイト対策**: `Sec-Fetch-Site` が cross-site の API リクエストを拒否。POST は
   `Origin` 完全一致、`X-Remote-Herdr: 1`、`Content-Type: application/json` を必須にする。
   cookie を使わないため、セッション奪取の対象もない。
5. **API の範囲**: 一覧、画面読み取り、会話記録、ファイル一覧・読み取り、プロンプト送信、
   承認・拒否、中断のみ。書き込み・実行・起動の API を追加しない。
6. **プロンプト**: `!` 始まり（Claude Code のシェル実行）は既定で拒否し、`--allow-shell` を付けたときだけ通す
   （tailnet 内の許可ユーザーとして届く端末が、そのままこのマシンのシェルになる）。`/` 始まりは `--allow-slash` で
   許可したものだけ。改行・タブ以外の制御文字（エスケープシーケンス、bracketed paste）を拒否。
   承認待ち（blocked）のエージェントには送らない。
7. **承認・選択**: blocked のときだけ受け付ける。画面の選択肢（`❯` カーソル付きの番号行・Submit 行）を
   サーバー側で読み取り、選ばれた行までの ↑↓ と Enter だけを送る。クライアントが見た行の文言と
   今の画面が一致しないときは何も送らない。自由入力（Type something）は端末で答える。キャンセルは Esc。
8. **ファイル**: cwd 配下のみ。シンボリックリンクを解決した後も cwd 内であること。
   `.` で始まるパス要素（.env, .git, .claude …）は拒否。画像・PDF は種類を限定し、
   応答に `Content-Security-Policy: sandbox` を付ける。テキストは UTF-8 かつ 2MB 以下。
9. **描画**: markdown の生 HTML は無効。最終 HTML は DOMPurify を通す。画像は同一オリジンの
   ファイル API か data:image のみ、リンクは http(s) のみ。
10. **CSP**: `script-src 'self'`。KaTeX のため `style-src` のみ `'unsafe-inline'`。
11. **監査**: プロンプト送信・承認・中断をユーザー名つきでログに出す（本文は出さない）。
    `!` 始まりは `shell` として区別して記録する。

## 既知の制約

- 端末で入力途中のときにスマホから送ると、入力欄の文字と混ざる。送信前に端末を確認する。
- JSONL は Claude Code の内部形式で、版によって変わる。会話タブは最善努力で、
  確実な表示は「画面」タブ（herdr の画面読み取り）。
- fork したセッションはペインの外でバックグラウンドとして動き、ペインには `claude attach` や
  `claude agents` しかない。herdr はセッション ID を返さないので、`~/.claude/sessions/<pid>.json` の
  `kind: bg` のうち、`name` がペインのタイトルと一致し `cwd` も同じものを 1 件だけ見つけたときに使う。
  一致が 0 件か 2 件以上なら会話タブは空になる（取り違えを防ぐため）。
- cwd がホームディレクトリのセッションでは、ホーム配下（ドット以外）を閲覧できる。
- 認証は tailnet の ACL と Tailscale のユーザー識別に依存する。tagged device からは使えない。

## 複数ホスト（将来の拡張・未実装）

今は 1 台だけを扱う。ほかのマシンの herdr もまとめる必要が出たときに、作り直さずに足せるよう
次の土台だけを入れてある。

- エージェントは `host`（`-host-name`、既定は hostname）を持ち、セッションの識別は `host/pane`。
  SSE のサイズ表も `host/pane` をキーにしている。
- UI は API の宛先を `web/src/lib/api.ts` の `agentBase()` だけで組み立てる。
- 一覧は、ホストが 2 台以上になったときだけホスト名を表示する。

想定する形は「各ホストで remote-herdr を動かし、1 台がハブとして束ねる」構成。

```
ブラウザ ─▶ ハブ ─┬─ ローカル herdr（unix socket）
                  └─▶ 他ホストの remote-herdr（tailnet の HTTPS）
```

- 会話記録とファイルは各ホストのディスクにあるため、読むのは各ホストの remote-herdr に任せる。
- ハブは `/api/hosts/{host}/...` を該当ホストの `/api/...` に中継し、一覧と SSE を合成する。
  中継してよいのは上の「API の範囲」と同じ経路だけ（許可リスト方式）。
- 他ホスト側の認証は今と同じ Tailscale ログイン照合。ハブのマシンも所有者の端末なので通る。
  ハブは POST に相手の origin と CSRF ヘッダを付け直す。各ホストの安全策は緩めない。
- ブラウザから各ホストへ直接つなぐ案は採らない（CORS を開けることになり、同一オリジンの前提が崩れる）。
- herdr 自身の SSH 経由の遠隔操作は画面しか扱えないため、会話記録・ファイルには使わない。
