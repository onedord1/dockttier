# DOCKTTIER

## MISSION STATEMENT

You are building **dockttier** — a Docker CLI prettifier written in Go. It is a production-quality open-source tool that installs as a transparent proxy on top of the Docker CLI. Once installed, every Docker command the user runs (`docker build`, `docker push`, `docker pull`, `docker images`, etc.) is intercepted by dockttier, which runs the real Docker binary internally, parses the raw output stream, and re-renders it in a visually rich, information-dense terminal UI using ANSI colors and styled formatting.

The name is inspired by **Prettier** (the code formatter) — dockttier does for Docker output what Prettier does for code. It takes something functional but noisy and transforms it into something that is both beautiful and more informative.

**The tool must feel like it IS Docker** — same exit codes, same stdin handling, zero extra latency perception, complete signal transparency. A user should never feel like they are running a wrapper. The ONLY difference is that the output is dramatically more useful and beautiful.

---

## WHAT THE OUTPUT MUST LOOK LIKE

This is the highest-priority requirement. The visual output has already been designed and approved. You MUST replicate it precisely in the terminal.

The approved design uses the following color palette (translate to ANSI/lipgloss):

```
brand (mint/teal accent): #00d4aa   → lipgloss hex "#00d4aa"
text (primary):           #e6edf3
textMuted:                #7d8590
textDim:                  #484f58
green:                    #3fb950
cyan:                     #39d353
blue:                     #58a6ff
teal:                     #2ea5a0
yellow:                   #d29922
orange:                   #f0883e
red:                      #f85149
purple:                   #bc8cff
border:                   #30363d
bgPanel:                  #161b22
```

Every renderer described below must match these colors exactly. The brand color `#00d4aa` is the signature color of dockttier and appears on: the "dockttier ›" prefix on every command header, image IDs, layer short-hashes, digest values, and the tool name itself.

### Universal Header Line
Every command output begins with this exact header line format:
```
dockttier › docker <subcommand>  [args dimmed in textMuted]
```
- `dockttier` in brand color `#00d4aa`, bold
- ` › ` in textDim `#484f58`
- `docker <subcommand>` in the command-specific accent color (see per-command specs)
- args/flags in textMuted `#7d8590`

### Section Labels
Subsections within output use UPPERCASE labels in textDim `#484f58`, small letter-spaced monospace:
```
── build stages ──────────────
```
(faint horizontal rule rendered with box-drawing chars or dashes, label inline)

### Progress Bars
Used for `docker push`, `docker pull`. Render inline using Unicode block characters:
```
████████████████████░░░░  78%
```
- Filled: `█` in the appropriate color
- Empty: `░` in `#1c2128` (very dark)
- Bar width: 24 characters fixed
- Percentage right-aligned after bar
- Each layer gets its own labeled bar row

### Status Dots / Icons
- ✓  done/success      → green `#3fb950`
- ⊙  cached/exists     → textDim `#484f58`
- ✕  removed/deleted   → red `#f85149`
- ⚠  warning/skipped   → yellow `#d29922`
- ↑  uploading         → orange `#f0883e`
- ↓  downloading       → cyan `#39d353`
- ●  running           → green `#3fb950`
- ◌  paused            → yellow `#d29922`
- ○  stopped/exited    → textDim `#484f58`

### Badges
Small inline pills used for tags like `CACHED`, `REMOVED`, `RUNNING`, `LAYER EXISTS`:
```
[ CACHED ]   [ RUNNING ]   [ LAYER EXISTS ]
```
Render as: `[` + text + `]` with the text colored in the badge's accent color, surrounded by brackets in a dimmer version of the same color. Uppercase, monospace, compact.

### Dividers
Between logical sections:
```
────────────────────────────────────────
```
Full-width dim line using `─` in border color `#30363d`.

### Summary Footer
Every command ends with a summary line:
```
● Done    removed 4   skipped 1   reclaimed ~48 MB
```
- Left: status dot + status word, bold
- Right: key-value pairs, label in textMuted, value in accent color

---

## TECH STACK — EXACT LIBRARIES AND VERSIONS

Use Go 1.22+. All dependencies managed via Go modules (`go.mod`).

### Core Dependencies

```
github.com/charmbracelet/lipgloss        v0.10.0   (terminal styling, colors, layout)
github.com/charmbracelet/bubbles         v0.18.0   (spinner component ONLY — no full TUI)
github.com/charmbracelet/bubbletea       v0.26.0   (only for spinner rendering, not full TUI)
github.com/spf13/cobra                   v1.8.0    (CLI command structure)
github.com/mattn/go-isatty               v0.0.20   (TTY detection)
github.com/moby/buildkit                 latest    (BuildKit progress stream types for parsing)
golang.org/x/term                        latest    (PTY/terminal size + raw mode)
golang.org/x/sys                         latest    (OS-level signal handling)
```

