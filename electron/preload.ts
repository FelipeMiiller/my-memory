import { contextBridge, ipcRenderer, type IpcRendererEvent } from "electron";
import { IPC_CHANNELS } from "@electron/constants";

/**
 * Phase 1 (MVP) — minimal safe IPC surface.
 *
 * Stubs return rejected promises; Phase 2 fills real handlers in main.ts
 * `ipcMain.handle()` for `mem:dataset:load`, `mem:cli:run`, `mem:reveal`.
 * Renderer NEVER touches ipcRenderer directly — only the `memAPI` object.
 */
const memAPI = {
  platform: process.platform,
  loadDataset: (): Promise<unknown> =>
    ipcRenderer.invoke(IPC_CHANNELS.MEM_DATASET_LOAD).catch((err: unknown) => {
      console.warn("[preload] mem:dataset:load not yet wired (Phase 2)", err);
      return Promise.reject(
        new Error("mem:dataset:load not implemented (Phase 2)"),
      );
    }),
  runCli: (args: string[]): Promise<unknown> =>
    ipcRenderer.invoke(IPC_CHANNELS.MEM_CLI_RUN, args).catch((err: unknown) => {
      console.warn("[preload] mem:cli:run not yet wired (Phase 2)", err);
      return Promise.reject(new Error("mem:cli:run not implemented (Phase 2)"));
    }),
  // Chat (ADR-051, ADR-052) — safe IPC surface; API keys NEVER cross this boundary.
  chat: {
    send: (req: ChatSendRequest): Promise<ChatSendResponse> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_SEND, req),
    abort: (requestId: string): Promise<{ aborted: boolean }> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_ABORT, { requestId }),
    getConfig: (): Promise<ChatConfig> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_CONFIG_GET),
    setConfig: (update: ChatConfigUpdate): Promise<ChatConfig> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_CONFIG_SET, update),
    memSearch: (query: string, topK = 5): Promise<MemSearchResponse> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_MEM_SEARCH, { query, topK }),
    getProviders: (): Promise<ChatProvidersResponse> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_PROVIDERS_GET),
    discoverModels: (vendor: string): Promise<{ vendor: string; models: ChatModelSummary[] }> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_CHAT_MODELS_DISCOVER, { vendor }),
    // Streaming listeners — return an unsubscribe function.
    onDelta: (cb: (e: ChatDeltaEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatDeltaEvent): void => cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_DELTA, handler);
      return () => ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_DELTA, handler);
    },
    onDone: (cb: (e: ChatDoneEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatDoneEvent): void => cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_DONE, handler);
      return () => ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_DONE, handler);
    },
    onError: (cb: (e: ChatErrorEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatErrorEvent): void => cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_ERROR, handler);
      return () => ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_ERROR, handler);
    },
  },
};

contextBridge.exposeInMainWorld("memAPI", memAPI);

export type MemAPI = typeof memAPI;