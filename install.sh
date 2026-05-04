#!/bin/sh
# cron-doctor installer.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/HeytalePazguato/cron-doctor/main/install.sh | sh
#
# Environment variables:
#   CRON_DOCTOR_VERSION  Tag to install (default: latest release).
#   PREFIX               Install prefix (default: /usr/local). Falls back to
#                        $HOME/.local if /usr/local is not writable.

set -eu

REPO="HeytalePazguato/cron-doctor"
BINARY="cron-doctor"
VERSION="${CRON_DOCTOR_VERSION:-latest}"

err() { printf 'error: %s\n' "$*" >&2; exit 1; }

need() {
    command -v "$1" >/dev/null 2>&1 || err "missing required command: $1"
}

need uname
need tar
need mkdir
need install
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD="wget -qO-"
else
    err "neither curl nor wget is available"
fi

os=$(uname -s)
case "$os" in
    Linux)  os=Linux ;;
    Darwin) os=Darwin ;;
    *)      err "unsupported OS: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64)  arch=x86_64 ;;
    arm64|aarch64) arch=arm64 ;;
    *)             err "unsupported arch: $arch" ;;
esac

# goreleaser archive naming: cron-doctor_<Os>_<Arch>.tar.gz
# Os comes from $GOOS title-cased: linux -> Linux, darwin -> Darwin.
asset="${BINARY}_${os}_${arch}.tar.gz"

if [ "$VERSION" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
    url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
fi

prefix="${PREFIX:-/usr/local}"
bindir="${prefix}/bin"
if ! mkdir -p "$bindir" 2>/dev/null || ! [ -w "$bindir" ]; then
    bindir="${HOME}/.local/bin"
    mkdir -p "$bindir"
    printf 'note: %s not writable, installing to %s\n' "${prefix}/bin" "$bindir"
fi

tmp=$(mktemp -d 2>/dev/null || mktemp -d -t cron-doctor)
trap 'rm -rf "$tmp"' EXIT

printf 'downloading %s\n' "$url"
$DOWNLOAD "$url" | tar -xz -C "$tmp"

install -m 0755 "${tmp}/${BINARY}" "${bindir}/${BINARY}"
printf 'installed %s to %s\n' "$BINARY" "${bindir}/${BINARY}"

case ":$PATH:" in
    *":${bindir}:"*) ;;
    *) printf 'note: %s is not on $PATH\n' "$bindir" ;;
esac
