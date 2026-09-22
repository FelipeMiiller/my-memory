/**
 * Chat types — renderer side.
 *
 * Mirrors the ambient declarations in `electron/types.d.ts`.
 * Keep both files in sync. If they drift, tsc will not catch it (these are
 * structural shapes), so the E2E test in src/tests/e2e/chat.test.ts should
 * import both sides and assert shape compatibility.
 */

export type ChatProvider = "anthropic" | "openai";

export const CHAT_MODELS: Record<ChatProvider, readonly string[]> = {
  anthropic: ["claude-opus-4-1", "claude-sonnet-4-5", "claude-haiku-4-5"],
  openai: ["gpt-4o", "gpt-4o-mini", "o3", "o3-mini"],
} as const;

export const CHAT_PROVIDER_NAMES: Record<ChatProvider, string> = {
  anthropic: "Anthropic Claude",
  openai: "OpenAI",
} as const;

/**
 * Per-model display + capability metadata used by the ChatModelPicker UI.
 *
 * Independent from `CHAT_MODELS` (which is the wire-side id list) so the
 * `chatLanguageModels.json` migration (ADR-052) can replace the id source
 * without touching the display layer. When `ChatModelMeta[id]` is missing,
 * the picker falls back to the raw id.
 */
export interface ChatModelMeta {
  /** Wire id (matches `CHAT_MODELS[*][i]` and `ChatConfig.model`). */
  id: string;
  /** Human-readable name shown in the picker. */
  displayName: string;
  /** Relative speed tier — rendered as `Nx` badge. Lower = faster. */
  speed: 1 | 2 | 3;
  /** Reasoning effort capability (mapped to `supportsReasoningEffort` in ADR-052). */
  reasoningEffort?: "low" | "medium" | "high";
}

export const CHAT_MODEL_META: Readonly<Record<string, ChatModelMeta>> = {
  "claude-haiku-4-5": {
    id: "claude-haiku-4-5",
    displayName: "Claude Haiku 4.5",
    speed: 1,
  },
  "claude-sonnet-4-5": {
    id: "claude-sonnet-4-5",
    displayName: "Claude Sonnet 4.5",
    speed: 1,
    reasoningEffort: "medium",
  },
  "claude-opus-4-1": {
    id: "claude-opus-4-1",
    displayName: "Claude Opus 4.1",
    speed: 2,
  },
  "gpt-4o-mini": {
    id: "gpt-4o-mini",
    displayName: "GPT-4o mini",
    speed: 1,
  },
  "gpt-4o": {
    id: "gpt-4o",
    displayName: "GPT-4o",
    speed: 1,
  },
  "o3-mini": {
    id: "o3-mini",
    displayName: "o3-mini",
    speed: 2,
    reasoningEffort: "medium",
  },
  o3: {
    id: "o3",
    displayName: "o3",
    speed: 3,
    reasoningEffort: "high",
  },
} as const;

export const DEFAULT_CHAT_CONFIG: ChatConfig = {
  provider: "anthropic",
  model: "claude-sonnet-4-5",
  useMemory: true,
  hasApiKey: false,
};

export interface ChatConfig {
  provider: ChatProvider;
  model: string;
  useMemory: boolean;
  hasApiKey: boolean;
}

export interface ChatConfigUpdate {
  provider?: ChatProvider;
  model?: string;
  apiKey?: string;
  useMemory?: boolean;
}

export interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  createdAt: number;
  sources?: ReadonlyArray<{ path: string; title: string; score: number }>;
}

export interface ChatSendRequest {
  conversationId: string;
  messages: ReadonlyArray<ChatMessage>;
  system?: string;
}

export interface ChatSendResponse {
  requestId: string;
}

export interface ChatDeltaEvent {
  requestId: string;
  delta: string;
}

export interface ChatDoneEvent {
  requestId: string;
  totalChars: number;
  latencyMs: number;
}

export type ChatErrorKind =
  | "auth"
  | "rate_limit"
  | "network"
  | "invalid"
  | "unknown";

export interface ChatError {
  kind: ChatErrorKind;
  message: string;
}

export interface ChatErrorEvent {
  requestId: string;
  error: ChatError;
}

export interface MemSearchResult {
  path: string;
  title: string;
  snippet: string;
  score: number;
}

export interface MemSearchResponse {
  query: string;
  results: ReadonlyArray<MemSearchResult>;
}

// Renderer-side conversation grouping (persisted in IndexedDB).
export interface ChatConversation {
  id: string;
  title: string;
  createdAt: number;
  updatedAt: number;
  messages: ChatMessage[];
}
