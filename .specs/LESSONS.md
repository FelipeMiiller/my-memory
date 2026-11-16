# LESSONS - auto-maintained by scripts/lessons.py

> Machine-owned. Do NOT hand-edit. Changes are overwritten on the next `lessons.py` write.
> Canonical state lives in `.specs/lessons.json`. Edit lessons only via the script.
> promote_threshold=2 distinct features · window_days=45 · quarantine_threshold=2

## Confirmed (load these at Specify/Design)

Corroborated across multiple features. Safe to apply as guidance.

_none_

## Candidates (under observation - do NOT load as guidance yet)

Seen once or not yet corroborated. Tracked, not trusted.

### L-001 - A test that wraps the call under test in a closure returning a hardcoded nil asserts nothing: capture the real return value so the exit-code contract is discriminated.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: mutant F (cmd/mem/main.go:1711) (cmd/mem)
- last seen: 2026-11-13T23:08:16Z

### L-002 - Assert the pinned version and the final install path of a downloaded artifact, not only that the bytes arrived.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/embedder` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC5 (internal/embedder/model_downloader.go:23) (internal/embedder)
- last seen: 2026-11-13T23:08:17Z

### L-003 - When the product advertises a label in the command output, assert that label at the command layer, not only at the internal library layer.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC4 (.github/workflows/release.yml:123) (cmd/mem)
- last seen: 2026-11-13T23:08:17Z

### L-004 - When two acceptance criteria of the same feature count the same thing, name the unit once and reuse it in both.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC9 (cmd/mem/main.go:1505) (spec)
- last seen: 2026-11-13T23:08:17Z

### L-005 - When a criterion only requires a message to declare a state, pin the wording in the spec; otherwise the test freezes an invented string.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-release-onnx-provider
- evidence: AC10 (cmd/mem/main.go:1515) (spec)
- last seen: 2026-11-13T23:08:18Z

### L-006 - A vector search that reads TurboQuant rows is not killed by zeroing the raw vector column: assert the path the search actually reads, or the sensor reports a false green.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: fix-release-onnx-provider
- evidence: cmd/mem/main.go:2222 (cmd/mem)
- last seen: 2026-11-14T11:14:09Z

### L-007 - Toda chave array nova no payload de uma tool MCP precisa entrar na normalizacao da fronteira do structuredContent e no teste que proibe null: o contrato do wire e por chave, nao por tipo.
- signal: `gate_fail` · recurrence: 1 feature(s) · scope: `internal/mcp` · harmful: 0
- features: fix-doctor-unindexed-links
- evidence: internal/mcp/structured_content.go:65 (internal/mcp)
- last seen: 2026-11-14T18:27:21Z

### L-008 - Numero de Health Score citado em Out of Scope envelhece quando outra spec mexe na mesma formula: remeça antes de reutilizar o valor medido por outra spec.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: fix-doctor-unindexed-links
- evidence: .specs/features/fix-doctor-unindexed-links/spec.md:68 (spec)
- last seen: 2026-11-14T18:27:21Z

### L-009 - A checked task box is not evidence: an acceptance criterion is covered only when a production call site and a test exercise it, because a formatter with no caller is an unimplemented feature.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/staleness` · harmful: 0
- features: staleness-detection-and-banners
- evidence: P3-1 (internal/staleness/detector.go:309) (internal/staleness)
- last seen: 2026-11-15T15:02:29Z

### L-010 - When the spec pins a literal output string, assert that exact literal and fix the code or the spec; never rewrite the docs to match drifted code.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: staleness-detection-and-banners
- evidence: P1-1 (internal/staleness/detector.go:302) (spec)
- last seen: 2026-11-15T15:02:29Z

### L-011 - When a spec enumerates categories of a report, define their representation in the data model before implementing, or the categories silently merge into one bucket.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: staleness-detection-and-banners
- evidence: P2-2 (internal/staleness/detector.go:31) (spec)
- last seen: 2026-11-15T15:02:30Z

### L-012 - Teste do default de uma flag nao cobre a renderizacao: a assercao tem que olhar a SAIDA do comando, nao o valor lido da flag.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cmd/mem` · harmful: 0
- features: progressive-context-loading
- evidence: cmd/mem/main.go:2368 (cmd/mem)
- last seen: 2026-11-15T15:05:30Z

### L-013 - O schema de uma tool MCP precisa de assercao propria: um teste comportamental passa mesmo com a chave removida do schema declarado.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `internal/mcp` · harmful: 0
- features: progressive-context-loading
- evidence: internal/mcp/tools.go:54 (internal/mcp)
- last seen: 2026-11-15T15:05:30Z

### L-014 - Orcamento de tamanho tem que ser medido no artefato emitido (o Markdown final), nao em um contador interno que ignora o enquadramento que o proprio codigo renderiza.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: internal/graph/pack.go:284 (internal/graph)
- last seen: 2026-11-15T15:17:57Z

### L-015 - Filtro que muda o conjunto selecionado nao pode ser testado por contagem: afirme o conjunto exato que cada modo do filtro deve produzir.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: M3 internal/graph/pack.go:147 (internal/graph)
- last seen: 2026-11-15T15:17:57Z

### L-016 - Quando o criterio define uma ordem de prioridade, afirme a ordem dos elementos selecionados, nao apenas o conjunto deles.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: M4 internal/graph/pack.go:213 (internal/graph)
- last seen: 2026-11-15T15:17:58Z

