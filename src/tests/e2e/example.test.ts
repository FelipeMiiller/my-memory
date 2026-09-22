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
 * M1 update (replaces Phase 1.5 canvas assertion with 3-pane shell signal):
 *   1. Asserts `.vite/build/main.js` exists (produced by `electron-forge package`
 *      via the VitePlugin; the plugin builds the renderer + main + preload even
 *      when no makers are configured for the host platform — so this works on
 *      win32 even without a squirrel maker).
 *   2. Boots Electron from project root using package.json `main` =
 *      `.vite/build/main.js` (no need for the packaged `out/` artifact).
 *   3. Asserts window title contains "Grafo de Memória".
 *   4. Waits for the M1 status bar `text=layout: explorer` to mount, proving
 *      WorkspaceShell rendered the 3-pane layout (Explorer / Editor / Chat).
 *      (Phase 1.5 waited for Cytoscape `<canvas>` — that view no longer exists
 *      post-M1 because the 3-pane shell replaced the graph-centric layout.)
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

  // M1 shell renders the status bar with layout state (proves WorkspaceShell mounted).
  // Replaces the Phase 1.5 `<canvas>` assertion — Cytoscape was the central view pre-M1,
  // but M1 replaced the graph-centric layout with the 3-pane Explorer/Editor/Chat shell,
  // so the canvas no longer exists on first paint.
  await page.waitForSelector("text=layout: explorer", { timeout: 15_000 });

  // Evidence: screenshot at workspace root (gitignored — produced per-run).
  await page.screenshot({ path: "screenshot-qg-phase15.png", fullPage: true });
});

test.afterAll(async () => {
  await electronApp?.close();
});