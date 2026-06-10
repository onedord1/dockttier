// Package style defines the dockttier visual language: the active color
// palette, status icons, badges, dividers, section labels, progress bars and
// the summary footer. The palette, progress-bar glyphs and spinner are all
// driven by a swappable Theme so users can pick a look in config.toml.
package style

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Theme is a complete, swappable visual scheme: a 14-color palette plus the
// progress-bar glyphs and spinner frames that give each theme its own feel.
type Theme struct {
	Name string

	// Palette (6-digit hex strings).
	Brand     string
	Text      string
	TextMuted string
	TextDim   string
	Green     string
	Cyan      string
	Blue      string
	Teal      string
	Yellow    string
	Orange    string
	Red       string
	Purple    string
	Border    string
	BgPanel   string
	BarEmpty  string // color of the empty portion of a progress bar

	// Progress / load-bar style.
	BarFilledGlyph string // e.g. "█"
	BarEmptyGlyph  string // e.g. "░"

	// Spinner animation frames.
	Spinner []string
}

// Active palette values. These are vars (not consts) so a Theme can replace
// them at startup; renderers read them at render time.
var (
	HexBrand     = "#00d4aa"
	HexBrandDim  = "#00a885"
	HexText      = "#e6edf3"
	HexTextMuted = "#7d8590"
	HexTextDim   = "#484f58"
	HexGreen     = "#3fb950"
	HexCyan      = "#39d353"
	HexBlue      = "#58a6ff"
	HexTeal      = "#2ea5a0"
	HexYellow    = "#d29922"
	HexOrange    = "#f0883e"
	HexRed       = "#f85149"
	HexPurple    = "#bc8cff"
	HexBorder    = "#30363d"
	HexBgPanel   = "#161b22"
	HexBarEmpty  = "#1c2128"
)

// Active progress/spinner glyphs.
var (
	barFilledGlyph = "█"
	barEmptyGlyph  = "░"
	spinnerFrames  = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// Lipgloss styles, rebuilt by rebuildStyles whenever the theme changes.
var (
	Brand    lipgloss.Style
	BrandDim lipgloss.Style
	Text     lipgloss.Style
	Muted    lipgloss.Style
	Dim      lipgloss.Style
	Green    lipgloss.Style
	Cyan     lipgloss.Style
	Blue     lipgloss.Style
	Teal     lipgloss.Style
	Yellow   lipgloss.Style
	Orange   lipgloss.Style
	Red      lipgloss.Style
	Purple   lipgloss.Style
	Border   lipgloss.Style
	Bold     = lipgloss.NewStyle().Bold(true)

	emptyStyle lipgloss.Style
)

var (
	brandOverride string
	disableEmoji  bool
)

// Apply installs a theme: it copies the palette and glyphs into the active
// values and rebuilds the derived styles and icon table.
func Apply(t Theme) {
	HexBrand = orDefault(t.Brand, HexBrand)
	HexText = orDefault(t.Text, HexText)
	HexTextMuted = orDefault(t.TextMuted, HexTextMuted)
	HexTextDim = orDefault(t.TextDim, HexTextDim)
	HexGreen = orDefault(t.Green, HexGreen)
	HexCyan = orDefault(t.Cyan, HexCyan)
	HexBlue = orDefault(t.Blue, HexBlue)
	HexTeal = orDefault(t.Teal, HexTeal)
	HexYellow = orDefault(t.Yellow, HexYellow)
	HexOrange = orDefault(t.Orange, HexOrange)
	HexRed = orDefault(t.Red, HexRed)
	HexPurple = orDefault(t.Purple, HexPurple)
	HexBorder = orDefault(t.Border, HexBorder)
	HexBgPanel = orDefault(t.BgPanel, HexBgPanel)
	HexBarEmpty = orDefault(t.BarEmpty, HexBarEmpty)

	if t.BarFilledGlyph != "" {
		barFilledGlyph = t.BarFilledGlyph
	}
	if t.BarEmptyGlyph != "" {
		barEmptyGlyph = t.BarEmptyGlyph
	}
	if len(t.Spinner) > 0 {
		spinnerFrames = t.Spinner
	}
	if brandOverride != "" {
		HexBrand = brandOverride
	}
	rebuildStyles()
}

// ApplyPreset installs a named built-in theme. Unknown names keep the default.
func ApplyPreset(name string) {
	if name == "" {
		return
	}
	if t, ok := Presets[name]; ok {
		Apply(t)
	}
}

func rebuildStyles() {
	fg := func(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
	Brand = fg(HexBrand)
	BrandDim = fg(HexBrandDim)
	Text = fg(HexText)
	Muted = fg(HexTextMuted)
	Dim = fg(HexTextDim)
	Green = fg(HexGreen)
	Cyan = fg(HexCyan)
	Blue = fg(HexBlue)
	Teal = fg(HexTeal)
	Yellow = fg(HexYellow)
	Orange = fg(HexOrange)
	Red = fg(HexRed)
	Purple = fg(HexPurple)
	Border = fg(HexBorder)
	emptyStyle = fg(HexBarEmpty)
	buildIcons()
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// SetBrandColor overrides the brand color independently of the active theme.
func SetBrandColor(hex string) {
	if hex == "" {
		return
	}
	brandOverride = hex
	HexBrand = hex
	Brand = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	buildIcons()
}

// SetDisableEmoji toggles ASCII fallback glyphs in place of Unicode icons.
func SetDisableEmoji(v bool) { disableEmoji = v }

// width tracking (responsive to SIGWINCH).
var (
	widthMu  sync.RWMutex
	curWidth = 80
)

const minWidth = 60

// RefreshWidth re-queries the terminal width. Falls back to 80 columns when it
// cannot be determined, and never reports below the 60-column minimum.
func RefreshWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w == 0 {
		w = 80
	}
	if w < minWidth {
		w = minWidth
	}
	widthMu.Lock()
	curWidth = w
	widthMu.Unlock()
	return w
}

// Width returns the most recently observed terminal width.
func Width() int {
	widthMu.RLock()
	defer widthMu.RUnlock()
	return curWidth
}

func init() {
	rebuildStyles()
	RefreshWidth()
}
