import type * as React from "react";
import { ChatView } from "@/components/chat";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  DockPlaceholder,
  EditorPane,
  FileExplorerStub,
} from "@/components/workspace";
import { useShortcut } from "@/hooks/use-shortcut";
import { editorWidthPct, useUiStore } from "@/lib/ui/use-ui-store";
import { StatusBar } from "./status-bar";
import { TopBar } from "./top-bar";

/**
 * WorkspaceShell — 3-pane layout + dock (ADR-053, M1).
 *
 * Layout structure (root):
 *   - TopBar
 *   - ResizablePanelGroup vertical:
 *       - main content (ResizablePanelGroup horizontal):
 *           - FileExplorer (left)
 *           - EditorPane (center)
 *           - ChatView (right)
 *       - DockPlaceholder (bottom, optional)
 *   - StatusBar
 *
 * Pane visibility comes from useUiStore (persisted to localStorage).
 * Keyboard shortcuts: Ctrl+B (explorer), Ctrl+J (chat), Ctrl+` (dock).
 */

export function WorkspaceShell(): React.JSX.Element {
  const layout = useUiStore((s) => s.layout);
  const togglePane = useUiStore((s) => s.togglePane);

  useShortcut("Ctrl+B", { onMatch: () => togglePane("explorer") });
  useShortcut("Ctrl+J", { onMatch: () => togglePane("chat") });
  useShortcut("Ctrl+`", { onMatch: () => togglePane("dock") });

  return (
    <div
      className="flex h-screen w-screen flex-col overflow-hidden bg-background"
      data-testid="workspace-shell"
    >
      <TopBar />
      <ResizablePanelGroup direction="vertical" className="flex-1">
        <ResizablePanel
          defaultSize={layout.dock.visible ? 75 : 100}
          minSize={40}
        >
          {/* Horizontal 3-pane content */}
          <ResizablePanelGroup
            direction="horizontal"
            autoSaveId="workspace-horizontal"
          >
            {layout.explorer.visible && (
              <>
                <ResizablePanel
                  defaultSize={layout.explorer.widthPct}
                  minSize={10}
                  maxSize={50}
                  collapsible={false}
                  order={1}
                >
                  <FileExplorerStub />
                </ResizablePanel>
                <ResizableHandle withHandle />
              </>
            )}
            <ResizablePanel
              defaultSize={editorWidthPct(layout)}
              minSize={20}
              order={2}
            >
              <EditorPane />
            </ResizablePanel>
            {layout.chat.visible && (
              <>
                <ResizableHandle withHandle />
                <ResizablePanel
                  defaultSize={layout.chat.widthPct}
                  minSize={10}
                  maxSize={50}
                  collapsible={false}
                  order={3}
                >
                  <ChatView />
                </ResizablePanel>
              </>
            )}
          </ResizablePanelGroup>
        </ResizablePanel>
        {layout.dock.visible && (
          <>
            <ResizableHandle withHandle />
            <ResizablePanel
              defaultSize={layout.dock.heightPct}
              minSize={10}
              maxSize={50}
              collapsible={false}
            >
              <DockPlaceholder />
            </ResizablePanel>
          </>
        )}
      </ResizablePanelGroup>
      <StatusBar />
    </div>
  );
}
