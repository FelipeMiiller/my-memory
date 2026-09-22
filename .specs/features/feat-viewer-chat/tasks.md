# Tasks: feat-viewer-chat — Chat LLM no viewer Electron

> Breakdown em ordem de execução. Cada task tem `Tests`, `Gate` e `Depends on`.

## Phase 0 — Bootstrap (decisões + infra)

### T1 — ADR-051 (Provider + IPC streaming + RAG)
**Description:** Documentar a decisão arquitetural de provider abstraction (AI SDK), onde roda o LLM call (main process), streaming via IPC events, persistência IndexedDB, e integração RAG via `mem search`. Status: **Proposed → Accepted**.

**Tests:** `docs/adr/051-viewer-chat-llm-provider-and-ipc-streaming.md` existe, linkado do STATE.md, revisado pelo operador.
**Gate:** ADR-051 merged em develop antes de T2.
**Depends on:** —
**Where:** `docs/adr/051-viewer-chat-llm-provider-and-ipc-streaming.md`

### T2 — Dependências + shadcn Textarea
**Description:** Adicionar `ai`, `@ai-sdk/openai`, `@ai-sdk/anthropic`, `zustand`, `idb-keyval`, e gerar `src/components/ui/textarea.tsx` via `npx shadcn@latest add textarea`.

**Tests:** `package.json` tem as deps; `src/components/ui/textarea.tsx` existe; `npm install` exit 0.
**Gate:** `npm run typecheck` exit 0.
**Depends on:** —
**Where:** `package.json`, `package-lock.json`, `src/components/ui/textarea.tsx`

### T3 — IPC channels + types compartilhados
**Description:** Adicionar `IPC_CHANNELS.MEM_CHAT_SEND`, `MEM_CHAT_ABORT`, `MEM_CHAT_CONFIG_GET`, `MEM_CHAT_CONFIG_SET`, `MEM_CHAT_MEM_SEARCH`, e `MEM_CHAT_DELTA` (evento one-way). Adicionar types em `electron/types.d.ts` para `ChatMessage`, `ChatProvider`, `ChatConfig`.

**Tests:** types compilam (`tsc --noEmit`).
**Gate:** `npm run typecheck` exit 0.
**Depends on:** T2.
**Where:** `electron/constants/index.ts`, `electron/types.d.ts`

## Phase 1 — Main process (LLM + RAG + config)

### T4 — `services/chat/config.ts` (ler/gravar chat-config.json)
**Description:** API no main para `getConfig()` / `setConfig({provider, model, apiKey, useMemory})`. Persiste em `app.getPath('userData')/chat-config.json`. Fail-safe: se arquivo corrompido, retorna defaults e loga warning.

**Tests:** 2 vitest tests (write+read, corrupted-file fallback).
**Gate:** testes PASS.
**Depends on:** T3.
**Where:** `electron/services/chat/config.ts`, `electron/services/chat/__tests__/config.test.ts`

### T5 — `services/chat/llm.ts` (wrapper AI SDK)
**Description:** Função `streamChat({provider, model, apiKey, system, messages}, onDelta, signal)` que cria o model client (Anthropic ou OpenAI via `@ai-sdk/anthropic`/`@ai-sdk/openai`), chama `streamText`, e invoca `onDelta(text)` para cada chunk. Suporta `AbortController` via `signal`. Mapeia erros do AI SDK → `ChatError` tipado (`{kind: 'auth'|'rate_limit'|'network'|'unknown', message}`).

**Tests:** 2 vitest tests (mock fetch para Anthropic streaming, abort mid-stream).
**Gate:** testes PASS.
**Depends on:** T3.
**Where:** `electron/services/chat/llm.ts`, `electron/services/chat/__tests__/llm.test.ts`

### T6 — `services/chat/mem-search.ts` (subprocess mem search)
**Description:** Função `memSearch(query, topK=5)` que executa `bin/mem.exe search "$query" --json --limit $topK` via `child_process.execFile`. Parseia output JSON. Retorna `{results: Array<{path, title, snippet, score}>}`. Retorna `[]` se mem não estiver no PATH (fail-soft).

**Tests:** 2 vitest tests (successo com output válido, fail-soft com ENOENT).
**Gate:** testes PASS.
**Depends on:** —
**Where:** `electron/services/chat/mem-search.ts`, `electron/services/chat/__tests__/mem-search.test.ts`

