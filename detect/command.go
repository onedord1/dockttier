// Package detect classifies a docker invocation (the argv after "docker") into
// the renderer that should handle it, and recognizes interactive commands that
// require a raw PTY.
package detect

import "strings"

// CommandType identifies which renderer handles an invocation.
type CommandType int

const (
	CmdBuild CommandType = iota
	CmdPush
	CmdPull
	CmdRm
	CmdRmi
	CmdImages
	CmdContainerLS
	CmdNetworkLS
	CmdVolumeLS
	CmdSystemDF
	CmdPrune
	CmdNetworkRm
	CmdVolumeRm
	CmdLogs
	CmdExec
	CmdTag
	CmdStats
	CmdFallback
)

// globalFlagsWithValue are the docker global flags that consume the next token
// as their value, so the classifier can skip past them to the subcommand.
var globalFlagsWithValue = map[string]bool{
	"--config":    true,
	"-c":          true,
	"--context":   true,
	"-H":          true,
	"--host":      true,
	"-l":          true,
	"--log-level": true,
	"--tlscacert": true,
	"--tlscert":   true,
	"--tlskey":    true,
}

// locateSubcommand returns the ordered sequence of non-flag tokens that begins
// the subcommand, skipping any leading global flags and their values.
func locateSubcommand(args []string) []string {
	var seq []string
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			// Flag of the form --flag=value never consumes the next token.
			if strings.Contains(a, "=") {
				i++
				continue
			}
			if globalFlagsWithValue[a] {
				i += 2
				continue
			}
			i++
			continue
		}
		seq = append(seq, a)
		i++
	}
	return seq
}

// matchPrefix reports whether seq begins with the given tokens in order.
func matchPrefix(seq, tokens []string) bool {
	if len(seq) < len(tokens) {
		return false
	}
	for i, t := range tokens {
		if seq[i] != t {
			return false
		}
	}
	return true
}

// Classify selects the renderer for the given argv. `passthrough` is the
// user-configured list of subcommands to always pass through unchanged.
func Classify(args []string, passthrough []string) CommandType {
	seq := locateSubcommand(args)
	if len(seq) == 0 {
		return CmdFallback
	}

	for _, p := range passthrough {
		if seq[0] == p {
			return CmdFallback
		}
	}

	switch {
	case matchPrefix(seq, []string{"buildx", "build"}),
		matchPrefix(seq, []string{"image", "build"}),
		matchPrefix(seq, []string{"build"}):
		return CmdBuild
	case matchPrefix(seq, []string{"image", "push"}),
		matchPrefix(seq, []string{"push"}):
		return CmdPush
	case matchPrefix(seq, []string{"image", "pull"}),
		matchPrefix(seq, []string{"pull"}):
		return CmdPull
	case matchPrefix(seq, []string{"container", "rm"}),
		matchPrefix(seq, []string{"container", "remove"}),
		matchPrefix(seq, []string{"rm"}):
		return CmdRm
	case matchPrefix(seq, []string{"image", "rm"}),
		matchPrefix(seq, []string{"image", "remove"}),
		matchPrefix(seq, []string{"rmi"}):
		return CmdRmi
	case matchPrefix(seq, []string{"image", "ls"}),
		matchPrefix(seq, []string{"image", "list"}),
		matchPrefix(seq, []string{"images"}):
		return CmdImages
	case matchPrefix(seq, []string{"container", "ls"}),
		matchPrefix(seq, []string{"container", "list"}),
		matchPrefix(seq, []string{"container", "ps"}),
		matchPrefix(seq, []string{"ps"}):
		return CmdContainerLS
	case matchPrefix(seq, []string{"network", "ls"}),
		matchPrefix(seq, []string{"network", "list"}):
		return CmdNetworkLS
	case matchPrefix(seq, []string{"volume", "ls"}),
		matchPrefix(seq, []string{"volume", "list"}):
		return CmdVolumeLS
	case matchPrefix(seq, []string{"system", "df"}):
		return CmdSystemDF
	case matchPrefix(seq, []string{"system", "prune"}),
		matchPrefix(seq, []string{"container", "prune"}),
		matchPrefix(seq, []string{"image", "prune"}),
		matchPrefix(seq, []string{"volume", "prune"}),
		matchPrefix(seq, []string{"network", "prune"}):
		return CmdPrune
	case matchPrefix(seq, []string{"network", "rm"}),
		matchPrefix(seq, []string{"network", "remove"}):
		return CmdNetworkRm
	case matchPrefix(seq, []string{"volume", "rm"}),
		matchPrefix(seq, []string{"volume", "remove"}):
		return CmdVolumeRm
	case matchPrefix(seq, []string{"container", "logs"}),
		matchPrefix(seq, []string{"logs"}):
		return CmdLogs
	case matchPrefix(seq, []string{"exec"}):
		return CmdExec
	case matchPrefix(seq, []string{"image", "tag"}),
		matchPrefix(seq, []string{"tag"}):
		return CmdTag
	case matchPrefix(seq, []string{"container", "stats"}),
		matchPrefix(seq, []string{"stats"}):
		return CmdStats
	}
	return CmdFallback
}

// Subcommand returns a human-friendly subcommand label for the header line
// (e.g. "system df", "image ls", "build").
func Subcommand(args []string) string {
	seq := locateSubcommand(args)
	if len(seq) == 0 {
		return ""
	}
	// Two-word docker subcommands worth showing verbatim.
	if len(seq) >= 2 {
		two := seq[0] + " " + seq[1]
		switch two {
		case "system df", "system prune", "image ls", "image list", "image build",
			"image push", "image pull", "image rm", "image remove", "image prune",
			"container ls", "container list", "container ps", "container rm",
			"container remove", "container prune", "container logs", "volume prune",
			"network prune", "buildx build", "network ls", "network list",
			"network rm", "network remove", "volume ls", "volume list",
			"volume rm", "volume remove", "image tag", "container stats":
			return two
		}
	}
	return seq[0]
}

// NeedsRawPTY reports whether the invocation is an interactive exec/run/attach
// session that must be wired through a raw pseudo-terminal.
func NeedsRawPTY(args []string) bool {
	seq := locateSubcommand(args)
	if len(seq) == 0 {
		return false
	}
	sub := seq[0]
	if sub != "exec" && sub != "run" && sub != "attach" {
		return false
	}
	for _, a := range args {
		switch a {
		case "-it", "-ti", "-i", "-t", "--interactive", "--tty":
			return true
		}
		// Combined short flags like -itd.
		if len(a) > 1 && a[0] == '-' && a[1] != '-' {
			if strings.ContainsAny(a[1:], "it") {
				return true
			}
		}
	}
	return false
}
