---
title: feat-viewer-chat — Chat LLM no viewer Electron
category: resource
summary: Adiciona uma aba/página de chat nativa no viewer Electron do my-memory, com suporte a múltiplos providers (Anthropic, OpenAI), streaming via IPC, persistência local em IndexedDB e injeção de contexto via `mem search`.
status: draft
tags: [viewer, chat, llm, ai-sdk, ipc, electron, mcp]
---

# feat-viewer-chat — Chat LLM nativo no viewer Electron

## Contexto

O viewer Electron do my-memory (Phase 1 MVP, commit `0f09ad8`) já tem shell React + Vite + shadcn/ui + Tailwind 4 + i18next + Cytoscape, e IPC stub `mem:dataset:load` / `mem:cli:run` placeholders em `electron/main.ts`. O usuário pediu um **fluxo completo de chat** tomando como referência arquitetural o [trendy-design/llmchat](https://github.com/trendy-design/llmchat) (Next.js 14 + AI SDK + Tiptap + Zustand + Dexie + Shadcn + IndexedDB + multi-provider).

Decisão: **manter o stack atual (Vite + React 19)**, sem Next.js. Pegar os padrões do llmchat (provider abstraction via AI SDK, state global, persistência local em IndexedDB, streaming de tokens) e adaptar para o esqueleto Vite/React já consolidado.

## Objetivo

Entregar uma aba **Chat** nativa no viewer Electron que:

1. Conversa com LLMs via providers configuráveis (Anthropic Claude default, OpenAI opcional).
2. **Streaming token a token** do LLM até o renderer via IPC do Electron.
3. **Persistência local** de múltiplas conversas em IndexedDB (sobrevive a restart do app).
4. **Integração com my-memory como RAG backend**: antes de cada envio, opcionalmente roda `mem search` via subprocess no main process e injeta os top-K resultados no system prompt.
5. **Configuração segura**: API keys nunca saem do main process; renderer só envia/provider/model/messages.

## Out of Scope

- Tools / function calling (chat puro; sem agent loops). Pode entrar em Phase 2 da feature.
- Voice input/output.
- Compartilhamento de conversa entre devices (sync).
- Tiptap rich-text — input simples com textarea + markdown render básico no message bubble. Tiptap pode entrar depois se virar dor.
- MCP-style `compile_not_retrieve` automático — manual via "inserir contexto" button.

## Decisões arquiteturais (resumo — detalhes no ADR-051)

| Decisão | Escolha | Por quê |
|---|---|---|
| Provider abstraction | Vercel AI SDK (`ai`, `@ai-sdk/openai`, `@ai-sdk/anthropic`) | Padrão de mercado, multi-provider, streaming SSE nativo, type-safe. |
| Onde roda o LLM call | Electron **main process** via `child_process.spawn`-style `fetch` (não Node SDK direto) | API keys nunca tocam o renderer (security by design). |
| Streaming | IPC streaming — main emite `mem:chat:delta` events via `webContents.send`; renderer acumula | Mais simples que SSE/HTTP custom, já usa o pipeline IPC. |
| State global | **Zustand** (1 store único `useChatStore`) | Leve, sem boilerplate, idiomático em Vite/React. Não usar Redux/Context. |
| Persistência local | **IndexedDB via `idb-keyval`** | Wrapper mínimo (~600B), sem dependências nativas, padrão do llmchat. |
| RAG | `mem search` rodando no main via `child_process.execFile('./bin/mem.exe', ['search', q, '--json'])` | Reaproveita o CLI já existente (sem reescrever busca em JS). |
| Config do provider | JSON em `app.getPath('userData')/chat-config.json` | Persiste, não commita, lido no main. |
| UI library | shadcn/ui (já instalado) — adicionar `<Textarea>` via shadcn CLI | Consistência com o resto do viewer. |

## Acceptance Criteria (EARS)

- **AC-1 (EARS-U):** Quando o usuário abre a aba Chat pela primeira vez, vê um empty-state com CTA "Nova conversa".
- **AC-2 (EARS-U):** Quando o usuário digita uma mensagem e pressiona Enter (Shift+Enter = newline), o sistema envia ao LLM via IPC e mostra a resposta em streaming token a token.
- **AC-3 (EARS-U):** Quando o usuário clica em "Stop" durante o streaming, a geração é abortada e o texto parcial permanece visível.
- **AC-4 (EARS-U):** Quando o usuário cria uma segunda conversa, ela aparece na sidebar esquerda; ao trocar, as mensagens anteriores carregam do IndexedDB.
- **AC-5 (EARS-U):** Quando o usuário reabre o app, todas as conversas anteriores estão lá com mensagens preservadas.
- **AC-6 (EARS-U):** Quando o usuário abre Settings, pode escolher provider (Anthropic/OpenAI), modelo, e colar a API key. Após salvar, o renderer recebe ack e a próxima mensagem usa o provider escolhido.
- **AC-7 (EARS-U):** Quando o usuário ativa "Use memory" e envia uma pergunta, o sistema roda `mem search` no main, pega top-5 resultados, e injeta no system prompt antes do LLM call. A resposta cita as fontes usadas em nota de rodapé.
- **AC-8 (EARS-S):** O renderer **nunca** recebe, armazena ou loga a API key. Toda config sensitive vive no main e é referenciada por ID.
- **AC-9 (EARS-S):** Se o provider retornar erro (401, 429, 5xx), o renderer mostra toast com a mensagem sanitizada (sem vazar API key nem stack trace completa).
- **AC-10 (EARS-S):** IPC channel `mem:chat:send` valida `messages` (array não vazio, max 100 itens, cada item com role+content+timestamp) e rejeita payloads inválidos com erro tipado.
- **AC-11 (EARS-S):** `npm run typecheck` exit 0; `npm run test` (vitest) PASS incluindo 3 novos testes unitários do store Zustand; `npm run smoke` exit 0 + screenshot atualizado.
- **AC-12 (EARS-S):** `biome lint` não introduz erros novos (74 pré-existentes permanecem, mas não cresce).

## Estrutura de arquivos

```
src/
  components/chat/
    chat-view.tsx              # componente raiz da aba
    chat-thread.tsx            # lista de mensagens com auto-scroll
    chat-message.tsx           # 1 mensagem (user / assistant / system)
    chat-input.tsx             # textarea + botões (send, stop, attach context)
    chat-sidebar.tsx           # lista de conversas + botão nova + settings
    chat-settings-modal.tsx    # provider/model/API key
    chat-empty-state.tsx       # CTA inicial
    index.ts
  lib/chat/
    store.ts                   # Zustand store (conversations, streaming, config)
    persistence.ts             # idb-keyval adapter
    ipc.ts                     # type-safe wrappers sobre window.memAPI.chat
    prompts.ts                 # system prompt base + formatador de contexto RAG
    types.ts                   # ChatMessage, ChatRole, ChatProvider, etc.
  i18n/pt-BR/chat.json         # strings (pt-BR + en-US)
  i18n/en-US/chat.json

electron/
  ipc/chat.ts                  # handlers mem:chat:send / abort / config / mem-search
  ipc/constants.ts             # adicionar IPC_CHANNELS.MEM_CHAT_*
  services/chat/
    llm.ts                     # wrapper AI SDK
    config.ts                  # leitura/escrita de chat-config.json
    mem-search.ts              # spawn mem CLI + parse JSON

.vite/build/                   # output (gerado pelo build)
docs/adr/051-viewer-chat-llm-provider-and-ipc-streaming.md
.specs/features/feat-viewer-chat/
  spec.md                      # este arquivo
  tasks.md                     # breakdown
  validation.md                # após Execute
```

## Critérios de quality gate (Phase 1 close)

- `gofmt -l .` clean
- `go build ./...` exit 0
- `go test -count=1 ./...` todos PASS
- `npm run typecheck` exit 0
- `npm run test` (vitest) PASS — incluindo testes novos do store
- `npm run smoke` exit 0 + screenshot `screenshot-qg-feat-chat.png`
- `biome lint` não cresce além dos 74 pré-existentes
- ADR-051 merged em `develop` antes do PR da feature
- `.agents/skills/my-memory/SKILL.md` atualizado (§8.5 nova seção "Chat no viewer")
