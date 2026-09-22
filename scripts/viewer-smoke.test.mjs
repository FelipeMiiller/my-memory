/**
 * Viewer Vite dev smoke test — Playwright.
 *
 * Boots Vite dev server (via electron-forge start), fetches / and asserts HTML
 * contains Graf de Memória + CSP meta tag.
 *
 * RUN: `npx playwright install chromium`
 *      `npm run test:e2e`
 */
import { test, expect } from "@playwright/test";

test("Electron Forge renderer loads Graf de Memória title", async ({ page }) => {
  await page.goto("http://localhost:5173");
  await expect(page).toHaveTitle(/Grafo de Memória/);
  await expect(page.locator("#app")).toBeAttached();
});

test("index.html declares strict CSP meta tag", async ({ page }) => {
  await page.goto("http://localhost:5173");
  const csp = await page
    .locator('meta[http-equiv="Content-Security-Policy"]')
    .getAttribute("content");
  expect(csp).toBeTruthy();
  expect(csp).toContain("default-src 'self'");
  expect(csp).toContain("object-src 'none'");
  expect(csp).toContain("frame-src 'none'");
});