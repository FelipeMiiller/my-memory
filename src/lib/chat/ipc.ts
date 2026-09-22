import type {
  ChatConfig,
  ChatConfigUpdate,
  ChatSendRequest,
  ChatSendResponse,
  ChatDeltaEvent,
  ChatDoneEvent,
  ChatErrorEvent,
  MemSearchResponse,
  ChatProvidersResponse,
  ChatModelSummary,
} from "@/lib/chat/types";

/**
 * Type-safe wrappers around the chat IPC surface exposed by preload.
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
          memSearch: (query: string, topK?: number) => Promise<MemSearchResponse>;
          getProviders: () => Promise<ChatProvidersResponse>;
          discoverModels: (vendor: string) => Promise<{ vendor: string; models: ChatModelSummary[] }>;
          onDelta: (cb: (e: ChatDeltaEvent) => void) => () => void;
          onDone: (cb: (e: ChatDoneEvent) => void) => () => void;
          onError: (cb: (e: ChatErrorEvent) => void) => () => void;
        };
      }
    | undefined;
}

const FALLBACK_CONFIG: ChatConfig = {
  activeVendor: "anthropic",
  activeModelId: "claude-sonnet-4-5",
  useMemory: true,
  providers: [],
};

export const chatApi = {
  send: (req: ChatSendRequest): Promise<ChatSendResponse> => {
    if (!globalThis.memAPI) return Promise.reject(new Error("memAPI not loaded"));
    return globalThis.memAPI.chat.send(req);
  },
  abort: (requestId: string): Promise<{ aborted: boolean }> => {
    if (!globalThis.memAPI) return Promise.resolve({ aborted: false });
    return globalThis.memAPI.chat.abort(requestId);
  },
  getConfig: (): Promise<ChatConfig> => {
    if (!globalThis.memAPI) return Promise.resolve(FALLBACK_CONFIG);
    return globalThis.memAPI.chat.getConfig();
  },
  setConfig: (update: ChatConfigUpdate): Promise<ChatConfig> => {
    if (!globalThis.memAPI) return Promise.reject(new Error("memAPI not loaded"));
    return globalThis.memAPI.chat.setConfig(update);
  },
  memSearch: (query: string, topK = 5): Promise<MemSearchResponse> => {
    if (!globalThis.memAPI) return Promise.resolve({ query, results: [] });
    return globalThis.memAPI.chat.memSearch(query, topK);
  },
  getProviders: (): Promise<ChatProvidersResponse> => {
    if (!globalThis.memAPI) return Promise.resolve({ providers: [] });
    return globalThis.memAPI.chat.getProviders();
  },
  discoverModels: (vendor: string) => {
    if (!globalThis.memAPI) return Promise.resolve({ vendor, models: [] });
    return globalThis.memAPI.chat.discoverModels(vendor);
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
