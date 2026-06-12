// Package cmd wires configuration, detection, interception and the renderers
// into the dockttier entry point.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dockttier/dockttier/config"
	"github.com/dockttier/dockttier/detect"
	"github.com/dockttier/dockttier/intercept"
	"github.com/dockttier/dockttier/renderers"
	"github.com/dockttier/dockttier/style"
	"github.com/mattn/go-isatty"
)

// version is injected at build time via -ldflags.
var version = "dev"

// Execute runs dockttier and returns the process exit code (identical to the
// real docker exit code in every path).
func Execute() int {
	args := os.Args[1:]

	// dockttier's own version probe (does not shadow any docker subcommand).
	if len(args) == 1 && args[0] == "--dockttier-version" {
		fmt.Println("dockttier", version)
		return 0
	}

	cfg := config.Load()
	style.ApplyPreset(cfg.ThemePreset)
	// DOCKTTIER_THEME env var overrides the configured preset (handy for testing).
	if t := os.Getenv("DOCKTTIER_THEME"); t != "" {
		style.ApplyPreset(t)
	}
	style.SetBrandColor(cfg.BrandColor)
	style.SetDisableEmoji(cfg.DisableEmoji)

	// Offline preview of the active theme (no docker required).
	if len(args) == 1 && args[0] == "--dockttier-demo" {
		style.RefreshWidth()
		renderers.Demo()
		return 0
	}

	real, err := intercept.FindRealDocker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: %v\n", err)
		return 1
	}

	// Fast path: passthrough when output isn't a terminal or is disabled.
	if passthrough() {
		return intercept.RunPassthrough(real, args)
	}

	// Interactive sessions need a raw PTY, no transformation.
	if detect.NeedsRawPTY(args) {
		return intercept.RunRawPTY(real, args)
	}

	style.RefreshWidth()

	ctx := renderers.Context{
		Real:          real,
		Args:          args,
		Subcommand:    detect.Subcommand(args),
		ShowBuildLogs: cfg.ShowBuildLogs,
		StatsTimeout:  time.Duration(cfg.StatsTimeoutMS) * time.Millisecond,
		PreSnapshot:   cfg.PreSnapshot,
	}

	r := pick(detect.Classify(args, cfg.Passthrough))
	return r.Run(ctx)
}

// passthrough reports whether dockttier should disable all formatting.
func passthrough() bool {
	if os.Getenv("DOCKTTIER_DISABLE") == "1" {
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return true
	}
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	return false
}

// pick maps a classified command to its renderer.
func pick(t detect.CommandType) renderers.Renderer {
	switch t {
	case detect.CmdBuild:
		return renderers.Build{}
	case detect.CmdPush:
		return renderers.Push{}
	case detect.CmdPull:
		return renderers.Pull{}
	case detect.CmdRm:
		return renderers.Remove{Kind: "container"}
	case detect.CmdRmi:
		return renderers.Remove{Kind: "image"}
	case detect.CmdNetworkRm:
		return renderers.Remove{Kind: "network"}
	case detect.CmdVolumeRm:
		return renderers.Remove{Kind: "volume"}
	case detect.CmdImages:
		return renderers.Images{}
	case detect.CmdContainerLS:
		return renderers.Containers{}
	case detect.CmdNetworkLS:
		return renderers.Networks{}
	case detect.CmdVolumeLS:
		return renderers.Volumes{}
	case detect.CmdSystemDF:
		return renderers.DiskUsage{}
	case detect.CmdPrune:
		return renderers.Prune{}
	case detect.CmdLogs:
		return renderers.Logs{}
	case detect.CmdExec:
		return renderers.Exec{}
	case detect.CmdTag:
		return renderers.Tag{}
	case detect.CmdStats:
		return renderers.Stats{}
	}
	return renderers.Fallback{}
}
