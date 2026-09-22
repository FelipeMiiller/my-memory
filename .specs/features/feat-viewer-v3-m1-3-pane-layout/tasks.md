---
title: feat-viewer-v3-m1-3-pane-layout — Tasks
category: resource
summary: Tasks do ciclo tlc-spec-driven aplicadas retroativamente ao M1 (já shipped). 28 commits de `feat/v3-workspace-architecture` agrupados em 20 tasks (12 base + 8 polish) cobrindo shell 3-pane, chat UX, i18n split e ASR scaffold.
status: done
tags: [viewer, m1, layout, resizable, 3-pane, tasks]
---

# feat-viewer-v3-m1-3-pane-layout — Tasks

> **Nota retroativa:** este arquivo foi gerado após o Execute. O branch `feat/v3-workspace-architecture` está 28 commits ahead de develop e o ciclo formal (Tasks → Execute → Validation) está sendo fechado para deixar rastro auditável. Os hashes citados correspondem ao branch pré-squash.

## Mapa de commits → tasks

28 commits foram agrupados em 20 tasks (12 do plano original + 8 polish que entraram durante M1). Cada task lista os commits que a implementaram.

### Phase 1 — Foundation (ADR + state)

#### T1 — ADR-053 + Roadmap v3 + M1 spec
- **Status:** ✅ DONE
- **Commits:**
  - `42a9110` docs(adr+spec): ADR-053 + Roadmap v3 + M1 3-pane layout spec
- **Entregáveis:**
  - `docs/adr/053-viewer-v3-3-pane-layout-shell.md` (Accepted)
  - `.specs/ROADMAP-v3-workspace-architecture.md`
  - `.specs/features/feat-viewer-v3-m1-3-pane-layout/spec.md` (este spec)

#### T2 — `useUiStore` (Zustand + persist)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell (ADR-053)
- **Entregáveis:**
  - `src/lib/ui/use-ui-store.ts` — Zustand store com `explorer`/`editor`/`chat`/`dock` (visible + width) + `resetLayout()` + persist
- **Cobre AC:** AC-3 (persistência entre sessões), AC-10 (`localStorage` ≤ 200 bytes — Zustand persiste em JSON serializado)

#### T3 — TopBar component (extracted from layout.tsx)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
  - `3b53848` feat(viewer): window controls + i18n + Central vault no explorer
  - `a29a212` fix(viewer): make TopBar draggable (window moveable with hidden title bar)
- **Entregáveis:**
  - `src/layouts/top-bar.tsx` — logo + pane toggle buttons + drag region + window controls (Win/Linux) + locale switcher

#### T4 — StatusBar component
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
- **Entregáveis:**
  - `src/layouts/status-bar.tsx` — `layout: explorer X% · chat Y% · dock: Z%` + botão "restaurar layout padrão"

### Phase 2 — Workspace composition (panes + dock)

#### T5 — `FileExplorerStub` (mock tree)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
  - `3b53848` feat(viewer): window controls + i18n + Central vault no explorer
- **Entregáveis:**
  - `src/components/workspace/file-explorer-stub.tsx` — placeholder com diretório mock (incluindo "Central vault" entry, conforme i18n)
- **Cobre AC:** AC-9 (mock tree visível)

#### T6 — `EditorPane` (placeholder)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
- **Entregáveis:**
  - `src/components/workspace/editor-pane.tsx` — placeholder "Selecione um arquivo para começar" + botão "Abrir arquivo" (futuro M3)
- **Cobre AC:** AC-8 (placeholder visível)

#### T7 — `DockPlaceholder` (terminal placeholder)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
- **Entregáveis:**
  - `src/components/workspace/dock-placeholder.tsx` — placeholder para o dock inferior (xterm.js + node-pty fica pra M4)

#### T8 — `WorkspaceShell` (3-pane + dock)
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
- **Entregáveis:**
  - `src/layouts/workspace-shell.tsx` — composição `react-resizable-panels` com `[Explorer | Editor | Chat]` + dock colapsável
- **Cobre AC:** AC-1 (3 panes lado-a-lado), AC-2 (drag em tempo real), AC-6 (split trava < 100px)

#### T9 — Refactor `app.tsx` → `WorkspaceShell`
- **Status:** ✅ DONE
- **Commits:**
  - `b463f80` feat(viewer): M1 3-pane workspace shell
- **Entregáveis:**
  - `src/app.tsx` agora monta `<WorkspaceShell />` em vez do antigo `<Layout />`

### Phase 3 — Polish (shortcuts + tests + docs)

