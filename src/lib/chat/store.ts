import { create } from "zustand";
import type {
  ChatConfig,
  ChatConversation,
  ChatError,
  ChatMessage,
  ChatProviderConfig,
  LanguageModelConfig,
} from "@/lib/chat/types";
import {
  listConversations,
  saveConversation,
  deleteConversation as deleteConv,
} from "@/lib/chat/persistence";
import { chatApi } from "@/lib/chat/ipc";

/**
 * Zustand store (ADR-051, ADR-052).
 *
 * Phase 2: providers come from chatLanguageModels.json (dynamic). Store
 * tracks activeVendor + activeModelId. Active provider's `hasApiKey` is
 * derived from config.providers list (set by main, never the key itself).
 */

export type StreamingState =
  | { kind: "idle" }
  | { kind: "streaming"; requestId: string; assistantMessageId: string }
  | { kind: "aborted"; assistantMessageId: string };

interface ChatState {
  conversations: ChatConversation[];
  activeConversationId: string | null;
  streaming: StreamingState;
  config: ChatConfig;
  providers: ChatProviderConfig[];
  error: ChatError | null;

  loadConversations: () => Promise<void>;
  loadProviders: () => Promise<void>;
  discoverModels: (vendor: string) => Promise<void>;
  createConversation: () => Promise<string>;
  ensureActiveConversation: () => Promise<string>;
  selectConversation: (id: string) => void;
  deleteConversation: (id: string) => Promise<void>;
  renameConversation: (id: string, title: string) => Promise<void>;

  appendUserMessage: (text: string) => string;
  startAssistantMessage: () => string;
  setStreamingRequestId: (requestId: string) => void;
  appendDelta: (text: string) => void;
  finalizeAssistantMessage: () => Promise<void>;
  abortStreaming: () => Promise<void>;

  loadConfig: () => Promise<void>;
  setConfig: (update: Partial<ChatConfig>) => Promise<void>;

  setError: (e: ChatError | null) => void;
  clearError: () => void;
}

const CONV_TITLE_MAX = 60;
const FALLBACK_CONFIG: ChatConfig = {
  activeVendor: "anthropic",
  activeModelId: "claude-sonnet-4-5",
  useMemory: true,
  providers: [],
};

