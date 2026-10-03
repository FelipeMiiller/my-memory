import { Terminal } from "lucide-react";
import type * as React from "react";
import { useTranslation } from "react-i18next";

/**
 * DockPlaceholder — Terminal placeholder for M4 (ADR-053).
 */

export function DockPlaceholder(): React.JSX.Element {
  const { t } = useTranslation("workspace");
  return (
    <div
      className="flex h-full flex-col bg-black/80"
      data-testid="dock-placeholder"
    >
      <div className="flex items-center gap-2 border-b border-border/40 px-3 py-1.5 text-[10px] text-muted-foreground">
        <Terminal className="h-3 w-3" />
        <span className="font-mono uppercase tracking-wider">
          {t("dock.placeholder")}
        </span>
      </div>
      <div className="flex flex-1 items-center justify-center font-mono text-xs text-muted-foreground">
        <div className="text-center">
          <p className="mb-1">$ {t("dock.comingInM4")}</p>
          <p className="text-[10px] italic">{t("dock.toggleHint")}</p>
        </div>
      </div>
    </div>
  );
}
