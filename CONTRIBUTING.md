# Contributing to glowm

Thanks for your interest in contributing to glowm.

## Development setup

Requirements:

- Go, matching the version in `go.mod`
- Chrome or Chromium for Mermaid rendering and PDF-related tests

Run tests:

```bash
go test ./...
```

Run the CLI locally:

```bash
go run ./cmd/glowm README.md
```

## Useful areas for contributions

- Terminal image compatibility: iTerm2, Kitty, Ghostty, tmux, SSH
- Mermaid rendering edge cases
- Markdown rendering quality
- PDF export behavior
- Packaging and installation improvements
- Documentation examples and demo material

## Pull request guidelines

- Keep changes focused and small when possible
- Add or update tests for behavior changes
- Update README/docs for user-facing changes
- Run `go test ./...` before opening a PR

## Reporting terminal compatibility issues

Please include:

- OS and version
- Terminal app and version
- Whether you are using tmux, SSH, or a remote dev container
- Output of `glowm --version`
- A minimal Markdown/Mermaid example that reproduces the issue
