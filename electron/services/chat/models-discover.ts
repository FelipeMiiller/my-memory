/**
 * Auto-discover models for a provider by calling its `/models` endpoint.
 *
 * Supports:
 *   - OpenAI-compatible: GET {baseUrl}/models (with optional Bearer auth)
 *   - Anthropic:         GET {baseUrl}/v1/models (with `x-api-key` header)
 *
 * Returns a summary list (id + display name when available). The renderer
 * can use this to populate the model picker for any provider with
 * `autoDiscover: true`.
 */

export interface ChatModelSummary {
  id: string;
  name?: string;
}

const DISCOVERY_TIMEOUT_MS = 5_000;

/**
 * SSRF guard: refuse to fetch from loopback / link-local / private ranges
 * unless the URL is on localhost (Ollama is allowed locally). This prevents
 * a malicious config from pinging internal services.
 */
function assertSafeUrl(rawUrl: string): URL {
  const url = new URL(rawUrl);
  const host = url.hostname;
  // Allow http/https only.
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error(`Refusing non-http(s) URL: ${url.protocol}`);
  }
  // Allow localhost and 127.0.0.1 (Ollama / LM Studio).
  if (host === "localhost" || host === "127.0.0.1" || host === "::1") return url;
  // Refuse private/loopback/link-local IPs.
  const parts = host.split(".").map(Number);
  if (parts.length === 4 && parts.every((n) => !Number.isNaN(n))) {
    const [a, b] = parts;
    if (
      a === 10 ||
      a === 127 ||
      a === 0 ||
      (a === 172 && b >= 16 && b <= 31) ||
      (a === 192 && b === 168) ||
      (a === 169 && b === 254)
    ) {
      throw new Error(`Refusing private IP: ${host}`);
    }
  }
  return url;
}

export async function discoverProviderModels(
  provider: ChatProviderConfig,
  apiKey: string,
): Promise<ChatModelSummary[]> {
  if (!provider.baseUrl) return [];
  const apiType = provider.apiType ?? "chat-completions";

  let endpoint: string;
  let headers: Record<string, string> = {};

  if (apiType === "messages" || provider.vendor === "anthropic") {
    // Anthropic: GET /v1/models with x-api-key + anthropic-version.
    const base = provider.baseUrl.replace(/\/+$/, "");
    endpoint = `${base}/v1/models`;
    if (apiKey) {
      headers["x-api-key"] = apiKey;
      headers["anthropic-version"] = "2023-06-01";
    }
  } else {
    // OpenAI-compatible: GET /models (relative to baseUrl).
    const base = provider.baseUrl.replace(/\/+$/, "");
    endpoint = `${base}/models`;
    if (apiKey) headers["Authorization"] = `Bearer ${apiKey}`;
  }

  // Apply custom headers (e.g., OpenRouter needs HTTP-Referer).
  if (provider.models[0]?.requestHeaders) {
    headers = { ...headers, ...provider.models[0].requestHeaders };
  }

  let parsedUrl: URL;
  try {
    parsedUrl = assertSafeUrl(endpoint);
  } catch (err) {
    console.warn(`[chat/discover] blocked ${provider.vendor}:`, err);
    return [];
  }

  try {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), DISCOVERY_TIMEOUT_MS);
    const res = await fetch(parsedUrl, { method: "GET", headers, signal: controller.signal });
    clearTimeout(timer);
    if (!res.ok) {
      console.warn(`[chat/discover] ${provider.vendor} → ${res.status}`);
      return [];
    }
    const body = (await res.json()) as unknown;
    return parseModelsResponse(body, apiType);
  } catch (err: unknown) {
    console.warn(`[chat/discover] ${provider.vendor} fetch failed:`, err);
    return [];
  }
}

function parseModelsResponse(body: unknown, apiType: string): ChatModelSummary[] {
  if (typeof body !== "object" || body === null) return [];
  const b = body as { data?: unknown; models?: unknown };
  const arr = Array.isArray(b.data) ? b.data : Array.isArray(b.models) ? b.models : [];
  const out: ChatModelSummary[] = [];
  for (const item of arr) {
    if (typeof item !== "object" || item === null) continue;
    const o = item as Record<string, unknown>;
    const id = typeof o.id === "string" ? o.id : typeof o.name === "string" ? o.name : null;
    if (!id) continue;
    const displayName = typeof o.display_name === "string" ? o.display_name : undefined;
    out.push(displayName ? { id, name: displayName } : { id });
  }
  return out;
}
