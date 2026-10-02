import { IPC_CHANNELS } from "@electron/constants";
import { contextBridge, type IpcRendererEvent, ipcRenderer } from "electron";

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
  // Window controls (custom title-bar buttons)
  window: {
    minimize: (): Promise<void> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_WINDOW_MINIMIZE),
    toggleMaximize: (): Promise<boolean> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_WINDOW_MAXIMIZE_TOGGLE),
    close: (): Promise<void> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_WINDOW_CLOSE),
    isMaximized: (): Promise<boolean> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_WINDOW_IS_MAXIMIZED),
    onMaximizeChanged: (cb: (maximized: boolean) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, maximized: boolean): void =>
        cb(maximized);
      ipcRenderer.on("mem:window:maximize-changed", handler);
      return () =>
        ipcRenderer.removeListener("mem:window:maximize-changed", handler);
    },
  },
  // Chat (ADR-051) — safe IPC surface; API keys NEVER cross this boundary.
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
    // Streaming listeners — return an unsubscribe function.
    onDelta: (cb: (e: ChatDeltaEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatDeltaEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_DELTA, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_DELTA, handler);
    },
    onDone: (cb: (e: ChatDoneEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatDoneEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_DONE, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_DONE, handler);
    },
    onError: (cb: (e: ChatErrorEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: ChatErrorEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_CHAT_ERROR, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_CHAT_ERROR, handler);
    },
  },
  // ASR (Nemotron streaming, ADR-045) — UI scaffold. Real model lives in Go.
  // Renderer captures PCM via getUserMedia, sends chunks; main acknowledges
  // and (currently) echoes synthetic partials. Replace with Go Nemotron client
  // once spec nemotron-asr-streaming T1-T9 land.
  asr: {
    start: (req?: AsrStartRequest): Promise<AsrStartResponse> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_ASR_START, req ?? {}),
    chunk: (req: AsrChunkRequest): Promise<{ accepted: boolean }> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_ASR_CHUNK, req),
    stop: (req: AsrStopRequest): Promise<AsrStopResponse> =>
      ipcRenderer.invoke(IPC_CHANNELS.MEM_ASR_STOP, req),
    onPartial: (cb: (e: AsrPartialEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: AsrPartialEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_ASR_PARTIAL, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_ASR_PARTIAL, handler);
    },
    onFinal: (cb: (e: AsrFinalEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: AsrFinalEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_ASR_FINAL, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_ASR_FINAL, handler);
    },
    onError: (cb: (e: AsrErrorEvent) => void): (() => void) => {
      const handler = (_e: IpcRendererEvent, payload: AsrErrorEvent): void =>
        cb(payload);
      ipcRenderer.on(IPC_CHANNELS.MEM_ASR_ERROR, handler);
      return () =>
        ipcRenderer.removeListener(IPC_CHANNELS.MEM_ASR_ERROR, handler);
    },
  },
};

contextBridge.exposeInMainWorld("memAPI", memAPI);

export type MemAPI = typeof memAPI;
