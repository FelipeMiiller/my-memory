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
  anthropic: [
    "claude-opus-4-1",
    "claude-sonnet-4-5",
    "claude-haiku-4-5",
  ],
  openai: ["gpt-4o", "gpt-4o-mini", "o3", "o3-mini"],
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

export type ChatErrorKind = "auth" | "rate_limit" | "network" | "invalid" | "unknown";

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
