# PRD

## Summary
glowm はローカル Markdown を端末で美しく表示し、Mermaid 記法が含まれる場合は PDF を標準出力できる軽量 CLI です。外部コマンド実行には依存せず、Go で実装します。

## Goals and Success Metrics
- 1コマンドで Markdown を ANSI 出力できる
- `--pdf` 指定時に Mermaid 図を 1つの PDF として stdout に出せる
- 入力がファイル/STDINのみでも使い勝手が良い
- 失敗時に明確なエラーメッセージを返す

## Non-Goals
- URL/GitHub/GitLab の入力対応
- SVG/PNG 出力
- Markdown 再出力（Mermaid 図を画像参照に置換する用途）
- TUI（ブラウザ型 UI）

## Users and Personas
- 端末上で Markdown を読む開発者
- Mermaid 図を PDF 化して他ツールへ渡したい開発者

## User Stories
- ファイル指定で Markdown をきれいに読みたい
- パイプ入力の Markdown を ANSI で表示したい
- Mermaid ブロックを PDF にして他ツールへ渡したい

## Functional Requirements
- 入力: ローカルファイル / STDIN
- 出力(標準): ANSI レンダリング結果を stdout
- 出力(PDF): `--pdf` 指定時、Mermaid ブロックから生成した PDF を stdout
- Mermaid: ```mermaid ブロック検出
- 複数 Mermaid ブロックは 1 PDF に統合
- stdout が TTY の場合は Mermaid ブロックをそのままコード表示
- stdout が非 TTY の場合は Mermaid ブロックを代替表現へ置換
- 文字幅指定: `-w`
- スタイル指定: `-s` (dark/light/JSON)
- ページャ指定: `-p`

## Non-Functional Requirements
- Go 実装
- 外部コマンド実行に依存しない
- 依存関係は最小限
- 標準出力が PDF の場合は余計なログを出さない

## Privacy and Security
- ネットワークアクセスは不要
- 入力はローカル/STDINのみを扱う

## Analytics and Telemetry (if any)
- なし

## Milestones
- M1: CLI 雛形と ANSI レンダリング
- M2: Mermaid 抽出と PDF 生成
- M3: テスト手順とドキュメント整備

## Open Questions
- Mermaid のレンダリング方式（内部 JS 実行 vs. 事前埋め込み SVG）
- PDF 生成時のページサイズ/余白
