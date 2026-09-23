import { IPC_CHANNELS } from "@electron/constants";
import { type BrowserWindow, ipcMain } from "electron";

/**
 * Window controls IPC (ADR-053 / M1 polish).
 *
 * Because `titleBarStyle: 'hidden'` is set on the main BrowserWindow, we
 * expose custom buttons in the TopBar that drive these handlers. Native
 * menu bar / title bar is suppressed (`autoHideMenuBar: true` + hidden title
 * bar style), so the renderer MUST call these to close / minimize / maximize.
 *
 * Restoring from maximized requires knowing the previous state — we expose
 * `MEM_WINDOW_IS_MAXIMIZED` for the renderer to display the right glyph.
 */

export function registerWindowHandlers(
  getWindow: () => BrowserWindow | null,
): void {
  ipcMain.handle(IPC_CHANNELS.MEM_WINDOW_MINIMIZE, () => {
    const win = getWindow();
    if (!win) return false;
    win.minimize();
    return true;
  });

  ipcMain.handle(IPC_CHANNELS.MEM_WINDOW_MAXIMIZE_TOGGLE, () => {
    const win = getWindow();
    if (!win) return false;
    if (win.isMaximized()) {
      win.unmaximize();
    } else {
      win.maximize();
    }
    return win.isMaximized();
  });

  ipcMain.handle(IPC_CHANNELS.MEM_WINDOW_CLOSE, () => {
    const win = getWindow();
    if (!win) return false;
    win.close();
    return true;
  });

  ipcMain.handle(IPC_CHANNELS.MEM_WINDOW_IS_MAXIMIZED, () => {
    const win = getWindow();
    return win ? win.isMaximized() : false;
  });
}