### Why These Choices
- **lipgloss**: CSS-like API for terminal styling. Supports exact hex colors on 24-bit terminals. No TUI event loop needed — just renders strings. Perfect for streaming output.
- **cobra**: Same library Docker CLI itself uses. Familiar pattern.
- **go-isatty**: Detect if output is a real terminal. If piped (`docker images | grep foo`), disable all color/formatting and pass raw docker output through unchanged.
- **bubbles spinner**: Cherry-picked just the spinner component for the "waiting" state during long operations. Does not require the full Bubbletea TUI loop when used in simple render mode.
- **moby/buildkit**: Has the exact Go types for deserializing BuildKit's `--progress=rawjson` stream. Use `github.com/moby/buildkit/client/llb` and progress event types.

### Build & Package
```
github.com/goreleaser/goreleaser   (build + release pipeline)
```
GoReleaser produces: Linux `.deb`, `.rpm`, `tar.gz`, macOS binary, and a Homebrew formula in a single `goreleaser release` run.

---

## PROJECT STRUCTURE

Build the project with this exact directory layout:

```
dockttier/
├── go.mod
├── go.sum
├── main.go                          # Entry point — cobra root command setup
│
├── cmd/
│   └── root.go                      # Root cobra command; handles the proxy/shim logic
│
├── intercept/
│   ├── proxy.go                     # Core: exec real docker, pipe streams, handle exit code
│   ├── pty.go                       # PTY passthrough for interactive commands (docker exec -it)
│   └── signals.go                   # Forward SIGINT, SIGTERM, SIGWINCH to child process
│
├── detect/
│   └── command.go                   # Parse argv to determine which renderer to use
│
├── renderers/
│   ├── renderer.go                  # Interface: type Renderer interface { Render(line string) }
│   ├── build.go                     # docker build — BuildKit JSON stream parser + renderer
│   ├── push.go                      # docker push — layer progress bars
│   ├── pull.go                      # docker pull — layer progress bars
│   ├── rm.go                        # docker rm / rmi — removal list renderer
│   ├── images.go                    # docker images — styled table renderer
│   ├── container.go                 # docker container ls/ps — container table with CPU/mem
│   ├── df.go                        # docker system df — disk usage with stacked bar
│   ├── prune.go                     # docker system prune — deletion list + summary
│   ├── logs.go                      # docker logs — log level colorizer
│   ├── exec.go                      # docker exec — PTY passthrough (no transformation)
│   └── fallback.go                  # All other commands — passthrough with header only
│
├── style/
│   ├── theme.go                     # lipgloss style definitions, color constants
│   ├── table.go                     # Reusable table layout utilities
│   ├── progress.go                  # Reusable progress bar renderer (Unicode block chars)
│   └── badge.go                     # Reusable badge/pill renderer
│
├── packaging/
│   ├── dockttier.service            # Unused — placeholder
│   ├── debian/
│   │   ├── control
│   │   ├── postinst                 # update-alternatives --install
│   │   └── prerm                   # update-alternatives --remove
│   └── install.sh                   # Standalone install script (curl | bash)
│
├── .goreleaser.yml
├── Makefile
└── README.md
```

---

## HOW THE INTERCEPTION WORKS — FULL TECHNICAL SPEC

### The Shim Model

dockttier works as a **PATH shim**. The `.deb` postinstall script does:

```bash
# postinst
update-alternatives --install /usr/bin/docker docker /usr/local/bin/dockttier 100
```

This places dockttier at higher priority than the real Docker binary (`/usr/bin/docker.real` or wherever apt installed it). The real binary is never removed — it's still present and callable directly. `dockttier` calls it internally via its absolute path (resolved at startup by walking `$PATH` and skipping itself).

### Finding the Real Docker Binary

At startup, `proxy.go` must find the real docker binary by:
1. Reading its own executable path via `os.Executable()`
2. Walking `$PATH` entries in order
3. Returning the first `docker` binary whose absolute path does NOT equal its own path
4. Caching this path in a package-level var

```go
// pseudo-code
func findRealDocker() (string, error) {
    self, _ := os.Executable()
    selfResolved, _ := filepath.EvalSymlinks(self)
    for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
        candidate := filepath.Join(dir, "docker")
        if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
            if resolved != selfResolved {
                return candidate, nil
            }
        }
    }
    return "", fmt.Errorf("real docker binary not found in PATH")
}
```

### The Proxy Execution Flow

```go
// proxy.go — core execution
func RunProxied(args []string, renderer Renderer) int {
    realDocker, _ := findRealDocker()
    cmd := exec.Command(realDocker, args...)

    // Wire stdin always — needed for interactive commands
    cmd.Stdin = os.Stdin

    // If renderer is nil (passthrough mode) or not a TTY — wire stdout/stderr directly
    if renderer == nil || !isatty.IsTerminal(os.Stdout.Fd()) {
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.Run()
        return cmd.ProcessState.ExitCode()
    }

    // Otherwise: pipe stdout and stderr through the renderer
    stdoutPipe, _ := cmd.StdoutPipe()
    stderrPipe, _ := cmd.StderrPipe()

    cmd.Start()

    // Forward signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGWINCH)
    go func() {
        for sig := range sigCh {
            cmd.Process.Signal(sig)
        }
    }()

    // Print the header line first
    renderer.PrintHeader()

    // Stream stdout through renderer
    go renderer.Stream(stdoutPipe)
    // Merge stderr into renderer too (docker often writes progress to stderr)
    go renderer.StreamErr(stderrPipe)

    cmd.Wait()
    renderer.PrintSummary()

    return cmd.ProcessState.ExitCode()
}
```

