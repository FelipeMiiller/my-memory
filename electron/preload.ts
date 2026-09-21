import { contextBridge, ipcRenderer } from 'electron';

/**
 * Phase 1 (MVP) — minimal safe IPC surface.
 *
 * Stubs return rejected promises; Phase 2 fills real handlers in
 * electron/main.ts `ipcMain.handle()` for: `mem:dataset:load`, `mem:cli:run`,
 * `mem:reveal`. Renderer NEVER touches ipcRenderer directly — only the
 * `memAPI` object exposed here.
 */
const memAPI = {
  platform: process.platform,
  loadDataset: (): Promise<unknown> =>
    ipcRenderer.invoke('mem:dataset:load').catch((err: unknown) => {
      console.warn('[preload] mem:dataset:load not yet wired (Phase 2)', err);
      return Promise.reject(new Error('mem:dataset:load not implemented (Phase 2)'));
    }),
  runCli: (args: string[]): Promise<unknown> =>
    ipcRenderer.invoke('mem:cli:run', args).catch((err: unknown) => {
      console.warn('[preload] mem:cli:run not yet wired (Phase 2)', err);
      return Promise.reject(new Error('mem:cli:run not implemented (Phase 2)'));
    }),
};

contextBridge.exposeInMainWorld('memAPI', memAPI);

// Type declaration for renderer (Phase 2 will move to viewer/src/types/mem-api.d.ts)
export type MemAPI = typeof memAPI;
