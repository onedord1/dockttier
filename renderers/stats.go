package renderers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

func accentStats() string { return style.HexTeal }

// Stats renders `docker stats` as a live, colorful table that updates in place
// (or once, for --no-stream). Docker is run on a PTY so it streams samples.
type Stats struct{}

type statsBoard struct {
	mu        sync.Mutex
	order     []string
	rows      map[string]statsRecord
	rendered  int
	lastPaint time.Time
	cols      []style.Column
}

func (Stats) Run(ctx Context) int {
	printHeader(ctx, accentStats(), "")
	out("")
	if !hasNoStream(ctx.Args) {
		out(style.Dim.Render("  live — press Ctrl-C to stop"))
		out("")
	}

	b := &statsBoard{
		rows: map[string]statsRecord{},
		cols: []style.Column{
			{Title: "NAME", Width: 16},
			{Title: "CPU%", Width: 7, Right: true},
			{Title: "MEM%", Width: 6, Right: true},
			{Title: "MEM", Width: 16},
			{Title: "MEM USAGE / LIMIT", Width: 20},
			{Title: "NET I/O", Width: 17},
			{Title: "BLOCK I/O", Width: 17},
			{Title: "PIDS", Width: 5, Right: true},
		},
	}
	out(style.HeaderRow(b.cols))
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))

	// Inject --format so docker emits one JSON object per container per sample.
	args := ctx.Args
	if !hasFormatFlag(args) {
		args = append(append([]string{}, args...), "--format", "{{json .}}")
	}

	code := intercept.RunOutputPTY(ctx.Real, args, func(r io.Reader) {
		streamStats(b, r)
	})
	out("")
	return code
}

func streamStats(b *statsBoard, r io.Reader) {
	buf := make([]byte, 8192)
	var acc []byte
	flush := func(line string) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			return
		}
		var st statsRecord
		if json.Unmarshal([]byte(line), &st) != nil || st.ID == "" {
			return
		}
		b.mu.Lock()
		key := shortHash(st.ID)
		if _, ok := b.rows[key]; !ok {
			b.order = append(b.order, key)
		}
		b.rows[key] = st
		b.repaintThrottled()
		b.mu.Unlock()
	}
	for {
		n, err := r.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			for {
				i := indexCRLF(acc)
				if i < 0 {
					break
				}
				flush(stripAnsi(acc[:i]))
				acc = acc[i+1:]
			}
		}
		if err != nil {
			break
		}
	}
	b.mu.Lock()
	b.repaint()
	b.mu.Unlock()
}

func (b *statsBoard) repaintThrottled() {
	if time.Since(b.lastPaint) < 200*time.Millisecond {
		return
	}
	b.repaint()
}

func (b *statsBoard) repaint() {
	if b.rendered > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", b.rendered)
	}
	for _, id := range b.order {
		fmt.Fprint(os.Stdout, "\r\033[2K")
		out(b.renderRow(b.rows[id]))
	}
	b.rendered = len(b.order)
	b.lastPaint = time.Now()
}

func (b *statsBoard) renderRow(st statsRecord) string {
	var cpuPct, memPct float64
	fmt.Sscanf(strings.TrimSuffix(strings.TrimSpace(st.CPUPerc), "%"), "%f", &cpuPct)
	fmt.Sscanf(strings.TrimSuffix(strings.TrimSpace(st.MemPerc), "%"), "%f", &memPct)

	cpuC := cpuColor(cpuPct)
	memC := memPctColor(memPct)

	memBar := style.SlimBar(clampFrac(memPct/100), 14, lipColor(memC))

	styled := []string{
		style.Brand.Render(st.Name),
		lipgloss.NewStyle().Foreground(lipColor(cpuC)).Render(st.CPUPerc),
		lipgloss.NewStyle().Foreground(lipColor(memC)).Render(st.MemPerc),
		memBar,
		style.Muted.Render(st.MemUsage),
		style.Muted.Render(st.NetIO),
		style.Muted.Render(st.BlockIO),
		style.Dim.Render(st.PIDs),
	}
	plain := []string{
		st.Name, st.CPUPerc, st.MemPerc, strings.Repeat("─", 14),
		st.MemUsage, st.NetIO, st.BlockIO, st.PIDs,
	}
	return style.Row(b.cols, styled, plain)
}

func clampFrac(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func indexCRLF(b []byte) int {
	for i, c := range b {
		if c == '\n' || c == '\r' {
			return i
		}
	}
	return -1
}

func hasFormatFlag(args []string) bool {
	for _, a := range args {
		if a == "--format" || strings.HasPrefix(a, "--format=") {
			return true
		}
	}
	return false
}

func hasNoStream(args []string) bool {
	for _, a := range args {
		if a == "--no-stream" {
			return true
		}
	}
	return false
}
