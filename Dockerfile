ARG BUILD_FROM
FROM golang:1.21-alpine as builder
ARG BUILD_ARCH

COPY . /go/src/traefik-admin

RUN apk add --update git

RUN set -ex; \
        case "$BUILD_ARCH" in \
                armhf) GOARCH='arm'; GOARM=6 ;; \
                armv7) GOARCH='arm'; GOARM=7 ;; \
                aarch64) GOARCH='arm64' ;; \
                amd64) GOARCH='amd64' ;; \
                *) echo >&2 "error: unsupported architecture: $BUILD_ARCH"; exit 1 ;; \
        esac; \
        cd /go/src/traefik-admin; \
        CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH GOARM=$GOARM go build -a -ldflags "-X github.com/pheelee/traefik-admin/internal/server.VERSION=`git describe --tags`" -o dist/traefik-admin ./cmd/traefik-admin

FROM $BUILD_FROM
ARG TRAEFIK_VERSION=2.11.8

LABEL org.opencontainers.image.source=https://github.com/pheelee/traefik-admin
LABEL org.opencontainers.image.licenses=MIT
LABEL org.opencontainers.image.description="Reverse Proxy for Web Services in a Hass.io Addon"

EXPOSE 80
EXPOSE 443

ENV LANG C.UTF-8

# Get Traefik Release
RUN apk --no-cache add ca-certificates tzdata
RUN set -ex; \
        apkArch="$(apk --print-arch)"; \
        case "$apkArch" in \
                armhf) arch='armv6' ;; \
                armv7) arch='armv7' ;; \
                aarch64) arch='arm64' ;; \
                x86_64) arch='amd64' ;; \
                *) echo >&2 "error: unsupported architecture: $apkArch"; exit 1 ;; \
        esac; \
        wget --quiet -O /tmp/traefik.tar.gz "https://github.com/containous/traefik/releases/download/v${TRAEFIK_VERSION}/traefik_v${TRAEFIK_VERSION}_linux_$arch.tar.gz"; \
        tar xzvf /tmp/traefik.tar.gz -C /usr/local/bin traefik; \
        rm -f /tmp/traefik.tar.gz; \
        chmod +x /usr/local/bin/traefik

WORKDIR /data

COPY rootfs /
#COPY dist/traefik-admin-linux-$ARCH /web/traefik-admin
COPY --from=builder /go/src/traefik-admin/dist/traefik-admin /web/traefik-admin
