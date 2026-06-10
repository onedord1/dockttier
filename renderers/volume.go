package renderers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Volumes renders `docker volume ls` as a styled table.
type Volumes struct{}

func accentVolume() string { return style.HexPurple }

type volumeRecord struct {
	Driver     string `json:"Driver"`
	Name       string `json:"Name"`
	Mountpoint string `json:"Mountpoint"`
	Links      string `json:"Links"`
	Size       string `json:"Size"`
}

func (Volumes) Run(ctx Context) int {
	printHeader(ctx, accentVolume(), "")

	args := append([]string{}, ctx.Args...)
	args = append(args, "--format", "{{json .}}")
	raw, err := intercept.Capture(ctx.Real, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: docker volume query failed: %v\n", err)
		return 1
	}

	var rows []volumeRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r volumeRecord
		if json.Unmarshal([]byte(line), &r) == nil {
			rows = append(rows, r)
		}
	}

	cols := []style.Column{
		{Title: "VOLUME NAME", Width: 40},
		{Title: "DRIVER", Width: 10},
		{Title: "LINKS", Width: 8, Right: true},
	}
	out("")
	out(style.HeaderRow(cols))
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))

	for _, r := range rows {
		links := r.Links
		if links == "" || links == "N/A" {
			links = "—"
		}
		styled := []string{
			style.Brand.Render(r.Name),
			style.Muted.Render(r.Driver),
			style.Dim.Render(links),
		}
		plain := []string{r.Name, r.Driver, links}
		out(style.Row(cols, styled, plain))
	}

	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	out(style.Footer(style.IconRunning, "Volumes",
		style.KV("total", fmt.Sprintf("%d", len(rows)), lipColor(accentVolume())),
	))
	return 0
}
