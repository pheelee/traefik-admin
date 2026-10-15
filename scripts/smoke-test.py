#!/usr/bin/env python3
"""Exercise a compiled admin binary using temporary configuration and local HTTP."""

import argparse
import json
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener


def smoke_test(binary, expected_version=None):
    # Ignore HTTP proxy settings for this local-only check.
    client = build_opener(ProxyHandler({}))
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]

    with tempfile.TemporaryDirectory(prefix="traefik-admin-smoke-") as directory:
        root = Path(directory)
        configs = root / "dynamic.d"
        configs.mkdir()
        with (root / "server.log").open("w+") as log:
            process = subprocess.Popen(
                [str(binary), "--ConfigPath", str(configs), "--Port", str(port),
                 "--CookieSecret", secrets.token_hex(32)],
                stdout=log, stderr=subprocess.STDOUT,
            )

            def request(path, method="GET", body=None, expected_status=200, ajax=True):
                headers = {"Content-Type": "application/json"}
                if ajax:
                    headers["X-Requested-With"] = "XMLHttpRequest"
                req = Request(
                    f"http://127.0.0.1:{port}{path}", method=method,
                    headers=headers, data=json.dumps(body).encode() if body is not None else None,
                )
                try:
                    response = client.open(req, timeout=3)
                except HTTPError as error:
                    response = error
                with response:
                    if response.status != expected_status:
                        raise AssertionError(
                            f"{method} {path}: expected {expected_status}, got "
                            f"{response.status}: {response.read().decode()}"
                        )
                    return response.read()

            try:
                deadline = time.monotonic() + 15
                while True:
                    if process.poll() is not None:
                        raise RuntimeError(f"Admin exited with status {process.returncode}")
                    try:
                        features = json.loads(request("/features"))
                        break
                    except (URLError, TimeoutError):
                        if time.monotonic() >= deadline:
                            raise TimeoutError("Admin did not become ready within 15 seconds")
                        time.sleep(0.1)

                assert features["version"]
                if expected_version is not None:
                    assert features["version"] == expected_version, features
                assert features["forwardauth"]["enabled"] is False
                assert b"traefik" in request("/").lower()
                assert b"Vue" in request("/static/js/vue.min.js")
                assert b"Materialize" in request("/static/js/materialize.min.js")
                assert b"Materialize" in request("/static/css/vendor/materialize.min.css")
                assert json.loads(request("/config/")) == []
                request("/config/", expected_status=400, ajax=False)

                entry = {
                    "name": "SmokeTest", "domain": "smoke.example.com",
                    "backend": {"url": f"http://127.0.0.1:{port}"},
                    "https": False, "forcetls": False,
                }
                invalid = dict(entry, name="!")
                assert json.loads(request("/config/new", "POST", invalid, 400))["valid"] is False
                created = json.loads(request("/config/new", "POST", entry))
                config_id = created["id"]
                assert created["backend"]["healthy"] is True
                assert (configs / f"{config_id}.yaml").is_file()
                assert json.loads(request("/config/"))[0]["id"] == config_id
                stored = json.loads(request(f"/config/{config_id}"))
                assert stored["HTTP"]["Routers"][config_id + "-http"]["Rule"] == "Host(`smoke.example.com`)"

                updated = json.loads(request(
                    f"/config/{config_id}", "PUT", dict(entry, id=config_id, domain="updated.example.com")
                ))
                assert updated["domain"] == "updated.example.com"
                request(f"/config/{updated['id']}", "DELETE")
                assert json.loads(request("/config/")) == []
                assert not list(configs.glob("SmokeTest_*.yaml"))
                request(f"/config/{updated['id']}", expected_status=404)
                print("PASS: UI, assets, request validation, config CRUD, and backend health")
            except Exception:
                log.flush()
                log.seek(0)
                print(log.read(), file=sys.stderr)
                raise
            finally:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("--expected-version")
    args = parser.parse_args()
    smoke_test(args.binary.resolve(), args.expected_version)
