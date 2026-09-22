import { Plus, Search, Settings, Trash2 } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/lib/chat/store";
import type { ChatConversation } from "@/lib/chat/types";
import { cn } from "@/utils/tailwind";

/**
 * ChatSidebar — list of conversations + new chat + settings (ADR-051 + M1 polish).
 *
 * VS Code Sessions-style layout (top → bottom):
 *   • Header row: title + new conversation + settings
 *   • Search input
 *   • Grouped list: "Newer" (last 30 days) → "Older"
 *
 * Each item shows title + relative time (`updatedAt`) as the subtitle, matching
 * the VS Code pattern where multi-user diff stats would appear (we don't have
 * multi-user/git diff in this build — just time).
 */

interface ChatSidebarProps {
  onOpenSettings: () => void;
}

const NEWER_THRESHOLD_MS = 30 * 24 * 60 * 60 * 1000; // 30 days

export function ChatSidebar({
  onOpenSettings,
}: ChatSidebarProps): React.JSX.Element {
  const { t } = useTranslation();
  const conversations = useChatStore((s) => s.conversations);
  const activeId = useChatStore((s) => s.activeConversationId);
  const createConversation = useChatStore((s) => s.createConversation);
  const selectConversation = useChatStore((s) => s.selectConversation);
  const deleteConversation = useChatStore((s) => s.deleteConversation);
  const [search, setSearch] = React.useState("");

  const needle = search.trim().toLowerCase();
  const filtered = React.useMemo(() => {
    if (needle === "") return conversations;
    return conversations.filter((c) => c.title.toLowerCase().includes(needle));
  }, [conversations, needle]);

  const { newer, older } = React.useMemo(() => {
    const cutoff = Date.now() - NEWER_THRESHOLD_MS;
    const newer: ChatConversation[] = [];
    const older: ChatConversation[] = [];
    for (const c of filtered) {
      if (c.updatedAt >= cutoff) newer.push(c);
      else older.push(c);
    }
    // Most-recent first in each bucket.
    newer.sort((a, b) => b.updatedAt - a.updatedAt);
    older.sort((a, b) => b.updatedAt - a.updatedAt);
    return { newer, older };
  }, [filtered]);

  return (
    <div className="flex h-full flex-col border-r border-border bg-card/20">
      <div className="flex items-center justify-between border-b border-border px-3 py-2">
        <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {t("chat.sidebar.title")}
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={onOpenSettings}
            aria-label={t("chat.sidebar.settingsTitle")}
            data-testid="chat-sidebar-settings"
          >
            <Settings className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => void createConversation()}
            aria-label={t("chat.sidebar.newConversation")}
            data-testid="chat-sidebar-new"
          >
            <Plus className="h-4 w-4" />
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
            <>
              {newer.length > 0 && (
                <Group
                  label={t("chat.sidebar.newer")}
                  count={newer.length}
                  testId="chat-sidebar-group-newer"
                >
                  {newer.map((c) => (
                    <SessionItem
                      key={c.id}
                      conv={c}
                      active={activeId === c.id}
                      onSelect={() => selectConversation(c.id)}
                      onDelete={() => void deleteConversation(c.id)}
                      deleteLabel={t("chat.sidebar.delete")}
                    />
                  ))}
                </Group>
              )}
              {older.length > 0 && (
                <Group
                  label={t("chat.sidebar.older")}
                  count={older.length}
                  testId="chat-sidebar-group-older"
                >
                  {older.map((c) => (
                    <SessionItem
                      key={c.id}
                      conv={c}
                      active={activeId === c.id}
                      onSelect={() => selectConversation(c.id)}
                      onDelete={() => void deleteConversation(c.id)}
                      deleteLabel={t("chat.sidebar.delete")}
                    />
                  ))}
                </Group>
              )}
            </>
          )}
        </nav>
      </ScrollArea>
    </div>
  );
}

interface GroupProps {
  label: string;
  count: number;
  testId: string;
  children: React.ReactNode;
}

function Group({
  label,
  count,
  testId,
  children,
}: GroupProps): React.JSX.Element {
  return (
    <div className="mt-1 first:mt-0" data-testid={testId}>
      <div className="flex items-center justify-between px-2 py-1">
        <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          {label}
        </span>
        <span className="rounded bg-muted/40 px-1 text-[10px] tabular-nums text-muted-foreground">
          {count}
        </span>
      </div>
      <div className="flex flex-col gap-0.5">{children}</div>
    </div>
  );
}

interface SessionItemProps {
  conv: ChatConversation;
  active: boolean;
  onSelect: () => void;
  onDelete: () => void;
  deleteLabel: string;
}

function SessionItem({
  conv,
  active,
  onSelect,
  onDelete,
  deleteLabel,
}: SessionItemProps): React.JSX.Element {
  const relative = formatRelative(conv.updatedAt);
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
        title={conv.title}
      >
        <span className="w-full truncate text-xs font-medium">
          {conv.title}
        </span>
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
