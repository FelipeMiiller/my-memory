# Viewer Desktop (Electron + React + Vite + shadcn/ui)

> Phase 1 MVP — bundled Cytoscape rendering `data-central.json` sem dependências de CDN.
> Decisão arquitetural: [ADR-048](adr/048-viewer-electron-react-vite-shadcn.md).

O Viewer Desktop é a substituição moderna do antigo `graph.html` / `graph-v2.html` (Cytoscape + Tailwind via CDN). Foi **deletado** em 2026-09-21 quando Felipe pediu rebuild com stack moderno (ADR-048).

---

## Setup

### Pré-requisitos

| Ferramenta | Versão | Notas |
| :--- | :--- | :--- |
| **Node.js** | 20 LTS ou superior | Compatível com Electron 30 |
| **npm** | 10+ | Bundled com Node 20+ |
| **Windows / Linux / macOS** | qualquer | Cross-platform via Electron |
| **Plataforma alvo Electron** | Windows 10+/Linux com X11/macOS 11+ | Electron 30+ |

### Comandos

```bash
# 1. Instalar dependências
npm ci

# 2. Modo desenvolvimento (Vite + Electron + HMR)
npm run dev

# 3. Build production (gera viewer/dist + electron/dist)
npm run build

# 4. Type check estrito (TypeScript strict)
npm run typecheck

# 5. Test suite (vitest + jsdom)
npm test

# 6. Smoke tests Electron (Playwright — opcional)
npm run smoke
```

### Estrutura de Diretórios

```text
viewer/
├── index.html               # CSP meta tag (Phase 1 relaxado)
├── public/
│   └── data-central.json    # Dataset estático (gerado pelo mem graph export)
├── src/
│   ├── main.tsx             # React entry (i18n side-effect import)
│   ├── App.tsx              # Mount <Layout />
│   ├── components/
│   │   ├── Layout.tsx       # Topbar + Sidebar + Tabs shell
│   │   ├── LocaleSwitcher.tsx  # pt-BR / en-US dropdown
│   │   └── ui/              # shadcn/ui primitives (button, card, tabs, ...)
│   ├── i18n/
│   │   ├── index.ts         # i18next bootstrap (reads localStorage)
│   │   └── locales/
│   │       ├── pt-BR.json   # default (per VE-16)
│   │       └── en-US.json
│   ├── lib/
│   │   ├── cytoscape-init.ts # Cytoscape core + cose layout
│   │   └── utils.ts         # cn() helper
│   ├── views/
│   │   └── GraphView.tsx    # Fetch + Cytoscape + Selected Node Panel
│   └── styles/
│       └── globals.css      # Tailwind base + shadcn HSL variables
├── tailwind.config.ts
├── postcss.config.mjs
├── tsconfig.json
└── vite.config.ts
electron/
├── main.ts                  # BrowserWindow + CSP + IPC handlers
├── preload.ts                # contextBridge: window.memAPI
├── tsconfig.json
└── tsconfig.node.json
scripts/
├── dev.mjs                  # Vite + Electron orchestrator
└── viewer-smoke.test.mjs    # Playwright smoke
```

---

## Stack

| Camada | Tecnologia | Versão | Por quê |
| :--- | :--- | :--- | :--- |
| **Runtime** | Electron | 30+ | Desktop cross-platform; IPC tipado |
| **Renderer** | React | 18.3 | Componentização sólida + Hooks |
| **Build** | Vite | 5.4 | HMR rápido; ESM nativo |
| **TypeScript** | TypeScript | 5.6 strict | Type safety (ADR-002 philosophy) |
| **Styling** | Tailwind CSS | 3.4 | Utility-first; bundled (zero-CDN) |
| **Components** | shadcn/ui (Radix UI) | latest | Copy-paste; Radix a11y; Tailwind-styled |
| **Graph** | Cytoscape | 3.30 | bundled ESM (zero-CDN) |
| **i18n** | i18next + react-i18next | 23 / 15 | Default `pt-BR`, fallback `en-US` |
| **Test** | Vitest + Testing Library | 2.1 / 16 | jsdom; user-event; @testing-library/jest-dom |
| **Smoke** | Playwright | 1.x | E2E Electron + Vite dev (Phase 1.5) |

---

## Locale Switching

O Viewer suporta **dois locales** carregados como bundles estáticos (sem CDN):

