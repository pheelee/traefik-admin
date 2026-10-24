import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page, baseURL }) => {
  // The UI, including icon fonts, must work without third-party network access.
  const origin = new URL(baseURL).origin;
  await page.route("**/*", route => {
    if (new URL(route.request().url()).origin === origin) return route.continue();
    return route.abort();
  });
});

test("manage proxy entries with Vue 3 under the application CSP", async ({ page, baseURL }) => {
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.addInitScript(() => {
    window.policyViolations = [];
    document.addEventListener("securitypolicyviolation", event => {
      window.policyViolations.push(event.violatedDirective);
    });
  });
  const response = await page.goto("/");
  expect(response.headers()["content-security-policy"]).toBe("script-src 'self'");
  await expect(page.locator("#emptyMsg")).toBeVisible();
  await expect(page.locator("footer")).not.toContainText("{{");

  await page.locator(".modal-trigger").click();
  const editor = page.locator("#editModal");
  await expect(editor).toBeVisible();
  await editor.getByRole("link", { name: "Create", exact: true }).click();
  await expect(editor.locator("#name")).toHaveClass(/invalid/);
  await editor.locator("#name").fill("BrowserTest");
  await editor.locator("#domain").fill("browser.example.com");
  await editor.locator("#backend0").fill(baseURL)
  await editor.locator("#backend1").fill(baseURL.replace("127.0.0.1", "localhost"));
  await editor.locator("label").filter({ hasText: "Forward http to https" }).click();
  await editor.locator("label").filter({ hasText: "SSL/TLS" }).click();
  await editor.getByRole("link", { name: "Headers", exact: true }).click();
  await editor.locator("#headername0").fill("X-Browser-Test");
  await editor.locator("#headervalue0").fill("Vue3");
  await editor.getByRole("link", { name: "Create", exact: true }).click();
  await expect(editor).toBeHidden();
  const cards = page.locator(".proxycfg");
  await expect(cards).toHaveCount(1);
  await expect(cards.first()).toContainText("browser.example.com");

  await cards.getByRole("link", { name: "Edit", exact: true }).click();
  await expect(editor.locator("#domain")).toHaveValue("browser.example.com");
  await expect(editor.locator("#backend0")).toHaveValue(baseURL);
  await expect(editor.locator("#backend1")).toHaveValue(baseURL.replace("127.0.0.1", "localhost"));
  await editor.locator("#domain").fill("updated.example.com");
  await editor.getByRole("link", { name: "Update", exact: true }).click();
  await expect(editor).toBeHidden();
  await expect(cards.first()).toContainText("updated.example.com");
  await page.reload();
  await expect(cards.first()).toContainText("updated.example.com");

  const filter = page.locator("#filterConfigs");
  await filter.fill("does-not-exist");
  await filter.press("Enter");
  await expect(cards).toHaveCount(0);
  await filter.fill("localhost");
  await filter.press("Enter");
  await expect(cards).toHaveCount(1);

  const configs = await page.request.get("/config/", { headers: {
    "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest",
  } });
  const [stored] = await configs.json();
  expect(stored.domain).toBe("updated.example.com");
  expect(stored.backends.filter(b => b.url)).toEqual([
    { url: baseURL, healthy: true },
    { url: baseURL.replace("127.0.0.1", "localhost"), healthy: true },
  ]);
  expect(stored.headers).toContainEqual({ Name: "X-Browser-Test", Value: "Vue3" });

  // Closing an unsaved edit must not mutate the displayed backend list.
  await cards.getByRole("link", { name: "Edit", exact: true }).click();
  await editor.locator("#backend0").fill("http://127.0.0.1:1234");
  await editor.getByRole("link", { name: "Close", exact: true }).click();
  await expect(cards.locator(".green-text").first()).toContainText(baseURL);

  // Remove one backend, then verify both persisted data and the card after reload.
  await cards.getByRole("link", { name: "Edit", exact: true }).click();
  await editor.locator("#backend1").fill("");
  await editor.getByRole("link", { name: "Update", exact: true }).click();
  await expect(editor).toBeHidden();
  await filter.fill("BrowserTest");
  await filter.press("Enter");
  await expect(cards).toHaveCount(1);
  await page.reload();
  await expect(cards.locator(".green-text")).toHaveCount(1);
  await cards.getByRole("link", { name: "Edit", exact: true }).click();
  await expect(editor.locator("#backend1")).toHaveValue("");
  await editor.getByRole("link", { name: "Close", exact: true }).click();

  await cards.getByRole("link", { name: "Remove", exact: true }).click();
  await page.locator("#confirmModal").getByRole("link", { name: "No", exact: true }).click();
  await expect(cards).toHaveCount(1);
  await expect(page.locator("#confirmModal")).toBeHidden();
  await cards.getByRole("link", { name: "Remove", exact: true }).click();
  await page.locator("#confirmModal").getByRole("link", { name: "Yes", exact: true }).click();
  await expect(cards).toHaveCount(0);
  await expect(page.locator("#emptyMsg")).toBeVisible();
  expect(errors).toEqual([]);
  expect(await page.evaluate(() => window.policyViolations)).toEqual([]);
});

