import type {
  AsrChunkRequest,
  AsrErrorEvent,
  AsrFinalEvent,
  AsrPartialEvent,
  AsrStartRequest,
  AsrStartResponse,
  AsrStopRequest,
  AsrStopResponse,
  ChatConfig,
  ChatConfigUpdate,
  ChatDeltaEvent,
  ChatDoneEvent,
  ChatErrorEvent,
  ChatSendRequest,
  ChatSendResponse,
  MemSearchResponse,
} from "@/lib/chat/types";

/**
 * Type-safe wrappers around the chat IPC surface exposed by preload.
 *
 * `window.memAPI.chat` is set up in `electron/preload.ts` via `contextBridge`.
 * We declare it here (instead of importing from electron) because renderer
 * code never imports from `electron/` directly.
 */

declare global {
  // eslint-disable-next-line no-var
  var memAPI:
    | {
        platform: NodeJS.Platform;
        loadDataset: () => Promise<unknown>;
        runCli: (args: string[]) => Promise<unknown>;
        window: {
          minimize: () => Promise<void>;
          toggleMaximize: () => Promise<boolean>;
          close: () => Promise<void>;
          isMaximized: () => Promise<boolean>;
          onMaximizeChanged: (cb: (maximized: boolean) => void) => () => void;
        };
        chat: {
          send: (req: ChatSendRequest) => Promise<ChatSendResponse>;
          abort: (requestId: string) => Promise<{ aborted: boolean }>;
          getConfig: () => Promise<ChatConfig>;
          setConfig: (update: ChatConfigUpdate) => Promise<ChatConfig>;
          memSearch: (
            query: string,
            topK?: number,
          ) => Promise<MemSearchResponse>;
          onDelta: (cb: (e: ChatDeltaEvent) => void) => () => void;
          onDone: (cb: (e: ChatDoneEvent) => void) => () => void;
          onError: (cb: (e: ChatErrorEvent) => void) => () => void;
        };
        asr: {
          start: (req?: AsrStartRequest) => Promise<AsrStartResponse>;
          chunk: (req: AsrChunkRequest) => Promise<{ accepted: boolean }>;
          stop: (req: AsrStopRequest) => Promise<AsrStopResponse>;
          onPartial: (cb: (e: AsrPartialEvent) => void) => () => void;
          onFinal: (cb: (e: AsrFinalEvent) => void) => () => void;
          onError: (cb: (e: AsrErrorEvent) => void) => () => void;
        };
      }
    | undefined;
}

export const chatApi = {
  send: (req: ChatSendRequest): Promise<ChatSendResponse> => {
    if (!globalThis.memAPI) {
      return Promise.reject(
        new Error("memAPI not available — preload not loaded"),
      );
    }
    return globalThis.memAPI.chat.send(req);
  },
  abort: (requestId: string): Promise<{ aborted: boolean }> => {
    if (!globalThis.memAPI) return Promise.resolve({ aborted: false });
    return globalThis.memAPI.chat.abort(requestId);
  },
  getConfig: (): Promise<ChatConfig> => {
    if (!globalThis.memAPI) {
      return Promise.resolve({
        provider: "anthropic",
        model: "claude-sonnet-4-5",
        useMemory: true,
        hasApiKey: false,
      });
    }
    return globalThis.memAPI.chat.getConfig();
  },
  setConfig: (update: ChatConfigUpdate): Promise<ChatConfig> => {
    if (!globalThis.memAPI) {
      return Promise.reject(new Error("memAPI not available"));
    }
    return globalThis.memAPI.chat.setConfig(update);
  },
  memSearch: (query: string, topK = 5): Promise<MemSearchResponse> => {
    if (!globalThis.memAPI) return Promise.resolve({ query, results: [] });
    return globalThis.memAPI.chat.memSearch(query, topK);
  },
  onDelta: (cb: (e: ChatDeltaEvent) => void): (() => void) => {
    return globalThis.memAPI?.chat.onDelta(cb) ?? (() => {});
  },
  onDone: (cb: (e: ChatDoneEvent) => void): (() => void) => {
    return globalThis.memAPI?.chat.onDone(cb) ?? (() => {});
  },
  onError: (cb: (e: ChatErrorEvent) => void): (() => void) => {
    return globalThis.memAPI?.chat.onError(cb) ?? (() => {});
  },
};

export const asrApi = {
  start: (req?: AsrStartRequest): Promise<AsrStartResponse> => {
    if (!globalThis.memAPI) {
      return Promise.reject(new Error("memAPI not available"));
    }
    return globalThis.memAPI.asr.start(req);
  },
  chunk: (req: AsrChunkRequest): Promise<{ accepted: boolean }> => {
    if (!globalThis.memAPI) {
      return Promise.reject(new Error("memAPI not available"));
    }
    return globalThis.memAPI.asr.chunk(req);
  },
  stop: (req: AsrStopRequest): Promise<AsrStopResponse> => {
    if (!globalThis.memAPI) {
      return Promise.resolve({ finalized: false });
    }
    return globalThis.memAPI.asr.stop(req);
  },
  onPartial: (cb: (e: AsrPartialEvent) => void): (() => void) => {
    return globalThis.memAPI?.asr.onPartial(cb) ?? (() => {});
  },
  onFinal: (cb: (e: AsrFinalEvent) => void): (() => void) => {
    return globalThis.memAPI?.asr.onFinal(cb) ?? (() => {});
  },
  onError: (cb: (e: AsrErrorEvent) => void): (() => void) => {
    return globalThis.memAPI?.asr.onError(cb) ?? (() => {});
  },
};
