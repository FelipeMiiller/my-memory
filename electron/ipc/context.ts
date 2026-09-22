import type { BrowserWindow } from "electron";

/**
 * IPC context — single shared BrowserWindow reference.
 *
 * Phase 1 keeps this minimal (no ORPC middleware). Phase 2 ADR-049 can
 * layer typed IPC channels on top without changing this base.
 */
class IPCContext {
  mainWindow: BrowserWindow | undefined;

  setMainWindow(window: BrowserWindow) {
    this.mainWindow = window;
  }
}

export const ipcContext = new IPCContext();