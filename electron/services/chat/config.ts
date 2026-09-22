import { app } from "electron";
import { promises as fs } from "node:fs";
import path from "node:path";

/**
 * Chat config persistence (ADR-051).
 *
 * Stores user-chosen provider, model, and API key as JSON in
 * `app.getPath('userData')/chat-config.json`. This file lives in the user
 * profile (NOT in the app bundle), survives app updates, and is .gitignored
 * by default. API keys never travel to the renderer.
 *
 * Fail-safe: if the file is missing, malformed, or unreadable, we return
 * default config and log a warning. We never throw on read — the IPC handler
 * should return defaults so the UI can prompt the user to fill the key.
 */

const CONFIG_FILE = "chat-config.json";
const SCHEMA_VERSION = 1;

export interface PersistedChatConfig {
  schemaVersion: number;
  provider: ChatProvider;
  model: string;
  apiKey: string;
  useMemory: boolean;
  updatedAt: number;
}

export const DEFAULT_CONFIG: Omit<PersistedChatConfig, "updatedAt"> = {
  schemaVersion: SCHEMA_VERSION,
  provider: "anthropic",
  model: "claude-sonnet-4-5",
  apiKey: "",
  useMemory: true,
};

function configPath(): string {
  return path.join(app.getPath("userData"), CONFIG_FILE);
}

export async function loadConfig(): Promise<PersistedChatConfig> {
  const file = configPath();
  try {
    const raw = await fs.readFile(file, "utf8");
    const parsed = JSON.parse(raw) as Partial<PersistedChatConfig>;
    // Validate shape; fall back to defaults for any missing field.
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      typeof parsed.provider !== "string" ||
      typeof parsed.model !== "string" ||
      typeof parsed.apiKey !== "string" ||
      typeof parsed.useMemory !== "boolean"
    ) {
      console.warn("[chat/config] malformed config — falling back to defaults");
      return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
    }
    return {
      schemaVersion: SCHEMA_VERSION,
      provider: parsed.provider,
      model: parsed.model,
      apiKey: parsed.apiKey,
      useMemory: parsed.useMemory,
      updatedAt: typeof parsed.updatedAt === "number" ? parsed.updatedAt : Date.now(),
    };
  } catch (err: unknown) {
    if (isNodeError(err) && err.code === "ENOENT") {
      // First run — return defaults; do not write yet.
      return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
    }
    console.warn("[chat/config] failed to read config — falling back to defaults", err);
    return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
  }
}

export async function saveConfig(update: ChatConfigUpdate): Promise<PersistedChatConfig> {
  const current = await loadConfig();
  const next: PersistedChatConfig = {
    schemaVersion: SCHEMA_VERSION,
    provider: update.provider ?? current.provider,
    model: update.model ?? current.model,
    apiKey: update.apiKey !== undefined ? update.apiKey : current.apiKey,
    useMemory: update.useMemory !== undefined ? update.useMemory : current.useMemory,
    updatedAt: Date.now(),
  };
  const file = configPath();
  await fs.mkdir(path.dirname(file), { recursive: true });
  await fs.writeFile(file, JSON.stringify(next, null, 2), "utf8");
  return next;
}

/**
 * Returns the safe view of the config that the renderer is allowed to see.
 * NEVER includes the API key — only `hasApiKey: boolean`.
 */
export function publicView(c: PersistedChatConfig): ChatConfig {
  return {
    provider: c.provider,
    model: c.model,
    useMemory: c.useMemory,
    hasApiKey: c.apiKey.length > 0,
  };
}

function isNodeError(err: unknown): err is NodeJS.ErrnoException {
  return typeof err === "object" && err !== null && "code" in err;
}
