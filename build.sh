#!/bin/bash
set -e
set -o pipefail

ROOT=$(dirname "$(readlink -f "$0")")
mkdir -p $ROOT/dist

# Prepare Webroot
rm -rf "$ROOT/webroot"
cp -r "$ROOT/webrootSrc" "$ROOT/webroot"
JS=($(sha1sum webroot/js/traefik-admin.js))
CSS=($(sha1sum webroot/css/traefik-admin.css))
sed -i "s/traefik-admin\.css/traefik-admin\.css?v=${CSS:0:8}/g" webroot/index.html
sed -i "s/traefik-admin\.js/traefik-admin\.js?v=${JS:0:8}/g" webroot/index.html
exit 0
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin
tar cfz $ROOT/dist/traefik-admin-linux-amd64.tar.gz webroot traefik-admin
rm traefik-admin
CGO_ENABLED=0 GOARCH=arm64 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin
tar cfz $ROOT/dist/traefik-admin-linux-arm64.tar.gz webroot traefik-admin
rm traefik-admin
CGO_ENABLED=0 GOARCH=arm GOARM=7 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o traefik-admin
tar cfz $ROOT/dist/traefik-admin-linux-armv7.tar.gz webroot traefik-admin
rm traefik-admin

