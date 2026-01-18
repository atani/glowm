# glowm 要件定義（ドラフト）

## 目的
- Glow相当のMarkdown表示機能を提供し、Markdown内のmermaid記法が存在する場合はmermaid-cli相当の変換機能を提供する。
- 基本は標準出力。PDF出力はオプションで標準出力に出す。

## 参照元（要点）
- GlowはCLI/TUIのMarkdownレンダラで、ファイル/STDIN/URL/GitHub等から読み込みできる。幅指定（-w）、ページャ（-p）、スタイル（-s）を持ち、設定ファイルで永続化できる。
- mermaid-cliはmmdファイルをSVG/PNG/PDFに変換でき、Markdown内の```mermaidブロックを検出して画像参照に置換する変換もできる。STDIN入力も可能。

## スコープ
### 対象機能（必須）
- **入力**
  - ファイルパス
  - STDIN
- **出力**
  - デフォルト: MarkdownをANSIレンダリングして標準出力。
  - stdoutがTTYかつ対応ターミナル(iTerm2/Kitty)ならmermaidブロックを画像表示。
  - 非対応ターミナルや非TTYではコードブロック表示/代替表現に置換。
  - PDFオプション指定時: mermaid記法からPDFを生成し、標準出力へ出す（PDFのみ）。
- **mermaid対応（mermaid-cli相当）**
  - Markdown内の```mermaidブロックを検出する。
  - 変換対象のmermaid定義からPDFを生成する。
- **表示オプション（Glow相当）**
  - 幅指定（-w）
  - ページャ出力（-p）
  - スタイル選択（-s; dark/light/JSON）
  - 可能なら設定ファイル（glow.yml相当）対応。

### 非対象（現時点）
- SVG/PNG出力
- 画像の埋め込みを行うMarkdown再出力（mermaid-cliのMarkdown変換を「そのまま」返すモード）
- TUI機能（必要なら後で検討）

## 仕様（ドラフト）
### コマンド
- `glowm [input] [options]`

### 入力
- `glowm README.md`
- `echo "..." | glowm -`

### 出力モード
- **標準**: Glow相当のANSIレンダリングをstdoutへ出力。
- **PDFモード**: `--pdf` で有効化。mermaidブロックをPDFとしてstdoutへ出力（PDFのみ）。
  - 複数mermaidブロックは統合して1つのPDFにする。

### 例（案）
- `glowm README.md`
- `glowm --pdf README.md > diagram.pdf`

## 制約
- Goで実装。
- 外部コマンド実行に依存しない（glow / mermaid-cliを呼び出さない）。
- 外部依存は最小限に。

## 主要な未決事項
1) **ANSIレンダリング時のmermaidブロックの扱い**
   - iTerm2/Kitty: 画像表示
   - それ以外: コードブロック表示または代替表現

