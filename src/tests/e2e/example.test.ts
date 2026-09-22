import {
  type ElectronApplication,
  _electron as electron,
  expect,
  type Page,
  test,
} from "@playwright/test";
import { findLatestBuild, parseElectronApp } from "electron-playwright-helpers";

/**
 * E2E smoke — my-memory viewer (Electron Forge).
 *
 * Boots the packaged build via `npm run package`, opens the first window,
 * and asserts the title contains "Grafo de Memória".
 *
 * Run:
 *   npm run package       # builds the app
 *   npm run test:e2e      # runs Playwright
 */

let electronApp: ElectronApplication;

test.beforeAll(async () => {
  const latestBuild = findLatestBuild();
  const appInfo = parseElectronApp(latestBuild);
  process.env.CI = "e2e";

  electronApp = await electron.launch({
    args: [appInfo.main],
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

test("renders the my-memory viewer", async () => {
  const page: Page = await electronApp.firstWindow();

  const title = await page.title();
  expect(title).toContain("Grafo de Memória");
});

test.afterAll(async () => {
  await electronApp.close();
});