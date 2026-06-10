package renderers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Build renders `docker build` from the BuildKit --progress=rawjson stream as a
// live, in-place step list with an animated spinner, ticking elapsed time and a
// progress bar on the currently-running stage.
type Build struct{}

// Minimal BuildKit SolveStatus types (we deserialize only what we render to
// avoid depending on the very large moby/buildkit module).
type bkVertex struct {
	Digest    string     `json:"digest"`
	Name      string     `json:"name"`
	Started   *time.Time `json:"started"`
	Completed *time.Time `json:"completed"`
	Cached    bool       `json:"cached"`
	Error     string     `json:"error"`
}

type bkStatus struct {
	ID      string `json:"id"`
	Vertex  string `json:"vertex"`
	Current int64  `json:"current"`
	Total   int64  `json:"total"`
}

type bkSolveStatus struct {
	Vertexes []bkVertex `json:"vertexes"`
	Statuses []bkStatus `json:"statuses"`
}

type buildStep struct {
	digest    string
	name      string
	started   *time.Time
	completed *time.Time
	cached    bool
	failed    bool
	statuses  map[string]bkStatus // keyed by status id, for byte progress
}

func (s *buildStep) progress() (cur, tot int64) {
	for _, st := range s.statuses {
		cur += st.Current
		tot += st.Total
	}
	return
}

func (s *buildStep) running() bool { return s.started != nil && s.completed == nil && !s.failed }

type buildBoard struct {
	mu       sync.Mutex
	order    []string
	steps    map[string]*buildStep
	rendered int
	tick     int
	errLines []string
}

func (Build) Run(ctx Context) int {
	args := ctx.Args
	if !hasProgressFlag(args) {
		args = append(append([]string{}, args...), "--progress=rawjson")
	}

	printHeader(ctx, accentBuild(), "[BUILDKIT]")
	out("")
	out(style.SectionLabel("build stages"))
	out("")

	b := &buildBoard{steps: map[string]*buildStep{}}
	start := time.Now()

	// Animation ticker: repaint periodically so the spinner, pulse bar and
	// live elapsed time keep moving even when BuildKit is quiet.
	stopTick := make(chan struct{})
	var tickWG sync.WaitGroup
	tickWG.Add(1)
	go func() {
		defer tickWG.Done()
		t := time.NewTicker(120 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stopTick:
				return
			case <-t.C:
				b.mu.Lock()
				if b.anyRunning() {
					b.tick++
					b.repaint()
				}
				b.mu.Unlock()
			}
		}
	}()

	consume := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			var st bkSolveStatus
			if err := json.Unmarshal(sc.Bytes(), &st); err != nil {
				if t := strings.TrimSpace(sc.Text()); t != "" {
					b.mu.Lock()
					b.errLines = append(b.errLines, t)
					b.mu.Unlock()
				}
				continue
			}
			b.mu.Lock()
			b.apply(st)
			b.repaint()
			b.mu.Unlock()
		}
	}

	code := intercept.RunStreamed(ctx.Real, args, nil, consume)

	close(stopTick)
	tickWG.Wait()
	b.mu.Lock()
	b.repaint() // final state
	b.mu.Unlock()

	out("")
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	if code == 0 {
		tag := buildTag(ctx.Args)
		sz, id := imageInfo(ctx.Real, tag)
		pairs := []string{
			style.KV("stages", fmt.Sprintf("%d", len(b.order)), lipColor(accentBuild())),
			style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexBlue)),
		}
		if tag != "" {
			pairs = append([]string{style.KV("image", tag, lipColor(style.HexBrand))}, pairs...)
		}
		if sz > 0 {
			pairs = append(pairs, style.KV("size", humanSize(sz), lipColor(style.HexGreen)))
		}
		out(style.Footer(style.IconRunning, "Build successful", pairs...))
		if id != "" {
			out("  " + style.KV("image id", shortHash(id), lipColor(style.HexBrand)))
		}
	} else {
		if info, ok := summarizeError(b.errLines); ok {
			renderErrorPanel(info)
		}
		out(style.Footer(style.IconRemoved, "Build failed",
			style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexRed)),
		))
	}
	return code
}

func (b *buildBoard) anyRunning() bool {
	for _, d := range b.order {
		if b.steps[d].running() {
			return true
		}
	}
	return false
}

func (b *buildBoard) apply(st bkSolveStatus) {
	for _, v := range st.Vertexes {
		if v.Error != "" {
			b.errLines = append(b.errLines, v.Error)
		}
		if v.Name == "" || strings.HasPrefix(v.Name, "[internal]") {
			continue
		}
		s, ok := b.steps[v.Digest]
		if !ok {
			s = &buildStep{digest: v.Digest, name: cleanStageName(v.Name), statuses: map[string]bkStatus{}}
			b.steps[v.Digest] = s
			b.order = append(b.order, v.Digest)
		}
		s.started = v.Started
		s.completed = v.Completed
		s.cached = v.Cached
		s.failed = v.Error != ""
	}
	for _, stt := range st.Statuses {
		if s, ok := b.steps[stt.Vertex]; ok {
			s.statuses[stt.ID] = stt
		}
	}
}

