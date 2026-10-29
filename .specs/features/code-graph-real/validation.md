---
title: "Validation: code-graph-real — Fase 2 (grafo estrutural) + Fase 3 (ponte notas↔código, travessia limitada, drift estrutural)"
category: resource
summary: "Evidência consolidada do code-graph-real: Fase 2 (grafo estrutural — arestas calls/imports/inherits/uses_type com call-site, provenance EXTRACTED/INFERRED e bucket de incertas; 7 commits 4f1182a..a680955) + Fase 3 (ponte documents↔code_symbols, travessia com hub threshold p99 + orçamento de tokens + ordem por hop, drift estrutural por ast_hash; commits 96a734b..6c64297)."
type: validation
created_at: 2026-10-23T23:40:00-03:00
tags: [validation, code-graph-real, code-ast, epistemic-status, call-site, bridge, hub-threshold, token-budget, drift]
---

# Validation: code-graph-real — Fase 2 (grafo estrutural) + Fase 3 (ponte notas↔código, travessia limitada, drift estrutural)

> [!info] **Documento consolidado (2026-10-28).** Este arquivo passou a reunir a
> evidência das **duas** fases da feature `code-graph-real`, que antes viviam em
> dois arquivos separados (um só com a Fase 2, outro só com a Fase 3). O arquivo
> da Fase 3 foi **absorvido** neste documento em 2026-10-28 e removido do disco.
> O texto das duas fases foi mantido **na íntegra**: a única alteração é a
> numeração das seções da Fase 3 (1–7 → 7–13) e o nível dos títulos, para que
> as duas partes convivam no mesmo documento sem colidir.
>
> - **Parte I — seções 1–6**: Fase 2 (grafo estrutural).
> - **Parte II — seções 7–13**: Fase 3 (T13, T15 e T17).
>
> Os relatórios independentes — `verification.md` (16 hipóteses),
> `validation-revalidation.md` (29 checagens) e `validation-d7-d8.md` — continuam
> em **arquivos separados**, por decisão do gestor: não são evidência de execução,
> são veredictos de terceiro.

## Parte I — Fase 2 (grafo estrutural)

**Data**: 2026-10-23
**Worker**: mavis (sub-agente, branch `mvs_138754ab783a4581bbbeb2f33abd08aa`)
**Spec**: `.specs/features/code-graph-real/spec.md`
**Tasks**: `.specs/features/code-graph-real/tasks.md` (T7–T12; T7–T12 desta fase)
**Commits**: `4f1182a` → `a342518` → `b986c37` → `870a323` → `5f52c2d` → `056a3f6` → `a680955`
**Verdict**: **PASS** — o code graph deixou de ser "símbolos + `references` por nome" e virou grafo estrutural com call-site e provenance por aresta.

---

### 1. O que foi entregue (por tarefa)

| Tarefa | Entrega | Arquivo:linha | Commit |
| :--- | :--- | :--- | :--- |
| T7 | Backend AST em Go puro (`go/parser`+`go/ast`+`go/token`, stdlib): símbolos com kind, nome, qualified name, assinatura, início/fim (linha+coluna), doc comment e receiver | `internal/codeast/go_parser.go:1` | `4f1182a` |
| T8 | `newBackendParser()` deixou de ser código morto e devolve o motor Go; o pipeline usa o backend padrão (mock só como fallback das linguagens sem backend AST); log de boot honesto | `internal/codeast/treesitter_disabled.go:33`, `internal/codeast/pipeline.go:80` | `a342518` |
| T9 | Arestas `calls` com **call-site**; coluna `code_edges.epistemic_status` (TEXT, default `EXTRACTED`) + migração idempotente (SQLite e Postgres) + `UpsertCodeEdgeWithStatus`; resolução em 3 níveis | `internal/codeast/go_parser.go:263`, `internal/db/code_migration.go:60` | `b986c37` |
| T10 | Arestas `imports` (com os símbolos sintéticos que dão corpo ao grafo) + `inherits` + `uses_type` | `internal/codeast/go_parser.go:359` | `870a323` |
| T11 | Fechamento do roteamento para `code_edges_uncertain` (confidence + reason), teste de idempotência da migração e default correto das linhas pré-existentes | `internal/codeast/pipeline.go:452`, `internal/db/code_migration_test.go:1` | `5f52c2d` |
| T12 | Resposta MCP com `epistemic_status`, `confidence`, call-site e posição do destino (campos aditivos) | `internal/mcp/code_handlers.go:1` | `056a3f6` |
| — | Correção achada pela própria prova empírica: aresta de TIPO não pode resolver para um MÉTODO de mesmo nome | `internal/codeast/pipeline.go:486` | `a680955` |

