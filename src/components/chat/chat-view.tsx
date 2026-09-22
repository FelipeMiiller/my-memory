import * as React from "react";
import { ChatSidebar } from "./chat-sidebar";
import { ChatThread } from "./chat-thread";
import { ChatInput } from "./chat-input";
import { ChatSettingsModal } from "./chat-settings-modal";
import { useChatStore } from "@/lib/chat/store";
import { chatApi } from "@/lib/chat/ipc";
import { buildSystemPrompt } from "@/lib/chat/prompts";
import type { ChatMessage, MemSearchResult } from "@/lib/chat/types";

/**
 * ChatView — root of the Chat tab (ADR-051, ADR-052).
 *
 * Phase 2: providers come from chatLanguageModels.json (loaded on mount).
 * Sends use { vendor, modelId } resolved from the active config.
 */

export function ChatView(): React.JSX.Element {
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const config = useChatStore((s) => s.config);
  const streaming = useChatStore((s) => s.streaming);
  const error = useChatStore((s) => s.error);
  const loadConversations = useChatStore((s) => s.loadConversations);
  const loadConfig = useChatStore((s) => s.loadConfig);
  const loadProviders = useChatStore((s) => s.loadProviders);
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

  React.useEffect(() => {
    void loadConversations();
    void loadConfig();
    void loadProviders();
  }, [loadConversations, loadConfig, loadProviders]);

  React.useEffect(() => {
    const unsubDelta = chatApi.onDelta((e) => appendDelta(e.delta));
    const unsubDone = chatApi.onDone(() => void finalizeAssistantMessage());
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

  const activeHasKey =
    config.providers.find((p) => p.vendor === config.activeVendor)?.hasApiKey ?? false;

  async function handleSend(text: string): Promise<void> {
    try {
      await ensureActiveConversation();
      appendUserMessage(text);
      startAssistantMessage();

      const conv = useChatStore.getState().conversations.find(
        (c) => c.id === useChatStore.getState().activeConversationId,
      );
      if (!conv) return;

      let ragResults: ReadonlyArray<MemSearchResult> = [];
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
        vendor: config.activeVendor,
        modelId: config.activeModelId,
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
      <aside className="w-64 shrink-0" data-testid="chat-view-sidebar">
        <ChatSidebar onOpenSettings={() => setSettingsOpen(true)} />
      </aside>
      <main className="flex min-w-0 flex-1 flex-col" data-testid="chat-view-main">
        <div className="flex items-center justify-between border-b border-border bg-card/30 px-4 py-2">
          <div className="text-xs text-muted-foreground">
            {config.activeVendor} · {config.activeModelId}
            {config.useMemory ? " · com memória" : " · sem memória"}
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
          hasApiKey={activeHasKey}
          onSend={(t) => void handleSend(t)}
          onStop={() => void abortStreaming()}
        />
      </main>
      <ChatSettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <span className="sr-only" data-testid="chat-conv-count">
        {conversations.length} conversation(s) loaded
        {activeId ? "" : " (no active)"}
      </span>
    </div>
  );
}
