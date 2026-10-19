# Renderizador vis-network para `mem graph view`

## Problem Statement

`mem graph view` já gera um grafo interativo real: `internal/graphview/template.go` são 38 KB de HTML inline com simulação de forças, busca por texto, filtro por PageRank, sidebars e export SVG/canvas. O modelo de dados em `internal/graphview/model.go` já carrega o que um visualizador profissional precisa — `Node.Color`, `Node.Radius`, `Node.PageRank`, `Node.CommunityID`, `Node.CommunityColor`, `Edge.Weight`, `Edge.Color`, `GraphStats.Modularity`.

O custo é que esse renderer é **feito à mão**: 38 KB de JS e CSS que escrevemos e mantemos nós mesmos, sem-community contributions, com simulação de forças aproximada e sem clustering nativo.

O histórico do repositório registra o custo do caminho oposto: o viewer Electron (`b477200`, 112 arquivos, 27.858 linhas) introduziu `package.json`, Vite, React, Forge e Playwright, e foi revertido. A lição não é "não use bibliotecas", é **não introduza build step**.

vis-network atende exatamente a restrição: build **standalone UMD**, carregado por `<script>`, sem bundler. Versão 10.1.2 (2026-08-19), dual-license Apache-2.0 OU MIT, ~862k downloads semana, mantida ativamente (último push 2026-09-20, repositório não arquivado).

## Goals

- [ ] Entregar um renderizador vis-network **sem build step** e **sem CDN**, que abre offline
- [ ] Manter o renderizador atual como padrão, sem alterar sua saída
- [ ] Vendorizar o asset com licença, proveniência e verificação por hash
- [ ] Preservar PageRank e comunidade LPA do Go — a vis não tem modularidade Q

## Out of Scope

| Feature | Reason |
|---|---|
| Remover o renderizador atual | `mem graph view` sem flag precisa continuar funcionando como hoje |
| Usar clustering da vis-network | Ela agrupa mas não calcula modularidade Q; o LPA de `internal/graph` já entrega isso (ADR-022) |
| CDN em vez de vendorizar | Reintroduz dependência de rede no artefato que abre offline — exatamente o que o revert de `b477200` eliminou |
| Layout calculado em Go | O layout é o que a vis faz bem; recalcular em Go seria reimplementar física |
| `package.json` / npm / bundler | Proibido pela lição de `b477200` |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | --------------- | --------- | ---------- |
| Engine atual como padrão | `mem graph view` sem flag gera exatamente o HTML de hoje | Quem já usa não pode ter o output quebrado por um upgrade | y |
| Asset vendorizado e **commitado** | sim, em `internal/graphview/assets/` | Um asset gitignored precisa ser baixado na primeira visualização — isso é dependência de rede disfarçada. Licença dupla permite commitar | y |
| Versão pinada | `10.1.2` com hash fixado | vis-network publica por semantic-release; `package.json` no repo tem `0.0.0-no-version`, então a versão só existe no dist-tag do npm | y |
| Física para grafo pequeno | desligada abaixo de 30 nós | Para 8 nós a simulação é ruído; posições determinísticas deixam o viewer reproduzível | n |
| Sem alteração em `builder.go` | o modelo existente é mapeado 1:1 | `Node` e `Edge` já carregam cor, raio, peso e comunidade | y |

**Open questions:** none.

### Dimensões de requisito implícito (sweep Large)

| Dimensão | Destino |
|---|---|
| Input validation & bounds | PARITY-19 — `--engine` com valor desconhecido precisa falhar com erro claro, não silenciosamente cair no padrão |
| Failure / partial-failure states | PARITY-16 — `--engine vis` sem o asset no disco precisa explicar como obter, não emitir HTML quebrado |
| Idempotency / retry | PARITY-05 — `--out` repetido reescreve sem acumular estado |
| Auth boundaries & rate limits | N/A — o grafo é local; nada exposto além do HTML gerado |
| Concurrency / ordering | N/A — geração é single-shot |
| Data lifecycle / expiry | PARITY-04 — pin de versão com hash impede que um upgrade não auditado mude o render silenciosamente |
| Observability | PARITY-12 — a versão da vis usada aparece no HTML gerado, para saber de onde veio a saída |
| External-dependency failure | PARITY-04, PARITY-16 — o asset é local justamente para não haver falha de rede |
| State-transition integrity | N/A — nenhuma migration |

