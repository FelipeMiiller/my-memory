---
title: Roadmap v3 — Modular Workspace Architecture (Electron viewer evolution)
category: resource
summary: Roadmap multi-milestone evoluindo o viewer Electron do my-memory do MVP atual (Phase 1 MVP, ADR-048) para um app workspace AI-modular completo (3-pane layout, editor, terminal, Playwright, prompt hierarchy). Postgres fica intacto no backend my-memory (ADR-040). Viewer consome `mem` via MCP sem dependência direta.
status: draft
tags: [viewer, electron, mcp, roadmap, workspace]
---

# Roadmap v3 — Modular Workspace Architecture

> Plano multi-milestone para evoluir o viewer Electron do my-memory (atualmente Phase 1 MVP, ADR-048 + ADR-051 + ADR-052) para a arquitetura-alvo definida em [System Architecture Specification](#) (anexo do user 2026-09-22).

## Princípios

1. **Backend my-memory intacto.** O CLI Go + servidor MCP continuam como estão. Postgres fica como opt-in (ADR-040) para Central Vault multi-repo. O viewer evolui **em volta** do MCP.
2. **Sem Postgres no viewer.** O viewer consome `mem` via MCP. Zero dependência direta de DB no Electron app.
3. **Milestones incrementais.** Cada M shippable, testável, com quality gate verde. Sem "big bang" release.
4. **Compatibilidade retroativa.** Cada M não quebra os Ms anteriores. v2.0.0 (Chat tab) continua funcionando até M2+.
5. **Decisões arquiteturais viram ADR.** Mudanças estruturais (state split, layout shell, IPC contracts) ganham ADR novo. Decisões pequenas (qual lib, qual estilo) ficam no PR.

## Roadmap

### M1 — 3-pane layout foundation
**Estimativa:** 2-3 semanas · **Bloqueia:** M2, M3, M4, M5

- Layout shell resizable: `[Explorer | Editor | Chat]` + dock inferior colapsável
- Splitter component (shadcn `resizable`) — já temos `resizable.tsx` instalado
- Persistência do layout (Zustand `useUiStore`)
- File explorer **stub** (esqueleto, sem carga real)
- Chat pane mantém tab atual (ADR-051 + ADR-052)
- Editor pane vazio com placeholder "Select a file"

**Entrega:**
- Shell 3-pane funcional com splitters persistidos
- Atalhos: toggle Explorer, toggle Chat, toggle Terminal dock
- ADR-053: layout shell architecture

### M2 — React Query split + service layer
**Estimativa:** 3-4 dias · **Bloqueia:** M4 (Playwright cache), M6 (MCP multi-server)

- Adicionar `@tanstack/react-query` (provider no root)
- Zustand fica para **client state only** (UI, layout, chat local state)
- React Query assume **server state** (IPC calls, fetch MCP, Playwright cache)
- Migrar gradualmente chamadas existentes em `chatApi.getProviders`/`discoverModels` para `useQuery`
- Persistência local (IndexedDB) continua Zustand (Zustand já tem subscribe pattern; React Query não substitui isso)

**Entrega:**
- `QueryClientProvider` no root
- 3+ queries migradas (providers, models, mem-search)
- ADR-054: state split rationale (Zustand vs React Query)

### M3 — File explorer + Markdown editor
**Estimativa:** 2-3 semanas · **Bloqueia:** M5 (active doc context binding)

- File explorer real: lê árvore do diretório via IPC novo `mem:fs:list`
- Tab system no editor (multi-file open)
- Markdown editor: CodeMirror 6 (não Monaco — mais leve, MIT-friendly)
- Live preview side-by-side ou toggle (VS Code style)
- Wikilinks `[[Nota]]` clickáveis → abrem a nota
- `#tag` autocomplete
- Save on Cmd+S / autosave debounced
- IPC: `mem:fs:read`, `mem:fs:write` (atomic, ADR-044)

**Entrega:**
- Explorer + editor funcionais end-to-end
- Wikilinks resolvem para notas reais (via `mem doctor --fix` ou lookup direto)
- ADR-055: editor choice (CodeMirror 6 vs alternatives)

### M4 — Integrated terminal (xterm.js + node-pty)
**Estimativa:** 1 semana · **Bloqueia:** (nada direto, mas desbloqueia M6)

- xterm.js no renderer (WebGL addon)
- node-pty no main process spawna shell (PowerShell no win32, bash no unix)
- IPC: `pty:spawn`, `pty:write`, `pty:resize` + event `pty:data`
- Dock inferior colapsável
- Múltiplas tabs de terminal
- Killswitch (Ctrl+Alt+T toggle dock)

**Entrega:**
- Terminal interativo funcionando dentro do viewer
- ADR-056: PTY security (sandbox, env sanitization)

### M5 — Active document context binding
**Estimativa:** 3-4 dias · **Bloqueia:** (nada direto, mas desbloqueia tool calling)

- Editor event bus publica `{filePath, fileName, content, selection, isDirty}` em `useEditorStore`
- Chat lê active doc + injeta no system prompt automaticamente
- Tag chip no header do chat mostrando doc ativo
- Active doc detection: mudanças debounced (300ms) pra não re-injectar a cada keystroke

**Entrega:**
- Chat usa contexto do arquivo aberto sem copy-paste
- ADR-057: context propagation semantics

### M6 — Playwright web automation + tool calling
**Estimativa:** 2 semanas · **Bloqueia:** (nada, é capability add-on)

- Playwright rodando no main process (Node.js runtime)
- MCP server Playwright (já existe `microsoft/playwright-mcp` — integrar)
- Tool calling no chat: AI pode chamar `web.search`, `web.fetch`, `web.screenshot`
- React Query cache pra resultados de fetch
- Streaming output: tool result chega em chunks (markdown parcial)

**Entrega:**
- Chat com tools: "Busca o preço do iPhone 16 na Apple" → AI usa Playwright
- ADR-058: tool calling security (read-only, scope, audit log)

### M7 — Prompts hierarchy + workspace config
**Estimativa:** 1 semana · **Bloqueia:** (nada, polimento final)

- `models.json` (extensão do `chatLanguageModels.json` com system prompts)
- Workspace config: `.memory/workspace.yaml` com system prompt de projeto
- Per-chat override (UI: prompt textarea no header do chat)
- Precedence: chat-level > workspace > default (determinístico)

**Entrega:**
- System prompts em 3 níveis, hierárquicos
- ADR-059: prompts precedence formalization

## Reordenação estratégica

A ordem proposta (M1 → M7) prioriza fundação visual + state, depois features. Alternativas:

- **M1 → M2 → M3 → M4 → M5 → M6 → M7** (proposta — UI-first)
- **M1 → M2 → M5 → M3 → M4 → M7 → M6** (context-first — M5 antes do editor completo)
- **M1 → M2 → M4 → M6 → M5 → M3 → M7** (terminal+web-first — power user)

## ADRs a criar (resumo)

- ADR-053: 3-pane layout shell architecture
- ADR-054: Zustand vs React Query state split
- ADR-055: Markdown editor choice (CodeMirror 6 vs alternatives)
- ADR-056: PTY security (sandbox, env)
- ADR-057: Active document context propagation
- ADR-058: Tool calling security (Playwright MCP)
- ADR-059: Prompts precedence hierarchy

## Estimativa total

~10-12 semanas de trabalho focado (2-3 meses). Com parallelização em batches de worker, dá pra comprimir pra 6-8 semanas.

## Postgres (esclarecimento)

O spec anexo diz "No Dedicated Relational Database / PostgreSQL" — isso se aplica ao **viewer** (M1-M7). O **backend my-memory** mantém Postgres como opt-in (ADR-040). Memória semântica do chat usa my-memory via MCP server (sem Postgres direto no viewer).

## Migração da v2.0.0

A v2.0.0 (Chat tab) está shipped. M1 adiciona 3-pane **em volta** do chat atual. O chat continua funcionando na aba Chat existente até M3 trazer o novo layout.
