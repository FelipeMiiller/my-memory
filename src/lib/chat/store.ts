import { create } from "zustand";
import type {
  ChatConfig,
  ChatConversation,
  ChatError,
  ChatMessage,
} from "@/lib/chat/types";
import { DEFAULT_CHAT_CONFIG } from "@/lib/chat/types";
import {
  listConversations,
  saveConversation,
  deleteConversation as deleteConv,
} from "@/lib/chat/persistence";
import { chatApi } from "@/lib/chat/ipc";

/**
 * Zustand store (ADR-051).
 *
 * Single source of truth for chat UI state. Persists conversations to
 * IndexedDB on every mutation. Holds the streaming requestId and the
 * in-flight assistant message id.
 *
 * Notes:
 * - API keys NEVER live here. Only `config.hasApiKey: boolean` (set by main).
 * - Streaming state is a small tagged union so React components can pattern-match.
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
  error: ChatError | null;

  // Conversations
  loadConversations: () => Promise<void>;
  createConversation: () => Promise<string>;
  ensureActiveConversation: () => Promise<string>;
  selectConversation: (id: string) => void;
  deleteConversation: (id: string) => Promise<void>;
  renameConversation: (id: string, title: string) => Promise<void>;

  // Messages
  appendUserMessage: (text: string) => string;
  startAssistantMessage: () => string;
  setStreamingRequestId: (requestId: string) => void;
  appendDelta: (text: string) => void;
  finalizeAssistantMessage: () => Promise<void>;
  abortStreaming: () => Promise<void>;

  // Config
  loadConfig: () => Promise<void>;
  setConfig: (update: Partial<ChatConfig>) => Promise<void>;

  // Errors
  setError: (e: ChatError | null) => void;
  clearError: () => void;
}

const CONV_TITLE_MAX = 60;

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

function mutateActiveConversation(
  set: (partial: (s: ChatState) => Partial<ChatState>) => void,
  get: () => ChatState,
  fn: (c: ChatConversation) => ChatConversation,
): void {
  const id = get().activeConversationId;
  if (!id) return;
  set((s) => ({
    conversations: s.conversations.map((c) => (c.id === id ? fn(c) : c)),
  }));
  void persistActive(get);
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
  config: DEFAULT_CHAT_CONFIG,
  error: null,

  loadConversations: async () => {
    const list = await listConversations();
    set({ conversations: list });
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
    set({
      activeConversationId: id,
      streaming: { kind: "idle" },
    });
  },

  deleteConversation: async (id) => {
    await deleteConv(id);
    set((s) => {
      const remaining = s.conversations.filter((c) => c.id !== id);
      const nextActive =
        s.activeConversationId === id ? (remaining[0]?.id ?? null) : s.activeConversationId;
      return {
        conversations: remaining,
        activeConversationId: nextActive,
      };
    });
  },

  renameConversation: async (id, title) => {
    const conv = get().conversations.find((c) => c.id === id);
    if (!conv) return;
    const updated: ChatConversation = { ...conv, title, updatedAt: Date.now() };
    set((s) => ({
      conversations: s.conversations.map((c) => (c.id === id ? updated : c)),
    }));
    await saveConversation(updated);
  },

  appendUserMessage: (text) => {
    const id = generateId();
    const convId = get().activeConversationId;
    if (!convId) {
      // Defensive — caller should call ensureActiveConversation first.
      throw new Error("No active conversation — call ensureActiveConversation() first.");
    }
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
    if (!convId) {
      throw new Error("No active conversation — call ensureActiveConversation() first.");
    }
    const msg: ChatMessage = {
      id,
      role: "assistant",
      content: "",
      createdAt: Date.now(),
    };
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
    set((s) => {
      if (s.streaming.kind !== "streaming") return s;
      return {
        streaming: { ...s.streaming, requestId },
      };
    });
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
                  m.id === assistantMessageId
                    ? { ...m, content: m.content + text }
                    : m,
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
    // Keep partial content; mark aborted then immediately return to idle.
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
      const cfg = await chatApi.setConfig(update);
      set({ config: cfg });
    } catch (err) {
      console.warn("[chat/store] setConfig failed:", err);
      set({ error: { kind: "unknown", message: String(err) } });
    }
  },

  setError: (e) => set({ error: e }),
  clearError: () => set({ error: null }),
}));
