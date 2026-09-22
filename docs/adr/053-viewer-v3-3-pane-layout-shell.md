# ADR-053: Layout Shell 3-pane + Dock (M1 do Roadmap v3)

- **Date**: 2026-09-22
- **Status**: Proposed (vai virar Accepted quando M1 fechar)
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: viewer, electron, layout, react, m1

## Context and Problem Statement

O viewer Electron do my-memory (ADR-048 Accepted, Phase 1 MVP shipped em v2.0.0) tem uma arquitetura de **top-level tabs**: Graf / Code / Staleness / Chat. Cada tab é uma página cheia que ocupa toda a janela. Chat foi adicionado como mais uma tab (ADR-051 + ADR-052) em v2.0.0.

O Roadmap v3 (`.specs/ROADMAP-v3-workspace-architecture.md`, baseado no spec anexo de 2026-09-22) descreve uma evolução para um **workspace AI-modular** com:

- **3-pane layout**: `[File Explorer | Editor | Chat]` side-by-side
- **Bottom dock** colapsável (terminal em M4)
- **Active document context binding** (editor → chat)
- **Tool calling** com Playwright (M6)

Esse ADR responde **só o M1**: como construir o shell 3-pane + dock, deixando cada pane como um **slot** vazio que Ms seguintes preenchem (File Explorer real em M3, Editor Markdown em M3, Terminal em M4, etc.).

## Decision Drivers

- **DR-1**: Ms seguintes (M2-M7) plugam no shell. Layout não deve ser refatorado em cada M.
- **DR-2**: Estado persistido entre sessões (tamanho dos panes, qual pane está oculto).
- **DR-3**: Atalhos de teclado para togglar panes (VS Code / Obsidian).
- **DR-4**: Performance: layout shell não pode virar gargalo de render (shadcn resizable já testado).
- **DR-5**: Compatibilidade retroativa: Chat tab atual tem que continuar funcionando durante M1 (não pode quebrar v2.0.0).
- **DR-6**: Sem dependências novas — shadcn `resizable` já está instalado (`src/components/ui/resizable.tsx`).
- **DR-7**: Acessibilidade — splitters devem ser keyboard-accessible.

## Considered Options

### 1. shadcn `resizable` puro
Usa o componente já instalado. Sem nova dep.

- ✅ Zero novas deps
- ✅ Keyboard-accessible out-of-the-box
- ⚠️ Não tem bottom dock pronto — preciso montar manualmente com 2 `ResizablePanelGroup` aninhados (um vertical pros 3 panes, um horizontal pro dock).

### 2. `react-mosaic` ou `flexlayout-react`
Libs prontas de layout management.

- ❌ Adiciona 50-100KB de bundle
- ❌ Curva de aprendizado
- ❌ API complexa pra M1 simples
- ❌ M3+ provavelmente querem layout customizado (CodeMirror + sidebar context)

**Rejeitado** — overkill pra M1.

### 3. CSS Grid puro com `grid-template-columns: var(--explorer-w) 1fr var(--chat-w)`
Mais leve, mas splitters ficam manuais (mouse drag precisa de JS pra resize).

- ❌ Splitters keyboard-accessible precisam de muito JS
- ❌ Acessibilidade ruim (não há equivalente nativo de `role="separator"`)

**Rejeitado** — perde acessibilidade.

## Decision Outcome

**Opção escolhida: shadcn `resizable` (vertical + horizontal aninhados).**

### Estrutura

```
<ResizablePanelGroup direction="horizontal">      ← 3 panes horizontais
  <ResizablePanel defaultSize={20}>                ← File Explorer
    <FileExplorer />
  </ResizablePanel>
  <ResizableHandle />
  <ResizablePanel defaultSize={60}>                ← Editor
    <EditorPane />
  </ResizablePanel>
  <ResizableHandle />
  <ResizablePanel defaultSize={20}>                ← Chat
    <ChatView />
  </ResizablePanel>
</ResizablePanelGroup>

<ResizablePanelGroup direction="vertical">        ← Dock inferior (opcional)
  <ResizablePanel>                                 ← Conteúdo principal (re-render dos 3 panes)
  </ResizablePanel>
  <ResizableHandle />
  <ResizablePanel defaultSize={25}>                ← Terminal dock (placeholder em M1)
  </ResizablePanel>
</ResizablePanelGroup>
```

Wait — na verdade, a estrutura correta é 1 `ResizablePanelGroup` vertical no root, com 2 panels (main content + dock). E dentro do "main content", outro `ResizablePanelGroup` horizontal com 3 panels.

