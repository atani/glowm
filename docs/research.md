# Research Log

## Scope
- Glow 相当の Markdown 表示機能の要点
- Mermaid の CLI/PDF 変換の要点
- Go で ANSI レンダリングと Mermaid PDF を実現する手段

## Assumptions
- 入力はローカルファイル/STDINのみ
- 外部コマンド実行は禁止
- 依存は最小限、ただし Mermaid の実行にはブラウザ相当の環境が必要になる可能性が高い

## Sources
- Mermaid JS dist (2026-01-18) https://cdn.jsdelivr.net/npm/mermaid@10.9.5/dist/mermaid.min.js
- Glow README (2026-01-18) https://github.com/charmbracelet/glow
- Mermaid CLI README (2026-01-18) https://github.com/mermaid-js/mermaid-cli
- Mermaid API Usage (2026-01-18) https://mermaid.js.org/config/usage.html
- Glamour README (2026-01-18) https://github.com/charmbracelet/glamour

## Findings
- Mermaid JS はブラウザで描画する前提の JS バンドルとして配布されている。 (source: Mermaid JS dist)
- Glow は CLI で Markdown を表示し、スタイルや幅指定などのレンダリング設定を持つ。 (source: Glow README)
- Mermaid CLI は Mermaid 定義から SVG/PNG/PDF を生成でき、Markdown 内の mermaid ブロック変換も可能。 (source: Mermaid CLI README)
- Mermaid は JS API (`mermaid.initialize` / `mermaid.run`) を提供し、描画を API 経由で行える。 (source: Mermaid API Usage)
- Glamour は Go で Markdown を ANSI 出力するためのレンダラで、スタイルやワードラップなどの設定が可能。 (source: Glamour README)

## Decisions and Tradeoffs
- ANSI レンダリングは Go ライブラリ（Glamour）を採用する想定。
- Mermaid PDF 生成は、Mermaid JS を実行できる環境（ヘッドレスブラウザ相当）を Go から利用する方針が現実的。
  - 代替案: Mermaid を自前実装するのはコストが高すぎるため採用しない。

## Risks
- Mermaid PDF 生成にはブラウザ/JS 実行環境が必要で、完全な依存排除が難しい。
- PDF 出力のページサイズやレイアウト最適化に調整が必要。

## Open Questions (with proposed resolution)
- Mermaid 実行方法: Go から Mermaid JS を実行する方式を選択し、ヘッドレスブラウザ利用を前提とする。
- PDF レイアウト: 初期は A4 固定、後続でオプション化を検討。
