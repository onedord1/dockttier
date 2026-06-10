// Package intercept locates the real docker binary and runs it as a child
// process, wiring streams for either transparent passthrough, raw PTY sessions,
// or renderer-driven streaming. It guarantees exit-code and signal transparency.
package intercept

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

var (
	realOnce sync.Once
	realPath string
	realErr  error
)

// FindRealDocker resolves the real docker binary by walking $PATH and skipping
// dockttier's own executable. The result is cached for the process lifetime.
func FindRealDocker() (string, error) {
	realOnce.Do(func() { realPath, realErr = findRealDocker() })
	return realPath, realErr
}

func findRealDocker() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot resolve own executable path: %w", err)
	}
	selfResolved, err := filepath.EvalSymlinks(self)
	if err != nil {
		selfResolved = self
	}

	path := os.Getenv("PATH")
	if path == "" {
		return "", errors.New("PATH is empty; cannot locate real docker binary")
	}

	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "docker")
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}
		if resolved != selfResolved {
			return candidate, nil
		}
	}
	return "", errors.New("real docker binary not found in PATH")
}

// exitCode extracts the process exit code from a (possibly nil) Run/Wait error,
// translating signal termination to the POSIX 128+signal convention.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if status, ok := ee.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return 128 + int(status.Signal())
			}
			return status.ExitStatus()
		}
		return ee.ExitCode()
	}
	return 1
}

// RunPassthrough executes docker with stdio wired directly to the parent — no
// buffering, no transformation. Used for pipe mode, NO_COLOR, DOCKTTIER_DISABLE
// and unknown commands.
func RunPassthrough(real string, args []string) int {
	cmd := exec.Command(real, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	forward := ForwardSignals(cmd)
	defer forward()
	return exitCode(cmd.Run())
}

// StreamFunc consumes one of the child's output streams.
type StreamFunc func(io.Reader)

// RunStreamed pipes the child's stdout and stderr through the supplied
// consumers, forwarding signals, and returns the child's exit code. stdin is
// wired straight through. Either consumer may be nil to send that stream to the
// corresponding parent stream unchanged.
func RunStreamed(real string, args []string, onOut, onErr StreamFunc) int {
	cmd := exec.Command(real, args...)
	cmd.Stdin = os.Stdin

	var outR, errR io.ReadCloser
	var err error
	if onOut != nil {
		if outR, err = cmd.StdoutPipe(); err != nil {
			fmt.Fprintf(os.Stderr, "dockttier: %v\n", err)
			return 1
		}
	} else {
		cmd.Stdout = os.Stdout
	}
	if onErr != nil {
		if errR, err = cmd.StderrPipe(); err != nil {
			fmt.Fprintf(os.Stderr, "dockttier: %v\n", err)
			return 1
		}
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: failed to launch docker: %v\n", err)
		return 1
	}
	forward := ForwardSignals(cmd)
	defer forward()

	var wg sync.WaitGroup
	if onOut != nil {
		wg.Add(1)
		go func() { defer wg.Done(); onOut(outR) }()
	}
	if onErr != nil {
		wg.Add(1)
		go func() { defer wg.Done(); onErr(errR) }()
	}
	wg.Wait()
	return exitCode(cmd.Wait())
}

// Capture runs docker and returns combined stdout (only) for commands queried
// internally by table renderers (e.g. `docker images --format json`). stderr is
// discarded; the error reflects launch/exit status.
func Capture(real string, args ...string) ([]byte, error) {
	cmd := exec.Command(real, args...)
	out, err := cmd.Output()
	return out, err
}
