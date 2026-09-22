/**
 * Chat types — renderer side.
 *
 * Mirrors the ambient declarations in `electron/types.d.ts`. Keep in sync.
 */

export type ChatProvider = "anthropic" | "openai";

export type ApiType = "chat-completions" | "responses" | "messages";
export type ReasoningEffort = "low" | "medium" | "high";

export interface LanguageModelConfig {
  id: string;
  name: string;
  url?: string;
  apiType?: ApiType;
  toolCalling?: boolean;
  vision?: boolean;
  thinking?: boolean;
  streaming?: boolean;
  maxInputTokens?: number;
  maxOutputTokens?: number;
  supportsReasoningEffort?: ReasoningEffort[];
  zeroDataRetentionEnabled?: boolean;
  requestHeaders?: Record<string, string>;
}

export interface ChatProviderConfig {
  vendor: string;
  name: string;
  apiType?: ApiType;
  baseUrl?: string;
  models: LanguageModelConfig[];
  autoDiscover?: boolean;
}

export interface ChatLanguageModelsConfig {
  schemaVersion: 1;
  providers: ChatProviderConfig[];
}

export interface ChatModelSummary {
  id: string;
  name?: string;
}

export interface ChatProvidersResponse {
  providers: ChatProviderConfig[];
}

/** Renderer-safe view of chat config — never includes apiKeys. */
export interface ChatConfig {
  activeVendor: string;
  activeModelId: string;
  useMemory: boolean;
  providers: Array<{ vendor: string; hasApiKey: boolean }>;
}

export interface ChatConfigUpdate {
  apiKeys?: Record<string, string>;
  useMemory?: boolean;
  activeVendor?: string;
  activeModelId?: string;
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
  vendor: string;
  modelId: string;
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

export interface ChatConversation {
  id: string;
  title: string;
  createdAt: number;
  updatedAt: number;
  messages: ChatMessage[];
}
