package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Divider returns a full-width horizontal rule in the border color.
func Divider() string {
	return Border.Render(strings.Repeat("─", Width()))
}

// SectionLabel returns an inline section heading: "── label ──────…" spanning
// the full terminal width, label and rule both in textDim.
func SectionLabel(label string) string {
	label = strings.ToLower(label)
	prefix := "── " + label + " "
	pad := Width() - lipgloss.Width(prefix)
	if pad < 0 {
		pad = 0
	}
	return Dim.Render(prefix + strings.Repeat("─", pad))
}

// Truncate shortens s to fit width columns, appending an ellipsis when cut.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(s)
	out := make([]rune, 0, width)
	w := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > width-1 {
			break
		}
		out = append(out, r)
		w += rw
	}
	return string(out) + "…"
}

// PadRight pads s with spaces to width columns (no truncation).
func PadRight(s string, width int) string {
	gap := width - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// PadLeft right-aligns s within width columns.
func PadLeft(s string, width int) string {
	gap := width - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return strings.Repeat(" ", gap) + s
}

// Cell truncates then left-pads a value to exactly width columns.
func Cell(s string, width int) string {
	return PadRight(Truncate(s, width), width)
}
