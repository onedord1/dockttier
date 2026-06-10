package renderers

import "github.com/dockttier/dockttier/intercept"

// Fallback emits the header line then passes docker's stdout/stderr through
// completely unchanged. Used for every command dockttier does not specialize,
// guaranteeing it never breaks an unknown docker subcommand.
type Fallback struct{}

func (Fallback) Run(ctx Context) int {
	printHeader(ctx, accentFallback(), "")
	// nil consumers => streams wired directly to the parent, unmodified.
	return intercept.RunStreamed(ctx.Real, ctx.Args, nil, nil)
}
