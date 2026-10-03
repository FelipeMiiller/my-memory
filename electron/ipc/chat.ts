import { ipcMain, type BrowserWindow } from "electron";
import { randomUUID } from "node:crypto";
import { IPC_CHANNELS } from "@electron/constants";
import {
  loadConfig,
  saveConfig,
  publicView,
} from "@electron/services/chat/config";
import { streamChat } from "@electron/services/chat/llm";
import { memSearch } from "@electron/services/chat/mem-search";
import {
  loadModelsConfig,
  publicProvidersView,
  findModel,
} from "@electron/services/chat/models-config";
import { discoverProviderModels } from "@electron/services/chat/models-discover";

/**
 * IPC handlers for chat (ADR-051, ADR-052).
 *
 * Phase 2 (ADR-052): provider + model are now resolved from
 * `chatLanguageModels.json` via `loadModelsConfig()`. The renderer never
 * sends free-form provider/model strings — it sends `vendor` + `modelId`
 * and we validate against the loaded config.
 */

const MAX_MESSAGES = 100;
const MAX_CONTENT_LENGTH = 32_000;
const RAG_TOP_K = 5;
const RAG_CONTEXT_MAX_CHARS = 8_000;

const inflight = new Map<string, AbortController>();

interface ValidSendRequest {
  conversationId: string;
  messages: ChatMessage[];
  system?: string;
  vendor: string;
  modelId: string;
}

function validateSendRequest(req: unknown, models: ChatLanguageModelsConfig): ValidSendRequest | { error: ChatError } {
  if (typeof req !== "object" || req === null) {
    return { error: { kind: "invalid", message: "Payload must be an object." } };
  }
  const r = req as Partial<ValidSendRequest>;
  if (typeof r.conversationId !== "string" || r.conversationId.length === 0) {
    return { error: { kind: "invalid", message: "conversationId missing." } };
  }
  if (typeof r.vendor !== "string" || r.vendor.length === 0) {
    return { error: { kind: "invalid", message: "vendor missing." } };
  }
  if (typeof r.modelId !== "string" || r.modelId.length === 0) {
    return { error: { kind: "invalid", message: "modelId missing." } };
  }
  try {
    findModel(models, r.vendor, r.modelId);
  } catch (err) {
    return {
      error: {
        kind: "invalid",
        message: err instanceof Error ? err.message : "Unknown vendor/model",
      },
    };
  }
  if (!Array.isArray(r.messages) || r.messages.length === 0) {
    return { error: { kind: "invalid", message: "messages[] must be non-empty." } };
  }
  if (r.messages.length > MAX_MESSAGES) {
    return {
      error: { kind: "invalid", message: `Too many messages (max ${MAX_MESSAGES}).` },
    };
  }
  for (const m of r.messages) {
    if (typeof m !== "object" || m === null) {
      return { error: { kind: "invalid", message: "Invalid message." } };
    }
    const msg = m as Partial<ChatMessage>;
    if (typeof msg.id !== "string" || typeof msg.content !== "string") {
      return { error: { kind: "invalid", message: "Message needs id + content." } };
    }
    if (msg.content.length > MAX_CONTENT_LENGTH) {
      return {
        error: { kind: "invalid", message: `Message too long (max ${MAX_CONTENT_LENGTH} chars).` },
      };
    }
    if (msg.role !== "user" && msg.role !== "assistant" && msg.role !== "system") {
      return { error: { kind: "invalid", message: "Invalid role." } };
    }
  }
  if (r.system !== undefined && typeof r.system !== "string") {
    return { error: { kind: "invalid", message: "system must be a string." } };
  }
  return r as ValidSendRequest;
}

