import {
  ListFilter,
  MessageSquare,
  Plus,
  Settings,
  Sparkles,
} from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { chatApi } from "@/lib/chat/ipc";
import { buildSystemPrompt } from "@/lib/chat/prompts";
import { useChatStore } from "@/lib/chat/store";
import type { ChatMessage } from "@/lib/chat/types";
import { ChatInput } from "./chat-input";
import { ChatModelPicker } from "./chat-model-picker";
import { ChatSettingsModal } from "./chat-settings-modal";
import { ChatSidebar } from "./chat-sidebar";
import { ChatThread } from "./chat-thread";

/**
 * ChatView — root of the Chat tab.
 *
 * Layout (top → bottom) matches the VS Code chat-with-secondary-sidebar
 * pattern shown in the photo:
 *
 *   ┌─ Chat header (title + 4 icon actions) ─────────────┐
 *   │ Chat                  [+ new] [∨ model] [⚙] [☰]  │
 *   ├─ Collapsible sessions section ─────────────────────┤
 *   │ ▾ Conversas (count)                              │
 *   │   ● ola · 1 mo ago                               │
 *   │   ● feat: ... · 4 mos ago                        │
 *   ├───────────────────────────────────────────────────┤
 *   │ [thread — messages render here]                  │
 *   ├───────────────────────────────────────────────────┤
 *   │ Tip: ...                                         │
 *   │ [textarea "Pergunte algo…"]                      │
 *   │ [🎤] [▶]                                         │
 *   └───────────────────────────────────────────────────┘
 */

export function ChatView(): React.JSX.Element {
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const config = useChatStore((s) => s.config);
  const streaming = useChatStore((s) => s.streaming);
  const error = useChatStore((s) => s.error);
  const loadConversations = useChatStore((s) => s.loadConversations);
  const loadConfig = useChatStore((s) => s.loadConfig);
  const ensureActiveConversation = useChatStore(
    (s) => s.ensureActiveConversation,
  );
  const appendUserMessage = useChatStore((s) => s.appendUserMessage);
  const startAssistantMessage = useChatStore((s) => s.startAssistantMessage);
  const setStreamingRequestId = useChatStore((s) => s.setStreamingRequestId);
  const appendDelta = useChatStore((s) => s.appendDelta);
  const finalizeAssistantMessage = useChatStore(
    (s) => s.finalizeAssistantMessage,
  );
  const abortStreaming = useChatStore((s) => s.abortStreaming);
  const setError = useChatStore((s) => s.setError);
  const clearError = useChatStore((s) => s.clearError);
  const createConversation = useChatStore((s) => s.createConversation);
  const { i18n } = useTranslation();

  const [settingsOpen, setSettingsOpen] = React.useState(false);
  const [sessionsOpen, setSessionsOpen] = React.useState(true);

  React.useEffect(() => {
    void loadConversations();
    void loadConfig();
  }, [loadConversations, loadConfig]);

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

      const conv = useChatStore
        .getState()
        .conversations.find(
          (c) => c.id === useChatStore.getState().activeConversationId,
        );
      if (!conv) return;

      let ragResults: ReadonlyArray<
        import("@/lib/chat/types").MemSearchResult
      > = [];
      if (config.useMemory) {
        try {
          const rag = await chatApi.memSearch(text);
          ragResults = rag.results;
        } catch (err) {
          console.warn("[chat] RAG search failed:", err);
        }
      }

      const messages: ReadonlyArray<ChatMessage> = conv.messages.map((m) => ({
        id: m.id,
        role: m.role,
        content: m.content,
        createdAt: m.createdAt,
      }));

      const system = buildSystemPrompt(config, ragResults);

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

  function handleNewConversation(): void {
    void createConversation();
  }

  return (
    <div className="flex h-full w-full flex-col" data-testid="chat-view">
      {/* Chat header: title + 4 icon actions */}
      <header
        className="flex items-center justify-between gap-2 border-b border-border bg-card/30 px-3 py-1.5"
        data-testid="chat-view-header"
      >
        <div className="flex items-center gap-2">
          <MessageSquare className="h-3.5 w-3.5 opacity-70" aria-hidden />
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Chat
          </span>
          <span className="text-[10px] text-muted-foreground">
            · {config.useMemory ? "com memória" : "sem memória"}
          </span>
        </div>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleNewConversation}
            aria-label="Nova conversa"
            data-testid="chat-header-new"
          >
            <Plus className="h-4 w-4" />
          </Button>
          {/* Compact model trigger — opens the same popover as the picker
              previously did in the input toolbar. */}
          <ChatModelPicker
            compact
            onOpenSettings={() => setSettingsOpen(true)}
          />
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSettingsOpen(true)}
            aria-label="Configurações do chat"
            data-testid="chat-header-settings"
          >
            <Settings className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Mais ações"
            data-testid="chat-header-menu"
          >
            <ListFilter className="h-4 w-4" />
          </Button>
        </div>
      </header>

      {error && (
        <button
          type="button"
          onClick={() => clearError()}
          className="border-b border-border bg-destructive/15 px-3 py-1 text-left text-[11px] text-destructive hover:bg-destructive/25"
          data-testid="chat-error-banner"
          title={error.message}
        >
          {error.kind}: {error.message.slice(0, 80)}
          <span className="ml-2 opacity-60">×</span>
        </button>
      )}

      {/* Collapsible sessions section — was the left <aside>; now sits at
          the top of the chat panel matching the VS Code layout. */}
      <section
        className="border-b border-border bg-card/10"
        data-testid="chat-sessions-section"
      >
        <button
          type="button"
          onClick={() => setSessionsOpen((o) => !o)}
          aria-expanded={sessionsOpen}
          className="flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left hover:bg-accent/20"
          data-testid="chat-sessions-toggle"
        >
          <span className="flex items-center gap-2">
            <Sparkles className="h-3 w-3 opacity-50" aria-hidden />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              Conversas
            </span>
            <span className="rounded bg-muted/40 px-1 text-[10px] tabular-nums text-muted-foreground">
              {conversations.length}
            </span>
          </span>
          <span className="text-[10px] text-muted-foreground">
            {sessionsOpen ? "fechar" : "abrir"}
          </span>
        </button>
        {sessionsOpen && (
          <div
            className="max-h-[240px] overflow-y-auto"
            data-testid="chat-sessions-body"
          >
            <ChatSidebar onOpenSettings={() => setSettingsOpen(true)} />
          </div>
        )}
      </section>

      {/* Thread + input — main chat surface */}
      <div
        className="flex min-h-0 flex-1 flex-col"
        data-testid="chat-view-main"
      >
        <ChatThread />
        <ChatInput
          asrLocale={i18n.language.startsWith("pt") ? "pt-BR" : "en-US"}
          streaming={streaming.kind === "streaming"}
          hasApiKey={config.hasApiKey}
          onSend={(t) => void handleSend(t)}
          onStop={() => void abortStreaming()}
        />
      </div>

      <ChatSettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />

      {/* Hidden — for test selectors */}
      <span className="sr-only" data-testid="chat-conv-count">
        {conversations.length} conversation(s) loaded
        {activeId ? "" : " (no active)"}
      </span>
    </div>
  );
}
