# ADR-037: Rewrite do Viewer com Vite + Vanilla TypeScript

- **Date**: 2026-09-17
- **Status**: Accepted
- **Deciders**: Felipe Miiller
- **Tags**: viewer, graphview, vite, typescript, refactor, frontend-tooling, deferred

## Context and Problem Statement

O viewer interativo de grafo (`graph-v2.html`, ~42KB, 13 componentes) funciona e está visualmente aceitável após os fixes desta sessão (fCoSE + text-background labels + hide-tags toggle). **Mas é monolítico e baseado em CDN**:

1. **Tailwind via CDN** (`cdn.tailwindcss.com`) — geração de CSS em runtime, slow first-paint, warning explícito do Tailwind "não use em produção".
2. **Cytoscape + dagre + fcose + cose-base via unpkg** — 3 scripts externos, dependem de CDN estar disponível.
3. **`const DATA = {...}` inline** (agora extraído pra `data-central.json` via fetch), ainda com bootstrap async manual.
4. **Zero build step** — TypeScript? ESM? Nada. JS cru inline em `<script>`.
5. **Acoplamento direto DOM ↔ Cytoscape ↔ state** — sem camada de separação, debug difícil.
6. **Features pendentes no ADR-036 que ficam em modo "manual patch" no v2**:
   - B4: `mem open` reader registry
   - B5: OpenWith modal persistence (localStorage)
   - C3: Mini-mapa interativo (drag-to-navigate)
   - C4: Animações de entrada
   - C5: Edge labels polish

Cada um desses foi "deferred" mas continua exigindo patches incrementais no mesmo arquivo monolítico.

**Decisão do Felipe nesta sessão:** viewer vai pro final do roadmap. Quando voltar, **rewrite do zero com stack moderno** (Vite + Vanilla TypeScript), não mais patch incremental no HTML legado.

## Decision Drivers

- **Tech debt estrutural não justifica patches**: Tailwind CDN + scripts CDN inline não escalam pra features planejadas (multi-tab, B4 reader registry, drag-to-navigate no mini-mapa).
- **DX**: TypeScript traz autocomplete e refactor safety. Vite traz HMR (hot module replacement) — ver mudanças de CSS sem F5.
- **Build step necessário**: Tree-shaking do Cytoscape, separação vendor chunks, source maps pra debug.
- **Sem framework lock-in**: Não queremos React/Vue/Svelte ancorando o repo. Vanilla TS + Web Components mantém o viewer agnóstico.
- **Reutilizar lógica do v2**: O algoritmo de fCoSE, text-background labels, hide-tags, type chips, repos registry — tudo vira módulo TS reutilizável.
- **Migração paralela**: Manter `graph-v2.html` funcionando durante a migração. Sem deadline, sem big-bang switch.

## Considered Options

### 1. Manter v2 + patches incrementais (rejeitado)
Patch cada item B4/B5/C3/C4/C5 direto no `graph-v2.html`. Custo de manutenção crescente, impossível testar unitariamente, zero type safety.
- ❌ Acumula tech debt
- ❌ CDN dependencies continuam
- ❌ Sem HMR
- ✅ Sem custo de migração

### 2. Migrar pra Vite + React (rejeitado)
React + Vite é o combo mais popular mas pesado (~40KB gzipped só do React runtime) e adiciona paradigma declarativo que conflita com a manipulação direta de Cytoscape.
- ❌ Bundle size inflado
- ❌ Paradigma declarativo vs imperativo (Cytoscape é imperativo)
- ✅ Ecossistema enorme
- ✅ Familiar pra maioria dos devs

### 3. Migrar pra Vite + Vue 3 (rejeitado)
Vue é mais leve que React mas ainda é framework. Single-File Components (SFC) ajudam mas o lock-in continua.
- ❌ Lock-in de framework
- ❌ Bundle ainda tem runtime Vue
- ❌ SFCs adicionam camada (.vue files)
- ✅ Reativo (composition API combina com state do viewer)

### 4. Migrar pra Vite + Svelte (rejeitado)
Compila pra JS vanilla, super leve, mas Felipe não usou Svelte antes e o ecosistema pra Cytoscape wrappers é mínimo.
- ❌ Stack não conhecida
- ❌ Poucos exemplos de integração Cytoscape+Svelte
- ❌ Lock-in de qualquer forma
- ✅ Bundle minúsculo (compilação)
- ✅ Reativo built-in

