# Feature: graph-hygiene-and-doctor

## Problem Statement
O My-Memory implementa cache incremental baseado em hash SHA-256 para reindexação rápida de notas Markdown existentes. No entanto, quando um arquivo `.md` é deletado do disco, a reindexação não detecta sua ausência, acumulando registros órfãos nas tabelas de documentos, chunks, índices vetoriais (`sqlite-vec`, `pgvector`, `TurboQuant`) e arestas do grafo. Além disso, os usuários e agentes de IA não dispõem de ferramentas para auditar a integridade estrutural do grafo de conhecimento, como detecção de *dead links* (wikilinks apontando para notas inexistentes), notas isoladas sem conexões (*orphan notes*), *self-loops* e consistência de vetores.

## Goals
- [ ] Implementar detecção e pruning em cascata de arquivos deletados durante a indexação (`mem index`) no SQLite e PostgreSQL.
- [ ] Fornecer a flag `--no-prune` na CLI para permitir indexação parcial de subpastas sem expurgo.
- [ ] Implementar motor de diagnóstico de integridade de grafo (`DoctorReport`) no contrato `Store`.
- [ ] Disponibilizar o comando de terminal `mem doctor [--fix]` com dashboard ASCII e métrica de saúde (*Health Score*).
- [ ] Expor a ferramenta MCP `memory_doctor` para permitir auditoria de memória por agentes de IA.
- [ ] Documentar a estratégia de ciclo de vida e diagnóstico na ADR-013.

## Out of Scope
- Criação automática de arquivos de notas para resolver dead links (o agente ou usuário decide o conteúdo).
- Limpeza destrutiva de histórico Git.

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Comportamento Padrão de Pruning | Ativado por padrão durante `mem index` | Garante que o banco seja uma projeção fiel do filesystem | y |
| Escopo de Pruning | Documentos pertencentes à raiz indexada ou repositório | Evita remoção acidental de notas indexadas de outros caminhos | y |
| Flag de Preservação | `--no-prune` | Permite indexação seletiva de subárvores sem apagar outras notas | y |
| Ação de Correção (`--fix`) | Remove arestas de self-loops e links quebrados para notas inexistentes | Permite cura automática de inconsistências transitórias do grafo | y |
| Health Score | Escala 0 a 100 com penalidade por dead links, órfãos e chunks dessincronizados | Fornece métrica sintética e imediata da qualidade da base | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Pruning de Arquivos Deletados no Indexador ⭐ MVP

**User Story**: As a desenvolvedor, I want que notas apagadas do disco sejam automaticamente removidas do banco de conhecimento so that o grafo e a busca não retornem dados obsoletos.

**Why P1**: Mantém a integridade referencial entre o filesystem e a base de memória vetorial e relacional.

**Acceptance Criteria**:
1. WHEN `mem index` executes without `--no-prune` THEN the system SHALL identify documents stored in the database whose corresponding files no longer exist on disk.
2. WHEN deleted documents are identified THEN the system SHALL cascade delete their entries from `documents`, `chunks`, `chunks_fts`, `chunks_vec`, `chunks_turboquant`, and `graph_edges`.
3. WHERE `--no-prune` flag is supplied THEN the system SHALL preserve existing database records even if missing from the current scan.
4. The system SHALL report the count and titles of pruned documents in the indexation summary.

**Independent Test**: Indexar um diretório de teste, remover um arquivo, executar `mem index` novamente e asserir que o documento e seus chunks foram expurgados do banco.

---

### P2: Motor de Diagnóstico e Linter de Grafo no Store

**User Story**: As a engenheiro ou agente de IA, I want auditar a integridade topológica do grafo de conhecimento so that eu possa identificar links quebrados, notas isoladas e anomalias.

**Why P2**: Garante qualidade semântica e evita que agentes de IA naveguem em nós fantasmas ou fiquem presos em ilhas isoladas.

