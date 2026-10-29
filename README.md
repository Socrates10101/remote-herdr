# remote-herdr

herdr のペインで動いている Claude Code セッションを、tailnet 内のスマホや PC のブラウザから
読んで、プロンプトを送り、承認や選択式の質問に答えるための薄い Web UI です。

- 今動いているセッションに、そのまま接続します（ラッパーで起動し直す必要はありません）。
- markdown・数式（KaTeX）・コード・画像・PDF を読みやすく表示します。
- できる操作は、端末の前にいる人が「プロンプトを打つ・↑↓/Enter で選ぶ・Esc を押す」ことだけです。
  シェル実行・ファイル書き込み・エージェントの起動はできません（その API を持っていません）。
- アカウントや中継サーバーはありません。認証と通信は Tailscale に任せます。

設計と安全上の約束は [ARCHITECTURE.md](ARCHITECTURE.md)、見た目の決まりは [DESIGN.md](DESIGN.md) にあります。

## 必要なもの

| もの | 版 | 用途 |
|---|---|---|
| [herdr](https://herdr.dev) | 0.9 以上 | セッション一覧・画面の読み取り・入力（ソケット API） |
| Claude Code | 2.1 系で確認 | 会話記録（`~/.claude/projects`）と確認ダイアログの形式 |
| Tailscale | `tailscale serve` が使えること | HTTPS 化と、tailnet 内だけへの公開・ユーザー識別 |
| Go / Node.js / pnpm | 1.26+ / 22+ / 11+ | ビルド時のみ |

一人ひとりが自分のマシンで、自分の herdr に対して 1 つ動かす想定です。

## 使い始める（約 15 分）

1. ビルドする

   ```sh
   git clone <この repo> && cd remote-herdr
   make build          # UI をビルドして ./remote-herdr に埋め込む
   ```

2. 自分の公開 URL とログイン名を調べる

   ```sh
   tailscale status --self --json | grep DNSName   # 例: mybox.tail1234.ts.net.
   tailscale whois $(tailscale ip -4)              # 「User:」の下の Name がログイン名（例: alice@github）
   ```

   公開 URL は `https://<DNSName（末尾の . を除く）>:<空いているポート>` です（以下 8450 の例）。

3. 起動する

   ```sh
   ./remote-herdr \
     -origin https://mybox.tail1234.ts.net:8450 \
     -allow-user alice@github
   ```

   起動時のログに待ち受けるソケットのパスが出ます
   （Linux の既定は `$XDG_RUNTIME_DIR/remote-herdr/remote-herdr.sock`）。

4. tailnet に公開する（root が必要）

   ```sh
   sudo tailscale serve --bg --https=8450 unix:/run/user/$(id -u)/remote-herdr/remote-herdr.sock
   ```

5. スマホや PC のブラウザで `https://mybox.tail1234.ts.net:8450` を開く

### 常駐させる（Linux）

[contrib/remote-herdr.service](contrib/remote-herdr.service) の `-origin` と `-allow-user` を書き換えて
`~/.config/systemd/user/` に置き、次を実行します。

```sh
systemctl --user enable --now remote-herdr
sudo loginctl enable-linger $USER     # ログインしていなくても起動させる場合
```

更新するときは `make build` のあと `systemctl --user restart remote-herdr` です。
macOS 向けの launchd 設定はまだ用意していません。

## 使い方

- **一覧**: herdr の全セッションが「承認が必要 / 作業中 / そのほか」に分かれて並びます。
  PC 幅では左のサイドバー、スマホでは最初の画面です。
- **会話**: Claude の出力と、ツール呼び出し（1 行ずつ、タップで中身）を表示します。
  下の入力欄から送ったプロンプトは、端末の Claude にそのまま入力されます。
  作業中は送信ボタンが中断ボタン（Esc）に変わります。
- **答える**: 権限確認や選択式の質問が出ると、選択肢がボタンで並びます。
  複数選択は項目を選んでから Submit。自由入力（Type something）だけは端末で答えます。
- **ファイル**: セッションの作業ディレクトリを読み取り専用で見られます（画像・PDF・md・tex など）。
  `.` で始まるファイルやディレクトリは表示しません。
- **スマホ**: ホーム画面に追加すると全画面で使えます。戻る操作（端からのスワイプ）で一覧に戻ります。

### 注意

- 端末で入力の途中にスマホから送ると、入力欄の文字と混ざります。
- `!` で始まる入力（Claude Code のシェル実行）は送れません。スラッシュコマンドも既定では送れません
  （`-allow-slash /compact` のように個別に許可できます）。

## 安全のために

- 待ち受けは unix ソケット（権限 0600）だけで、外へは `tailscale serve` 経由でしか出ません。
- `-allow-user` 以外の Tailscale ユーザーは 403 になります。tailnet の ACL でも、
  このポートに届く端末を自分のものに絞ってください。
- **Funnel（インターネット公開）では使わないでください。** Tailscale のユーザー識別が付かず、前提が崩れます。
- 作業ディレクトリがホームのセッションでは、ホーム配下（ドット以外）を閲覧できます。

## うまく動かないとき

| 症状 | 確認すること |
|---|---|
| 画面が真っ白 | ブラウザがスクリプトを止めていないか（Brave は Shields の「スクリプトをブロック」をこのサイトだけオフ） |
| `not allowed`（403） | `-allow-user` のログイン名が `tailscale whois` の User の Name と一致しているか |
| `unexpected host`（421） | `-origin` がブラウザで開いている URL（ポート込み）と一致しているか |
| 一覧が空・`herdr unavailable` | `herdr status server` の `socket:` を `-herdr-socket` に指定する |
| 会話が表示されない | `CLAUDE_CONFIG_DIR` を使っているなら、その下の `projects` を `-projects-dir` に指定する |
| 選択肢が出ない | Claude Code の画面の形が変わった可能性。端末で答え、画面の内容を issue で共有する |

原因を調べるときは `-access-log` を付けると、リクエストごとの結果が記録されます（プロンプトの本文は記録しません）。

## オプション

| フラグ | 既定 | 説明 |
|---|---|---|
| `-origin` | （必須） | ブラウザから見た URL（`https://host:port`） |
| `-allow-user` | （必須） | 許可する Tailscale ログイン（カンマ区切り） |
| `-listen` | `unix:$XDG_RUNTIME_DIR/remote-herdr/remote-herdr.sock` | `unix:/path` か `tcp:127.0.0.1:PORT`（loopback のみ） |
| `-allow-slash` | なし | リモートから送ってよいスラッシュコマンド（カンマ区切り） |
| `-herdr-socket` | `$HERDR_SOCKET_PATH` か `~/.config/herdr/herdr.sock` | herdr の API ソケット |
| `-projects-dir` | `$CLAUDE_CONFIG_DIR/projects`（未設定なら `~/.claude/projects`） | Claude Code の会話記録 |
| `-sessions-dir` | `$CLAUDE_CONFIG_DIR/sessions`（未設定なら `~/.claude/sessions`） | 動いているセッションの一覧。fork したセッションの会話記録を探すのに使う |
| `-host-name` | hostname | 画面に出すマシン名（将来の複数ホスト対応用） |
| `-access-log` | オフ | 調査用のアクセスログ |

## 開発

```sh
make test     # go vet + go test + svelte-check
make build
```

- サーバー: Go 標準ライブラリのみ（`cmd/`, `internal/`）
- UI: Svelte 5 + Vite + TypeScript、pnpm（`web/`）。ビルド結果はバイナリに埋め込まれます。
- 作業ルールは [AGENTS.md](AGENTS.md)。安全上の約束を変える変更にはテストを付けてください。

## ライセンス

MIT（[LICENSE](LICENSE)）
