# ADR-048: Viewer Desktop — Electron + React + Vite + shadcn/ui

- **Date**: 2026-09-21
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Supersedes**: ADR-037 (Rewrite do Viewer com Vite + Vanilla TypeScript — **Deferred**)
- **Tags**: viewer, electron, react, vite, shadcn, typescript, tailwind, desktop, mvp

## Context and Problem Statement

O my-memory nasceu com viewer HTML standalone — `graph.html` (ADR-020) evoluiu para `graph-v2.html` (Cytoscape + Tailwind CDN + dark/light toggle). Os ADRs 020, 024, 038 e 049 planejado documentam uma família de capacidades do viewer (visualização cirúrgica triptych, site único + dataset fixo, CSP strict) que a stack atual **não alcança de forma agregada**:

1. **Bundle único, sem CDN**: `graph-v2.html` carrega Tailwind + cytoscape via `unpkg.com` (ADR-050 LLM03 — supply chain).
2. **CSP strict (ADR-049)**: scripts inline + `<script src="https://...">` violam proposta ADR-049 de `default-src 'none'`.
3. **Componentização**: o viewer HTML único mistura rendering + estado + estilo num arquivo de 49 KB; adicionar code symbols, staleness dashboard e busca incrementalmente **dobra** a complexidade.
4. **Distribuição desktop**: o viewer atual vive no repo; usuário precisa de `python -m http.server` ou abrir `file://` (sem IPC para backend Go).
5. **ADR-037 Deferred**: o rewrite anterior previa Vite + Vanilla TS — Felipe pediu **Electron + React + shadcn/ui** em 2026-09-21 e delete dos HTMLs legados para refazer do zero.

A pergunta que este ADR responde: **qual stack usar para o viewer desktop do my-memory, entregando bundle zero-CDN, componentização sólida e alinhamento com ADR-049 (CSP strict) + ADR-050 (LLM05 viewer controls)?**

## Decision Drivers

- **DR-1**: Stack pedido explicitamente pelo operador em 2026-09-21 ("consegue cria em electron com react vite e https://ui.shadcn.com/").
- **DR-2**: Bundle **zero-CDN** — Tailwind/cytoscape bundled localmente (fecha ADR-050 LLM03).
- **DR-3**: CSP strict (`default-src 'none'; img-src 'self' data:`) sem `<iframe>` ou `<img src="http...">` (fecha ADR-049 planejado + ADR-050 LLM05).
- **DR-4**: IPC tipado entre main process (Node) e renderer (React) via `contextBridge` + `ipcRenderer.invoke` — preload script como única superfície exposta ao renderer.
- **DR-5**: Componentização reutilizável via shadcn/ui (Radix UI + Tailwind) — copy-paste components owned pelo repo (não dependência externa versionada).
- **DR-6**: Compatibilidade com Go-first stack (ADR-002) — Electron app **invoca** o binário `mem.exe` via CLI args ou socket; **não** duplica lógica de domínio.
- **DR-7**: Cross-platform build via `electron-builder` (Windows .exe, Linux AppImage/deb, macOS .dmg).
- **DR-8**: Phase 1 = MVP funcional (`npm run dev` abre Electron + 1 tab Graf carregando `data-central.json` estático). Phase 2 = IPC para `mem code-*` + code symbols + staleness dashboard + shiki syntax highlight.

## Considered Options

1. **Status quo (HTML legados)** — *descartado*: Felipe pediu delete em 2026-09-21.
2. **Tauri + React + Vite + shadcn** — *descartado*: stack similar, mas Felipe pediu Electron explicitamente. Tauri é alternativa para ADR futuro se bundle size crítico.
3. **Web app puro (sem Electron)** — *descartado*: viewer continua preso a hosting local; sem IPC para `mem.exe`; usuário precisa servir manualmente.
4. **Electron + React + Vite + TypeScript + Tailwind + shadcn/ui** — **Opção Escolhida**.

## Decision Outcome

