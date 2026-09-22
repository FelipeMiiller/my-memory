import { Plus, Settings, Trash2 } from "lucide-react";
import type * as React from "react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/lib/chat/store";
import { cn } from "@/utils/tailwind";

/**
 * ChatSidebar — list of conversations + new chat button + settings trigger.
 * The settings modal itself is owned by the parent ChatView.
 */

interface ChatSidebarProps {
  onOpenSettings: () => void;
}

export function ChatSidebar({
  onOpenSettings,
}: ChatSidebarProps): React.JSX.Element {
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const createConversation = useChatStore((s) => s.createConversation);
  const selectConversation = useChatStore((s) => s.selectConversation);
  const deleteConversation = useChatStore((s) => s.deleteConversation);

  return (
    <div className="flex h-full flex-col border-r border-border bg-card/20">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Conversas
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={onOpenSettings}
            aria-label="Configurações do chat"
            data-testid="chat-sidebar-settings"
          >
            <Settings className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => void createConversation()}
            aria-label="Nova conversa"
            data-testid="chat-sidebar-new"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <ScrollArea className="flex-1" data-testid="chat-sidebar-list">
        <nav className="flex flex-col gap-1 p-2">
          {conversations.length === 0 ? (
            <div className="px-2 py-4 text-center text-[11px] text-muted-foreground">
              Nenhuma conversa ainda.
              <br />
              Clique em <strong>+</strong> para começar.
            </div>
          ) : (
            conversations.map((c) => (
              <div
                key={c.id}
                className={cn(
                  "group flex items-center justify-between gap-1 rounded-md px-2 py-1.5 text-left text-sm transition-colors",
                  "focus-within:ring-2 focus-within:ring-ring",
                  activeId === c.id
                    ? "bg-accent text-accent-foreground"
                    : "hover:bg-accent/60 hover:text-accent-foreground",
                )}
                data-testid="chat-sidebar-item"
                data-active={activeId === c.id}
              >
                <button
                  type="button"
                  onClick={() => selectConversation(c.id)}
                  className="flex-1 truncate text-left"
                  title={c.title}
                >
                  {c.title}
                </button>
                <button
                  type="button"
                  onClick={() => void deleteConversation(c.id)}
                  aria-label="Excluir conversa"
                  className="invisible text-muted-foreground hover:text-destructive group-hover:visible"
                  data-testid="chat-sidebar-delete"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))
          )}
        </nav>
      </ScrollArea>
    </div>
  );
}
