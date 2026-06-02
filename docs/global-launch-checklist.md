# Global launch checklist

This checklist is for maintainers preparing glowm for English-speaking developer communities.

## Repository metadata

Recommended GitHub settings:

- Description: `A terminal Markdown viewer for architecture docs, with inline Mermaid diagrams and PDF export.`
- Homepage: `https://github.com/atani/glowm`
- Enable Discussions
- Topics:
  - `markdown`
  - `mermaid`
  - `terminal`
  - `cli`
  - `tui`
  - `golang`
  - `developer-tools`
  - `documentation`
  - `architecture`
  - `adr`
  - `iterm2`
  - `kitty`
  - `ghostty`
  - `pdf`
  - `glow`

## Before posting

- README explains the Mermaid-in-terminal value in the first screen
- Demo GIF is visible near the top
- Install commands are visible and short
- Chrome/Chromium requirement is documented
- Comparison table explains why glowm is different from Glow, browser preview, and editor preview
- License, contributing guide, security policy, code of conduct, issue templates, and PR template exist
- Latest release assets are available
- `go test ./...` and CI are green

## Launch order

Use small, relevant communities first. Improve wording based on feedback before posting to larger communities.

1. `r/commandline`
2. `r/golang`
3. Kitty / Ghostty / iTerm2 user communities
4. Mermaid / docs-as-code communities
5. Lobsters
6. Show HN
7. Product Hunt or broader launch surfaces, only if the project has a polished demo video

## Post timing

- Avoid posting everywhere on the same day
- Post when you can reply for 2-4 hours
- Treat early comments as product research, not just promotion
- Update README wording if the same question appears repeatedly

## Response principles

- Lead with the browser-switching pain for Mermaid previews
- Say glowm is Glow-like, but Mermaid-focused
- Be explicit about supported terminal image protocols
- Ask for compatibility reports from terminal users
- Thank users who report install issues or terminal rendering edge cases

## Metrics to watch

- GitHub stars after each post
- Clones and Homebrew installs after each post
- Questions about Chrome/Chromium requirement
- Terminal compatibility reports: iTerm2, Kitty, Ghostty, tmux, SSH
- Requests for additional output formats or diagrams
