import { X } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useChatStore } from "@/lib/chat/store";
import {
  CHAT_MODELS,
  type ChatConfigUpdate,
  type ChatProvider,
} from "@/lib/chat/types";

/**
 * ChatSettingsModal — provider, model, API key (write-only), useMemory toggle.
 *
 * API key is WRITE-ONLY. The field is always empty on open; the user types
 * a new key to replace. We never read or display the stored key anywhere.
 */

interface ChatSettingsModalProps {
  open: boolean;
  onClose: () => void;
}

export function ChatSettingsModal({
  open,
  onClose,
}: ChatSettingsModalProps): React.JSX.Element | null {
  const config = useChatStore((s) => s.config);
  const setConfig = useChatStore((s) => s.setConfig);
  const loadConfig = useChatStore((s) => s.loadConfig);

  const [provider, setProvider] = React.useState<ChatProvider>(config.provider);
  const [model, setModel] = React.useState<string>(config.model);
  const [apiKey, setApiKey] = React.useState<string>("");
  const [useMemory, setUseMemory] = React.useState<boolean>(config.useMemory);
  const [saving, setSaving] = React.useState(false);

  // Re-sync when opened.
  React.useEffect(() => {
    if (open) {
      setProvider(config.provider);
      setModel(config.model);
      setUseMemory(config.useMemory);
      setApiKey(""); // always start blank — write-only field
    }
  }, [open, config.provider, config.model, config.useMemory]);

  if (!open) return null;

  async function handleSave(): Promise<void> {
    setSaving(true);
    try {
      const update: ChatConfigUpdate = {
        provider,
        model,
        useMemory,
      };
      if (apiKey.trim().length > 0) {
        update.apiKey = apiKey.trim();
      }
      await setConfig(update);
      await loadConfig(); // refresh hasApiKey etc.
      onClose();
    } finally {
      setSaving(false);
    }
  }

  const models = CHAT_MODELS[provider];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      data-testid="chat-settings-modal"
      role="dialog"
      aria-modal="true"
    >
      <Card className="w-full max-w-md">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Configurações do Chat</CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            aria-label="Fechar"
          >
            <X className="h-4 w-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Provider
            </span>
            <div className="flex gap-2">
              {(["anthropic", "openai"] as const).map((p) => (
                <Button
                  key={p}
                  variant={provider === p ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    setProvider(p);
                    const m = CHAT_MODELS[p][0];
                    if (m) setModel(m);
                  }}
                  data-testid={`chat-settings-provider-${p}`}
                >
                  {p === "anthropic" ? "Anthropic" : "OpenAI"}
                </Button>
              ))}
            </div>
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
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="w-full rounded-md border border-input bg-input/20 px-2 py-1.5 text-sm"
              data-testid="chat-settings-model"
            >
              {models.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label
              htmlFor="chat-settings-apikey"
              className="text-xs font-medium uppercase tracking-wider text-muted-foreground"
            >
              API Key
              {config.hasApiKey && (
                <span className="ml-2 normal-case text-[10px] text-emerald-500">
                  ● já configurada
                </span>
              )}
            </label>
            <Input
              id="chat-settings-apikey"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={
                config.hasApiKey
                  ? "Cole uma nova key para substituir a existente"
                  : "Cole sua API key aqui"
              }
              autoComplete="off"
              data-testid="chat-settings-apikey"
            />
            <p className="text-[10px] text-muted-foreground">
              Armazenada em chat-config.json no userData. Nunca exposta ao
              renderer.
            </p>
          </div>

          <div className="flex items-center justify-between rounded-md border border-border bg-card/30 px-3 py-2">
            <div>
              <div className="text-sm font-medium">Usar memória do vault</div>
              <div className="text-[11px] text-muted-foreground">
                Injeta contexto via{" "}
                <code className="font-mono">mem search</code> antes de cada
                envio.
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

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" onClick={onClose} disabled={saving}>
              Cancelar
            </Button>
            <Button
              onClick={handleSave}
              disabled={saving}
              data-testid="chat-settings-save"
            >
              {saving ? "Salvando…" : "Salvar"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
