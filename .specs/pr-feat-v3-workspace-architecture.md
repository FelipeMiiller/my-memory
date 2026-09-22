# feat(viewer): M1 3-pane workspace shell + chat UX (ADR-053)

Evolução do viewer Electron do MVP v2.0.0 (Chat tab) para um workspace AI-modular de 3 panes (Explorer / Editor / Chat). Cobre o **Milestone 1** do [ROADMAP-v3-workspace-architecture.md](ROADMAP-v3-workspace-architecture.md) — layout foundation + window controls + drag region + i18n per-namespace + chat sidebar polida.

PR #7 contra `develop`. Branch: `feat/v3-workspace-architecture`.

## O que entra

### M1 base (ADR-053)
- **3-pane layout shell**: `Layout` (File Explorer left / Editor center / Chat right) com `react-resizable-panels`
- **Dock inferior colapsável** (placeholder xterm.js + node-pty pra M4)
- **TopBar**: drag region + 4 pane toggle buttons + window controls (Win/Linux)
- **Workspace shell**: `useUiStore` Zustand com layout persistence + `resetLayout()`
- **Status bar**: `layout: explorer X% · chat Y% · dock: Z%` + botão "restaurar layout padrão"

### Chat UX (M1 polish)
- **Sessions-style chat sidebar** (Sheet from right) com header `+ ↻`, search filter, flat list (title + relative time `agora/Xmin/Xh/Xd/Xsem/Xmeses/Xanos`), trash icon só no hover
- **Chat header**: `☰ sidebar toggle` + `+ new` + `⚙ ChatSettingsModal` + `☰ menu`
- **ChatModelPicker** VS Code-style (search + grouped + speed badges `Nx` + `Upgrade` link) — depois removido do header, funcionalidade preservada em ChatSettingsModal
- **ASR scaffold** (ADR-045): MicButton com `getUserMedia` 16kHz mono + `ScriptProcessor` + IPC pipeline (`MEM_ASR_START/CHUNK/STOP/PARTIAL/FINAL/ERROR`). Implementação real do Nemotron fica pros 9 tasks do spec nemotron-asr-streaming.
- **i18n per-namespace split** + bug fix: cada componente usa `useTranslation("namespace")`; JSONs sem wrapper duplicado na raiz. Ver `src/localization/i18n.ts` docstring pra receita "como adicionar nova língua / namespace".

### Storage / config
- `.memory/config.yaml` declarativo com escopo `docs/, specs/, .specs/, README.md`
- `.memory/.gitignore` auto (não commita `.env` nem `.db`)
- ADR-040 global config opt-in (Postgres) — não aplicado neste branch

## Mudanças

### Novos arquivos (renderer)
- `src/components/chat/chat-sidebar.tsx` — sessions list com search + relative time
- `src/components/chat/mic-button.tsx` — ASR MicButton
- `src/components/ui/sheet.tsx` — shadcn Sheet com `container` + `position` props (viewport/contained)
- `src/layouts/top-bar.tsx` — drag region + pane toggles + locale switcher + window controls
- `src/layouts/workspace-shell.tsx` — splitter container 3-pane
- `src/lib/ui/use-ui-store.ts` — Zustand store do layout (explorer/editor/chat/dock visible+width)

### Novos arquivos (electron)
- `electron/ipc/asr.ts` — ASR IPC handlers com synthetic partials (Nemotron stub)

### Modificados (renderer)
- `src/localization/i18n.ts` — bootstrap com `NAMESPACES` + `SUPPORTED_LOCALES` + docstring "add language"
- `src/localization/locales/{pt-BR,en-US}/{common,topbar,chat,graph,workspace}.json` — per-namespace split
- `src/components/chat/chat-view.tsx` — header com 4 ícones + Sheet constrained inside pane + chatPaneRef
- `src/components/chat/chat-input.tsx` — recebe MicButton + dual text/partial state
- `src/components/workspace/{editor-pane,file-explorer-stub,dock-placeholder,window-controls}.tsx` — i18n bindings
- `src/components/graph/graph-view.tsx` — i18n bindings (GraphView + InspectorPanel)
- `src/layouts/{layout,status-bar,top-bar}.tsx` — i18n bindings
- `src/components/locale-switcher.tsx` — `useTranslation("common")`
- `src/lib/chat/{types,ipc}.ts` — AsrStart/Chunk/Stop/Partial/Final/Error ambient types + ipc wrappers

### Modificados (electron)
- `electron/constants/index.ts` — + `MEM_ASR_*`
- `electron/types.d.ts` — + Asr* ambient types
- `electron/preload.ts` — + `memAPI.asr.{start,chunk,stop,onPartial,onFinal,onError}`
- `electron/main.ts` — + `registerAsrHandlers`

### Removidos (dead code após limpeza)
- `src/components/chat/chat-model-picker.tsx` (376 LOC) — substituído por ChatSettingsModal
- `chat.picker.*` i18n keys em pt-BR + en-US
- `ChatModelMeta`, `CHAT_MODEL_META`, `CHAT_PROVIDER_NAMES` exports de `lib/chat/types.ts`

