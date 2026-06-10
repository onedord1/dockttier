#!/usr/bin/env bash
#
# dockttier real-docker demo
# ---------------------------
# Runs a guided tour of dockttier against your REAL docker daemon so you can see
# every renderer with live data: images / ps / network / volume / system df,
# a live pull, an animated build, tag, and styled removals.
#
# Everything it creates (a demo image, tag, network and volume, and one pulled
# image tag) is removed again at the end. Your existing resources are untouched.
#
# Usage:
#   ./dockttier-demo.sh          # interactive: press Enter between steps
#   ./dockttier-demo.sh --auto   # no prompts, short pauses instead
#
set -uo pipefail

AUTO=0
[ "${1:-}" = "--auto" ] && AUTO=1

# ---- locate the dockttier binary -------------------------------------------
BIN=""
for c in "$HOME/Desktop/dockttier/bin/dockttier" "./bin/dockttier" "$(command -v dockttier 2>/dev/null || true)"; do
    if [ -n "$c" ] && [ -x "$c" ]; then BIN="$c"; break; fi
done
if [ -z "$BIN" ]; then
    echo "dockttier binary not found."
    echo "Build it first:  cd ~/Desktop/dockttier && make build"
    exit 1
fi

# ---- demo resource names (clearly namespaced, removed at the end) ----------
DEMO_IMAGE="dockttier-demo"
DEMO_PULL="alpine:3.19"        # pulled fresh to show live progress, removed after
DEMO_NET="dockttier_demo_net"
DEMO_VOL="dockttier_demo_vol"
PULLED_IT=0

cleanup() {
    echo
    echo "Cleaning up demo resources…"
    "$BIN" rmi "${DEMO_IMAGE}:latest" "${DEMO_IMAGE}:v2" >/dev/null 2>&1 || true
    docker network rm "$DEMO_NET" >/dev/null 2>&1 || true
    docker volume rm "$DEMO_VOL" >/dev/null 2>&1 || true
    [ "$PULLED_IT" = "1" ] && docker rmi "$DEMO_PULL" >/dev/null 2>&1 || true
    rm -rf "$CTX" 2>/dev/null || true
    echo "Done."
}
trap cleanup EXIT

step() {
    echo
    printf '\033[38;2;0;212;170m━━ %s \033[0m\033[2m' "$1"
    printf '━%.0s' $(seq 1 50); printf '\033[0m\n\n'
    if [ "$AUTO" = "1" ]; then sleep 1; else
        printf '\033[2m   (press Enter to run: %s)\033[0m' "$2"; read -r _
    fi
}

# ---- check the daemon ------------------------------------------------------
if ! docker info >/dev/null 2>&1; then
    echo "The docker daemon doesn't appear to be running. Start it and retry."
    exit 1
fi

clear
echo "dockttier real-docker demo — using: $BIN"
echo "(creates a throwaway image/tag/network/volume, all cleaned up at the end)"

step "docker images"            "docker images";            "$BIN" images
step "docker ps -a"             "docker ps -a";             "$BIN" ps -a
step "docker network ls"        "docker network ls";        "$BIN" network ls
step "docker volume ls"         "docker volume ls";         "$BIN" volume ls
step "docker system df"         "docker system df";         "$BIN" system df

step "docker pull (live progress)" "docker pull $DEMO_PULL"
# Remember whether the user already had this image; only remove it on cleanup
# if the demo introduced it. We remove it now so the pull shows live progress.
if docker image inspect "$DEMO_PULL" >/dev/null 2>&1; then PULLED_IT=0; else PULLED_IT=1; fi
docker rmi "$DEMO_PULL" >/dev/null 2>&1 || true
"$BIN" pull "$DEMO_PULL"

# ---- build context for the animated build demo ----------------------------
CTX="$(mktemp -d)"
cat > "$CTX/Dockerfile" <<'EOF'
FROM alpine:3.19
RUN echo "resolving dependencies..." && sleep 2
RUN apk add --no-cache curl >/dev/null 2>&1 || true
RUN echo "compiling application..." && sleep 2
RUN echo "finalizing image..." && sleep 1
CMD ["echo", "hello from the dockttier demo"]
EOF

step "docker build (animated stages)" "docker build --no-cache -t ${DEMO_IMAGE}:latest"
"$BIN" build --no-cache -t "${DEMO_IMAGE}:latest" "$CTX"

step "docker tag"               "docker tag ${DEMO_IMAGE}:latest ${DEMO_IMAGE}:v2"
"$BIN" tag "${DEMO_IMAGE}:latest" "${DEMO_IMAGE}:v2"

step "docker network rm"        "docker network rm $DEMO_NET"
docker network create "$DEMO_NET" >/dev/null 2>&1 || true
"$BIN" network rm "$DEMO_NET"

step "docker volume rm"         "docker volume rm $DEMO_VOL"
docker volume create "$DEMO_VOL" >/dev/null 2>&1 || true
"$BIN" volume rm "$DEMO_VOL"

step "docker rmi (styled removal)" "docker rmi ${DEMO_IMAGE}:v2 ${DEMO_IMAGE}:latest"
"$BIN" rmi "${DEMO_IMAGE}:v2" "${DEMO_IMAGE}:latest"

echo
echo "That's the tour. (Demo resources are being cleaned up now.)"
