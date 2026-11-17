# fix-graph-node-identity Validation Report

- **Date**: 2026-11-17
- **Spec**: `.specs/features/fix-graph-node-identity/spec.md`
- **Tasks**: `.specs/features/fix-graph-node-identity/tasks.md`
- **Issue**: ISSUE-047 (`.specs/ISSUES.md`) · FIX-2 da validação de `graph-path-discovery`
- **ADR**: ADR-066 (`docs/adr/066-a-identidade-canonica-do-no-no-grafo.md`)
- **Verifier**: Independent Verifier (sub-agente independente: author != verifier, evidence-or-zero)
- **Result**: PASS
- **Verdict**: PASS

---

## Validation: PASS

**Verdict**: **PASS** — Todos os 18 critérios de aceitação (GNI-01 a GNI-18) verificados com citações `file:line` e asserções discriminantes. Sensor de Discriminação executado com 3/3 mutantes eliminados (0 sobreviventes) e árvore revertida 100% limpa. Validação no PostgreSQL real (`repo_central` e `repo_e4c8f3b1a2d5`) confirmando paridade exata de arestas (545 == 545, 0 divididas), 0 nós legados sem grau e rota de 2 saltos restaurada. O motor de pathfinding (`internal/graph/path.go`) permaneceu 100% intocado. Gate Go completo verde (27/27 pacotes PASS).

---

## Task Completion (T1 a T12)

| Task | Status | Description & Evidence |
| :--- | :--- | :--- |
| **T1** | ✅ Done | ADR-066 registrada em `docs/adr/066-a-identidade-canonica-do-no-no-grafo.md:1-176` e indexada em `docs/adr/README.md:69`. Decisão arquitetural de unificação da identidade do nó no `documents.id`. |
| **T2** | ✅ Done | Gravação em lote com destino canônico (`documents.id`) no Postgres (`cmd/mem/main.go:2334`) e SQLite (`cmd/mem/main.go:2352`). Testado em `cmd/mem/index_identity_test.go:45-122`. |
| **T3** | ✅ Done | Gravação cirúrgica do watcher com destino canônico no SQLite (`internal/watcher/indexer.go:124`) e Postgres (`internal/watcher/indexer.go:275`). Testado em `internal/watcher/index_identity_test.go:38-120`. |
| **T4** | ✅ Done | Nó do documento no SQLite carrega título como rótulo (`internal/db/store.go:156-163`). Testado em `internal/db/graph_identity_test.go:215-280`. |
| **T5** | ✅ Done | Nó do documento no Postgres carrega título como rótulo (`internal/store/postgres.go:348-360`). Testado em `internal/store/graph_identity_test.go:35-80`. |
| **T6** | ✅ Done | Resolvedor canônico SQLite unifica path, título e alias (`internal/db/graph.go:670-760`). Testado em `internal/db/graph_identity_test.go:21-120`. |
| **T7** | ✅ Done | Resolvedor canônico Postgres unifica path, título e alias (`internal/store/postgres.go:1060-1150`). Testado em `internal/store/graph_identity_test.go:85-145`. |
| **T8** | ✅ Done | Doctor SQLite conta e pruneia nós legados sem grau (`internal/db/graph.go:470-490`, `:620-632`). Testado em `internal/db/graph_identity_test.go:130-205`. |
| **T9** | ✅ Done | Doctor Postgres conta e pruneia nós legados sem grau (`internal/store/doctor.go:160-180`, `:210-230`). Testado em `internal/store/doctor_identity_test.go:30-85`. |
| **T10** | ✅ Done | Teste E2E ponta a ponta provando 1 salto no `mem path` e coerência de consumidores (`cmd/mem/path_identity_test.go:40-185`). |
| **T11** | ✅ Done | Procedimento de reparo e queries de medição documentados em `docs/CLI_GUIDE.md:464-530`. |
| **T12** | ✅ Done | Reparo executado no cofre central e medição real comprovada via SQL e CLI no PostgreSQL (`repo_central` e `repo_e4c8f3b1a2d5`). |

---

## Acceptance Criteria Traceability Matrix (GNI-01 a GNI-18)

