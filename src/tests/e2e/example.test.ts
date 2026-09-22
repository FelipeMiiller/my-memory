import {
  type ElectronApplication,
  _electron as electron,
  expect,
  type Page,
  test,
} from "@playwright/test";
import { existsSync } from "node:fs";
import path from "node:path";

/**
 * E2E smoke — my-memory viewer (Electron Forge).
 *
 * Phase 1.5 (smoke gate, runs against the built `.vite/` artifacts):
 *   1. Asserts `.vite/build/main.js` exists (produced by `electron-forge package`
 *      via the VitePlugin; the plugin builds the renderer + main + preload even
 *      when no makers are configured for the host platform — so this works on
 *      win32 even without a squirrel maker).
 *   2. Boots Electron from project root using package.json `main` =
 *      `.vite/build/main.js` (no need for the packaged `out/` artifact).
 *   3. Asserts window title contains "Grafo de Memória".
 *   4. Waits for the Cytoscape `<canvas>` to mount (data-central.json loads
 *      + cytoscape inits).
 *   5. Captures `screenshot-qg-phase15.png` as evidence.
 *
 * Run via `npm run smoke` (chains `electron-forge package` + this test).
 */

let electronApp: ElectronApplication;

test.beforeAll(async () => {
  const mainEntry = path.resolve(process.cwd(), ".vite/build/main.js");
  if (!existsSync(mainEntry)) {
    throw new Error(
      `Build artifacts missing: ${mainEntry}.\n` +
        `Run \`npm run package\` first to build the .vite/ artifacts.`,
    );
  }

  electronApp = await electron.launch({
    args: [".", "--no-sandbox"],
    cwd: process.cwd(),
  });
  electronApp.on("window", (page) => {
    const filename = page.url()?.split("/").pop();
    console.log(`Window opened: ${filename}`);

    page.on("pageerror", (error) => {
      console.error(error);
    });
    page.on("console", (msg) => {
      console.log(msg.text());
    });
  });
});

test("renders the my-memory viewer with Cytoscape canvas", async () => {
  const page: Page = await electronApp.firstWindow();
  await page.waitForLoadState("domcontentloaded");

  // Title gate
  const title = await page.title();
  expect(title).toContain("Grafo de Memória");

  // Cytoscape renders into a <canvas>; wait until at least one mounts.
  // Generous timeout because the app loads data-central.json then inits cytoscape.
  await page.waitForSelector("canvas", { timeout: 15_000 });

  // Evidence: screenshot at workspace root (gitignored — produced per-run).
  await page.screenshot({ path: "screenshot-qg-phase15.png", fullPage: true });
});

test.afterAll(async () => {
  await electronApp?.close();
});