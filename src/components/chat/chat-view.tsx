import {
  HardDrive,
  ListFilter,
  MessageSquare,
  PanelRight,
  Plus,
  Settings,
  ShieldCheck,
} from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { chatApi } from "@/lib/chat/ipc";
import { buildSystemPrompt } from "@/lib/chat/prompts";
import { useChatStore } from "@/lib/chat/store";
import type { ChatMessage } from "@/lib/chat/types";
import { ChatInput } from "./chat-input";
import { ChatSettingsModal } from "./chat-settings-modal";
import { ChatSidebar } from "./chat-sidebar";
import { ChatThread } from "./chat-thread";

/**
 * ChatView — root of the Chat tab.
 *
 * Layout (top → bottom):
 *
 *   ┌─ Chat header ──────────────────────────────────────────┐
 *   │ Chat · com memória  [☰ sessions] [+ new] [∨] [⚙] [☰] │
 *   ├─────────────────────────────────────────────────────────┤
 *   │ Sidebar (toggle) │ Thread                              │
 *   │  ● Sessions      │                                     │
 *   │  [search]        │                                     │
 *   │  ● ola           │                                     │
 *   │  ● feat:...      ├─────────────────────────────────────┤
 *   │                  │ Tip: ...                            │
 *   │                  │ [textarea]               [🎤] [▶]  │
 *   ├──────────────────┴─────────────────────────────────────┤
 *   │ 💬 Local · 🛡 Default permissions                     │
 *   └─────────────────────────────────────────────────────────┘
 *
 * The conversations sidebar is collapsible: `sidebarOpen` defaults to
 * `true` (matches the VS Code default with the secondary sidebar visible).
 * Click the `☰ sessions` button in the header to toggle.
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
  const [sidebarOpen, setSidebarOpen] = React.useState(true);

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

  const showTip = conversations.length === 0 || !activeId;

  // Container that hosts the conversations Sheet. The Sheet portals into
  // this node and uses `position: absolute` (instead of `fixed`) so the
  // slide-in stays inside the chat pane and doesn't overlay the whole
  // workspace window.
  const chatPaneRef = React.useRef<HTMLDivElement>(null);

  return (
    <div
      ref={chatPaneRef}
      className="relative flex h-full w-full flex-col"
      data-testid="chat-view"
    >
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
            onClick={() => setSidebarOpen((o) => !o)}
            aria-label={
              sidebarOpen ? "Fechar painel de conversas" : "Abrir conversas"
            }
            aria-pressed={sidebarOpen}
            title={
              sidebarOpen ? "Fechar painel de conversas" : "Abrir conversas"
            }
            data-testid="chat-header-toggle-sidebar"
            className="h-7 w-7"
          >
            <PanelRight className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleNewConversation}
            aria-label="Nova conversa"
            data-testid="chat-header-new"
            className="h-7 w-7"
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSettingsOpen(true)}
            aria-label="Configurações do chat"
            data-testid="chat-header-settings"
            className="h-7 w-7"
          >
            <Settings className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Mais ações"
            data-testid="chat-header-menu"
            className="h-7 w-7"
          >
            <ListFilter className="h-3.5 w-3.5" />
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

      <div className="flex min-h-0 flex-1">
        <div
          className="flex min-w-0 flex-1 flex-col"
          data-testid="chat-view-main"
        >
          <ChatThread />
          {showTip && (
            <div
              className="mx-3 mb-2 rounded border border-border/60 bg-card/40 px-3 py-2 text-[11px] text-muted-foreground"
              data-testid="chat-input-tip"
            >
              <strong className="text-foreground/80">Tip:</strong>{" "}
              {config.hasApiKey
                ? "Pergunte algo sobre o vault — respostas citam [[ADR-XXX]] e trechos."
                : "Adicione sua API key nas Configurações para começar a conversar."}
            </div>
          )}
          <ChatInput
            asrLocale={i18n.language.startsWith("pt") ? "pt-BR" : "en-US"}
            streaming={streaming.kind === "streaming"}
            hasApiKey={config.hasApiKey}
            onSend={(t) => void handleSend(t)}
            onStop={() => void abortStreaming()}
          />
        </div>
      </div>

      {/* Conversations slide-in panel (shadcn Sheet, constrained to this
          chat pane via the container ref + position="contained"). */}
      <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <SheetContent
          side="right"
          position="contained"
          container={chatPaneRef.current}
          className="w-80 gap-0 p-0 sm:max-w-sm"
          data-testid="chat-sessions-sheet"
        >
          <SheetHeader className="sr-only">
            <SheetTitle>Conversas</SheetTitle>
            <SheetDescription>
              Lista de conversas salvas neste workspace.
            </SheetDescription>
          </SheetHeader>
          <ChatSidebar
            onOpenSettings={() => {
              setSettingsOpen(true);
              setSidebarOpen(false);
            }}
            onClose={() => setSidebarOpen(false)}
          />
        </SheetContent>
      </Sheet>

      <footer
        className="flex items-center justify-between gap-3 border-t border-border bg-card/30 px-3 py-1 text-[11px] text-muted-foreground"
        data-testid="chat-status-bar"
      >
        <div className="flex items-center gap-1">
          <HardDrive className="h-3 w-3" aria-hidden />
          <span>Local</span>
        </div>
        <div className="flex items-center gap-1">
          <ShieldCheck className="h-3 w-3" aria-hidden />
          <span>Default permissions</span>
        </div>
      </footer>

      <ChatSettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />

      <span className="sr-only" data-testid="chat-conv-count">
        {conversations.length} conversation(s) loaded
        {activeId ? "" : " (no active)"}
      </span>
    </div>
  );
}
