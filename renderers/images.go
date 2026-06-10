package renderers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Images renders `docker images` as a styled table with color-coded sizes,
// inline size bars and per-image layer counts.
type Images struct{}

type imageRecord struct {
	Repository   string `json:"Repository"`
	Tag          string `json:"Tag"`
	ID           string `json:"ID"`
	CreatedAt    string `json:"CreatedAt"`
	CreatedSince string `json:"CreatedSince"`
	Size         string `json:"Size"`

	bytes  int64
	layers int
}

func (Images) Run(ctx Context) int {
	printHeader(ctx, accentImages(), "")

	raw, err := intercept.Capture(ctx.Real, "images", "--format", "{{json .}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: docker images query failed: %v\n", err)
		return 1
	}

	var imgs []imageRecord
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r imageRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue // skip unparseable record, keep going
		}
		r.bytes = parseSize(r.Size)
		imgs = append(imgs, r)
	}

	// Layer counts via concurrent docker inspect.
	enrichLayers(ctx.Real, imgs)

	var maxBytes int64 = 1
	for _, r := range imgs {
		if r.bytes > maxBytes {
			maxBytes = r.bytes
		}
	}

	cols := []style.Column{
		{Title: "REPOSITORY", Width: 15},
		{Title: "TAG", Width: 11},
		{Title: "IMAGE ID", Width: 13},
		{Title: "CREATED", Width: 13},
		{Title: "SIZE", Width: 10, Right: true},
		{Title: "LAYERS", Width: 12},
	}

	out("")
	out(style.HeaderRow(cols))
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))

	var totalBytes int64
	dangling := 0
	for _, r := range imgs {
		totalBytes += r.bytes
		if r.Repository == "<none>" && r.Tag == "<none>" {
			dangling++
		}
		out(imageRow(cols, r, maxBytes))
	}

	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	out(style.Footer(style.IconRunning, "Images",
		style.KV("total", fmt.Sprintf("%d", len(imgs)), lipColor(accentImages())),
		style.KV("dangling", fmt.Sprintf("%d", dangling), lipColor(style.HexYellow)),
		style.KV("size", humanSize(totalBytes), lipColor(style.HexBlue)),
	))
	if dangling > 0 {
		out("  " + style.Dim.Render("tip: ") + style.Brand.Render("docker image prune") +
			style.Dim.Render(fmt.Sprintf("  to remove %d dangling image(s)", dangling)))
	}
	return 0
}

func imageRow(cols []style.Column, r imageRecord, maxBytes int64) string {
	none := func(s string) bool { return s == "<none>" }

	repoStyled := style.Blue.Render(r.Repository)
	if none(r.Repository) {
		repoStyled = style.Dim.Render(r.Repository)
	}
	tagStyled := style.Brand.Render(r.Tag)
	if none(r.Tag) {
		tagStyled = style.Dim.Render(r.Tag)
	}

	sizeColor := style.HexGreen
	switch {
	case r.bytes > 300*1000*1000:
		sizeColor = style.HexRed
	case r.bytes >= 100*1000*1000:
		sizeColor = style.HexYellow
	}
	sizeStyled := lipgloss.NewStyle().Foreground(lipColor(sizeColor)).Render(humanSize(r.bytes))

	created := r.CreatedSince
	if created == "" {
		created = r.CreatedAt
	}

	filled := 0
	if maxBytes > 0 {
		filled = int(float64(r.bytes)/float64(maxBytes)*6 + 0.5)
	}
	bar := style.Bar(filled, 6, lipColor(sizeColor))
	layers := bar + "  " + style.Dim.Render(fmt.Sprintf("%d", r.layers))

	styled := []string{
		repoStyled,
		tagStyled,
		style.Dim.Render(shortHash(r.ID)),
		style.Muted.Render(created),
		sizeStyled,
		layers,
	}
	plain := []string{
		r.Repository, r.Tag, shortHash(r.ID), created, humanSize(r.bytes),
		strings.Repeat("█", 6) + "  " + fmt.Sprintf("%d", r.layers),
	}
	return style.Row(cols, styled, plain)
}

// enrichLayers fills the layer count of each image via concurrent inspects.
func enrichLayers(real string, imgs []imageRecord) {
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range imgs {
		if imgs[i].ID == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			out, err := intercept.Capture(real, "inspect", "--format", "{{len .RootFS.Layers}}", imgs[idx].ID)
			if err != nil {
				return
			}
			var n int
			if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); err == nil {
				imgs[idx].layers = n
			}
		}(i)
	}
	wg.Wait()
}
