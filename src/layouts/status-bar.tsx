import * as React from "react";
import { useTranslation } from "react-i18next";
import { useUiStore } from "@/lib/ui/use-ui-store";

/**
 * StatusBar — bottom strip with layout info (ADR-053).
 */

export function StatusBar(): React.JSX.Element {
  const { t } = useTranslation();
  const layout = useUiStore((s) => s.layout);
  const resetLayout = useUiStore((s) => s.resetLayout);

  const explorerLabel = layout.explorer.visible
    ? `${t("workspace.statusBar.explorer")} ${layout.explorer.widthPct}%`
    : `${t("workspace.statusBar.explorer")} ${t("workspace.statusBar.off")}`;
  const chatLabel = layout.chat.visible
    ? `${t("workspace.statusBar.chat")} ${layout.chat.widthPct}%`
    : `${t("workspace.statusBar.chat")} ${t("workspace.statusBar.off")}`;
  const dockLabel = layout.dock.visible
    ? `${t("workspace.statusBar.dock")} ${layout.dock.heightPct}%`
    : `${t("workspace.statusBar.dock")} ${t("workspace.statusBar.off")}`;

  return (
    <div className="flex h-6 shrink-0 items-center justify-between border-t border-border bg-card/30 px-3 text-[10px] text-muted-foreground">
      <div className="flex items-center gap-3">
        <span>my-memory viewer</span>
        <span>·</span>
        <span>
          {t("workspace.statusBar.layout")}: {explorerLabel} · {chatLabel} · {dockLabel}
        </span>
      </div>
      <button
        type="button"
        onClick={() => resetLayout()}
        className="hover:text-foreground"
        title={t("workspace.statusBar.reset")}
        data-testid="statusbar-reset-layout"
      >
        {t("workspace.statusBar.reset")}
      </button>
    </div>
  );
}
