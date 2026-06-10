package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Badge renders an inline pill like `[ CACHED ]`: the uppercased text in the
// accent color, surrounded by brackets in the dim textDim shade.
func Badge(text string, accent lipgloss.Color) string {
	bracket := Dim.Render
	inner := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(strings.ToUpper(text))
	return bracket("[ ") + inner + bracket(" ]")
}

// BadgeType renders a compact type badge like `[container]` used by the prune
// renderer (lowercase label, no inner padding).
func BadgeType(text string, accent lipgloss.Color) string {
	bracket := Dim.Render
	inner := lipgloss.NewStyle().Foreground(accent).Render(text)
	return bracket("[") + inner + bracket("]")
}