### Exit Code Contract

`main()` MUST call `os.Exit()` with the exact exit code returned by the real docker process. Never swallow exit codes. If docker returns 1, dockttier returns 1.

```go
func main() {
    os.Exit(run())
}
```

### PTY Passthrough for Interactive Commands

Commands with `-it`, `-i`, or `-t` flags AND which launch a shell/exec session must use raw PTY mode. Detect this in `detect/command.go`:

```go
func NeedsRawPTY(args []string) bool {
    cmd := extractSubcommand(args) // "exec", "run", "attach"
    hasInteractive := slices.Contains(args, "-it") || 
                      slices.Contains(args, "-i") || 
                      slices.Contains(args, "-t")
    return (cmd == "exec" || cmd == "run" || cmd == "attach") && hasInteractive
}
```

For PTY commands, use `golang.org/x/term` to set raw mode and forward the terminal size via `SIGWINCH`.

---

## PER-COMMAND RENDERER SPECIFICATIONS

Each renderer implements:
```go
type Renderer interface {
    PrintHeader()
    Stream(r io.Reader)
    StreamErr(r io.Reader)
    PrintSummary()
}
```

---

### 1. `docker build` Renderer (`renderers/build.go`)

**Command color**: blue `#58a6ff`
**Trigger args**: `build`, `buildx build`, `image build`

**How Docker streams build output**:
When you add `--progress=rawjson` to the docker build command internally (inject this flag before passing to real docker when `--progress` is not already set), BuildKit emits newline-delimited JSON objects. Each object is a `buildkit/client.SolveStatus` progress event. The important fields:

```json
{
  "vertexes": [{ "digest": "sha256:...", "name": "RUN npm ci", "completed": "2024-01-01T...", "cached": false, "error": "" }],
  "statuses": [{ "id": "extracting sha256:a1b2", "vertex": "sha256:...", "name": "extracting", "current": 1048576, "total": 10485760 }],
  "logs": [{ "vertex": "sha256:...", "data": "npm warn ...\n", "timestamp": "..." }]
}
```

**What to render**:
```
dockttier › docker build -t myapp:latest .             [BUILDKIT]

── build stages ──────────────────────────────────────

 1  ✓  FROM node:20-alpine                              2.1s   sha256:a1b2c3d4
 2  ⊙  WORKDIR /app                                     cached
 3  ⊙  COPY package*.json ./                            cached
 4  ✓  RUN npm ci --only=production                    18.4s   sha256:e5f6a7b8
 5  ✓  COPY . .                                         0.3s
 6  ✓  RUN npm run build                               24.7s   sha256:c9d0e1f2
 7  ⊙  EXPOSE 3000                                     cached
 8  ✓  CMD ["node", "dist/index.js"]                    0.0s

── layer diff ────────────────────────────────────────

  sha256:a1b2  node:20-alpine base              71.2 MB
  sha256:e5f6  npm ci --only=production         14.3 MB   +14.3 MB
  sha256:c9d0  npm run build artifacts           2.1 MB    +2.1 MB

─────────────────────────────────────────────────────
● Build successful    image myapp:latest    total 87.6 MB    elapsed 46.1s

  image id  sha256:7f3a91b0c8e2d4f5...
```

**Color rules**:
- Step number: textDim
- ✓: green, ⊙: textDim
- Step name: text (white) if built, textDim if cached
- Time: yellow for >1s, textDim for cached/0s
- Layer hash (short): brand `#00d4aa`
- Size: blue for base, green for added layers
- Delta: green with `+` prefix

**Live rendering during build**:
As JSON events stream in, update the step list in-place. Use ANSI cursor-up (`\033[A`) + carriage return to rewrite the last N lines as steps complete. Show a spinner on the currently-running step.

---

### 2. `docker push` Renderer (`renderers/push.go`)

**Command color**: orange `#f0883e`
**Trigger args**: `push`, `image push`

**How Docker streams push output**:
Docker push writes JSON to stderr by default. Each line is:
```json
{"status":"Pushing","progressDetail":{"current":2097152,"total":14336000},"id":"e5f6a7b8"}
{"status":"Layer already exists","id":"a1b2c3d4"}
{"status":"Pushed","id":"7f3a91b0"}
{"status":"latest: digest: sha256:4a5b6c7d... size: 1234"}
```

Parse these JSON lines from stderr. Maintain a `map[string]*LayerState` keyed by `id`. On each update, rerender all layer rows.

**What to render**:
```
dockttier › docker push docker.io/acmecorp/myapp:latest

  registry  docker.io     repo  acmecorp/myapp    tag  latest

── layers ────────────────────────────────────────────

  ✓  7f3a91b0  app layer                  2.1 MB   ████████████████████████  pushed
  ✓  e5f6a7b8  npm modules               14.3 MB   ████████████████████████  pushed
  ⊙  a1b2c3d4  node:20-alpine base       71.2 MB   ████████████████████████  layer exists
  ⊙  f0a1b2c3  alpine linux               3.4 MB   ████████████████████████  layer exists
  ✓  9b8c7d6e  metadata                   1.2 KB   ████████████████████████  pushed

─────────────────────────────────────────────────────
● Push complete    new data 17.6 MB    skipped 74.6 MB    elapsed 8.3s

  digest  sha256:4a5b6c7d8e9f0a1b2c3d...
```