| Req ID | Acceptance Criterion | Implementation Location | Verification Evidence & Test Assertion | Result |
| :--- | :--- | :--- | :--- | :--- |
| **GNI-01** | AC1 — Destino resolvido vira o `documents.id` do documento | `cmd/mem/main.go:2334`, `cmd/mem/main.go:2352`, `internal/watcher/indexer.go:124`, `internal/watcher/indexer.go:275` | `cmd/mem/index_identity_test.go:66-75` (`target != docB`), `internal/watcher/index_identity_test.go:73-81`. Mutante M1 eliminado. | ✅ PASS |
| **GNI-02** | AC2 — Origem gravada é o `documents.id` do documento | `cmd/mem/main.go:2208`, `cmd/mem/main.go:1960`, `internal/watcher/indexer.go:119`, `internal/watcher/indexer.go:267` | `cmd/mem/index_identity_test.go:99-112` (`origensOrfas != 0`), `internal/watcher/index_identity_test.go:96-105`. | ✅ PASS |
| **GNI-03** | AC3 — Alvo não resolvido mantém o texto bruto do wikilink | `cmd/mem/main.go:2334`, `cmd/mem/main.go:2352`, `internal/store/alias_index.go:50-58` | `cmd/mem/index_identity_test.go:88-97` (`dangling != "Alvo Inexistente"`), `internal/watcher/index_identity_test.go:83-94`. | ✅ PASS |
| **GNI-04** | AC4 — SQLite e Postgres gravam o mesmo alvo idêntico | `cmd/mem/main.go:2334` (PG), `cmd/mem/main.go:2352` (SQLite) | `cmd/mem/index_identity_test.go:45-122`, `internal/watcher/index_identity_test.go:38-120`. Contrato unificado via `store.ResolveEdgeTarget`. | ✅ PASS |
| **GNI-05** | AC5 — Uma única identidade de nó por documento (`id = documents.id`) | `internal/db/store.go:156-163`, `internal/store/postgres.go:348-360` | `internal/db/graph_identity_test.go:256-267` (`porTitulo != 0`), `internal/store/graph_identity_test.go:55-65`. | ✅ PASS |
| **GNI-06** | AC6 — Nó do documento carrega o `title:` como rótulo (`name`) | `internal/db/store.go:159`, `internal/store/postgres.go:351` | `internal/db/graph_identity_test.go:248-254` (`name != "Minha Nota"`). Mutante M2 eliminado. | ✅ PASS |
| **GNI-07** | AC7 — Path, `title:` e `alias:` devolvem o mesmo `documents.id` | `internal/db/graph.go:670-760`, `internal/store/postgres.go:1060-1150` | `internal/db/graph_identity_test.go:55-80` (`byPath != notaPath`, `byTitle != notaPath`, `byAlias != notaPath`), `internal/store/graph_identity_test.go:85-115`. | ✅ PASS |
| **GNI-08** | AC8 — Sem raiz do vault, alias devolve erro de nó não encontrado | `internal/db/graph.go:740-755`, `internal/store/postgres.go:1130-1145` | `internal/db/graph_identity_test.go:82-95` (`err == nil`), `internal/store/graph_identity_test.go:118-130`. | ✅ PASS |
| **GNI-09** | AC9 — Título resolve para `documents.id`, nunca para nó de estilo título | `internal/db/graph.go:685-715`, `internal/store/postgres.go:1075-1105` | `internal/db/graph_identity_test.go:60-75`, `internal/store/graph_identity_test.go:95-110`. | ✅ PASS |
| **GNI-10** | AC10 — `mem path` acha rota de 1 salto entre notas ligadas | `cmd/mem/path.go:100-180`, `internal/graph/path.go` (intocado) | `cmd/mem/path_identity_test.go:50-70` (`strings.Contains(out, "Saltos:          1")`, `Status:          ✅ Encontrado`). | ✅ PASS |
| **GNI-11** | AC11 — `mem impact`, `mem inspect` e `mem path` devolvem mesmo id | `cmd/mem/impact.go:40-60`, `cmd/mem/inspect.go:50-70`, `cmd/mem/path.go:60-80` | `cmd/mem/path_identity_test.go:95-125` (`byPath != byImpact || byPath != byInspect`). | ✅ PASS |
| **GNI-12** | AC12 — Ponte `references_code` mantém `source_id = documents.id` | `internal/db/code_bridge.go:50-80`, `internal/store/code_bridge.go:40-70` | `cmd/mem/path_identity_test.go:130-150` (`sourceID != docA`). | ✅ PASS |
| **GNI-13** | AC13 — `mem doctor` não acusa aresta de documento como link morto | `internal/db/graph.go:420-460`, `internal/store/doctor.go:110-150` | `internal/db/graph_identity_test.go:174-179`, `cmd/mem/path_identity_test.go:155-175`. | ✅ PASS |
| **GNI-14** | AC14 — Relatório do doctor expõe contagem de nós legados sem grau | `internal/db/graph.go:470-490`, `internal/store/doctor.go:160-180` | `internal/db/graph_identity_test.go:170-173` (`report.OrphanLegacyNodeCount != 1`), `internal/store/doctor_identity_test.go:45-55`. | ✅ PASS |
| **GNI-15** | AC15 — No cofre central reindexado, arestas divididas são 0 | Procedimento executado no PostgreSQL `repo_central` | Query 1 = 545, Query 2 = 545 (Query 1 == Query 2 → 0 divididas). Medição comprovada em `tasks.md:507-511`. | ✅ PASS |
| **GNI-16** | AC16 — No cofre central reparado, nós legados sem grau são 0 | Procedimento `mem doctor --fix` no PostgreSQL `repo_central` | Query 3 = 0 nós legados sem grau. Medição comprovada em `tasks.md:511`. Mutante M3 eliminado. | ✅ PASS |
| **GNI-17** | AC17 — `mem path "CONTRIBUTING.md" "README.md" --undirected` acha rota | CLI executada no repositório real (`repo_e4c8f3b1a2d5`) | `Status: ✅ Encontrado` (2 saltos, custo 2.000 via `code:repo_e4c8f3b1a2d5:import:os`). | ✅ PASS |
| **GNI-18** | AC18 — A identidade canônica fica registrada em ADR nova | `docs/adr/066-a-identidade-canonica-do-no-no-grafo.md:1-176` | Listada em `docs/adr/README.md:69`. Leitura cruzada confirmando status e alinhamento normativo. | ✅ PASS |

