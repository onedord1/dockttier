# Testing dockttier locally (without installing)

You already have the binary at `./bin/dockttier`. You can exercise everything
**without** registering it as the system `docker` shim. Nothing here touches
`/usr/bin/docker` or `update-alternatives`.

> Rebuild any time with: `make build` (or `go build -o bin/dockttier .`)

---

## 0. The fastest preview — no docker needed

dockttier has a built-in offline preview that renders sample output (build
steps, push bars, an images table, status icons, badges, and a distilled error
panel) using the **active theme**:

```bash
./bin/dockttier --dockttier-demo
```

Preview any theme instantly with the `DOCKTTIER_THEME` env var:

```bash
DOCKTTIER_THEME=neon      ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=dracula   ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=solarized ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=matrix    ./bin/dockttier --dockttier-demo
DOCKTTIER_THEME=midnight  ./bin/dockttier --dockttier-demo
```

This is the quickest way to judge "does it look good" before pointing it at a
real daemon.

---

## 1. Run it against the real docker, transparently

dockttier finds the real `docker` on your `$PATH` automatically (it skips
itself). So you can just call it in place of docker:

```bash
./bin/dockttier images
./bin/dockttier ps -a
./bin/dockttier system df
./bin/dockttier pull alpine:latest
./bin/dockttier build -t demo:latest .
```

### Make a temporary `docker` alias (recommended for a real test drive)

This makes `docker …` resolve to dockttier **only in your current shell** — no
system changes, fully reversible by closing the shell:

```bash
alias docker="$PWD/bin/dockttier"

docker images
docker ps
docker pull node:20-alpine
docker build -t myapp:latest .

unalias docker     # revert
```

### Or put it first on PATH for one shell

```bash
mkdir -p /tmp/dockttier-shim
ln -sf "$PWD/bin/dockttier" /tmp/dockttier-shim/docker
export PATH="/tmp/dockttier-shim:$PATH"

docker images      # now goes through dockttier
# open a new shell (or unset PATH entry) to revert
```

---

## 2. Important: colors only show on a real terminal (TTY)

dockttier is **pipe-safe by design**. If stdout is not a terminal (piped or
redirected), it passes raw docker output through unchanged so scripts keep
working. That means:

```bash
./bin/dockttier images            # in your terminal  -> styled, colorful
./bin/dockttier images | cat      # piped             -> raw docker output
./bin/dockttier images > out.txt  # redirected        -> raw docker output
```

If you are capturing output through a tool/log and want to *see* the styled
version, force a pseudo-terminal with `script`:

```bash
script -qec "./bin/dockttier images" /dev/null
```

To strip ANSI codes for a clean text capture:

```bash
script -qec "./bin/dockttier images" /dev/null | sed -r 's/\x1b\[[0-9;]*m//g'
```

---

## 3. What to try for each renderer

| Command | What you'll see |
|---|---|
| `docker images` | table with color-coded sizes, mini size bars, layer counts |
| `docker ps` / `docker ps -a` | container table with live CPU% / MEM, status dots |
| `docker network ls` | styled table, driver color-coding |
| `docker volume ls` | styled table with driver and links |
| `docker build -t x .` | live BuildKit step list with timings + summary |
| `docker pull <img>` | per-layer progress bars filling in real time |
| `docker push <img>` | per-layer progress bars, "layer exists" detection |
| `docker rm <id>` / `docker rmi <id>` | removal list with badges + reclaimed summary |
| `docker network rm <id>` / `docker volume rm <id>` | styled removal list with badges |
| `docker system df` | per-category usage + stacked proportional bar |
| `docker system prune` | deletion list with type badges + reclaimed summary |
| `docker logs <ctr>` | log levels colorized (ERROR red, WARN yellow, …) |
| `docker exec -it <ctr> sh` | raw interactive shell (untouched passthrough) |
| `docker network ls`, etc. | header line, then unchanged passthrough (fallback) |

---

## 4. Test the escape hatches

```bash
DOCKTTIER_DISABLE=1 ./bin/dockttier images   # full raw passthrough, zero styling
NO_COLOR=1          ./bin/dockttier images   # also disables styling
./bin/dockttier images | grep alpine         # piping auto-disables styling
```

## 5. Verify transparency (it must behave exactly like docker)

```bash
# Exit codes must match docker exactly:
./bin/dockttier rm no_such_container; echo "dockttier=$?"
docker          rm no_such_container; echo "docker=$?"

# Ctrl-C during a pull/build is forwarded to the real docker process.
```

## 6. Test the error panel

Trigger a real failure and watch dockttier distill docker's noisy output into a
short panel (title + key message + hint):

```bash
script -qec "./bin/dockttier pull totally/nonexistent-image-xyz:v9" /dev/null
```

See [THEMES.md](THEMES.md) for switching color schemes, and
[README.md](README.md) for installing it system-wide when you're happy.
