# Feature: feat-viewer-electron (Phase 1 — MVP)

## Problem Statement

O viewer do my-memory (`graph-v2.html`, 49 KB, Cytoscape + Tailwind CDN) virou legado em 2026-09-21 quando Felipe pediu delete + rebuild com stack moderno. Os HTMLs viewers (incluindo screenshots e scripts Playwright de validação) foram movidos para `mavis-trash` (recuperáveis por 30 dias).

**ADR-048** estabelece o caminho: **Electron + React 18 + Vite 5 + TypeScript + Tailwind CSS + shadcn/ui**, monorepo com `electron/` (main + preload) + `viewer/` (Vite + React renderer). Phase 1 = MVP funcional (`npm run dev` abre Electron + 1 tab Graf carregando `data-central.json` estático). Phase 2 (IPC real + code symbols + syntax highlight + CSP strict completo + release) fica para ADR futuro.

Esta spec cobre **Phase 1 apenas** — entregar um Electron app rodável com tab Graf + Cytoscape bundled + zero-CDN, validado por Playwright (skill do Felipe).

## Out of Scope

(Phase 2 — deferred)

- IPC real com `mem` CLI (Phase 1 usa fetch local a `data-central.json` estático)
- Code symbols no grafo (feat-code-ast integration)
- Staleness dashboard tab
- Syntax highlight shiki nos snippets
- CSP strict enforcement (Phase 1 inclui meta tag mas permite `'unsafe-inline'` em script/style por causa do HMR do Vite em dev)
- `electron-builder` release cross-platform assinado
- Dark/light theme toggle (shadcn/ui supports via `next-themes`; Phase 1 hardcode dark)
- Triptych Node Inspector (3-coluna quando selecionado)
- Search bar live

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Node version | Node 20 LTS | Compatível com Electron 30 |
| Package manager | `npm` (sem yarn/pnpm no projeto) | Compat com CI Go inalterado + simplicidade |
| Electron version | Latest stable (30+) ao instalar | `npm i -D electron` |
| Vite version | 5.x | Padrão atual; `electron-vite` plugin supports |
| TypeScript version | 5.x | Padrão; strict mode |
| React version | 18.x | Compat com shadcn/ui |
| shadcn/ui init | `npx shadcn@latest init` (CLI-driven) | Padrão shadcn; components ficam em `viewer/src/components/ui/` |
| Tema inicial | Hardcoded **dark** (matches v2 final) | Phase 1 só valida stack; toggle em Phase 2 |
| Strict mode TS | yes | ADR-002 Go philosophy applied ao TS |
| Tailwind config | `viewer/tailwind.config.ts` com `darkMode: 'class'` | shadcn/ui standard |
| Cytoscape source | bundled ESM via `npm i cytoscape` | Zero-CDN per ADR-050 |
| `data-central.json` carregamento | `fetch('./data-central.json')` direto no renderer | Phase 2 substitui por IPC com `mem` |
| Convention | Conventional Commits + commit atômico por task | Mesmo padrão feat-code-ast |
| Open questions: none | Phase 1 escopo definido em ADR-048 §Decision Outcome; nenhuma decisão pendente para o Execute. |

## User Stories

- **US1**: Como usuário, rodo `npm run dev` no repo root e Electron abre janela dark com 1 tab "Grafo" mostrando o dataset `data-central.json` via Cytoscape.
- **US2**: Como usuário, dou zoom/pan e clico em nodes — toltip shadcn aparece com label + degree + community.
- **US3**: Como dev, rodo `npm run build` e gera artifact production em `dist-electron/` (sem release cross-platform ainda).
- **US4**: Como dev, rodo `npm run typecheck` e TypeScript strict-mode passa sem erros em `viewer/`, `electron/`, `shared/`.

## Requirements (EARS Notation)

### VE-01: Estrutura monorepo Electron + Vite + React
- **UBIQUITOUS**: The system SHALL live in monorepo with `electron/` (main + preload + tsconfig.node.json), `viewer/` (Vite + React + Tailwind + shadcn/ui), and workspace-root `package.json` (private, contains scripts `dev`, `build`, `typecheck`).
- **WHERE**: When `npm run dev` is invoked, the system SHALL start `vite` dev server + `electron .` in parallel via `electron-vite` plugin (or `concurrently`).

