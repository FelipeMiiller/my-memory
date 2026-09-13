# Feature Validation: temporal-decay-and-recency-search

**Date**: 2026-09-13
**Spec**: .specs/features/temporal-decay-and-recency-search/spec.md
**Verifier**: Independent Verifier (author != verifier)

---

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1: Motor Matemático de Decaimento Temporal e Fusão RRF Ponderada | ✅ Done | internal/store/rrf.go:88, internal/store/decay_test.go:27, internal/store/decay_test.go:88 |
| T2: Enriquecimento de SearchResult com UpdatedAt e Queries SQLite | ✅ Done | internal/db/graph.go:21, internal/db/hybrid.go:38, internal/db/hybrid_test.go:13 |
| T3: Suporte a Decaimento Temporal no PostgreSQL e Interface Store | ✅ Done | internal/store/store.go:133, internal/store/postgres.go:574, internal/store/postgres_test.go:44 |
| T4: Parâmetros de Decaimento na Ferramenta MCP memory_search | ✅ Done | internal/mcp/tools.go:42, internal/mcp/handlers.go:260, internal/mcp/handlers_test.go:760 |
| T5: Flags de Decaimento no Comando CLI mem search | ✅ Done | cmd/mem/main.go:82, cmd/mem/main.go:483, cmd/mem/main.go:545 |
| T6: ADR-015, Validação Final e Documentação | ✅ Done | docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md:1, docs/adr/README.md:25, docs/CLI_GUIDE.md:38, docs/REPOSITORY_BRAIN.md:86, README.md:107 |

---

## Spec-Anchored Acceptance Criteria

### P1: Motor Matemático de Decaimento Temporal e RRF ⭐ MVP

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DECAY-01 | The system SHALL provide `CalculateTimeDecay` calculating the exponential multiplier $(1-w) + w \cdot 2^{-\Delta t / T_{half}}$ for a given delta time, half-life, and weight. | Returns decay factor between $(1-w)$ and $1.0$ | internal/store/rrf.go:110 - CalculateTimeDecay formula with asymptotic floor | ✅ PASS |
| DECAY-01 | WHEN `updated_at` equals the reference time ($\Delta t = 0$) THEN the system SHALL return a decay multiplier of exactly $1.0$. | Multiplier is exactly 1.0 for deltaT=0 | internal/store/rrf.go:117, internal/store/decay_test.go:44 - deltaT=0 returns 1.0 | ✅ PASS |
| DECAY-01 | WHEN $\Delta t$ equals the half-life duration THEN the system SHALL return a decay multiplier of exactly $1.0 - 0.5w$. | Multiplier is $1 - 0.5w$ for 1 half-life | internal/store/decay_test.go:57 - mHalf equals 0.85 for w=0.3 | ✅ PASS |
| DECAY-02 | WHEN `FuseSearchResultsWithDecay` executes with `DecayOptions.Enabled = true` THEN the system SHALL multiply each fused item's score by its temporal multiplier. | Scores weighted by recency decay | internal/store/rrf.go:189, internal/store/decay_test.go:134 - chunk_new surpasses chunk_old | ✅ PASS |
| DECAY-02 | IF `DecayOptions.Enabled` is false THEN the system SHALL preserve identical RRF scores to standard `FuseSearchResults`. | Default RRF unweighted fusion preserved | internal/store/rrf.go:209, internal/store/decay_test.go:121 - unweighted ranking preserved | ✅ PASS |

### P2: Enriquecimento de Resultados com UpdatedAt no Store

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DECAY-03 | The system SHALL include `UpdatedAt` field in `SearchResult` struct in `internal/store`, `internal/db`, and `internal/mcp`. | `UpdatedAt int64` present across all SearchResult types | internal/store/store.go:20, internal/db/graph.go:21, internal/mcp/handlers.go:52 | ✅ PASS |
| DECAY-03 | WHEN `SearchFTS` or `SearchKNN` executes in SQLite THEN the system SHALL populate `UpdatedAt` from the corresponding document's `updated_at` column. | `UpdatedAt` extracted from documents table via LEFT JOIN | internal/db/hybrid.go:21, internal/db/graph.go:32, internal/db/hybrid_test.go:25 | ✅ PASS |
| DECAY-03 | WHEN `SearchFTS` or `SearchKNN` executes in PostgreSQL THEN the system SHALL populate `UpdatedAt` from `documents.updated_at`. | `COALESCE(d.updated_at, 0)` mapped to UpdatedAt | internal/store/postgres.go:464, internal/store/postgres.go:537, internal/store/postgres_test.go:44 | ✅ PASS |
| DECAY-04 | The system SHALL declare and implement `SearchHybridRRFWithDecay` on the `Store` interface. | Method present on Store interface and implemented by PostgresStore | internal/store/store.go:133, internal/store/postgres.go:574 | ✅ PASS |

### P3: Protocolo MCP memory_search com Suporte a Decaimento

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DECAY-05 | The system SHALL support optional `decay`, `half_life`, and `decay_weight` properties in `memory_search` MCP tool schema. | Schema exposes decay parameters | internal/mcp/tools.go:42, internal/mcp/handlers_test.go:760 - schema properties validated | ✅ PASS |
| DECAY-05 | IF `decay` is omitted or false in `memory_search` THEN the system SHALL execute standard hybrid search without time decay. | Defaults to standard hybrid search | internal/mcp/handlers.go:260, internal/mcp/handlers_test.go:88 | ✅ PASS |
| DECAY-05 | WHEN `decay` is true in `memory_search` THEN the system SHALL forward `DecayOptions` to `SearchHybridRRFWithDecay`. | DecayOptions dispatched and result formatted | internal/mcp/handlers.go:297, internal/mcp/handlers_test.go:776 | ✅ PASS |

### P4: Interface CLI mem search com Flags de Recência

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DECAY-06 | WHEN `mem search` is invoked with `--decay` THEN the system SHALL execute hybrid search applying time decay. | `--decay` flag triggers SearchHybridRRFWithDecay | cmd/mem/main.go:82, cmd/mem/main.go:103, cmd/mem/main.go:506, cmd/mem/main.go:573 | ✅ PASS |
| DECAY-06 | WHERE `--half-life` and `--decay-weight` flags are passed THEN the system SHALL configure the corresponding decay hyperparameters. | CLI flags configure DecayOptions | cmd/mem/main.go:83, cmd/mem/main.go:84, cmd/mem/main.go:94 | ✅ PASS |

---

## Test Execution Summary

- **Store Tests**: `go test -v ./internal/store/...` (100% PASS, including `TestDefaultDecayOptions`, `TestTimeDecay_MathematicalCorrectness`, `TestTimeDecay_RankingInversion`, `TestPostgresStore_SearchHybridRRFWithDecay_EmptyInputs`)
- **DB Tests**: `go test -v ./internal/db/...` (100% PASS, including `TestConversion_StoreToDb_UpdatedAt`)
- **MCP Tests**: `go test -v ./internal/mcp/...` (100% PASS, including `TestToolMemorySearch_SchemaDecayProperties`, `TestServer_ToolsCall_MemorySearch_WithDecay`)
- **Full Suite**: `go test ./...` (100% PASS across all packages)
- **CLI Compilation**: `go build ./cmd/mem` (100% PASS, help output verified)
