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
 * The whole bar is `-webkit-app-region: drag` so the user can grab it to
 * move the window (required because `titleBarStyle: "hidden"` removes the
 * native title bar on Windows / Linux). All interactive elements (buttons,
 * locale switcher, window controls) override with `no-drag` so clicks still
 * register.
 *
 * macOS keeps the system traffic lights via `titleBarStyle: "hiddenInset"`,
 * so the entire bar is draggable there too (system reserves space for the
 * traffic lights on the left).
 */

export function TopBar(): React.JSX.Element {
  const { t } = useTranslation(["common", "topbar"]);
  const layout = useUiStore((s) => s.layout);
  const togglePane = useUiStore((s) => s.togglePane);

  // Inline style for `-webkit-app-region` (no Tailwind utility for it).
  const dragRegionStyle: React.CSSProperties = { WebkitAppRegion: "drag" } as React.CSSProperties;
  const noDragStyle: React.CSSProperties = { WebkitAppRegion: "no-drag" } as React.CSSProperties;

  return (
    <div
      className="flex h-10 shrink-0 select-none items-center justify-between border-b border-border bg-card/40 px-3"
      style={dragRegionStyle}
      data-testid="topbar"
    >
      <div
        className="flex items-center gap-2"
        data-testid="topbar-drag-region-title"
      >
        <span className="text-xs font-semibold tracking-tight">{t("app.title")}</span>
        <span className="text-[10px] text-muted-foreground">·</span>
        <span className="text-[10px] text-muted-foreground">{t("app.subtitle")}</span>
      </div>
      <div className="flex items-center gap-1" style={noDragStyle} data-testid="topbar-controls">
        <Button
          variant={layout.explorer.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("explorer")}
          aria-label={t("toggleExplorer")}
          title={t("toggleExplorer")}
          data-testid="topbar-toggle-explorer"
        >
          <PanelLeft className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.chat.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("chat")}
          aria-label={t("toggleChat")}
          title={t("toggleChat")}
          data-testid="topbar-toggle-chat"
        >
          <MessageSquare className="h-4 w-4" />
        </Button>
        <Button
          variant={layout.dock.visible ? "secondary" : "ghost"}
          size="icon"
          onClick={() => togglePane("dock")}
          aria-label={t("toggleDock")}
          title={t("toggleDock")}
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
