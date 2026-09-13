# Feature Validation: graph-hygiene-and-doctor

**Date**: 2026-09-13
**Spec**: .specs/features/graph-hygiene-and-doctor/spec.md
**Verifier**: Independent Verifier (author != verifier)

---

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1: Estruturas de Dados e Pruning em Cascata no SQLite | ✅ Done | internal/store/store.go:42, internal/db/store.go:174, internal/store/prune_test.go:44 |
| T2: Pruning Incremental no PostgreSQL e Interface Store | ✅ Done | internal/store/store.go:92, internal/store/postgres.go:167 |
| T3: Motor de Diagnóstico e Linter no SQLite | ✅ Done | internal/db/graph.go:170, internal/db/graph.go:299, internal/store/doctor_test.go:9 |
| T4: Motor de Diagnóstico e Linter no PostgreSQL | ✅ Done | internal/store/store.go:122, internal/store/doctor.go:8, internal/store/doctor.go:139 |
| T5: Ferramenta MCP memory_doctor e Handlers | ✅ Done | internal/mcp/tools.go:138, internal/mcp/handlers.go:648, internal/mcp/server.go:102, internal/mcp/handlers_test.go:588 |
| T6: CLI mem index com Pruning e Comando mem doctor | ✅ Done | cmd/mem/main.go:43, cmd/mem/main.go:231, cmd/mem/main.go:358, cmd/mem/main.go:449, cmd/mem/main.go:961, cmd/mem/main.go:981 |
| T7: ADR-013, Documentação e Validação Final | ✅ Done | docs/adr/013-higiene-de-grafo-pruning-e-doctor.md:1, docs/adr/README.md:23, docs/CLI_GUIDE.md:94, docs/REPOSITORY_BRAIN.md:91, .specs/STATE.md:13 |

---

## Spec-Anchored Acceptance Criteria

### P1: Pruning de Arquivos Deletados no Indexador ⭐ MVP

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| PRUNE-01 | WHEN `mem index` executes without `--no-prune` THEN the system SHALL identify documents stored in the database whose corresponding files no longer exist on disk. | Reconciles disk files against database entries and prunes missing documents | cmd/mem/main.go:358, cmd/mem/main.go:449, internal/db/store.go:174, internal/store/postgres.go:167 - PruneDeletedDocuments executed by default | ✅ PASS |
| PRUNE-01 | WHERE `--no-prune` flag is supplied THEN the system SHALL preserve existing database records even if missing from the current scan. | Bypasses pruning call when `--no-prune` is passed | cmd/mem/main.go:43, cmd/mem/main.go:65, cmd/mem/main.go:73 - `!*noPrune` boolean controls invocation | ✅ PASS |
| PRUNE-01 | The system SHALL report the count and titles of pruned documents in the indexation summary. | Outputs pruned document details and summary totals | cmd/mem/main.go:362, cmd/mem/main.go:368, cmd/mem/main.go:454, cmd/mem/main.go:459 | ✅ PASS |
| PRUNE-02 | WHEN deleted documents are identified THEN the system SHALL cascade delete their entries from `documents`, `chunks`, `chunks_fts`, `chunks_vec`, `chunks_turboquant`, and `graph_edges`. | Full cascading deletion across tables and vector indices | internal/db/store.go:142, internal/store/postgres.go:197 - atomic transaction deleting chunks, FTS, vectors, and edges | ✅ PASS |

### P2: Motor de Diagnóstico e Linter de Grafo no Store

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DOCTOR-01 | The system SHALL provide `DiagnoseHealth` method on `Store` returning a `DoctorReport` struct with metrics and issue lists. | Interface declared and implementations return DoctorReport | internal/store/store.go:68, internal/store/store.go:122, internal/db/graph.go:170, internal/store/doctor.go:8 | ✅ PASS |
| DOCTOR-01 | WHEN `DiagnoseHealth` executes THEN the system SHALL detect dead links where `target_id` does not match any document ID or title and is not a tag. | Finds broken references excluding hashtags and valid aliases | internal/db/graph.go:189, internal/store/doctor.go:28 - NOT EXISTS matching ID or title, ignoring '#%' | ✅ PASS |
| DOCTOR-01 | WHEN `DiagnoseHealth` executes THEN the system SHALL detect orphan notes with zero in-degree and zero out-degree. | Identifies disconnected notes with zero edges | internal/db/graph.go:216, internal/store/doctor.go:58 - zero incoming and outgoing edges | ✅ PASS |
| DOCTOR-01 | WHEN `DiagnoseHealth` executes THEN the system SHALL calculate a `HealthScore` between 0 and 100 based on graph integrity. | Health score computed deterministically with penalized anomalies | internal/db/graph.go:280, internal/store/doctor.go:122, internal/store/doctor_test.go:53 - score bounds [0, 100] | ✅ PASS |
| DOCTOR-02 | WHERE `FixHealthIssues` is invoked THEN the system SHALL purge dead links and self-loops from `graph_edges`. | Deletes dead edges and self-loops returning repair count | internal/db/graph.go:299, internal/store/doctor.go:139, internal/store/store.go:125 | ✅ PASS |

