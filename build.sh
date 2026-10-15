#!/bin/bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
architectures=()
for argument in "$@"; do
    case "$argument" in
        --amd64) architectures+=(amd64) ;;
        --arm64) architectures+=(arm64) ;;
        --armv7) architectures+=(armv7) ;;
        --all) architectures=(amd64 arm64 armv7) ;;
        *) echo "Usage: $0 [--amd64] [--arm64] [--armv7] [--all]" >&2; exit 2 ;;
    esac
done
if (( ${#architectures[@]} == 0 )); then
    architectures=(amd64 arm64 armv7)
fi

VERSION=${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty)}
if [[ ! "$VERSION" =~ ^[a-zA-Z0-9][a-zA-Z0-9._+-]*$ ]]; then
    echo "VERSION must contain only letters, digits, dots, underscores, pluses, or hyphens" >&2
    exit 2
fi
DIST_DIR=${DIST_DIR:-$ROOT/dist}
mkdir -p "$DIST_DIR"
DIST_DIR=$(cd "$DIST_DIR" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
archives=()
cd "$ROOT"
for architecture in "${architectures[@]}"; do
    archive="traefik-admin-linux-${architecture}.tar.gz"
    # Repeated flags should not produce duplicate manifest entries.
    if [[ " ${archives[*]} " == *" $archive "* ]]; then continue; fi
    goarch=$architecture
    goarm=
    if [[ "$architecture" == armv7 ]]; then goarch=arm; goarm=7; fi
    CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" GOARM="$goarm" \
        go build -mod=readonly -trimpath -buildvcs=false \
        -ldflags "-X github.com/pheelee/traefik-admin/internal/server.VERSION=$VERSION" \
        -o "$work/traefik-admin" ./cmd/traefik-admin
    tar --sort=name --mtime=@0 --owner=0 --group=0 --numeric-owner \
        -cf - -C "$work" traefik-admin | gzip -n > "$DIST_DIR/$archive"
    archives+=("$archive")
done
cd "$DIST_DIR"
sha256sum "${archives[@]}" > SHA256SUMS
sha256sum -c SHA256SUMS
