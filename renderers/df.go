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

// DiskUsage renders `docker system df` with per-category rows and a single
// stacked proportional usage bar.
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
	out(style.SectionLabel("storage usage"))
	out("")

	for _, r := range recs {
		out(dfRow(r, total))
	}

	out("")
	out("  " + style.Dim.Render("total usage (proportional)"))
	out("  " + stackedBar(recs, total))
	out("")
	out(style.Footer(style.IconRunning, "Disk usage",
		style.KV("used", humanSize(total), lipColor(accentDF())),
		style.KV("reclaimable", humanSize(totalReclaim), lipColor(style.HexYellow)),
	))
	out("  " + style.Dim.Render("run ") + style.Brand.Render("docker system prune") + style.Dim.Render(" to reclaim"))
	return 0
}

func dfRow(r dfRecord, total int64) string {
	pct := 0.0
	if total > 0 {
		pct = float64(r.bytes) / float64(total) * 100
	}
	barColor := categoryColor(r.Type)
	bar := style.BarFromRatio(r.bytes, max64(total, 1), 30, lipColor(barColor))

	reclaim := style.Dim.Render("—")
	if r.reclaimable > 0 {
		reclaim = style.Yellow.Render("↩ " + humanSize(r.reclaimable))
	}

	name := style.Cell(categoryLabel(r.Type), 12)
	count := style.Cell(r.count, 5)
	size := style.PadLeft(humanSize(r.bytes), 10)
	return "  " + style.Text.Render(name) +
		style.Dim.Render(count) + "  " +
		lipgloss.NewStyle().Foreground(lipColor(barColor)).Render(size) + "  " +
		bar + style.Muted.Render(fmt.Sprintf("  %4.0f%%  ", pct)) + reclaim
}

func stackedBar(recs []dfRecord, total int64) string {
	const width = 50
	if total <= 0 {
		return style.EmptyBar(width)
	}
	var b strings.Builder
	used := 0
	for _, r := range recs {
		seg := int(float64(r.bytes) / float64(total) * width)
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
		b.WriteString(style.Bar(0, width-used, lipColor(style.HexBarEmpty)))
	}
	return "[" + b.String() + "]"
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

// parseReclaimable handles docker's "84.2MB (11%)" reclaimable format.
func parseReclaimable(s string) int64 {
	if i := strings.Index(s, "("); i >= 0 {
		s = s[:i]
	}
	return parseSize(strings.TrimSpace(s))
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
