import type { ChatConfig, MemSearchResult } from "@/lib/chat/types";

/**
 * System prompt assembly (ADR-051).
 *
 * The base prompt establishes the assistant's persona and constraints. When
 * RAG is enabled, we append a "Memória do vault" block with the top-K
 * snippets from `mem search` so the model can cite real sources instead of
 * hallucinating.
 */

export const BASE_SYSTEM_PROMPT = `You are a helpful assistant integrated into the my-memory viewer — a desktop app that visualizes a knowledge graph backed by the user's local Markdown vault.

Guidelines:
- Answer in the user's language (Portuguese by default; English if they write in English).
- Be concise. Prefer 2-4 short paragraphs over walls of text.
- When citing vault content (injected below), mention the source path naturally (e.g. "in [[ADR-040]]") so the user can navigate.
- If you don't know something, say so. Never invent file paths, code, or ADR numbers.
- Code snippets should be formatted in fenced blocks with the language tag.`;

const RAG_HEADER = `## Vault context (auto-injected via \`mem search\`)

Use these snippets only if they're relevant to the user's question. Cite the source path when you use them.

`;

const RAG_MAX_CHARS = 8_000;

export function buildSystemPrompt(
  config: ChatConfig,
  ragResults: ReadonlyArray<MemSearchResult>,
): string {
  if (!config.useMemory || ragResults.length === 0) {
    return BASE_SYSTEM_PROMPT;
  }
  const lines: string[] = [BASE_SYSTEM_PROMPT, "", RAG_HEADER];
  let chars = 0;
  for (let i = 0; i < ragResults.length; i++) {
    const r = ragResults[i];
    const snippet =
      r.snippet.length > 400 ? `${r.snippet.slice(0, 400)}…` : r.snippet;
    const block = `[${i + 1}] ${r.title} (${r.path}, score=${r.score.toFixed(3)})\n${snippet}\n\n`;
    if (chars + block.length > RAG_MAX_CHARS) break;
    lines.push(block);
    chars += block.length;
  }
  return lines.join("");
}
