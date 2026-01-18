# PoC

## Objective
- Go から Mermaid 定義を PDF に変換できるか検証する

## Setup
- Go 環境
- Mermaid JS を埋め込んだ HTML を生成
- ヘッドレスブラウザ経由で PDF を生成する

## Steps Taken
- Mermaid JS (v10.9.5) を同梱
- `chromedp` を使った PDF 生成コードを実装

## Results
- Mermaid ブロックを含む Markdown から PDF を生成できた（1ページ）。
- Chrome/Chromium の存在が前提。

## Limitations
- Chrome/Chromium が必要になる可能性
- PDF のレイアウトは初期値（A4固定）

## Next Actions
- PDF レイアウト（用紙サイズ/余白）の調整を検討
- Mermaid オプション（テーマ等）の追加検討
