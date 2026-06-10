#!/bin/bash
# dockttier standalone installer: downloads the latest release binary for the
# current platform and registers it as the docker shim via update-alternatives.
#
#   curl -fsSL https://raw.githubusercontent.com/onedord1/dockttier/main/packaging/install.sh | bash
#
set -euo pipefail

REPO="onedord1/dockttier"
BINDIR="/usr/local/bin"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

echo "Resolving latest dockttier release for ${os}/${arch}…"
tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -m1 '"tag_name"' | cut -d '"' -f4)"
if [ -z "${tag}" ]; then
    echo "could not determine latest release tag" >&2
    exit 1
fi
version="${tag#v}"

url="https://github.com/${REPO}/releases/download/${tag}/dockttier_${version}_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

echo "Downloading ${url}…"
curl -fsSL "${url}" -o "${tmp}/dockttier.tar.gz"
tar -xzf "${tmp}/dockttier.tar.gz" -C "${tmp}"

echo "Installing to ${BINDIR}/dockttier (sudo may prompt)…"
sudo install -m 0755 "${tmp}/dockttier" "${BINDIR}/dockttier"

if command -v update-alternatives >/dev/null 2>&1; then
    sudo update-alternatives --install /usr/bin/docker docker "${BINDIR}/dockttier" 100
    echo "dockttier registered as the docker shim."
else
    echo "update-alternatives not found; add ${BINDIR} ahead of the real docker on your PATH,"
    echo "or symlink ${BINDIR}/dockttier as 'docker'."
fi

echo "Done. Disable temporarily with DOCKTTIER_DISABLE=1 docker <command>."
