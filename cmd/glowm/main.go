package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/atani/glowm/internal/input"
	"github.com/atani/glowm/internal/markdown"
	"github.com/atani/glowm/internal/mermaid"
	"github.com/atani/glowm/internal/pager"
	"github.com/atani/glowm/internal/render"
	"github.com/atani/glowm/internal/termimage"
	"github.com/atani/glowm/internal/terminal"
)

// Version information (set by goreleaser ldflags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		width       = flag.Int("w", 0, "word wrap width")
		style       = flag.String("s", "auto", "style name or JSON path")
		usePager    = flag.Bool("p", false, "force pager output")
		noPager     = flag.Bool("no-pager", false, "disable pager")
		pdf         = flag.Bool("pdf", false, "output mermaid diagrams as PDF to stdout")
		showVersion = flag.Bool("version", false, "show version information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("glowm %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

	md, err := input.Read(flag.Args())
	if err != nil {
		exitWithError(err)
	}

	if *pdf {
		result := markdown.ExtractMermaid(md, false)
		if len(result.Blocks) == 0 {
			exitWithError(errors.New("no mermaid blocks found"))
		}
		pdfBytes, err := mermaid.RenderPDF(result.Blocks)
		if err != nil {
			exitWithError(err)
		}
		if _, err := os.Stdout.Write(pdfBytes); err != nil {
			exitWithError(err)
		}
		return
	}

	stdoutTTY := terminal.StdoutIsTTY()
	imageFormat := termimage.Detect()

	usePagerDefault := stdoutTTY && !*noPager
	usePagerFinal := *usePager || usePagerDefault

	if stdoutTTY && imageFormat != termimage.FormatNone {
		result := markdown.ExtractMermaidWithMarkers(md)
		if len(result.Blocks) > 0 {
			w := *width
			if w == 0 {
				w = terminal.StdoutWidth(80)
			}
			images, err := mermaid.RenderPNGs(result.Blocks, w)
			if err != nil {
				exitWithError(err)
			}
			output, err := render.ANSI(result.Markdown, render.RenderOptions{
				Width: w,
				Style: *style,
				TTY:   stdoutTTY,
			})
			if err != nil {
				exitWithError(err)
			}
			output = termimage.ReplaceMarkersWithImages(output, result.Markers, images, imageFormat, w)

			if usePagerFinal {
				if err := pager.Page(output); err != nil {
					exitWithError(err)
				}
				return
			}
			if _, err := fmt.Fprint(os.Stdout, output); err != nil {
				exitWithError(err)
			}
			return
		}
	}

	keepBlocks := stdoutTTY
	result := markdown.ExtractMermaid(md, keepBlocks)

	w := *width
	if w == 0 {
		w = terminal.StdoutWidth(80)
	}

	output, err := render.ANSI(result.Markdown, render.RenderOptions{
		Width: w,
		Style: *style,
		TTY:   stdoutTTY,
	})
	if err != nil {
		exitWithError(err)
	}

	if usePagerFinal && stdoutTTY {
		if err := pager.Page(output); err != nil {
			exitWithError(err)
		}
		return
	}

	if _, err := fmt.Fprint(os.Stdout, output); err != nil {
		exitWithError(err)
	}
}

func exitWithError(err error) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, render.FormatError(err))
	os.Exit(1)
}
