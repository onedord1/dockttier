package intercept

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// ForwardSignals relays SIGINT, SIGTERM and SIGWINCH to the child process until
// the returned stop function is called. Forwarding failures (e.g. the child has
// already exited) are ignored so the proxy never crashes on a late signal.
func ForwardSignals(cmd *exec.Cmd) (stop func()) {
	ch := make(chan os.Signal, 8)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case sig := <-ch:
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			case <-done:
				return
			}
		}
	}()

	return func() {
		signal.Stop(ch)
		close(done)
	}
}