export function registerChatHandlers(getWindow: () => BrowserWindow | null): void {
  ipcMain.handle(IPC_CHANNELS.MEM_CHAT_CONFIG_GET, async (): Promise<ChatConfig> => {
    const c = await loadConfig();
    return publicView(c);
  });

  ipcMain.handle(
    IPC_CHANNELS.MEM_CHAT_CONFIG_SET,
    async (_event, update: unknown): Promise<ChatConfig> => {
      if (typeof update !== "object" || update === null) {
        throw { kind: "invalid", message: "update must be an object" };
      }
      const next = await saveConfig(update as ChatConfigUpdate);
      return publicView(next);
    },
  );

  ipcMain.handle(
    IPC_CHANNELS.MEM_CHAT_MEM_SEARCH,
    async (_event, payload: unknown): Promise<MemSearchResponse> => {
      if (typeof payload !== "object" || payload === null) {
        return { query: "", results: [] };
      }
      const p = payload as { query?: unknown; topK?: unknown };
      const query = typeof p.query === "string" ? p.query : "";
      const topK = typeof p.topK === "number" ? p.topK : RAG_TOP_K;
      return memSearch({ query, topK });
    },
  );

  ipcMain.handle(IPC_CHANNELS.MEM_CHAT_PROVIDERS_GET, async (): Promise<ChatProvidersResponse> => {
    const models = await loadModelsConfig();
    return {
      providers: publicProvidersView(models),
    };
  });

  ipcMain.handle(
    IPC_CHANNELS.MEM_CHAT_MODELS_DISCOVER,
    async (_event, payload: unknown): Promise<{ vendor: string; models: ChatModelSummary[] }> => {
      if (typeof payload !== "object" || payload === null) {
        throw { kind: "invalid", message: "payload must include vendor" };
      }
      const p = payload as { vendor?: unknown };
      if (typeof p.vendor !== "string") {
        throw { kind: "invalid", message: "vendor required" };
      }
      const models = await loadModelsConfig();
      const cfg = await loadConfig();
      const provider = models.providers.find((pr) => pr.vendor === p.vendor);
      if (!provider || !provider.autoDiscover) {
        return { vendor: p.vendor, models: [] };
      }
      const apiKey = cfg.apiKeys?.[p.vendor] ?? "";
      const discovered = await discoverProviderModels(provider, apiKey);
      return { vendor: p.vendor, models: discovered };
    },
  );

  ipcMain.handle(
    IPC_CHANNELS.MEM_CHAT_SEND,
    async (event, payload: unknown): Promise<ChatSendResponse> => {
      const models = await loadModelsConfig();
      const validated = validateSendRequest(payload, models);
      if ("error" in validated) {
        throw validated.error;
      }
      const req = validated;

      const { provider, model } = findModel(models, req.vendor, req.modelId);
      const config = await loadConfig();
      const apiKey = config.apiKeys?.[req.vendor] ?? "";

      const requestId = randomUUID();
      const controller = new AbortController();
      inflight.set(requestId, controller);

      let system = req.system ?? "";
      if (config.useMemory && req.messages.length > 0) {
        const lastUserMsg = [...req.messages].reverse().find((m) => m.role === "user");
        if (lastUserMsg) {
          try {
            const rag = await memSearch({ query: lastUserMsg.content, topK: RAG_TOP_K });
            const ragBlock = formatRagBlock(rag.results);
            if (ragBlock.length > 0) {
              system = `${system}${system ? "\n\n" : ""}${ragBlock}`;
            }
          } catch (err) {
            console.warn("[chat] RAG search failed:", err);
          }
        }
      }

      const aiMessages = req.messages.map((m) => ({ role: m.role, content: m.content }));

      void streamChat(
        {
          provider,
          model,
          apiKey,
          system,
          messages: aiMessages,
          signal: controller.signal,
        },
        {
          onDelta: (delta) => {
            event.sender.send(IPC_CHANNELS.MEM_CHAT_DELTA, {
              requestId,
              delta,
            } satisfies ChatDeltaEvent);
          },
          onDone: ({ totalChars, latencyMs }) => {
            event.sender.send(IPC_CHANNELS.MEM_CHAT_DONE, {
              requestId,
              totalChars,
              latencyMs,
            } satisfies ChatDoneEvent);
            inflight.delete(requestId);
          },
          onError: (error) => {
            event.sender.send(IPC_CHANNELS.MEM_CHAT_ERROR, {
              requestId,
              error,
            } satisfies ChatErrorEvent);
            inflight.delete(requestId);
          },
        },
      );

      return { requestId };
    },
  );

  ipcMain.handle(
    IPC_CHANNELS.MEM_CHAT_ABORT,
    async (_event, payload: unknown): Promise<{ aborted: boolean }> => {
      if (typeof payload !== "object" || payload === null) {
        return { aborted: false };
      }
      const p = payload as { requestId?: unknown };
      if (typeof p.requestId !== "string") {
        return { aborted: false };
      }
      const controller = inflight.get(p.requestId);
      if (!controller) return { aborted: false };
      controller.abort();
      inflight.delete(p.requestId);
      return { aborted: true };
    },
  );
}

function formatRagBlock(results: ReadonlyArray<MemSearchResult>): string {
  if (results.length === 0) return "";
  const lines = [
    "## Memória do vault (auto-injetada via `mem search`)",
    "Use estes trechos apenas se forem relevantes à pergunta do usuário. Cite a fonte quando usar.",
    "",
  ];
  let chars = 0;
  for (let i = 0; i < results.length; i++) {
    const r = results[i];
    const snippet = r.snippet.length > 400 ? `${r.snippet.slice(0, 400)}…` : r.snippet;
    const line = `[${i + 1}] ${r.title} (${r.path}, score=${r.score.toFixed(3)})\n${snippet}\n`;
    if (chars + line.length > RAG_CONTEXT_MAX_CHARS) break;
    lines.push(line);
    chars += line.length;
  }
  return lines.join("\n");
}