**Opção 4 escolhida**: substituir os HTMLs legados por desktop app Electron + React 18 + Vite 5 + TypeScript + Tailwind CSS + shadcn/ui. Estrutura monorepo leve:

```
my-memory/
├── electron/                  # NEW — main process + preload + Electron-specific
│   ├── main.ts                # BrowserWindow lifecycle, IPC handlers
│   ├── preload.ts             # contextBridge exposing safe IPC API
│   ├── tsconfig.json
│   └── tsconfig.node.json
├── viewer/                    # NEW — Vite + React + shadcn/ui
│   ├── src/
│   │   ├── main.tsx           # React entry
│   │   ├── App.tsx            # shadcn/ui Layout: Sidebar + Tabs (Grafo, Code, Staleness)
│   │   ├── components/
│   │   │   ├── ui/            # shadcn/ui primitives (Button, Tabs, Card, ScrollArea, ...)
│   │   │   └── graph/         # Cytoscape wrapper
│   │   ├── lib/
│   │   │   ├── cytoscape.ts   # bundled ESM
│   │   │   └── utils.ts       # shadcn cn() helper
│   │   ├── styles/
│   │   │   └── globals.css    # Tailwind base + shadcn variables
│   │   └── views/
│   │       ├── GraphView.tsx  # Phase 1: tabs loading data-central.json
│   │       ├── CodeView.tsx   # Phase 2 placeholder
│   │       └── StalenessView.tsx # Phase 2 placeholder
│   ├── index.html             # CSP strict meta
│   ├── tailwind.config.ts
│   ├── postcss.config.js
│   ├── tsconfig.json
│   ├── vite.config.ts         # plugins: react + electron (electron-vite)
│   └── package.json           # deps: react 18, vite 5, electron 30, shadcn/ui
├── package.json               # NEW — workspace root
├── electron-builder.yml       # NEW — cross-platform packaging
└── ... (resto do projeto Go inalterado)
```

### Pontos arquiteturais críticos

- **CSP strict em `viewer/index.html`** (ADR-049 + ADR-050):
  ```html
  <meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; font-src 'self'; object-src 'none'; frame-src 'none';">
  ```
- **IPC surface mínima** via `electron/preload.ts`:
  ```ts
  import { contextBridge, ipcRenderer } from 'electron';
  contextBridge.exposeInMainWorld('memAPI', {
    loadDataset: () => ipcRenderer.invoke('mem:dataset:load'),
    runCli: (args: string[]) => ipcRenderer.invoke('mem:cli:run', args),
    showInFolder: (path: string) => ipcRenderer.invoke('mem:reveal', path),
    platform: process.platform,
  });
  ```
- **Renderer isolated** (`BrowserWindow` com `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`).
- **Tailwind bundled** via `tailwindcss` + `postcss` + `autoprefixer`; **zero** `@import` de CDN.
- **Cytoscape bundled**: `npm i cytoscape` e usar import direto no renderer; **zero CDN**.
- **shadcn/ui copy-paste** via `npx shadcn@latest add <component>` (no `viewer/src/components/ui/`). Componentes são **owned** pelo repo.

### Estratégia de Phase

| Phase | Escopo | Status |
|---|---|---|
| **Phase 1 (MVP — esta sessão)** | Electron shell + Vite + React + TS + Tailwind + shadcn/ui instalados; 1 tab "Grafo" carregando `data-central.json` estático via fetch local; Cytoscape bundled; build `npm run dev` abre Electron funcionando; `npm run build` gera artifact cross-platform; Playwright valida janela abre. | **in scope deste ADR** |
| **Phase 2 (futura)** | IPC real com `mem.exe` (carrega `mem code-*`, `mem status`, `mem search`); code symbols no Cytoscape; syntax highlight shiki; staleness dashboard; CSP strict enforcement (sem `'unsafe-inline'`); release com `electron-builder`. | **ADR-049/050/051 futuros** |

### Compatibilidade

