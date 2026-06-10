package renderers

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/style"
)

// transferMode distinguishes pull (download) from push (upload) styling.
type transferMode int

const (
	modePull transferMode = iota
	modePush
)

// layerState tracks one layer's transfer progress for live rendering.
type layerState struct {
	id      string
	status  string // human label shown on the row
	current int64
	total   int64
	done    bool
	exists  bool
}

// layerBoard renders an ordered set of layer rows, repainting them in place via
// ANSI cursor movement as docker's live progress streams in.
type layerBoard struct {
	mu          sync.Mutex
	order       []string
	layers      map[string]*layerState
	rendered    int
	infoPrinted bool
	lastPaint   time.Time
	doneColor   lipgloss.Color
	mode        transferMode

	from     string
	digest   string
	status   string
	errLines []string
}

func newLayerBoard(doneColor string, mode transferMode) *layerBoard {
	return &layerBoard{
		layers:    map[string]*layerState{},
		doneColor: lipColor(doneColor),
		mode:      mode,
	}
}

func (b *layerBoard) get(id string) *layerState {
	l, ok := b.layers[id]
	if !ok {
		l = &layerState{id: id}
		b.layers[id] = l
		b.order = append(b.order, id)
	}
	return l
}

func (b *layerBoard) downloadedBytes() int64 {
	var n int64
	for _, id := range b.order {
		if l := b.layers[id]; l.done {
			n += l.total
		}
	}
	return n
}

func (b *layerBoard) skippedBytes() int64 {
	var n int64
	for _, id := range b.order {
		if l := b.layers[id]; l.exists {
			n += l.total
		}
	}
	return n
}

// Docker progress lines (TTY or plain) look like:
//
//	b4a248c845e5: Download complete
//	830625e1ac85: Downloading [====>      ]  24.12MB/33.31MB
//	a1b2c3d4e5f6: Already exists
var (
	reLayer  = regexp.MustCompile(`^([0-9a-f]{12}):\s+(.*)$`)
	reBytes  = regexp.MustCompile(`([0-9.]+\s*[kKMGTP]?i?B)\s*/\s*([0-9.]+\s*[kKMGTP]?i?B)`)
	reAnsi   = regexp.MustCompile(`\x1b\[[\?0-9;]*[A-Za-z]`)
	reAnsiOS = regexp.MustCompile(`\x1b\][^\x07]*\x07`)
)

func stripAnsi(b []byte) string {
	b = reAnsiOS.ReplaceAll(b, nil)
	b = reAnsi.ReplaceAll(b, nil)
	return string(b)
}

// feedLine parses one line of docker output, updating board state. It returns
// true when a layer row changed.
func (b *layerBoard) feedLine(t string) bool {
	t = strings.TrimSpace(t)
	if t == "" {
		return false
	}
	m := reLayer.FindStringSubmatch(t)
	if m == nil {
		b.handleTop(t)
		return false
	}
	id, rest := m[1], m[2]
	l := b.get(id)

	statusWord := rest
	if i := strings.Index(rest, "["); i >= 0 {
		statusWord = strings.TrimSpace(rest[:i])
	}
	if bm := reBytes.FindStringSubmatch(rest); bm != nil {
		l.current = parseSize(bm[1])
		l.total = parseSize(bm[2])
	}
	l.status = strings.ToLower(statusWord)

	switch {
	case strings.Contains(statusWord, "Already exists"), strings.HasPrefix(statusWord, "Mounted from"):
		l.exists = true
	case statusWord == "Pull complete", statusWord == "Pushed":
		l.done = true
	case statusWord == "Download complete":
		if l.total > 0 {
			l.current = l.total // download phase finished
		}
	}
	return true
}

func (b *layerBoard) handleTop(t string) {
	lower := strings.ToLower(t)
	switch {
	case strings.Contains(lower, "pulling from"):
		if i := strings.Index(lower, "pulling from"); i >= 0 {
			b.from = "docker.io/" + strings.TrimSpace(t[i+len("pulling from"):])
		}
	case strings.HasPrefix(t, "The push refers to repository"):
		b.from = extractBracket(t)
	case strings.Contains(lower, "digest:"):
		if d := extractDigest(t); d != "" {
			b.digest = d
		}
	case strings.HasPrefix(t, "Status:"):
		b.status = strings.TrimSpace(strings.TrimPrefix(t, "Status:"))
	case containsAny(lower, "error", "denied", "not found", "unauthorized", "no space"):
		b.errLines = append(b.errLines, t)
	}
}

