// Command dockttier is a transparent prettifier shim for the Docker CLI. It
// runs the real docker binary, parses its output, and re-renders it with rich
// terminal styling while preserving exit codes, signals and stdin behavior.
package main

import (
	"os"

	"github.com/dockttier/dockttier/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
