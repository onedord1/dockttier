# dockttier - Simplify Logs with enhanced visual !!

**Docker output, made beautiful.**

dockttier is a transparent prettifier for the Docker CLI, written in Go. It
installs as a shim in front of the real `docker` binary: every command you run
(`docker build`, `push`, `pull`, `images`, `ps`, `rm`, `system df`, `system
prune`, `logs`, …) is intercepted, run against the real docker internally, and
re-rendered as a rich, information-dense terminal UI.

It is named after [Prettier](https://prettier.io) — dockttier does for Docker
output what Prettier does for code.

> The tool feels like it **is** Docker: identical exit codes, full stdin
> handling, complete signal transparency, and zero changes to your workflow.
> The only difference is that the output is dramatically more useful.

---

## Features

- **Transparent shim** — runs the real docker binary, never modifies it.
- **Exit-code & signal transparent** — same exit codes; `SIGINT`/`SIGTERM`/`SIGWINCH` forwarded.
- **Pipe-safe** — when stdout isn't a terminal (`docker images | grep node`), output is raw docker, unchanged.
- **Per-command renderers**:
  - `build` — BuildKit step list with live timing (parses `--progress=rawjson`).
  - `push` / `pull` — per-layer progress bars, "layer exists" detection.
  - `images` — styled table with color-coded sizes, size bars and layer counts.
  - `ps` / `container ls` — table with live CPU and memory per container.
  - `network ls` — styled table, driver color-coding.
  - `volume ls` — styled table with driver and links.
  - `rm` / `rmi` / `network rm` / `volume rm` — removal list with status badges and name enrichment.
  - `system df` — per-category usage with a stacked proportional bar.
  - `system prune` (and `container`/`image`/`volume`/`network prune`) — deletion list + reclaimed summary.
  - `logs` — log-level colorizer for JSON, logfmt and plain text.
  - `exec -it` — raw PTY passthrough (interactive shells work perfectly).
  - everything else — header line, then unchanged passthrough.
- **Responsive** — adapts to terminal width and `SIGWINCH`.
- **Configurable** — optional `~/.config/dockttier/config.toml`.
- **Single static binary** (~5 MB), no runtime dependencies.

---

## Try it without installing

You don't have to register the shim to test it. Preview the look offline (no
docker needed):

```bash
make build
./bin/dockttier --dockttier-demo                 # preview the active theme
DOCKTTIER_THEME=neon ./bin/dockttier --dockttier-demo
```

Or alias it for one shell:

```bash
alias docker="$PWD/bin/dockttier"
docker images
unalias docker
```

See [TESTING.md](TESTING.md) for the full local test-drive guide and
[THEMES.md](THEMES.md) for the 5 built-in themes.

---

## Themes

5 built-in themes, each with its own palette, progress-bar glyphs and spinner:
`midnight` (default), `neon`, `dracula`, `solarized`, `matrix`. Select one in
config:

```toml
[theme]
preset = "neon"
```

Quick-switch for testing with `DOCKTTIER_THEME=<name>`. Full details and
ready-to-copy config files (`themes/*.toml`) are in [THEMES.md](THEMES.md).

---

## Error handling

When a docker operation fails (build/pull/push/rm), dockttier replaces the
usual wall of daemon text with a compact **error panel**: a short title, the one
key message, and an actionable hint. For example a failed pull becomes:

```
── error ──────────────────────────
  ✕  Image not found
     pull access denied for acme/app, repository does not exist or may require 'docker login'
     ↳ hint: check the image name and tag are correct
```

The real docker exit code is always preserved.

---

## Install

### Script - Recommanded

```bash
curl -fsSL https://raw.githubusercontent.com/onedord1/dockttier/main/packaging/install.sh | bash
```

### Debian / Ubuntu (`.deb`) - *Coming Soon*

```bash
sudo apt install ./dockttier-plugin_0.1.0_amd64.deb
```

The post-install script registers dockttier as the `docker` alternative
(`update-alternatives --install /usr/bin/docker docker /usr/local/bin/dockttier 100`).
The real docker binary is left in place.

### From source

```bash
make build          # produces ./bin/dockttier
make install        # copies to /usr/local/bin and registers the shim
```

---

## Usage

Just use docker as you always have:

```bash
docker build -t myapp:latest .
docker pull node:20-alpine
docker images
docker ps
docker system df
```

Escape hatches:

```bash
DOCKTTIER_DISABLE=1 docker images   # full raw passthrough, zero overhead
NO_COLOR=1 docker images            # also disables formatting
docker images | grep node           # piped => automatic raw passthrough
```

Uninstall:

```bash
sudo apt remove dockttier-plugin    # or: make uninstall
```

---

## Configuration

Optional, at `~/.config/dockttier/config.toml` (see [`config.example.toml`](config.example.toml)):

```toml
[theme]
brand_color   = "#00d4aa"
disable_emoji = false

[behavior]
show_build_logs  = false
stats_timeout_ms = 500
pre_snapshot     = true

[passthrough]
commands = ["login", "logout", "trust"]
```

Invalid values fall back to defaults with a warning on stderr; a missing file
is the normal case and uses built-in defaults silently.

---

## How it works

1. At startup dockttier resolves its own path and walks `$PATH` to find the
   **real** docker binary (the first `docker` whose resolved path differs from
   its own).
2. It classifies the argv to pick a renderer (`detect/`).
3. It runs the real docker as a child process (`intercept/`), wiring stdin and
   forwarding signals, and either:
   - pipes stdout/stderr through the renderer (`renderers/`), or
   - wires streams directly (passthrough / pipe / `NO_COLOR` / `DOCKTTIER_DISABLE`), or
   - allocates a raw PTY for interactive `exec`/`run`/`attach -it` sessions.
4. It exits with docker's exact exit code.

---

## Design notes & deviations

This implementation follows the dockttier design spec, with two deliberate,
documented engineering choices that keep the binary small (~5 MB ≪ 15 MB) and
the build fast:

- **BuildKit types are defined locally** rather than importing the very large
  `github.com/moby/buildkit` module. We deserialize only the `vertexes` fields
  of the `--progress=rawjson` stream that we actually render.
- **The spinner is a lightweight frame renderer** instead of running a full
  Bubbletea event loop, matching the spec's guidance to avoid a TUI loop and
  use sequential ANSI cursor control for in-place updates.
- The transparent shim forwards argv directly rather than routing through a
  cobra command, which is the most faithful way to avoid intercepting docker's
  own `--help`/`help`/completion handling.

---

## Development

```bash
make build      # build ./bin/dockttier
make fmt        # gofmt
make vet        # go vet ./...
make snapshot   # local goreleaser build of all artifacts
```

Requires Go 1.22+.

## License

MIT
