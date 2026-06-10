package intercept

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// RunOutputPTY runs docker attached to a pseudo-terminal so it emits its rich,
// live progress output (docker suppresses incremental progress when it detects
// a pipe). The PTY master is handed to consume for parsing; we render our own
// UI to the real stdout. Falls back to piped streaming if a PTY can't be
// allocated. The user terminal is NOT put in raw mode (this is not interactive).
func RunOutputPTY(real string, args []string, consume func(io.Reader)) int {
	cmd := exec.Command(real, args...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		// Fall back to plain piped streaming (no live byte progress).
		return RunStreamed(real, args, consume, nil)
	}
	defer func() { _ = ptmx.Close() }()

	// A wide window so docker renders full-width progress without truncating.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 60, Cols: 220})

	forward := ForwardSignals(cmd)
	defer forward()

	consume(ptmx)
	return exitCode(cmd.Wait())
}

// RunRawPTY runs docker attached to a pseudo-terminal in raw mode, forwarding
// the terminal size on SIGWINCH and restoring the original terminal state on
// exit. Used for interactive sessions (docker exec -it, run -it, attach).
func RunRawPTY(real string, args []string) int {
	cmd := exec.Command(real, args...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: could not start interactive session: %v\n", err)
		return 1
	}
	defer func() { _ = ptmx.Close() }()

	// Forward window-size changes.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	go func() {
		for range winch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	winch <- syscall.SIGWINCH // set the initial size
	defer signal.Stop(winch)

	// Put the user terminal into raw mode for the duration of the session.
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
		}
	}

	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	return exitCode(cmd.Wait())
}
