/**
 * Viewer Vite dev smoke test — placeholder for Phase 1.5.
 *
 * Boots Vite dev server, fetches / and asserts HTML contains Graf de Memória.
 * Will be expanded in Phase 2 to assert Cytoscape canvas + data-central.json fetch.
 *
 * RUN: `node --check scripts/viewer-smoke.test.mjs`  (syntax only)
 *      `npm install --save-dev @playwright/test`   (Phase 1.5 — not in Phase 1)
 *      `npx playwright install chromium`          (Phase 1.5)
 *      `npm run smoke`                             (Phase 1.5)
 *
 * The @playwright/test import resolves only when Playwright is installed.
 * Per the worker's scope rules for Phase 1, we deliberately DO NOT install
 * Playwright (300 MB download) — this file is committed as future scaffolding.
 */
import { test, expect } from '@playwright/test';

test('Vite dev server serves viewer/index.html with title', async ({ page }) => {
  await page.goto('http://localhost:5173');
  await expect(page).toHaveTitle(/Grafo de Mem\u00f3ria/);
  await expect(page.locator('#root')).toBeAttached();
});

test('viewer/index.html declares strict CSP meta tag', async ({ page }) => {
  await page.goto('http://localhost:5173');
  const csp = await page.locator('meta[http-equiv="Content-Security-Policy"]').getAttribute('content');
  expect(csp).toBeTruthy();
  expect(csp).toContain("default-src 'self'");
  expect(csp).toContain("object-src 'none'");
  expect(csp).toContain("frame-src 'none'");
});