func (b *buildBoard) repaint() {
	if b.rendered > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", b.rendered)
	}
	now := time.Now()
	order := b.displayOrder()
	for i, d := range order {
		fmt.Fprint(os.Stdout, "\r\033[2K")
		out(b.renderStep(i+1, b.steps[d], now))
	}
	b.rendered = len(order)
}

// displayOrder returns step digests sorted chronologically by start time, with
// not-yet-started steps kept in insertion order at the end. This keeps the
// running stage visually after the completed ones regardless of the order
// BuildKit reports vertices.
func (b *buildBoard) displayOrder() []string {
	idx := make(map[string]int, len(b.order))
	for i, d := range b.order {
		idx[d] = i
	}
	order := append([]string(nil), b.order...)
	sort.SliceStable(order, func(i, j int) bool {
		si, sj := b.steps[order[i]], b.steps[order[j]]
		switch {
		case si.started == nil && sj.started == nil:
			return idx[order[i]] < idx[order[j]]
		case si.started == nil:
			return false
		case sj.started == nil:
			return true
		default:
			return si.started.Before(*sj.started)
		}
	})
	return order
}

func (b *buildBoard) renderStep(n int, s *buildStep, now time.Time) string {
	num := style.Dim.Render(fmt.Sprintf("%2d", n))

	const nameW = 40
	var icon, nameStyled, trailing string

	switch {
	case s.failed:
		icon = style.RenderIcon(style.IconRemoved)
		nameStyled = style.Red.Render(style.Truncate(s.name, nameW))
		trailing = style.Red.Render("failed")
	case s.cached:
		icon = style.RenderIcon(style.IconCached)
		nameStyled = style.Dim.Render(style.Truncate(s.name, nameW))
		trailing = style.Dim.Render("cached")
	case s.completed != nil:
		icon = style.RenderIcon(style.IconDone)
		nameStyled = style.Text.Render(style.Truncate(s.name, nameW))
		if s.started != nil {
			secs := s.completed.Sub(*s.started).Seconds()
			if secs > 1 {
				trailing = style.Yellow.Render(fmt.Sprintf("%.1fs", secs))
			} else {
				trailing = style.Dim.Render(fmt.Sprintf("%.1fs", secs))
			}
		}
	case s.running():
		icon = style.SpinnerFrame(b.tick)
		nameStyled = style.Text.Render(style.Truncate(s.name, nameW))
		trailing = b.runningTrailing(s, now)
	default: // pending (queued, not started)
		icon = style.RenderIcon(style.IconStopped)
		nameStyled = style.Dim.Render(style.Truncate(s.name, nameW))
	}

	return " " + num + "  " + icon + "  " + padName(nameStyled, style.Truncate(s.name, nameW), nameW) + trailing
}

// runningTrailing builds the live progress segment for a running step: a real
// percentage bar when byte progress is known, otherwise an indeterminate pulse,
// followed by the ticking elapsed time.
func (b *buildBoard) runningTrailing(s *buildStep, now time.Time) string {
	barW := 18
	cur, tot := s.progress()

	var bar string
	if tot > 0 {
		frac := float64(cur) / float64(tot)
		bar = style.SlimBar(frac, barW, lipColor(style.HexCyan)) +
			" " + style.Cyan.Render(fmt.Sprintf("%3.0f%%", frac*100))
	} else {
		bar = style.PulseBar(barW, b.tick, lipColor(style.HexBlue)) + "     "
	}

	elapsed := ""
	if s.started != nil {
		elapsed = "  " + style.Yellow.Render(fmt.Sprintf("%.1fs", now.Sub(*s.started).Seconds()))
	}
	return bar + elapsed
}

// padName pads a styled name string to width using its plain length.
func padName(styled, plain string, width int) string {
	gap := width - lipgloss.Width(plain)
	if gap < 1 {
		gap = 1
	}
	return styled + strings.Repeat(" ", gap)
}

func cleanStageName(name string) string {
	// BuildKit names look like "[2/8] RUN npm ci" or "RUN npm ci".
	if i := strings.Index(name, "] "); i >= 0 && strings.HasPrefix(name, "[") {
		name = name[i+2:]
	}
	return name
}

func hasProgressFlag(args []string) bool {
	for _, a := range args {
		if a == "--progress" || strings.HasPrefix(a, "--progress=") {
			return true
		}
	}
	return false
}
