package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atani/glowm/internal/config"
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

// options holds parsed command-line flags.
type options struct {
	width       int
	style       string
	usePager    bool
	noPager     bool
	pdf         bool
	showVersion bool
	positional  []string
}

// parseFlags parses argv (excluding the program name) into options.
func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("glowm", flag.ContinueOnError)
	var opts options
	fs.IntVar(&opts.width, "w", 0, "word wrap width")
	fs.StringVar(&opts.style, "s", "auto", "style name or JSON path")
	fs.BoolVar(&opts.usePager, "p", false, "force pager output")
	fs.BoolVar(&opts.noPager, "no-pager", false, "disable pager")
	fs.BoolVar(&opts.pdf, "pdf", false, "output mermaid diagrams as PDF to stdout")
	fs.BoolVar(&opts.showVersion, "version", false, "show version information")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	opts.positional = fs.Args()
	return opts, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the program with the given args, writing primary output to
// stdout and diagnostics to stderr. It returns the process exit code.
//
// TTY-dependent rendering (terminal images, interactive pager) reads global
// terminal state and is only engaged when stdout is a real terminal; in the
// common non-TTY path the rendered markdown is written directly to stdout,
// which makes run fully testable with in-memory writers.
func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args)
	if err != nil {
		// flag already reported the error to its own output.
		return 2
	}

	if opts.showVersion {
		fmt.Fprintf(stdout, "glowm %s (commit: %s, built: %s)\n", version, commit, date)
		return 0
	}

	md, err := input.Read(opts.positional)
	if err != nil {
		return fail(stderr, err)
	}

	if opts.pdf {
		return runPDF(md, stdout, stderr)
	}

	stdoutTTY := terminal.StdoutIsTTY()
	imageFormat := termimage.Detect()

	cfg := config.Load()
	pagerMode := pager.Mode(strings.ToLower(cfg.Pager.Mode))
	if !pager.ValidMode(pagerMode) {
		fmt.Fprintf(stderr, "glowm: unknown pager mode %q, using more\n", cfg.Pager.Mode)
		pagerMode = pager.ModeMore
	}
	shouldUsePager := opts.usePager || (stdoutTTY && !opts.noPager)

	if stdoutTTY && imageFormat != termimage.FormatNone {
		handled, code := runWithImages(md, opts, stdoutTTY, imageFormat, pagerMode, shouldUsePager, stdout, stderr)
		if handled {
			return code
		}
	}

	result, err := markdown.ExtractMermaid(md, stdoutTTY)
	if err != nil {
		return fail(stderr, err)
	}

	w := opts.width
	if w == 0 {
		w = terminal.StdoutWidth(80)
	}

	output, err := render.ANSI(result.Markdown, render.RenderOptions{
		Width: w,
		Style: opts.style,
		TTY:   stdoutTTY,
	})
	if err != nil {
		return fail(stderr, err)
	}

	if shouldUsePager && stdoutTTY {
		if err := pager.PageWithMode(output, pagerMode); err != nil {
			return fail(stderr, err)
		}
		return 0
	}

	if _, err := fmt.Fprint(stdout, output); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// runPDF renders mermaid blocks to PDF bytes written to stdout.
func runPDF(md string, stdout, stderr io.Writer) int {
	result, err := markdown.ExtractMermaid(md, false)
	if err != nil {
		return fail(stderr, err)
	}
	if len(result.Blocks) == 0 {
		return fail(stderr, errors.New("no mermaid blocks found"))
	}
	pdfBytes, err := mermaid.RenderPDF(result.Blocks)
	if err != nil {
		return fail(stderr, err)
	}
	if _, err := stdout.Write(pdfBytes); err != nil {
		return fail(stderr, err)
	}
	return 0
}

// runWithImages renders markdown with inline terminal images. It returns
// handled=false when there are no mermaid blocks or rendering fails, so the
// caller can fall back to the text-only path.
func runWithImages(md string, opts options, stdoutTTY bool, imageFormat termimage.Format, pagerMode pager.Mode, shouldUsePager bool, stdout, stderr io.Writer) (handled bool, code int) {
	result, err := markdown.ExtractMermaidWithMarkers(md)
	if err != nil {
		return true, fail(stderr, err)
	}
	if len(result.Blocks) == 0 {
		return false, 0
	}
	w := opts.width
	if w == 0 {
		w = terminal.StdoutWidth(80)
	}
	images, renderErr := mermaid.RenderPNGs(result.Blocks, w)
	if renderErr != nil {
		fmt.Fprintf(stderr, "warning: mermaid rendering failed: %v\n", renderErr)
		return false, 0
	}
	output, err := render.ANSI(result.Markdown, render.RenderOptions{
		Width: w,
		Style: opts.style,
		TTY:   stdoutTTY,
	})
	if err != nil {
		return true, fail(stderr, err)
	}
	if shouldUsePager {
		output = replaceMarkersForPagerMode(output, result.Markers, images, imageFormat, w, pagerMode)
		if err := pager.PageWithMode(output, pagerMode); err != nil {
			return true, fail(stderr, err)
		}
		return true, 0
	}
	output = termimage.ReplaceMarkersWithImages(output, result.Markers, images, imageFormat, w)
	if _, err := fmt.Fprint(stdout, output); err != nil {
		return true, fail(stderr, err)
	}
	return true, 0
}

func replaceMarkersForPagerMode(output string, markers []string, images [][]byte, imageFormat termimage.Format, width int, pagerMode pager.Mode) string {
	if pagerMode == pager.ModeMore {
		return termimage.ReplaceMarkersWithImagesForPager(output, markers, images, imageFormat, width)
	}
	return termimage.ReplaceMarkersWithImages(output, markers, images, imageFormat, width)
}

// fail writes a formatted error to stderr and returns exit code 1.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, render.FormatError(err))
	return 1
}