**Progress bar fill calculation**: `(current / total) * 24` filled blocks. Use `█` for filled, `░` for empty. When `status == "Layer already exists"`, bar is full but rendered in textDim `#484f58`. When `status == "Pushed"`, bar is full in green `#3fb950`.

**Layer name inference**: Docker doesn't provide human-readable layer names in the push stream. Maintain a lookup table from `docker inspect` output (run `docker inspect --format json <image>` before pushing to get layer names). If unavailable, show the ID only.

---

### 3. `docker pull` Renderer (`renderers/pull.go`)

**Command color**: cyan `#39d353`
**Trigger args**: `pull`, `image pull`

Same JSON stream format as push, different statuses:
- `"Pulling fs layer"` → show spinner, bar at 0%
- `"Downloading"` → show progress bar filling
- `"Pull complete"` → full bar in cyan, ✓ icon
- `"Already exists"` → full bar in textDim, ⊙ icon
- `"Verifying Checksum"` → show brief spinner
- `"Download complete"` → transition to pull complete

**What to render**:
```
dockttier › docker pull node:20-alpine

  from  docker.io/library/node:20-alpine

── layers ────────────────────────────────────────────

  ✓  3c4d5e6f  alpine base               3.4 MB   ████████████████████████  pull complete
  ✓  a7b8c9d0  node runtime             41.8 MB   ████████████████████████  pull complete
  ✓  e1f2a3b4  npm + yarn               25.9 MB   ████████████████████████  pull complete
  ⊙  c5d6e7f8  (already exists)          1.2 KB   ░░░░░░░░░░░░░░░░░░░░░░░░  already exists
  ✓  a9b0c1d2  entrypoint config         4.1 KB   ████████████████████████  pull complete

─────────────────────────────────────────────────────
● Pull complete    downloaded 71.1 MB    elapsed 11.7s

  digest  sha256:node20alpine...
  status  Image is up to date for node:20-alpine
```

---

### 4. `docker rm` / `docker rmi` Renderer (`renderers/rm.go`)

**Command color**: red `#f85149`
**Trigger args**: `rm`, `rmi`, `container rm`, `image rm`, `image remove`

Docker `rm` outputs one container ID or name per line to stdout on success, or an error line on failure. Parse line by line:
- Line with just an ID/name → success removal
- Line starting with `Error:` → failure, extract reason

Cross-reference with pre-run `docker ps -a` snapshot (run before the rm command) to get names and images for each ID.

**What to render**:
```
dockttier › docker rm $(docker ps -aq -f status=exited)

── containers ────────────────────────────────────────

  ✕  f3a8b2c1d9e0  myapp_dev           myapp:latest          [REMOVED]
  ✕  7e6d5c4b3a29  redis_cache         redis:7-alpine        [REMOVED]
  ✕  2b1a0f9e8d7c  postgres_test       postgres:15           [REMOVED]
  ⚠  9c8b7a6f5e4d  nginx_proxy         nginx:alpine          [RUNNING — SKIPPED]
  ✕  4d3c2b1a0f9e  worker_queue        myapp:latest          [REMOVED]

─────────────────────────────────────────────────────
● Done    removed 4    skipped 1    reclaimed ~48 MB
```

**For `docker rmi`**: same pattern but show image repo:tag, size reclaimed.

---

### 5. `docker images` Renderer (`renderers/images.go`)

**Command color**: purple `#bc8cff`
**Trigger args**: `images`, `image ls`, `image list`

Run the real `docker images --format json` (JSON output flag) then parse the structured JSON for accurate data. Format:
```json
{"Repository":"myapp","Tag":"latest","ID":"7f3a91b0c8e2","CreatedAt":"2024-01-15 10:23:41","Size":"87.6MB"}
```

**What to render** (fixed-width column table):
```
dockttier › docker images

  REPOSITORY     TAG           IMAGE ID      CREATED          SIZE       LAYERS
  ─────────────────────────────────────────────────────────────────────────────
  myapp          latest        7f3a91b0c8e2  2 mins ago        87.6 MB  ████░░  8
  myapp          v1.2.3        4c5d6e7f8a9b  3 days ago        86.1 MB  ████░░  8
  nginx          alpine        b3c4d5e6f7a8  2 weeks ago       41.0 MB  ██░░░░  6
  postgres       15            a1b2c3d4e5f6  1 month ago      379.2 MB  ████████ 14
  redis          7-alpine      9e8d7c6b5a4f  3 weeks ago       40.4 MB  ██░░░░  7
  node           20-alpine     3f2e1d0c9b8a  5 days ago       126.0 MB  █████░  5
  <none>         <none>        f1e2d3c4b5a6  1 week ago        84.2 MB  ████░░  7

  ─────────────────────────────────────────────────────────────────────────────
  total 7 images    dangling 1    total size 758.5 MB
  tip: docker image prune  to remove 1 dangling image
```

