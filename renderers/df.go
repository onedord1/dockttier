package renderers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// DiskUsage renders `docker system df` with bold color-coded categories, thick
// proportional bars and a stacked total-usage bar.
type DiskUsage struct{}

type dfRecord struct {
	Type        string `json:"Type"`
	TotalCount  string `json:"TotalCount"`
	Active      string `json:"Active"`
	Size        string `json:"Size"`
	Reclaimable string `json:"Reclaimable"`

	bytes       int64
	reclaimable int64
	count       string
}

func (DiskUsage) Run(ctx Context) int {
	printHeader(ctx, accentDF(), "")

	raw, err := intercept.Capture(ctx.Real, "system", "df", "--format", "{{json .}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: docker system df query failed: %v\n", err)
		return 1
	}

	var recs []dfRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r dfRecord
		if json.Unmarshal([]byte(line), &r) != nil {
			continue
		}
		r.bytes = parseSize(r.Size)
		r.reclaimable = parseReclaimable(r.Reclaimable)
		r.count = r.TotalCount
		recs = append(recs, r)
	}

	var total, totalReclaim int64
	for _, r := range recs {
		total += r.bytes
		totalReclaim += r.reclaimable
	}

	out("")
	out("  " + style.Dim.Render("S T O R A G E   U S A G E"))
	out("")

	for _, r := range recs {
		out(dfRow(r, total))
	}

	out("")
	out("  " + stackedBar(recs, total))
	out("")
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-4)))
	out("  " + style.Muted.Render("total used ") + style.Bold.Render(humanSize(total)) +
		"    " + style.Muted.Render("reclaimable ") + style.Bold.Render(humanSize(totalReclaim)))
	out("  " + style.Dim.Render("run ") + style.Brand.Render("docker system prune") + style.Dim.Render(" to reclaim"))
	return 0
}

func dfRow(r dfRecord, total int64) string {
	barColor := categoryColor(r.Type)

	frac := 0.0
	if total > 0 {
		frac = float64(r.bytes) / float64(total)
	}
	bar := thickBar(frac, 34, barColor)

	name := lipgloss.NewStyle().Foreground(lipColor(barColor)).Bold(true).Render(
		style.PadRight(categoryLabel(r.Type), 12))
	count := style.Dim.Render(style.PadLeft(r.count, 4))
	size := style.Orange.Render(style.PadLeft(humanSize(r.bytes), 10))

	// Reclaimable, right-aligned to the terminal edge.
	var reclaimStyled, reclaimPlain string
	if r.reclaimable > 0 {
		reclaimPlain = "↩ " + humanSize(r.reclaimable)
		reclaimStyled = style.Brand.Render("↩ ") + style.Text.Render(humanSize(r.reclaimable))
	} else {
		reclaimPlain = "—"
		reclaimStyled = style.Dim.Render("—")
	}

	left := "  " + name + "  " + count + "  " + size + "   " + bar
	leftW := lipgloss.Width("  "+style.PadRight(categoryLabel(r.Type), 12)+"  "+
		style.PadLeft(r.count, 4)+"  "+style.PadLeft(humanSize(r.bytes), 10)+"   ") + 34
	gap := style.Width() - leftW - lipgloss.Width(reclaimPlain) - 2
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + reclaimStyled
}

// thickBar renders a solid bar: filled cells in `color`, the remaining track in
// a medium grey (border color), giving the chunky look of the target design.
func thickBar(frac float64, width int, color string) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	if filled == 0 && frac > 0 {
		filled = 1
	}
	full := lipgloss.NewStyle().Foreground(lipColor(color)).Render(strings.Repeat("█", filled))
	track := style.Border.Render(strings.Repeat("█", width-filled))
	return full + track
}

func stackedBar(recs []dfRecord, total int64) string {
	width := style.Width() - 4
	if width < 20 {
		width = 20
	}
	if total <= 0 {
		return style.Border.Render(strings.Repeat("█", width))
	}
	var b strings.Builder
	used := 0
	for _, r := range recs {
		seg := int(float64(r.bytes) / float64(total) * float64(width))
		if seg <= 0 {
			continue
		}
		if used+seg > width {
			seg = width - used
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipColor(categoryColor(r.Type))).
			Render(strings.Repeat("█", seg)))
		used += seg
	}
	if used < width {
		b.WriteString(style.Border.Render(strings.Repeat("█", width-used)))
	}
	return b.String()
}

func categoryColor(t string) string {
	switch {
	case strings.Contains(t, "Image"):
		return style.HexBlue
	case strings.Contains(t, "Container"):
		return style.HexTeal
	case strings.Contains(t, "Volume"):
		return style.HexPurple
	case strings.Contains(t, "Cache"):
		return style.HexYellow
	}
	return style.HexTextMuted
}

func categoryLabel(t string) string {
	switch {
	case strings.Contains(t, "Image"):
		return "Images"
	case strings.Contains(t, "Container"):
		return "Containers"
	case strings.Contains(t, "Volume"):
		return "Volumes"
	case strings.Contains(t, "Cache"):
		return "Build cache"
	}
	return t
}

func parseReclaimable(s string) int64 {
	if i := strings.Index(s, "("); i >= 0 {
		s = s[:i]
	}
	return parseSize(strings.TrimSpace(s))
}