### T7 — `ipc/chat.ts` (handlers + streaming via webContents.send)
**Description:** Registra handlers para `MEM_CHAT_SEND` (gera `requestId`, faz `memSearch` se `useMemory`, monta system prompt, chama `streamChat` e emite `MEM_CHAT_DELTA` por chunk, depois `MEM_CHAT_DONE` ou `MEM_CHAT_ERROR`), `MEM_CHAT_ABORT` (cancela `AbortController` correspondente), `MEM_CHAT_CONFIG_GET`/`SET`. Mantém mapa `Map<requestId, AbortController>` para abort.

**Tests:** 1 vitest test (handler send valida payload e rejeita inválido).
**Gate:** testes PASS.
**Depends on:** T4, T5, T6.
**Where:** `electron/ipc/chat.ts`, `electron/ipc/chat.test.ts`

## Phase 2 — Preload (memAPI.chat surface)

### T8 — preload expõe `window.memAPI.chat`
**Description:** Adicionar objeto `chat` em `memAPI` com métodos: `send`, `abort`, `getConfig`, `setConfig`, `memSearch`. Listener `onDelta(callback): unsubscribe` para receber eventos de streaming.

**Tests:** typecheck exit 0; manual: `console.log(window.memAPI.chat)` mostra objeto.
**Gate:** `npm run typecheck` exit 0.
**Depends on:** T7.
**Where:** `electron/preload.ts`

## Phase 3 — Renderer (UI + state + persistence)

### T9 — `lib/chat/types.ts` + `lib/chat/ipc.ts`
**Description:** Types `ChatMessage`, `ChatRole`, `ChatProvider`, `ChatConversation`, `ChatConfig`. Wrapper type-safe `chatIpc.send(...)`, `chatIpc.onDelta(cb)`, etc., usando `window.memAPI.chat`.

**Tests:** —
**Gate:** typecheck exit 0.
**Depends on:** T8.
**Where:** `src/lib/chat/types.ts`, `src/lib/chat/ipc.ts`

### T10 — `lib/chat/persistence.ts` (idb-keyval)
**Description:** Funções `loadConversations()`, `saveConversation(c)`, `deleteConversation(id)`, `loadConfig()`, `saveConfig(c)`. Usa store `mem-chat-v1`. Schema-versionado pra migração futura.

**Tests:** 2 vitest tests (save+load roundtrip, delete remove).
**Gate:** testes PASS.
**Depends on:** T2.
**Where:** `src/lib/chat/persistence.ts`, `src/lib/chat/__tests__/persistence.test.ts`

### T11 — `lib/chat/store.ts` (Zustand)
**Description:** Store com state: `conversations[]`, `activeConversationId`, `streamingMessageId`, `config`, `error`. Actions: `createConversation`, `deleteConversation`, `selectConversation`, `appendMessage`, `updateMessageContent`, `setStreaming`, `setConfig`, `clearError`. Persistência automática via subscription → `persistence.saveConversation`.

**Tests:** 3 vitest tests (createConversation adiciona, appendMessage atualiza lista, setStreaming marca estado).
**Gate:** testes PASS.
**Depends on:** T10.
**Where:** `src/lib/chat/store.ts`, `src/lib/chat/__tests__/store.test.ts`

### T12 — `lib/chat/prompts.ts` (system prompt base + RAG formatter)
**Description:** Constante `BASE_SYSTEM_PROMPT`. Função `buildSystemPrompt(config, ragContext)` que monta o system final incluindo `{top_results}` formatados se `config.useMemory && ragContext`.

**Tests:** 1 vitest test (com/sem RAG produz outputs diferentes).
**Gate:** testes PASS.
**Depends on:** —
**Where:** `src/lib/chat/prompts.ts`, `src/lib/chat/__tests__/prompts.test.ts`

### T13 — `components/chat/*` (componentes visuais)
**Description:** Implementa `chat-sidebar.tsx`, `chat-empty-state.tsx`, `chat-thread.tsx`, `chat-message.tsx`, `chat-input.tsx`, `chat-settings-modal.tsx`, `chat-view.tsx` (composição). Layout 2 colunas (sidebar + thread). Auto-scroll on new message. Markdown render básico em `chat-message.tsx` (sem deps — regex headings/bold/code/list). Streaming: `chat-input` mostra botão "Stop" durante streaming.

