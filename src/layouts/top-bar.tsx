import * as React from "react";
import { PanelLeft, MessageSquare, Terminal, Settings2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { useTranslation } from "react-i18next";
import { useUiStore } from "@/lib/ui/use-ui-store";

/**
 * TopBar — global navigation bar (ADR-053).
 *
 * Contains: window title + toggle buttons for each pane + locale switcher.
 * Replaces the tab strip from the Phase 1 layout.
 */

export function TopBar(): React.JSX.Element {
  const { t } = useTranslation();
  const layout = useUiStore((s) => s.layout);
  const togglePane = useUiStore((s) => s.togglePane);

  return (
    <div className="flex h-10 shrink-0 items-center justify-between border-b border-border bg-card/40 px-3">
      <div className="flex items-center gap-2">
        <span className="text-xs font-semibold tracking-tight">{t("app.title", "Grafo de Memória")}</span>
        <span className="text-[10px] text-muted-foreground">·</span>
        <span className="text-[10px] text-muted-foreground">{t("app.subtitle", "v3 workspace")}</span>
      </div>
      <div className="flex items-center gap-1">
        <Button
          variant={layout.explorer.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("explorer")}
          aria-label="Toggle Explorer (Ctrl+B)"
          data-testid="topbar-toggle-explorer"
          title="Toggle Explorer (Ctrl+B)"
        >
          <PanelLeft className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.chat.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("chat")}
          aria-label="Toggle Chat (Ctrl+J)"
          data-testid="topbar-toggle-chat"
          title="Toggle Chat (Ctrl+J)"
        >
          <MessageSquare className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.dock.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("dock")}
          aria-label="Toggle Terminal Dock (Ctrl+`)"
          data-testid="topbar-toggle-dock"
          title="Toggle Terminal (Ctrl+`)"
        >
          <Terminal className="h-4 w-4" />
        </Button>
        <span className="mx-1 h-4 w-px bg-border" />
        <LocaleSwitcher />
      </div>
    </div>
  );
}
