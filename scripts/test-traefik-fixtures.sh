#!/usr/bin/env bash
# Exercise the YAML fixtures with the Traefik version tracked by Renovate.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

if [[ -n "${TRAEFIK_BINARY:-}" ]]; then
  exec go test -count=1 -run '^TestTraefikFixtures$' -v ./config
fi

version=$(sed -n 's/^ARG TRAEFIK_VERSION=//p' Dockerfile)
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo 'Expected one semantic TRAEFIK_VERSION in Dockerfile.' >&2
  exit 1
fi

case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo 'Fixture runner supports Linux amd64 and arm64.' >&2; exit 1 ;;
esac
test "$(uname -s)" = Linux
directory=$(mktemp -d)
trap 'rm -rf "$directory"' EXIT
archive="traefik_v${version}_linux_${arch}.tar.gz"
release="https://github.com/traefik/traefik/releases/download/v${version}"
curl --fail --silent --show-error --location --retry 3 --output "$directory/$archive" "$release/$archive"
curl --fail --silent --show-error --location --retry 3 --output "$directory/checksums.txt" "$release/traefik_v${version}_checksums.txt"
awk -v file="$archive" '$2 == file { print }' "$directory/checksums.txt" > "$directory/expected.txt"
test "$(wc -l < "$directory/expected.txt")" -eq 1
(cd "$directory" && sha256sum --check expected.txt)
tar -xzf "$directory/$archive" -C "$directory" traefik
TRAEFIK_BINARY="$directory/traefik" go test -count=1 -run '^TestTraefikFixtures$' -v ./config