### 5. Vite + Vanilla TypeScript + Web Components (escolhido)
Mínimo de magia. Cytoscape é imperativo e usa diretamente. Web Components permitem encapsular sub-componentes (chips, modals, mini-mapa) sem framework.
- ✅ Zero framework runtime
- ✅ Tree-shaking automático (Vite + ESM)
- ✅ TypeScript + HMR out-of-box
- ✅ Reutiliza lógica Cytoscape diretamente
- ✅ Web Components pra encapsular (mini-mapa, modal, chips)
- ✅ Bundle target: < 50KB gzipped (vs ~70KB CDN atual)
- ❌ Mais código boilerplate que framework
- ❌ Sem SFC, sem reactive primitives built-in

### 6. Migrar pra Astro (rejeitado)
Astro é ótimo pra sites content-heavy (docs, blog) com islands. Viewer é altamente interativo — não se encaixa no modelo "maioria estático".
- ❌ Modelo errado (interativo vs content)
- ❌ Adiciona camada SSR que não agrega nada

## Decision Outcome

**Opção 5 escolhida**: **Vite + Vanilla TypeScript + Web Components**, com `@webcomponents/custom-elements` polyfill pra Safari < 16.4.

### Estrutura proposta

```
graph-v3/
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html                # shell HTML, mount <graph-app>
├── src/
│   ├── main.ts               # entry, monta <graph-app>
│   ├── data/
│   │   ├── loader.ts         # fetch JSON, cache, hash-check
│   │   └── schema.ts         # TypeScript types
│   ├── components/
│   │   ├── graph-app.ts      # root Web Component, monta Cytoscape
│   │   ├── header-bar.ts     # título + stats + chips
│   │   ├── type-chips.ts     # filter chips (B1)
│   │   ├── repos-panel.ts    # B2 list
│   │   ├── open-with-modal.ts # B5 (futuro)
│   │   ├── mini-map.ts       # C3 (futuro, com drag)
│   │   └── config-modal.ts
│   ├── graph/
│   │   ├── renderer.ts       # Cytoscape setup + styles
│   │   ├── layouts.ts        # fcose, cose, grid-community, etc.
│   │   └── filters.ts        # toggleTagNodes, type filter
│   ├── state/
│   │   └── store.ts          # signal-based reactive store (sem framework)
│   ├── readers/
│   │   ├── registry.ts       # B4: Obsidian/VSCode/Typora/Mark Text/Joplin/Zettlr
│   │   └── open.ts           # dispatch por reader
│   └── styles/
│       ├── tokens.css        # CSS custom properties (cores, spacing)
│       ├── tailwind.css      # @tailwind directives (build-time, não CDN)
│       └── components.css    # per-component styles
├── public/                   # assets estáticos
└── tests/
    ├── unit/                 # Vitest, *.test.ts
    └── e2e/                  # Playwright, *.spec.ts
```

### Stack lock-in (mínimo)

| Package | Propósito | Versão-alvo |
|---|---|---|
| `vite` | Build/dev server | ^5.x |
| `typescript` | Type safety | ^5.x |
| `cytoscape` | Graph engine | ^3.30.x |
| `cytoscape-fcose` | Force-directed layout | ^2.2.x |
| `cytoscape-dagre` | Hierarchical layout | ^2.5.x |
| `@webcomponents/custom-elements` | Polyfill | ^1.6.x |
| `tailwindcss` | Utility CSS (build-time) | ^3.4.x |
| `@nanostores/vanilla` ou signal próprio | Reactive store (~2KB) | — |

**Decisão importante: SEM React/Vue/Svelte.** State via store próprio (~50 LOC) com Proxy + Signal pattern, ou usar `@nanostores/vanilla` (~600B).

### Plano de migração (ordem sugerida)

1. **Sprint 1 (Sessão futura)** — Scaffold
   - `npm create vite@latest graph-v3 -- --template vanilla-ts`
   - Configurar `vite.config.ts` com paths aliases (`@components/`, `@graph/`, `@state/`)
   - Configurar `tsconfig.json` strict mode
   - Setup Tailwind build-time (não CDN)
   - Adicionar Cytoscape + fcose + dagre como deps