**Tests:** 1 vitest test (render de `chat-message` com mensagem user/assistant).
**Gate:** testes PASS.
**Depends on:** T11, T12.
**Where:** `src/components/chat/*.tsx`

### T14 — Layout integrando Tab "Chat"
**Description:** Em `src/layouts/layout.tsx`, adicionar `SidebarLink` + `TabsTrigger value="chat"` + `TabsContent value="chat"`. Adicionar strings em `src/localization/langs.ts` (`nav.chat`, `nav.chatDesc`, `tab.chat`).

**Tests:** typecheck exit 0.
**Gate:** `npm run typecheck` exit 0.
**Depends on:** T13.
**Where:** `src/layouts/layout.tsx`, `src/localization/langs.ts`

## Phase 4 — i18n

### T15 — Strings pt-BR + en-US
**Description:** Criar `src/localization/langs/pt-BR/chat.json` e `en-US/chat.json` com ~30 chaves (nav, tab, sidebar, empty state, input placeholders, settings labels, errors).

**Tests:** —
**Gate:** `npm run typecheck` exit 0.
**Depends on:** T14.
**Where:** `src/localization/langs/{pt-BR,en-US}/chat.json`

## Phase 5 — Smoke + Docs + PR

### T16 — Smoke gate (electron-forge package + playwright)
**Description:** Atualizar `src/tests/e2e/example.test.ts` (ou criar novo) que abre o Electron, navega para a aba Chat, digita "ping", espera resposta (mock provider ou usa key de teste), captura screenshot `screenshot-qg-feat-chat.png`. Atualizar `npm run smoke` se necessário.

**Tests:** smoke exit 0 + screenshot gerado.
**Gate:** `npm run smoke` exit 0.
**Depends on:** T15.
**Where:** `src/tests/e2e/chat.test.ts`, `package.json`

### T17 — Docs (VIEWER.md + skill my-memory §8.5)
**Description:** Adicionar seção "Chat no viewer" em `docs/VIEWER.md` (criado no commit `05aa6d2`). Atualizar `.agents/skills/my-memory/SKILL.md` §8.5 com link pra nova seção + §12 com nota sobre `mem:chat:*` IPC.

**Tests:** —
**Gate:** diff revisado.
**Depends on:** T16.
**Where:** `docs/VIEWER.md`, `.agents/skills/my-memory/SKILL.md`

### T18 — STATE.md + ADR-051 link + validation.md
**Description:** Adicionar ADR-051 em `Decisions` de `STATE.md`. Criar `.specs/features/feat-viewer-chat/validation.md` após smoke + testes PASS (executa Verifier).

**Tests:** —
**Gate:** validation.md reporta PASS-WITH-DEFER ou PASS.
**Depends on:** T17.
**Where:** `.specs/STATE.md`, `.specs/features/feat-viewer-chat/validation.md`

### T19 — Commit atômico + push + PR target develop
**Description:** Commits por phase (0-5) seguindo Conventional Commits. PR target `develop`. Aguarda review do Felipe (regra: não promove pra main sozinho).

**Tests:** —
**Gate:** PR aberto, CI verde.
**Depends on:** T18.
**Where:** `git log`, GitHub PR

---

## Phase Execution Map

```
T1 (ADR-051)
  ↓
T2 (deps) → T3 (IPC types)
                ↓
                T4 (config svc) ──┐
                T5 (LLM svc) ────┤
                T6 (mem-search) ──┤
                                  ↓
                                  T7 (IPC handlers)
                                    ↓
                                    T8 (preload memAPI.chat)
                                      ↓
                                      T9 (renderer types + ipc wrappers)
                                        ↓
                                        T10 (persistence) ──┐
                                        T11 (store) ───────┤
                                        T12 (prompts) ─────┤
                                                           ↓
                                                           T13 (componentes)
                                                             ↓
                                                             T14 (layout Tab)
                                                               ↓
                                                               T15 (i18n)
                                                                 ↓
                                                                 T16 (smoke + screenshot)
                                                                   ↓
                                                                   T17 (docs)
                                                                     ↓
                                                                     T18 (STATE + validation)
                                                                       ↓
                                                                       T19 (PR)
```

19 tasks, ~3-4 horas de trabalho focado em single-session. > 8 ⇒ recomenda **worker batches** paralelos em T13 (1 worker = componentes) e T15 (1 worker = i18n). Demais tasks sequenciais por dependência.
