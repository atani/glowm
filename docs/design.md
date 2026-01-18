# Design

## Architecture Overview
- `cmd/glowm`: CLI エントリ
- `internal/input`: ファイル/STDIN 読み込み
- `internal/markdown`: Mermaid ブロック抽出/置換
- `internal/render`: ANSI レンダリング
- `internal/mermaid`: Mermaid PDF/PNG 生成
- `internal/termimage`: iTerm2/Kitty 画像出力

## Data Flow
1) 入力取得（ファイル/STDIN）
2) Mermaid ブロック抽出
3) `--pdf` の場合: Mermaid 定義を統合 → PDF 生成 → stdout
4) それ以外: 対応ターミナルなら Mermaid を PNG 生成 → ANSI 出力へ差し込み
5) 非対応ターミナル: Mermaid ブロックを表示方針に応じて置換 → ANSI レンダリング → stdout

## Storage Model
- 永続ストレージなし
- 一時的にメモリ内で Markdown と Mermaid 定義を保持

## APIs and Integrations
- ANSI レンダリング: `github.com/charmbracelet/glamour`
- Mermaid PDF: Go から Mermaid JS を実行し、ヘッドレスブラウザで PDF を生成する方式
  - `chromedp` 系ライブラリを利用し、HTML に Mermaid JS を埋め込み（v10.9.5 を同梱）

## Permissions and Security
- ネットワークアクセス不要（埋め込み JS を利用）
- ローカルファイル/STDINのみを扱う

## Error Handling
- 入力ファイル不存在: 明確なエラー
- Chrome/Chromium が無い場合は PDF 生成に失敗し、理由をエラー表示
- Mermaid ブロック未検出で `--pdf`: エラーで終了
- PDF 生成失敗: 理由を stderr に出力

## Testing Strategy
- 自動テストは最小限。Mermaid 抽出/置換のユニットテストを追加
- 手動テストで ANSI 出力と PDF 出力を確認

## Rollout and Release Plan
- MVP リリース後にオプション拡張を検討
- 既存利用者への互換性を壊さない
