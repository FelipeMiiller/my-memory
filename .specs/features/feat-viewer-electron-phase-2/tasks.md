# Tasks: feat-viewer-electron Phase 2 — Real IPC + Code Symbols + Syntax Highlight + CSP Strict + Cross-platform Release

## Context

Phase 1 MVP shipped (`0f09ad8`) and the Phase 1.5 smoke gate is green (commit TBD). The Electron viewer boots, mounts Cytoscape, renders `data-central.json`, and Playwright captures `screenshot-qg-phase15.png`. Stubs in `src/ipc/{handler,manager,router}.ts` reject every IPC call with `"not implemented (Phase 2)"` — that's the floor for Phase 2.

Phase 2 promotes the viewer from MVP demo to a usable desktop app:

1. **Real IPC** — replace stubs with handlers that shell out to the `mem` CLI binary (`./mem.exe` on Windows) via `child_process.spawn`. Renderer calls `window.memAPI.loadDataset()` and gets actual `data-central.json` content (or a fresh dataset from `mem search`).
2. **Code symbols in graph** — integrate `feat-code-ast` (ADR-047) symbols alongside docs, so a single graph view mixes `[[wikilinks]]` + `code_symbols` nodes.
3. **Syntax highlight in tooltips / snippet panels** — Shiki (or Monaco's tokenization) for code spans inside node tooltips and the future Triptych Inspector (ADR-024).
4. **CSP strict** — remove `'unsafe-inline'` from `script-src` and `style-src` via Vite nonces + `style-src` hash for bundled CSS. ADR-049 documents the threat model + nonce strategy.
5. **Cross-platform release** — fix `forge.config.ts` makers so `electron-forge make` produces installers for win32 (squirrel), darwin (zip + dmg), and linux (deb + rpm). Currently `MakerSquirrel` no-op on win32; `MakerRpm/Deb` filtered; `MakerZIP` only `["darwin"]`. Investigate root cause.

## Out of Scope

- Triptych Node Inspector 3-column layout (ADR-024) — separate feature
- Dark/light theme toggle (Phase 1 hardcoded dark)
- Live `mem` supervisor integration (`mymemoryd`) — depends on ADR-042/044 (Proposed)
- macOS code signing / notarization — Phase 3
- Auto-update via `update-electron-app` (already a dep, deferred)
- Search bar with live results — separate feature

## Phase 2 Sub-fronts

| # | Front | Files touched | New ADR? |
|---|---|---|---|
| F1 | Real IPC (`mem:dataset:load`, `mem:cli:run`, `mem:reveal`) | `src/main.ts` (`ipcMain.handle`), `src/preload.ts`, `src/ipc/{handler,manager,router}.ts`, `src/constants/index.ts` | No (continuation of VE-02) |
| F2 | Code symbols in graph (ADR-047 fan-in) | `src/components/graph/graph-view.tsx`, `src/lib/cytoscape-init.ts`, `src/ipc/router.ts` (new `mem:code:symbols` channel) | No (cross-ref ADR-047 §Deferral Plano A) |
| F3 | Syntax highlight (Shiki) | `src/components/graph/graph-view.tsx` (tooltip), `package.json` (add `shiki`) | No |
| F4 | CSP strict (remove `'unsafe-inline'`) | `src/main.ts` (CSP nonce), `vite.renderer.config.mts` (nonce plugin), `index.html` | **Yes — ADR-049** |
| F5 | Cross-platform release (electron-forge make) | `forge.config.ts` (makers), `.github/workflows/release.yml` (optional) | No (debug makers; possibly add ADR if non-trivial) |

## Suggested execution order

```
F4 (ADR-049)  ── first, because it touches src/main.ts + Vite config which F1 also needs
   ↓
F1 (Real IPC) ── depends on F4 CSP to be loose enough for runtime IPC bootstrap
   ↓
F2 (Code symbols) ── depends on F1 IPC contract for `mem:code:symbols`
   ↓
F3 (Syntax highlight) ── standalone, but visually completes F2's node tooltips
   ↓
F5 (electron-forge make) ── independent, do last once F1-F4 land
```

**Total: 5 sub-fronts.** Just over the 8-task threshold (counting each sub-front as 1+ tasks). Recommend a **single batch** worker per sub-front, with **F4 → F1 → F2 → F3 → F5** execution order (sequential; F4 is the gate that the others assume).

## Pre-flight: Achados from Phase 1.5 (carry into Phase 2)

1. **`forge.config.ts` makers don't run on win32** — `MakerSquirrel` filter unknown; `MakerRpm/Deb` filtered for non-linux; `MakerZIP` restricted to `["darwin"]`. Need to investigate root cause + decide on F5 whether to (a) filter `MakerSquirrel` explicitly for win32-only, (b) switch to `@electron-forge/maker-dmg`, (c) drop `MakerRpm/Deb` and use `MakerZIP` cross-platform.
2. **`src/types.d.ts` redeclares `MAIN_WINDOW_VITE_DEV_SERVER_URL` + `MAIN_WINDOW_VITE_NAME`** — conflict with `node_modules/@electron-forge/plugin-vite/forge-vite-env.d.ts`. Worked around in Phase 1.5 by removing the redundant `tsc -p forge.env.d.ts` from the typecheck script. Cleaner fix: delete `src/types.d.ts` (electron-forge types are canonical) or wrap with `declare global` like the forge-vite-env file does. Decide at start of F4 (ADR-049 will touch both).
3. **Cytoscape emits "Do not assign mappings to elements without corresponding data" warnings** — `node.community_color` style mapping runs against `data-central.json` nodes that lack `community_color`. Cosmetic. Either filter style by `[community_color]` selector or fix dataset. Decide during F2.
4. **`internal/mcp/x.md` untracked** — pre-existing; not blocking, not part of viewer scope.

## Quality gate (Phase 2 close)

- `gofmt -l .` clean
- `go build ./...` exit 0
- `go test -count=1 ./...` all PASS
- `npm run typecheck` exit 0
- `npm run test` (vitest) all PASS
- `npm run smoke` exit 0 + screenshot
- `npm run lint` — target 0 errors (currently 74 pre-existing; Phase 2 should NOT add more)
- `npm run make` (electron-forge) — produces installers for win32 + linux at minimum (macOS gated on having @electron-forge/maker-dmg installed)
- New ADR-049 merged before F1 lands