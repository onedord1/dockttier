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

// Networks renders `docker network ls` as a styled table.
type Networks struct{}

func accentNetwork() string { return style.HexBlue }

type networkRecord struct {
	ID     string `json:"ID"`
	Name   string `json:"Name"`
	Driver string `json:"Driver"`
	Scope  string `json:"Scope"`
}

func (Networks) Run(ctx Context) int {
	printHeader(ctx, accentNetwork(), "")

	args := append([]string{}, ctx.Args...)
	args = append(args, "--format", "{{json .}}")
	raw, err := intercept.Capture(ctx.Real, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: docker network query failed: %v\n", err)
		return 1
	}

	var rows []networkRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r networkRecord
		if json.Unmarshal([]byte(line), &r) == nil {
			rows = append(rows, r)
		}
	}

	cols := []style.Column{
		{Title: "NETWORK ID", Width: 14},
		{Title: "NAME", Width: 22},
		{Title: "DRIVER", Width: 12},
		{Title: "SCOPE", Width: 10},
	}
	out("")
	out(style.HeaderRow(cols))
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))

	drivers := map[string]int{}
	for _, r := range rows {
		drivers[r.Driver]++
		dc := networkDriverColor(r.Driver)
		styled := []string{
			style.Brand.Render(shortHash(r.ID)),
			style.Blue.Render(r.Name),
			lipgloss.NewStyle().Foreground(lipColor(dc)).Render(r.Driver),
			style.Muted.Render(r.Scope),
		}
		plain := []string{shortHash(r.ID), r.Name, r.Driver, r.Scope}
		out(style.Row(cols, styled, plain))
	}

	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	out(style.Footer(style.IconRunning, "Networks",
		style.KV("total", fmt.Sprintf("%d", len(rows)), lipColor(accentNetwork())),
		style.KV("drivers", driverSummary(drivers), lipColor(style.HexTeal)),
	))
	return 0
}

func networkDriverColor(d string) string {
	switch d {
	case "bridge":
		return style.HexTeal
	case "host":
		return style.HexPurple
	case "overlay":
		return style.HexBlue
	case "null", "none":
		return style.HexTextDim
	case "macvlan", "ipvlan":
		return style.HexOrange
	}
	return style.HexTextMuted
}

func driverSummary(m map[string]int) string {
	var parts []string
	for d, n := range m {
		parts = append(parts, fmt.Sprintf("%s %d", d, n))
	}
	return strings.Join(parts, " · ")
}