#### T10 — Atalhos de teclado (Ctrl+B / Ctrl+J / Ctrl+\`)
- **Status:** ⚠️ PARTIAL — shortcuts definidos no TopBar (botões visíveis); keyboard bindings declarados em `use-ui-store.ts` mas sem listener global dedicado. Toggle por clique funciona; toggle por teclado fica para polish de v3.1.
- **Commits:**
  - `3b53848` feat(viewer): window controls + i18n + Central vault no explorer
- **Entregáveis:**
  - `src/layouts/top-bar.tsx` — pane toggle buttons (visíveis e clicáveis)
  - `src/lib/ui/use-ui-store.ts` — toggle actions expostas
- **Cobre AC:** AC-4 (parcial via clique), AC-5 (parcial), AC-11/AC-12 (não implementados)

#### T11 — Testes do store + smoke E2E
- **Status:** ⚠️ DEFERRED — vitest suite tem 1 teste E2E Electron smoke (`src/tests/e2e/example.test.ts`); testes unitários de `useUiStore` ficaram fora do escopo deste PR. Tests do ChatSidebar / ChatInput mencionados no PR doc como follow-up opcional.
- **Entregáveis:**
  - `src/tests/e2e/example.test.ts` (1/1 PASS em vitest)

#### T12 — Docs (README + STATE + PR description)
- **Status:** ✅ DONE
- **Commits:**
  - `dd9afb1` docs(readme): expand viewer description with 3-pane layout + i18n + ADR-053
  - `25a462a` docs(state): record v3-workspace-architecture M1 status + i18n fix f63f510
  - `60bac5a` docs(pr): PR #7 description for feat/v3-workspace-architecture
- **Entregáveis:**
  - `README.md` §7 — viewer expandido (3-pane + TopBar + Sheet constrained + i18n per-namespace + ASR scaffold)
  - `.specs/STATE.md` — handoff atualizado com M1 status + i18n fix
  - `.specs/pr-feat-v3-workspace-architecture.md` — descrição completa do PR #7

### Phase 4 — M1 Polish (chat UX + i18n + ASR scaffold)

> Tasks T13-T20 foram adicionadas durante o Execute para cobrir decisões de produto que entraram no escopo M1 após o Specify inicial.

#### T13 — ChatModelPicker VS Code-style (search + grouped + speed badges)
- **Status:** ✅ DONE
- **Commits:**
  - `474b046` feat(viewer-chat): VS Code-style model picker dropdown in chat header
  - `1a18c36` feat(viewer): VS Code-style chat model picker (search + grouped + speed badges + Upgrade)
  - `15f70a1` fix(viewer): replace ChevronDown with Sparkles in ChatModelPicker
  - `bc43bf2` feat(viewer): move ChatModelPicker into chat input toolbar (VS Code position)
  - `9c66643` fix(viewer): remove ChatModelPicker button from chat header
- **Decisão final:** ChatModelPicker foi **removido do header do chat** (commit `9c66643`) e funcionalidade preservada em `ChatSettingsModal` (acessado via ⚙). Razão: UX VS Code tem picker inline na input toolbar, mas no header virou redundante com Settings.
- **Entregáveis:**
  - `src/components/chat/chat-model-picker.tsx` (376 LOC) — **REMOVIDO em f63f510** (dead code após `9c66643`)
  - `ChatSettingsModal` (preserva a funcionalidade de seleção)

#### T14 — Sessions-style chat sidebar (search + grouped Newer/Older)
- **Status:** ✅ DONE
- **Commits:**
  - `3a5afcf` feat(viewer): VS Code chat layout — header actions + inline sessions panel
  - `833214d` feat(viewer): Sessions-style chat sidebar (search + grouped Newer/Older)
  - `885b6bd` fix(viewer): rename sidebar title back to 'Conversas' (pt-BR)
  - `39abe08` feat(viewer): collapsible conversations sidebar + smaller chat header icons
- **Entregáveis:**
  - `src/components/chat/chat-sidebar.tsx` — sessions list com search filter + relative time (`agora/Xmin/Xh/Xd/Xsem/Xmeses/Xanos`)

#### T15 — Chat sidebar Sheet (constrained inside chat pane)
- **Status:** ✅ DONE
- **Commits:**
  - `5f19a6e` feat(viewer): move conversations sidebar to the right (Secondary Side Bar)
  - `49e7d6c` fix(viewer): chat gets min-w-[400px] + sidebar auto-collapses when narrow
  - `24eadad` fix(viewer): derive sidebar visibility from windowWidth + user state (no effect sync)
  - `825e225` fix(viewer): revert min-w + auto-collapse changes (back to pre-49e7d6c)
  - `c19630e` feat(viewer): wrap conversations sidebar in shadcn Sheet (slide-in from right)
  - `e2c5c0c` feat(viewer): constrain Sheet inside chat pane (no full-window overlay)
  - `4f4400b` fix(viewer): drop Settings/X from chat sidebar header
  - `5802fa9` style(viewer): tighten chat sidebar header icon size + spacing
