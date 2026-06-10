package renderers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Remove renders resource removal commands (rm, rmi, network rm, volume rm).
// Kind is one of: "container", "image", "network", "volume".
type Remove struct{ Kind string }

type psSnapshot struct {
	ID    string `json:"ID"`
	Names string `json:"Names"`
	Image string `json:"Image"`
	State string `json:"State"`
}

func (r Remove) Run(ctx Context) int {
	accent := removeAccent(r.Kind)
	printHeader(ctx, accent, "")

	// Pre-run snapshot enriches container IDs with names/images.
	snap := map[string]psSnapshot{}
	if ctx.PreSnapshot && r.Kind == "container" {
		if raw, err := intercept.Capture(ctx.Real, "ps", "-a", "--format", "{{json .}}"); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var s psSnapshot
				if json.Unmarshal([]byte(line), &s) == nil {
					snap[shortHash(s.ID)] = s
					if len(s.ID) >= 12 {
						snap[s.ID[:12]] = s
					}
					snap[s.Names] = s
				}
			}
		}
	}

	out("")
	out(style.SectionLabel(r.Kind + "s"))
	out("")

	removed, skipped := 0, 0
	var errLines []string

	collect := func(reader io.Reader) {
		sc := bufio.NewScanner(reader)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if isErrorLine(line) {
				skipped++
				errLines = append(errLines, line)
				out("  " + style.RenderIcon(style.IconWarn) + "  " +
					style.Muted.Render(style.Truncate(stripANSI(line), style.Width()-22)) + "  " +
					style.Badge("SKIPPED", lipColor(style.HexYellow)))
				continue
			}
			removed++
			out(r.removalRow(line, snap))
		}
	}

	code := intercept.RunStreamed(ctx.Real, ctx.Args, collect, collect)

	out("")
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	if removed == 0 && skipped > 0 {
		if info, ok := summarizeError(errLines); ok {
			renderErrorPanel(info)
			out("")
		}
	}
	out(style.Footer(style.IconRunning, "Done",
		style.KV("removed", fmt.Sprintf("%d", removed), lipColor(style.HexGreen)),
		style.KV("skipped", fmt.Sprintf("%d", skipped), lipColor(style.HexYellow)),
	))
	return code
}

func (r Remove) removalRow(line string, snap map[string]psSnapshot) string {
	icon := style.RenderIcon(style.IconRemoved)
	badge := style.Badge("REMOVED", lipColor(style.HexRed))

	switch r.Kind {
	case "image":
		// rmi prints "Untagged: repo:tag" or "Deleted: sha256:…".
		label := line
		if strings.HasPrefix(line, "Untagged: ") {
			label = strings.TrimPrefix(line, "Untagged: ")
		} else if strings.HasPrefix(line, "Deleted: ") {
			label = shortHash(strings.TrimPrefix(line, "Deleted: "))
		}
		return "  " + icon + "  " +
			style.Text.Render(style.Truncate(label, style.Width()-22)) + "  " + badge

	case "container":
		short := shortHash(line)
		name, image := "", ""
		if s, ok := snap[short]; ok {
			name, image = s.Names, s.Image
		} else if s, ok := snap[line]; ok {
			name, image = s.Names, s.Image
		}
		return "  " + icon + "  " +
			style.Brand.Render(style.Cell(short, 14)) +
			style.Text.Render(style.Cell(name, 20)) +
			style.Muted.Render(style.Cell(image, 22)) + badge

	default: // network, volume
		return "  " + icon + "  " +
			style.Brand.Render(style.Truncate(line, style.Width()-22)) + "  " + badge
	}
}

func removeAccent(kind string) string {
	switch kind {
	case "network":
		return style.HexGreen
	case "volume":
		return style.HexPurple
	case "image":
		return style.HexBlue
	}
	return style.HexRed // container
}

func isErrorLine(line string) bool {
	l := strings.ToLower(stripANSI(line))
	return strings.HasPrefix(l, "error") || strings.Contains(l, "cannot remove") ||
		strings.Contains(l, "conflict") || strings.Contains(l, "is using") ||
		strings.Contains(l, "in use") || strings.Contains(l, "no such")
}
