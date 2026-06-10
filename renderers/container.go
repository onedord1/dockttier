package renderers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Containers renders `docker ps` / `docker container ls` as a styled table with
// live per-container CPU and memory usage.
type Containers struct{}

type containerRecord struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`

	cpu     string // formatted, "" => unknown
	mem     string
	cpuPct  float64
	memByte int64
	statsOK bool
}

type statsRecord struct {
	ID       string `json:"ID"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
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

	enrichStats(ctx.Real, rows, ctx.StatsTimeout)

	cols := []style.Column{
		{Title: "ID", Width: 12},
		{Title: "NAME", Width: 14},
		{Title: "IMAGE", Width: 18},
		{Title: "STATUS", Width: 10},
		{Title: "UPTIME", Width: 12},
		{Title: "PORTS", Width: 22},
		{Title: "CPU%", Width: 6, Right: true},
		{Title: "MEM", Width: 9, Right: true},
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
		totalMem += r.memByte
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
	// Status dot + word.
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

	uptime := r.Status

	cpuStyled, cpuPlain := "-", "-"
	memStyled, memPlain := "-", "-"
	if r.statsOK {
		cpuColor := style.HexGreen
		switch {
		case r.cpuPct > 3.0:
			cpuColor = style.HexOrange
		case r.cpuPct > 0.5:
			cpuColor = style.HexYellow
		}
		cpuPlain = r.cpu
		cpuStyled = lipgloss.NewStyle().Foreground(lipColor(cpuColor)).Render(r.cpu)

		memColor := style.HexGreen
		switch {
		case r.memByte > 70*1000*1000:
			memColor = style.HexRed
		case r.memByte > 40*1000*1000:
			memColor = style.HexYellow
		}
		memPlain = humanSize(r.memByte)
		memStyled = lipgloss.NewStyle().Foreground(lipColor(memColor)).Render(memPlain)
	}

	ports := r.Ports
	if ports == "" {
		ports = "—"
	}

	styled := []string{
		style.Brand.Render(shortHash(r.ID)),
		style.Text.Render(r.Names),
		style.Muted.Render(r.Image),
		statusStyled,
		style.Muted.Render(uptime),
		style.Dim.Render(ports),
		cpuStyled,
		memStyled,
	}
	plain := []string{
		shortHash(r.ID), r.Names, r.Image, statusPlain, uptime, ports, cpuPlain, memPlain,
	}
	return style.Row(cols, styled, plain)
}

// enrichStats fills CPU/mem for each container via concurrent, timeout-bounded
// `docker stats --no-stream` queries.
func enrichStats(real string, rows []containerRecord, timeout time.Duration) {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	for i := range rows {
		if rows[i].State != "running" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			rec, ok := queryStats(real, rows[idx].ID, timeout)
			if !ok {
				return
			}
			rows[idx].statsOK = true
			rows[idx].cpu = rec.CPUPerc
			fmt.Sscanf(strings.TrimSuffix(rec.CPUPerc, "%"), "%f", &rows[idx].cpuPct)
			// MemUsage looks like "62.4MiB / 1.94GiB"; take the first field.
			if parts := strings.SplitN(rec.MemUsage, "/", 2); len(parts) > 0 {
				rows[idx].memByte = parseSize(strings.TrimSpace(parts[0]))
			}
		}(i)
	}
	wg.Wait()
}

func queryStats(real, id string, timeout time.Duration) (statsRecord, bool) {
	type result struct {
		rec statsRecord
		ok  bool
	}
	ch := make(chan result, 1)
	go func() {
		out, err := intercept.Capture(real, "stats", "--no-stream", "--format", "{{json .}}", id)
		if err != nil {
			ch <- result{}
			return
		}
		var rec statsRecord
		if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &rec); err != nil {
			ch <- result{}
			return
		}
		ch <- result{rec, true}
	}()
	select {
	case r := <-ch:
		return r.rec, r.ok
	case <-time.After(timeout):
		return statsRecord{}, false
	}
}
