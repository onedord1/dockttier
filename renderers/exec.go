package renderers

import (
	"github.com/dockttier/dockttier/detect"
	"github.com/dockttier/dockttier/intercept"
)

// Exec handles docker exec. Interactive sessions are delegated to the raw PTY
// passthrough with no header/footer and no transformation; non-interactive
// exec gets the header line then unchanged passthrough.
type Exec struct{}

func (Exec) Run(ctx Context) int {
	if detect.NeedsRawPTY(ctx.Args) {
		return intercept.RunRawPTY(ctx.Real, ctx.Args)
	}
	printHeader(ctx, accentFallback(), "")
	return intercept.RunStreamed(ctx.Real, ctx.Args, nil, nil)
}
