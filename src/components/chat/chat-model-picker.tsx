import { Check, ChevronRight, Search, Settings, Sparkles } from "lucide-react";
import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useChatStore } from "@/lib/chat/store";
import {
  CHAT_MODEL_META,
  CHAT_MODELS,
  CHAT_PROVIDER_NAMES,
  type ChatModelMeta,
  type ChatProvider,
} from "@/lib/chat/types";
import { cn } from "@/utils/tailwind";

/**
 * ChatModelPicker — VS Code-style language model dropdown (ADR-051 + M1 polish).
 *
 * Layout (top to bottom) mirrors the VS Code Copilot Chat picker:
 *   1. Search input + settings gear (header)
 *   2. "Auto" option — picks best available model (no-op while hardcoded;
 *      real wiring lands with ADR-052 `mode: "auto" | "manual"`)
 *   3. Current provider's models (always expanded)
 *   4. "Other Models" group (collapsible) — every other provider
 *
 * Per-row right-aligned badge:
 *   - `Nx` when no reasoning effort (relative speed tier from metadata)
 *   - `Medium · 2x` / `High · 3x` when reasoningEffort is set
 *   - `Upgrade` link when the provider has no API key configured yet
 *
 * Phase 1 MVP (this branch): reads from hardcoded `CHAT_MODELS` +
 * `CHAT_MODEL_META`. When ADR-052 lands, the metadata source switches to
 * `chatLanguageModels.json` without UI changes.
 */

interface ChatModelPickerProps {
  onOpenSettings?: () => void;
  /** When true, render the trigger as an icon-only button (for the chat
   *  header). Otherwise the default "Provider · Model" label variant is
   *  used (kept for back-compat / future re-introduction). */
  compact?: boolean;
}

