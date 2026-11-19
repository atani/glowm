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
	"github.com/atani/glowm/internal/watch"
)

// Version information (set by goreleaser ldflags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// options holds parsed command-line flags.
type options struct {
	width        int
	style        string
	usePager     bool
	noPager      bool
	pdf          bool
	showVersion  bool
	showLinkURLs bool
	watch        bool
	noWatch      bool
	positional   []string
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
	fs.BoolVar(&opts.showLinkURLs, "show-link-urls", false, "show raw link URLs instead of just the link text")
	fs.BoolVar(&opts.watch, "watch", false, "watch the input file and re-render on changes")
	fs.BoolVar(&opts.noWatch, "no-watch", false, "disable watch (overrides config)")
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

	cfg := config.Load()
	if !mermaid.IsKnownTheme(cfg.Mermaid.Theme) {
		fmt.Fprintf(stderr, "glowm: unknown mermaid theme %q, using default\n", cfg.Mermaid.Theme)
	}

	if opts.pdf {
		return runPDF(md, cfg.Mermaid.Theme, stdout, stderr)
	}

	stdoutTTY := terminal.StdoutIsTTY()
	imageFormat := termimage.Detect()

	pagerMode := pager.Mode(strings.ToLower(cfg.Pager.Mode))
	if !pager.ValidMode(pagerMode) {
		fmt.Fprintf(stderr, "glowm: unknown pager mode %q, using more\n", cfg.Pager.Mode)
		pagerMode = pager.ModeMore
	}
	shouldUsePager := opts.usePager || (stdoutTTY && !opts.noPager)

	// Watch mode re-renders and refreshes the pager when the file changes. It
	// only applies to a real file rendered to a terminal; otherwise fall back to
	// a one-shot render with a note.
	shouldWatch := opts.watch || (cfg.Pager.Watch && !opts.noWatch)
	if shouldWatch {
		filePath := ""
		if len(opts.positional) == 1 && opts.positional[0] != "-" {
			filePath = opts.positional[0]
		}
		if filePath == "" || !stdoutTTY {
			fmt.Fprintln(stderr, "glowm: watch ignored (requires a file argument and a terminal)")
		} else {
			return runWatch(filePath, opts, imageFormat, cfg.Mermaid.Theme, stderr)
		}
	}

	if stdoutTTY && imageFormat != termimage.FormatNone {
		handled, code := runWithImages(md, opts, stdoutTTY, imageFormat, pagerMode, shouldUsePager, cfg.Mermaid.Theme, stdout, stderr)
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
		Width:        w,
		Style:        opts.style,
		TTY:          stdoutTTY,
		ShowLinkURLs: opts.showLinkURLs,
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
func runPDF(md, mermaidTheme string, stdout, stderr io.Writer) int {
	result, err := markdown.ExtractMermaid(md, false)
	if err != nil {
		return fail(stderr, err)
	}
	if len(result.Blocks) == 0 {
		return fail(stderr, errors.New("no mermaid blocks found"))
	}
	pdfBytes, err := mermaid.RenderPDF(result.Blocks, effectiveMermaidTheme(mermaidTheme))
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
func runWithImages(md string, opts options, stdoutTTY bool, imageFormat termimage.Format, pagerMode pager.Mode, shouldUsePager bool, mermaidTheme string, stdout, stderr io.Writer) (handled bool, code int) {
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
	images, renderErr := mermaid.RenderPNGs(result.Blocks, w, effectiveMermaidTheme(mermaidTheme))
	if renderErr != nil {
		fmt.Fprintf(stderr, "warning: mermaid rendering failed: %v\n", renderErr)
		return false, 0
	}
	output, err := render.ANSI(result.Markdown, render.RenderOptions{
		Width:        w,
		Style:        opts.style,
		TTY:          stdoutTTY,
		ShowLinkURLs: opts.showLinkURLs,
	})
	if err != nil {
		return true, fail(stderr, err)
	}
	if shouldUsePager {
		// less mode on a Kitty-graphics terminal scrolls images smoothly by
		// cropping them per row, so it needs the raw images and the unmodified
		// marker lines rather than pre-baked image escapes.
		if pagerMode == pager.ModeLess && imageFormat == termimage.FormatKitty {
			if err := pager.PageLessKitty(output, result.Markers, images, w); err != nil {
				return true, fail(stderr, err)
			}
			return true, 0
		}
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

// runWatch renders path and keeps the less pager open, re-rendering and
// refreshing in place whenever the file changes. Uses the smooth Kitty pager on
// Kitty-graphics terminals and the text pager elsewhere.
func runWatch(path string, opts options, imageFormat termimage.Format, mermaidTheme string, stderr io.Writer) int {
	w := opts.width
	if w == 0 {
		w = terminal.StdoutWidth(80)
	}
	kitty := imageFormat == termimage.FormatKitty
	theme := effectiveMermaidTheme(mermaidTheme)

	// Resolve "auto" to a concrete style now, before the pager takes the
	// terminal into raw mode. Re-rendering with "auto" on each reload would make
	// glamour re-query the terminal background mid-pager, racing our input
	// reader and flipping the text theme.
	style := opts.style
	if s := strings.TrimSpace(style); s == "" || strings.EqualFold(s, "auto") {
		style = render.AutoStyle()
	}

	renderContent := func() (pager.Content, error) {
		md, err := input.Read([]string{path})
		if err != nil {
			return pager.Content{}, err
		}
		if kitty {
			res, err := markdown.ExtractMermaidWithMarkers(md)
			if err != nil {
				return pager.Content{}, err
			}
			if len(res.Blocks) > 0 {
				if images, rerr := mermaid.RenderPNGs(res.Blocks, w, theme); rerr == nil {
					out, err := render.ANSI(res.Markdown, render.RenderOptions{Width: w, Style: style, TTY: true, ShowLinkURLs: opts.showLinkURLs})
					if err != nil {
						return pager.Content{}, err
					}
					return pager.Content{Output: out, Markers: res.Markers, Images: images, WidthCells: w}, nil
				}
				// Mermaid render failed; fall through to plain text rendering so
				// diagrams degrade to code blocks rather than showing raw markers.
			}
		}
		res, err := markdown.ExtractMermaid(md, true)
		if err != nil {
			return pager.Content{}, err
		}
		out, err := render.ANSI(res.Markdown, render.RenderOptions{Width: w, Style: style, TTY: true, ShowLinkURLs: opts.showLinkURLs})
		if err != nil {
			return pager.Content{}, err
		}
		return pager.Content{Output: out}, nil
	}

	initial, err := renderContent()
	if err != nil {
		return fail(stderr, err)
	}

	watcher := watch.New(path, 0, 0)
	watcher.Start()
	defer watcher.Stop()

	if kitty {
		if err := pager.PageLessKittyWatch(initial, watcher.C, renderContent); err != nil {
			return fail(stderr, err)
		}
	} else {
		if err := pager.PageLessWatch(initial, watcher.C, renderContent); err != nil {
			return fail(stderr, err)
		}
	}
	return 0
}

func replaceMarkersForPagerMode(output string, markers []string, images [][]byte, imageFormat termimage.Format, width int, pagerMode pager.Mode) string {
	if pagerMode == pager.ModeMore || pagerMode == pager.ModeLess {
		return termimage.ReplaceMarkersWithImagesForPager(output, markers, images, imageFormat, width)
	}
	return termimage.ReplaceMarkersWithImages(output, markers, images, imageFormat, width)
}

// effectiveMermaidTheme resolves the configured theme. "auto" (or empty) becomes
// "dark" or "default" based on the detected terminal background; any explicit
// theme is passed through unchanged. Detection falls back to the light default
// when the background can't be determined (e.g. output is not a terminal, as
// when exporting a PDF to a file).
func effectiveMermaidTheme(configured string) string {
	if configured != "" && !strings.EqualFold(configured, "auto") {
		return configured
	}
	if dark, ok := terminal.BackgroundIsDark(); ok && dark {
		return "dark"
	}
	return "default"
}

// fail writes a formatted error to stderr and returns exit code 1.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, render.FormatError(err))
	return 1
}
