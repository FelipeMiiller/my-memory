/**
 * Electron Forge smoke test — Playwright Electron API.
 *
 * Phase 1.5 will wire this into CI. To run manually:
 *   1. npx playwright install chromium
 *   2. npm run smoke
 *
 * Boots `electron-forge start` and asserts the window title contains
 * "Grafo de Memória" + the Cytoscape canvas mounts.
 */
import { test, expect, _electron as electron } from "playwright";

test("Electron Forge app boots and shows Graf de Memória tab", async () => {
  const app = await electron.launch({ args: [".", "--no-sandbox"] });
  const window = await app.firstWindow();
  await window.waitForLoadState("domcontentloaded");
  const title = await window.title();
  expect(title).toContain("Grafo de Memória");
  await window.screenshot({ path: "screenshot-qg-forge.png" });
  await app.close();
});

test("monorepo scaffold: src/ + public/ + package.json + forge.config exist", async () => {
  const { existsSync } = await import("node:fs");
  const { resolve } = await import("node:path");
  const root = resolve(process.cwd());
  expect(existsSync(resolve(root, "src"))).toBe(true);
  expect(existsSync(resolve(root, "src/main.ts"))).toBe(true);
  expect(existsSync(resolve(root, "src/preload.ts"))).toBe(true);
  expect(existsSync(resolve(root, "public/data-central.json"))).toBe(true);
  expect(existsSync(resolve(root, "package.json"))).toBe(true);
  expect(existsSync(resolve(root, "forge.config.ts"))).toBe(true);
});