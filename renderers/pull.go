package renderers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Pull renders `docker pull` with live per-layer progress bars. Docker is run
// on a PTY so it emits its real-time byte progress (which it hides when piped).
type Pull struct{}

func (Pull) Run(ctx Context) int {
	printHeader(ctx, accentPull(), "")
	out("")
	out(style.SectionLabel("layers"))
	out("")

	board := newLayerBoard(style.HexCyan, modePull)
	start := time.Now()

	code := intercept.RunOutputPTY(ctx.Real, ctx.Args, func(r io.Reader) {
		streamText(board, r)
	})
	if code == 0 {
		board.finalize()
	}

	out("")
	out("  " + style.Border.Render(strings.Repeat("─", style.Width()-2)))
	if code != 0 {
		if info, ok := summarizeError(board.errLines); ok {
			renderErrorPanel(info)
		}
		out(style.Footer(style.IconRemoved, "Pull failed",
			style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexRed)),
		))
		return code
	}

	// Look up the final on-disk image size.
	imgSize := imageSize(ctx.Real, lastImageRef(ctx.Args))

	pairs := []string{
		style.KV("downloaded", sizeOrDash(board.downloadedBytes()), lipColor(accentPull())),
		style.KV("layers", fmt.Sprintf("%d", len(board.order)), lipColor(style.HexBlue)),
		style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexBlue)),
	}
	if imgSize > 0 {
		pairs = append(pairs, style.KV("image size", humanSize(imgSize), lipColor(style.HexGreen)))
	}
	out(style.Footer(style.IconRunning, "Pull complete", pairs...))
	if board.digest != "" {
		out("  " + style.KV("digest", board.digest, lipColor(style.HexBrand)))
	}
	if board.status != "" {
		out("  " + style.KV("status", board.status, lipColor(style.HexTextMuted)))
	}
	return code
}

func extractDigest(s string) string {
	for _, f := range strings.Fields(s) {
		if strings.HasPrefix(f, "sha256:") {
			return f
		}
	}
	return ""
}