func (b *layerBoard) repaint() {
	if b.rendered == 0 && !b.infoPrinted {
		if b.from != "" {
			out("  " + style.KV("from", b.from, lipColor(style.HexTextMuted)))
			out("")
		}
		b.infoPrinted = true
	}
	if b.rendered > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", b.rendered)
	}
	for _, id := range b.order {
		fmt.Fprint(os.Stdout, "\r\033[2K")
		out(b.renderRow(b.layers[id]))
	}
	b.rendered = len(b.order)
	b.lastPaint = time.Now()
}

// repaintThrottled coalesces rapid updates to avoid flicker.
func (b *layerBoard) repaintThrottled() {
	if time.Since(b.lastPaint) < 60*time.Millisecond {
		return
	}
	b.repaint()
}

func (b *layerBoard) renderRow(l *layerState) string {
	barW := style.ResponsiveBarWidth(46)

	var icon string
	var barColor lipgloss.Color
	var frac float64
	var showPct bool

	activeColor := lipColor(style.HexCyan) // pull = downloading
	if b.mode == modePush {
		activeColor = lipColor(style.HexOrange) // push = uploading
	}

	switch {
	case l.exists:
		icon = style.RenderIcon(style.IconCached)
		barColor = lipColor(style.HexTextDim)
		frac = 1
	case l.done:
		icon = style.RenderIcon(style.IconDone)
		barColor = lipColor(style.HexGreen)
		frac, showPct = 1, true
	case l.total > 0:
		if b.mode == modePush {
			icon = style.RenderIcon(style.IconUp)
		} else {
			icon = style.RenderIcon(style.IconDown)
		}
		barColor = activeColor
		frac = float64(l.current) / float64(l.total)
		showPct = true
	default: // pending / waiting (no bytes yet)
		icon = style.RenderIcon(style.IconStopped)
		barColor = lipColor(style.HexTextDim)
		frac = 0
	}

	bar := style.SlimBar(frac, barW, barColor)

	pct := "    "
	if showPct {
		pct = fmt.Sprintf("%3.0f%%", frac*100)
	}

	size := ""
	if l.total > 0 {
		size = humanSize(l.total)
	}
	return "  " + icon + "  " +
		style.Brand.Render(style.Cell(l.id, 12)) +
		style.PadLeft(style.Muted.Render(size), 9) + "  " +
		bar + " " +
		lipgloss.NewStyle().Foreground(barColor).Render(pct) + "  " +
		style.Dim.Render(l.status)
}

// streamText reads docker's byte stream (TTY or plain), splits on CR/LF,
// strips ANSI control sequences, and drives the board live.
func streamText(b *layerBoard, r io.Reader) {
	buf := make([]byte, 8192)
	var acc []byte
	for {
		n, err := r.Read(buf)
		if n > 0 {
			acc = append(acc, buf[:n]...)
			for {
				i := bytes.IndexAny(acc, "\r\n")
				if i < 0 {
					break
				}
				line := stripAnsi(acc[:i])
				acc = acc[i+1:]
				b.mu.Lock()
				if b.feedLine(line) {
					b.repaintThrottled()
				}
				b.mu.Unlock()
			}
		}
		if err != nil {
			break
		}
	}
	if len(acc) > 0 {
		b.mu.Lock()
		b.feedLine(stripAnsi(acc))
		b.mu.Unlock()
	}
	b.mu.Lock()
	b.repaint() // ensure final 100% state is shown
	b.mu.Unlock()
}

func extractBracket(s string) string {
	if i := strings.Index(s, "["); i >= 0 {
		if j := strings.Index(s[i:], "]"); j >= 0 {
			return strings.TrimSpace(s[i+1 : i+j])
		}
	}
	return ""
}

// finalize marks all non-skipped layers complete (used on a successful
// pull/push so config blobs that docker only reports as "download complete"
// settle to a clean ✓ state) and repaints once more.
func (b *layerBoard) finalize() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, id := range b.order {
		l := b.layers[id]
		if !l.exists {
			l.done = true
			if b.mode == modePush {
				l.status = "pushed"
			} else {
				l.status = "pull complete"
			}
		}
	}
	b.repaint()
}
