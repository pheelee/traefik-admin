# Configuration regression fixtures

These files are synthetic test data. Keep identifiers, domains and expected
behavior stable so the same cases can check parser migrations and Traefik bumps.
Runtime backends are loopback placeholders. The Basic Auth user `test` and password
`password` are public fixture credentials, not production secrets.

| Directory/file                            | Purpose                                                                                                 |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `traefik/Plain_10000001.yaml`             | Plain HTTP routing and backend/path preservation                                                        |
| `traefik/Headers_20000002.yaml`           | Request headers containing YAML-looking strings and Unicode                                             |
| `traefik/Secure_30000003.yaml`            | HTTP-to-HTTPS redirect, TLS, HSTS and Basic Auth                                                        |
| `traefik/Forward_40000004.yaml`           | Forward Auth allow/deny behavior                                                                        |
| `traefik/sys_middlewares.yaml`            | Shared middleware used by those routes                                                                  |
| `traefik/cases.json`                      | Expected domains, feature flags and UI interpretation                                                   |
| `yaml/scalars-and-aliases.yaml`           | Scalar strings, anchors, sequence aliases and merge keys                                                |
| `invalid/`                                | Duplicate keys, malformed syntax and incorrect value types; all must fail                               |
| `compatibility.yaml` and `../mock/*.yaml` | Existing config/parser compatibility, including legacy IP restrictions and a named certificate resolver |

The ordinary Go suite loads and saves all valid fixtures, compares their meaning,
and checks the route fixtures against `cases.json` through `ToUserInput`.
Invalid fixtures must return an error identifying the file, remain unloaded,
and leave the existing in-memory config unchanged. Duplicate keys are rejected
by YAML v3; existing files with duplicate keys must be corrected before loading.
Saved YAML may be reformatted by the new encoder.

## Check a Traefik release

From the repository root:

```sh
bash scripts/test-traefik-fixtures.sh
```

The runner reads `TRAEFIK_VERSION` from `Dockerfile`, downloads the release for
Linux amd64/arm64, verifies the exact archive against the publisher's SHA-256
manifest, and runs `TestTraefikFixtures`. CI runs it on every PR, including
Renovate's Traefik version bumps. No Docker daemon or Python packages are needed.

To check an already downloaded binary, including a proposed upgrade:

```sh
TRAEFIK_BINARY=/absolute/path/to/traefik bash scripts/test-traefik-fixtures.sh
```

The test loads the `traefik/` fixtures with the admin parser, rewrites only backend
and auth addresses to temporary local mock servers, saves them with the admin
encoder, and gives those files to the real Traefik file provider. It exercises
HTTP routing, exact path/query preservation, header values, HTTPS redirects,
TLS with Traefik's generated test certificate, HSTS, Basic Auth denial/acceptance
and credential removal, and Forward Auth denial/acceptance. Listeners bind to
loopback and processes/files are removed after the test.

The generic Go suite skips real-engine checks when `TRAEFIK_BINARY` is unset;
the dedicated CI step downloads the pinned release and requires those checks.
Named ACME resolvers, real certificate issuance and legacy IP middleware are
covered only by parser fixtures here, not by the real-engine test.

When adding a runtime case, use an admin-style `Name_hash.yaml` filename and
matching router/service IDs, add it to `traefik/cases.json`, and extend the runtime
assertions for its expected behavior. Keep invalid cases in `invalid/` so they
cannot be loaded by the real-engine runner.
