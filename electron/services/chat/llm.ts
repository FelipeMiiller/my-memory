import { createAnthropic } from "@ai-sdk/anthropic";
import { createOpenAI } from "@ai-sdk/openai";
import { streamText } from "ai";

/**
 * LLM streaming wrapper (ADR-051).
 *
 * Thin facade over Vercel AI SDK's `streamText` so the IPC layer doesn't
 * need to know provider-specific SDK shapes. The AI SDK handles the SSE
 * parsing, retries, and per-provider quirks; we just normalize the output
 * to `{onDelta(text), onDone(totalChars, latencyMs), onError(ChatError)}`.
 *
 * API key is required; we don't fall back to env vars here (the IPC layer
 * is responsible for loading the key from chat-config.json).
 */

export interface StreamChatParams {
  provider: ChatProvider;
  model: string;
  apiKey: string;
  system: string;
  messages: ReadonlyArray<{
    role: "user" | "assistant" | "system";
    content: string;
  }>;
  signal: AbortSignal;
}

export interface StreamChatHandlers {
  onDelta: (text: string) => void;
  onDone: (info: { totalChars: number; latencyMs: number }) => void;
  onError: (err: ChatError) => void;
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
    const providerClient =
      params.provider === "anthropic"
        ? createAnthropic({ apiKey: params.apiKey })
        : createOpenAI({ apiKey: params.apiKey });
    const modelClient = providerClient(params.model);

    const result = streamText({
      model: modelClient,
      system: params.system,
      messages: [...params.messages],
      abortSignal: params.signal,
      onError: ({ error }) => {
        // Map AI SDK errors to our typed shape.
        const c = classifyError(error);
        handlers.onError(c);
      },
    });

    let totalChars = 0;
    for await (const chunk of result.textStream) {
      if (params.signal.aborted) {
        break;
      }
      totalChars += chunk.length;
      handlers.onDelta(chunk);
    }

    if (!params.signal.aborted) {
      handlers.onDone({ totalChars, latencyMs: Date.now() - start });
    }
  } catch (err: unknown) {
    if (params.signal.aborted) {
      // Silent on abort — caller already knows.
      return;
    }
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
    return {
      kind: "rate_limit",
      message: "Rate limit reached. Try again in a moment.",
    };
  }
  if (
    lower.includes("econnrefused") ||
    lower.includes("etimedout") ||
    lower.includes("enotfound") ||
    lower.includes("network") ||
    lower.includes("fetch failed")
  ) {
    return {
      kind: "network",
      message: "Network error. Check your connection.",
    };
  }
  return { kind: "unknown", message: sanitizeMessage(message) };
}

/**
 * Strips suspicious fragments (stack traces, paths, hex tokens) from error
 * messages before returning to the renderer. We keep the first 200 chars of
 * a readable sentence.
 */
function sanitizeMessage(msg: string): string {
  // Take only the first sentence or 200 chars, whichever is shorter.
  const sentenceMatch = msg.split(/[.\n]/).find((s) => s.trim().length > 0);
  const first = (sentenceMatch ?? msg).slice(0, 200).trim();
  return first || "An error occurred. See main process logs for details.";
}