2. **Sprint 2** — Core rendering
   - Migrar `GraphRenderer.init()` pra `src/graph/renderer.ts`
   - Migrar styles (CSS custom properties + tailwind utilities)
   - Migrar `data-central.json` → `src/data/loader.ts` com hash-check
   - Validar: v3 renderiza o mesmo grafo que v2 (Playwright visual diff)

3. **Sprint 3** — Web Components
   - `<graph-app>` shell
   - `<header-bar>` (título + stats)
   - `<type-chips>` (B1)
   - `<repos-panel>` (B2)
   - `<config-modal>`

4. **Sprint 4** — Features deferred (ADR-036)
   - B4: `readers/registry.ts` + `readers/open.ts`
   - B5: `localStorage` persistence em `<open-with-modal>`
   - C3: `<mini-map>` com drag-to-navigate
   - C4: Stagger animations (CSS `animation-delay`)
   - C5: Edge labels polish

5. **Sprint 5** — Switch
   - `graph-v2.html` mantido como fallback
   - `graph-v3/index.html` em paralelo
   - Quality gate passa em ambos
   - Switch oficial: docs/README aponta pro v3

### Critério de aceitação da migração

- ✅ Bundle gzipped < 50KB (vs ~70KB atual com CDN)
- ✅ First Contentful Paint < 200ms local (vs ~800ms CDN)
- ✅ Lighthouse Performance ≥ 95
- ✅ TypeScript strict: 0 erros
- ✅ Testes: Vitest (unit) + Playwright (e2e) cobrem ≥ 80% do código
- ✅ `graph-v2.html` continua funcionando durante a migração (sem big-bang)

## Consequences

### Positive
- **Tech debt eliminado**: zero CDN em runtime, build-time Tailwind, type safety.
- **DX salto**: HMR no dev server, autocomplete TypeScript, refactor safety.
- **Features deferred aceleram**: cada item B4/B5/C3/C4/C5 cabe em sprint próprio com testes.
- **Bundle menor**: ~30% menor gzipped, self-contained (funciona offline).
- **Testabilidade**: Cytoscape logic isolada em módulos testáveis com Vitest.

### Negative
- **Setup inicial pesado**: ~1 sprint só pra scaffold (Vite + TS + Tailwind + Web Components + tooling).
- **Novo vetor de bugs**: build step pode introduzir issues que não existiam no v2 (tree-shaking quebrado, paths errados, etc).
- **Manutenção dupla durante migração**: v2 e v3 coexistem, ambos precisam de quality gate.
- **Curva Web Components**: Felipe não usou antes. É Vanilla JS API, simples mas verbosa.

### Neutral
- **ADR-036 não muda**: itens B4/B5/C3/C4/C5 continuam listados lá. Este ADR-037 só reorganiza o COMO, não O QUÊ.
- **`mem graph` standalone continua**: a saída HTML standalone do `internal/graphview/template.go` não é afetada. v3 é uma alternativa paralela, não substituição do export CLI.
- **CDN dependencies removidas gradualmente**: Cytoscape, fcose, dagre viram deps npm. Tailwind vira build-time.

## Implementation Plan

### Sessão 1 (próxima, só viewer-side)
- Decidir se Felipe quer scaffoldar com template oficial `npm create vite@latest` ou escrever config manual (decisão de DX).
- Criar `graph-v3/` separado de `graph-v2.html` (não substituir imediatamente).
- Setup Tailwind build-time (resolver warning "use Tailwind CLI").
- Adicionar TypeScript strict.

### Sessão 2-N (sprint 2+)
- Migrar lógica incrementalmente (renderer → layout → filters → chips → modals).
- Quality gate em ambos os viewers a cada commit.
- Manter `graph-v2.html` como fallback até v3 estar feature-complete.

## References

- [ADR-020: Visualizador Interativo de Grafo em HTML/SVG Standalone](./020-visualizador-interativo-de-grafo-em-html-svg.md) — v1 viewer, base arquitetural
- [ADR-036: Consolidação Pós-Release v1.3.0](./036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md) — backlog Trilha B (B4, B5) + Trilha C (C3, C4, C5)
- [ADR-006: Interoperabilidade Obsidian Flavored Markdown e JSON Canvas](./008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md) — origem do viewer
- [Vite docs](https://vitejs.dev/) — guia oficial
- [Lit docs](https://lit.dev/) — **alternativa descartada** mas boa referência de Web Components minimalistas
- [Tailwind CSS install](https://tailwindcss.com/docs/installation) — modo CLI/build (não CDN)