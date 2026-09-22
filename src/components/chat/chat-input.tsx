import { Send, Square } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/utils/tailwind";
import { type AsrTranscriptMode, MicButton } from "./mic-button";

interface ChatInputProps {
  /** ChatModelPicker node — rendered in the input toolbar (VS Code style). */
  modelPicker?: React.ReactNode;
  /** Optional locale hint forwarded to the ASR start request. */
  asrLocale?: string;
  disabled?: boolean;
  streaming: boolean;
  hasApiKey: boolean;
  onSend: (text: string) => void;
  onStop: () => void;
}

export function ChatInput({
  modelPicker,
  asrLocale,
  disabled = false,
  streaming,
  hasApiKey,
  onSend,
  onStop,
}: ChatInputProps): React.JSX.Element {
  const [text, setText] = React.useState("");
  const [partial, setPartial] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);

  const displayValue = React.useMemo(() => {
    if (!partial) return text;
    const sep = text.length > 0 && !text.endsWith(" ") ? " " : "";
    return `${text}${sep}${partial}`;
  }, [text, partial]);

  const canSend =
    !disabled && !streaming && displayValue.trim().length > 0 && hasApiKey;

  function submit(): void {
    const trimmed = displayValue.trim();
    if (!trimmed || streaming || !hasApiKey) return;
    onSend(trimmed);
    setText("");
    setPartial("");
    textareaRef.current?.focus();
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  }

  function handleUserChange(e: React.ChangeEvent<HTMLTextAreaElement>): void {
    // Any user keystroke clears the live ASR partial — they're now editing
    // the committed text directly.
    setText(e.target.value);
    if (partial) setPartial("");
  }

  function handleTranscript(t: string, mode: AsrTranscriptMode): void {
    if (mode === "partial") {
      setPartial(t);
    } else {
      // Final: append the committed tail to `text` and clear the partial.
      setText((cur) => {
        const sep = cur.length > 0 && !cur.endsWith(" ") ? " " : "";
        return cur + sep + t;
      });
      setPartial("");
    }
  }

  function handleAsrError(message: string): void {
    setError(message);
    window.setTimeout(() => setError(null), 4000);
  }

  return (
    <div
      className="border-t border-border bg-card/40 p-3"
      data-testid="chat-input"
    >
      <Textarea
        ref={textareaRef}
        value={displayValue}
        onChange={handleUserChange}
        onKeyDown={handleKeyDown}
        placeholder={
          hasApiKey
            ? "Pergunte algo… (Enter envia · Shift+Enter quebra linha)"
            : "Adicione sua API key nas Configurações para começar."
        }
        disabled={disabled || !hasApiKey}
        className={cn("min-h-[44px] max-h-[200px]", !hasApiKey && "opacity-60")}
        rows={1}
        data-testid="chat-input-textarea"
      />
      {error && (
        <p
          className="mt-1 text-[10px] text-red-500"
          role="status"
          data-testid="chat-input-asr-error"
        >
          {error}
        </p>
      )}
      <div className="mt-2 flex items-center justify-between gap-2">
        <div
          className="flex min-w-0 items-center gap-1"
          data-testid="chat-input-toolbar"
        >
          {modelPicker}
        </div>
        <div className="flex items-center gap-1">
          <MicButton
            onTranscript={handleTranscript}
            onError={handleAsrError}
            disabled={disabled || !hasApiKey}
            locale={asrLocale}
          />
          {streaming ? (
            <Button
              onClick={onStop}
              variant="destructive"
              size="icon"
              data-testid="chat-input-stop"
              aria-label="Parar geração"
            >
              <Square className="h-4 w-4" />
            </Button>
          ) : (
            <Button
              onClick={submit}
              disabled={!canSend}
              size="icon"
              data-testid="chat-input-send"
              aria-label="Enviar mensagem"
            >
              <Send className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