- **`pt-BR`** (default) — usado quando `localStorage['mem.locale']` é null.
- **`en-US`** — fallback quando uma chave falta em `pt-BR`.

### Mecanismo

1. `viewer/src/i18n/index.ts` lê `localStorage['mem.locale']` no boot. Se null, usa `pt-BR`.
2. `viewer/src/components/LocaleSwitcher.tsx` (topbar, botão com ícone `Globe`) abre um DropdownMenu shadcn com as opções:
   - `Português (BR)` (`pt-BR`)
   - `English (US)` (`en-US`)
3. Ao selecionar, persiste a escolha em `localStorage['mem.locale']` e chama `i18n.changeLanguage(code)`. Reload do app preserva a escolha.
4. Todas as strings de UI são traduzidas via `useTranslation()`:
   - `t('app.title')`, `t('app.subtitle')`
   - `t('tab.graph')`, `t('tab.code')`, `t('tab.staleness')`
   - `t('badge.dark')`, `t('badge.hub')`
   - `t('phase2.coming_soon')`, `t('phase2.codeTooltip')`
   - `t('graph.loading')`, `t('graph.error', { msg })`
   - `t('inspector.id')`, `t('inspector.color')`, etc.

### Adicionar nova string

1. Adicione a chave em **ambos** os arquivos `viewer/src/i18n/locales/{pt-BR,en-US}.json`.
2. Use `t('minha.chave')` no componente.
3. Rode `npm test` — falha se a chave existe em um locale mas não no outro.

---

## Phase 2 Roadmap

| Feature | ADR | Status |
| :--- | :--- | :--- |
| IPC real com `mem.exe` (`window.memAPI.loadDataset()`, `runCli()`) | ADR-049 | pending |
| Code symbols no grafo (`feat-code-ast` integration) | ADR-047 + ADR-051 | pending |
| Staleness dashboard tab | ADR-031 + ADR-051 | pending |
| Syntax highlight com **shiki** nos snippets | TBD | pending |
| CSP strict enforcement (sem `'unsafe-inline'`, com nonce Vite) | ADR-049 | pending |
| Dark/light theme toggle (`next-themes`) | Phase 2 UX | pending |
| Triptych Node Inspector (3-coluna) | ADR-024 | pending |
| Live search bar | Phase 2 UX | pending |
| `electron-builder` release cross-platform (.exe/.dmg/.AppImage) | TBD | pending |
| Auto-update via `electron-updater` | TBD | pending |
| Code-signing Windows / notarização macOS | TBD | pending |

---

## Restrições de Phase 1

- **`data-central.json` é estático**: copiado para `viewer/public/data-central.json` durante o build. Phase 2 substitui por IPC.
- **CSP relaxado**: `script-src 'self' 'unsafe-inline' 'unsafe-eval'` em dev (HMR Vite). Phase 2 endurece com nonce.
- **Sem auto-update**: installer cross-platform + signing são Phase 2 (electron-builder).
- **Theme hardcoded dark**: toggle via `next-themes` virá em Phase 2.

---

## Comandos úteis

```bash
# Rodar o viewer Vite standalone (sem Electron)
npm run viewer:dev

# Build apenas do viewer (gera viewer/dist/index.html)
npm run viewer:build

# Build apenas do Electron main + preload (gera electron/dist/)
npm run electron:build

# Rodar Electron standalone (assume viewer dev server na porta 5173)
npm run electron:dev
```

---

## Referências

- **[ADR-048](adr/048-viewer-electron-react-vite-shadcn.md)** — Decisão arquitetural (Accepted 2026-09-21).
- **[ADR-049](README.md#adr-049-planejado--viewer-csp-strict)** — CSP strict (Phase 2).
- **[ADR-050](README.md#adr-050)** — Threat Model OWASP LLM05 (markdown sanitizado, sem imagens remotas).
- **[ADR-051](README.md#adr-051-planejado)** — Code symbols integration.
- **[ADR-037](adr/037-rewrite-viewer-com-vite-vanilla-ts.md)** — Superseded por ADR-048.
- **[`LuanRoger/electron-shadcn`](https://github.com/LuanRoger/electron-shadcn)** — referência canônica (1.4.1, MIT) para Electron + shadcn/ui.

> Veja também a spec ativa em `.specs/features/feat-viewer-electron/spec.md` (16 ACs em EARS).