export function ChatModelPicker({
  onOpenSettings,
  compact = false,
}: ChatModelPickerProps = {}): React.JSX.Element {
  const { t } = useTranslation();
  const config = useChatStore((s) => s.config);
  const setConfig = useChatStore((s) => s.setConfig);
  const [open, setOpen] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const [otherOpen, setOtherOpen] = React.useState(true);
  const containerRef = React.useRef<HTMLDivElement>(null);
  const searchRef = React.useRef<HTMLInputElement>(null);

  // Click-outside + Escape dismiss; auto-focus search when opening.
  React.useEffect(() => {
    if (!open) return;
    searchRef.current?.focus();
    function handlePointer(e: MouseEvent): void {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
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

  // Reset transient UI state when the popover closes.
  React.useEffect(() => {
    if (!open) setSearch("");
  }, [open]);

  const providers = Object.keys(CHAT_MODELS) as ChatProvider[];
  const currentProvider = config.provider;
  const otherProviders = providers.filter((p) => p !== currentProvider);
  const disabled = !config.hasApiKey;

  const needle = search.trim().toLowerCase();
  const matches = React.useCallback(
    (id: string): boolean => {
      if (needle === "") return true;
      const meta = CHAT_MODEL_META[id];
      const haystack = `${meta?.displayName ?? id} ${id}`.toLowerCase();
      return haystack.includes(needle);
    },
    [needle],
  );

  const currentModels = CHAT_MODELS[currentProvider].filter(matches);
  const otherGroups = otherProviders
    .map((p) => ({
      provider: p,
      models: CHAT_MODELS[p].filter(matches),
    }))
    .filter((g) => g.models.length > 0);

  function pick(provider: ChatProvider, model: string): void {
    void setConfig({ provider, model });
    setOpen(false);
  }

  function pickAuto(): void {
    // Auto wiring lands with ADR-052 (`mode: "auto" | "manual"`). For now we
    // close the popover and keep the current model — selection visually
    // matches VS Code without changing runtime behaviour.
    setOpen(false);
  }

  function handleSettings(e: React.MouseEvent): void {
    e.stopPropagation();
    setOpen(false);
    onOpenSettings?.();
  }

  const activeDisplay =
    CHAT_MODEL_META[config.model]?.displayName ?? config.model;
  const providerDisplay = CHAT_PROVIDER_NAMES[currentProvider];

  return (
    <div
      ref={containerRef}
      className="relative"
      data-testid="chat-model-picker"
    >
      <Button
        variant="ghost"
        size={compact ? "icon" : "sm"}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className={cn(compact ? "h-7 w-7" : "h-7 gap-1 px-2 text-xs")}
        data-testid="chat-model-picker-button"
      >
        {compact ? (
          <Sparkles className="h-3.5 w-3.5 opacity-70" />
        ) : (
          <>
            <Sparkles className="h-3 w-3 opacity-70" />
            <span className="font-medium">{providerDisplay}</span>
            <span className="text-muted-foreground">·</span>
            <span className="text-muted-foreground">{activeDisplay}</span>
          </>
        )}
      </Button>

      {open && (
        <div
          className="absolute bottom-full left-0 z-50 mb-1 w-80 overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-lg"
          data-testid="chat-model-picker-popover"
          role="listbox"
        >
          {/* Header: search + settings gear */}
          <div className="flex items-center gap-1 border-b border-border/40 px-2 py-1.5">
            <Search className="h-3.5 w-3.5 shrink-0 opacity-50" aria-hidden />
            <Input
              ref={searchRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("chat.picker.search")}
              className="h-6 border-0 bg-transparent px-1 text-xs shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
              data-testid="chat-model-picker-search"
            />
            <button
              type="button"
              onClick={handleSettings}
              title={t("chat.picker.settingsTitle")}
              aria-label={t("chat.picker.settingsTitle")}
              className="shrink-0 rounded-sm p-1 opacity-60 transition-opacity hover:opacity-100 focus:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              data-testid="chat-model-picker-settings"
            >
              <Settings className="h-3.5 w-3.5" aria-hidden />
            </button>
          </div>

          <div className="max-h-[60vh] overflow-y-auto p-1">
            {/* Auto option */}
            <button
              type="button"
              onClick={pickAuto}
              className={cn(
                "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-xs transition-colors",
                "hover:bg-accent/40",
              )}
              data-testid="chat-model-picker-auto"
            >
              <Check
                className={cn(
                  "h-3 w-3 shrink-0",
                  !disabled && config.model === "auto"
                    ? "opacity-100"
                    : "opacity-0",
                )}
                aria-hidden
              />
              <Sparkles className="h-3 w-3 shrink-0 opacity-70" aria-hidden />
              <span className="flex-1 truncate font-medium">Auto</span>
              <span className="text-[10px] text-muted-foreground">
                {t("chat.picker.autoBadge")}
              </span>
            </button>

            {/* Current provider section */}
            <SectionLabel
              provider={currentProvider}
              isActive
              disabled={disabled}
            />
            {currentModels.length === 0 && (
              <EmptyHint text={t("chat.picker.emptyMatch")} />
            )}
            {currentModels.map((m) => (
              <ModelRow
                key={`${currentProvider}/${m}`}
                meta={CHAT_MODEL_META[m]}
                modelId={m}
                isActive={!disabled && config.model === m}
                disabled={disabled}
                onClick={() => pick(currentProvider, m)}
              />
            ))}

            {/* Other Models group (collapsible) */}
            {otherProviders.length > 0 && (
              <button
                type="button"
                onClick={() => setOtherOpen((o) => !o)}
                className="mt-1 flex w-full items-center gap-1 rounded-sm px-2 py-1 text-left text-[10px] font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:bg-accent/30"
                data-testid="chat-model-picker-other-toggle"
                aria-expanded={otherOpen}
              >
                <ChevronRight
                  className={cn(
                    "h-3 w-3 shrink-0 transition-transform",
                    otherOpen && "rotate-90",
                  )}
                  aria-hidden
                />
                <span className="flex-1">{t("chat.picker.otherModels")}</span>
              </button>
            )}
            {otherOpen &&
              otherGroups.map(({ provider, models }) => (
                <div key={provider}>
                  <SectionLabel provider={provider} isActive={false} disabled />
                  {models.map((m) => (
                    <ModelRow
                      key={`${provider}/${m}`}
                      meta={CHAT_MODEL_META[m]}
                      modelId={m}
                      isActive={false}
                      disabled
                      onClick={() => pick(provider, m)}
                    />
                  ))}
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface SectionLabelProps {
  provider: ChatProvider;
  isActive: boolean;
  disabled: boolean;
}

function SectionLabel({
  provider,
  isActive,
  disabled,
}: SectionLabelProps): React.JSX.Element {
  const { t } = useTranslation();
  return (
    <div
      className={cn(
        "flex items-center gap-1.5 px-2 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider",
        isActive ? "text-primary" : "text-muted-foreground",
      )}
      data-testid={`chat-model-picker-provider-${provider}`}
    >
      <span className="flex-1 truncate">{CHAT_PROVIDER_NAMES[provider]}</span>
      {disabled && (
        <span className="rounded bg-muted/40 px-1 text-[9px] normal-case tracking-normal">
          {t("chat.picker.noKeyBadge")}
        </span>
      )}
    </div>
  );
}

interface ModelRowProps {
  meta: ChatModelMeta | undefined;
  modelId: string;
  isActive: boolean;
  disabled: boolean;
  onClick: () => void;
}

function ModelRow({
  meta,
  modelId,
  isActive,
  disabled,
  onClick,
}: ModelRowProps): React.JSX.Element {
  const { t } = useTranslation();
  const display = meta?.displayName ?? modelId;
  const speedBadge = `${meta?.speed ?? 1}x`;
  const effortBadge = meta?.reasoningEffort;
  const badgeText =
    effortBadge !== undefined
      ? `${effortBadge.charAt(0).toUpperCase()}${effortBadge.slice(1)} · ${speedBadge}`
      : speedBadge;

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={
        disabled ? `${display} — ${t("chat.picker.upgradeLabel")}` : display
      }
      className={cn(
        "flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left text-xs transition-colors",
        !disabled && "hover:bg-accent/40",
        isActive && "bg-accent/60",
        disabled && "cursor-not-allowed opacity-50",
      )}
      data-testid={`chat-model-picker-model-${modelId}`}
    >
      <Check
        className={cn(
          "h-3 w-3 shrink-0",
          isActive ? "opacity-100" : "opacity-0",
        )}
        aria-hidden
      />
      <span className="flex-1 truncate">{display}</span>
      {disabled ? (
        <span className="shrink-0 text-[10px] font-medium text-blue-400">
          Upgrade
        </span>
      ) : (
        <span className="shrink-0 rounded bg-muted/40 px-1.5 py-0.5 text-[10px] tabular-nums text-muted-foreground">
          {badgeText}
        </span>
      )}
    </button>
  );
}

function EmptyHint({ text }: { text: string }): React.JSX.Element {
  return (
    <div className="px-2 py-1.5 text-[10px] italic text-muted-foreground">
      {text}
    </div>
  );
}

export default ChatModelPicker;
