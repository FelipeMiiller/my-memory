---
title: "Validation: code-graph-real — Fase 3 (ponte notas↔código, travessia limitada, drift estrutural)"
category: resource
summary: "Prova empírica da Fase 3: a ponte documents↔code_symbols no grafo único, a travessia de código com hub threshold p99 + orçamento de tokens + ordem por hop, e o drift estrutural por ast_hash. Commits 96a734b..6c64297."
type: validation
created_at: 2026-10-24T07:20:00-03:00
tags: [validation, code-graph-real, code-ast, bridge, hub-threshold, token-budget, drift]
---

# Validação empírica — T13, T15 e T17 (code-graph-real)

Escopo: a ponte notas↔código (CG-17/CG-18), a travessia com orçamento e hub
threshold (CG-21/CG-22) e o drift de mudança estrutural (CG-25/CG-26), mais as
duas reconciliações (A: `include_code`; B: boost CA-05 duplicado).

Ambiente: `go build ./cmd/mem`, SQLite isolado (`--storage=sqlite --db <tmp>`),
sem tocar no Postgres real.

---

## 1. A ponte `documents` ↔ `code_symbols` (T13 / CG-17, CG-18 / AC 18, 19)

Indexação deste repositório (`mem index --storage=sqlite --db <tmp> --code-ast`):

| Métrica | Valor |
| --- | --- |
| documentos indexados | 266 |
| `code_files` | 401 |
| `code_symbols` | 7001 |
| **pares documento→símbolo (`graph_edges.relation='references_code'`)** | **34 181** |
| pares distintos (documento, símbolo) | 34 181 |
| **`graph_nodes` com `type='code_symbol'`** | **2 827** |
| documentos distintos na ponte | 265 |
| arestas com `epistemic_status='EXTRACTED'` | 28 851 |
| arestas com `epistemic_status='INFERRED'` | 5 330 |

Exemplo concreto:

| source_id | target_id | relation | epistemic_status | node type | node name |
| --- | --- | --- | --- | --- | --- |
| `.agents/rules/always-quality-gate.md` | `code:log` | `references_code` | `EXTRACTED` | `code_symbol` | `log` |

Exemplo do match fraco (só pelo nome, com fronteira de palavra):

| source_id | target_id | relation | epistemic_status |
| --- | --- | --- | --- |
| `.agents/rules/always-quality-gate.md` | `code:codeast.Pipeline` | `references_code` | `INFERRED` |

**Idempotência**: um segundo `mem index` (tudo em cache no Markdown, code stage
de novo) devolveu exatamente os mesmos números — 34 181 arestas, 2 827 nós
`code_symbol`, 265 documentos na ponte. **Nenhuma aresta duplicada.**

O código do `mem code-stage` confirma o resultado:

```
[code-stage] code graph atualizado: 401 arquivos, 7001 simbolos
[code-stage] ponte notas↔codigo: 34181 aresta(s) em 266 documento(s)
```

## 2. Impacto: de um documento para os símbolos de código (D3)

`memory_get_impact` com `node_id = .specs/ISSUES.md` (max_depth 2) devolve a
seção **"Símbolos de Código Alcançados"** com **788 símbolos de código**,
todos de hop 1 e relação `references_code` — a ponte atravessa a fronteira
entre o grafo de notas e o de código.

## 3. Travessia: hub threshold, orçamento e ordem por hop (T15 / CG-21, CG-22 / AC 22, 23)

Mesmo grafo, mesmo seed, com o hub threshold no default (p99) e desligado
(`MY_MEMORY_CODE_HUB_PERCENTILE=100`, que equivale ao percentil do grau máximo
— nada é podado):