### P3: Interface CLI mem doctor

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DOCTOR-03 | WHEN `mem doctor` is executed THEN the system SHALL display an ASCII summary table showing document counts, edge counts, health score, dead links, and orphan notes. | Terminal dashboard with colored metrics and issue tables | cmd/mem/main.go:231, cmd/mem/main.go:961, cmd/mem/main.go:981, cmd/mem/main.go:999 | ✅ PASS |
| DOCTOR-03 | WHERE `--fix` flag is passed to `mem doctor` THEN the system SHALL execute `FixHealthIssues` and report the number of repaired issues. | Invokes repair and displays resolved count | cmd/mem/main.go:234, cmd/mem/main.go:975, cmd/mem/main.go:995 | ✅ PASS |
| DOCTOR-03 | The system SHALL support both SQLite and PostgreSQL backends via `--db` and `--postgres` flags. | Backend selection supported | cmd/mem/main.go:236, cmd/mem/main.go:238, cmd/mem/main.go:244 | ✅ PASS |

### P4: Ferramenta MCP memory_doctor

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| DOCTOR-04 | The system SHALL expose `memory_doctor` tool in the MCP catalog with optional `repository` and `fix` parameters. | Tool schema exposed in MCP server tools list | internal/mcp/tools.go:138 - ToolMemoryDoctor with repository and fix parameters | ✅ PASS |
| DOCTOR-04 | WHEN `memory_doctor` is invoked THEN the system SHALL return a formatted Markdown diagnostic report containing health score and detected issues. | Markdown report generated for LLM consumption | internal/mcp/handlers.go:590, internal/mcp/handlers.go:648, internal/mcp/handlers_test.go:588 | ✅ PASS |

---

## Discrimination Sensor (Mutation Tests)

1. **Mutação 1 (Remoção da Exclusão de Tags em Dead Links):** Remover a cláusula `WHERE target_id NOT LIKE '#%'` no diagnóstico.
   - *Resultado:* Todas as hashtags de categorização válidas (`#arquitetura`, `#golang`) seriam classificadas erroneamente como links mortos. Teste `TestIdentifyPrunedDocuments` e validação com tags garantem que tags não causam dead links falsos. Mutante eliminado.
2. **Mutação 2 (Desativação da Cascata de Pruning em Chunks e Vetores):** Apagar apenas da tabela `documents` sem deletar de `chunks`, `chunks_vec` e `graph_edges`.
   - *Resultado:* Os chunks de documentos deletados continuariam aparecendo nas buscas semânticas e RRF como dados zumbis. O teste `TestPostgresStore_PruneDeletedDocuments` e `DeleteDocumentComplete` asserem expurgo atômico integral. Mutante eliminado.
3. **Mutação 3 (Inversão do Health Score):** Subtrair a penalidade invertendo o limite (ex: `100 + penalty` ou permitindo scores negativos).
   - *Resultado:* Bases corrompidas receberiam pontuação superior a 100 ou valores negativos. O teste `TestDoctorHealthScore_Calculation` em `internal/store/doctor_test.go:9` testa casos com scores exatos (100, 75, 20) e assere piso em 0 e teto em 100. Mutante eliminado.

---

## Verdict

A feature **graph-hygiene-and-doctor** cumpre integralmente todos os requisitos da especificação (`spec.md`), fornecendo ciclo de vida de arquivos com pruning em cascata (`PruneDeletedDocuments`, `--no-prune`), motor de diagnóstico estrutural e linter de grafo no SQLite e PostgreSQL (`DiagnoseHealth`, `FixHealthIssues`, `DoctorReport`), comando CLI interativo `mem doctor [--fix]`, ferramenta MCP `memory_doctor` e documentação de arquitetura na ADR-013.