---

## User Stories

### P1: Asset vendorizado com proveniência ⭐ MVP

**User Story**: Como maintainer, quero que o HTML do grafo abra offline e que eu saiba exatamente qual versão da biblioteca gerou a saída, para não depender de CDN nem de um upgrade invisível.

**Why P1**: Sem isso, nenhuma das outras histórias faz sentido — o resto do renderizador depende do asset existir.

**Acceptance Criteria**

1. The repository SHALL contain `internal/graphview/assets/vis-network.min.js` at version `10.1.2`, the standalone UMD build
2. The repository SHALL contain `internal/graphview/assets/LICENSE-vis-network.txt` carrying the Apache-2.0 / MIT dual-license notice of vis.js
3. The repository SHALL contain `internal/graphview/assets/PROVENANCE.md` recording the source URL, the exact version, the SHA-256 of the file, and the retrieval date
4. IF the vendored file's SHA-256 differs from the one recorded in `PROVENANCE.md` THEN a test SHALL fail
5. The vendored asset SHALL be tracked by git, not gitignored
6. The asset SHALL be embedded in the binary with `go:embed`, so a freshly installed `mem` can render the vis engine with no extra setup
7. IF `internal/graphview/assets/vis-network.min.js` is missing THEN `go build ./...` SHALL fail with a clear message, because the asset is git-tracked and its absence is a visible error rather than a silently broken binary
8. `mem graph view --engine vis --out g.html` SHALL write **two** files next to each other: `g.html` and `vis-network.min.js`, taken from the embedded bytes
9. IF the output directory is copied elsewhere THEN the pair SHALL still render with no network access
10. The build SHALL NOT require the `.map` source map, which SHALL NOT be vendored

**Independent Test**: hash the vendored file, compare against `PROVENANCE.md`.

---

### P2: Renderizador vis-network

**User Story**: Como pessoa explorando o grafo, quero física de simulação, zoom, pan e seleção de vizinhança sem plugin e sem build step, para achar a nota que procuro pelo lugar que ela ocupa no mapa.

**Why P2**: É a entrega de valor; o resto é suporte.

**Acceptance Criteria**

1. WHEN `mem graph view --engine vis` runs THEN it SHALL emit HTML that loads `vis-network.min.js` from a path relative to the emitted HTML file
2. The emitted HTML SHALL NOT reference any CDN, unpkg, jsdelivr or other remote origin
3. IF the emitted HTML is opened with no network access THEN it SHALL render the graph
4. Each graph node SHALL map `id` to `id`, `Title` to `label`, `Color` to `color`, and `Radius` to `value`
5. Each graph edge SHALL map `Source` to `from`, `Target` to `to`, and `Weight` to `width`
6. `Node.PageRank` SHALL be visible in the node detail panel
7. `Node.CommunityColor` SHALL be used as the node colour when `Node.Color` is empty
8. WHEN a node is clicked THEN the page SHALL expose the note's deeplink through `internal/deeplink`
9. IF the graph has fewer than 30 nodes THEN physics SHALL be disabled and positions SHALL be deterministic
10. IF the graph has 30 nodes or more THEN physics SHALL be enabled
11. The emitted HTML SHALL record the vis-network version that rendered it
12. `--root` and `--depth` SHALL behave identically under both engines
13. `GraphStats` (total nodes, edges, density, community count, modularity) SHALL appear in the rendered page

**Independent Test**: emit from a fixture graph, assert the HTML references the local asset, contains no remote origin, and that node/edge counts in the embedded JSON match the model.

---

### P3: A flag no CLI

**User Story**: Como usuário do CLI, quero escolher o renderizador e ver um erro claro quando escolher algo que não existe, para não receber HTML inesperado.

**Acceptance Criteria**

1. `mem graph view` with no `--engine` SHALL produce byte-identical output to the pre-feature renderer
2. `--engine vis` SHALL select the vis-network renderer
3. IF `--engine` receives an unrecognised value THEN the command SHALL exit non-zero naming the accepted values
4. IF `--engine vis` is requested and the vendored asset is missing from disk THEN the command SHALL exit non-zero explaining how to obtain it
5. `printHelp()` SHALL document `--engine` and its two accepted values
6. A test SHALL assert every dispatch value of `--engine` appears in `printHelp()`