- **`data-central.json` legado**: viewer Phase 1 lê o mesmo JSON que `graph-v2.html` lia — zero migração de dados.
- **CI Go inalterado**: Electron convive com Go via build matrix (`go test` + `npm run build`); workspace root `package.json` define scripts cross-stack.
- **MCP não muda**: tools MCP continuam em `internal/mcp/`; viewer usa IPC local ao `mem` CLI, **não** MCP.
- **ADRs preservados como background**: ADR-020 (visualização), ADR-024 (triptych inspector), ADR-038 (site único + dataset fixo) **não** são revogados — informam design do Phase 2.

## Consequences

### Positive

- **Bundle zero-CDN** — fecha ADR-050 LLM03 (supply chain).
- **CSP strict pronto** — Phase 2 só endurece.
- **Componentização via shadcn/ui** — adicionar code symbols / staleness / busca é trivial.
- **IPC tipado** — preload script único ponto de superfície exposta.
- **Cross-platform** — Windows/Linux/macOS via `electron-builder`.
- **Delete dos HTMLs legados** — código morto removido, fonte única de verdade.

### Negative

- **Toolchain Node.js** além da stack Go — desenvolvedor precisa de Node 18+ e npm.
- **Bundle size maior** que HTML standalone (~10–30 MB Electron runtime + ~2 MB viewer) — mitigação: install sob demanda via installer, não no repo.
- **Phase 1 não conecta ao `mem` CLI ainda** — `data-central.json` é estático. Phase 2 fecha.
- **shadcn/ui copy-paste** exige `npx shadcn@latest add` em build — divergência entre versões é responsabilidade do dev (não versionado em dep).

### Neutral

- **ADR-037 fica Superseded by this ADR** — não reverter o slot 047; review note em `docs/adr/README.md`.
- **Viewer v1/v2 legados são recuperáveis** via `mavis-trash` durante 30 dias.

## Implementation Plan

1. ✅ Spec `tlc-spec-driven` `.specs/features/feat-viewer-electron/` — Phase 1 MVP (Specify + Tasks).
2. Estrutura monorepo: `viewer/` (Vite + React + shadcn) + `electron/` (main + preload) + `package.json` workspace root.
3. ADR-037 marcado como `Superseded by ADR-048` em `docs/adr/README.md`.
4. shadcn/ui configurado: `viewer/src/components/ui/` com Button, Card, Tabs, ScrollArea, Resizable, Tooltip.
5. Cytoscape bundled; sem CDN; sem CSP exception para `https://unpkg.com`.
6. Build CI matrix mantém Go tests + adiciona `npm ci && npm run build` na matriz linux/windows/macos.
7. Playwright valida Electron abre + tab Graf renderiza dataset.
8. Update `README.md` + `docs/AGENT_INTEGRATION_GUIDE.md` com seção "Viewer Desktop".

## References

- **ADR-020** — Visualizador Interativo de Grafo (background; preservado).
- **ADR-024** — Triptych Node Inspector (background; preservado).
- **ADR-037** — Rewrite Vite + Vanilla TS (Superseded by this ADR).
- **ADR-038** — Viewer Site Único + Dataset Fixo (preservado; bate com Phase 1).
- **ADR-046** — Deferred Until framework (caso Phase 2 vire defer, aplica-se).
- **ADR-049** (planejado) — Viewer CSP strict — Phase 2 ativa.
- **ADR-050** (transversal) — Threat Model OWASP LLM05 Viewer controls (markdown sanitizado, sem imagens remotas/iframes, CSP strict) — Phase 2 ativa.
- **`electron-vite`** — plugin Vite + Electron integrado.
- **`shadcn/ui`** — `https://ui.shadcn.com/` — componentes React copy-paste (Radix + Tailwind).
- **`cytoscape.js`** — graph library bundled (v2 já usava via CDN; v3 bundled).
- **`DeusData/codebase-memory-mcp`** — referência de UX de code-graph (43.9k⭐).
- **`iwe-org/iwe`** — referência de LSP e graph-first UX (1.7k⭐).