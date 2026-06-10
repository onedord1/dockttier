package renderers

import (
	"regexp"
	"strings"

	"github.com/dockttier/dockttier/style"
)

// errorInfo is a distilled view of a docker failure: a short title, the key
// message, and an optional actionable hint.
type errorInfo struct {
	title  string
	detail string
	hint   string
}

var (
	reExitCode = regexp.MustCompile(`(?i)exit code:?\s*(\d+)`)
	reProcess  = regexp.MustCompile(`(?i)process "([^"]+)"`)
)

// noisePrefixes are stripped from candidate error lines.
var noisePrefixes = []string{
	"ERROR: ", "error: ", "Error: ",
	"Error response from daemon: ",
	"failed to solve: ",
}

// summarizeError scans collected error/diagnostic lines and returns a concise
// errorInfo. ok is false when nothing error-like was found.
func summarizeError(lines []string) (errorInfo, bool) {
	// Keep only non-empty, trimmed lines.
	var cand []string
	for _, l := range lines {
		l = strings.TrimSpace(stripANSI(l))
		if l != "" {
			cand = append(cand, l)
		}
	}
	if len(cand) == 0 {
		return errorInfo{}, false
	}

	joined := strings.ToLower(strings.Join(cand, "\n"))

	// Pick the most informative line: prefer one containing a known signal,
	// else the last non-empty line.
	detail := bestLine(cand)
	for _, p := range noisePrefixes {
		detail = strings.TrimPrefix(detail, p)
		// also strip mid-string occurrences of "failed to solve:"
	}
	if i := strings.Index(detail, "failed to solve: "); i >= 0 {
		detail = detail[i+len("failed to solve: "):]
	}
	detail = strings.TrimSpace(detail)

	info := errorInfo{title: "Failed", detail: detail}

	switch {
	case containsAny(joined, "denied: requested access", "push access denied",
		"unauthorized", "authentication required", "no basic auth credentials"):
		info.title = "Access denied"
		info.hint = "run `docker login` for this registry, then retry"

	case containsAny(joined, "manifest unknown", "manifest for", "not found: manifest",
		"pull access denied", "repository does not exist"):
		info.title = "Image not found"
		info.hint = "check the image name and tag are correct"

	case containsAny(joined, "cannot connect to the docker daemon", "is the docker daemon running"):
		info.title = "Docker daemon unreachable"
		info.hint = "start the docker daemon (e.g. `sudo systemctl start docker`)"

	case containsAny(joined, "no space left on device"):
		info.title = "Out of disk space"
		info.hint = "free space with `docker system prune`"

	case containsAny(joined, "no such host", "dial tcp", "connection refused",
		"i/o timeout", "timeout exceeded", "network is unreachable"):
		info.title = "Network error"
		info.hint = "check connectivity and the registry address"

	case containsAny(joined, "conflict", "is using its referenced image",
		"container is running", "in use by container"):
		info.title = "Resource in use"
		info.hint = "stop or force-remove the resource (`-f`) first"

	case containsAny(joined, "failed to solve", "did not complete successfully", "exit code"):
		info.title = "Build failed"
		if m := reProcess.FindStringSubmatch(strings.Join(cand, "\n")); m != nil {
			info.detail = "step `" + truncateRunes(m[1], 60) + "` failed"
		}
		if m := reExitCode.FindStringSubmatch(joined); m != nil {
			info.detail += " (exit code " + m[1] + ")"
		}
		info.hint = "inspect the failing step's output above"
	}

	info.detail = strings.TrimSpace(info.detail)
	if info.detail == "" {
		info.detail = cand[len(cand)-1]
	}
	return info, true
}

// bestLine chooses the most diagnostic candidate line.
func bestLine(cand []string) string {
	keywords := []string{"denied", "unauthorized", "manifest", "failed to solve",
		"did not complete", "exit code", "no space", "cannot connect", "no such host",
		"conflict", "error response from daemon"}
	for i := len(cand) - 1; i >= 0; i-- {
		lc := strings.ToLower(cand[i])
		for _, k := range keywords {
			if strings.Contains(lc, k) {
				return cand[i]
			}
		}
	}
	return cand[len(cand)-1]
}

// renderErrorPanel prints the distilled error block.
func renderErrorPanel(info errorInfo) {
	out("")
	out(style.SectionLabel("error"))
	out("")
	out("  " + style.RenderIcon(style.IconRemoved) + "  " + style.Red.Bold(true).Render(info.title))
	wrapW := style.Width() - 6
	for _, line := range wrapText(info.detail, wrapW) {
		out("     " + style.Muted.Render(line))
	}
	if info.hint != "" {
		out("     " + style.Dim.Render("↳ hint: ") + style.Brand.Render(info.hint))
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// wrapText wraps s to width columns on word boundaries.
func wrapText(s string, width int) []string {
	if width < 10 {
		width = 10
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	lines = append(lines, cur)
	return lines
}
