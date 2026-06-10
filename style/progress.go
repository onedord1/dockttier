package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DefaultBarWidth is the fixed bar width used by push/pull layer rows.
const DefaultBarWidth = 24

const (
	minBar = 8
	maxBar = 40
)

// Bar renders a progress bar of the given total width with `filled` filled
// cells in the supplied color and the remainder as themed empty cells. `filled`
// is clamped to the range [0, width].
func Bar(filled, width int, color lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	full := strings.Repeat(barFilledGlyph, filled)
	empty := strings.Repeat(barEmptyGlyph, width-filled)
	return lipgloss.NewStyle().Foreground(color).Render(full) + emptyStyle.Render(empty)
}

// BarFromRatio renders a bar from a current/total ratio. A non-positive total
// yields an empty bar; current > total renders fully filled.
func BarFromRatio(current, total int64, width int, color lipgloss.Color) string {
	if total <= 0 {
		return Bar(0, width, color)
	}
	if current > total {
		return Bar(width, width, color)
	}
	filled := int(float64(current) / float64(total) * float64(width))
	return Bar(filled, width, color)
}

// FullBar renders a completely filled bar in the supplied color.
func FullBar(width int, color lipgloss.Color) string { return Bar(width, width, color) }

// EmptyBar renders a completely empty bar.
func EmptyBar(width int) string { return Bar(0, width, lipgloss.Color(HexBarEmpty)) }

// ResponsiveBarWidth fits a bar into the available terminal width minus the
// space reserved for labels, constrained to [8, 40] characters.
func ResponsiveBarWidth(reserved int) int {
	avail := Width() - reserved
	if avail < minBar {
		return minBar
	}
	if avail > maxBar {
		return maxBar
	}
	return avail
}

// Slim glyphs for thin, pip-style progress bars.
const (
	slimFilled = "━" // heavy horizontal line (filled)
	slimHead   = "╸" // partial/lead cap
	slimEmpty  = "─" // light horizontal line (track)
)

var slimTrackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(HexTextDim))

// SlimBar renders a thin pip-style progress bar at the given fraction (0..1).
// Filled cells use the supplied color; the remaining track is dim. A partial
// cell is shown with a lead cap for finer granularity.
func SlimBar(frac float64, width int, color lipgloss.Color) string {
	if width < 1 {
		width = 1
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	exact := frac * float64(width)
	filled := int(exact)
	colStyle := lipgloss.NewStyle().Foreground(color)

	var b strings.Builder
	b.WriteString(colStyle.Render(strings.Repeat(slimFilled, filled)))
	rem := width - filled
	if rem > 0 {
		// A partial lead cap when we're mid-cell and not yet full.
		if exact-float64(filled) >= 0.25 && filled < width {
			b.WriteString(colStyle.Render(slimHead))
			rem--
		}
		if rem > 0 {
			b.WriteString(slimTrackStyle.Render(strings.Repeat(slimEmpty, rem)))
		}
	}
	return b.String()
}

// PulseBar renders an indeterminate thin bar with a lit segment that bounces
// left-right based on tick, for work with no measurable percentage (e.g. a
// long-running RUN step). The track is dim; the moving window is colored.
func PulseBar(width, tick int, color lipgloss.Color) string {
	if width < 4 {
		width = 4
	}
	seg := width / 4
	if seg < 2 {
		seg = 2
	}
	span := width - seg
	if span < 1 {
		span = 1
	}
	pos := tick % (2 * span)
	if pos > span {
		pos = 2*span - pos
	}
	colStyle := lipgloss.NewStyle().Foreground(color)
	var b strings.Builder
	b.WriteString(slimTrackStyle.Render(strings.Repeat(slimEmpty, pos)))
	b.WriteString(colStyle.Render(strings.Repeat(slimFilled, seg)))
	tail := width - pos - seg
	if tail > 0 {
		b.WriteString(slimTrackStyle.Render(strings.Repeat(slimEmpty, tail)))
	}
	return b.String()
}
