import * as React from "react";
import { PanelLeft, MessageSquare, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { WindowControls } from "@/components/workspace/window-controls";
import { useTranslation } from "react-i18next";
import { useUiStore } from "@/lib/ui/use-ui-store";

/**
 * TopBar — global navigation bar (ADR-053).
 *
 * Contains: window title + toggle buttons for each pane + locale switcher +
 * window controls (Windows / Linux). Replaces the tab strip from Phase 1.
 */

export function TopBar(): React.JSX.Element {
  const { t } = useTranslation();
  const layout = useUiStore((s) => s.layout);
  const togglePane = useUiStore((s) => s.togglePane);

  return (
    <div className="flex h-10 shrink-0 items-center justify-between border-b border-border bg-card/40 px-3">
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold tracking-tight">{t("app.title")}</span>
        <span className="text-[10px] text-muted-foreground">·</span>
        <span className="text-[10px] text-muted-foreground">{t("app.subtitle")}</span>
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant={layout.explorer.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("explorer")}
          aria-label={t("workspace.topbar.toggleExplorer")}
          title={t("workspace.topbar.toggleExplorer")}
          data-testid="topbar-toggle-explorer"
        >
          <PanelLeft className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.chat.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("chat")}
          aria-label={t("workspace.topbar.toggleChat")}
          title={t("workspace.topbar.toggleChat")}
          data-testid="topbar-toggle-chat"
        >
          <MessageSquare className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.dock.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("dock")}
          aria-label={t("workspace.topbar.toggleDock")}
          title={t("workspace.topbar.toggleDock")}
          data-testid="topbar-toggle-dock"
        >
          <Terminal className="h-4 w-4" />
        </Button>
        <span className="mx-1 h-4 w-px bg-border" />
        <LocaleSwitcher />
        <WindowControls />
      </div>
    </div>
  );
}