---

### 2. Prova empírica (números, não adjetivos)

Code stage real sobre **este repositório**, SQLite local, banco temporário
(`--storage=sqlite --db <tmp>`), escopo `--include=**/*.go --lang=go`:

```
code AST: backend go (stdlib); tree-sitter: disabled
{"engine":"sqlite","repository":"proof/repo","files_scanned":349,"files_indexed":349,
 "files_skipped":0,"files_errors":0,"symbols_total":6754}
```

#### 2.1 Arestas por kind (`code_edges`)

| kind | arestas |
| :--- | ---: |
| `calls` | 4055 |
| `imports` | 1984 |
| `uses_type` | 908 |
| `references` | 1 |
| `inherits` | 0 |
| **TOTAL** | **6948** |

* `references` = 1: é a **varredida legada por nome**, que só roda quando o parser
  não devolveu arestas — `internal/supervisor/lock_windows.go:17` (arquivo sem
  nenhuma chamada/import/tipo). Provenance `INFERRED`, confidence 0.80.
* `inherits` = 0 **é o número correto para este repositório**: não existe
  embedding de struct nem de interface no código do repo. A única ocorrência
  textual é dentro da string-fonte do fixture de teste
  (`internal/codeast/go_parser_test.go:284`, `Base` embutido), que o parser Go
  não vê como código. A implementação é coberta pelo teste unitário
  `TestGoParserExtraiImportsEInherits` (2 arestas `inherits` no fixture).

#### 2.2 Provenance (`epistemic_status`)

| status | arestas |
| :--- | ---: |
| `EXTRACTED` (resolved by package scope) | 6367 |
| `INFERRED` (resolved by name only) | 581 |

Nas arestas `calls`: `EXTRACTED` 3486, `INFERRED` 569.

`code_edges_uncertain`: **10593 linhas** (nunca promovidas a aresta resolvida)

| kind | linhas | confidence média |
| :--- | ---: | ---: |
| `calls` | 8811 | 0.328 |
| `uses_type` | 1782 | 0.300 |

Por bucket de razão: 9350 `destino não encontrado no repositório` (chamadas a
`testing.T`, `log.Errorf`, `context.Background`, … — destinos fora do repo) e
1243 `ambíguo: N símbolos com o nome "X"` (múltiplos candidatos).

#### 2.3 Call-site ≠ linha da definição (exemplos reais)

```
main.newBuiltinEmbedder --calls--> main.defaultModelsDir
   call-site : cmd/mem/builtin_embedder.go:41  (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : cmd/mem/builtin_embedder.go:14

main.newBuiltinEmbedder --calls--> embedder.NewBuiltinClient
   call-site : cmd/mem/builtin_embedder.go:49  (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : internal/embedder/builtin.go:71

main.runClustersCLI --calls--> main.runClustersCommand
   call-site : cmd/mem/clusters.go:30           (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : cmd/mem/clusters.go:34
```

Caso `INFERRED` (destino resolvido só pelo nome, confidence 0.80):

```
main.formatClustersJSON --calls--> embedder.Tokenizer.Encode
   call-site : cmd/mem/clusters.go:118  (INFERRED, conf=0.80)
   definicao : internal/embedder/tokenizer.go:79
```

#### 2.4 `imports` com destino que EXISTE

```
import:os --imports--> os (kind do destino=import) em cmd/mem/builtin_embedder.go:6  status=EXTRACTED
import:github.com/FelipeMiiller/my-memory/internal/config --imports-->
    github.com/FelipeMiiller/my-memory/internal/config (kind do destino=import)
    em cmd/mem/builtin_embedder.go:9  status=EXTRACTED
```

`imports: 1984 | com destino inexistente: 0` — nenhuma aresta `imports` aponta
para símbolo ausente ou para o símbolo errado.

#### 2.5 Código-fonte NÃO virou nota

Indexação completa (`mem index --storage=sqlite --db <tmp>`) deste repositório:

```
documentos no total: 265
documentos .go: 0
```

Nenhum documento com path `.go`; o code stage é um estágio separado
(`code_files`/`code_symbols`, não `documents`).

#### 2.6 Baseline (símbolos por arquivo)

