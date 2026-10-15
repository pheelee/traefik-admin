import { defineConfig } from "@playwright/test";

const port = process.env.TRAEFIK_ADMIN_TEST_PORT || "18099";
export default defineConfig({
  testDir: "./tests/browser",
  workers: 1,
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    headless: true,
    launchOptions: process.env.CHROMIUM_PATH
      ? { executablePath: process.env.CHROMIUM_PATH }
      : {},
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "node scripts/browser-server.mjs",
    url: `http://127.0.0.1:${port}/features`,
    reuseExistingServer: false,
    timeout: 15000,
  },
});
