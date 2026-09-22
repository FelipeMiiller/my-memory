import * as React from "react";
import { ChatSidebar } from "./chat-sidebar";
import { ChatThread } from "./chat-thread";
import { ChatInput } from "./chat-input";
import { ChatSettingsModal } from "./chat-settings-modal";
import { useChatStore } from "@/lib/chat/store";
import { chatApi } from "@/lib/chat/ipc";
import { buildSystemPrompt } from "@/lib/chat/prompts";
import type { ChatMessage } from "@/lib/chat/types";

/**
 * ChatView — root of the Chat tab.
 *
 * Composes sidebar + thread + input + settings modal. Owns the streaming
 * orchestration: wires IPC listeners (onDelta/onDone/onError) to the store.
 */

export function ChatView(): React.JSX.Element {
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const config = useChatStore((s) => s.config);
  const streaming = useChatStore((s) => s.streaming);
  const error = useChatStore((s) => s.error);
  const loadConversations = useChatStore((s) => s.loadConversations);
  const loadConfig = useChatStore((s) => s.loadConfig);
  const ensureActiveConversation = useChatStore((s) => s.ensureActiveConversation);
  const appendUserMessage = useChatStore((s) => s.appendUserMessage);
  const startAssistantMessage = useChatStore((s) => s.startAssistantMessage);
  const setStreamingRequestId = useChatStore((s) => s.setStreamingRequestId);
  const appendDelta = useChatStore((s) => s.appendDelta);
  const finalizeAssistantMessage = useChatStore((s) => s.finalizeAssistantMessage);
  const abortStreaming = useChatStore((s) => s.abortStreaming);
  const setError = useChatStore((s) => s.setError);
  const clearError = useChatStore((s) => s.clearError);

  const [settingsOpen, setSettingsOpen] = React.useState(false);

  // Bootstrap: load conversations + config from persistent stores.
  React.useEffect(() => {
    void loadConversations();
    void loadConfig();
  }, [loadConversations, loadConfig]);

  // Wire IPC streaming listeners once on mount.
  React.useEffect(() => {
    const unsubDelta = chatApi.onDelta((e) => {
      appendDelta(e.delta);
    });
    const unsubDone = chatApi.onDone(() => {
      void finalizeAssistantMessage();
    });
    const unsubError = chatApi.onError((e) => {
      void finalizeAssistantMessage();
      setError(e.error);
    });
    return () => {
      unsubDelta();
      unsubDone();
      unsubError();
    };
  }, [appendDelta, finalizeAssistantMessage, setError]);

  async function handleSend(text: string): Promise<void> {
    try {
      await ensureActiveConversation();
      appendUserMessage(text);
      startAssistantMessage();

      // Build messages array from the (now updated) conversation.
      const conv = useChatStore.getState().conversations.find(
        (c) => c.id === useChatStore.getState().activeConversationId,
      );
      if (!conv) return;

      // Run RAG search on the main process if enabled.
      let ragResults: ReadonlyArray<import("@/lib/chat/types").MemSearchResult> = [];
      if (config.useMemory) {
        try {
          const rag = await chatApi.memSearch(text, 5);
          ragResults = rag.results;
        } catch (err) {
          console.warn("[chat] RAG search failed:", err);
        }
      }

      const system = buildSystemPrompt(config, ragResults);
      const messages: ChatMessage[] = conv.messages;

      const { requestId } = await chatApi.send({
        conversationId: conv.id,
        messages,
        system,
      });
      setStreamingRequestId(requestId);
    } catch (err) {
      console.error("[chat] handleSend failed:", err);
      void finalizeAssistantMessage();
      setError({ kind: "unknown", message: String(err) });
    }
  }

  return (
    <div className="flex h-full w-full" data-testid="chat-view">
      {/* Sidebar — narrow, fixed-width */}
      <aside className="w-64 shrink-0" data-testid="chat-view-sidebar">
        <ChatSidebar onOpenSettings={() => setSettingsOpen(true)} />
      </aside>

      {/* Main chat area */}
      <main className="flex min-w-0 flex-1 flex-col" data-testid="chat-view-main">
        <div className="flex items-center justify-between border-b border-border bg-card/30 px-4 py-2">
          <div className="text-xs text-muted-foreground">
            {config.provider} · {config.model} ·{" "}
            {config.useMemory ? "com memória" : "sem memória"}
          </div>
          {error && (
            <button
              type="button"
              onClick={() => clearError()}
              className="rounded bg-destructive/15 px-2 py-0.5 text-[11px] text-destructive hover:bg-destructive/25"
              data-testid="chat-error-banner"
              title={error.message}
            >
              {error.kind}: {error.message.slice(0, 80)}
              <span className="ml-2 opacity-60">×</span>
            </button>
          )}
        </div>
        <ChatThread />
        <ChatInput
          streaming={streaming.kind === "streaming"}
          hasApiKey={config.hasApiKey}
          onSend={(t) => void handleSend(t)}
          onStop={() => void abortStreaming()}
        />
      </main>

      <ChatSettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />

      {/* Hidden — for test selectors */}
      <span className="sr-only" data-testid="chat-conv-count">
        {conversations.length} conversation(s) loaded
        {activeId ? "" : " (no active)"}
      </span>
    </div>
  );
}
