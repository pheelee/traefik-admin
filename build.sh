#!/bin/bash
set -e
set -o pipefail

if [[ $# -eq 0 ]]; then
    AMD64=yes
    ARM64=yes
    ARMv7=yes
fi

while [[ $# -gt 0 ]]
do
key="$1"
case $key in
    --amd64)
    AMD64=yes
    shift 
    ;;
    --arm64)
    ARM64=yes
    shift 
    ;;
    --armv7)
    ARMv7=yes
    shift
    ;;
    --all)
    AMD64=yes
    ARM64=yes
    ARMv7=yes
    shift
    ;;
esac
done

ROOT=$(dirname "$(readlink -f "$0")")
mkdir -p $ROOT/dist

if [[ $AMD64 = "yes" ]]; then
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags "-X github.com/pheelee/traefik-admin/internal/server.VERSION=`git describe --tags`" -o traefik-admin ./cmd/traefik-admin
    tar cfz $ROOT/dist/traefik-admin-linux-amd64.tar.gz traefik-admin
    rm traefik-admin
fi

if [[ $ARM64 = "yes" ]]; then
    CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -ldflags "-X github.com/pheelee/traefik-admin/internal/server.VERSION=`git describe --tags`" -o traefik-admin ./cmd/traefik-admin
    tar cfz $ROOT/dist/traefik-admin-linux-arm64.tar.gz traefik-admin
    rm traefik-admin
fi

if [[ $ARMv7 = "yes" ]]; then
    CGO_ENABLED=0 GOARCH=arm GOARM=7 GOOS=linux go build -a -ldflags "-X github.com/pheelee/traefik-admin/internal/server.VERSION=`git describe --tags`" -o traefik-admin ./cmd/traefik-admin
    tar cfz $ROOT/dist/traefik-admin-linux-armv7.tar.gz traefik-admin
    rm traefik-admin
fi


