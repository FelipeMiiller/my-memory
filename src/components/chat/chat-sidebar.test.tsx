/**
 * ChatSidebar — vitest unit tests.
 *
 * Closes T11 from M1 retro (`feat-viewer-v3-m1-3-pane-layout/tasks.md`):
 *   - Render header (title + new/clear buttons + search input)
 *   - Click on `+` creates a conversation
 *   - Search input filters the conversation list by title (case-insensitive)
 *
 * Persistence + IPC layers are mocked because jsdom doesn't have IndexedDB
 * or the Electron bridge. The Zustand store is real (no mocking) so the
 * tests exercise the actual state flow that the UI consumes.
 */

import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { ChatConversation } from "@/lib/chat/types";
import { ChatSidebar } from "./chat-sidebar";

// Mock persistence so createConversation() doesn't try IndexedDB.
vi.mock("@/lib/chat/persistence", () => ({
  saveConversation: vi.fn().mockResolvedValue(undefined),
  deleteConversation: vi.fn().mockResolvedValue(undefined),
  listConversations: vi.fn().mockResolvedValue([]),
}));

// Mock IPC (not exercised by ChatSidebar but imported transitively by the store).
vi.mock("@/lib/chat/ipc", () => ({
  chatApi: {
    getConfig: vi.fn().mockResolvedValue({}),
    setConfig: vi.fn().mockResolvedValue({}),
    abort: vi.fn().mockResolvedValue(undefined),
  },
}));

import { useChatStore } from "@/lib/chat/store";

function makeConversation(
  partial: Partial<ChatConversation>,
): ChatConversation {
  return {
    id: partial.id ?? crypto.randomUUID(),
    title: partial.title ?? "Sample conversation",
    createdAt: partial.createdAt ?? Date.now(),
    updatedAt: partial.updatedAt ?? Date.now(),
    messages: partial.messages ?? [],
  };
}

function resetStore(): void {
  useChatStore.setState({
    conversations: [],
    activeConversationId: null,
    streaming: { kind: "idle" },
    error: null,
  });
}

describe("ChatSidebar", () => {
  beforeEach(() => {
    resetStore();
  });

  test("renders header with title, actions, and empty-state message", () => {
    render(<ChatSidebar />);

    // Header title (pt-BR default from chat.json sidebar.title).
    expect(screen.getByText("Conversas")).toBeInTheDocument();

    // Action buttons (data-testids from chat-sidebar.tsx).
    expect(screen.getByTestId("chat-sidebar-new")).toBeInTheDocument();
    expect(screen.getByTestId("chat-sidebar-refresh")).toBeInTheDocument();

    // Search input.
    expect(screen.getByTestId("chat-sidebar-search")).toBeInTheDocument();

    // Empty state when there are no conversations.
    expect(screen.getByText(/Nenhuma conversa ainda/i)).toBeInTheDocument();
  });

  test("clicking the new-conversation button creates a conversation", () => {
    render(<ChatSidebar />);

    // Baseline: 0 conversations in the store.
    expect(useChatStore.getState().conversations).toHaveLength(0);

    fireEvent.click(screen.getByTestId("chat-sidebar-new"));

    // After click: 1 conversation, "Nova conversa" title (default), and it
    // becomes the active one.
    const state = useChatStore.getState();
    expect(state.conversations).toHaveLength(1);
    expect(state.conversations[0]?.title).toBe("Nova conversa");
    expect(state.activeConversationId).toBe(state.conversations[0]?.id);
  });

  test("search input filters the conversation list by title (case-insensitive)", () => {
    // Seed three conversations with distinct titles + staggered updatedAt
    // so the most recent comes first in the default sort.
    const base = Date.now();
    useChatStore.setState({
      conversations: [
        makeConversation({
          title: "Filtros no Postgres",
          updatedAt: base + 3000,
        }),
        makeConversation({ title: "RRF vs BM25", updatedAt: base + 2000 }),
        makeConversation({
          title: "Árvore de decisão",
          updatedAt: base + 1000,
        }),
      ],
      activeConversationId: null,
    });

    render(<ChatSidebar />);

    // Baseline: all three titles visible.
    expect(screen.getByText("Filtros no Postgres")).toBeInTheDocument();
    expect(screen.getByText("RRF vs BM25")).toBeInTheDocument();
    expect(screen.getByText("Árvore de decisão")).toBeInTheDocument();

    // Type a needle that only matches one title (mixed case).
    fireEvent.change(screen.getByTestId("chat-sidebar-search"), {
      target: { value: "rRf" },
    });

    expect(screen.getByText("RRF vs BM25")).toBeInTheDocument();
    expect(screen.queryByText("Filtros no Postgres")).not.toBeInTheDocument();
    expect(screen.queryByText("Árvore de decisão")).not.toBeInTheDocument();

    // Empty needle restores full list.
    fireEvent.change(screen.getByTestId("chat-sidebar-search"), {
      target: { value: "" },
    });
    expect(screen.getByText("Filtros no Postgres")).toBeInTheDocument();
    expect(screen.getByText("RRF vs BM25")).toBeInTheDocument();
    expect(screen.getByText("Árvore de decisão")).toBeInTheDocument();
  });
});