**Acceptance Criteria**:
1. The system SHALL provide `DiagnoseHealth` method on `Store` returning a `DoctorReport` struct with metrics and issue lists.
2. WHEN `DiagnoseHealth` executes THEN the system SHALL detect dead links where `target_id` does not match any document ID or title and is not a tag.
3. WHEN `DiagnoseHealth` executes THEN the system SHALL detect orphan notes with zero in-degree and zero out-degree.
4. WHEN `DiagnoseHealth` executes THEN the system SHALL calculate a `HealthScore` between 0 and 100 based on graph integrity.
5. WHERE `FixHealthIssues` is invoked THEN the system SHALL purge dead links and self-loops from `graph_edges`.

**Independent Test**: Testes unitários com grafo contendo dead link, nota órfã e self-loop asserindo detecção precisa e correção via `FixHealthIssues`.

---

### P3: Interface CLI mem doctor

**User Story**: As a usuário no terminal, I want executar `mem doctor` so that eu possa visualizar o relatório de saúde do repositório em formato de dashboard legível.

**Why P3**: Fornece feedback visual e interativo imediato para manutenção do conhecimento.

**Acceptance Criteria**:
1. WHEN `mem doctor` is executed THEN the system SHALL display an ASCII summary table showing document counts, edge counts, health score, dead links, and orphan notes.
2. WHERE `--fix` flag is passed to `mem doctor` THEN the system SHALL execute `FixHealthIssues` and report the number of repaired issues.
3. The system SHALL support both SQLite and PostgreSQL backends via `--db` and `--postgres` flags.

**Independent Test**: Execução de `mem doctor` no terminal sobre base de teste validando saída tabular e código de saída 0.

---

### P4: Ferramenta MCP memory_doctor

**User Story**: As a agente de IA conectado via MCP, I want chamar a ferramenta `memory_doctor` so that eu possa avaliar a qualidade do repositório e sugerir reparos estruturais.

**Why P4**: Permite que modelos como Claude e Cursor atuem na curadoria autônoma da base de notas.

**Acceptance Criteria**:
1. The system SHALL expose `memory_doctor` tool in the MCP catalog with optional `repository` and `fix` parameters.
2. WHEN `memory_doctor` is invoked THEN the system SHALL return a formatted Markdown diagnostic report containing health score and detected issues.

**Independent Test**: Chamada JSON-RPC `tools/call` com `name: "memory_doctor"` validando resposta estruturada.

---

## Edge Cases
- IF all documents are fully interconnected with no dead links THEN the system SHALL report a HealthScore of 100 with zero issues.
- IF an empty database is diagnosed THEN the system SHALL return a report with zero counts without panicking.
- IF a wikilink points to an alias or heading (`[[Nota#Seção]]`) THEN the system SHALL resolve against the base note name to avoid false dead-link warnings.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| PRUNE-01 | P1: Pruning de Arquivos Deletados no Indexador | Tasks | Pending |
| PRUNE-02 | P1: Pruning de Arquivos Deletados no Indexador | Tasks | Pending |
| DOCTOR-01 | P2: Motor de Diagnóstico e Linter de Grafo no Store | Tasks | Pending |
| DOCTOR-02 | P2: Motor de Diagnóstico e Linter de Grafo no Store | Tasks | Pending |
| DOCTOR-03 | P3: Interface CLI mem doctor | Tasks | Pending |
| DOCTOR-04 | P4: Ferramenta MCP memory_doctor | Tasks | Pending |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] `PruneDeletedDocuments` implementado e validado em SQLite e PostgreSQL.
- [ ] Flag `--no-prune` funcional no comando `mem index`.
- [ ] `DiagnoseHealth` e `FixHealthIssues` implementados e cobertos por testes unitários.
- [ ] Comando CLI `mem doctor [--fix]` operacional com dashboard ASCII.
- [ ] Ferramenta MCP `memory_doctor` disponível e testada via JSON-RPC.
- [ ] Registro de decisão ADR-013 documentado.
