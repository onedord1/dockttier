package renderers

import (
	"os/exec"
	"strings"

	"github.com/dockttier/dockttier/style"
)

// Tag renders `docker tag SOURCE TARGET`. Docker prints nothing on success, so
// dockttier emits a styled confirmation; failures get the error panel.
type Tag struct{}

func (Tag) Run(ctx Context) int {
	printHeader(ctx, accentTag(), "")

	source, target := tagOperands(ctx.Args)

	cmd := exec.Command(ctx.Real, ctx.Args...)
	stderr, _ := cmd.StderrPipe()
	_ = cmd.Start()
	var errBuf strings.Builder
	if stderr != nil {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				errBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}
	err := cmd.Wait()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}

	out("")
	if code == 0 {
		out("  " + style.RenderIcon(style.IconDone) + "  " +
			style.Muted.Render("tagged  ") +
			style.Text.Render(source) + "  " +
			style.Dim.Render("→") + "  " +
			style.Brand.Render(target))
	} else {
		lines := strings.Split(strings.TrimSpace(errBuf.String()), "\n")
		if info, ok := summarizeError(lines); ok {
			renderErrorPanel(info)
		}
	}
	return code
}

// tagOperands returns the source and target image references (the two non-flag
// arguments following the subcommand).
func tagOperands(args []string) (string, string) {
	var ops []string
	skip := false
	for _, a := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		ops = append(ops, a)
	}
	// ops includes the subcommand token(s) like "tag" or "image tag".
	// Drop leading subcommand tokens, keep the last two as source/target.
	if len(ops) >= 2 {
		return ops[len(ops)-2], ops[len(ops)-1]
	}
	if len(ops) == 1 {
		return ops[0], ""
	}
	return "", ""
}
