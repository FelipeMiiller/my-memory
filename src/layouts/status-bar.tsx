import * as React from "react";
import { useTranslation } from "react-i18next";
import { useUiStore } from "@/lib/ui/use-ui-store";

/**
 * StatusBar — bottom strip with layout info (ADR-053).
 */

export function StatusBar(): React.JSX.Element {
  const { t } = useTranslation("workspace");
  const layout = useUiStore((s) => s.layout);
  const resetLayout = useUiStore((s) => s.resetLayout);

  const explorerLabel = layout.explorer.visible
    ? `${t("statusBar.explorer")} ${layout.explorer.widthPct}%`
    : `${t("statusBar.explorer")} ${t("statusBar.off")}`;
  const chatLabel = layout.chat.visible
    ? `${t("statusBar.chat")} ${layout.chat.widthPct}%`
    : `${t("statusBar.chat")} ${t("statusBar.off")}`;
  const dockLabel = layout.dock.visible
    ? `${t("statusBar.dock")} ${layout.dock.heightPct}%`
    : `${t("statusBar.dock")} ${t("statusBar.off")}`;

  return (
    <div className="flex h-6 shrink-0 items-center justify-between border-t border-border bg-card/30 px-3 text-[10px] text-muted-foreground">
      <div className="flex items-center gap-3">
        <span>my-memory viewer</span>
        <span>·</span>
        <span>
          {t("statusBar.layout")}: {explorerLabel} · {chatLabel} · {dockLabel}
        </span>
      </div>
      <button
        type="button"
        onClick={() => resetLayout()}
        className="hover:text-foreground"
        title={t("statusBar.reset")}
        data-testid="statusbar-reset-layout"
      >
        {t("statusBar.reset")}
      </button>
    </div>
  );
}
