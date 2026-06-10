package renderers

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Prune renders the various `docker … prune` commands: a scope warning, a
// deletion list with type badges, and a reclaimed-space summary.
type Prune struct{}

func (Prune) Run(ctx Context) int {
	printHeader(ctx, accentPrune(), "")

	out("")
	out("  " + style.RenderIcon(style.IconWarn) + "  " + style.Muted.Render(
		"This removes stopped containers, dangling images, unused"))
	out("     " + style.Muted.Render("networks, build cache, and (with --volumes) volumes. Running containers are kept."))
	out("")
	out(style.SectionLabel("deleted resources"))
	out("")

	start := time.Now()
	count := 0
	var reclaimed int64

	consume := func(reader io.Reader) {
		sc := bufio.NewScanner(reader)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if r, ok := parseReclaimedTotal(line); ok {
				reclaimed = r
				continue
			}
			if row, kind, ok := pruneRow(line); ok {
				count++
				_ = kind
				out(row)
			}
		}
	}

	code := intercept.RunStreamed(ctx.Real, ctx.Args, consume, consume)

	out("")
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	status := "Prune complete"
	out(style.Footer(style.IconRunning, status,
		style.KV("removed", fmt.Sprintf("%d resources", count), lipColor(style.HexRed)),
		style.KV("reclaimed", humanSize(reclaimed), lipColor(style.HexGreen)),
		style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(accentPrune())),
	))
	return code
}

// pruneRow classifies a docker prune output line into a styled row. Docker
// prints section headers ("Deleted Containers:") then bare IDs; we infer the
// current resource type from the most recent header.
var currentKind string

func pruneRow(line string) (string, string, bool) {
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "deleted containers"), strings.HasPrefix(lower, "deleted container"):
		currentKind = "container"
		return "", "", false
	case strings.HasPrefix(lower, "deleted images"), strings.HasPrefix(lower, "untagged"):
		currentKind = "image"
		if strings.HasPrefix(lower, "untagged") {
			break
		}
		return "", "", false
	case strings.HasPrefix(lower, "deleted volumes"), strings.HasPrefix(lower, "deleted volume"):
		currentKind = "volume"
		return "", "", false
	case strings.HasPrefix(lower, "deleted networks"), strings.HasPrefix(lower, "deleted network"):
		currentKind = "network"
		return "", "", false
	case strings.HasPrefix(lower, "deleted build cache"), strings.HasPrefix(lower, "id:"):
		currentKind = "cache"
		return "", "", false
	case strings.HasPrefix(lower, "total reclaimed"):
		return "", "", false
	}

	kind := currentKind
	if kind == "" {
		kind = "image"
	}
	id := shortHash(strings.TrimPrefix(line, "Untagged: "))
	badge := style.BadgeType("["+kind+"]", lipColor(pruneKindColor(kind)))
	row := "  " + style.RenderIcon(style.IconRemoved) + "  " + badge + "  " +
		style.Dim.Render(style.Truncate(id, style.Width()-20))
	return row, kind, true
}

func pruneKindColor(kind string) string {
	switch kind {
	case "container":
		return style.HexTeal
	case "image":
		return style.HexBlue
	case "volume":
		return style.HexPurple
	case "cache":
		return style.HexYellow
	case "network":
		return style.HexGreen
	}
	return style.HexTextMuted
}

func parseReclaimedTotal(line string) (int64, bool) {
	const p = "Total reclaimed space:"
	if !strings.HasPrefix(line, p) {
		return 0, false
	}
	return parseSize(strings.TrimSpace(strings.TrimPrefix(line, p))), true
}
