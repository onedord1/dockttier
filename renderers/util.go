package renderers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dockttier/dockttier/intercept"
)

// parseSize converts a docker-style size string ("87.6MB", "1.2kB", "0B",
// "379.2 MB") into a byte count. Returns 0 on parse failure.
func parseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0B" || s == "N/A" {
		return 0
	}
	// Split numeric prefix from unit suffix.
	i := 0
	for i < len(s) && (s[i] == '.' || s[i] == '-' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	numStr := strings.TrimSpace(s[:i])
	unit := strings.ToLower(strings.TrimSpace(s[i:]))
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	var mult float64 = 1
	switch {
	case strings.HasPrefix(unit, "t"):
		mult = 1000 * 1000 * 1000 * 1000
	case strings.HasPrefix(unit, "g"):
		mult = 1000 * 1000 * 1000
	case strings.HasPrefix(unit, "m"):
		mult = 1000 * 1000
	case strings.HasPrefix(unit, "k"):
		mult = 1000
	case strings.HasPrefix(unit, "b"), unit == "":
		mult = 1
	}
	return int64(num * mult)
}

// humanSize formats a byte count as a compact decimal size (e.g. "87.6 MB").
func humanSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"kB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), units[exp])
}

// shortHash trims a "sha256:" prefix and shortens a digest to 12 hex chars.
func shortHash(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// sizeOrDash formats a byte count, or "—" when zero/unknown.
func sizeOrDash(b int64) string {
	if b <= 0 {
		return "—"
	}
	return humanSize(b)
}

// lastImageRef returns the final positional argument (docker pull/push put the
// image reference last), or "" if the last token looks like a flag.
func lastImageRef(args []string) string {
	if len(args) == 0 {
		return ""
	}
	last := args[len(args)-1]
	if strings.HasPrefix(last, "-") {
		return ""
	}
	return last
}

// imageSize returns the on-disk size of an image ref in bytes (0 on failure).
func imageSize(real, ref string) int64 {
	if ref == "" {
		return 0
	}
	out, err := intercept.Capture(real, "image", "inspect", "--format", "{{.Size}}", ref)
	if err != nil {
		return 0
	}
	var sz int64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &sz)
	return sz
}

// imageInfo returns the on-disk size (bytes) and full id of an image ref.
func imageInfo(real, ref string) (int64, string) {
	if ref == "" {
		return 0, ""
	}
	out, err := intercept.Capture(real, "image", "inspect", "--format", "{{.Size}}|{{.Id}}", ref)
	if err != nil {
		return 0, ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	var sz int64
	fmt.Sscanf(parts[0], "%d", &sz)
	id := ""
	if len(parts) > 1 {
		id = parts[1]
	}
	return sz, id
}

// buildTag extracts the first image tag from -t/--tag flags of a build command.
func buildTag(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-t" || a == "--tag":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "--tag="):
			return strings.TrimPrefix(a, "--tag=")
		case strings.HasPrefix(a, "-t="):
			return strings.TrimPrefix(a, "-t=")
		}
	}
	return ""
}
