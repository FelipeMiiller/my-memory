# prune-dead-code-2026-11 Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/prune-dead-code-2026-11/spec.md`
**Diff range**: `15ebe9c..HEAD`
**Verifier**: execução **inline** (2 tarefas — sem sub-agente, conforme a regra do lote único)

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 — os 14 cortes | ✅ Done | 14 blocos removidos, **223 linhas**; 3 importações órfãs limpas |
| T2 — a prova e o fechamento | ✅ Done | `deadcode` 17 → **3**; gates full e taggeado verdes; cofre 100/100; ISSUE-042 + STATE.md |

**Cobertura**: 2 tarefas, 2 concluídas, 0 pendentes.

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| **AC1** — as 14 funções sumiram do código | 14 blocos removidos, nada mais alterado | `internal/codeast/parser.go` (`ValidKinds`, `IsCamelOrPascalSegment`) · `internal/db/graph.go` (`SearchTurboQuant`) · `internal/embedder/model_downloader.go` (`DownloadProviderArtifacts`) · `internal/event_runtime/audit_subscriber.go` (`auditLogPathForTesting`) · `internal/event_runtime/dispatcher.go` (`IsReady`) · `internal/event_runtime/log.go` (`ReadRange`) · `internal/federation/search.go` (`ReplaceDatabaseInPostgresURL`) · `internal/graph/inspector.go` (`DefaultInspectorOptions`) · `internal/mcp/handlers.go` (`NewDBNeighborsHandler`) · `internal/mcp/server.go` (`StalenessDetector`) · `internal/supervisor/audit.go` (`CurrentSize`, `CurrentPath`) · `internal/vaultfs/target.go` (`Root`) — **diff: 12 arquivos, 223 remoções, 0 adições** | ✅ PASS |
| **AC2** — gate completo verde, comportamento idêntico | nenhum teste removido ou alterado | `go test -count=1 ./...` → **27/27 pacotes `ok`**, exit 0 | ✅ PASS |
| **AC3** — gate taggeado verde | a build do release passa | `go test -tags "sqlite_fts5 onnx" -count=1 ./...` → **27/27 pacotes `ok`**, exit 0 | ✅ PASS |
| **AC4** — o `deadcode` reporta só as 3 do `Subscriber` | **3** (as que o `deadcode` não vê) | `deadcode -test -tags "sqlite_fts5 onnx" ./...` → `AuditSubscriber.Name`, `.EventTypes`, `.MaxAckPending` (chamadas via a interface `Subscriber`) — **era 17** | ✅ PASS |
| **AC5** — cada corte registrado com ID | 14 linhas na ISSUE-042, com `arquivo:linha` e a prova | `.specs/ISSUES.md` — tabela "Os 14 cortes (cada um com a prova)" | ✅ PASS |
| **AC6** — o cofre continua 100/100 | 100/100, 0 link rompido, 0 chunk sem vetor | `mem doctor --storage=postgres --repo repo_central` → **100/100**, 0 rompido, 0 órfã, 0 chunk sem vetor | ✅ PASS |
| **AC7** — a ISSUE-042 lista arquivo:linha e a prova | tabela por função | `.specs/ISSUES.md` — as 14 linhas + a tabela "O que **não** foi podado (e por quê)" | ✅ PASS |
| **AC8** — o STATE registra a fase 2 e o que ficou | entrada no handoff + decisão do AD-065 | `.specs/STATE.md` — handoff da fase 2 e o AD-065 com a fase 2 concluída | ✅ PASS |

**Status**: ✅ 8/8 ACs cobertas com evidência, 0 gap de precisão.

---

## Discrimination Sensor

O sensor de uma poda é **o próprio teste de que a poda não quebrou nada**: cada
mutação tem que ser **morta** pelo gate (a suíte tem que passar), e o que não
passa seria uma função que ainda era usada.

| Mutação | Verificação | Resultado |
| ------- | ----------- | --------- |
| M1 — remover `Log.ReadRange` (66 linhas) | `mem events replay` continua funcionando: a CLI usa a **própria** query (`cmd/mem/events.go:291`), nunca o método removido | ✅ Killed — o comando `replay` tem teste (`cmd/mem/events_test.go:191`, `:214`) e segue verde |
| M2 — remover `NewDBNeighborsHandler` | a tool `memory_get_neighbors` continua: o servidor usa `NewMemoryNeighborsHandler` (`internal/mcp/handlers.go:664`) | ✅ Killed — `TestServer_ToolsCall_MemoryGetNeighbors_Success` (`internal/mcp/handlers_test.go:152`) verde |
| M3 — remover `SearchTurboQuant` | a busca vetorial do SQLite continua: o caminho vivo é `SearchTurboQuantWithOptions` (`internal/db/graph.go:125`), com os testes de busca do `internal/db` verdes | ✅ Killed — suíte `internal/db` verde |
| M4 — remover `Resolver.Root` (meu, da spec 3) | o doctor continua classificando: a raiz vem do `vaultfs.Resolver`, não do accessor | ✅ Killed — suíte `internal/vaultfs` e o doctor do cofre verdes |

**Sensor depth**: lightweight (4 mutações, uma por risco distinto: comando
existente, tool MCP existente, caminho de busca existente, API nova).
**Resultado**: **4/4 mortas** — PASS ✅

O sensor mais amplo é o próprio `deadcode`: depois do corte ele reporta **3**
(funções vivas, por dispatch por interface) — o número esperado pela spec.

