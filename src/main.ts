import path from "node:path";
import { app, BrowserWindow, shell } from "electron";
import { IPC_CHANNELS, inDevelopment } from "@/constants";
import { getBasePath } from "@/utils/path";

/**
 * my-memory viewer — Electron main process.
 *
 * Security posture (deliberate downgrade vs template default):
 *   - contextIsolation: true   (no Node in renderer)
 *   - nodeIntegration: false   (renderer cannot require())
 *   - sandbox: true            (OS-level sandbox)
 *   - preload exposes safe `memAPI` via contextBridge
 *
 * Hardcoded CSP via session.webRequest (Phase 1 keeps `'unsafe-inline'` for
 * HMR convenience; Phase 2 ADR-049 promotes strict CSP without unsafe-inline).
 */

function createMainWindow(): BrowserWindow {
  const basePath = getBasePath();
  const preload = path.join(basePath, "preload.js");
  const mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    show: false,
    autoHideMenuBar: true,
    title: "Grafo de Memória [FelipeMiiller/my-memory]",
    backgroundColor: "#0a0a0a",
    titleBarStyle: process.platform === "darwin" ? "hiddenInset" : "hidden",
    trafficLightPosition:
      process.platform === "darwin" ? { x: 5, y: 5 } : undefined,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      nodeIntegrationInSubFrames: false,
      sandbox: true,
      devTools: inDevelopment,
      preload,
    },
  });

  // Hardcoded CSP — mirrors index.html meta tag, applied at network layer.
  mainWindow.webContents.session.webRequest.onHeadersReceived(
    (details, callback) => {
      callback({
        responseHeaders: {
          ...details.responseHeaders,
          "Content-Security-Policy": [
            "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' http://localhost:5173 ws://localhost:5173; font-src 'self' data:; object-src 'none'; frame-src 'none';",
          ],
        },
      });
    },
  );

  // Block external navigation (security hardening).
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: "deny" };
  });

  if (MAIN_WINDOW_VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(MAIN_WINDOW_VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(
      path.join(basePath, `../renderer/${MAIN_WINDOW_VITE_NAME}/index.html`),
    );
  }

  mainWindow.once("ready-to-show", () => {
    mainWindow.show();
  });

  return mainWindow;
}

app.whenReady().then(() => {
  createMainWindow();

  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createMainWindow();
    }
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

// Wire minimal IPC handlers — Phase 2 expands via ADR-049.
import { ipcMain } from "electron/main";

ipcMain.handle(IPC_CHANNELS.MEM_DATASET_LOAD, async () => {
  return Promise.reject(new Error("mem:dataset:load not implemented (Phase 2)"));
});

ipcMain.handle(IPC_CHANNELS.MEM_CLI_RUN, async () => {
  return Promise.reject(new Error("mem:cli:run not implemented (Phase 2)"));
});