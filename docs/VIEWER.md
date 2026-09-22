# Viewer Desktop (Electron Forge 7 + React 19 + Vite 8 + shadcn v4 + Tailwind 4)

> Phase 1 MVP — bundled Cytoscape rendering `data-central.json` sem dependências de CDN.
> Decisões arquiteturais: [ADR-048](adr/048-viewer-electron-react-vite-shadcn.md) (stack) + [ADR-049](adr/049-viewer-restructure-electron-forge-and-src-split.md) (layout split).

O Viewer Desktop é a substituição moderna dos antigos `graph.html` /
`graph-v2.html` (Cytoscape + Tailwind via CDN). Foi **deletado** em
2026-09-21 quando Felipe pediu rebuild com stack moderno (ADR-048).
Em 2026-09-22, o codebase migrou do template
[`LuanRoger/electron-shadcn`](https://github.com/LuanRoger/electron-shadcn)
e depois foi separado em `electron/` + `src/` (ADR-049) para tornar o
limite de processo visível no filesystem.

---

## Setup

### Pré-requisitos

| Ferramenta | Versão | Notas |
| :--- | :--- | :--- |
| **Node.js** | 20 LTS ou superior | Compatível com Electron 44 |
| **npm** | 10+ | Bundled com Node 20+ |
| **Windows / Linux / macOS** | qualquer | Cross-platform via Electron |
| **Plataforma alvo Electron** | Windows 10+/Linux com X11/macOS 11+ | Electron 44 |

### Comandos

```bash
# 1. Instalar dependências (root único — single-package layout)
npm ci

# 2. Dev mode (Vite HMR + Electron Forge start)
npx electron-forge start
#   ou simplesmente:
npm start

# 3. Build production (gera .vite/build/{main,preload}.js + .vite/renderer/main_window/)
npx electron-forge package
#   ou:
npm run package

# 4. Build installers (squirrel/deb/rpm/zip; cross-platform)
npx electron-forge make

# 5. Type check estrito (TypeScript strict, electron/ + src/)
npm run typecheck

# 6. Test suite (vitest + jsdom)
npm test

# 7. Smoke test Electron (Playwright E2E; roda electron-forge package antes)
npm run smoke
```

> **`npm run dev` foi removido.** O template LuanRoger usa `electron-forge start`
> que inicia o Vite dev server + Electron automaticamente. Não há script
> `dev` no `package.json`.

### Estrutura de Diretórios

```text
electron/                            ← Electron main process ONLY
  main.ts                            BrowserWindow + CSP + IPC stubs
  preload.ts                         contextBridge.memAPI (window.memAPI)
  ipc/
    context.ts                       IPCContext (Phase 1 no-op)
    handler.ts                       rpcHandler stub
    manager.ts                       IPC manager (Phase 1 no-op)
    router.ts                        channel router stub
  constants/
    index.ts                         IPC_CHANNELS, LOCAL_STORAGE_KEYS, inDevelopment
  utils/
    path.ts                          getBasePath (ESM-safe dirname)
  types.d.ts                         MAIN_WINDOW_VITE_* magic consts (VitePlugin)

src/                                 ← Renderer (React 19, flat)
  app.tsx                            React root component
  renderer.ts                        Vite renderer entry (side-effect import)
  components/
    graph/
      graph-view.tsx                 Cytoscape canvas + node tooltip
      index.ts                       barrel export
    locale-switcher.tsx              pt-BR / en-US dropdown
    ui/                              shadcn v4 primitives (button, card, tabs, ...)
  layouts/
    layout.tsx                       Topbar + Sidebar + Tabs shell
  lib/
    cytoscape-init.ts                Cytoscape ESM init + cose layout
  localization/
    i18n.ts                          i18next bootstrap
    langs.ts                         locale metadata
    language.ts                      language helpers
    locales/
      pt-BR.json                     default (per VE-16)
      en-US.json
  styles/
    global.css                       Tailwind 4 + shadcn HSL variables
  tests/
    unit/                            vitest (sum.test.ts + setup.ts)
    e2e/                             playwright (example.test.ts = smoke)
  types/
    css.d.ts                         CSS module declarations
  utils/
    tailwind.ts                      cn() helper (clsx + tailwind-merge)

forge.config.ts                      Electron Forge config (VitePlugin, makers, fuses)
vite.main.config.mts                 Vite config — electron main process
vite.preload.config.mts              Vite config — preload
vite.renderer.config.mts             Vite config — renderer (React + Tailwind)
playwright.config.ts                 Playwright config (testDir: ./src/tests/e2e)
components.json                      shadcn v4 config (style: radix-mira)
biome.jsonc                          biome lint+format config
tsconfig.json                        TypeScript strict, paths: @/* e @electron/*

scripts/                              Go release tooling (não relacionado ao viewer)
```

### Path Aliases

| Alias | Resolves to | Quem usa |
| :--- | :--- | :--- |
| `@/*` | `./src/*` | renderer (app.tsx, components/, layouts/, lib/, localization/) |
| `@electron/*` | `./electron/*` | main, preload, ipc handlers |

Aliases configurados em:
- `tsconfig.json` (TypeScript IDE/check)
- `vite.main.config.mts` (Vite resolve no build do main)
- `vite.preload.config.mts` (Vite resolve no build do preload)

Renderer (`src/components/...`, `src/app.tsx`, `src/layouts/layout.tsx`)
só importa de `@/*`. Main + preload (`electron/main.ts`,
`electron/preload.ts`, `electron/ipc/*`) só importa de `@electron/*`.
Imports cruzando o limite de processo vão **sempre** via
`window.memAPI` (preload) — nunca via path alias.

---

## Stack

| Camada | Tecnologia | Versão | Por quê |
| :--- | :--- | :--- | :--- |
| **Forge** | Electron Forge | 7 | Makers (squirrel/deb/rpm/zip) + VitePlugin |
| **Runtime** | Electron | 44.4 | Desktop cross-platform; IPC tipado |
| **Renderer** | React | 19.3 | Componentização + Hooks + React Compiler |
| **Build** | Vite | 8 | HMR rápido; ESM nativo |
| **TypeScript** | TypeScript | 6 strict | Type safety (ADR-002 philosophy) |
| **Styling** | Tailwind CSS | 4 (CSS-first) | `@tailwindcss/vite` plugin |
| **Components** | shadcn v4 (Radix UI) | latest | Copy-paste; Radix a11y; Tailwind-styled |
| **Graph** | Cytoscape | 3.30 | bundled ESM (zero-CDN) |
| **i18n** | i18next + react-i18next | 26 / 17 | Default `pt-BR`, fallback `en-US` |
| **Lint/Format** | Biome | 2.5 | Rust-based, fast (substitui ESLint+Prettier) |
| **Unit Test** | Vitest | 5 | jsdom; user-event; @testing-library |
| **E2E / Smoke** | Playwright | 1.63 | Electron API (`_electron.launch`) + Vite dev |

---

## Locale Switching

O Viewer suporta **dois locales** carregados como bundles estáticos
(sem CDN):

- **`pt-BR`** (default) — usado quando `localStorage['mem.locale']` é null.
- **`en-US`** — fallback quando uma chave falta em `pt-BR`.

### Mecanismo

1. `src/localization/i18n.ts` lê `localStorage['mem.locale']` no boot. Se null, usa `pt-BR`.
2. `src/components/locale-switcher.tsx` (topbar, dropdown shadcn) abre um menu com:
   - `Português (BR)` (`pt-BR`)
   - `English (US)` (`en-US`)
3. Ao selecionar, persiste a escolha em `localStorage['mem.locale']` e chama `i18n.changeLanguage(code)`. Reload preserva.
4. Strings traduzidas via `useTranslation()`: `t('tab.graph')`, `t('tab.code')`, `t('badge.dark')`, `t('phase2.coming_soon')`, etc.

### Adicionar nova string

1. Adicione a chave em **ambos** os arquivos `src/localization/locales/{pt-BR,en-US}.json`.
2. Use `t('minha.chave')` no componente.
3. Rode `npm test` — falha se a chave existe em um locale mas não no outro.

---

## Phase 2 Roadmap

Decomposed em `.specs/features/feat-viewer-electron-phase-2/tasks.md`.
Ordem de execução recomendada:

| # | Front | ADR | Status |
| :--- | :--- | :--- | :--- |
| F4 | CSP strict (remove `'unsafe-inline'` via Vite nonce) | **ADR-049 promotion** | pending |
| F1 | Real IPC (`mem:dataset:load`, `mem:cli:run`, `mem:reveal` via `child_process.spawn`) | continuation | pending |
| F2 | Code symbols no grafo (fan-in ADR-047) | ADR-047 + new ADR | pending |
| F3 | Syntax highlight (Shiki) | new ADR | pending |
| F5 | `electron-forge make` cross-platform (fix makers filtered em win32) | new ADR or PR | pending |

---

## Chat LLM (ADR-051)

A aba **Chat** (`src/components/chat/`) é uma extensão da Phase 2 que cabe
inteira no stack atual (Vite + React 19 + shadcn/ui) sem introduzir Next.js
ou outro framework. Padrões de referência adaptados do
[trendy-design/llmchat](https://github.com/trendy-design/llmchat) (Next.js 14
+ AI SDK + Zustand + Dexie) para o esqueleto Vite + Electron já existente.

### Capacidades (Phase 1 da feature `feat-viewer-chat`)

- Multi-provider via Vercel AI SDK: **Anthropic** (default — Claude Opus/Sonnet/Haiku 4.x) e **OpenAI** (GPT-4o, o3, o3-mini).
- Streaming token a token via IPC `webContents.send('mem:chat:delta', …)` (latência 5-15ms por hop).
- Persistência local de múltiplas conversas em IndexedDB (`idb-keyval`); sobrevive a restart.
- RAG opcional via `mem search --json` rodando como subprocess no main process; top-5 snippets injetados no system prompt.
- API keys **nunca** saem do main process — gravadas em `app.getPath('userData')/chat-config.json`; renderer vê apenas `hasApiKey: boolean`.

### Arquitetura

```
┌────────────────── Renderer (React 19 + Vite) ──────────────────┐
│  components/chat/*  (shadcn Textarea, Button, Card)            │
│       │                                                        │
│       ▼                                                        │
│  lib/chat/store.ts  (Zustand: conversations + streaming state) │
│       │                                                        │
│       ▼                                                        │
│  lib/chat/ipc.ts    (type-safe wrappers de window.memAPI.chat) │
└──────────────────────────┬─────────────────────────────────────┘
                           │ contextBridge (electron/preload.ts)
┌──────────────────────────▼─────────────────────────────────────┐
│                  Main Process (Node 22)                       │
│  ipc/chat.ts         (ipcMain.handle + webContents.send)      │
│       │                                                        │
│       ▼                                                        │
│  services/chat/llm.ts        (Vercel AI SDK — streamText)      │
│  services/chat/mem-search.ts (child_process.execFile mem)     │
│  services/chat/config.ts     (JSON em app.getPath('userData')) │
└────────────────────────────────────────────────────────────────┘
```

### IPC Surface (Phase 1)

| Channel | Direção | Payload |
|---|---|---|
| `mem:chat:send` | renderer → main (invoke) | `{conversationId, messages, system?}` → `{requestId}` |
| `mem:chat:abort` | renderer → main (invoke) | `{requestId}` → `{aborted: boolean}` |
| `mem:chat:delta` | main → renderer (event) | `{requestId, delta}` |
| `mem:chat:done` | main → renderer (event) | `{requestId, totalChars, latencyMs}` |
| `mem:chat:error` | main → renderer (event) | `{requestId, error: {kind, message}}` |
| `mem:chat:mem-search` | renderer → main (invoke) | `{query, topK?}` → `{results: [...]}` |
| `mem:chat:config-get` | renderer → main (invoke) | `{}` → `{provider, model, useMemory, hasApiKey}` (nunca retorna a key) |
| `mem:chat:config-set` | renderer → main (invoke) | `{provider?, model?, apiKey?, useMemory?}` → `{ok}` |

### Componentes

| Arquivo | Função |
|---|---|
| `chat-view.tsx` | Raiz da aba. Compõe sidebar + thread + input + settings modal. Orquestra streaming (wiring de listeners IPC). |
| `chat-sidebar.tsx` | Lista de conversas + nova conversa + abre settings. |
| `chat-thread.tsx` | ScrollArea auto-rolante para o fim com cada novo delta. |
| `chat-message.tsx` | Bolha user/assistant/system com MarkdownLite (regex-based: code, bold, headings, lists). |
| `chat-input.tsx` | Textarea + Send/Stop. Enter envia; Shift+Enter quebra linha. |
| `chat-settings-modal.tsx` | Provider/model/API key/useMemory. API key é **write-only**. |

### Configuração

1. Abrir a aba Chat no viewer.
2. Clicar no ícone de engrenagem (sidebar) → **Configurações do Chat**.
3. Escolher **Provider** (Anthropic ou OpenAI).
4. Escolher **Modelo**.
5. Colar a **API Key** (write-only — não conseguimos ler de volta).
6. (Opcional) Ativar **Usar memória do vault** para injetar contexto via `mem search`.
7. **Salvar**.

A key é gravada em `<userData>/chat-config.json`. Nunca toca o renderer, o bundle, ou o IndexedDB.

### Recursos futuros (Plano B deferred)

- **Tiptap rich-text** (input + markdown render rico via `react-markdown`) — quando ≥3 pedidos.
- **Tool calling / agent loop** (chat invoca `mem note create`, `mem search`, etc.) — quando Felipe pedir.
- **Sync entre devices** — converter conversas em notas via `mem compile`.

Ver `.specs/features/feat-viewer-chat/spec.md` + `tasks.md` + `ADR-051` para detalhes.

---

## Restrições de Phase 1

- **`data-central.json` é estático**: copiado para `public/data-central.json` durante o build, fetchado pelo renderer em runtime. Phase 2 substitui por IPC via `mem graph export`.
- **CSP relaxado**: `script-src 'self' 'unsafe-inline' 'unsafe-eval'` em dev (HMR Vite + `unsafe-eval` para o Babel/React Compiler). Hardcoded em `electron/main.ts` via `session.webRequest.onHeadersReceived`. Phase 2 endurece com nonce (F4).
- **IPC stubs**: `electron/main.ts` registra `ipcMain.handle(MEM_DATASET_LOAD, ...)` que rejeita com `"not implemented (Phase 2)"`. Renderer chama `window.memAPI.loadDataset()` que loga warning + rejeita. Phase 2 (F1) substitui por handlers reais que fazem `child_process.spawn('./mem.exe', [...])`.
- **Sem auto-update**: `update-electron-app` está instalado mas sem wiring. Phase 2 (F5 ou depois) configura.
- **Theme hardcoded dark**: `<html class="dark">` em `index.html`. Toggle via `next-themes` virá em Phase 2 (F5+).
- **electron-forge `package` não produz `out/` em win32**: makers `MakerSquirrel/Deb/Rpm` filtrados em win32, `MakerZIP` restrito a `["darwin"]`. Bundles via `.vite/` funcionam; installer cross-platform é F5.

---

## Comandos úteis

```bash
# Rodar smoke test + screenshot (1 passed esperado, exit 0)
npm run smoke

# Rodar Electron standalone (assume build via `npm run package`)
npm start

# Build installers (Phase 2 F5)
npm run make
```

---

## Referências

- **[ADR-048](adr/048-viewer-electron-react-vite-shadcn.md)** — Stack original (Electron + React + Vite + shadcn/ui).
- **[ADR-049](adr/049-viewer-restructure-electron-forge-and-src-split.md)** — Layout split (`electron/` ↔ `src/`) + path aliases.
- **[ADR-037](adr/037-rewrite-viewer-com-vite-vanilla-ts.md)** — Superseded por ADR-048.
- **[`LuanRoger/electron-shadcn`](https://github.com/LuanRoger/electron-shadcn)** — referência canônica (1.4.1, MIT) para Electron + shadcn/ui.
- **`.specs/features/feat-viewer-electron-phase-2/tasks.md`** — Phase 2 sub-fronts (F1-F5).

> Veja também a spec ativa em `.specs/features/feat-viewer-electron/spec.md`
> (marcada SUPERSEDED no topo após a migração de layout).