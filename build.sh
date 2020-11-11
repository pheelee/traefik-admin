#!/bin/bash
set -e
set -o pipefail

ROOT=`pwd`
mkdir -p $ROOT/dist

CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin-linux-amd64
CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin-linux-arm64
CGO_ENABLED=0 GOARCH=arm GOARM=7 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin-linux-armv7

tar cfz $ROOT/dist/traefik-admin-linux-amd64.tar.gz webroot traefik-admin-linux-amd64
tar cfz $ROOT/dist/traefik-admin-linux-arm64.tar.gz webroot traefik-admin-linux-arm64
tar cfz $ROOT/dist/traefik-admin-linux-armv7.tar.gz webroot traefik-admin-linux-armv7

rm traefik-admin-linux-amd64
rm traefik-admin-linux-arm64
rm traefik-admin-linux-armv7