# ADR-051: Chat LLM no Viewer — Provider Abstraction + IPC Streaming + RAG via `mem search`

- **Date**: 2026-09-22
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: viewer, chat, llm, ai-sdk, ipc, electron, streaming, rag, security, mvp

## Context and Problem Statement

O viewer Electron do my-memory (ADR-048 Accepted, Phase 1 MVP commit `0f09ad8`) já entrega:

- Shell React 19 + Vite 8 + shadcn/ui v4 + Tailwind 4 + i18next
- IPC stub (`mem:dataset:load`, `mem:cli:run`) em `electron/main.ts` rejeitando com `"not implemented (Phase 2)"`
- Tabs Graf (ativo), Code/Staleness (Phase 2 placeholders)
- Persistência leve via `localStorage` (locale)
- Integração com `mem.exe` por **spec** (F1 Phase 2) mas não implementada

Em 2026-09-22, Felipe pediu **fluxo completo de chat** no viewer tomando como referência arquitetural o [trendy-design/llmchat](https://github.com/trendy-design/llmchat) (Next.js 14 + AI SDK + Tiptap + Zustand + Dexie + shadcn + multi-provider).

A pergunta que este ADR responde: **como adicionar um chat LLM nativo no viewer Electron, mantendo o stack atual (Vite + React), expondo múltiplos providers (Anthropic, OpenAI), com segurança ponta-a-ponta (API keys nunca saem do main process), streaming token-a-token via IPC, persistência local de conversas e integração opcional com `mem search` como RAG backend?**

## Decision Drivers

- **DR-1**: Manter stack atual (Vite + React 19 + shadcn/ui) — Felipe pediu "mantenha o vite react" em 2026-09-22. **Não introduzir Next.js**.
- **DR-2**: Segurança por design — API keys **nunca** tocam o renderer; toda comunicação de credenciais passa pelo main process.
- **DR-3**: Streaming token-a-token para UX moderna (resposta incremental, sensação de baixa latência).
- **DR-4**: Provider abstraction — adicionar/swap providers sem reescrever UI (Anthropic Claude default; OpenAI opcional).
- **DR-5**: Integração com my-memory como RAG backend — opcional, via `mem search` rodando no main como subprocess.
- **DR-6**: Persistência local de múltiplas conversas (sobrevive a restart do app).
- **DR-7**: Compatibilidade com Phase 2 F1 (`mem:cli:run` IPC) — chat usa o mesmo pipeline IPC stub + mem subprocess já planejado.
- **DR-8**: Bundle pequeno — sem Tiptap/Dexie (deps grandes). Adicionar apenas o estritamente necessário.
- **DR-9**: Type safety end-to-end (renderer ↔ IPC ↔ main ↔ provider).
- **DR-10**: Testabilidade — store e serviços com vitest; IPC handlers mockáveis.

## Considered Options

### 1. Chat como servidor Node externo (HTTP/SSE)
Subir um servidor Node ao lado do Electron, exposto em `localhost:NNNN`, que faz proxy para LLM providers. Renderer faz fetch direto.

- ❌ Adiciona processo/servidor a mais
- ❌ API keys transitam pelo localhost (mesma máquina, mas ainda vetor de ataque)
- ❌ CORS/CSP mais complexo
- ❌ Sem benefícios reais vs IPC
- ✅ Streaming via SSE é trivial

**Rejeitado**: duplica infra que Electron main já provê.

### 2. AI SDK rodando no renderer (browser-side)
Usar Vercel AI SDK direto no renderer (`@ai-sdk/react` + `@ai-sdk/anthropic`). API key fornecida via env no build time ou settings no renderer.

- ❌ **API key exposta no bundle** (mesmo ofuscada, fica no localStorage/IndexedDB legível)
- ❌ CORS do browser para algumas APIs LLM (ex: Anthropic bloqueia browser-origin por padrão)
- ❌ CSP violada (necessita `connect-src` liberado para `api.anthropic.com`)
- ✅ Streaming nativo via `useChat` hook
- ✅ Bundle menor (sem hop IPC)

**Rejeitado**: viola DR-2 (security) e ADR-049 planejado (CSP strict).

### 3. IPC streaming via `webContents.send` + `ipcRenderer.on` (Escolhida)
Renderer chama `window.memAPI.chat.send(...)` via `ipcRenderer.invoke` (request/response). Main process stream tokens emitindo eventos `mem:chat:delta` via `webContents.send`. Renderer acumula via listener.

- ✅ API key 100% no main
- ✅ CSP fica simples (apenas IPC, sem `connect-src` extra)
- ✅ Streaming com baixa latência percebida (~30-50ms por token após LLM responder)
- ✅ Reaproveita preload `contextBridge` (já existe)
- ✅ Type-safe com types compartilhados
- ⚠️ Latência do 1º token: 1 hop IPC (~5-15ms) — negligível vs latência da API
- ⚠️ Erro mid-stream: precisa emitir `mem:chat:error` event após delta parcial

### 4. IPC request/response sem streaming
Renderer chama, espera resposta inteira, recebe.

- ❌ UX ruim (tela parada até resposta completa — pode levar 5-30s)
- ❌ Sem cancelamento mid-flight fácil
- ✅ Mais simples

**Rejeitado**: viola DR-3 (UX moderna de streaming).

## Decision Outcome

**Opção escolhida: IPC streaming via `webContents.send` + `ipcRenderer.on`**, com os componentes:

### Camadas

```
┌─────────────────── Renderer (React 19 + Vite 8) ──────────────────┐
│                                                                  │
│  components/chat/*     ← UI shadcn (Textarea, Button, Card)     │
│       │                                                          │
│       ▼                                                          │
│  lib/chat/store.ts     ← Zustand: conversations, streaming state│
│       │                                                          │
│       ▼                                                          │
│  lib/chat/ipc.ts       ← type-safe wrappers                      │
│       │              window.memAPI.chat.send/abort/config        │
└──────────────────────────────┬───────────────────────────────────┘
                               │ contextBridge (electron/preload)
┌──────────────────────────────▼───────────────────────────────────┐
│                  Main Process (Node 22)                         │
│                                                                  │
│  ipc/chat.ts             ← ipcMain.handle + webContents.send    │
│       │                       MemChatSend → emits delta/done    │
│       ▼                                                          │
│  services/chat/llm.ts    ← Vercel AI SDK (streamText)            │
│       │                     @ai-sdk/anthropic / @ai-sdk/openai │
│       ▼                                                          │
│  services/chat/mem-search.ts ← child_process.execFile bin/mem   │
│                                                                  │
│  services/chat/config.ts ← JSON em app.getPath('userData')      │
│                            chat-config.json (apiKey included)   │
└─────────────────────────────────────────────────────────────────┘
```

### Escolhas técnicas pontuais

| Decisão | Escolha | Por quê |
|---|---|---|
| Provider SDK | **Vercel AI SDK** (`ai` + `@ai-sdk/anthropic` + `@ai-sdk/openai`) | Padrão de mercado, multi-provider, streaming SSE nativo, type-safe. Mesmo pacote usado em llmchat. |
| Streaming | `streamText` do AI SDK + `onChunk` callback no main → `webContents.send('mem:chat:delta', {requestId, delta})` | Latência ~5-15ms por hop, negligível. |
| State global | **Zustand** (1 store `useChatStore`) | 1.2KB gzipped, sem boilerplate, idiomático em Vite/React. |
| Persistência local | **IndexedDB via `idb-keyval`** (600B) | Wrapper mínimo, sem dependências nativas, mesma abordagem de llmchat (que usa Dexie — descartado por ser overkill). |
| RAG | `mem search --json` via `child_process.execFile('./bin/mem.exe', ['search', q, '--json', '--limit', '5'])` | Reaproveita CLI já existente. Fail-soft se mem não estiver no PATH. |
| Config do provider | `app.getPath('userData')/chat-config.json` (escrito pelo main, lido pelo main) | Persiste, não commita, lido apenas no main. Renderer não tem acesso ao path. |
| Input rich-text | **Textarea simples + markdown render básico** (regex headings/bold/code/list) | Tiptap descartado (deps + 50KB+) para Phase 1. Pode entrar depois se virar dor. |
| Cancelamento | `AbortController` por `requestId` no main; `mem:chat:abort(requestId)` do renderer | Padrão Web, integra com `signal` do AI SDK. |
| Erro handling | `ChatError` tipado `{kind: 'auth'|'rate_limit'|'network'|'unknown', message}`; renderer mostra toast sanitizado | Nunca vaza API key ou stack raw. |

### IPC Surface (Phase 1)

| Channel | Direção | Payload |
|---|---|---|
| `mem:chat:send` | renderer → main (invoke) | `{conversationId, messages, provider, model, useMemory}` → `{requestId}` |
| `mem:chat:abort` | renderer → main (invoke) | `{requestId}` → `{aborted: boolean}` |
| `mem:chat:delta` | main → renderer (event) | `{requestId, delta}` |
| `mem:chat:done` | main → renderer (event) | `{requestId, totalTokens?, latencyMs}` |
| `mem:chat:error` | main → renderer (event) | `{requestId, error: ChatError}` |
| `mem:chat:mem-search` | renderer → main (invoke) | `{query, topK?}` → `{results: [...]}` |
| `mem:chat:config-get` | renderer → main (invoke) | `{}` → `{provider, model, useMemory, hasApiKey: boolean}` (nunca retorna a key) |
| `mem:chat:config-set` | renderer → main (invoke) | `{provider, model, apiKey?, useMemory}` → `{ok: boolean}` |

**Importante:** `mem:chat:config-get` **nunca** retorna `apiKey` no payload — apenas `hasApiKey: boolean`. Renderer não tem como ler a key, mesmo pedindo.

### Segurança

- API keys vivem em `chat-config.json` no `userData` (fora do bundle, fora do IndexedDB)
- Renderer não tem `Node integration` (já é `false` em `electron/main.ts`)
- Renderer chama IPC via `contextBridge` (sem acesso direto a `ipcRenderer`)
- Mensagens de erro do provider são **sanitizadas** no main antes de emitir (remove stack traces, paths internos, headers)
- `connect-src` CSP **não precisa** ser ampliado (IPC é `self`-only)
- ADR-049 (CSP strict) **não é bloqueado** por esta feature — 오히려 ajuda (CSP não muda)

### Dependencies adicionadas

```jsonc
{
  "dependencies": {
    "ai": "^4",                  // Vercel AI SDK core
    "@ai-sdk/anthropic": "^1",  // Claude provider
    "@ai-sdk/openai": "^1",     // GPT provider
    "zustand": "^5",            // state management
    "idb-keyval": "^6"          // IndexedDB wrapper
  },
  "devDependencies": {
    // (já temos vitest, @testing-library/react)
  }
}
```

Bundle size impact estimado: ~40KB gzipped (AI SDK + Anthropic + OpenAI providers + Zustand + idb-keyval). Aceitável para Electron (já tem 100MB+ de runtime).

## Consequences

### Positivo

- ✅ API keys seguras por design (nunca saem do main)
- ✅ Multi-provider via AI SDK (swap com 1 linha)
- ✅ Streaming com UX moderna
- ✅ Persistência local sobrevive a restart
- ✅ RAG opcional integrado ao my-memory existente
- ✅ Reaproveita stub IPC do Phase 2 F1 (`mem:cli:run`)
- ✅ ADR-049 (CSP strict) **ganha** com esta feature (não muda CSP)
- ✅ Type-safe end-to-end via types compartilhados em `electron/types.d.ts`
- ✅ Testável: store Zustand + serviços com vitest

### Negativo / Trade-offs

- ⚠️ Latência do 1º token: 1 hop IPC extra (~5-15ms) vs chamada direta do browser. **Negligível** vs latência da API LLM (500ms-2s).
- ⚠️ Complexidade do `requestId` + mapa de `AbortController` no main — bug clássico de "memory leak se abort não for chamado".
- ⚠️ Zustand adiciona dep — alternativa seria `useReducer` puro, mas Zustand compensa em boilerplate (subscriptions, persist integration).
- ⚠️ Markdown render regex-based é frágil — para Phase 1 OK; longo prazo considerar `react-markdown` (~30KB gzipped).
- ⚠️ Chat history persistido em IndexedDB local — **não sincroniza entre devices**. Phase 2 pode integrar com `mem` para virar knowledge atômica.
- ⚠️ Se usuário usar `mem search` com vault não-indexado, retorna `[]` silenciosamente. UX fica estranha ("chat usou memória" mas sem contexto). Fail-soft intencional mas precisa tooltip/empty state claro.

## Deferral — Plano B (alternativas descartadas)

### Plano B: Tiptap + react-markdown como upgrade pós-Phase 1
Se Phase 1 chat virar popular e usuários reclamarem de input/output pobre:

- **Gatilho**: ≥3 feature requests de "rich text input" ou "code blocks mal renderizados"
- **Tarefas**: spec `feat-viewer-chat-rich-text`, adicionar Tiptap + react-markdown, substituir `chat-input.tsx` e `chat-message.tsx`
- **Promoção**: 1 ADR novo + spec + 1 worker batch

### Plano B: Chat com tools (function calling / agent loop)
Se chat precisar invocar `mem search` automaticamente, criar notas, fazer graph queries, etc.:

- **Gatilho**: Felipe pedir "quero que o chat edite notas" ou "chat deve criar nota automaticamente"
- **Tarefas**: spec `feat-viewer-chat-tools`, integrar AI SDK `tools` schema, novos IPC channels (`mem:note:create`, etc.)
- **Promoção**: ADR novo + spec + worker batches

### Plano B: Sync de conversas entre devices
- **Gatilho**: ≥1 pedido explícito de Felipe
- **Tarefas**: spec `feat-viewer-chat-sync`, integrar com `mem` (chat vira notas do vault) ou servidor opcional
- **Promoção**: ADR novo + decisão sobre server vs federated

## Cross-references

- **ADR-048**: Viewer Desktop — Electron + React + Vite + shadcn/ui (Accepted; base deste ADR)
- **ADR-049**: CSP strict planejado (não bloqueado por esta feature)
- **ADR-050**: Threat Model OWASP Top 10 — LLM02 (injection), LLM05 (supply chain), LLM07 (insecure output handling) cobertos pela arquitetura
- **ADR-002**: Go como linguagem principal — chat reutiliza `mem` CLI via subprocess, não duplica lógica
- **ADR-006**: MCP como integração IA — chat é alternativa UX-side ao MCP; não substitui (MCP continua sendo a surface canônica para tools externos)
- **ADR-018**: Compile-not-Retrieve — chat pode opcionalmente usar `mem compile` para cristalizar conversa em nota
- **ADR-043**: event_runtime (Proposed) — Phase 2 pode emitir `chat.message_committed` no barramento

## Implementation Plan

Spec detalhado: `.specs/features/feat-viewer-chat/spec.md`
Tasks breakdown: `.specs/features/feat-viewer-chat/tasks.md`

Resumido:

1. ADR-051 merged (este doc) — antes de T2
2. Deps + shadcn Textarea (T2)
3. IPC channels + types (T3)
4. Services: config (T4), LLM wrapper (T5), mem-search (T6)
5. IPC handlers + streaming (T7)
6. Preload `memAPI.chat` (T8)
7. Renderer: types + ipc wrappers (T9), persistence (T10), store Zustand (T11), prompts (T12)
8. Componentes (T13)
9. Layout Tab integration (T14)
10. i18n (T15)
11. Smoke test + screenshot (T16)
12. Docs (T17)
13. STATE.md + validation.md (T18)
14. PR target develop (T19)

19 tasks, ~3-4 horas de trabalho focado.
