import { Send, Square } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/utils/tailwind";

interface ChatInputProps {
  disabled?: boolean;
  streaming: boolean;
  hasApiKey: boolean;
  onSend: (text: string) => void;
  onStop: () => void;
}

export function ChatInput({
  disabled = false,
  streaming,
  hasApiKey,
  onSend,
  onStop,
}: ChatInputProps): React.JSX.Element {
  const [text, setText] = React.useState("");
  const textareaRef = React.useRef<HTMLTextAreaElement>(null);

  const canSend =
    !disabled && !streaming && text.trim().length > 0 && hasApiKey;

  function submit(): void {
    const trimmed = text.trim();
    if (!trimmed || streaming || !hasApiKey) return;
    onSend(trimmed);
    setText("");
    textareaRef.current?.focus();
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      submit();
    }
  }

  return (
    <div
      className="border-t border-border bg-card/40 p-3"
      data-testid="chat-input"
    >
      <div className="flex items-end gap-2">
        <Textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={
            hasApiKey
              ? "Pergunte algo… (Enter envia · Shift+Enter quebra linha)"
              : "Adicione sua API key nas Configurações para começar."
          }
          disabled={disabled || !hasApiKey}
          className={cn(
            "min-h-[44px] max-h-[200px]",
            !hasApiKey && "opacity-60",
          )}
          rows={1}
          data-testid="chat-input-textarea"
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
  );
}
