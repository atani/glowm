# ADR 0001: glowm 初期要件とスコープ

## Status
Accepted

## Context
- glowmはGlow相当のMarkdown表示機能を持つCLIとして構想された。
- mermaid記法がある場合はmermaid-cli相当の変換機能を提供し、出力はPDFのみをサポートする。
- 外部コマンドの実行（glow / mermaid-cliの呼び出し）は禁止し、Goで軽量に実装する。
- 入力は最小構成として「ローカルファイル」「STDIN」のみに限定する。
- 出力は基本的に標準出力。PDFモードも標準出力に出す。

## Decision
- コマンド名は `glowm` とする。
- 入力はローカルファイルとSTDINのみ対応する。
- PDF出力はオプション（例: `--pdf`）で有効化し、mermaidブロックから生成したPDFのみをstdoutに出力する。
- 複数のmermaidブロックは統合して1つのPDFにする。
- ANSIレンダリング可能な端末（stdoutがTTY）の場合はmermaidブロックをそのままコードブロック表示する。
  非TTYでは代替表現に置換する。
- 実装はGo、外部コマンド実行に依存しない。

## Consequences
- URL/GitHub/GitLab入力は当面サポートしない。
- SVG/PNG出力、Markdown再出力（画像参照に置換）は対象外。
- TUI機能は必須ではなく、必要になれば再検討する。