| Travessia | p99 (com hub threshold) | p100 (sem) | nós que caem |
| --- | --- | --- | --- |
| `db.BuildCodeBridge`, out, depth 3 | 39 vizinhos · threshold 9 | 40 vizinhos · threshold 10 | **1** |
| `db.BuildCodeBridge`, both, depth 3 | 49 vizinhos · threshold 9 | 50 vizinhos · threshold 10 | **1** |
| `db.InitDB`, in, depth 2 | 386 vizinhos · threshold 12 | 389 vizinhos · threshold 154 | **3** |
| `db.InitDB`, both, depth 3 | 445 vizinhos · threshold 16 | 468 vizinhos · threshold 154 | **23** |

O corte por **orçamento de tokens** (mesmo estimador do `memory_pack_context`,
ADR-029 — `graph.EstimateTokens`):

| Orçamento | vizinhos devolvidos | tokens estimados | truncado | dica |
| --- | --- | --- | --- | --- |
| 4000 (default) | 33 | 3988 | sim | sim |
| 120 | 1 | 109 | sim | sim |

A dica devolvida:

```
Resposta truncada. Estreite a query: use o qualified name completo, reduza a
profundidade (depth) ou filtre por language/kind. (1 no(s)-hub removido(s)
(grau acima do percentil); 117 vizinho(s) fora do orcamento de 4000 tokens).
```

A **ordem por distância em saltos** é respeitada em todas as travessias
(distribuição por hop de `db.BuildCodeBridge` out depth 3: `{1:1, 2:9, 3:29}`,
ordem crescente = true).

`structuredContent` é emitido pelas duas tools de código (`memory_code_search` e
`memory_code_neighbors`) — aditivo: o texto JSON continua sendo a saída humana.

## 4. Drift de mudança estrutural (T17 / CG-25, CG-26 / AC 26, 27)

Cenário isolado (repo git próprio): `pkg/alpha.go` e `pkg/beta.go` indexados
pelo code stage, uma nota que referencia `pkg.Alpha` e `pkg.Beta`. Depois:

- `pkg/alpha.go`: **mudança de assinatura** (`func Alpha(a int)` →
  `func Alpha(a int, b string)`);
- `pkg/beta.go`: **só o texto do comentário** (mesmo número de linhas).

`mem drift --since HEAD~1..HEAD --threshold 0.0 --json`:

```json
"drifted_notes": [
  {
    "title": "Nota Alpha",
    "affected_files": ["pkg/alpha.go"],
    "structural_changes": ["pkg/alpha.go"]
  }
],
"structural_changes": [
  {
    "file_path": "pkg/alpha.go",
    "package": "pkg",
    "status": "M",
    "additions": 1,
    "deletions": 1,
    "symbols_changed": ["pkg.Alpha"],
    "note_ids": ["...\\docs\\nota.md"],
    "attribution": "bridge"
  }
]
```

- `pkg/alpha.go` (assinatura) → **reportado** como mudança estrutural, atribuído
  à nota **pela ponte** (`attribution: "bridge"`).
- `pkg/beta.go` (comentário) → **não reportado**: não está em
  `structural_changes`, nem em `affected_files`, e `uncovered_code` ficou `null`
  (o arquivo foi descartado por ser cosmético).
- **Sem `ast_hash` gravado** → não é erro: o drift segue pelo mecanismo textual
  de menção de path (coberto por `TestDriftSemAstHashNaoQuebra`).

Nota de semântica: o `ast_hash` do pipeline é `kind|qualified_name|start_line`
(ADR-047) — ele **não** cobre a assinatura. Por isso a classificação é
"hash mudou **ou** a assinatura de um símbolo mudou": a mudança de assinatura
puro mantém o hash e ainda assim é estrutural. Uma mudança que *insere* uma
linha (comentário novo) desloca as linhas e **muda** o `ast_hash` — nesse caso
o drift reporta, por contrato do hash.

## 5. Reconciliações

**A — `include_code`**: a fonte única é a variável `includeCode`, calculada
**uma vez** em `cmd/mem/main.go` (`buildSQLiteSearchFunc` e o caminho
Postgres) e repassada para **os dois** canais:

