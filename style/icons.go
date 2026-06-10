package style

import "github.com/charmbracelet/lipgloss"

// Icon identifies a semantic status glyph.
type Icon int

const (
	IconDone    Icon = iota // ✓ success
	IconCached              // ⊙ cached / exists
	IconRemoved             // ✕ removed / deleted
	IconWarn                // ⚠ warning / skipped
	IconUp                  // ↑ uploading
	IconDown                // ↓ downloading
	IconRunning             // ● running
	IconPaused              // ◌ paused
	IconStopped             // ○ stopped / exited
)

type iconDef struct {
	glyph string
	ascii string
	color lipgloss.Style
}

// iconTable is rebuilt by buildIcons() whenever the theme changes so icon
// colors track the active palette.
var iconTable = map[Icon]iconDef{}

func buildIcons() {
	iconTable = map[Icon]iconDef{
		IconDone:    {"✓", "v", Green},
		IconCached:  {"⊙", "o", Dim},
		IconRemoved: {"✕", "x", Red},
		IconWarn:    {"⚠", "!", Yellow},
		IconUp:      {"↑", "^", Orange},
		IconDown:    {"↓", "v", Cyan},
		IconRunning: {"●", "*", Green},
		IconPaused:  {"◌", "~", Yellow},
		IconStopped: {"○", "-", Dim},
	}
}

// RenderIcon returns the colored status glyph (or an ASCII fallback when emoji
// are disabled via config).
func RenderIcon(i Icon) string {
	d, ok := iconTable[i]
	if !ok {
		return " "
	}
	g := d.glyph
	if disableEmoji {
		g = d.ascii
	}
	return d.color.Render(g)
}

// Glyph returns the raw (uncolored) glyph for an icon, respecting the emoji
// toggle. Useful when callers need to measure or pad around it.
func Glyph(i Icon) string {
	d, ok := iconTable[i]
	if !ok {
		return " "
	}
	if disableEmoji {
		return d.ascii
	}
	return d.glyph
}
