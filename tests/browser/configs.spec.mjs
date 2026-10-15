import { test, expect } from "@playwright/test";

test("manage proxy entries with Vue 3 under the application CSP", async ({ page, baseURL }) => {
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  // Font availability is unrelated to this local workflow.
  await page.route("https://fonts.googleapis.com/**", route => route.abort());
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
  await editor.locator("#backend").fill(baseURL);
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
  await filter.fill("BrowserTest");
  await filter.press("Enter");
  await expect(cards).toHaveCount(1);

  const configs = await page.request.get("/config/", { headers: {
    "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest",
  } });
  const [stored] = await configs.json();
  expect(stored.domain).toBe("updated.example.com");
  expect(stored.backend.healthy).toBe(true);
  expect(stored.headers).toContainEqual({ Name: "X-Browser-Test", Value: "Vue3" });

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