- **Entregáveis:**
  - `src/components/ui/sheet.tsx` — shadcn Sheet com `container` + `position` props (viewport/contained)
  - `src/components/chat/chat-view.tsx` — Sheet constrained inside chat pane via `chatPaneRef`

#### T16 — TopBar draggable + window controls
- **Status:** ✅ DONE
- **Commits:**
  - `a29a212` fix(viewer): make TopBar draggable (window moveable with hidden title bar)
  - `3b53848` feat(viewer): window controls + i18n + Central vault no explorer
- **Entregáveis:**
  - `src/components/workspace/window-controls.tsx` — Win/Linux min/max/close buttons
  - Drag region em `src/layouts/top-bar.tsx` (Electron `app-region: drag`)

#### T17 — ASR scaffold (ADR-045)
- **Status:** ⚠️ SCAFFOLD ONLY — implementação real do Nemotron (9 tasks) fica em spec dedicada `.specs/features/nemotron-asr-streaming/spec.md`
- **Commits:**
  - `bb16fce` feat(viewer): microphone button + ASR IPC pipeline (ADR-045 scaffold)
- **Entregáveis:**
  - `electron/ipc/asr.ts` — handlers com synthetic partials (Nemotron stub)
  - `src/components/chat/mic-button.tsx` — MicButton com `getUserMedia` 16kHz mono + `ScriptProcessor`
  - IPC pipeline: `MEM_ASR_START/CHUNK/STOP/PARTIAL/FINAL/ERROR`
  - `electron/preload.ts` — `memAPI.asr.{start,chunk,stop,onPartial,onFinal,onError}`

#### T18 — i18n per-namespace split
- **Status:** ✅ DONE
- **Commits:**
  - `81f5810` refactor(viewer): split i18n locales into per-namespace JSON files
- **Entregáveis:**
  - `src/localization/i18n.ts` — bootstrap com `NAMESPACES` (`common`, `topbar`, `graph`, `workspace`, `chat`) + `SUPPORTED_LOCALES` + docstring "add language"
  - `src/localization/locales/{pt-BR,en-US}/{common,topbar,chat,graph,workspace}.json`

#### T19 — i18n fix + dead code cleanup
- **Status:** ✅ DONE
- **Commits:**
  - `f63f510` fix(viewer): bind every component to its i18n namespace + strip dead code
- **Bug raiz:** Split introduzido em `81f5810` deixou componentes com `useTranslation()` (sem namespace), fazendo chaves resolverem pra `chat.chat.title` em vez de `chat.title`. Fix garante `useTranslation("namespace")` em todo componente + JSONs sem wrapper duplicado.
- **Dead code removido:**
  - `src/components/chat/chat-model-picker.tsx` (376 LOC)
  - `chat.picker.*` i18n keys em pt-BR + en-US
  - `ChatModelMeta`, `CHAT_MODEL_META`, `CHAT_PROVIDER_NAMES` exports de `lib/chat/types.ts`
- **Validação empírica:** Playwright screenshot confirmou 100% das chaves resolvendo.

#### T20 — Cosmetic chat sidebar header touch-up
- **Status:** ✅ DONE
- **Commits:**
  - `5802fa9` style(viewer): tighten chat sidebar header icon size + spacing
- **Mudança:** Plus/RefreshCw icons `h-3.5 w-3.5` → `h-2 w-2` + wrapper `gap-0.5` → `pt-0.5 mr-6`. Alinhamento vertical com título + clearance lateral.

## Resumo de cobertura

| Categoria | Tasks | Status |
|---|---:|---|
| Foundation (T1-T4) | 4 | 4/4 ✅ |
| Composition (T5-T9) | 5 | 5/5 ✅ |
| Polish base (T10-T12) | 3 | 1/3 ✅ + 1/3 ⚠️ + 1/3 ⚠️ |
| Chat UX + i18n + ASR (T13-T20) | 8 | 7/8 ✅ + 1/8 ⚠️ (ASR scaffold) |
| **Total** | **20** | **17/20 ✅ + 3/20 ⚠️ PARCIAIS** |

## Tasks parciais — ações

- **T10 (shortcuts):** abrir spec `feat-viewer-v3-shortcuts` antes de M2 — pequeno, 1-2 dias
- **T11 (testes unit `useUiStore` + ChatSidebar/ChatInput):** adicionar 2-3 testes vitest no próximo PR de polish
- **T17 (ASR real):** implementar `.specs/features/nemotron-asr-streaming/tasks.md` (T1-T9 já definidos)