```
<div className="h-screen flex flex-col">
  <TopBar />
  <ResizablePanelGroup direction="vertical" className="flex-1">
    <ResizablePanel>
      {/* main content area */}
      <ResizablePanelGroup direction="horizontal">
        <ResizablePanel defaultSize={20}><Explorer /></ResizablePanel>
        <ResizableHandle />
        <ResizablePanel defaultSize={60}><Editor /></ResizablePanel>
        <ResizableHandle />
        <ResizablePanel defaultSize={20}><Chat /></ResizablePanel>
      </ResizablePanelGroup>
    </ResizablePanel>
    <ResizableHandle />
    <ResizablePanel defaultSize={0} collapsible collapsedSize={0} minSize={0} maxSize={40}>
      <DockPlaceholder />
    </ResizablePanel>
  </ResizablePanelGroup>
  <StatusBar />
</div>
```

### Estado persistido (`useUiStore`)

```typescript
interface UiState {
  layout: {
    explorer: { visible: boolean; widthPct: number };  // 0-100
    chat:     { visible: boolean; widthPct: number };
    dock:     { visible: boolean; heightPct: number };  // terminal
    editor:   { widthPct: number };                       // computed = 100 - explorer - chat
  };
  togglePane: (pane: 'explorer' | 'chat' | 'dock') => void;
  resizePane: (pane: ..., pct: number) => void;
  resetLayout: () => void;
}
```

Persistência via `zustand/middleware/persist` em `localStorage['mem.ui-layout']` (já temos `mem.locale` e `mem.theme` como keys).

### Atalhos (VS Code / Obsidian padrão)

| Atalho | Ação |
|---|---|
| `Ctrl+B` | Toggle File Explorer |
| `Ctrl+J` | Toggle Chat |
| `Ctrl+`` | Toggle Terminal Dock |
| `Ctrl+Shift+P` | Command palette (futuro M7+) |
| `Ctrl+1` / `Ctrl+2` / `Ctrl+3` | Foco em Explorer / Editor / Chat |

Implementado via `useHotkeys` hook do `react-hotkeys-hook` (já temos? vou checar; se não, ~5KB e adiciono).

### Compatibilidade retroativa com v2.0.0

- **Tabs Graf / Code / Staleness** que existiam no Phase 1 MVP **desaparecem**. Graf vai para a tab do Editor (vai ler `.memory/data-central.json` ou `mem graph view --json` e renderizar cytoscape dentro do EditorPane em M3 ou antes).
- **Chat tab** migra para o right pane — mesmo componente, novo wrapper.
- **Top tabs** (que sumiram) ganham **command palette** como substituto (M7).

### Decisões adiadas (não no M1)

- **Splitter snap points** (VS Code tem 50% quando drag termina no centro) — Fase 2
- **Multi-window** (sair do app = janela separada) — fora de roadmap
- **Custom layout profiles** (developer / writer / researcher) — fora de roadmap

## Consequences

### Positivo

- ✅ Ms seguintes plugam direto em `<Explorer />`, `<Editor />`, `<Chat />`
- ✅ Atalhos de teclado familiares (VS Code / Obsidian)
- ✅ Zero deps novas (resizable já instalado)
- ✅ Estado persistido automaticamente
- ✅ Acessibilidade keyboard-first via Radix

### Negativo / Trade-offs

- ⚠️ Migration do layout de **tabs** para **panes** é breaking na UI (mas não na API). Usuário precisa se adaptar.
- ⚠️ Splitter duplo (vertical + horizontal) tem pequena curva de aprendizado — usuário pode demorar pra achar o "sweet spot".
- ⚠️ Dock inferior vazio em M1 — pode parecer "feature faltando". Comunicar que vem em M4.
- ⚠️ 80KB+ de re-arquivamento no renderer (não crítico).

## Implementation Plan

Spec detalhado: `.specs/features/feat-viewer-v3-m1-3-pane-layout/spec.md`
Tasks breakdown: `.specs/features/feat-viewer-v3-m1-3-pane-layout/tasks.md`

Resumido (12 tasks):

1. ADR-053 merge (este doc)
2. `useUiStore` (Zustand + persist)
3. `WorkspaceShell` component
4. Splitter config (default sizes + min/max)
5. `FileExplorerStub` component
6. `EditorPane` placeholder component
7. Refactor `Layout` para usar `WorkspaceShell`
8. Mover `ChatView` da tab para right pane
9. Atalhos de teclado (Ctrl+B/J/`)
10. Testes unitários do store + smoke E2E
11. Docs (VIEWER.md § 3-pane layout)
12. PR target develop

Estimativa: 5-7 dias de trabalho focado.
