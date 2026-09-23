import { Maximize2, Minus, Square, X } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/utils/tailwind";
import "@/lib/chat/ipc"; // augments globalThis.memAPI with the `window` field

/**
 * WindowControls — minimize / maximize-restore / close (ADR-053).
 *
 * Renders only on non-macOS (Windows + Linux have hidden title bar via
 * `titleBarStyle: "hidden"`). macOS keeps the system traffic lights.
 */

export function WindowControls(): React.JSX.Element | null {
  const { t } = useTranslation("topbar");
  const isMac = React.useMemo(
    () => (globalThis.navigator?.userAgent ?? "").toLowerCase().includes("mac"),
    [],
  );
  const [maximized, setMaximized] = React.useState(false);

  React.useEffect(() => {
    if (isMac) return;
    let unsubscribe: (() => void) | null = null;
    let cancelled = false;
    (async () => {
      try {
        const initial = await globalThis.memAPI?.window.isMaximized();
        if (!cancelled) setMaximized(Boolean(initial));
        unsubscribe =
          globalThis.memAPI?.window.onMaximizeChanged((m: boolean) =>
            setMaximized(m),
          ) ?? null;
      } catch (err) {
        console.warn("[window-controls] init failed:", err);
      }
    })();
    return () => {
      cancelled = true;
      unsubscribe?.();
    };
  }, [isMac]);

  if (isMac) return null;

  async function handleMinimize(): Promise<void> {
    await globalThis.memAPI?.window.minimize();
  }
  async function handleToggleMaximize(): Promise<void> {
    const next = await globalThis.memAPI?.window.toggleMaximize();
    if (typeof next === "boolean") setMaximized(next);
  }
  async function handleClose(): Promise<void> {
    await globalThis.memAPI?.window.close();
  }

  return (
    <div className="flex items-center" data-testid="window-controls">
      <Button
        variant="ghost"
        size="icon"
        onClick={() => void handleMinimize()}
        aria-label={t("minimize")}
        title={t("minimize")}
        data-testid="window-control-minimize"
        className="h-8 w-8 rounded-none"
      >
        <Minus className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => void handleToggleMaximize()}
        aria-label={maximized ? t("restore") : t("maximize")}
        title={maximized ? t("restore") : t("maximize")}
        data-testid="window-control-maximize"
        className="h-8 w-8 rounded-none"
      >
        {maximized ? (
          <Maximize2 className="h-3.5 w-3.5" />
        ) : (
          <Square className="h-3.5 w-3.5" />
        )}
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => void handleClose()}
        aria-label={t("close")}
        title={t("close")}
        data-testid="window-control-close"
        className={cn(
          "h-8 w-8 rounded-none",
          "hover:bg-destructive hover:text-destructive-foreground",
        )}
      >
        <X className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