| | baseline Fase 1 | agora | delta |
| :--- | ---: | ---: | ---: |
| arquivos indexados | 331 | 349 | +18 |
| símbolos | ~2595 | 6754 | +4159 |

**Razão do delta de arquivos**: entre o commit de baseline (`340c9cb`) e o HEAD
entraram 8 arquivos `.go` rastreados — 2 meus (`internal/codeast/go_parser.go`,
`internal/codeast/go_parser_test.go`) e 6 de workers paralelos
(`cmd/mem/lesson*.go`, `internal/db/code_ast_hash_test.go`,
`internal/db/hybrid_code_test.go`, `internal/store/code_search*.go`). O resto
(+10) é a diferença de escopo do snapshot de 331 da Fase 1, que não é
reproduzível a partir da árvore atual.

**Razão do delta de símbolos** — decomposição de 6754:

| kind | símbolos |
| :--- | ---: |
| `import` | 3968 (= 1984 specs × 2 nós sintéticos: declaração + pacote) |
| `function` | 1813 |
| `method` | 355 |
| `struct` | 252 |
| `constant` | 172 |
| `variable` | 135 |
| `class` | 49 |
| `interface` | 10 |

Nenhum símbolo foi **perdido**: o mock por regex contava só linhas `func`/
`type`/`import `; o backend AST extrai também métodos, constantes, variáveis,
tipos nomeados e interfaces, e cria os dois nós sintéticos por import
(decisão documentada no item 6 do briefing). É crescimento de cobertura
estrutural, não regressão.

---

### 3. Decisões do gestor aplicadas

1. **Go puro** — `go/parser`/`go/ast`/`go/token` da stdlib. `go.mod` sem
   dependência nova, sem CGO, sem build tag. A ADR-047 **não** foi tocada: o
   deferral do tree-sitter segue intacto (0/3 gatilhos) e o caminho com a tag
   `treesitter` continua isolado e com precedência
   (`internal/codeast/treesitter_enabled.go`).
2. **Contrato do Parser aditivo** — `FileResult.Edges []Edge` (novo campo);
   `MockParser` e os testes existentes seguem funcionando. A forma do `Edge`
   está documentada no doc comment (`internal/codeast/parser.go:1`).
3. **Schema antes das arestas** — a coluna `epistemic_status` e o helper de
   upsert entraram no commit de T9, antes da primeira aresta ser gravada; a
   migração é idempotente nos dois engines (`EnsureCodeTables`, com
   `pragma_table_info` no SQLite e `information_schema.columns` no Postgres).
4. **Semântica do `calls`** — origem = símbolo ENCLOSING; destino = símbolo
   chamado; `file_id/start_line/end_line` = **call-site**. A mudança de
   semântica da coluna está documentada em `UpsertCodeEdgeWithStatus`.
5. **3 níveis de resolução** — escopo de pacote → `EXTRACTED`; só nome (único)
   → `INFERRED`; ambíguo/não resolvido → `code_edges_uncertain` com
   `confidence` + `reason`, **nunca** promovido a aresta resolvida.
6. **`imports` com alvo que existe** — opção (a): cada import cria dois
   símbolos sintéticos de kind `import` (a declaração, origem da aresta, e o
   pacote importado, destino da aresta). A aresta `imports` liga a declaração
   ao pacote; o destino é sempre uma linha real de `code_symbols`
   (1984/1984, zero órfãs). O nó do pacote é por arquivo, e a resolução da
   aresta é escopada por `file_id`.
7. **A varredida por nome não duplica** — `resolveAndInsertEdges` só roda
   quando o parser **não** devolveu arestas; nesse caso o comportamento
   legado continua idêntico (e as arestas `references` são `INFERRED`, por
   terem sido resolvidas por nome).
8. **Log de boot** — agora imprime `code AST: backend go (stdlib);
   tree-sitter: disabled` (não é mais "(code pipeline skipped)"), porque o
   code pipeline roda. `IsTreesitterEnabled()` continua `false` sem a tag e
   `true` com a tag.
9. **Vocabulário de arestas** — `ValidEdgeKinds()` / `ValidEdgeKind()` com
   `calls`, `imports`, `inherits`, `uses_type`, `references`.

---

### 4. Gate executado (verde antes de cada commit)

