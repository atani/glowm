# glowm launch kit

Use this when introducing glowm to English-speaking developer communities.

## One-line positioning

A terminal Markdown viewer for architecture docs, with inline Mermaid diagrams and PDF export.

## Short description

`glowm` is a Glow-like Markdown CLI for terminal-first documentation workflows. It renders Markdown in the terminal, displays Mermaid diagrams inline on iTerm2, Kitty, and Ghostty, and can export diagrams to PDF.

## Core message

Stop opening a browser just to preview Mermaid diagrams in Markdown. If your architecture docs, ADRs, runbooks, and READMEs live next to your code, glowm lets you read them from the terminal.

## Show HN

Title:

```text
Show HN: glowm – view Markdown with inline Mermaid diagrams in your terminal
```

Body:

```text
Hi HN,

I built glowm, a terminal Markdown viewer for architecture docs and Mermaid-heavy READMEs.

It is inspired by Glow, but adds Mermaid rendering for terminals that support inline images, including iTerm2, Kitty, and Ghostty. It can also export Mermaid diagrams to PDF.

Why I built it: I keep design docs and ADRs next to code, and I wanted to preview Mermaid diagrams without switching to a browser or editor preview pane.

Install:
  brew tap atani/tap
  brew install glowm

Or:
  go install github.com/atani/glowm/cmd/glowm@latest

GitHub: https://github.com/atani/glowm

I would love feedback from people who keep architecture docs, ADRs, or Mermaid diagrams in Git.
```

## Reddit: r/commandline

Title:

```text
I built a terminal Markdown viewer that renders Mermaid diagrams inline
```

Body:

```text
I built glowm, a terminal-first Markdown viewer for docs-as-code workflows.

It renders Markdown in the terminal and can render Mermaid diagrams inline on iTerm2, Kitty, and Ghostty. It also supports exporting Mermaid diagrams to PDF.

The use case is simple: if your READMEs, ADRs, runbooks, or architecture docs live in Git and include Mermaid diagrams, you can preview them without opening a browser or editor preview.

GitHub: https://github.com/atani/glowm

I would appreciate feedback from people who live in the terminal or maintain Markdown-heavy repos.
```

## Reddit: r/golang

Title:

```text
I built glowm, a Go CLI for reading Markdown docs with inline Mermaid diagrams
```

Body:

```text
I built glowm in Go: a Markdown CLI for terminal-first documentation workflows.

It is Glow-like, but it renders Mermaid diagrams inline on modern terminals such as iTerm2, Kitty, and Ghostty, and can export Mermaid diagrams to PDF.

Repo: https://github.com/atani/glowm

I would especially appreciate feedback on packaging, terminal image support, and cross-platform behavior.
```

## X / Bluesky

```text
I built glowm: a terminal Markdown viewer for architecture docs.

It renders Mermaid diagrams inline in iTerm2, Kitty, and Ghostty, and can export diagrams to PDF.

Useful if your READMEs, ADRs, or design docs live next to your code.

https://github.com/atani/glowm
```

## Launch order

1. r/commandline
2. r/golang
3. terminal-specific communities: Kitty, Ghostty, iTerm2 users
4. Mermaid / docs-as-code communities
5. Lobsters
6. Show HN

## Response principles

- Lead with the pain: switching to a browser just to preview Mermaid diagrams.
- Ask for terminal compatibility feedback.
- Do not describe glowm as a full documentation platform.
- Be clear that Chrome/Chromium is required for Mermaid rendering and PDF export.
- If people compare it to Glow, say it is Glow-like but Mermaid-focused.
