import type {
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