```
go build ./...                                                    → 0
go vet ./...                                                      → 0
go test -count=1 ./internal/codeast/... ./internal/db/... ./internal/mcp/...   → ok (3/3)
go test '-bench=.' '-benchmem' '-run=^$' ./internal/codeast ./internal/db ./internal/mcp → ok (3/3)
gofmt -l (cópias LF dos arquivos tocados)                          → vazio
```

Suíte completa (`go test ./...`) **não** foi rodada: o gestor roda no final,
sem escritores ativos (a regra da tarefa). Durante a fase, o único teste que
falhou no pacote `internal/db` foi `TestHybridRRFIncludeCodeEntraNoFanIn`,
pertencente ao trabalho **paralelo** (T14) em `internal/db/hybrid.go` +
`internal/store/code_search.go` (arquivos fora da minha propriedade); ele
voltou a passar quando o worker parallel fechou o código.

---

### 5. Desvio declarado (arquivo fora da minha propriedade)

- `internal/store/postgres.go` (proibido para mim) contém o DDL Postgres de
  `code_edges` **sem** a coluna `epistemic_status`. O caminho Postgres não
  quebra por isso: a coluna é criada pela migração idempotente
  `ALTER TABLE ... ADD COLUMN` de `EnsureCodeTables` (dialect-aware), que roda
  no início do pipeline, antes de qualquer escrita. O `CREATE TABLE` do
  Postgres precisa da coluna para bases novas — **pendência para o gestor /
  para o dono do arquivo**.

### 6. O que NÃO foi feito (e por quê)

- **Não** criei ADR novo: a mudança de schema (coluna `epistemic_status`) segue
  o padrão já existente em `graph_edges` (ADR-011) e a migração é aditiva; a
  regra "mudança de banco exige ADR" continua valendo para o gestor decidir.
- **Não** troquei o `CREATE TABLE` Postgres (arquivo de outro worker).
- **Não** alterei `cmd/mem/codegraph.go`: a tabela markdown do CLI ainda mostra
  só `file`/`line` (que já é o call-site); os campos novos de provenance
  aparecem na **tool MCP** `memory_code_neighbors`, que é o contrato da
  feature. O CLI está fora da minha propriedade.
- **Não** implementei satisfação de interface em geral (um tipo que implementa
  uma interface sem embedding): isso exige type-checking (`go/types`), fora do
  escopo do backend AST. O embedding (struct e interface) é coberto.
- **Não** rodei `go test ./...` (regra da tarefa) nem escrevi em
  `.specs/features/code-graph-real/tasks.md` (o gestor atualiza os
  checkboxes).

## Parte II — Fase 3: Validação empírica — T13, T15 e T17 (code-graph-real)

**Data**: 2026-10-24
**Evidência original**: arquivo da Fase 3, absorvido neste `validation.md` em 2026-10-28
**Commits**: `96a734b` → `6c64297`

Escopo: a ponte notas↔código (CG-17/CG-18), a travessia com orçamento e hub
threshold (CG-21/CG-22) e o drift de mudança estrutural (CG-25/CG-26), mais as
duas reconciliações (A: `include_code`; B: boost CA-05 duplicado).

Ambiente: `go build ./cmd/mem`, SQLite isolado (`--storage=sqlite --db <tmp>`),
sem tocar no Postgres real.

---

### 7. A ponte `documents` ↔ `code_symbols` (T13 / CG-17, CG-18 / AC 18, 19)

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

### 8. Impacto: de um documento para os símbolos de código (D3)

`memory_get_impact` com `node_id = .specs/ISSUES.md` (max_depth 2) devolve a
seção **"Símbolos de Código Alcançados"** com **788 símbolos de código**,
todos de hop 1 e relação `references_code` — a ponte atravessa a fronteira
entre o grafo de notas e o de código.

### 9. Travessia: hub threshold, orçamento e ordem por hop (T15 / CG-21, CG-22 / AC 22, 23)

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

### 10. Drift de mudança estrutural (T17 / CG-25, CG-26 / AC 26, 27)

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

### 11. Reconciliações

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

### 12. Testes automatizados

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

### 13. Gate

```
go build ./...                       → ok
go vet ./...                         → ok
go test -count=1 ./internal/db/... ./internal/mcp/... ./internal/drift/... \
    ./internal/watcher/... ./internal/store/... ./cmd/mem/...   → todo verde
go test '-bench=.' '-benchmem' '-run=^$' ./internal/db ./internal/mcp ./internal/drift
                                       → todo verde
gofmt -l (cópias LF dos arquivos tocados) → vazio
```