---

## Verificação do Motor de Caminho (Pathfinding Engine Invariant)

Conforme estabelecido nos requisitos essenciais e Out of Scope da spec, o motor de cálculo e busca de caminhos **não deve sofrer modificações**:
- **Arquivo**: `internal/graph/path.go`
- **Validação Git**:
  - `git diff 84aab52~1..b086450 -- internal/graph/path.go` → **0 linhas alteradas**.
  - `git diff HEAD -- internal/graph/path.go` → **0 linhas alteradas**.
- **Status**: ✅ **INTOCADO**. O defeito residia estritamente na identidade dos nós no indexador e resolvedor canônico.

---

## Discrimination Sensor (3/3 Mutants Killed)

O sensor de mutação foi executado diretamente pelo Verifier Independente para garantir a sensibilidade dos testes implementados. Cada mutação foi testada individualmente e revertida de imediato, assegurando que o working tree retornasse ao estado limpo.

| # | Mutante | Arquivo & Linha | Mutação Aplicada | Teste Executado | Resultado & Mensagem de Falha | Status |
| :- | :--- | :--- | :--- | :--- | :--- | :--- |
| **M1** | Reversão do alvo para texto bruto em vez de `documents.id` | `cmd/mem/main.go:2352` | Substituição de `store.ResolveEdgeTarget(edge.target, aliasIndex)` por `edge.target` no indexador SQLite | `go test -v -count=1 ./cmd/mem/... -run TestIndexIdentity` | `index_identity_test.go:70: aresta de links_to para o documento de destino não encontrada: sql: no rows in result set` | 💀 **MORTO** |
| **M2** | Corrupção do rótulo do nó (`name`) para string vazia | `internal/db/store.go:159` | Atribuição de `name = ''` em vez de `documents.title` no `upsertGraphNode` | `go test -v -count=1 ./internal/db/... -run TestNodeIdentity` | `graph_identity_test.go:253: name do nó = ""; want "Minha Nota" (o título é o rótulo do nó)` | 💀 **MORTO** |
| **M3** | Desativação da remoção de nós legados sem grau no doctor prune | `internal/db/graph.go:621` | Inserção de `WHERE 1 = 0` na query de deleção de nós legados de `FixHealthIssues` | `go test -v -count=1 ./internal/db/... -run TestDoctorLegacyNodes` | `graph_identity_test.go:192: nós que sobraram = 4; want 3`<br>`graph_identity_test.go:201: nó legado sem grau = 1; want 0 (o --fix remove)` | 💀 **MORTO** |

**Resultado do Sensor**: **3/3 mutantes mortos** (100% de discriminação). Árvore de trabalho limpa após reversão.

---

## Validação de Dados Reais no PostgreSQL

### 1. Cofre Central (`repo_central`)
Medição executada diretamente contra a instância PostgreSQL de produção conectada (`MY_MEMORY_PG_URL`):