**Column color rules**:
- REPOSITORY: blue `#58a6ff`, `<none>` in textDim
- TAG: brand `#00d4aa`, `<none>` in textDim
- IMAGE ID: textDim
- CREATED: textMuted
- SIZE: green if <100MB, yellow if 100-300MB, red if >300MB. Inline mini bar (8 chars) relative to largest image.
- LAYERS: textDim
- `<none>` rows rendered at 60% opacity (dimmed entire row)
- Dangling image tip line: textDim label + brand color command

**Getting layer count**: After `docker images --format json`, run `docker inspect <id>` and count `RootFS.Layers`. Do this in parallel with goroutines for all images.

---

### 6. `docker container ls` / `docker ps` Renderer (`renderers/container.go`)

**Command color**: teal `#2ea5a0`
**Trigger args**: `ps`, `container ls`, `container list`, `container ps`

Run real docker with `--format json` or `--no-trunc --format "{{json .}}"` to get structured output.

**What to render**:
```
dockttier › docker container ls

  ID            NAME           IMAGE              STATUS   UPTIME     PORTS                  CPU%   MEM
  ─────────────────────────────────────────────────────────────────────────────────────────────────────
  9c8b7a6f5e4d  nginx_proxy    nginx:alpine       ● run    3 days     0.0.0.0:80→80/tcp      0.4%   4.2 MB
  1a2b3c4d5e6f  api_server     myapp:latest       ● run    2 hours    0.0.0.0:3000→3000/tcp  1.8%   62.4 MB
  7f8a9b0c1d2e  postgres_db    postgres:15        ● run    3 days     5432/tcp               0.1%   84.7 MB
  3d4e5f6a7b8c  redis_cache    redis:7-alpine     ● run    3 days     6379/tcp               0.0%   8.1 MB
  5b6c7d8e9f0a  worker_proc    myapp:latest       ● run    45 mins    —                      3.2%   58.3 MB
  2e3f4a5b6c7d  scheduler      myapp:latest       ◌ paused 1 day      —                      0.0%   44.1 MB

  ─────────────────────────────────────────────────────────────────────────────────────────────────────
  running 5    paused 1    total mem 261.8 MB
```

**CPU and MEM** are obtained by running `docker stats --no-stream --format json <id>` for each container in parallel (goroutines). Merge results before rendering.

**Color rules for CPU%**:
- >3% → orange `#f0883e`
- >0.5% → yellow `#d29922`
- ≤0.5% → green `#3fb950`

**Color rules for MEM**:
- >70MB → red `#f85149`
- >40MB → yellow `#d29922`
- ≤40MB → green `#3fb950`

**Status dot + word**:
- running → `● run` in green
- paused → `◌ paused` in yellow
- exited → `○ exited` in textDim

---

### 7. `docker system df` Renderer (`renderers/df.go`)

**Command color**: yellow `#d29922`
**Trigger args**: `system df`

Run real `docker system df --format json` for structured data.

**What to render**:
```
dockttier › docker system df

── storage usage ─────────────────────────────────────

  Images      7     758.5 MB  ██████████████████████░░░░░░░░  62%   ↩ 84.2 MB
  Containers  6      48.3 MB  ███░░░░░░░░░░░░░░░░░░░░░░░░░░░   4%   —
  Volumes     4     312.7 MB  ████████░░░░░░░░░░░░░░░░░░░░░░  26%   ↩ 21.4 MB
  Build cache 128   104.2 MB  ███░░░░░░░░░░░░░░░░░░░░░░░░░░░   8%   ↩ 104.2 MB

  ── total usage bar (stacked proportional) ──────────
  [████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░] (images=blue, vols=purple, cache=yellow, containers=teal)

  total used 1.22 GB    reclaimable 209.8 MB
  run  docker system prune  to reclaim
```

**Stacked bar**: A single 50-character bar where each segment color represents one category proportional to its share of total size. Characters per segment = `(size/total) * 50`. Colors: images=blue, containers=teal, volumes=purple, build cache=yellow.

**Reclaimable column**: yellow `#d29922` with `↩` icon if reclaimable > 0, textDim `—` if nothing to reclaim.

---

### 8. `docker system prune` Renderer (`renderers/prune.go`)

**Command color**: orange `#f0883e`
**Trigger args**: `system prune`, `container prune`, `image prune`, `volume prune`, `network prune`

Parse the real docker prune output line by line. Docker prints deleted IDs/names one per line then a "Total reclaimed space" line at the end.

**What to render**:
```
dockttier › docker system prune --volumes

  ⚠  This will remove all stopped containers, dangling images,
     unused networks, and volumes. Running containers are unaffected.

── deleted resources ─────────────────────────────────

  ✕  [container]  f3a8b2c1d9e0   myapp_dev_old              0 B
  ✕  [container]  2b1a0f9e8d7c   postgres_test              0 B
  ✕  [image]      f1e2d3c4b5a6   <none>:<none>          84.2 MB
  ✕  [volume]     myapp_data_v1  myapp_data_v1          21.4 MB
  ✕  [cache]      (128 entries)  build cache layers    104.2 MB

─────────────────────────────────────────────────────
● Prune complete    removed 5 resources    reclaimed 209.8 MB    elapsed 0.4s
```

