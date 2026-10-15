import { spawn } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

if (!process.env.TRAEFIK_ADMIN_BINARY) {
  throw new Error("Set TRAEFIK_ADMIN_BINARY to the compiled admin binary");
}
const directory = mkdtempSync(join(tmpdir(), "traefik-admin-browser-"));
const server = spawn(resolve(process.env.TRAEFIK_ADMIN_BINARY), [
  "--ConfigPath", directory,
  "--CookieSecret", randomBytes(32).toString("hex"),
  "--Port", process.env.TRAEFIK_ADMIN_TEST_PORT || "18099",
], { stdio: "inherit" });
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.once(signal, () => server.kill("SIGTERM"));
}
server.once("error", (error) => {
  console.error(error);
  rmSync(directory, { recursive: true, force: true });
  process.exit(1);
});
server.once("exit", (code) => {
  rmSync(directory, { recursive: true, force: true });
  process.exit(code ?? 0);
});
