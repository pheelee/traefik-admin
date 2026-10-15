# Admin releases and the add-on interface

`traefik-admin` owns the Go application, frontend assets, CLI, and published
binaries. `hassio-addon-traefik` owns the deployed Traefik version, Home Assistant
options, service supervision, and selection of an admin release.

The beta add-on's `Dockerfile`, `config.yaml`, and `rootfs` are retained here until
the add-on repository has a replacement beta installation path. Do not remove
them as part of the binary release transition.

## Build and validate locally

From the repository root on Linux with Go, GNU tar, gzip, and sha256sum installed:

```bash
VERSION=v0.5.3 DIST_DIR=/tmp/admin-release ./build.sh --all
(cd /tmp/admin-release && sha256sum -c SHA256SUMS)
mkdir -p /tmp/admin-release-smoke
tar -xzf /tmp/admin-release/traefik-admin-linux-amd64.tar.gz -C /tmp/admin-release-smoke
python3 scripts/smoke-test.py /tmp/admin-release-smoke/traefik-admin --expected-version v0.5.3
```

The smoke command runs on an amd64 Linux machine. ARM artifacts are cross-compiled,
not executed. The script also accepts `--amd64`, `--arm64`, and `--armv7` individually
or together. Unknown flags fail. Defaults are all architectures, `dist/`, and a
version from `git describe --tags --always --dirty`. Explicit `VERSION` is used for
releases; every binary exposes it through `GET /features`.

The build script works from any directory and leaves the source tree untouched.
Archives contain one executable named `traefik-admin`. Archive timestamps and
ownership are normalized and gzip timestamps omitted; binaries omit local source
paths and VCS metadata. `SHA256SUMS` lists only the selected artifacts, even if the
output directory contains files from an earlier build. Use an empty output
directory for release preparation.

## Publish a release

Merge the reviewed changes into `main`, then push a new tag such as `v0.5.3` or
`v0.5.3-beta.1`. The tag must point to a commit reachable from `main`. The release
workflow validates frontend assets, dependencies, and tests, builds all three
architectures, and smoke-tests the packaged amd64 binary and embedded version.
Only then does it upload a draft release and publish it. Tags with a suffix are
marked as prereleases.

The workflow can also be dispatched with an existing release tag. Do not create
the release manually first: an existing release causes publication to fail,
without overwriting assets. If upload or publication fails after draft creation,
inspect and remove only the incomplete unpublished draft before retrying. Never
replace assets of a published version; publish a new tag instead. No release is
created by ordinary PR or branch CI.

## Consume a release in the add-on

Pin an explicit release tag and download its archive and `SHA256SUMS` from that
same release. Verify the selected archive before extraction; a missing checksum,
failed download, or checksum mismatch must fail the image build. Historical
releases without a checksum manifest need a version upgrade before this policy
can be enabled. Checksums detect corruption; they are not a signature or an
independent attestation of the publisher.

| Home Assistant architecture | Release asset |
| --- | --- |
| `amd64` | `traefik-admin-linux-amd64.tar.gz` |
| `aarch64` | `traefik-admin-linux-arm64.tar.gz` |
| `armv7` | `traefik-admin-linux-armv7.tar.gz` |

Asset names preserve the existing add-on download URLs. The GitHub release tag
provides the version; never use a moving `latest` download URL. ARMv6/`armhf` is not
part of the supported release matrix.

## Runtime contract

- `--ConfigPath`: required writable directory for generated dynamic YAML. Startup
  can migrate existing files; validate upgrades against a copy of production data.
- `--CookieSecret`: required session secret supplied by the deployment.
- `--Port`: HTTP listener, default `8099`.
- `--CertResolver`: Traefik resolver name, default `http01`; the add-on must define
  that resolver if TLS entries use it.
- `--AuthEndpoint`: optional authorization endpoint; omitting it disables forward
  authentication. `--WebRoot` overrides embedded assets for development.
- `GET /features` exposes the build version and forward-auth configuration. The
  configuration API under `/config/` requires JSON content type and the
  `X-Requested-With: XMLHttpRequest` header.

Generated files use Traefik v2 configuration conventions, including the `web` and
`websecure` entrypoints, file-provider middlewares, and the selected certificate
resolver. Compatibility with every Traefik v2 release is not guaranteed by this
document. The add-on must validate its chosen Traefik/admin pair with a functional
routing check. A Traefik major upgrade, especially middleware/schema changes,
requires separate compatibility testing before deployment.