**Type badge colors**:
- `[container]` in teal `#2ea5a0`
- `[image]` in blue `#58a6ff`
- `[volume]` in purple `#bc8cff`
- `[cache]` in yellow `#d29922`
- `[network]` in green `#3fb950`

---

### 9. `docker logs` Renderer (`renderers/logs.go`)

**Command color**: textMuted `#7d8590`
**Trigger args**: `logs`, `container logs`

Stream log lines. Detect log format:
1. **JSON structured logs**: Parse `{"level":"error","msg":"connection refused","ts":"..."}` — extract level, message, timestamp
2. **Logfmt**: `level=error msg="connection refused" ts=2024-01-15T10:23:41Z`
3. **Plain text**: Scan for level keywords

**Level color mapping**:
```
ERROR / FATAL / CRIT    → red     #f85149  + bold
WARN / WARNING          → yellow  #d29922
INFO                    → blue    #58a6ff
DEBUG / TRACE           → textDim #484f58
```

**What to render (JSON structured)**:
```
dockttier › docker logs api_server --follow

  10:23:41.123  INFO   Server started on port 3000
  10:23:41.456  INFO   Database connected  host=postgres:5432
  10:23:44.789  WARN   Slow query detected  duration=2.3s  query=SELECT *
  10:23:51.012  ERROR  Connection refused  target=redis:6379  retries=3
  10:23:51.234  DEBUG  Retrying in 5s
```

Timestamp: textDim. Level word: level-specific color, fixed 5-char width. Message: text primary. Additional key=value pairs: key in textDim, value in textMuted.

---

### 10. Fallback Renderer (`renderers/fallback.go`)

For ALL other docker commands not specifically handled (`docker network`, `docker volume`, `docker inspect`, `docker tag`, `docker login`, `docker logout`, `docker start`, `docker stop`, `docker restart`, etc.):

1. Print the dockttier header line
2. Pass stdout/stderr through completely unchanged (no transformation)
3. Print no summary footer (too risky without knowing command semantics)

This ensures dockttier NEVER breaks any Docker command it doesn't know about.

---

## STYLE SYSTEM — `style/` PACKAGE

### `style/theme.go`

```go
package style

import "github.com/charmbracelet/lipgloss"

var (
    Brand    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4aa"))
    BrandDim = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a885"))
    Text     = lipgloss.NewStyle().Foreground(lipgloss.Color("#e6edf3"))
    Muted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7d8590"))
    Dim      = lipgloss.NewStyle().Foreground(lipgloss.Color("#484f58"))
    Green    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950"))
    Cyan     = lipgloss.NewStyle().Foreground(lipgloss.Color("#39d353"))
    Blue     = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff"))
    Teal     = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea5a0"))
    Yellow   = lipgloss.NewStyle().Foreground(lipgloss.Color("#d29922"))
    Orange   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
    Red      = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
    Purple   = lipgloss.NewStyle().Foreground(lipgloss.Color("#bc8cff"))
    Bold     = lipgloss.NewStyle().Bold(true)
)

// Compose styles: Brand.Copy().Bold(true).Render("dockttier")
// Use .Copy() to derive from base without mutating
```

### `style/progress.go`

Implement a pure-string progress bar renderer (no Bubbletea required):
```go
func Bar(filled int, total int, color lipgloss.Color) string {
    // filled: number of filled blocks (0-total)
    // total: total bar width in characters (default 24)
    // Returns a rendered string like: ████████████░░░░░░░░░░░░
    filledStr := strings.Repeat("█", filled)
    emptyStr  := strings.Repeat("░", total-filled)
    return lipgloss.NewStyle().Foreground(color).Render(filledStr) +
           lipgloss.NewStyle().Foreground(lipgloss.Color("#1c2128")).Render(emptyStr)
}

func BarFromPercent(pct float64, width int, color lipgloss.Color) string {
    filled := int(pct / 100.0 * float64(width))
    return Bar(filled, width, color)
}
```

### `style/badge.go`

```go
func Badge(text string, color lipgloss.Color) string {
    style := lipgloss.NewStyle().
        Foreground(color).
        Border(lipgloss.NormalBorder(), false, false, false, false).
        Padding(0, 1)
    // Render as [ TEXT ] with brackets in dimmer color
    bracket := lipgloss.NewStyle().Foreground(color).Faint(true).Render
    inner   := lipgloss.NewStyle().Foreground(color).Bold(true).Render(text)
    return bracket("[") + inner + bracket("]")
}
```

### `style/table.go`

A simple fixed-width column layout helper that:
- Takes column widths `[]int` and a row of values `[]string`
- Returns a padded, colored row string
- Handles truncation with `…` for values exceeding column width
- Renders the header row with textDim + letter-spacing (spaces between chars)

---

## `detect/command.go` — Command Classification