### L-017 - Tratamento especial de um no especifico so esta coberto quando a assercao fixa o tier dele, e nao apenas as contagens agregadas do resultado.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: M2 internal/graph/pack.go:247 (internal/graph)
- last seen: 2026-11-15T15:17:58Z

### L-018 - Criterio que pede um resumo sem dizer a superficie de saida nao e testavel: fixe se o resumo vai no artefato ou so na saida do comando.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: SP1 R3.1 (spec)
- last seen: 2026-11-15T15:17:58Z

### L-019 - Feature que promete prioridade precisa definir a ordem esperada na spec, senao nao existe valor a afirmar e o sensor nao mata mutante.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: context-packager-and-subgraph-bundle
- evidence: SP2 G1 (spec)
- last seen: 2026-11-15T15:17:59Z

### L-020 - Um branch dedicado so esta coberto quando a assercao fixa o que ele produz de diferente do caminho generico: se o algoritmo geral devolve o mesmo resultado, a mutacao do branch sobrevive.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: graph-path-discovery
- evidence: internal/graph/path.go:166 (mutant M7) (internal/graph)
- last seen: 2026-11-15T15:24:51Z

### L-021 - Cobertura de um metodo que filtra por repositorio exige dois repositorios no teste: um caminho para o proprio no nao exercita o filtro.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/store` · harmful: 0
- features: graph-path-discovery
- evidence: internal/store/postgres_test.go:272 (AC G2) (internal/store)
- last seen: 2026-11-15T15:24:51Z

### L-022 - Pino no criterio o campo de saida que o usuario precisa (status epistematico, peso da aresta), nao apenas que a aresta existe: sem o valor nao existe valor a afirmar.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: graph-path-discovery
- evidence: internal/graph/path_test.go:103 (AC P1.1) (spec)
- last seen: 2026-11-15T15:24:52Z

### L-023 - Resolucao de nome ambiguo precisa de desambiguacao explicita: um LIMIT 1 sem ORDER BY escolhe uma nota qualquer entre as homonimas e aponta o usuario para o arquivo errado.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/store` · harmful: 0
- features: graph-path-discovery
- evidence: internal/store/postgres.go:1030 (AC P1.1) (internal/store)
- last seen: 2026-11-15T15:24:52Z

### L-024 - Quando a feature expoe uma flag e a sua inversa, a spec precisa dizer qual vence quando as duas aparecem.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `spec` · harmful: 0
- features: graph-path-discovery
- evidence: spec.md:30 (S3) (spec)
- last seen: 2026-11-15T15:24:52Z

### L-025 - Helper de default chamado so por teste esta morto em producao: quando os defaults sao re-declarados como literais em outro lugar, helper e chamadores divergem.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: graph-path-discovery
- evidence: internal/graph/path.go:26 (AC G1) (internal/graph)
- last seen: 2026-11-15T15:24:53Z

### L-026 - Resolva o no do grafo para a identidade que as arestas usam: indexar com um id e percorrer com outro transforma toda rota em nao encontrado.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `internal/graph` · harmful: 0
- features: graph-path-discovery
- evidence: internal/store/postgres.go:1018 (AC P1.1) (internal/graph)
- last seen: 2026-11-15T15:25:02Z

### L-027 - Assert the negative side of a CLI filter: prove the excluded target is absent (no file written and client not listed), not only that the selected one was written.
- signal: `surviving_mutant` · recurrence: 1 feature(s) · scope: `cli` · harmful: 0
- features: ai-tool-auto-wiring
- evidence: M7,M8 / cmd/mem/install.go:85 (cli)
- last seen: 2026-11-15T15:45:25Z

### L-028 - When a later feature repurposes a CLI verb, update in the same commit every spec, ADR and guide that advertises the old meaning of that verb.
- signal: `ac_gap` · recurrence: 1 feature(s) · scope: `cli/docs` · harmful: 0
- features: ai-tool-auto-wiring
- evidence: spec.md:16 / cmd/mem/main.go:794 (cli/docs)
- last seen: 2026-11-15T15:45:25Z

### L-029 - State in the spec the exact observable output a command must produce (which fields, which values), so the assertion targets that output instead of a proxy badge.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `autowire` · harmful: 0
- features: ai-tool-auto-wiring
- evidence: internal/autowire/installer.go:91 (autowire)
- last seen: 2026-11-15T15:45:25Z

### L-030 - Write each acceptance criterion as a single physical line under a header the gate recognises, because the collector judges only the first physical line and opens the block only on the exact header form.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `skill/spec` · harmful: 0
- features: fix-ears-gate-blindness
- evidence: validate_spec.py:163 (skill/spec)
- last seen: 2026-11-16T17:27:17Z

### L-031 - Measure a repo-wide count with the script that ships in the repo, not with a throwaway scratch script, because the two disagreed on the number of affected specs (19 vs 42) while agreeing on the totals.
- signal: `spec_precision_gap` · recurrence: 1 feature(s) · scope: `governanca` · harmful: 0
- features: fix-ears-gate-blindness
- evidence: validation.md:226 (governanca)
- last seen: 2026-11-16T17:27:18Z

## Quarantined (failed when applied - ignore)

A confirmed lesson that recurred alongside failure. Kept for the maintainer to review.

_none_
