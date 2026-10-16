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
go mod tidy -diff
go test -mod=readonly -count=1 ./...
go build -mod=readonly -o /tmp/traefik-admin ./cmd/traefik-admin
python3 scripts/smoke-test.py /tmp/traefik-admin
```

The smoke check requires Python 3 and starts its own server on a temporary port,
using a temporary configuration directory. It checks the UI, static assets,
request validation, configuration create/read/update/delete, and backend health.
It requires no Home Assistant instance, external backend, or credentials.

CI and release checks require `go mod tidy -diff` to succeed, catching unused
dependencies and incomplete module-path upgrades. YAML tests load existing
configuration fixtures, check supported fields, and save/reload them without
changing their meaning. They also cover malformed input and document the current
parser's last-value-wins handling of duplicate keys; rejecting duplicates in a
future parser migration must be an explicit compatibility decision.

IndieAuth tests use a local authorization server to verify login redirects,
session creation, independent login attempts, and rejection of missing, unknown
or reused states. They also check that failed verification responses create no
session.

Pull requests also cross-compile Linux binaries for amd64, arm64, and ARMv7.
These are compilation checks; only the native binary is exercised by the smoke
check. Configure branch protection to require the CI test and build checks.

## Dependency updates

Renovate owns dependency update PRs for this repository. Its Dependency Dashboard
tracks Go modules (including indirect modules), the Go builder image, GitHub
Actions, frontend packages in `package.json`, and the Traefik release still used
by the Dockerfile. Go directive and builder-image updates are grouped as **Go
toolchain**. Major updates require
approval in the dashboard; all PRs require manual merging.

The `Renovate` workflow runs daily and supports manual runs from GitHub Actions.
Follow [the runner setup](.github/RENOVATE.md) to configure the GitHub App credentials and
disable the hosted Renovate App for this repository before the first run.
Dependabot version updates are disabled to avoid duplicate PRs; GitHub security
alerts can remain enabled independently.

Downloaded versions use an annotation above their Dockerfile argument:

```dockerfile
# renovate: datasource=github-releases depName=traefik/traefik
ARG TRAEFIK_VERSION=2.11.8
```

The externally supplied add-on base image and unpinned Alpine packages still need
explicit version declarations before Renovate can update them. Consolidation of
Home Assistant packaging into `hassio-addon-traefik` is a separate follow-up change.

## Frontend dependencies

Vue and Materialize are pinned in `package.json` and `package-lock.json` and tracked
by Renovate. Node 24 and npm are needed to refresh or verify the vendored files:

```bash
npm ci --ignore-scripts --no-audit --no-fund
npm run vendor
npm run vendor:check
```

The vendor script copies the Vue 3 production runtime and Materialize JS/CSS,
including license headers, into `internal/server/webrootSrc`.
It also compiles the
`#app` template in `index.html` into `js/app-render.js`. This lets the UI run under
the existing `script-src 'self'` policy without runtime template compilation or
`unsafe-eval`. Keep the end-of-app marker in place and run the vendor script after
editing the template, package manifest, or lockfile. Commit the generated files.
Go embeds these tracked assets, so Go-only development and Docker builds do not
need Node or `node_modules`. Renovate groups Vue and its compiler updates together.

Material Icons are bundled from the pinned `@fontsource/material-icons` package,
including the font license. The browser loads their CSS and font files from the
app itself, so icons work without access to Google Fonts or another CDN. Browser
checks block third-party requests and verify all five icon ligatures used by the UI.

CI and releases check the assets against a clean npm installation. After a
Renovate frontend update, run the commands above on its branch and commit the
regenerated files before merging. Renovate is not configured to execute the vendor
script itself. Asset verification is part of the existing required **Tests and API
smoke check** job, so no branch-protection changes are needed.

Browser checks use Playwright Chromium against a compiled binary with temporary
configuration. They cover create/update/delete, validation, filtering, custom
headers, and execution under the production content security policy:

```bash
go build -mod=readonly -o /tmp/traefik-admin ./cmd/traefik-admin
npx playwright install --with-deps chromium
TRAEFIK_ADMIN_BINARY=/tmp/traefik-admin npm run test:browser
```

For an existing local Chromium installation, set `CHROMIUM_PATH` to its executable
instead of downloading a browser. `TRAEFIK_ADMIN_TEST_PORT` overrides the test port
18099. CI and releases run these browser checks too, including literal-text
notification handling and mobile dialog dismissal.

Materialize comes from the maintained `@materializecss/materialize` package. The
patched 1.2.x line preserves the classic appearance; Renovate is limited to that
major version so a redesign requires a separate migration. The unpatched
`materialize-css` 1.0.0 package is no longer used. Dialogs use the native
HTML `dialog` API, and notifications use the text-only Toast API rather than
interpolating HTML. The UI requires a browser with native dialog support.

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
Push a new version tag after merging to `main` to build and publish validated
binary archives and their SHA-256 manifest. See [release instructions and the
add-on runtime contract](docs/releases.md) for commands, architecture names, and
compatibility requirements. The beta Home Assistant packaging remains here until
a replacement is available in the add-on repository.

# MIT License
Copyright 2021-2024 Philipp Ritter

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
