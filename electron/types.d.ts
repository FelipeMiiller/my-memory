// Ambient declarations for the Electron main process.
//
// This file MUST remain a script (no top-level `export` or `import`), so all
// `declare const` here are visible to `electron/main.ts` without qualification.
// The @electron-forge/plugin-vite plugin also declares these in
// `forge-vite-env.d.ts`, but those live in node_modules and aren't auto-loaded
// by `tsc --noEmit`, so we redeclare them here for the typecheck step.

declare const MAIN_WINDOW_VITE_DEV_SERVER_URL: string;
declare const MAIN_WINDOW_VITE_NAME: string;

/**
 * Chat types (ADR-051, ADR-052) — shared between main and renderer.
 * Kept here as ambient declarations so both sides see them without imports.
 */

// Legacy hardcoded provider — kept for backward compat, but renderer should
// use the dynamic `ChatProviderConfig` from chatLanguageModels.json now.
type ChatProvider = "anthropic" | "openai";

const CHAT_MODELS: Record<ChatProvider, readonly string[]>;

// Renderer-safe view of chat config (no apiKeys ever).
interface ChatConfig {
  activeVendor: string;
  activeModelId: string;
  useMemory: boolean;
  providers: Array<{ vendor: string; hasApiKey: boolean }>;
}

// Payload for setConfig — apiKeys merged with existing.
interface ChatConfigUpdate {
  apiKeys?: Record<string, string>;
  useMemory?: boolean;
  activeVendor?: string;
  activeModelId?: string;
}

// One chat message.
interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  createdAt: number;
  sources?: ReadonlyArray<{ path: string; title: string; score: number }>;
}

// Streaming request from renderer to main (ADR-052: vendor + modelId).
interface ChatSendRequest {
  conversationId: string;
  messages: ReadonlyArray<ChatMessage>;
  system?: string;
  vendor: string;
  modelId: string;
}

interface ChatSendResponse {
  requestId: string;
}

interface ChatDeltaEvent {
  requestId: string;
  delta: string;
}

interface ChatDoneEvent {
  requestId: string;
  totalChars: number;
  latencyMs: number;
}

type ChatErrorKind = "auth" | "rate_limit" | "network" | "invalid" | "unknown";

interface ChatError {
  kind: ChatErrorKind;
  message: string;
}

interface ChatErrorEvent {
  requestId: string;
  error: ChatError;
}

interface MemSearchResult {
  path: string;
  title: string;
  snippet: string;
  score: number;
}

interface MemSearchResponse {
  query: string;
  results: ReadonlyArray<MemSearchResult>;
}

// ───────────────────────────────────────────────────────────
// ADR-052: dynamic providers via chatLanguageModels.json
// (VS Code-style schema)
// ───────────────────────────────────────────────────────────

type ApiType = "chat-completions" | "responses" | "messages";
type ReasoningEffort = "low" | "medium" | "high";

interface LanguageModelConfig {
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

interface ChatProviderConfig {
  vendor: string;
  name: string;
  apiType?: ApiType;
  baseUrl?: string;
  models: LanguageModelConfig[];
  autoDiscover?: boolean;
}

interface ChatLanguageModelsConfig {
  schemaVersion: 1;
  providers: ChatProviderConfig[];
}

interface ChatModelSummary {
  id: string;
  name?: string;
}

interface ChatProvidersResponse {
  providers: ChatProviderConfig[];
}

// Full config as stored in main process (NEVER returned to renderer).
interface PersistedChatConfig {
  schemaVersion: number;
  apiKeys: Record<string, string>;
  useMemory: boolean;
  activeVendor: string;
  activeModelId: string;
  updatedAt: number;
}

// ASR (Nemotron streaming, ADR-045) — shared between main and renderer.
//
// Audio pipeline:
//   1. Renderer calls getUserMedia({audio: {sampleRate: 16000, channelCount: 1}})
//   2. Captures 16 kHz mono Float32, converts to Int16 PCM, batches into 160ms chunks
//   3. Each chunk → `mem:asr:chunk` (Int16Array — structured-cloneable)
//   4. Main emits `mem:asr:partial` per intermediate transcript and
//      `mem:asr:final` on end-of-utterance (>500ms silence, ADR-045 §Configuration).
//
// The real Nemotron 3.5 ONNX client lives in `internal/asr/nemotron_onnx.go`.
// These handlers are stubs until the model is downloaded (`mem asr download`).

interface AsrStartRequest {
  /** Optional BCP-47 locale (e.g. "pt-BR", "en-US"). Defaults to "auto". */
  locale?: string;
  /** Correlation id from the upstream chat turn — propagated into envelopes. */
  conversationTurnId?: string;
}

interface AsrStartResponse {
  sessionId: string;
}

interface AsrChunkRequest {
  sessionId: string;
  /** Int16 little-endian mono PCM at 16 kHz. Renderer batches 160ms (2560 samples). */
  pcm: Int16Array;
  /** Sample rate in Hz. Always 16000 in this build. */
  sampleRate: number;
}

interface AsrStopRequest {
  sessionId: string;
}

interface AsrStopResponse {
  finalized: boolean;
}

interface AsrPartialEvent {
  sessionId: string;
  /** Interim transcript text. Replaces prior partial for the same session. */
  transcript: string;
  /** Confidence in [0, 1] when reported by the model; omit if unavailable. */
  confidence?: number;
}

interface AsrFinalEvent {
  sessionId: string;
  transcript: string;
  durationMs: number;
  locale: string;
}

type AsrErrorKind =
  | "model_missing"
  | "model_loading"
  | "permission_denied"
  | "invalid_audio"
  | "aborted"
  | "unknown";

interface AsrErrorEvent {
  sessionId: string;
  kind: AsrErrorKind;
  message: string;
}
