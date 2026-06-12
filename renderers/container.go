package renderers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Containers renders `docker ps` / `docker container ls` as a styled table with
// live per-container CPU and memory (usage / limit) pulled from docker stats.
type Containers struct{}

type containerRecord struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`

	cpuPct  float64
	memPct  float64
	memUsed int64
	memText string // "38.07MiB / 512MiB"
	statsOK bool
}

type statsRecord struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
	MemPerc  string `json:"MemPerc"`
	NetIO    string `json:"NetIO"`
	BlockIO  string `json:"BlockIO"`
	PIDs     string `json:"PIDs"`
}

func (Containers) Run(ctx Context) int {
	printHeader(ctx, accentContainer(), "")

	args := append([]string{}, ctx.Args...)
	args = append(args, "--format", "{{json .}}")
	raw, err := intercept.Capture(ctx.Real, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: docker container query failed: %v\n", err)
		return 1
	}

	var rows []containerRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r containerRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		rows = append(rows, r)
	}

	attachStats(ctx.Real, rows, ctx.StatsTimeout)

	cols := []style.Column{
		{Title: "ID", Width: 12},
		{Title: "NAME", Width: 16},
		{Title: "IMAGE", Width: 18},
		{Title: "STATUS", Width: 9},
		{Title: "UPTIME", Width: 12},
		{Title: "CPU%", Width: 7, Right: true},
		{Title: "MEM%", Width: 7, Right: true},
		{Title: "MEM USAGE / LIMIT", Width: 20},
	}

	out("")
	out(style.HeaderRow(cols))
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))

	running, paused := 0, 0
	var totalMem int64
	for _, r := range rows {
		switch r.State {
		case "running":
			running++
		case "paused":
			paused++
		}
		totalMem += r.memUsed
		out(containerRow(cols, r))
	}

	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	out(style.Footer(style.IconRunning, "Containers",
		style.KV("running", fmt.Sprintf("%d", running), lipColor(style.HexGreen)),
		style.KV("paused", fmt.Sprintf("%d", paused), lipColor(style.HexYellow)),
		style.KV("total mem", humanSize(totalMem), lipColor(accentContainer())),
	))
	return 0
}

func containerRow(cols []style.Column, r containerRecord) string {
	var statusStyled, statusPlain string
	switch r.State {
	case "running":
		statusStyled = style.RenderIcon(style.IconRunning) + " " + style.Green.Render("run")
		statusPlain = "* run"
	case "paused":
		statusStyled = style.RenderIcon(style.IconPaused) + " " + style.Yellow.Render("paused")
		statusPlain = "~ paused"
	default:
		statusStyled = style.RenderIcon(style.IconStopped) + " " + style.Dim.Render("exited")
		statusPlain = "- exited"
	}

	cpuStyled, cpuPlain := style.Dim.Render("-"), "-"
	memPctStyled, memPctPlain := style.Dim.Render("-"), "-"
	memStyled, memPlain := style.Dim.Render("-"), "-"
	if r.statsOK {
		cpuColor := cpuColor(r.cpuPct)
		cpuPlain = fmt.Sprintf("%.2f%%", r.cpuPct)
		cpuStyled = lipgloss.NewStyle().Foreground(lipColor(cpuColor)).Render(cpuPlain)

		memColor := memPctColor(r.memPct)
		memPctPlain = fmt.Sprintf("%.1f%%", r.memPct)
		memPctStyled = lipgloss.NewStyle().Foreground(lipColor(memColor)).Render(memPctPlain)

		memPlain = r.memText
		memStyled = lipgloss.NewStyle().Foreground(lipColor(memColor)).Render(memPlain)
	}

	styled := []string{
		style.Brand.Render(shortHash(r.ID)),
		style.Text.Render(r.Names),
		style.Muted.Render(r.Image),
		statusStyled,
		style.Muted.Render(r.Status),
		cpuStyled,
		memPctStyled,
		memStyled,
	}
	plain := []string{
		shortHash(r.ID), r.Names, r.Image, statusPlain, r.Status, cpuPlain, memPctPlain, memPlain,
	}
	return style.Row(cols, styled, plain)
}

func cpuColor(pct float64) string {
	switch {
	case pct > 3.0:
		return style.HexOrange
	case pct > 0.5:
		return style.HexYellow
	}
	return style.HexGreen
}

func memPctColor(pct float64) string {
	switch {
	case pct > 80:
		return style.HexRed
	case pct > 50:
		return style.HexYellow
	}
	return style.HexGreen
}

// attachStats fetches CPU/memory for all running containers in a single
// `docker stats --no-stream` call and merges the results by id and name.
func attachStats(real string, rows []containerRecord, timeout time.Duration) {
	hasRunning := false
	for _, r := range rows {
		if r.State == "running" {
			hasRunning = true
			break
		}
	}
	if !hasRunning {
		return
	}

	// docker stats --no-stream samples for ~1.3s, so use a generous timeout.
	to := 6 * time.Second
	if timeout > to {
		to = timeout
	}

	byID := bulkStats(real, to)
	for i := range rows {
		key := shortHash(rows[i].ID)
		st, ok := byID[key]
		if !ok {
			st, ok = byID[rows[i].Names]
		}
		if !ok {
			continue
		}
		rows[i].statsOK = true
		rows[i].memText = st.MemUsage
		fmt.Sscanf(strings.TrimSuffix(strings.TrimSpace(st.CPUPerc), "%"), "%f", &rows[i].cpuPct)
		fmt.Sscanf(strings.TrimSuffix(strings.TrimSpace(st.MemPerc), "%"), "%f", &rows[i].memPct)
		if parts := strings.SplitN(st.MemUsage, "/", 2); len(parts) > 0 {
			rows[i].memUsed = parseSize(strings.TrimSpace(parts[0]))
		}
	}
}

func bulkStats(real string, timeout time.Duration) map[string]statsRecord {
	result := map[string]statsRecord{}
	type out struct{ data []byte }
	ch := make(chan out, 1)
	go func() {
		b, err := intercept.Capture(real, "stats", "--no-stream", "--format", "{{json .}}")
		if err != nil {
			ch <- out{}
			return
		}
		ch <- out{b}
	}()

	var data []byte
	select {
	case r := <-ch:
		data = r.data
	case <-time.After(timeout):
		return result
	}

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var st statsRecord
		if json.Unmarshal([]byte(line), &st) != nil {
			continue
		}
		if st.ID != "" {
			result[shortHash(st.ID)] = st
		}
		if st.Name != "" {
			result[st.Name] = st
		}
	}
	return result
}
