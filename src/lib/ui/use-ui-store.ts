import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

/**
 * Layout UI state (ADR-053) — M1 do Roadmap v3.
 *
 * Persisted to localStorage so pane sizes survive app restarts.
 * One source of truth for all visibility + size decisions; no prop drilling.
 */

export type PaneId = "explorer" | "editor" | "chat" | "dock";

interface PaneLayout {
  visible: boolean;
  /** Width as percentage of parent (0-100). For editor: computed. */
  widthPct: number;
}

interface DockLayout {
  visible: boolean;
  heightPct: number;
}

interface UiLayout {
  explorer: PaneLayout;
  chat: PaneLayout;
  dock: DockLayout;
}

export interface UiState {
  layout: UiLayout;
  /** Last focused pane (used for keyboard focus management). */
  focusedPane: PaneId;
  togglePane: (pane: PaneId) => void;
  resizePane: (pane: "explorer" | "chat", widthPct: number) => void;
  resizeDock: (heightPct: number) => void;
  setFocusedPane: (pane: PaneId) => void;
  resetLayout: () => void;
}

const DEFAULT_LAYOUT: UiLayout = {
  explorer: { visible: true, widthPct: 20 },
  chat: { visible: true, widthPct: 20 },
  dock: { visible: false, heightPct: 25 },
};

const STORAGE_KEY = "mem.ui-layout";

export const useUiStore = create<UiState>()(
  persist(
    (set) => ({
      layout: DEFAULT_LAYOUT,
      focusedPane: "editor",
      togglePane: (pane) => {
        set((s) => {
          if (pane === "dock") {
            return { layout: { ...s.layout, dock: { ...s.layout.dock, visible: !s.layout.dock.visible } } };
          }
          if (pane === "explorer") {
            return {
              layout: {
                ...s.layout,
                explorer: { ...s.layout.explorer, visible: !s.layout.explorer.visible },
              },
            };
          }
          if (pane === "chat") {
            return {
              layout: { ...s.layout, chat: { ...s.layout.chat, visible: !s.layout.chat.visible } },
            };
          }
          return s;
        });
      },
      resizePane: (pane, widthPct) => {
        const clamped = Math.max(10, Math.min(50, widthPct));
        set((s) => {
          if (pane === "explorer") {
            return { layout: { ...s.layout, explorer: { ...s.layout.explorer, widthPct: clamped } } };
          }
          return { layout: { ...s.layout, chat: { ...s.layout.chat, widthPct: clamped } } };
        });
      },
      resizeDock: (heightPct) => {
        const clamped = Math.max(10, Math.min(50, heightPct));
        set((s) => ({ layout: { ...s.layout, dock: { ...s.layout.dock, heightPct: clamped } } }));
      },
      setFocusedPane: (pane) => set({ focusedPane: pane }),
      resetLayout: () => set({ layout: DEFAULT_LAYOUT }),
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      // Only persist the layout (not the transient focusedPane).
      partialize: (state) => ({ layout: state.layout }),
    },
  ),
);

/**
 * Computes the editor pane width as the remaining percentage.
 * Used by WorkspaceShell when wiring up the horizontal resizable group.
 */
export function editorWidthPct(layout: UiLayout): number {
  const explorer = layout.explorer.visible ? layout.explorer.widthPct : 0;
  const chat = layout.chat.visible ? layout.chat.widthPct : 0;
  return Math.max(20, 100 - explorer - chat);
}
