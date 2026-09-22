# ADR-049: Viewer Restructure — Electron Forge 7 + electron/ ↔ src/ split

## Status

**Accepted** (2026-09-22). Implementation commits:

- `0f09ad8 feat(viewer)!: migrate to LuanRoger/electron-shadcn template` —
  collapses original `viewer/` + `electron/` monorepo into flat `src/`
  with Electron Forge 7 + Vite 8 + React 19 + shadcn v4 + Tailwind 4.
- `824585c fix(viewer): Phase 1.5 smoke gate verde + typecheck drift fix` —
  consolidates Playwright smoke into `src/tests/e2e/example.test.ts`,
  drops redundant `tsc -p forge.env.d.ts` from typecheck script.
- `b2603d7 docs(viewer+spec): feat-viewer-electron Phase 1.5 closeout +
  Phase 2 outline` — adds `.specs/features/feat-viewer-electron-phase-2/tasks.md`.
- `c00037b refactor(viewer): split electron/ + src/ — renderer stays flat` —
  splits electron-side files back into a dedicated `electron/` folder.

## Context

ADR-048 (Accepted 2026-09-21) established the original Electron + React +
Vite + shadcn/ui viewer in a `viewer/` + `electron/` monorepo layout. The
goal was to validate the stack (Electron + IPC + Cytoscape + i18next)
with a working MVP before committing to it long-term.

Two structural shifts happened since:

