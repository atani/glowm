# glowm

Glow相当のMarkdown表示 + Mermaid PDF 出力を提供する軽量CLI。

## 使い方

```bash
# Markdown を ANSI 表示
./glowm README.md

# STDIN から表示
cat README.md | ./glowm -

# Mermaid を PDF で出力
./glowm --pdf README.md > diagram.pdf
```

## オプション
- `-w` 文字幅
- `-s` スタイル名 (dark/light/notty/auto) または JSON パス
- `-p` ページャ表示
- `--pdf` Mermaid を PDF で stdout に出力

## 依存
- Go
- Chrome/Chromium (Mermaid の PDF/画像生成時に必要)

## Mermaid 図の表示
- iTerm2 / Kitty では Mermaid 図を画像として表示します。
- それ以外のターミナルでは Mermaid ブロックをコード表示します。

## 参考
- `docs/`