This module reads `os.Args` (everything after `dockttier` itself) and returns which renderer to use.

```go
type CommandType int

const (
    CmdBuild CommandType = iota
    CmdPush
    CmdPull
    CmdRm
    CmdRmi
    CmdImages
    CmdContainerLS
    CmdPS
    CmdSystemDF
    CmdSystemPrune
    CmdContainerPrune
    CmdImagePrune
    CmdVolumePrune
    CmdNetworkPrune
    CmdLogs
    CmdExec
    CmdRun
    CmdAttach
    CmdFallback
)

func Classify(args []string) CommandType {
    // Normalize: strip flags to find the subcommand sequence
    // e.g. ["build", "--no-cache", "-t", "myapp:latest", "."] → CmdBuild
    // e.g. ["image", "ls"] → CmdImages
    // e.g. ["system", "df"] → CmdSystemDF
    // ... etc
}
```

Map all Docker subcommand aliases:
- `build`, `image build`, `buildx build` → `CmdBuild`
- `push`, `image push` → `CmdPush`
- `pull`, `image pull` → `CmdPull`
- `rm`, `container rm`, `container remove` → `CmdRm`
- `rmi`, `image rm`, `image remove` → `CmdRmi`
- `images`, `image ls`, `image list` → `CmdImages`
- `ps`, `container ls`, `container list`, `container ps` → `CmdContainerLS`
- `system df` → `CmdSystemDF`
- `system prune` → `CmdSystemPrune`
- `container prune` → `CmdContainerPrune`
- `image prune` → `CmdImagePrune`
- `volume prune` → `CmdVolumePrune`
- `network prune` → `CmdNetworkPrune`
- `logs`, `container logs` → `CmdLogs`
- `exec` → `CmdExec`
- `run` with `-it`/`-i`/`-t` → `CmdRun` (raw PTY)
- everything else → `CmdFallback`

---

## TERMINAL WIDTH AWARENESS

All renderers MUST be responsive to terminal width. At startup and on `SIGWINCH`, query terminal width:

```go
import "golang.org/x/term"

func TerminalWidth() int {
    w, _, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil || w == 0 {
        return 80 // safe default
    }
    return w
}
```

- Progress bars: fill available width minus label columns (min 8 chars, max 40 chars)
- Tables: adjust column widths proportionally, truncate strings with `…`
- Dividers: full terminal width using `─`
- Minimum supported width: 60 columns

---

## NO-COLOR / PIPE MODE

If stdout is NOT a TTY (piped or redirected), dockttier MUST disable all formatting:
```go
if !isatty.IsTerminal(os.Stdout.Fd()) {
    // Wire real docker stdout/stderr directly — no transformation
    // This preserves scripting compatibility: docker images | awk '{print $1}'
}
```

Also respect `NO_COLOR=1` env var and `DOCKTTIER_DISABLE=1` env var (complete passthrough with zero overhead).

---

## CONFIGURATION FILE

Support an optional `~/.config/dockttier/config.toml`:

```toml
[theme]
brand_color   = "#00d4aa"    # Override brand color
disable_emoji = false        # Replace emoji icons with ASCII fallback

[behavior]
show_build_logs  = false     # Show raw build log lines under each step
stats_timeout_ms = 500       # Timeout for docker stats calls (container ls)
pre_snapshot     = true      # Run docker ps before rm for name enrichment

[passthrough]
commands = ["login", "logout", "trust"]  # Always passthrough these commands
```

---

## `main.go` — ENTRY POINT

```go
package main

import (
    "os"
    "github.com/yourname/dockttier/cmd"
)

func main() {
    os.Exit(cmd.Execute())
}
```

`cmd/root.go` uses cobra but only defines the root command with `DisableFlagParsing: true` so ALL flags are passed through verbatim to the real docker binary without cobra trying to parse them.

```go
var rootCmd = &cobra.Command{
    Use:                "docker",
    Short:              "dockttier — Docker output, made lucrative",
    DisableFlagParsing: true,
    SilenceUsage:       true,
    RunE: func(cmd *cobra.Command, args []string) error {
        return intercept.Run(args)
    },
}
```

---

## PACKAGING — `.deb` PACKAGE

### `packaging/debian/control`
```
Package: dockttier-plugin
Version: 0.1.0
Architecture: amd64
Maintainer: Your Name <you@example.com>
Depends: docker-ce | docker.io
Description: Docker CLI output prettifier
 dockttier enhances Docker command output with rich terminal formatting.
 Installs as a transparent proxy — use docker commands exactly as normal.
```

### `packaging/debian/postinst`
```bash
#!/bin/bash
set -e
update-alternatives --install /usr/bin/docker docker /usr/local/bin/dockttier 100
echo "dockttier installed. All docker commands are now prettified."
echo "To disable temporarily: DOCKTTIER_DISABLE=1 docker <command>"
echo "To uninstall: sudo apt remove dockttier-plugin"
```

### `packaging/debian/prerm`
```bash
#!/bin/bash
set -e
update-alternatives --remove docker /usr/local/bin/dockttier
```

---

## `.goreleaser.yml`