```sql
-- Query 1: Arestas com as DUAS pontas em id de documento
SELECT COUNT(*) AS arestas_com_duas_pontas_em_documento
  FROM graph_edges e
  JOIN documents d1 ON d1.repository = e.repository AND d1.id = e.source_id
  JOIN documents d2 ON d2.repository = e.repository AND d2.id = e.target_id
 WHERE e.repository = 'repo_central';
--> 545

-- Query 2: Arestas cujo ALVO resolve para um documento indexado
SELECT COUNT(*) AS arestas_com_alvo_em_documento
  FROM graph_edges e
  JOIN documents d ON d.repository = e.repository AND d.id = e.target_id
 WHERE e.repository = 'repo_central';
--> 545

-- Query 3: Nós legados sem grau (devem ser 0 após mem doctor --fix)
SELECT COUNT(*) AS nos_legados_sem_grau
  FROM graph_nodes n
 WHERE n.repository = 'repo_central'
   AND n.type <> 'code_symbol'
   AND NOT EXISTS (
       SELECT 1 FROM documents d WHERE d.repository = n.repository AND d.id = n.id
   )
   AND NOT EXISTS (
       SELECT 1 FROM graph_edges e
        WHERE e.repository = n.repository
          AND (e.source_id = n.id OR e.target_id = n.id)
   );
--> 0
```

- **Query 1**: 545
- **Query 2**: 545 (Query 1 == Query 2 → 0 arestas divididas entre id de documento e id de título)
- **Query 3**: 0 nós legados sem grau remanescentes

### 2. Prova de Rota no Repositório Real (`repo_e4c8f3b1a2d5`)
Execução do comando CLI:
```bash
./bin/mem.exe path "CONTRIBUTING.md" "README.md" --undirected --storage=postgres --repo repo_e4c8f3b1a2d5
```

**Resultado Observado**:
```text
Status:          ✅ Encontrado
Saltos:          2
Custo Total:     2.000

Rota Conectada:
  [D:\repository\my-memory\CONTRIBUTING.md]
   └──(references_code [INFERRED | forward] | custo: 1.000)──> 
  [code:repo_e4c8f3b1a2d5:import:os]
   └──(references_code [INFERRED | reverse] | custo: 1.000)<── 
  [D:\repository\my-memory\README.md]
```
A rota foi descoberta com sucesso com 2 saltos através do símbolo de import compartilhado.

---

## Gate de Testes e Compilação

| Comando | Escopo | Resultado |
| :--- | :--- | :--- |
| `go test -v -count=1 ./cmd/mem/... -run "TestIndexIdentity\|TestPathIdentity"` | Testes de identidade e E2E CLI | ✅ PASS (3.78s) |
| `go test -v -count=1 ./internal/db/... -run "TestNodeIdentity\|TestResolveCanonical\|TestDoctorLegacyNodes"` | Contratos SQLite de nó, resolvedor e doctor | ✅ PASS (0.45s) |
| `go test -v -count=1 ./internal/watcher/... -run "TestIndexIdentity"` | Indexação cirúrgica watcher | ✅ PASS (0.24s) |
| `MY_MEMORY_PG_TEST=1 go test -v -count=1 ./internal/store/... -run "TestPostgresNodeIdentity\|TestPostgresResolveCanonical\|TestPostgresDoctorLegacyNodes"` | Contratos PostgreSQL no Neon Cloud | ✅ PASS (28.13s) |
| `go test -count=1 ./...` | Suíte completa do repositório | ✅ PASS (27/27 pacotes testados) |
| `go build -v -o bin/mem.exe ./cmd/mem` | Compilação do binário | ✅ PASS (exit 0) |

---

## Scripts Determinísticos da Skill `tlc-spec-driven`

- `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-graph-node-identity --root .`
  - Saída: `validate_spec: 0 error(s), 0 warning(s)`
- `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py fix-graph-node-identity --root .`
  - Saída: `validate_tasks: 0 error(s)`
- `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-graph-node-identity --root .`
  - Executado após a gravação deste relatório de validação.

---

## Conclusão do Verifier Independente

A feature `fix-graph-node-identity` fecha definitivamente a ISSUE-047 (FIX-2). A identidade canônica dos nós no grafo foi unificada com sucesso sob a coluna `documents.id`, com o título atuando como rótulo legível em `graph_nodes.name`. O procedimento de reparo eliminou integralmente as anomalias no vault real do PostgreSQL, sem que nenhuma alteração fosse necessária no motor de busca de caminhos (`internal/graph/path.go`).

**Veredito Final**: 🟢 **PASS**
