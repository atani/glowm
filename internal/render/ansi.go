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

	cfg, builtin := styleConfig(style, opts.TTY)
	if builtin {
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

// AutoStyle resolves the "auto" style to a concrete "dark" or "light" by
// querying the terminal background. Callers that re-render repeatedly while
// holding the terminal in raw mode (e.g. watch mode) must call this once up
// front and pass the result, because the background query reads from the
// terminal and would otherwise race the pager's input reader on each redraw,
// flipping the theme.
func AutoStyle() string {
	if termenv.HasDarkBackground() {
		return "dark"
	}
	return "light"
}

// styleConfig resolves a style name to a concrete StyleConfig. The returned
// bool is true for built-in styles (the "auto"/dark/light variants and any
// name in glamour's DefaultStyles, e.g. notty/ascii/dracula/pink/tokyo-night);
// false means style is a path to a custom JSON style file that the caller must
// load via glamour itself. tty selects the dark/light variant for the "auto"
// style. The returned config is a value copy, so callers may mutate it (e.g.
// hideLinkURLs) without touching glamour's shared package-level styles.
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
// text. glamour renders a link as "<text> <url>"; the Link style's Format is a
// Go text/template applied to the URL token, and `{{""}}` is a template that
// renders to the empty string. (A plain empty Format string would instead be
// treated as "no formatting", leaving the URL intact, so the explicit
// empty-producing template is required.) This drops the URL token and the
// OSC 8 escape it carries, while the link text — which glamour wraps in its
// own OSC 8 hyperlink — stays intact and clickable.
//
// Known cosmetic limitation: glamour's link renderer prepends a hard-coded
// " " prefix to the URL element (ansi/link.go) and writes that prefix before
// the Format template runs, so a single space remains where the URL was. A
// link followed by text therefore shows a double space. There is no
// style-config-only way to remove it in the pinned glamour version, and a
// regex post-process would be more fragile than this template approach, so the
// residual space is accepted. See TestANSI_HiddenLinkLeavesResidualSpace.
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
