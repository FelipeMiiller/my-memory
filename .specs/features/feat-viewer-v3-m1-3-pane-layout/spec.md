---
title: feat-viewer-v3-m1-3-pane-layout — Shell 3-pane + Dock
category: resource
summary: M1 do Roadmap v3. Substitui o layout de top-level tabs (Graf/Code/Staleness/Chat) por um shell 3-pane resizable: File Explorer (left) + Editor (center) + Chat (right), com bottom dock colapsável para terminal (placeholder em M1).
status: draft
tags: [viewer, m1, layout, resizable, 3-pane]
---

# feat-viewer-v3-m1-3-pane-layout — M1 Shell

## Contexto

M1 do Roadmap v3 (`.specs/ROADMAP-v3-workspace-architecture.md`). Migra o viewer Electron do my-memory de layout de **top-level tabs** (Graf / Code / Staleness / Chat) para layout **3-pane resizable** + **bottom dock**.

## Objetivo

Substituir `src/layouts/layout.tsx` (Phase 1 MVP) por `src/layouts/workspace-shell.tsx` que monta:

```
+--------------------------------------------------------------+
| TopBar (logo + view switcher + status)                       |
+--------------------------------------------------------------+
| FileExplorer  |  EditorPane          |  ChatView              |
| (left pane)   |  (center pane)       |  (right pane)          |
| ~20%          |  ~60%                |  ~20%                  |
+--------------------------------------------------------------+
| Bottom Dock (placeholder em M1; Terminal em M4)              |
+--------------------------------------------------------------+
| StatusBar (convenção info, git status, mem version)          |
+--------------------------------------------------------------+
```

Splitters persistem tamanho entre sessões. Atalhos de teclado togglam panes.

## Out of Scope

- File Explorer real (vem em M3 — `mem:fs:list` IPC)
- Editor Markdown funcional (vem em M3 — CodeMirror 6)
- Terminal real (vem em M4 — xterm.js + node-pty)
- Active doc context binding (vem em M5)
- Tool calling / Playwright (vem em M6)
- Command palette (vem em M7)

## Acceptance Criteria (EARS)

- **AC-1 (EARS-U):** Quando o usuário abre o viewer, vê 3 panes (Explorer / Editor / Chat) lado-a-lado com splitters visíveis entre eles.
- **AC-2 (EARS-U):** Quando o usuário arrasta um splitter, o pane adjacente redimensiona em tempo real.
- **AC-3 (EARS-U):** Quando o usuário fecha e reabre o app, os tamanhos dos panes voltam como estavam antes de fechar.
- **AC-4 (EARS-U):** Quando o usuário pressiona `Ctrl+B`, o pane Explorer some/aparece. Idem `Ctrl+J` para Chat.
- **AC-5 (EARS-U):** Quando o usuário pressiona `Ctrl+\``, o dock inferior toggle (vazio em M1).
- **AC-6 (EARS-U):** Quando o usuário redimensiona um pane pra menos de 100px, o splitter trava (não permite pane invisível).
- **AC-7 (EARS-U):** Quando o usuário clica numa tab do Chat existente, o chat continua funcionando exatamente como em v2.0.0 (sem regressão).
- **AC-8 (EARS-U):** Quando o usuário abre a primeira vez, vê um placeholder "Select a file to start editing" no EditorPane.
- **AC-9 (EARS-U):** Quando o usuário abre a primeira vez, vê uma árvore de diretório mock no ExplorerStub.
- **AC-10 (EARS-S):** Estado persistido em `localStorage['mem.ui-layout']` (não cresce além de ~200 bytes).
- **AC-11 (EARS-S):** Atalhos não conflitam com atalhos do OS (Cmd+B no macOS = Bold; OK no Windows/Linux).
- **AC-12 (EARS-S):** Splitters são keyboard-accessible (Tab para focar, setas para mover).
- **AC-13 (EARS-S):** `gofmt -l .` clean, `go build` exit 0, `tsc --noEmit` exit 0, `vitest` PASS, `electron-forge package` exit 0.
- **AC-14 (EARS-S):** Lint não cresce além do baseline (74 em develop antes do PR).

## Estrutura de arquivos

```
src/layouts/
  workspace-shell.tsx     # NEW — main 3-pane layout
  top-bar.tsx             # NEW — extracted from layout.tsx
  status-bar.tsx          # NEW
  layout.tsx              # DEPRECATED — kept for backward compat durante transição

src/components/workspace/
  file-explorer-stub.tsx  # NEW — directory tree placeholder
  editor-pane.tsx         # NEW — placeholder
  dock-placeholder.tsx    # NEW — terminal placeholder
  index.ts

src/lib/ui/
  use-ui-store.ts         # NEW — Zustand + persist

src/hooks/
  use-shortcut.ts         # NEW — Ctrl+B/J/`
```

## Quality Gate

- gofmt clean
- go build exit 0
- tsc --noEmit exit 0
- vitest PASS (1+ novo teste para `useUiStore`)
- electron-forge package exit 0
- biome lint não cresce além de 74 baseline

## Implementation Plan (12 tasks)

T1: ADR-053 merge (este doc)
T2: `useUiStore` (Zustand + persist)
T3: TopBar component (extracted do layout atual)
T4: StatusBar component
T5: `FileExplorerStub` (mock tree)
T6: `EditorPane` (placeholder)
T7: `DockPlaceholder` (terminal placeholder)
T8: `WorkspaceShell` (composição dos 3 panes + dock)
T9: Refactor `app.tsx` para usar `WorkspaceShell`
T10: Atalhos de teclado (useShortcut hook + bindings)
T11: Testes do store + smoke E2E
T12: Docs (VIEWER.md § 3-pane) + PR
