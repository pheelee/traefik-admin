# Traefik-Admin

![Go Version](https://img.shields.io/github/go-mod/go-version/pheelee/traefik-admin)
![Version](https://img.shields.io/github/v/tag/pheelee/traefik-admin?color=green&label=Version)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/pheelee/traefik-admin/total)


This is a small webinterface to manage dynamic configs for traefik.

## Development checks

Use the Go version declared in `go.mod`. CI reads that file directly; the Docker
builder uses the same version. From the repository root, run:

```bash
go mod download
go mod verify
go test -mod=readonly -count=1 ./...
go build -mod=readonly -o /tmp/traefik-admin ./cmd/traefik-admin
python3 scripts/smoke-test.py /tmp/traefik-admin
```

The smoke check requires Python 3 and starts its own server on a temporary port,
using a temporary configuration directory. It checks the UI, static assets,
request validation, configuration create/read/update/delete, and backend health.
It requires no Home Assistant instance, external backend, or credentials.

Pull requests also cross-compile Linux binaries for amd64, arm64, and ARMv7.
These are compilation checks; only the native binary is exercised by the smoke
check. Configure branch protection to require the CI test and build checks.

## Dependency updates

Renovate owns dependency update PRs for this repository. Its Dependency Dashboard
tracks Go modules (including indirect modules), the Go builder image, GitHub
Actions, and the Traefik release still used by the Dockerfile. Go directive and
builder-image updates are grouped as **Go toolchain**. Major updates require
approval in the dashboard; all PRs require manual merging.

Enable the Renovate GitHub App for this repository if it is not already enabled.
The dashboard is created by the app after this configuration reaches the default
branch. Dependabot version updates are disabled to avoid duplicate PRs; GitHub
security alerts can remain enabled independently.

Downloaded versions use an annotation above their Dockerfile argument:

```dockerfile
# renovate: datasource=github-releases depName=traefik/traefik
ARG TRAEFIK_VERSION=2.11.8
```

Vendored frontend libraries do not yet have a package manifest. The externally
supplied add-on base image and unpinned Alpine packages also need explicit version
declarations before Renovate can update them. Frontend dependency tracking and
consolidation of Home Assistant packaging into `hassio-addon-traefik` are separate
follow-up changes.

## Build on Homeassistant for testing
The following section describes the build process for the Home Assistant Operating System installation. For other installation methods please adopt accordingly.

Clone this repo to the `/addons` folder on your homeassistant instance (e.g using the SSH & Web Terminal Addon). Then edit the `config.json` and remove the line

`"image": "pheelee/hassio-addon-traefik-{arch}",`

In the addon store click the three dots (upper right corner) and select **Reload**. After that the addon should be visible in the **Local add-ons** section and be installable like any other addon.

## Build and publish to registry locally

```bash
docker run --rm --privileged -v ~/.docker:/root/.docker -v `pwd`:/data -v /var/run/docker.sock:/var/run/docker.sock homeassistant/amd64-builder --all -t /data
```
for more information about the homeassistant builder please visit https://github.com/home-assistant/builder

## Build using Github Actions
see `.github/workflows/release.yml` in this repo for inspiration

# MIT License
Copyright 2021-2024 Philipp Ritter

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
