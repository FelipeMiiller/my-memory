/**
 * Electron smoke test — Playwright Electron API.
 *
 * Phase 1.5 will add Playwright to package.json. Until then, this file
 * exists for spec compliance (VE-10). To run manually:
 *   1. npm install --save-dev playwright
 *   2. node scripts/electron-smoke.test.mjs
 */
import { test, expect, _electron as electron } from 'playwright';

test('Electron app boots and shows tab Graf', async () => {
  const app = await electron.launch({ args: ['.'] });
  const window = await app.firstWindow();
  await window.waitForLoadState('domcontentloaded');
  const title = await window.title();
  expect(title).toContain('Grafo de Memória');
  await window.screenshot({ path: 'screenshot-qg-batch1.png' });
  await app.close();
});

test('monorepo scaffold: electron/ + viewer/ + package.json exist', async () => {
  const { existsSync } = await import('node:fs');
  const { resolve } = await import('node:path');
  const root = resolve(process.cwd());
  expect(existsSync(resolve(root, 'electron'))).toBe(true);
  expect(existsSync(resolve(root, 'viewer'))).toBe(true);
  expect(existsSync(resolve(root, 'package.json'))).toBe(true);
  expect(existsSync(resolve(root, 'viewer/src'))).toBe(true);
  expect(existsSync(resolve(root, 'viewer/index.html'))).toBe(true);
});
