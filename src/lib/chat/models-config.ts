/**
 * Schema for `chatLanguageModels.json` — VS Code-style chat config.
 *
 * Reference: https://code.visualstudio.com/docs/agent-customization/language-models
 *
 * Mirrors the ambient declarations in `electron/types.d.ts`. Keep in sync.
 *
 * The JSON file lives in `app.getPath('userData')/chatLanguageModels.json`. On
 * first boot, the bundled defaults from `resources/chatLanguageModels.default.json`
 * are copied to that path if no user file exists. Subsequent edits are user-driven.
 */

export type ApiType = "chat-completions" | "responses" | "messages";

export type ReasoningEffort = "low" | "medium" | "high";

export interface LanguageModelConfig {
  id: string;
  name: string;
  /** Full URL to the model endpoint. If omitted, `${baseUrl}/${id}` is used. */
  url?: string;
  /** Override the provider-level apiType for this specific model. */
  apiType?: ApiType;
  toolCalling?: boolean;
  vision?: boolean;
  thinking?: boolean;
  /** Default true. False = model is non-streaming. */
  streaming?: boolean;
  maxInputTokens?: number;
  maxOutputTokens?: number;
  supportsReasoningEffort?: ReasoningEffort[];
  zeroDataRetentionEnabled?: boolean;
  /** HTTP headers to add to every request (e.g., custom auth). */
  requestHeaders?: Record<string, string>;
}

export interface ChatProviderConfig {
  /** Vendor key (e.g., "anthropic", "openai", "ollama", "customendpoint"). */
  vendor: string;
  /** Display name in the settings UI. */
  name: string;
  /** API protocol; determines which AI SDK client to use. */
  apiType?: ApiType;
  /** Base URL for the provider's API. */
  baseUrl?: string;
  /** Curated list of models. May be empty if `autoDiscover: true`. */
  models: LanguageModelConfig[];
  /** If true, app fetches models from `${baseUrl}/models` on first boot + on demand. */
  autoDiscover?: boolean;
}

export interface ChatLanguageModelsConfig {
  schemaVersion: 1;
  providers: ChatProviderConfig[];
}

export const DEFAULT_VENDOR_KEY = "customendpoint";

/**
 * Validate raw JSON before trusting it. Returns the sanitized config or
 * throws with a helpful error message naming the first invalid field.
 */
export function parseConfig(raw: unknown): ChatLanguageModelsConfig {
  if (typeof raw !== "object" || raw === null) {
    throw new Error("chatLanguageModels.json: root must be an object");
  }
  const r = raw as Partial<ChatLanguageModelsConfig>;
  if (r.schemaVersion !== 1) {
    throw new Error(
      `chatLanguageModels.json: unsupported schemaVersion ${String(r.schemaVersion)} (expected 1)`,
    );
  }
  if (!Array.isArray(r.providers)) {
    throw new Error("chatLanguageModels.json: providers must be an array");
  }

  const providers: ChatProviderConfig[] = [];
  const seenVendors = new Set<string>();
  for (let i = 0; i < r.providers.length; i++) {
    const p = r.providers[i] as Partial<ChatProviderConfig>;
    if (typeof p.vendor !== "string" || p.vendor.length === 0) {
      throw new Error(`chatLanguageModels.json: providers[${i}].vendor missing`);
    }
    if (typeof p.name !== "string" || p.name.length === 0) {
      throw new Error(`chatLanguageModels.json: providers[${i}].name missing`);
    }
    if (seenVendors.has(p.vendor)) {
      throw new Error(`chatLanguageModels.json: duplicate vendor "${p.vendor}"`);
    }
    seenVendors.add(p.vendor);

    if (!Array.isArray(p.models)) {
      throw new Error(`chatLanguageModels.json: providers[${i}].models must be an array`);
    }

    const models: LanguageModelConfig[] = [];
    const seenIds = new Set<string>();
    for (let j = 0; j < p.models.length; j++) {
      const m = p.models[j] as Partial<LanguageModelConfig>;
      if (typeof m.id !== "string" || m.id.length === 0) {
        throw new Error(
          `chatLanguageModels.json: providers[${i}].models[${j}].id missing`,
        );
      }
      if (seenIds.has(m.id)) {
        throw new Error(
          `chatLanguageModels.json: providers[${i}] has duplicate model id "${m.id}"`,
        );
      }
      seenIds.add(m.id);
      models.push({
        id: m.id,
        name: typeof m.name === "string" ? m.name : m.id,
        url: typeof m.url === "string" ? m.url : undefined,
        apiType: isApiType(m.apiType) ? m.apiType : undefined,
        toolCalling: typeof m.toolCalling === "boolean" ? m.toolCalling : undefined,
        vision: typeof m.vision === "boolean" ? m.vision : undefined,
        thinking: typeof m.thinking === "boolean" ? m.thinking : undefined,
        streaming: typeof m.streaming === "boolean" ? m.streaming : undefined,
        maxInputTokens: typeof m.maxInputTokens === "number" ? m.maxInputTokens : undefined,
        maxOutputTokens: typeof m.maxOutputTokens === "number" ? m.maxOutputTokens : undefined,
        supportsReasoningEffort: Array.isArray(m.supportsReasoningEffort)
          ? m.supportsReasoningEffort.filter(isReasoningEffort)
          : undefined,
        zeroDataRetentionEnabled:
          typeof m.zeroDataRetentionEnabled === "boolean" ? m.zeroDataRetentionEnabled : undefined,
        requestHeaders:
          typeof m.requestHeaders === "object" && m.requestHeaders !== null
            ? (m.requestHeaders as Record<string, string>)
            : undefined,
      });
    }

    providers.push({
      vendor: p.vendor,
      name: p.name,
      apiType: isApiType(p.apiType) ? p.apiType : undefined,
      baseUrl: typeof p.baseUrl === "string" ? p.baseUrl : undefined,
      models,
      autoDiscover: typeof p.autoDiscover === "boolean" ? p.autoDiscover : undefined,
    });
  }
  return { schemaVersion: 1, providers };
}

function isApiType(v: unknown): v is ApiType {
  return v === "chat-completions" || v === "responses" || v === "messages";
}

function isReasoningEffort(v: unknown): v is ReasoningEffort {
  return v === "low" || v === "medium" || v === "high";
}

/**
 * Resolve a model id within a provider. Throws if not found.
 */
export function findModel(
  config: ChatLanguageModelsConfig,
  vendor: string,
  modelId: string,
): { provider: ChatProviderConfig; model: LanguageModelConfig } {
  const provider = config.providers.find((p) => p.vendor === vendor);
  if (!provider) {
    throw new Error(`Unknown vendor: ${vendor}`);
  }
  const model = provider.models.find((m) => m.id === modelId);
  if (!model) {
    throw new Error(`Unknown model "${modelId}" for vendor "${vendor}"`);
  }
  return { provider, model };
}
