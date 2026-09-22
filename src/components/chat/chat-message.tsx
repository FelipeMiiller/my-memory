import type * as React from "react";
import type { ChatMessage } from "@/lib/chat/types";
import { cn } from "@/utils/tailwind";

/**
 * ChatMessage — renders a single message bubble with markdown-flavored
 * formatting. No external markdown lib in Phase 1: we handle headings,
 * bold, inline code, fenced code blocks, and unordered lists via regex.
 *
 * Streaming mode shows a blinking caret appended to the assistant bubble.
 */

interface ChatMessageProps {
  message: ChatMessage;
  streaming?: boolean;
}

// biome-ignore lint/suspicious/noRedeclare: false positive — TS type/value namespaces separate
export function ChatMessage({
  message,
  streaming = false,
}: ChatMessageProps): React.JSX.Element {
  const isUser = message.role === "user";
  const isAssistant = message.role === "assistant";

  return (
    <div
      className={cn("flex w-full", isUser ? "justify-end" : "justify-start")}
      data-testid={`chat-message-${message.role}`}
      data-message-id={message.id}
    >
      <div
        className={cn(
          "max-w-[85%] rounded-lg px-3 py-2 text-sm shadow-sm",
          isUser && "bg-primary text-primary-foreground",
          isAssistant && "bg-card border border-border text-foreground",
          message.role === "system" && "bg-muted text-muted-foreground italic",
        )}
      >
        <div className="mb-0.5 text-[10px] uppercase tracking-wider opacity-60">
          {message.role === "user"
            ? "Você"
            : message.role === "assistant"
              ? "Assistente"
              : "Sistema"}
        </div>
        <MarkdownLite content={message.content} />
        {streaming && isAssistant && (
          <span className="ml-0.5 inline-block animate-pulse">▍</span>
        )}
        {message.sources && message.sources.length > 0 && (
          <details className="mt-2 border-t border-border/40 pt-1.5 text-[11px] opacity-70">
            <summary className="cursor-pointer">
              Fontes ({message.sources.length})
            </summary>
            <ul className="mt-1 list-disc space-y-0.5 pl-4">
              {/* biome-ignore lint/suspicious/noArrayIndexKey: static list */}
              {message.sources.map((s, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static list
                <li key={i}>
                  <code className="rounded bg-muted px-1 font-mono text-[10px]">
                    {s.path}
                  </code>{" "}
                  <span className="opacity-70">score={s.score.toFixed(2)}</span>
                </li>
              ))}
            </ul>
          </details>
        )}
      </div>
    </div>
  );
}

/**
 * Minimal markdown renderer (Phase 1). Covers:
 * - Fenced code blocks ```lang\n...\n```
 * - Inline code `code`
 * - Bold **text**
 * - Headings # ## ###
 * - Unordered lists - item
 * - Empty lines (paragraph breaks)
 *
 * NOT covered: links, images, tables, ordered lists, blockquotes.
 * Phase 2 may swap to `react-markdown` if pain point.
 */
function MarkdownLite({ content }: { content: string }): React.JSX.Element {
  const blocks = content.split(/\n{2,}/);
  return (
    <div className="space-y-2">
      {/* biome-ignore lint/suspicious/noArrayIndexKey: static block index */}
      {blocks.map((block, bi) => {
        // Fenced code block
        const fence = block.match(/^```(\w*)\n([\s\S]*?)\n?```$/);
        if (fence) {
          return (
            <pre
              key={bi}
              className="overflow-x-auto rounded-md bg-muted px-2 py-1.5 font-mono text-[11px]"
            >
              <code>{fence[2]}</code>
            </pre>
          );
        }
        // Heading
        const heading = block.match(/^(#{1,3})\s+(.*)$/);
        if (heading && block.split("\n").length === 1) {
          const level = heading[1].length;
          const text = heading[2];
          const className =
            level === 1
              ? "font-semibold text-base"
              : level === 2
                ? "font-semibold text-sm"
                : "font-medium text-sm";
          return (
            <div key={bi} className={className}>
              {renderInline(text)}
            </div>
          );
        }
        // Unordered list
        if (block.match(/^[\s]*[-*]\s+/m)) {
          const items = block.split(/\n/).filter((l) => l.match(/^\s*[-*]\s+/));
          return (
            <ul key={bi} className="list-disc space-y-0.5 pl-5">
              {/* biome-ignore lint/suspicious/noArrayIndexKey: static list */}
              {items.map((it, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: static list
                <li key={i}>{renderInline(it.replace(/^\s*[-*]\s+/, ""))}</li>
              ))}
            </ul>
          );
        }
        // Plain paragraph
        return (
          <p key={bi} className="leading-relaxed">
            {renderInline(block)}
          </p>
        );
      })}
    </div>
  );
}

function renderInline(input: string): React.ReactNode {
  // Handle inline code first (so we don't format inside code).
  const parts: React.ReactNode[] = [];
  let cursor = 0;
  const codeRe = /`([^`]+)`/g;
  let match: RegExpExecArray | null;
  let idx = 0;
  // biome-ignore lint/suspicious/noAssignInExpressions: regex iteration idiom
  while ((match = codeRe.exec(input)) !== null) {
    if (match.index > cursor) {
      parts.push(renderBold(input.slice(cursor, match.index), idx++));
    }
    parts.push(
      <code
        key={`c-${idx++}`}
        className="rounded bg-muted px-1 font-mono text-[11px]"
      >
        {match[1]}
      </code>,
    );
    cursor = match.index + match[0].length;
  }
  if (cursor < input.length) {
    parts.push(renderBold(input.slice(cursor), idx++));
  }
  return parts;
}

function renderBold(text: string, keyBase: number): React.ReactNode {
  const parts: React.ReactNode[] = [];
  const boldRe = /\*\*([^*]+)\*\*/g;
  let cursor = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  // biome-ignore lint/suspicious/noAssignInExpressions: regex iteration idiom
  while ((m = boldRe.exec(text)) !== null) {
    if (m.index > cursor) parts.push(text.slice(cursor, m.index));
    parts.push(<strong key={`b-${keyBase}-${i++}`}>{m[1]}</strong>);
    cursor = m.index + m[0].length;
  }
  if (cursor < text.length) parts.push(text.slice(cursor));
  return parts;
}