### VE-02: Electron main + preload
- **UBIQUITOUS**: The system SHALL have `electron/main.ts` creating `BrowserWindow` with `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`, loading `viewer/dist/index.html` in production / `http://localhost:5173` in dev.
- **UBIQUITOUS**: The system SHALL have `electron/preload.ts` exposing safe IPC surface via `contextBridge.exposeInMainWorld('memAPI', { ... })` (Phase 1: stub methods returning promises; Phase 2: real `mem:dataset:load`, `mem:cli:run`).
- **WHERE**: When the renderer requests `window.memAPI.platform`, the system SHALL return `process.platform` (sanitized).

### VE-03: shadcn/ui configurado
- **UBIQUITOUS**: The system SHALL have `viewer/src/components/ui/` populated with `Button`, `Card`, `Tabs`, `ScrollArea`, `Tooltip`, `Resizable`, `Input`, `Badge` (or subset Phase 1 needs) via `npx shadcn@latest add`.
- **UBIQUITOUS**: The system SHALL have `viewer/src/lib/utils.ts` with `cn()` helper (clsx + tailwind-merge) — shadcn/ui standard.

### VE-04: Tailwind CSS bundled (zero-CDN)
- **UBIQUITOUS**: The system SHALL have `viewer/tailwind.config.ts` + `viewer/postcss.config.js` with `darkMode: 'class'` and content path `./src/**/*.{ts,tsx}`.
- **UBIQUITOUS**: The system SHALL have `viewer/src/styles/globals.css` with `@tailwind base/components/utilities` + shadcn CSS variables (HSL `--background`, `--foreground`, etc).
- **WHERE**: When `index.html` is loaded, the system SHALL NOT load any `<link rel="stylesheet" href="http...">` from CDN — strict CSP meta tag verified.

### VE-05: React App layout com shadcn Tabs
- **UBIQUITOUS**: The system SHALL have `viewer/src/App.tsx` rendering `Layout` (sidebar + topbar) with `Tabs` root containing `<TabsList>` and `<TabsContent value="graph">` for "Grafo" tab.
- **UBIQUITOUS**: The system SHALL have placeholder `<TabsContent value="code">Code (Phase 2)</TabsContent>` and `<TabsContent value="staleness">Staleness (Phase 2)</TabsContent>` rendered but disabled (with "Coming in Phase 2" message).
- **WHERE**: When dark mode is hardcoded (no toggle yet), the system SHALL add `class="dark"` on `<html>` root + tailwind base styles.

### VE-06: GraphView com Cytoscape bundled
- **UBIQUITOUS**: The system SHALL have `viewer/src/views/GraphView.tsx` that fetches `./data-central.json` on mount (via `useEffect`), parses, and renders via Cytoscape using `cytoscape` import (no `<script src="https://unpkg...">`).
- **UBIQUITOUS**: The system SHALL style nodes by `node.community_color` (from dataset) or fallback palette; edges by `edge.color`.
- **WHERE**: When user clicks a node, the system SHALL open `Tooltip` shadcn with `title: node.label`, `description: node.community_label`, `path: tt-path style`, `degree: tt-row`.

### VE-07: CSP strict meta tag (Phase 1 relaxado)
- **UBIQUITOUS**: The system SHALL include `<meta http-equiv="Content-Security-Policy" content="...">` in `viewer/index.html` with `default-src 'self'`, `script-src 'self' 'unsafe-inline'`, `style-src 'self' 'unsafe-inline'`, `img-src 'self' data:`, `connect-src 'self'`, `font-src 'self'`, `object-src 'none'`, `frame-src 'none'`.
- **WHERE**: When `'unsafe-inline'` is present in `script-src` and `style-src`, the system SHALL include a TODO comment `Phase 2: remove unsafe-inline; enable HMR via Vite + nonce`.

### VE-08: TypeScript strict
- **UBIQUITOUS**: The system SHALL have `viewer/tsconfig.json` with `"strict": true`, `"target": "ES2022"`, `"module": "ESNext"`, `"moduleResolution": "bundler"`, `"jsx": "react-jsx"`.
- **UBIQUITOUS**: The system SHALL have `electron/tsconfig.json` with same strict settings + `"types": ["node"]`.

### VE-09: npm scripts
- **UBIQUITOUS**: The workspace `package.json` SHALL define scripts: `dev` (start Vite + Electron), `build` (vite build + tsc -p electron/), `typecheck` (tsc --noEmit in viewer + electron), `lint` (eslint + prettier check).
- **UBIQUITOUS**: The system SHALL have `npm run typecheck` exit 0 before Phase 1 ships.

