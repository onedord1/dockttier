// Package renderers contains the per-command output renderers. Each renderer
// owns the execution of the real docker command it prettifies (streaming its
// output, or running structured `--format json` queries) and returns docker's
// exact exit code.
package renderers

import (
	"fmt"
	"os"
	"time"

	"github.com/dockttier/dockttier/style"
)

// Context carries everything a renderer needs to run.
type Context struct {
	Real          string        // resolved real docker binary path
	Args          []string      // original argv after "docker"
	Subcommand    string        // friendly label for the header line
	ShowBuildLogs bool          // config: show raw build log lines
	StatsTimeout  time.Duration // config: per-container stats timeout
	PreSnapshot   bool          // config: snapshot before rm for name enrichment
}

// Renderer prettifies a single class of docker command.
type Renderer interface {
	Run(ctx Context) int
}

// out writes a line to stdout, ignoring errors (broken pipe is handled upstream).
func out(s string) { fmt.Fprintln(os.Stdout, s) }

// printHeader emits the universal header line for a renderer.
func printHeader(ctx Context, accent string, note string) {
	args := headerArgs(ctx)
	out(style.Header(ctx.Subcommand, args, lipColor(accent), note))
}

// headerArgs returns the invocation arguments with the subcommand tokens
// stripped, for display in the dimmed tail of the header line.
func headerArgs(ctx Context) []string {
	// Count subcommand words so we can drop them from the display args.
	words := 1
	for i := 1; i < len(splitFields(ctx.Subcommand)); i++ {
		words++
	}
	var args []string
	skipped := 0
	for _, a := range ctx.Args {
		if skipped < words && !startsWithDash(a) {
			skipped++
			continue
		}
		args = append(args, a)
	}
	return args
}

func startsWithDash(s string) bool { return len(s) > 0 && s[0] == '-' }

func splitFields(s string) []string {
	var f []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				f = append(f, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		f = append(f, cur)
	}
	return f
}