**Independent Test**: run the command with a bad `--engine` and with a missing asset.

---

### P2: Documentação e decisão

**User Story**: Como leitor da documentação, quero saber que existe um segundo renderizador e por que ele foi escolhido, para não estranhar um artefato de 670 KB no repositório.

**Acceptance Criteria**

1. `docs/CLI_GUIDE.md` SHALL document `mem graph view --engine vis` and both accepted values
2. `docs/adr/README.md` SHALL index the ADR that records this decision
3. The ADR SHALL state the size cost, the licence, and why CDN was rejected
4. The ADR SHALL record that the existing renderer remains the default
5. `skills/mymemory-graph/SKILL.md` SHALL mention both engines
6. `CHANGELOG.md` `[Unreleased]` SHALL record the new engine
7. Files written in this spec SHALL be UTF-8 without BOM

**Independent Test**: grep `vis` across the guide, the skill and the changelog.

---

## Edge Cases

- IF the graph has 0 nodes THEN both renderers SHALL emit a page stating the vault has no indexed documents, not a blank canvas
- IF `Node.Color` and `Node.CommunityColor` are both empty THEN the vis renderer SHALL fall back to a neutral slate, matching `GetColorForCommunity`
- IF a node id contains characters that break HTML embedding THEN the JSON SHALL be emitted with the template's existing escaping, not hand-rolled
- IF the vis asset version on disk differs from `PROVENANCE.md` THEN `go test ./internal/graphview/...` SHALL fail, so the drift is caught before a viewer is generated
- IF `--engine vis` and `--out` point at a directory that does not exist THEN the command SHALL create it or fail with a clear message

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| VIS-01 | P1: Asset vendorizado | Execute | Pending |
| VIS-02 | P1: Asset vendorizado | Execute | Pending |
| VIS-03 | P1: Asset vendorizado | Execute | Pending |
| VIS-04 | P1: Asset vendorizado | Execute | Pending |
| VIS-05 | P1: Asset vendorizado | Execute | Pending |
| VIS-06 | P1: Asset vendorizado | Execute | Pending |
| VIS-07 | P1: Asset vendorizado | Execute | Pending |
| VIS-08 | P1: Asset vendorizado | Execute | Pending |
| VIS-09 | P1: Asset vendorizado | Execute | Pending |
| VIS-10 | P1: Asset vendorizado | Execute | Pending |
| VIS-11 | P2: Renderizador vis-network | Execute | Pending |
| VIS-12 | P2: Renderizador vis-network | Execute | Pending |
| VIS-13 | P2: Renderizador vis-network | Execute | Pending |
| VIS-14 | P2: Renderizador vis-network | Execute | Pending |
| VIS-15 | P2: Renderizador vis-network | Execute | Pending |
| VIS-16 | P2: Renderizador vis-network | Execute | Pending |
| VIS-17 | P2: Renderizador vis-network | Execute | Pending |
| VIS-18 | P2: Renderizador vis-network | Execute | Pending |
| VIS-19 | P2: Renderizador vis-network | Execute | Pending |
| VIS-20 | P2: Renderizador vis-network | Execute | Pending |
| VIS-21 | P2: Renderizador vis-network | Execute | Pending |
| VIS-22 | P2: Renderizador vis-network | Execute | Pending |
| VIS-23 | P2: Renderizador vis-network | Execute | Pending |
| VIS-24 | P3: A flag no CLI | Execute | Pending |
| VIS-25 | P3: A flag no CLI | Execute | Pending |
| VIS-26 | P3: A flag no CLI | Execute | Pending |
| VIS-27 | P3: A flag no CLI | Execute | Pending |
| VIS-28 | P3: A flag no CLI | Execute | Pending |
| VIS-29 | P3: A flag no CLI | Execute | Pending |
| VIS-30 | P2: Documentação e decisão | Execute | Pending |
| VIS-31 | P2: Documentação e decisão | Execute | Pending |
| VIS-32 | P2: Documentação e decisão | Execute | Pending |
| VIS-33 | P2: Documentação e decisão | Execute | Pending |
| VIS-34 | P2: Documentação e decisão | Execute | Pending |
| VIS-35 | P2: Documentação e decisão | Execute | Pending |
| VIS-36 | P2: Documentação e decisão | Execute | Pending |

**Coverage:** 36 total, 36 mapped to stories, 0 unmapped.
