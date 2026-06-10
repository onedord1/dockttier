// Package config loads the optional ~/.config/dockttier/config.toml file and
// applies safe defaults when it is missing or invalid.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/BurntSushi/toml"
)

// Config is the fully-resolved runtime configuration.
type Config struct {
	ThemePreset    string
	BrandColor     string
	DisableEmoji   bool
	ShowBuildLogs  bool
	StatsTimeoutMS int
	PreSnapshot    bool
	Passthrough    []string
}

// file mirrors the on-disk TOML structure.
type file struct {
	Theme struct {
		Preset       string `toml:"preset"`
		BrandColor   string `toml:"brand_color"`
		DisableEmoji bool   `toml:"disable_emoji"`
	} `toml:"theme"`
	Behavior struct {
		ShowBuildLogs  bool `toml:"show_build_logs"`
		StatsTimeoutMS int  `toml:"stats_timeout_ms"`
		PreSnapshot    bool `toml:"pre_snapshot"`
	} `toml:"behavior"`
	Passthrough struct {
		Commands []string `toml:"commands"`
	} `toml:"passthrough"`
}

const (
	defaultStatsTimeout = 500
	minStatsTimeout     = 50
	maxStatsTimeout     = 60000
)

var hexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		StatsTimeoutMS: defaultStatsTimeout,
		PreSnapshot:    true,
	}
}

// Path returns the config file location.
func Path() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "dockttier", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dockttier", "config.toml")
}

// Load reads and validates the config file, returning defaults (and a warning
// on stderr) for any missing or invalid values.
func Load() Config {
	cfg := Default()
	path := Path()

	data, err := os.ReadFile(path)
	if err != nil {
		// Missing file is the normal case; only warn on real read errors.
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "dockttier: could not read config %s: %v (using defaults)\n", path, err)
		}
		return cfg
	}

	var f file
	if _, err := toml.Decode(string(data), &f); err != nil {
		fmt.Fprintf(os.Stderr, "dockttier: could not parse config %s: %v (using defaults)\n", path, err)
		return cfg
	}

	if f.Theme.BrandColor != "" {
		if hexRe.MatchString(f.Theme.BrandColor) {
			cfg.BrandColor = f.Theme.BrandColor
		} else {
			fmt.Fprintf(os.Stderr, "dockttier: invalid brand_color %q in config (using default)\n", f.Theme.BrandColor)
		}
	}
	cfg.DisableEmoji = f.Theme.DisableEmoji
	if f.Theme.Preset != "" {
		if knownPreset(f.Theme.Preset) {
			cfg.ThemePreset = f.Theme.Preset
		} else {
			fmt.Fprintf(os.Stderr, "dockttier: unknown theme preset %q (using default)\n", f.Theme.Preset)
		}
	}
	cfg.ShowBuildLogs = f.Behavior.ShowBuildLogs
	cfg.PreSnapshot = f.Behavior.PreSnapshot

	if f.Behavior.StatsTimeoutMS != 0 {
		if f.Behavior.StatsTimeoutMS >= minStatsTimeout && f.Behavior.StatsTimeoutMS <= maxStatsTimeout {
			cfg.StatsTimeoutMS = f.Behavior.StatsTimeoutMS
		} else {
			fmt.Fprintf(os.Stderr, "dockttier: stats_timeout_ms %d out of range [%d,%d] (using %d)\n",
				f.Behavior.StatsTimeoutMS, minStatsTimeout, maxStatsTimeout, defaultStatsTimeout)
		}
	}

	cfg.Passthrough = f.Passthrough.Commands
	return cfg
}

// knownPreset reports whether name is a built-in theme preset.
func knownPreset(name string) bool {
	switch name {
	case "midnight", "neon", "dracula", "solarized", "matrix":
		return true
	}
	return false
}
