#!/usr/bin/env bash
# dockttier installer / uninstaller.
#
# Install (downloads the latest release and shims the docker CLI):
#   curl -fsSL https://raw.githubusercontent.com/onedord1/dockttier/main/packaging/install.sh | bash
#
# Uninstall (removes the shim and binary, restores the real docker):
#   curl -fsSL https://raw.githubusercontent.com/onedord1/dockttier/main/packaging/install.sh | bash -s -- --uninstall
#
set -euo pipefail

REPO="onedord1/dockttier"
BINDIR="/usr/local/bin"
BIN="${BINDIR}/dockttier"   # the real dockttier binary
SHIM="${BINDIR}/docker"     # symlink that shadows the real docker on PATH

# sudo helper (no-op when already root).
SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi

uninstall() {
    echo "Uninstalling dockttier…"
    # Remove the shim only if it is our symlink.
    if [ -L "${SHIM}" ]; then
        target="$(readlink -f "${SHIM}" 2>/dev/null || true)"
        if [ "${target}" = "$(readlink -f "${BIN}" 2>/dev/null || echo "${BIN}")" ] || [ "${target}" = "${BIN}" ]; then
            ${SUDO} rm -f "${SHIM}" && echo "  removed shim ${SHIM}"
        else
            echo "  ${SHIM} is not a dockttier symlink — leaving it untouched"
        fi
    fi
    # Restore any real docker we moved aside during install.
    if [ -e "${SHIM}.pre-dockttier" ]; then
        ${SUDO} mv -f "${SHIM}.pre-dockttier" "${SHIM}" && echo "  restored previous ${SHIM}"
    fi
    # Clean up any stale update-alternatives entry from older installs.
    if command -v update-alternatives >/dev/null 2>&1; then
        ${SUDO} update-alternatives --remove docker "${BIN}" >/dev/null 2>&1 || true
    fi
    ${SUDO} rm -f "${BIN}" && echo "  removed ${BIN}"
    echo
    echo "dockttier removed. 'docker' now points at the real Docker CLI again."
    echo "Open a new shell (or run 'hash -r') so your shell forgets the old path."
    exit 0
}

if [ "${1:-}" = "--uninstall" ] || [ "${1:-}" = "uninstall" ]; then
    uninstall
fi

# --- detect platform --------------------------------------------------------
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${arch}" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "unsupported architecture: ${arch}" >&2; exit 1 ;;
esac

# --- resolve the latest release tag -----------------------------------------
echo "Resolving latest dockttier release for ${os}/${arch}…"
api_json="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")" || {
    echo "could not reach the GitHub releases API for ${REPO}" >&2
    exit 1
}
tag="$(printf '%s\n' "${api_json}" | grep '"tag_name"' | head -n1 | cut -d '"' -f4)"
if [ -z "${tag}" ]; then
    echo "could not determine latest release tag (is there a published release?)" >&2
    exit 1
fi
version="${tag#v}"

# --- download + extract -----------------------------------------------------
url="https://github.com/${REPO}/releases/download/${tag}/dockttier_${version}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

echo "Downloading ${url}…"
curl -fsSL "${url}" -o "${tmp}/dockttier.tar.gz"
tar -xzf "${tmp}/dockttier.tar.gz" -C "${tmp}"

# --- install the binary -----------------------------------------------------
echo "Installing to ${BIN} (sudo may prompt)…"
${SUDO} install -m 0755 "${tmp}/dockttier" "${BIN}"

# --- install the shim via a PATH symlink ------------------------------------
# This works even when /usr/bin/docker is a regular file (docker-ce/docker.io),
# which update-alternatives refuses to replace. ${BINDIR} must precede the real
# docker's directory on PATH (it does on standard Debian/Ubuntu setups).

# Remove any stale/broken update-alternatives entry from earlier versions.
if command -v update-alternatives >/dev/null 2>&1; then
    ${SUDO} update-alternatives --remove docker "${BIN}" >/dev/null 2>&1 || true
fi

# Back up a pre-existing *real* docker living in ${BINDIR} (uncommon).
if [ -e "${SHIM}" ] && [ ! -L "${SHIM}" ]; then
    ${SUDO} mv -f "${SHIM}" "${SHIM}.pre-dockttier"
    echo "  backed up existing ${SHIM} -> ${SHIM}.pre-dockttier"
fi

${SUDO} ln -sf "${BIN}" "${SHIM}"
echo "dockttier registered as the docker shim (${SHIM} -> ${BIN})."

# --- verify the shim is actually ahead of the real docker -------------------
real_docker="$(PATH="${PATH}" command -v docker.real 2>/dev/null || true)"
# Find what a fresh shell would resolve 'docker' to.
resolved=""
IFS=':' read -r -a dirs <<< "${PATH}"
for d in "${dirs[@]}"; do
    if [ -x "${d}/docker" ]; then resolved="${d}/docker"; break; fi
done

echo
if [ "${resolved}" = "${SHIM}" ]; then
    echo "✓ Installed. 'docker' now resolves to dockttier."
else
    echo "⚠ Installed, but 'docker' currently resolves to: ${resolved:-unknown}"
    echo "  Ensure ${BINDIR} comes before that directory on your PATH."
fi
echo
echo "  IMPORTANT: run 'hash -r' (bash) / 'rehash' (zsh), or open a new terminal,"
echo "  so your current shell stops using the cached docker path."
echo
echo "  Disable once:   DOCKTTIER_DISABLE=1 docker <command>"
echo "  Uninstall:      curl -fsSL https://raw.githubusercontent.com/${REPO}/main/packaging/install.sh | bash -s -- --uninstall"
