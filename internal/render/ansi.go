package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour/v2"
	"github.com/charmbracelet/glamour/v2/ansi"
	"github.com/charmbracelet/glamour/v2/styles"
	"github.com/muesli/termenv"
)

type RenderOptions struct {
	Width int
	Style string
	TTY   bool
	// ShowLinkURLs keeps the raw URL that glamour appends after a link's
	// text. When false (the default) the URL is hidden on a TTY so that
	// [text](url) renders as just "text"; the text stays clickable via the
	// OSC 8 hyperlink that glamour still emits. On a non-TTY (piped or
	// redirected output) the URL is always kept, since OSC 8 links are not
	// clickable there and hiding the URL would lose the link target.
	ShowLinkURLs bool
}

func ANSI(md string, opts RenderOptions) (string, error) {
	options := []glamour.TermRendererOption{}

	if opts.Width > 0 {
		options = append(options, glamour.WithWordWrap(opts.Width))
	}

	style := strings.TrimSpace(opts.Style)
	// hideURLs is gated on TTY: on a non-TTY the appended URL is the only way
	// to recover the link target, so we never hide it there.
	hideURLs := opts.TTY && !opts.ShowLinkURLs

	cfg, ok := styleConfig(style, opts.TTY)
	if ok {
		if hideURLs {
			hideLinkURLs(&cfg)
		}
		options = append(options, glamour.WithStyles(cfg))
	} else {
		// Custom JSON style file: glamour loads it directly and we leave the
		// link styling under the user's control.
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

// styleConfig resolves a style name to a concrete StyleConfig. The second
// return value is false for custom JSON style paths, which the caller loads
// via glamour directly. tty selects the dark/light variant for the "auto"
// style.
func styleConfig(style string, tty bool) (ansi.StyleConfig, bool) {
	switch style {
	case "", "auto":
		if !tty {
			return styles.NoTTYStyleConfig, true
		}
		cfg := styles.DarkStyleConfig
		if !termenv.HasDarkBackground() {
			cfg = styles.LightStyleConfig
		}
		return withoutHeadingPrefix(cfg), true
	case "dark":
		return withoutHeadingPrefix(styles.DarkStyleConfig), true
	case "light":
		return withoutHeadingPrefix(styles.LightStyleConfig), true
	default:
		if base, found := styles.DefaultStyles[style]; found {
			return *base, true
		}
		return ansi.StyleConfig{}, false
	}
}

// hideLinkURLs suppresses the raw URL that glamour appends after a link's
// text. glamour renders a link as "<text> <url>"; setting the Link style's
// Format to an empty-producing template drops the URL token (and the OSC 8
// escape it carries) while leaving the link text — which glamour wraps in its
// own OSC 8 hyperlink — intact and clickable.
func hideLinkURLs(cfg *ansi.StyleConfig) {
	cfg.Link.Format = `{{""}}`
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
