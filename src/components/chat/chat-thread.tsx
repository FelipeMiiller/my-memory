import * as React from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChatStore } from "@/lib/chat/store";
import { ChatMessage } from "./chat-message";

/**
 * ChatThread — auto-scrolling list of messages for the active conversation.
 * Uses ScrollArea (Radix) with imperative scroll-to-bottom on new content.
 */

export function ChatThread(): React.JSX.Element {
  const activeId = useChatStore((s) => s.activeConversationId);
  const conversations = useChatStore((s) => s.conversations);
  const streaming = useChatStore((s) => s.streaming);
  const streamingMessageId =
    streaming.kind === "streaming" ? streaming.assistantMessageId : null;

  const conv = conversations.find((c) => c.id === activeId) ?? null;
  const viewportRef = React.useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom whenever messages change or deltas arrive.
  React.useEffect(() => {
    if (!conv) return;
    const viewport = viewportRef.current;
    if (!viewport) return;
    // Defer to next frame so the DOM has the new height.
    requestAnimationFrame(() => {
      viewport.scrollTop = viewport.scrollHeight;
    });
  }, [
    conv?.messages.length,
    conv?.messages.map((m) => m.content.length).join(","),
    conv?.id,
  ]);

  if (!conv) {
    return (
      <div className="flex flex-1 items-center justify-center p-6 text-sm text-muted-foreground">
        Selecione ou crie uma conversa para começar.
      </div>
    );
  }

  return (
    <ScrollArea className="flex-1" data-testid="chat-thread">
      <div
        ref={viewportRef}
        className="flex min-h-full flex-col gap-3 p-4"
        data-testid="chat-thread-viewport"
      >
        {conv.messages.length === 0 ? (
          <div className="m-auto max-w-md text-center text-sm text-muted-foreground">
            <p className="mb-1 font-medium">Nova conversa</p>
            <p>
              Pergunte algo sobre o seu vault, peça ajuda com ADR ou peça um
              resumo de qualquer nota. As respostas são geradas por LLM e podem
              usar contexto da sua memória local (se ativado nas configurações).
            </p>
          </div>
        ) : (
          conv.messages.map((msg) => (
            <ChatMessage
              key={msg.id}
              message={msg}
              streaming={msg.id === streamingMessageId}
            />
          ))
        )}
      </div>
    </ScrollArea>
  );
}