function generateId(): string {
  return (
    globalThis.crypto?.randomUUID?.() ??
    `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  );
}

function deriveTitle(firstUserMessage: string): string {
  const trimmed = firstUserMessage.trim().replace(/\s+/g, " ");
  if (trimmed.length === 0) return "Nova conversa";
  return trimmed.length > CONV_TITLE_MAX ? `${trimmed.slice(0, CONV_TITLE_MAX)}…` : trimmed;
}

async function persistActive(get: () => ChatState): Promise<void> {
  const id = get().activeConversationId;
  const conv = get().conversations.find((c) => c.id === id);
  if (!conv) return;
  await saveConversation(conv);
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  activeConversationId: null,
  streaming: { kind: "idle" },
  config: FALLBACK_CONFIG,
  providers: [],
  error: null,

  loadConversations: async () => {
    const list = await listConversations();
    set({ conversations: list });
  },

  loadProviders: async () => {
    try {
      const res = await chatApi.getProviders();
      set({ providers: res.providers });
    } catch (err) {
      console.warn("[chat/store] loadProviders failed:", err);
    }
  },

  discoverModels: async (vendor) => {
    try {
      const res = await chatApi.discoverModels(vendor);
      set((s) => ({
        providers: s.providers.map((p) =>
          p.vendor === vendor ? { ...p, models: res.models.map(toLanguageModelConfig) } : p,
        ),
      }));
    } catch (err) {
      console.warn(`[chat/store] discoverModels(${vendor}) failed:`, err);
    }
  },

  createConversation: async () => {
    const id = generateId();
    const now = Date.now();
    const conv: ChatConversation = {
      id,
      title: "Nova conversa",
      createdAt: now,
      updatedAt: now,
      messages: [],
    };
    set((s) => ({
      conversations: [conv, ...s.conversations],
      activeConversationId: id,
      streaming: { kind: "idle" },
    }));
    await saveConversation(conv);
    return id;
  },

  ensureActiveConversation: async () => {
    const id = get().activeConversationId;
    if (id) return id;
    return get().createConversation();
  },

  selectConversation: (id) => {
    set({ activeConversationId: id, streaming: { kind: "idle" } });
  },

  deleteConversation: async (id) => {
    await deleteConv(id);
    set((s) => {
      const remaining = s.conversations.filter((c) => c.id !== id);
      const nextActive =
        s.activeConversationId === id ? (remaining[0]?.id ?? null) : s.activeConversationId;
      return { conversations: remaining, activeConversationId: nextActive };
    });
  },

  renameConversation: async (id, title) => {
    const conv = get().conversations.find((c) => c.id === id);
    if (!conv) return;
    const updated: ChatConversation = { ...conv, title, updatedAt: Date.now() };
    set((s) => ({ conversations: s.conversations.map((c) => (c.id === id ? updated : c)) }));
    await saveConversation(updated);
  },

  appendUserMessage: (text) => {
    const id = generateId();
    const convId = get().activeConversationId;
    if (!convId) throw new Error("No active conversation — call ensureActiveConversation() first.");
    const msg: ChatMessage = {
      id,
      role: "user",
      content: text,
      createdAt: Date.now(),
    };
    set((s) => ({
      conversations: s.conversations.map((c) =>
        c.id === convId
          ? {
              ...c,
              messages: [...c.messages, msg],
              updatedAt: Date.now(),
              title: c.messages.length === 0 ? deriveTitle(text) : c.title,
            }
          : c,
      ),
    }));
    void persistActive(get);
    return id;
  },

  startAssistantMessage: () => {
    const id = generateId();
    const convId = get().activeConversationId;
    if (!convId) throw new Error("No active conversation — call ensureActiveConversation() first.");
    const msg: ChatMessage = { id, role: "assistant", content: "", createdAt: Date.now() };
    set((s) => ({
      conversations: s.conversations.map((c) =>
        c.id === convId
          ? { ...c, messages: [...c.messages, msg], updatedAt: Date.now() }
          : c,
      ),
      streaming: { kind: "streaming", requestId: "", assistantMessageId: id },
    }));
    return id;
  },

  setStreamingRequestId: (requestId) => {
    set((s) => (s.streaming.kind !== "streaming" ? s : { streaming: { ...s.streaming, requestId } }));
  },

  appendDelta: (text) => {
    set((s) => {
      if (s.streaming.kind !== "streaming") return s;
      const assistantMessageId = s.streaming.assistantMessageId;
      return {
        conversations: s.conversations.map((c) =>
          c.id === s.activeConversationId
            ? {
                ...c,
                messages: c.messages.map((m) =>
                  m.id === assistantMessageId ? { ...m, content: m.content + text } : m,
                ),
              }
            : c,
        ),
      };
    });
  },

  finalizeAssistantMessage: async () => {
    set({ streaming: { kind: "idle" } });
    await persistActive(get);
  },

  abortStreaming: async () => {
    const s = get();
    if (s.streaming.kind !== "streaming") return;
    const { requestId, assistantMessageId } = s.streaming;
    if (requestId) {
      try {
        await chatApi.abort(requestId);
      } catch (err) {
        console.warn("[chat/store] abort failed:", err);
      }
    }
    set({ streaming: { kind: "aborted", assistantMessageId } });
    await persistActive(get);
    set({ streaming: { kind: "idle" } });
  },

  loadConfig: async () => {
    try {
      const cfg = await chatApi.getConfig();
      set({ config: cfg });
    } catch (err) {
      console.warn("[chat/store] loadConfig failed:", err);
    }
  },

  setConfig: async (update) => {
    try {
      // Map the partial ChatConfig to ChatConfigUpdate shape for IPC.
      const ipcUpdate = {
        activeVendor: update.activeVendor,
        activeModelId: update.activeModelId,
        useMemory: update.useMemory,
        apiKeys: (update as { apiKeys?: Record<string, string> }).apiKeys,
      };
      const cfg = await chatApi.setConfig(ipcUpdate);
      set({ config: cfg });
    } catch (err) {
      console.warn("[chat/store] setConfig failed:", err);
      set({ error: { kind: "unknown", message: String(err) } });
    }
  },

  setError: (e) => set({ error: e }),
  clearError: () => set({ error: null }),
}));

function toLanguageModelConfig(s: { id: string; name?: string }): LanguageModelConfig {
  return { id: s.id, name: s.name ?? s.id };
}
