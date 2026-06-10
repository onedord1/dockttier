package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Header renders the universal command header line:
//
//	dockttier › docker <subcommand>  <args…>
//
// `dockttier` is brand+bold, the separator is textDim, `docker <subcommand>` is
// the command accent color, and the remaining args are textMuted. A trailing
// note (e.g. "[BUILDKIT]") is rendered dim and right-adjacent.
func Header(subcommand string, args []string, accent lipgloss.Color, note string) string {
	var b strings.Builder
	b.WriteString(Brand.Bold(true).Render("dockttier"))
	b.WriteString(Dim.Render(" › "))
	b.WriteString(lipgloss.NewStyle().Foreground(accent).Render("docker " + subcommand))
	if len(args) > 0 {
		b.WriteString(Muted.Render("  " + strings.Join(args, " ")))
	}
	if note != "" {
		b.WriteString(Dim.Render("   " + note))
	}
	return b.String()
}

// KV renders a "label value" pair with the label in textMuted and the value in
// the supplied accent color.
func KV(label, value string, accent lipgloss.Color) string {
	return Muted.Render(label+" ") + lipgloss.NewStyle().Foreground(accent).Render(value)
}

// Footer renders the summary footer: a leading status icon + bold status word,
// followed by accent-colored key/value pairs separated by wide gaps.
func Footer(icon Icon, status string, pairs ...string) string {
	var b strings.Builder
	b.WriteString(RenderIcon(icon))
	b.WriteString(" ")
	b.WriteString(Bold.Render(status))
	for _, p := range pairs {
		b.WriteString("    ")
		b.WriteString(p)
	}
	return b.String()
}