### Dependencies
- `+ @radix-ui/react-dialog@1.1.23` (shadcn Sheet)

## Quality Gate

- `tsc --noEmit` exit 0
- `vitest` 1/1 PASS (1 test file — example.test.ts Electron smoke)
- `biome check` tem 74 erros pré-existentes no repo (formatação + organizeImports + a11y `<html lang>`) — **fora do escopo deste PR**, fix limpo fica pra task dedicada
- `electron-forge package` exit 0 — makers filtrados em win32 (Squirrel/Deb/Rpm fora, ZIP só em macOS), `out/` vazio localmente; bundles via `.vite/` funcionam

## Validação visual

Playwright screenshot confirmou 100% das chaves i18n resolvendo (commit `f63f510`):
- "Grafo de Memória · my-memory · v3 workspace" (top bar)
- "EXPLORADOR" (CSS uppercase de "Explorador")
- "EDITOR" + "Selecione um arquivo para começar" + "Abrir arquivo" + "Buscar no vault"
- "CONVERSAS" (CSS uppercase, Sheet header)
- "Nenhuma conversa ainda. Clique em + para começar."
- Status bar: "layout: explorer 20% · chat 20% · dock: oculto" + "restaurar layout padrão"

## Commits no branch (27 ahead de develop)

Ordem cronológica (mais recente → mais antigo):

| Commit | Descrição |
|---|---|
| `25a462a` | docs(state): record v3-workspace-architecture M1 status + i18n fix |
| `f63f510` | fix(viewer): bind every component to its i18n namespace + strip dead code |
| `4f4400b` | fix(viewer): drop Settings/X from chat sidebar header |
| `e2c5c0c` | feat(viewer): constrain Sheet inside chat pane (no full-window overlay) |
| `c19630e` | feat(viewer): wrap conversations sidebar in shadcn Sheet (slide-in from right) |
| `825e225` | fix(viewer): revert min-w + auto-collapse changes (back to pre-49e7d6c) |
| `24eadad` | fix(viewer): derive sidebar visibility from windowWidth + user state |
| `49e7d6c` | fix(viewer): chat gets min-w-[400px] + sidebar auto-collapses when narrow |
| `9c66643` | fix(viewer): remove ChatModelPicker button from chat header |
| `15f70a1` | fix(viewer): replace ChevronDown with Sparkles in ChatModelPicker |
| `5f19a6e` | feat(viewer): move conversations sidebar to the right (Secondary Side Bar) |
| `39abe08` | feat(viewer): collapsible conversations sidebar + smaller chat header icons |
| `3a5afcf` | feat(viewer): VS Code chat layout — header actions + inline sessions panel |
| `885b6bd` | fix(viewer): rename sidebar title back to 'Conversas' (pt-BR) |
| `833214d` | feat(viewer): Sessions-style chat sidebar (search + grouped Newer/Older) |
| `bb16fce` | feat(viewer): microphone button + ASR IPC pipeline (ADR-045 scaffold) |
| `bc43bf2` | feat(viewer): move ChatModelPicker into chat input toolbar (VS Code position) |
| `1a18c36` | feat(viewer): VS Code-style chat model picker (search + grouped + speed badges + Upgrade) |
| `81f5810` | refactor(viewer): split i18n locales into per-namespace JSON files |
| `474b046` | feat(viewer-chat): VS Code-style model picker dropdown in chat header |
| `a29a212` | fix(viewer): make TopBar draggable (window moveable with hidden title bar) |
| `3b53848` | feat(viewer): window controls + i18n + Central vault no explorer (M1 polish) |
| `b463f80` | feat(viewer): M1 3-pane workspace shell (ADR-053) |
| `42a9110` | docs(adr+spec): ADR-053 + Roadmap v3 + M1 3-pane layout spec |

## Próximos passos (fora do escopo deste PR)

- **Squash/rebase antes do push final** — 27 commits é muito barulho, vale squash temático (M1 base / chat UX / sidebar / i18n fix)
- **Tests do ChatSidebar / ChatInput** — 2-3 testes vitest cobrindo: render do header, click no `+` cria conversation, search filtra lista
- **ADR-044** (writer atômico + outbox + reprojeção) — próximo ADR natural pós-ADR-043
- **PR #6 merge** (chat-language-models) — afeta fonte de dados quando ChatModelPicker voltar no futuro
- **M2 — React Query split + service layer** (ADR-054) — roadmap após M1 close

## Cross-references

- [`.specs/ROADMAP-v3-workspace-architecture.md`](ROADMAP-v3-workspace-architecture.md)
- [`.specs/features/feat-viewer-v3-m1-3-pane-layout/spec.md`](features/feat-viewer-v3-m1-3-pane-layout/spec.md)
- [`.specs/features/nemotron-asr-streaming/spec.md`](features/nemotron-asr-streaming/spec.md) (T1-T9 — real Nemotron fica pra spec dedicada)
- [docs/adr/053-viewer-v3-3-pane-layout-shell.md](../docs/adr/053-viewer-v3-3-pane-layout-shell.md)
- [docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md](../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md)