---

## Gate Check

- **Quick**: `go build ./...` → exit 0 · `go vet ./...` → exit 0
- **Full**: `go test -count=1 ./...` (com Postgres) → **27/27 pacotes `ok`**, exit 0
- **Tagged**: `go test -tags "sqlite_fts5 onnx" -count=1 ./...` → **27/27 pacotes `ok`**, exit 0
- **Format**: `gofmt -l` limpo nos 12 arquivos tocados
- **Medido**: `deadcode -test -tags "sqlite_fts5 onnx" ./...` → **3** (as do `Subscriber`)
- **Medido**: `mem doctor --storage=postgres --repo repo_central` → **100/100**
- **Diff**: 12 arquivos, **223 linhas removidas**, 0 adicionadas (além das 3 importações órfãs que a remoção deixou)
- **Spec**: `validate_spec.py` → 0 erros · `validate_tasks.py` → 0 erros
- **State**: `validate_state.py prune-dead-code-2026-11` → 0 erros

### As 3 importações órfãs

A remoção de código deixou 3 imports sem uso — o compilador accuse e o
`go vet` reprovou até a limpeza: `net/url` (`internal/federation/search.go`),
`time` (`internal/event_runtime/audit_subscriber.go`) e `database/sql`
(`internal/mcp/handlers.go`). Nenhuma mudança de comportamento: só o import
que não tinha mais a quem servir.

---

## Code Quality

| Principle | Status |
| --------- | ------ |
| Minimum code | ✅ — o podar **é** a redução; nada foi reescrito |
| Surgical changes | ✅ — diff de 223 remoções e 0 adições; nenhum comportamento tocado |
| No scope creep | ✅ — as 6 ADRs obsoletas, os 5 relatórios-plano e as 6 specs planejadas **ficaram** (e o motivo está na ISSUE-042) |
| Matches patterns | ✅ — convenção de commits e de registro do repo |
| Spec-anchored outcome check | ✅ — 8/8 ACs com `file:line` |
| Per-layer coverage expectation | ✅ — a cobertura existente de cada pacote continua verde |
| Every test maps to a spec AC | ✅ — nenhum teste novo; o gate é o verificador |
| Documented guidelines followed | ✅ — `AGENTS.md` (regras 1, 4, 7, 8), a ADR-065 (fase 2: "cada corte com o ID rastreável") |

---

## Edge Cases

- [x] Função que implementa interface → **não** removida: as 3 do `Subscriber`
      foram salvas pela prova de chamada (`internal/event_runtime/dispatcher.go:212`,
      `:234`, `:257`).
- [x] Função chamada só na build com a tag `onnx` → **não** removida:
      `NewTokenizerFromDir` e `Tokenizer.Close` (o `deadcode` sem a tag erra).
- [x] Reflexão por nome de método → verificado: o repo **não** usa
      (`reflect.ValueOf` / `MethodByName`: zero ocorrências), então a prova por
      texto é suficiente.
- [x] Atalho de conveniência de uma função viva → `SearchTurboQuant` removido; o
      vivo é `SearchTurboQuantWithOptions`.
- [x] Função com doc que promete um papel que não tem → `Log.ReadRange` dizia ser
      "the replay primitive used by `mem replay --since N`", mas o comando faz a
      própria query (`cmd/mem/events.go:291`). Corte seguro, e o doc enganoso
      foi junto.
- [x] Remoção quebrando um teste → o gate falha; o gate é o critério, e passou.

---

## Fix Plans

Nenhum gap aberto. Durante a execução, 3 importações ficaram órfãs e foram
limpas (o `go vet` reprovou até lá).

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| PDC-01 | Implementing | ✅ Verified (12 arquivos, 223 remoções) |
| PDC-02 | Implementing | ✅ Verified (gate full 27/27) |
| PDC-03 | Implementing | ✅ Verified (gate taggeado 27/27) |
| PDC-04 | Implementing | ✅ Verified (`deadcode` reporta 3) |
| PDC-05 | Implementing | ✅ Verified (ISSUE-042, 14 linhas) |
| PDC-06 | Implementing | ✅ Verified (cofre 100/100) |
| PDC-07 | Implementing | ✅ Verified (ISSUE-042) |
| PDC-08 | Implementing | ✅ Verified (STATE.md) |

**Coverage**: 8 total, 8 verificados, 0 unmapped.

---

## Summary

**Overall**: ✅ Pronto

**Spec-anchored check**: 8/8 ACs com `file:line`
**Sensor**: 4/4 mutações mortas
**Gate**: build e vet exit 0 · full 27/27 · taggeado 27/27 · cofre 100/100 ·
`deadcode` 17 → **3**

**O que foi removido**: 14 funções provadamente sem chamada, **223 linhas**. O
que **ficou** (e por quê) está na ISSUE-042: as 6 ADRs obsoletas (são o
registro das decisões revertidas), os 5 `validation.md` que são plano (dívida
de verificação, não de corte), as 6 specs planejadas (roadmap, decisão do
dono), e as 5 funções que o `deadcode` acusava mas que estão vivas.

**A lição do método**: `deadcode` é indicador nas **duas** pontas — build tag e
dispatch por interface. Sem a tag do release ele marca o provider como morto;
sem a prova de chamada ele marca 3 métodos vivos. A poda só é segura depois
das duas checagens.
