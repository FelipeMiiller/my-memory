import * as React from "react";
import { X, RefreshCw, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useChatStore } from "@/lib/chat/store";

/**
 * ChatSettingsModal — dynamic providers + models (ADR-052).
 *
 * Providers come from chatLanguageModels.json (loaded via store.loadProviders).
 * API keys are write-only (per vendor) — typed once, never read back.
 */

interface ChatSettingsModalProps {
  open: boolean;
  onClose: () => void;
}

export function ChatSettingsModal({ open, onClose }: ChatSettingsModalProps): React.JSX.Element | null {
  const config = useChatStore((s) => s.config);
  const providers = useChatStore((s) => s.providers);
  const setConfig = useChatStore((s) => s.setConfig);
  const loadProviders = useChatStore((s) => s.loadProviders);
  const discoverModels = useChatStore((s) => s.discoverModels);

  const [activeVendor, setActiveVendor] = React.useState<string>(config.activeVendor);
  const [activeModelId, setActiveModelId] = React.useState<string>(config.activeModelId);
  const [useMemory, setUseMemory] = React.useState<boolean>(config.useMemory);
  const [apiKeys, setApiKeys] = React.useState<Record<string, string>>({});
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    if (open) {
      setActiveVendor(config.activeVendor);
      setActiveModelId(config.activeModelId);
      setUseMemory(config.useMemory);
      setApiKeys({});
      void loadProviders();
      // Auto-discover for providers that support it.
      for (const p of providers) {
        if (p.autoDiscover && p.models.length === 0) {
          void discoverModels(p.vendor);
        }
      }
    }
  }, [open, config.activeVendor, config.activeModelId, config.useMemory, loadProviders, discoverModels, providers]);

  if (!open) return null;

  const activeProvider = providers.find((p) => p.vendor === activeVendor) ?? null;
  const activeModel = activeProvider?.models.find((m) => m.id === activeModelId) ?? null;
  const activeHasKey =
    config.providers.find((p) => p.vendor === activeVendor)?.hasApiKey ?? false;

  async function handleSave(): Promise<void> {
    setSaving(true);
    try {
      const update: Parameters<typeof setConfig>[0] = {
        activeVendor,
        activeModelId,
        useMemory,
      };
      // Only include apiKeys for vendors that have a value typed in.
      const keysToSave: Record<string, string> = {};
      for (const [k, v] of Object.entries(apiKeys)) {
        if (v.trim().length > 0) keysToSave[k] = v.trim();
      }
      if (Object.keys(keysToSave).length > 0) {
        (update as { apiKeys?: Record<string, string> }).apiKeys = keysToSave;
      }
      await setConfig(update);
      onClose();
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      data-testid="chat-settings-modal"
      role="dialog"
      aria-modal="true"
    >
      <Card className="max-h-[85vh] w-full max-w-2xl overflow-hidden">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Configurações do Chat</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClose} aria-label="Fechar">
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="max-h-[70vh] space-y-4 overflow-y-auto">
          {/* Use memory toggle */}
          <div className="flex items-center justify-between rounded-md border border-border bg-card/30 px-3 py-2">
            <div>
              <div className="text-sm font-medium">Usar memória do vault</div>
              <div className="text-[11px] text-muted-foreground">
                Injeta contexto via <code className="font-mono">mem search</code> antes de cada envio.
              </div>
            </div>
            <input
              type="checkbox"
              checked={useMemory}
              onChange={(e) => setUseMemory(e.target.checked)}
              className="h-4 w-4 cursor-pointer"
              data-testid="chat-settings-usememory"
            />
          </div>

          {/* Provider list */}
          <div className="space-y-2">
            <div className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Providers
            </div>
            <div className="grid gap-2">
              {providers.length === 0 && (
                <div className="text-[11px] text-muted-foreground">Carregando providers…</div>
              )}
              {providers.map((p) => {
                const hasKey = config.providers.find((x) => x.vendor === p.vendor)?.hasApiKey ?? false;
                const isActive = p.vendor === activeVendor;
                return (
                  <button
                    key={p.vendor}
                    type="button"
                    onClick={() => {
                      setActiveVendor(p.vendor);
                      const first = p.models[0];
                      if (first) setActiveModelId(first.id);
                    }}
                    className={`flex items-center justify-between rounded-md border px-3 py-2 text-left text-sm transition-colors ${
                      isActive ? "border-primary bg-primary/10" : "border-border hover:bg-accent/40"
                    }`}
                    data-testid={`chat-settings-provider-${p.vendor}`}
                  >
                    <div>
                      <div className="font-medium">{p.name}</div>
                      <div className="text-[10px] text-muted-foreground">
                        {p.models.length} model{p.models.length === 1 ? "" : "s"}
                        {p.autoDiscover ? " · auto-discover" : ""}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {hasKey ? (
                        <span className="text-emerald-500" title="API key configurada">
                          <Check className="h-3.5 w-3.5" />
                        </span>
                      ) : (
                        <span className="text-[10px] text-muted-foreground">no key</span>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Active provider details */}
          {activeProvider && (
            <div className="space-y-3 rounded-md border border-border bg-card/30 p-3">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium">{activeProvider.name}</div>
                  {activeProvider.baseUrl && (
                    <div className="text-[10px] text-muted-foreground font-mono">
                      {activeProvider.baseUrl}
                    </div>
                  )}
                </div>
                {activeProvider.autoDiscover && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => void discoverModels(activeProvider.vendor)}
                    data-testid="chat-settings-refresh"
                  >
                    <RefreshCw className="mr-1 h-3 w-3" />
                    Discover
                  </Button>
                )}
              </div>

              <div className="space-y-1">
                <label
                  htmlFor="chat-settings-model"
                  className="text-xs font-medium uppercase tracking-wider text-muted-foreground"
                >
                  Modelo
                </label>
                <select
                  id="chat-settings-model"
                  value={activeModelId}
                  onChange={(e) => setActiveModelId(e.target.value)}
                  className="w-full rounded-md border border-input bg-input/20 px-2 py-1.5 text-sm"
                  data-testid="chat-settings-model"
                >
                  {activeProvider.models.length === 0 && (
                    <option value="">{activeProvider.autoDiscover ? "Discovering…" : "(sem models)"}</option>
                  )}
                  {activeProvider.models.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name}
                      {m.maxInputTokens ? ` (${Math.round(m.maxInputTokens / 1000)}k ctx)` : ""}
                    </option>
                  ))}
                </select>
                {activeModel && (activeModel.toolCalling || activeModel.vision) && (
                  <div className="flex gap-1 text-[10px] text-muted-foreground">
                    {activeModel.toolCalling && <span>🛠 tool calling</span>}
                    {activeModel.vision && <span>👁 vision</span>}
                    {activeModel.supportsReasoningEffort && (
                      <span>🧠 reasoning: {activeModel.supportsReasoningEffort.join("/")}</span>
                    )}
                  </div>
                )}
              </div>

              <div className="space-y-1">
                <label
                  htmlFor={`chat-settings-apikey-${activeProvider.vendor}`}
                  className="text-xs font-medium uppercase tracking-wider text-muted-foreground"
                >
                  API Key {activeHasKey && <span className="ml-2 normal-case text-[10px] text-emerald-500">● já configurada</span>}
                </label>
                <Input
                  id={`chat-settings-apikey-${activeProvider.vendor}`}
                  type="password"
                  value={apiKeys[activeProvider.vendor] ?? ""}
                  onChange={(e) =>
                    setApiKeys((prev) => ({ ...prev, [activeProvider.vendor]: e.target.value }))
                  }
                  placeholder={activeHasKey ? "Cole uma nova key para substituir" : "Cole sua API key aqui"}
                  autoComplete="off"
                  data-testid="chat-settings-apikey"
                />
                <p className="text-[10px] text-muted-foreground">
                  Armazenada em chat-config.json (por vendor). Nunca exposta ao renderer.
                </p>
              </div>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" onClick={onClose} disabled={saving}>
              Cancelar
            </Button>
            <Button onClick={handleSave} disabled={saving} data-testid="chat-settings-save">
              {saving ? "Salvando…" : "Salvar"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
