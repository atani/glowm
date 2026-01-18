package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"
)

type RenderOptions struct {
	Width int
	Style string
	TTY   bool
}

func ANSI(md string, opts RenderOptions) (string, error) {
	options := []glamour.TermRendererOption{}

	if opts.Width > 0 {
		options = append(options, glamour.WithWordWrap(opts.Width))
	}

	style := strings.TrimSpace(opts.Style)
	if style == "" || style == "auto" {
		if !opts.TTY {
			options = append(options, glamour.WithStandardStyle("notty"))
		} else {
			cfg := styles.DarkStyleConfig
			if !termenv.HasDarkBackground() {
				cfg = styles.LightStyleConfig
			}
			cfg = withoutHeadingPrefix(cfg)
			options = append(options, glamour.WithStyles(cfg))
		}
	} else if style == "dark" {
		cfg := withoutHeadingPrefix(styles.DarkStyleConfig)
		options = append(options, glamour.WithStyles(cfg))
	} else if style == "light" {
		cfg := withoutHeadingPrefix(styles.LightStyleConfig)
		options = append(options, glamour.WithStyles(cfg))
	} else if style == "notty" || style == "ascii" || style == "dracula" || style == "pink" {
		options = append(options, glamour.WithStandardStyle(style))
	} else {
		options = append(options, glamour.WithStylesFromJSONFile(style))
	}

	renderer, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return "", err
	}
	defer renderer.Close()

	out, err := renderer.Render(md)
	if err != nil {
		return "", err
	}
	return out, nil
}

func withoutHeadingPrefix(cfg ansi.StyleConfig) ansi.StyleConfig {
	cfg.Heading.Prefix = ""
	cfg.H1.Prefix = ""
	cfg.H2.Prefix = ""
	cfg.H3.Prefix = ""
	cfg.H4.Prefix = ""
	cfg.H5.Prefix = ""
	cfg.H6.Prefix = ""
	return cfg
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("error: %v", err)
}