### VE-10: Playwright validation
- **UBIQUITOUS**: The system SHALL have `scripts/viewer-smoke.test.mjs` (Playwright Test runner) launching Electron + asserting window appears + Tab "Grafo" renders + Cytoscape canvas visible + `data-central.json` fetch succeeds.
- **WHERE**: When `npm run smoke` is invoked, the system SHALL run Playwright test headless (or with screenshot).

### VE-11: Build artifacts
- **UBIQUITOUS**: The system SHALL produce `viewer/dist/` (Vite output) and `electron/dist/main.js` (TS compile) when `npm run build` runs.
- **UBIQUITOUS**: The system SHALL NOT yet produce cross-platform installer (Phase 2 + `electron-builder`).

### VE-12: README + docs sync
- **UBIQUITOUS**: The system SHALL update `README.md` capability table with "Viewer Desktop (Electron + React + Vite + shadcn/ui)".
- **UBIQUITOUS**: The system SHALL create `docs/VIEWER.md` with setup steps (Node 20+, npm ci, npm run dev, npm run build).
- **UBIQUITOUS**: The system SHALL update `docs/adr/README.md` marking ADR-037 as "Superseded by ADR-048".

### VE-13: Migration de data
- **UBIQUITOUS**: The system SHALL NOT modify `data-central.json` (preserved as-is, served as static asset from viewer).
- **UBIQUITOUS**: The system SHALL document in `viewer/README.md` the dataset location + Phase 2 IPC plan.

### VE-14: Conventional Commits + atomic
- **UBIQUITOUS**: The system SHALL emit commits in the form `chore(viewer): ...`, `feat(viewer): ...`, `fix(viewer): ...` based on `git commit -F <file>` validated by `python3 .agents/skills/tlc-spec-driven/scripts/check_commit.py`.
- **UBIQUITOUS**: The system SHALL have ≥4 atomic commits by Phase 1 close (one per task minimum).

### VE-15: Tree-sitter binding deferred
- **WHERE**: When `mem code-*` integration is requested in Phase 2, the system SHALL follow ADR-047 §Deferral Plano A (gcc + CGO + go-tree-sitter).

### VE-16: i18n via i18next (pt-BR default + en-US)
- **UBIQUITOUS**: The system SHALL use `i18next` + `react-i18next` for internationalization with locale files `viewer/src/i18n/locales/pt-BR.json` (default) and `viewer/src/i18n/locales/en-US.json` (secondary).
- **UBIQUITOUS**: The system SHALL expose a locale switcher in the Layout topbar (dropdown or toggle) persisting choice in `localStorage` key `mem.locale`.
- **UBIQUITOUS**: The system SHALL translate all user-facing strings via `t('key')` — Tabs labels (Grafo/Code/Staleness), placeholder messages ("Coming in Phase 2"), topbar badges ("Dark"), and any error messages.
- **WHERE**: When locale is set to `pt-BR`, the system SHALL load only `pt-BR.json` (no dynamic loading of `en-US.json`).
- **WHERE**: When locale is set to `en-US`, the system SHALL load only `en-US.json`.
- **UBIQUITOUS**: The system SHALL NOT depend on any external i18n CDN or service (translation files are static assets bundled in the renderer).

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| VE-01 | Estrutura monorepo Electron + Vite + React | verified |
| VE-02 | Electron main + preload com contextBridge | verified |
| VE-03 | shadcn/ui configurado (Button, Card, Tabs, ScrollArea, Tooltip, Resizable, Input, Badge) | verified |
| VE-04 | Tailwind CSS bundled (zero-CDN) | verified |
| VE-05 | React App layout com shadcn Tabs | verified |
| VE-06 | GraphView com Cytoscape bundled | verified |
| VE-07 | CSP strict meta tag (Phase 1 relaxado) | verified |
| VE-08 | TypeScript strict | verified |
| VE-09 | npm scripts (dev, build, typecheck, lint, smoke) | verified |
| VE-10 | Playwright validation (window + tab + Cytoscape + fetch) | verified |
| VE-11 | Build artifacts (viewer/dist/ + electron/dist/) | verified |
| VE-12 | README + docs/VIEWER.md + ADR-037 superseded note | verified |
| VE-13 | data-central.json preservado | verified |
| VE-14 | Conventional Commits + ≥4 atomic commits | verified |
| VE-15 | Cross-reference ADR-047 §Deferral Plano A | documented |
| VE-16 | i18n via i18next (pt-BR default + en-US) | verified |