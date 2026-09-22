import { app } from "electron";
import { promises as fs } from "node:fs";
import path from "node:path";

/**
 * Chat config persistence (ADR-051, ADR-052).
 *
 * Stores per-vendor API keys + useMemory toggle. File lives in
 * `app.getPath('userData')/chat-config.json`.
 *
 * Phase 2 (ADR-052): keys moved from a single `apiKey` field to a
 * `apiKeys: Record<vendor, key>` map. Each vendor in
 * chatLanguageModels.json can have its own key.
 *
 * Fail-safe: malformed file → defaults + warning, never throw.
 */

const CONFIG_FILE = "chat-config.json";
const SCHEMA_VERSION = 2;

export interface PersistedChatConfig {
  schemaVersion: number;
  /** @deprecated kept for migration only — use apiKeys[vendor] */
  apiKey?: string;
  apiKeys: Record<string, string>;
  useMemory: boolean;
  activeVendor: string;
  activeModelId: string;
  updatedAt: number;
}

export const DEFAULT_CONFIG: Omit<PersistedChatConfig, "updatedAt"> = {
  schemaVersion: SCHEMA_VERSION,
  apiKeys: {},
  useMemory: true,
  activeVendor: "anthropic",
  activeModelId: "claude-sonnet-4-5",
};

function configPath(): string {
  return path.join(app.getPath("userData"), CONFIG_FILE);
}

export async function loadConfig(): Promise<PersistedChatConfig> {
  const file = configPath();
  try {
    const raw = await fs.readFile(file, "utf8");
    const parsed = JSON.parse(raw) as Partial<PersistedChatConfig>;
    return migrateAndValidate(parsed);
  } catch (err: unknown) {
    if (isNodeError(err) && err.code === "ENOENT") {
      return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
    }
    console.warn("[chat/config] failed to read config — falling back to defaults", err);
    return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
  }
}

function migrateAndValidate(parsed: Partial<PersistedChatConfig>): PersistedChatConfig {
  if (typeof parsed !== "object" || parsed === null) {
    return { ...DEFAULT_CONFIG, updatedAt: Date.now() };
  }
  // Schema 1 → 2 migration: move single apiKey into apiKeys[activeVendor]
  let apiKeys: Record<string, string> = {};
  if (typeof parsed.apiKeys === "object" && parsed.apiKeys !== null) {
    for (const [k, v] of Object.entries(parsed.apiKeys)) {
      if (typeof k === "string" && typeof v === "string") {
        apiKeys[k] = v;
      }
    }
  } else if (typeof parsed.apiKey === "string" && parsed.apiKey.length > 0) {
    const vendor = typeof parsed.activeVendor === "string" ? parsed.activeVendor : "anthropic";
    apiKeys[vendor] = parsed.apiKey;
  }

  return {
    schemaVersion: SCHEMA_VERSION,
    apiKeys,
    useMemory: typeof parsed.useMemory === "boolean" ? parsed.useMemory : DEFAULT_CONFIG.useMemory,
    activeVendor:
      typeof parsed.activeVendor === "string" ? parsed.activeVendor : DEFAULT_CONFIG.activeVendor,
    activeModelId:
      typeof parsed.activeModelId === "string"
        ? parsed.activeModelId
        : DEFAULT_CONFIG.activeModelId,
    updatedAt: typeof parsed.updatedAt === "number" ? parsed.updatedAt : Date.now(),
  };
}

export interface ChatConfigUpdate {
  apiKeys?: Record<string, string>;
  useMemory?: boolean;
  activeVendor?: string;
  activeModelId?: string;
}

export async function saveConfig(update: ChatConfigUpdate): Promise<PersistedChatConfig> {
  const current = await loadConfig();
  const next: PersistedChatConfig = {
    schemaVersion: SCHEMA_VERSION,
    apiKeys: update.apiKeys !== undefined ? { ...current.apiKeys, ...update.apiKeys } : current.apiKeys,
    useMemory: update.useMemory !== undefined ? update.useMemory : current.useMemory,
    activeVendor: update.activeVendor ?? current.activeVendor,
    activeModelId: update.activeModelId ?? current.activeModelId,
    updatedAt: Date.now(),
  };
  const file = configPath();
  await fs.mkdir(path.dirname(file), { recursive: true });
  await fs.writeFile(file, JSON.stringify(next, null, 2), "utf8");
  return next;
}

/**
 * Safe view for the renderer. Includes which provider is active + whether
 * each provider has a key configured, but NEVER the key itself.
 */
export interface ChatConfigPublic {
  activeVendor: string;
  activeModelId: string;
  useMemory: boolean;
  providers: Array<{ vendor: string; hasApiKey: boolean }>;
}

export function publicView(c: PersistedChatConfig, knownVendors: readonly string[] = []): ChatConfigPublic {
  return {
    activeVendor: c.activeVendor,
    activeModelId: c.activeModelId,
    useMemory: c.useMemory,
    providers: knownVendors.map((v) => ({ vendor: v, hasApiKey: (c.apiKeys[v] ?? "").length > 0 })),
  };
}

function isNodeError(err: unknown): err is NodeJS.ErrnoException {
  return typeof err === "object" && err !== null && "code" in err;
}
