import { Plus, RefreshCw, Search, Trash2 } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/lib/chat/store";
import { cn } from "@/utils/tailwind";

/**
 * ChatSidebar — list of conversations inside the chat sessions Sheet.
 *
 * VS Code Sessions-style layout (top → bottom) matching the photo:
 *   • Header row: "Conversas" title + [+] [↻] (new conversation, clear
 *     search filter)
 *   • Search input
 *   • Flat list of sessions, title + relative-time subtitle
 *
 * Closing the panel is the Sheet's job: shadcn's SheetContent renders an
 * own ✕ in the top-right corner and an overlay-click closes it. The chat
 * header's `PanelLeft` toggle reopens it. Settings lives in the chat
 * header (⚙ → ChatSettingsModal) — model config / API key / memory, no
 * need for a second entry point here.
 */

export function ChatSidebar(): React.JSX.Element {
  const { t } = useTranslation();
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const createConversation = useChatStore((s) => s.createConversation);
  const selectConversation = useChatStore((s) => s.selectConversation);
  const deleteConversation = useChatStore((s) => s.deleteConversation);
  const [search, setSearch] = React.useState("");

  const needle = search.trim().toLowerCase();
  const filtered = React.useMemo(() => {
    const sorted = [...conversations].sort((a, b) => b.updatedAt - a.updatedAt);
    if (needle === "") return sorted;
    return sorted.filter((c) => c.title.toLowerCase().includes(needle));
  }, [conversations, needle]);

  return (
    <div className="flex h-full flex-col">
      {/* Header: title + actions */}
      <div className="flex items-center justify-between border-b border-border px-3 py-1.5">
        <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
          {t("chat.sidebar.title")}
        </span>
        <div className="flex items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => void createConversation()}
            aria-label={t("chat.sidebar.newConversation")}
            data-testid="chat-sidebar-new"
            className="h-6 w-6"
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSearch("")}
            aria-label="Limpar busca"
            data-testid="chat-sidebar-refresh"
            className="h-6 w-6"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* Search bar */}
      <div
        className="flex items-center gap-1.5 border-b border-border px-2 py-1.5"
        data-testid="chat-sidebar-search-row"
      >
        <Search
          className="h-3.5 w-3.5 shrink-0 opacity-50"
          aria-hidden="true"
        />
        <Input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("chat.sidebar.searchPlaceholder")}
          className="h-7 border-0 bg-transparent px-0 text-xs shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          data-testid="chat-sidebar-search"
        />
      </div>

      <ScrollArea className="flex-1" data-testid="chat-sidebar-list">
        <nav className="flex flex-col gap-0.5 p-2">
          {filtered.length === 0 ? (
            <div className="px-2 py-4 text-center text-[11px] text-muted-foreground">
              {conversations.length === 0
                ? t("chat.sidebar.empty")
                : t("chat.sidebar.noMatch")}
            </div>
          ) : (
            filtered.map((c) => (
              <SessionItem
                key={c.id}
                title={c.title}
                updatedAt={c.updatedAt}
                active={activeId === c.id}
                onSelect={() => selectConversation(c.id)}
                onDelete={() => void deleteConversation(c.id)}
                deleteLabel={t("chat.sidebar.delete")}
              />
            ))
          )}
        </nav>
      </ScrollArea>
    </div>
  );
}

interface SessionItemProps {
  title: string;
  updatedAt: number;
  active: boolean;
  onSelect: () => void;
  onDelete: () => void;
  deleteLabel: string;
}

function SessionItem({
  title,
  updatedAt,
  active,
  onSelect,
  onDelete,
  deleteLabel,
}: SessionItemProps): React.JSX.Element {
  const relative = formatRelative(updatedAt);
  return (
    <div
      className={cn(
        "group flex items-start gap-1.5 rounded-md px-2 py-1.5 text-left text-sm transition-colors",
        "focus-within:ring-2 focus-within:ring-ring",
        active
          ? "bg-accent text-accent-foreground"
          : "hover:bg-accent/60 hover:text-accent-foreground",
      )}
      data-testid="chat-sidebar-item"
      data-active={active}
    >
      <button
        type="button"
        onClick={onSelect}
        className="flex min-w-0 flex-1 flex-col items-start text-left"
        title={title}
      >
        <span className="w-full truncate text-xs font-medium">{title}</span>
        <span className="mt-0.5 truncate text-[10px] text-muted-foreground">
          {relative}
        </span>
      </button>
      <button
        type="button"
        onClick={onDelete}
        aria-label={deleteLabel}
        className="invisible shrink-0 self-start pt-0.5 text-muted-foreground hover:text-destructive group-hover:visible"
        data-testid="chat-sidebar-delete"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

function formatRelative(ts: number): string {
  const diffMs = Date.now() - ts;
  const sec = Math.round(diffMs / 1000);
  if (sec < 45) return "agora";
  const min = Math.round(sec / 60);
  if (min < 60) return `${min} min atrás`;
  const hr = Math.round(min / 60);
  if (hr < 24) return `${hr} h atrás`;
  const day = Math.round(hr / 24);
  if (day < 7) return `${day} ${day === 1 ? "dia" : "dias"} atrás`;
  if (day < 30) {
    const w = Math.round(day / 7);
    return `${w} ${w === 1 ? "sem" : "sem"} atrás`;
  }
  if (day < 365) {
    const m = Math.round(day / 30);
    return `${m} ${m === 1 ? "mês" : "meses"} atrás`;
  }
  const y = Math.round(day / 365);
  return `${y} ${y === 1 ? "ano" : "anos"} atrás`;
}

export default ChatSidebar;