**Shift 1 — `0f09ad8` (template migration).** The hand-rolled `viewer/+electron`
split had grown into a maintenance burden: two `package.json`s, two
`tsconfig.json`s, separate Vite configs per side, a `dev.mjs`
orchestrator script to coordinate Vite dev server + Electron boot. To
eliminate this surface area, the codebase adopted the
[`LuanRoger/electron-shadcn`](https://github.com/LuanRoger/electron-shadcn)
template, which ships a single root `package.json`, a single
`tsconfig.json`, an `@electron-forge/plugin-vite` integration, and
opinionated defaults for shadcn/ui + Tailwind 4. The cost was losing the
visible `viewer/` and `electron/` directory split — everything moved into
flat `src/` (`src/main.ts`, `src/preload.ts`, `src/renderer.ts`,
`src/ipc/...`, `src/components/...`, etc.).

**Shift 2 — `c00037b` (electron/ split).** After living with the flat
layout for one cycle, the boundary between "this file runs in Electron
main process" and "this file runs in the renderer" was unclear. `main.ts`
and `preload.ts` are Electron main; `app.tsx` and `renderer.ts` are
renderer; `ipc/`, `constants/`, `utils/path.ts`, `types.d.ts` are
electron-only. Mixing them in `src/` made the IPC contract harder to
audit and broke a few mental models (e.g. importing `@/constants` for IPC
channel names from renderer code that doesn't need them).

The fix: split electron-side files back into `electron/` while keeping
renderer code (`src/`) flat. Path aliases formalize the boundary:

| Alias | Resolves to | Used by |
| :--- | :--- | :--- |
| `@/*` | `./src/*` | renderer |
| `@electron/*` | `./electron/*` | main, preload, ipc handlers |

This preserves the template's single-package, single-tsconfig, single-
Forge-config benefits (which we want — keep that), while making the
process boundary visible at the filesystem level (which we also want —
keep that). Best of both: keep template ergonomics + keep the file-level
separation of concerns that the original `viewer/+electron/` layout
provided.

## Decision Outcome

Adopt the **`electron/` + `src/` split** with the following layout:

```
electron/                            ← Electron main process only
  main.ts                            BrowserWindow + IPC handlers
  preload.ts                         contextBridge.memAPI
  ipc/{context,handler,manager,router}.ts
  constants/index.ts                 IPC_CHANNELS, LOCAL_STORAGE_KEYS
  utils/path.ts                      getBasePath (ESM-safe dirname)
  types.d.ts                         MAIN_WINDOW_VITE_* magic consts
src/                                 ← Renderer (React app, flat)
  app.tsx                            React root
  renderer.ts                        Vite renderer entry
  components/{graph,ui,locale-switcher}/
  layouts/layout.tsx                 Tabs + Sidebar shell
  lib/cytoscape-init.ts              Cytoscape ESM init
  localization/{i18n,locales}/
  styles/global.css                  Tailwind 4 + shadcn HSL variables
  tests/{unit,e2e}/                  vitest + playwright
  types/css.d.ts                     CSS module declarations
  utils/tailwind.ts                  cn() helper
```

Build tooling stays the **LuanRoger template**:

- **Electron Forge 7** with `@electron-forge/plugin-vite` for unified
  main + preload + renderer builds.
- **Vite 8** for the renderer (HMR in dev) and the main/preload (esbuild
  bundling via plugin-vite).
- **React 19** with `@vitejs/plugin-react` (React Compiler preset) +
  `babel-plugin-react-compiler`.
- **shadcn v4** components copied into `src/components/ui/`.
- **Tailwind 4** via `@tailwindcss/vite` (CSS-first config in
  `src/styles/global.css`).

Path aliases (in `vite.main.config.mts` + `vite.preload.config.mts` + tsconfig):

```json
{
  "paths": {
    "@/*": ["./src/*"],
    "@electron/*": ["./electron/*"]
  }
}
```

Renderer keeps `@/*` (used by `app.tsx`, `components/`, `layouts/`).
Main + preload use `@electron/*` exclusively. Imports that cross the
boundary (renderer asking for IPC channel names) go through the
`window.memAPI` surface, not through `@/`-style imports — keeping the
security boundary tight.

**Smoke test stays in `src/tests/e2e/example.test.ts`** (Playwright
Test runner, uses `testDir: "./src/tests/e2e"`). It boots Electron from
`.vite/build/main.js` (the VitePlugin flattens the output — see §Notes
below) and asserts the window title + screenshots the rendered graph.

## Consequences

### Positive

- Process boundary is visible at the filesystem level — easier to audit
  "what runs where" during security review (ADR-050 cycle).
- Renderer code (`src/`) stays flat and easy to navigate (the original
  intent of the LuanRoger migration).
- Single root `package.json` + single `tsconfig.json` retained from
  template — no monorepo ceremony.
- `npm run smoke` now passes deterministically (`1 passed in 2.4s`,
  Playwright Electron + screenshot capture).
- Typecheck (`tsc --noEmit`) now passes cleanly (exit 0) after dropping
  the redundant `tsc -p forge.env.d.ts` step that conflicted with the
  electron-forge Vite env declarations.

### Negative / Risks

- `electron-forge package` exits 0 but produces no `out/` directory on
  win32 — the `MakerSquirrel` / `MakerDeb` / `MakerRpm` entries in
  `forge.config.ts` are platform-filtered and none match win32. This is
  a pre-existing issue carried over from ADR-048 and not introduced by
  this restructure; deferred to Phase 2 F5
  (`.specs/features/feat-viewer-electron-phase-2/tasks.md`).
- `src/types.d.ts` (now `electron/types.d.ts`) redeclares
  `MAIN_WINDOW_VITE_DEV_SERVER_URL` / `MAIN_WINDOW_VITE_NAME` that
  `node_modules/@electron-forge/plugin-vite/forge-vite-env.d.ts` already
  declares. The redundant `tsc -p forge.env.d.ts` was the workaround;
  the cleaner fix (delete `electron/types.d.ts` or wrap in
  `declare global`) is deferred to Phase 2 F4 / ADR-049 promotion.
- Cytoscape emits "Do not assign mappings to elements without
  corresponding data" warnings for nodes that lack `community_color` in
  `data-central.json`. Cosmetic. Decision deferred to Phase 2 F2.

## Notes

**Why output is `.vite/build/main.js` not `.vite/build/electron/main.js`.**
The `@electron-forge/plugin-vite` VitePlugin uses the entry filename
(without extension) as the output basename. So `entry: 'electron/main.ts'`
produces `.vite/build/main.js`, not `.vite/build/electron/main.js`. The
template doesn't preserve the source directory in the output. This
matches the asar layout that the packaged app would receive
(`out/my-memory-win32-x64/resources/app.asar/main.js`), so package.json
`main: ".vite/build/main.js"` is correct.

**Path-alias migration cost.** Three imports changed:
`src/main.ts` (`@/constants` → `@electron/constants`, `@/utils/path` →
`@electron/utils/path`); `src/preload.ts` (`@/constants` →
`@electron/constants`). Plus `tsconfig.json` paths and tsconfig include
patterns, plus vite configs. Total churn: 15 files, of which 9 are pure
renames (git detected them as `R` operations). No logic changes.

## Related

- **ADR-048** (`docs/adr/048-viewer-electron-react-vite-shadcn.md`) —
  original Electron + React + Vite + shadcn/ui decision (Phase 1 MVP).
  **Superseded** by this ADR for the layout split; substantively the
  stack decision is unchanged.
- **ADR-037** (`docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md`) —
  predecessor (Vite + Vanilla TS approach). Already superseded by
  ADR-048.
- **ADR-050** (`docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md`)
  — Threat model. This restructure's clean process boundary helps
  future threat-model work; specifically the
  `src/types.d.ts` redeclaration will be cleaned up in Phase 2 F4.
- **Phase 2 tasks** (`.specs/features/feat-viewer-electron-phase-2/tasks.md`)
  — five sub-fronts (F4 ADR-049 CSP strict → F1 Real IPC → F2 Code
  symbols → F3 Shiki syntax → F5 electron-forge make). This restructure
  is a prerequisite for F1 (Real IPC) because the new `electron/ipc/*`
  files will host the real handlers replacing the current
  `Promise.reject("not implemented (Phase 2)")` stubs.
- **`.specs/features/feat-viewer-electron/spec.md`** — original Phase 1
  spec, now marked SUPERSEDED at the top.