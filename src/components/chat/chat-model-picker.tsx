import * as React from "react";
import { ChevronDown, Check, Sparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/utils/tailwind";
import { useChatStore } from "@/lib/chat/store";
import { CHAT_MODELS, type ChatProvider } from "@/lib/chat/types";

/**
 * ChatModelPicker — VS Code-style header dropdown (ADR-051).
 *
 * Phase 1 MVP (this branch): picks from the hardcoded CHAT_MODELS list
 * (anthropic + openai). When ADR-052 lands, the picker will read from
 * `chatLanguageModels.json` instead — this component will be refactored.
 *
 * Click-outside + Escape dismiss the popover.
 */

export function ChatModelPicker(): React.JSX.Element {
  const { t } = useTranslation();
  const config = useChatStore((s) => s.config);
  const setConfig = useChatStore((s) => s.setConfig);
  const [open, setOpen] = React.useState(false);
  const containerRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (!open) return;
    function handlePointer(e: MouseEvent): void {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function handleKey(e: KeyboardEvent): void {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", handlePointer);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handlePointer);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open]);

  function pick(provider: ChatProvider, model: string): void {
    void setConfig({ provider, model } as Parameters<typeof setConfig>[0]);
    setOpen(false);
  }

  const providerName =
    config.provider === "anthropic" ? "Anthropic Claude" : "OpenAI";

  return (
    <div ref={containerRef} className="relative" data-testid="chat-model-picker">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="h-7 gap-1 px-2 text-xs"
        data-testid="chat-model-picker-button"
      >
        <Sparkles className="h-3 w-3 opacity-70" />
        <span className="font-medium">{providerName}</span>
        <span className="text-muted-foreground">·</span>
        <span className="text-muted-foreground">{config.model}</span>
        <ChevronDown
          className={cn(
            "h-3 w-3 opacity-60 transition-transform",
            open && "rotate-180",
          )}
        />
      </Button>
      {open && (
        <div
          className="absolute left-0 top-full z-50 mt-1 w-80 rounded-md border border-border bg-popover p-1 shadow-lg"
          data-testid="chat-model-picker-popover"
          role="listbox"
        >
          <div className="px-2 pb-1 pt-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {t("workspace.topbar.toggleChat", "Provider & Model")}
          </div>
          <div className="max-h-[60vh] overflow-y-auto">
            {(Object.keys(CHAT_MODELS) as ChatProvider[]).map((p) => {
              const isActiveProvider = p === config.provider;
              const providerDisplay = p === "anthropic" ? "Anthropic Claude" : "OpenAI";
              return (
                <div key={p} className="mb-1">
                  <div
                    className={cn(
                      "flex items-center gap-1.5 px-2 py-1 text-[10px] font-semibold uppercase tracking-wider",
                      isActiveProvider ? "text-primary" : "text-muted-foreground",
                    )}
                    data-testid={`chat-model-picker-provider-${p}`}
                  >
                    <span className="flex-1 truncate">{providerDisplay}</span>
                    {!config.hasApiKey && (
                      <span className="rounded bg-muted/40 px-1 text-[9px] normal-case tracking-normal">
                        no key
                      </span>
                    )}
                  </div>
                  {CHAT_MODELS[p].map((m) => {
                    const isActiveModel = isActiveProvider && m === config.model;
                    return (
                      <button
                        type="button"
                        key={`${p}/${m}`}
                        onClick={() => pick(p, m)}
                        className={cn(
                          "flex w-full items-center gap-1 rounded-sm px-2 py-1 text-left text-xs transition-colors",
                          "hover:bg-accent/40",
                          isActiveModel && "bg-accent/60",
                        )}
                        data-testid={`chat-model-picker-model-${p}-${m}`}
                      >
                        {isActiveModel ? (
                          <Check className="h-3 w-3 shrink-0 opacity-80" />
                        ) : (
                          <span className="inline-block h-3 w-3 shrink-0" />
                        )}
                        <span className="flex-1 truncate">{m}</span>
                      </button>
                    );
                  })}
                </div>
              );
            })}
          </div>
          <div className="border-t border-border/40 px-2 py-1.5 text-[10px] text-muted-foreground">
            ⚙️ Settings → multi-provider (ADR-052) chega no próximo merge
          </div>
        </div>
      )}
    </div>
  );
}