test("notifications render untrusted markup as literal text", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("footer")).not.toContainText("{{");
  const sender = '<img src=x onerror="window.toastXss=true">';
  const message = '<svg onload="window.toastXss=true"></svg>';
  await page.evaluate(({ sender, message }) => {
    window.toastXss = false;
    Notify.Success(sender, message);
  }, { sender, message });
  await expect(page.locator(".toast").last()).toHaveText(`${sender} ${message}`);
  await expect(page.locator(".toast img, .toast svg")).toHaveCount(0);
  expect(await page.evaluate(() => window.toastXss)).toBe(false);
});

test("the editor fits mobile screens and resets after native dismissal", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const editor = page.locator("#editModal");
  await page.locator(".modal-trigger").click();
  await expect(editor).toBeVisible();
  const bounds = await editor.boundingBox();
  expect(bounds.x).toBeGreaterThanOrEqual(0);
  expect(bounds.x + bounds.width).toBeLessThanOrEqual(390);
  expect(bounds.y).toBeGreaterThanOrEqual(0);
  expect(bounds.y + bounds.height).toBeLessThanOrEqual(844);
  await expect(editor.getByRole("link", { name: "Create", exact: true })).toBeInViewport();
  for (const name of ["General", "Auth", "Allowed IP", "Headers"]) {
    await expect(editor.getByRole("link", { name, exact: true })).toBeInViewport({ ratio: 1 });
  }
  await editor.locator("#name").fill("DiscardMe");
  await page.keyboard.press("Escape");
  await expect(editor).toBeHidden();
  await page.locator(".modal-trigger").click();
  await expect(editor.locator("#name")).toHaveValue("");
  await page.mouse.click(2, 2);
  await expect(editor).toBeHidden();
});

test("icons load locally and render glyphs rather than their names", async ({ page }) => {
  const fontResponse = page.waitForResponse(response => response.url().endsWith("material-icons-latin-400-normal.woff2"));
  await page.goto("/");
  expect((await fontResponse).status()).toBe(200);
  const widths = await page.evaluate(async () => {
    const faces = await document.fonts.load('24px "Material Icons"', "search");
    if (!faces.length || faces.some(face => face.status !== "loaded")) {
      throw new Error("The bundled Material Icons font did not load");
    }
    const context = document.createElement("canvas").getContext("2d");
    context.font = '24px "Material Icons"';
    return ["search", "link", "arrow_upward", "arrow_downward", "format_list_numbered"]
      .map(name => ({ name, width: context.measureText(name).width }));
  });
  for (const { width } of widths) expect(width).toBeCloseTo(24, 1);
  await expect(page.locator("nav .material-icons")).toHaveCSS("font-family", '"Material Icons"');
});