```yaml
project_name: dockttier

builds:
  - main: ./main.go
    binary: dockttier
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

nfpms:
  - id: dockttier
    package_name: dockttier-plugin
    file_name_template: "{{ .PackageName }}_{{ .Version }}_{{ .Arch }}"
    homepage: https://github.com/yourname/dockttier
    description: Docker CLI output prettifier
    maintainer: Your Name <you@example.com>
    license: MIT
    formats: [deb, rpm]
    bindir: /usr/local/bin
    scripts:
      postinstall: packaging/debian/postinst
      preremove: packaging/debian/prerm
    dependencies:
      - docker-ce

brews:
  - repository:
      owner: yourname
      name: homebrew-tap
    directory: Formula
    description: Docker CLI output prettifier
    install: bin.install "dockttier"
```

---

## Makefile

```makefile
.PHONY: build run install dev test

build:
	go build -o bin/dockttier ./main.go

dev:
	go run ./main.go $(ARGS)

install: build
	sudo cp bin/dockttier /usr/local/bin/dockttier
	sudo update-alternatives --install /usr/bin/docker docker /usr/local/bin/dockttier 100

uninstall:
	sudo update-alternatives --remove docker /usr/local/bin/dockttier
	sudo rm -f /usr/local/bin/dockttier

test:
	go test ./...

release:
	goreleaser release --clean
```

---

## IMPLEMENTATION ORDER — EXECUTE IN THIS SEQUENCE

Build the project in this exact order. Do not skip ahead:

1. **Init** — `go mod init github.com/yourname/dockttier`, install all dependencies
2. **`style/` package** — theme, progress bar, badge, table utilities
3. **`intercept/proxy.go`** — real docker finder, exec wrapper, exit code handling
4. **`intercept/signals.go`** — signal forwarding
5. **`intercept/pty.go`** — PTY passthrough
6. **`detect/command.go`** — command classifier
7. **`renderers/renderer.go`** — interface definition
8. **`renderers/fallback.go`** — passthrough renderer (needed for all unimplemented commands)
9. **`cmd/root.go`** + **`main.go`** — wire everything, build and verify `docker help` passes through
10. **`renderers/images.go`** — start with tables (simpler, no streaming)
11. **`renderers/container.go`** — similar table pattern
12. **`renderers/rm.go`** — line-by-line simple
13. **`renderers/df.go`** — system df with stacked bar
14. **`renderers/prune.go`** — prune output
15. **`renderers/pull.go`** — streaming JSON progress
16. **`renderers/push.go`** — streaming JSON progress (same pattern as pull)
17. **`renderers/build.go`** — most complex: BuildKit JSON stream + live rerender
18. **`renderers/logs.go`** — log format detection + colorization
19. **`renderers/exec.go`** — PTY passthrough integration
20. **Config file** — `~/.config/dockttier/config.toml` support
21. **Packaging** — `.goreleaser.yml`, Makefile, debian scripts
22. **README.md** — install instructions, usage, feature list

---

## ACCEPTANCE CRITERIA

The project is complete when ALL of the following are true:

- [ ] `sudo apt install ./dockttier-plugin_0.1.0_amd64.deb` registers dockttier as the docker shim
- [ ] `docker images` renders the styled table with size bars and color coding
- [ ] `docker ps` renders the styled container table with CPU/mem stats
- [ ] `docker build -t foo .` renders BuildKit step-by-step with timing and layer diff
- [ ] `docker push` renders per-layer progress bars filling in real time
- [ ] `docker pull` renders per-layer progress bars filling in real time
- [ ] `docker rm <id>` renders the removal list with status badges
- [ ] `docker system df` renders the disk usage with stacked proportional bar
- [ ] `docker system prune` renders the deletion list and reclaimed space summary
- [ ] `docker logs <container>` colorizes log levels
- [ ] `docker exec -it <container> bash` works perfectly (raw PTY, no transformation)
- [ ] `docker images | grep node` works (pipe detection disables formatting)
- [ ] `DOCKTTIER_DISABLE=1 docker images` outputs raw docker output
- [ ] All commands return the EXACT same exit code as real docker
- [ ] Unknown docker subcommands passthrough completely unchanged
- [ ] Terminal resize (`SIGWINCH`) updates progress bar widths correctly
- [ ] `go build` produces a single static binary under 15MB

---

## IMPORTANT CONSTRAINTS

1. **Never modify the actual docker binary**. dockttier only sits in front of it.
2. **Zero latency for passthrough commands**. For unknown commands or pipe mode, exec real docker with `exec.Command` with direct stdio wiring — no buffering.
3. **Single binary**. No runtime dependencies. No Python, no Node.js, no config required on first run.
4. **The visual design is non-negotiable**. Every color, every icon, every column layout must match the approved design specified in this document exactly.
5. **Test on Docker version 24+ (BuildKit default)**. BuildKit is default since Docker 23.0. The build renderer assumes BuildKit.
6. **Do not use Bubbletea's full event loop**. You don't need a TUI. You're writing to stdout sequentially using ANSI cursor control for in-place updates during streaming commands.
7. **All ANSI escape codes must be suppressed** when `NO_COLOR` env var is set or when stdout is not a TTY.
