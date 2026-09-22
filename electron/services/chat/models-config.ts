import { app } from "electron";
import { promises as fs } from "node:fs";
import path from "node:path";

/**
 * Reader/writer for `chatLanguageModels.json` (VS Code-style schema).
 *
 * Resolution order on load:
 *   1. `<userData>/chatLanguageModels.json` (user-edited)
 *   2. Bundled `resources/chatLanguageModels.default.json` (first boot)
 *   3. Hardcoded fallback in code (last resort)
 *
 * Fail-safe: if the user file is malformed, log a warning and return the
 * bundled defaults. Never throw — IPC callers should receive a valid config.
 */

const FILE_NAME = "chatLanguageModels.json";

function configPath(): string {
  return path.join(app.getPath("userData"), FILE_NAME);
}

function bundledPath(): string {
  // In dev, app.getAppPath() is the repo root. In packaged builds, electron-forge
  // copies resources/ to the asar at app.getAppPath()/resources/.
  return path.join(app.getAppPath(), "resources", FILE_NAME);
}

const FALLBACK: ChatLanguageModelsConfig = {
  schemaVersion: 1,
  providers: [
    {
      vendor: "anthropic",
      name: "Anthropic Claude",
      apiType: "messages",
      models: [],
    },
    {
      vendor: "openai",
      name: "OpenAI",
      apiType: "responses",
      models: [],
    },
  ],
};

/**
 * Validate raw JSON shape (subset of the parser in src/lib/chat/models-config.ts).
 * Mirrors that schema but kept here to avoid renderer↔main cross-imports.
 */
function parseConfig(raw: unknown): ChatLanguageModelsConfig {
  if (typeof raw !== "object" || raw === null) {
    throw new Error("chatLanguageModels.json: root must be an object");
  }
  const r = raw as Partial<ChatLanguageModelsConfig>;
  if (r.schemaVersion !== 1) {
    throw new Error(`chatLanguageModels.json: unsupported schemaVersion ${String(r.schemaVersion)}`);
  }
  if (!Array.isArray(r.providers)) {
    throw new Error("chatLanguageModels.json: providers must be an array");
  }
  return r as ChatLanguageModelsConfig;
}

/**
 * Find a model by vendor+id. Throws if not found.
 */
function findModel(c: ChatLanguageModelsConfig, vendor: string, modelId: string): {
  provider: ChatProviderConfig;
  model: LanguageModelConfig;
} {
  const provider = c.providers.find((p) => p.vendor === vendor);
  if (!provider) throw new Error(`Unknown vendor: ${vendor}`);
  const model = provider.models.find((m) => m.id === modelId);
  if (!model) throw new Error(`Unknown model "${modelId}" for vendor "${vendor}"`);
  return { provider, model };
}

// findModel is exported as runtime; ambient types come from electron/types.d.ts.
export { findModel };

export async function loadModelsConfig(): Promise<ChatLanguageModelsConfig> {
  // 1. User override
  try {
    const raw = await fs.readFile(configPath(), "utf8");
    return parseConfig(JSON.parse(raw));
  } catch (err: unknown) {
    if (!isNodeError(err) || err.code !== "ENOENT") {
      console.warn("[chat/models-config] user file malformed — falling back:", err);
    }
  }
  // 2. Bundled defaults
  try {
    const raw = await fs.readFile(bundledPath(), "utf8");
    const parsed = parseConfig(JSON.parse(raw));
    // First boot: copy bundled to userData so user can edit.
    await writeModelsConfig(parsed).catch(() => {});
    return parsed;
  } catch (err: unknown) {
    console.warn("[chat/models-config] bundled file unavailable — using fallback:", err);
  }
  // 3. Hardcoded
  return FALLBACK;
}

export async function writeModelsConfig(c: ChatLanguageModelsConfig): Promise<void> {
  const file = configPath();
  await fs.mkdir(path.dirname(file), { recursive: true });
  await fs.writeFile(file, JSON.stringify(c, null, 2), "utf8");
}

/**
 * Returns a sanitized view of the providers — strips any `apiKey` field if
 * accidentally present. Defensive: the renderer NEVER sees keys.
 */
export function publicProvidersView(c: ChatLanguageModelsConfig): ChatProviderConfig[] {
  return c.providers.map((p) => ({
    vendor: p.vendor,
    name: p.name,
    apiType: p.apiType,
    baseUrl: p.baseUrl,
    models: p.models,
    autoDiscover: p.autoDiscover,
  }));
}

function isNodeError(err: unknown): err is NodeJS.ErrnoException {
  return typeof err === "object" && err !== null && "code" in err;
}
