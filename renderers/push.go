package renderers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/style"
)

// Push renders `docker push` with live per-layer progress bars. Docker is run
// on a PTY so it emits real-time byte progress (hidden when piped).
type Push struct{}

func (Push) Run(ctx Context) int {
	printHeader(ctx, accentPush(), "")
	out("")
	out(style.SectionLabel("layers"))
	out("")

	board := newLayerBoard(style.HexGreen, modePush)
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
		out(style.Footer(style.IconRemoved, "Push failed",
			style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexRed)),
		))
		return code
	}
	out(style.Footer(style.IconRunning, "Push complete",
		style.KV("new data", sizeOrDash(board.downloadedBytes()), lipColor(accentPush())),
		style.KV("skipped", sizeOrDash(board.skippedBytes()), lipColor(style.HexTextDim)),
		style.KV("layers", fmt.Sprintf("%d", len(board.order)), lipColor(style.HexBlue)),
		style.KV("elapsed", fmt.Sprintf("%.1fs", time.Since(start).Seconds()), lipColor(style.HexBlue)),
	))
	if board.digest != "" {
		out("  " + style.KV("digest", board.digest, lipColor(style.HexBrand)))
	}
	return code
}
