import * as React from "react";
import { useUiStore } from "@/lib/ui/use-ui-store";

/**
 * StatusBar — bottom strip with build/runtime info (ADR-053).
 */

export function StatusBar(): React.JSX.Element {
  const layout = useUiStore((s) => s.layout);
  const resetLayout = useUiStore((s) => s.resetLayout);

  return (
    <div className="flex h-6 shrink-0 items-center justify-between border-t border-border bg-card/30 px-3 text-[10px] text-muted-foreground">
      <div className="flex items-center gap-3">
        <span>my-memory viewer</span>
        <span>·</span>
        <span>
          layout: explorer {layout.explorer.visible ? `${layout.explorer.widthPct}%` : "off"} · chat{" "}
          {layout.chat.visible ? `${layout.chat.widthPct}%` : "off"} · dock{" "}
          {layout.dock.visible ? `${layout.dock.heightPct}%` : "off"}
        </span>
      </div>
      <button
        type="button"
        onClick={() => resetLayout()}
        className="hover:text-foreground"
        title="Reset layout to defaults"
        data-testid="statusbar-reset-layout"
      >
        reset layout
      </button>
    </div>
  );
}