```go
includeCode := params.IncludeCode || store.IncludeCodeFromContext(ctx)
ctx = store.WithIncludeCode(ctx, includeCode)
searchOpts := store.SearchOptions{ ..., IncludeCode: includeCode }
```

`store.IncludeCodeRequested` continua sendo um OR, mas as duas leituras recebem
o **mesmo** valor — não há como divergirem.

**B — boost CA-05 duplicado**: eliminada a cópia em `internal/mcp`. O pacote
`mcp` **importa** o `store` (o store não pode importar o mcp: ciclo
`mcp → db → store`), então a direção correta é o mcp **reusar** a do store:

- `internal/mcp/code_handlers.go` → `ApplyCodeSymbolBoostMCP` converte os hits
  e chama `store.ApplyCodeSymbolBoost`;
- `HasQualifiedNameShapeMCP` delega a `store.HasQualifiedNameShape`.

Comportamento idêntico — testes verdes:
`TestApplyCodeSymbolBoost_CA05`, `TestApplyCodeSymbolBoost_SemFormaDeQualifiedName`,
`TestMemoryCodeSearchHandler_BoostActiveByDefault`,
`TestMemoryCodeSearchHandler_NoBoostFlag`, `TestHasQualifiedNameShapeMCP`.

## 6. Testes automatizados

| Teste | Arquivo:linha | Cobre |
| --- | --- | --- |
| `TestDocumentCodeSymbolEdge` | `internal/db/code_bridge_test.go:46` | nó `code_symbol`, aresta `references_code`, EXTRACTED e **idempotência** (2 passes, 1 aresta) |
| `TestDocumentCodeSymbolEdgeInferido` | `internal/db/code_bridge_test.go:138` | match fraco (só o nome) → INFERRED |
| `TestBuildCodeBridgeForDocument` | `internal/db/code_bridge_test.go:174` | o indexador de Markdown reconstrói só o documento novo |
| `TestCodeBridgeQueries` | `internal/db/code_bridge_test.go:215` | documentos→símbolo, símbolo→documentos, nó→símbolo |
| `TestTravessiaAplicaHubThreshold` | `internal/mcp/code_handlers_test.go:590` | com p99 o hub sai (20 folhas sobrevivem, 1 nó podado); com p100 ele volta |
| `TestTravessiaRespeitaOrcamentoDeTokens` | `internal/mcp/code_handlers_test.go:642` | orçamento estoura → truncado + dica de estreitar |
| `TestTravessiaOrdenaPorHop` | `internal/mcp/code_handlers_test.go:688` | ordem por distância em saltos |
| `TestCodeNeighborsIncluiDocumentosDaPonte` | `internal/mcp/code_handlers_test.go:715` | a travessia devolve os documentos que referenciam o símbolo |
| `TestCodeToolsEmitemStructuredContent` | `internal/mcp/code_traversal_test.go:14` | `structuredContent` aditivo nas duas tools de código |
| `TestBuscaDeCodigoRespeitaOrcamento` | `internal/mcp/code_traversal_test.go:87` | a busca também respeita o orçamento e avisa |
| `TestDriftDetectaMudancaEstrutural` | `internal/drift/structural_test.go:121` | assinatura → drift estrutural; comentário → **não** é drift |
| `TestDriftSemAstHashNaoQuebra` | `internal/drift/structural_test.go:196` | sem `ast_hash` gravado o drift não quebra (AC 27) |

## 7. Gate

```
go build ./...                       → ok
go vet ./...                         → ok
go test -count=1 ./internal/db/... ./internal/mcp/... ./internal/drift/... \
    ./internal/watcher/... ./internal/store/... ./cmd/mem/...   → todo verde
go test '-bench=.' '-benchmem' '-run=^$' ./internal/db ./internal/mcp ./internal/drift
                                       → todo verde
gofmt -l (cópias LF dos arquivos tocados) → vazio
```
