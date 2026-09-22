import { anthropic } from "@ai-sdk/anthropic";
import { createAnthropic } from "@ai-sdk/anthropic";
import { createOpenAI } from "@ai-sdk/openai";
import { createOpenAICompatible } from "@ai-sdk/openai-compatible";
import { streamText, type LanguageModel } from "ai";

/**
 * LLM streaming wrapper (ADR-051, ADR-052).
 *
 * Dispatches to the right AI SDK provider client based on the model config's
 * `vendor` + `apiType` (resolved by the caller from chatLanguageModels.json).
 * Supports:
 *   - anthropic (messages API) → @ai-sdk/anthropic
 *   - openai (responses API)   → @ai-sdk/openai
 *   - any OpenAI-compatible (chat-completions) → @ai-sdk/openai-compatible
 *
 * The renderer NEVER touches the API keys. The IPC layer resolves the model
 * config from chatLanguageModels.json + the user-provided key from
 * chat-config.json, then passes both to this layer.
 */

export interface StreamChatParams {
  provider: ChatProviderConfig;
  model: LanguageModelConfig;
  apiKey: string;
  system: string;
  messages: ReadonlyArray<{ role: "user" | "assistant" | "system"; content: string }>;
  signal: AbortSignal;
}

export interface StreamChatHandlers {
  onDelta: (text: string) => void;
  onDone: (info: { totalChars: number; latencyMs: number }) => void;
  onError: (err: ChatError) => void;
}

function buildModelClient(
  provider: ChatProviderConfig,
  model: LanguageModelConfig,
  apiKey: string,
): LanguageModel {
  const apiType = model.apiType ?? provider.apiType ?? "chat-completions";
  const baseUrl = provider.baseUrl;

  // Anthropic native
  if (
    apiType === "messages" ||
    provider.vendor === "anthropic" ||
    baseUrl?.includes("anthropic.com")
  ) {
    // Use createAnthropic when baseUrl is overridden (proxy); default
    // instance when calling Anthropic directly.
    const factory = baseUrl ? createAnthropic({ apiKey, baseURL: baseUrl }) : anthropic;
    return factory(model.id);
  }

  // OpenAI native (responses API)
  if (provider.vendor === "openai" && !baseUrl) {
    return createOpenAI({ apiKey })(model.id);
  }

  // OpenAI-compatible (Groq, OpenRouter, Ollama, custom endpoints, etc.)
  const compat = createOpenAICompatible({
    name: provider.name,
    apiKey,
    baseURL: baseUrl ?? "https://api.openai.com/v1",
    headers: model.requestHeaders ?? {},
  });
  return compat(model.id);
}

export async function streamChat(
  params: StreamChatParams,
  handlers: StreamChatHandlers,
): Promise<void> {
  const start = Date.now();

  if (!params.apiKey || params.apiKey.trim().length === 0) {
    handlers.onError({
      kind: "auth",
      message: "API key not configured. Open Settings to add one.",
    });
    return;
  }

  try {
    const modelClient = buildModelClient(params.provider, params.model, params.apiKey);

    const result = streamText({
      model: modelClient,
      system: params.system,
      messages: [...params.messages],
      abortSignal: params.signal,
      onError: ({ error }) => {
        handlers.onError(classifyError(error));
      },
    });

    let totalChars = 0;
    for await (const chunk of result.textStream) {
      if (params.signal.aborted) break;
      totalChars += chunk.length;
      handlers.onDelta(chunk);
    }

    if (!params.signal.aborted) {
      handlers.onDone({ totalChars, latencyMs: Date.now() - start });
    }
  } catch (err: unknown) {
    if (params.signal.aborted) return;
    handlers.onError(classifyError(err));
  }
}

/**
 * Best-effort error classification. AI SDK errors have varied shapes across
 * providers, so we look at common signals.
 */
function classifyError(err: unknown): ChatError {
  const message = err instanceof Error ? err.message : String(err);
  const lower = message.toLowerCase();

  if (
    lower.includes("401") ||
    lower.includes("403") ||
    lower.includes("unauthorized") ||
    lower.includes("invalid api key") ||
    lower.includes("authentication")
  ) {
    return { kind: "auth", message: sanitizeMessage(message) };
  }
  if (lower.includes("429") || lower.includes("rate limit")) {
    return { kind: "rate_limit", message: "Rate limit reached. Try again in a moment." };
  }
  if (
    lower.includes("econnrefused") ||
    lower.includes("etimedout") ||
    lower.includes("enotfound") ||
    lower.includes("network") ||
    lower.includes("fetch failed")
  ) {
    return { kind: "network", message: "Network error. Check your connection." };
  }
  return { kind: "unknown", message: sanitizeMessage(message) };
}

function sanitizeMessage(msg: string): string {
  const sentenceMatch = msg.split(/[.\n]/).find((s) => s.trim().length > 0);
  const first = (sentenceMatch ?? msg).slice(0, 200).trim();
  // Defensive: redact any token-like fragments (sk-ant-…, sk-…, ghp_…, etc.)
  return first.replace(/(sk-|ghp_|gho_|sk-ant-)[a-zA-Z0-9_-]{10,}/g, "[REDACTED]") || "An error occurred.";
}
