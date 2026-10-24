---
title: "Verificação independente — code-graph-real"
category: resource
summary: "Relatório do verificador independente da feature code-graph-real (2026-10-24): veredito PARTIAL, 8 defeitos reproduzidos (2 de severidade alta) e os erros de análise encontrados nos próprios documentos da feature."
tags: [my-memory, code-graph, verificacao, adr-061, defeitos]
---

# Verificação independente — `code-graph-real`

> **Data:** 2026-10-24 · **Verificador:** agente `verifier`, read-only
> (o projeto não foi modificado: `git status` vazio em `3012f76`).
> **Briefing:** `D:\.minimax\scratch\briefing-verificador.txt` — 16 hipóteses
> numeradas para **atacar**, não para confirmar.
> **Artefatos da prova:** `D:\.minimax\scratch\subagents\verificador\`.

## Veredito: 🔶 PARTIAL — não é PASS

O núcleo técnico é sólido e **verificadamente real**: backend Go puro,
call-site correto, proveniência, ambiguidade → `code_edges_uncertain`, ponte no
grafo único, tools MCP vivas, viewer dentro do orçamento, `include_code` sem
regressão de ranking e documentação honesta.

Mas há **2 defeitos de severidade alta** que quebram ACs declaradas (AC-07/20,
AC-10, AC-26), **1 alto-médio** que contradiz a AC-14, e **3 médios**.

---

## 1. Resultado por hipótese

| # | Veredito | Evidência |
| :--- | :--- | :--- |
| **H1** | ✅ CONFIRMADA | `treesitter_disabled.go:33` → `NewGoParser()`; boot log `code AST: backend go (stdlib); tree-sitter: disabled`; binário sem CGO indexou 4 arquivos/8 símbolos. *Nuance:* o `MockParser` continua sendo o caminho **de produção** para não-Go. |
| **H2** | ✅ CONFIRMADA | Aresta `calls` gravada com `start_line=8` (a chamada), não 6 (a definição). `code_migration.go:250-258` fixa a semântica; o MCP devolve `call_site_file/line`. |
| **H3** | 🔶 PARCIAL | SQLite: `epistemic_status`/`confidence` presentes e populados; ambíguo (2 símbolos `one.Shared`) → `code_edges_uncertain` com `reason`/`confidence=0.5`, **ausente** de `code_edges`; `INFERRED` observado em aresta resolvida só por nome. **Postgres não verificável** (vedado). |
| **H4** | 🔶 PARCIAL | SQLite confirmado (`--storage=sqlite` → `engine:"sqlite"`). Postgres: só leitura de código (`codeStore.dialect.Bind`, conflict target `(repository,path)`) — **não executado**. |
| **H5** | ✅ CONFIRMADA | Servidor MCP real em `127.0.0.1:39321`: 26 tools; `memory_code_search` e `memory_code_neighbors` retornam dados, não "indisponível". |
| **H6** | ✅ CONFIRMADA | `structuredContent` presente com `epistemic_status`, `confidence`, `call_site_file/line`, `dst_file/dst_line`. |
| **H7** | 🔶 PARCIAL / **DEFEITO** | Ranking sem `include_code` **idêntico** (mesmos ids, score `0.0164`, fontes `fts:1, local`), sem aviso "CA-12 placeholder". **Mas** o fan-in de código só existe se o `repo_id` do vault for `default` (D2). |
| **H8** | ✅ CONFIRMADA | `graph_nodes` tem `('code:alpha.Alpha','code_symbol')`; `graph_edges` tem `references_code`. Idempotente (2 indexações → 1 aresta). `memory_get_impact` alcança `alpha.Alpha` a depth 1. |
| **H9** | 🔶 PARCIAL | p99 (`code_handlers.go:130`), estimador ADR-029 (`graph/pack.go:79`), ordem por hop e hint de estreitamento — todos confirmados. **Mas** o threshold removeu o vizinho **direto (hop 1)** e manteve o hop 2 (D4). |
| **H10** | 🔶 PARCIAL | Drift estrutural existe e atribui via ponte (`"attribution":"bridge"`, `note_ids` → a nota). **Mas** o discriminador disparou em edição cosmética e **não** disparou na mudança de assinatura (D6). |
| **H11** | ✅ CONFIRMADA | `mem lesson` grava no cofre central (`lessons/*.md`), nunca no vault do repo; frontmatter com `category: memory`, `summary`, `created_at/updated_at`; indexado. |
| **H12** | ✅ CONFIRMADA | Bundle **144.667 / 163.840 bytes** gzip-9 (folga 19.173); **zero** `unpkg/jsdelivr/cdn.`; rota `/code` real (`routes/code.tsx`, `routeTree.gen.ts:24`); `src/lib/code-view.test.ts` **é** coletado; zero `.test.tsx` órfãos; typecheck + 95 testes + build verdes. |
| **H13** | 🔶 PARCIAL / **DEFEITO** | `code-index`: `--repo` grava o repo, `--ast-hash=false` → `ast_hash` NULL, `--json` = JSON puro em stdout, `--no-embed` → `flag provided but not defined` + CHANGELOG registra. **Mas** `mem code-graph` ainda aceita e **descarta** `--repo` (D5). |
| **H14** | ✅ CONFIRMADA | `CLI_GUIDE.md:719,723`, `ARCHITECTURE.md:217`, `README.md:64` descrevem honestamente o backend Go, o fallback `MockParser` e o deferral da ADR-047. `docs/adr/README.md:154-179` corrige a imprecisão da ADR-054:37. |
| **H15** | ✅ CONFIRMADA | Zero código morto/placeholder no escopo; `newBackendParser()` vivo. **`gofmt -l` NÃO é defeito**: 194 arquivos listados por `core.autocrlf=true` (CRLF = 758/758). Cópia normalizada LF do mesmo arquivo → `gofmt -l` **vazio**. |
| **H16** | 🔶 PARCIAL | `go build`/`go vet` exit 0. `go test -count=1 ./...` **verde na 2ª execução** (24 `ok`), mas **falhou na 1ª** em `internal/ci` (`TestWorkflowCoversEveryPackage`: `pacote "./..." fora do modulo`). Reproduz 3/3 verde isolado → flake sob carga, não regressão (a feature não tocou `internal/ci`). Frontend: typecheck 0, 95/95, build ok, orçamento ok. |

---

## 2. Defeitos (por severidade)

### D1 — 🔴 ALTA · migração em banco existente quebra
`EnsureCodeTables` (`internal/db/code_migration.go:79-101`) migra
`code_files`/`code_symbols`/`code_edges`, mas **não**
`code_edges_uncertain.repository` — que o DDL pré-feature não tinha
(`git show 1bf7e75~1:internal/db/schema.go`) e que `InsertCodeEdgeUncertain`
(`code_migration.go:290`) insere.

**Reprodução:** com o DDL antigo, `mem code-index` →
`Erro: Pipeline.Run falhou: ... SQL logic error: table code_edges_uncertain has
no column named repository (1)`, exit 1.

**Impacto:** qualquer vault anterior à feature **quebra** ao indexar código.
**Correção:** incluir `code_edges_uncertain` na lista de migração (SQLite e
Postgres), idempotente.

### D2 — 🔴 ALTA · `include_code=true` é no-op quando o `repo_id` ≠ `default`
`internal/db/hybrid.go:205` usa `RepositoryFromContext(ctx)`; `db.WithRepository`
só é chamado em `cmd/mem/main.go:1624` (caminho do `mem index`) — **nunca no
servidor MCP**. `RepositoryFromContext` devolve `"default"`
(`internal/db/repository.go`).

**Reprodução:** vault indexado com `repo=acme` → `include_code=true` devolve
resultado **idêntico** ao `false` (1 item, `0.0164`); o mesmo vault indexado com
`repo=default` → 2 itens, `code:alpha.Alpha` com fonte `code:1`.

**Impacto:** AC-20/CG-19 e AC-07 valem **só** para `repo_id == "default"`.
**Correção:** propagar o `repository` do vault no caminho de busca do servidor
MCP, como o `mem index` já faz.

### D3 — 🟠 MÉDIA-ALTA · build `-tags treesitter` degrada em silêncio
`internal/codeast/treesitter_enabled.go:44-59` é stub: `ParseFile` →
`ErrTreesitterDisabled`, e o pipeline trata como no-op.

**Reprodução:** build padrão = 8 símbolos/4 arestas; build `-tags treesitter` =
**1 símbolo, 0 arestas**, `files_indexed:4`, `files_errors:0`, exit 0.

**Impacto:** quem seguir os passos de ativação documentados no próprio arquivo
(linhas 10-13) recebe um code graph vazio **sem erro**. Contradiz a AC-14.
**Correção:** a tag **não pode** sombrear o backend Go — deve avisar no boot log
que o binding não está implementado e usar o backend Go da stdlib.

### D4 — 🟡 MÉDIA · hub threshold remove o vizinho direto
Em `depth=2`, `nodes_pruned:1` eliminou `beta.Beta` (hop 1, o `calls` de
`alpha.Alpha`) e **mantive** `gamma.Gamma` (hop 2). Com p99 sobre 3 nós,
`int(n*0.99)-1` cai no penúltimo índice → o nó de maior grau é **sempre**
removido em subgrafos pequenos, invertendo a relevância.

**Correção:** só podar acima de um mínimo de nós (o p99 do Graphify é para
grafos grandes) e garantir que o podado é o de **maior grau**.

### D5 — 🟡 MÉDIA · `mem code-graph` descarta flag
`cmd/mem/codegraph.go:32` declara `--repo`; `:39-40` → `_ = defaultRepo;
_ = repoSlug`. Também não tem `--storage` (usa sempre `db.InitDB` local).

**Correção:** honrar `--repo` e ganhar `--storage`/`--db`/`--postgres`.

### D6 — 🟡 MÉDIA · discriminador de drift estrutural frágil
`HashSymbols` usa `HashKey()` **com número de linha**
(`internal/drift/drift.go:548-551` — o próprio código admite que o hash não cobre
a assinatura).

**Reprodução:** edição **cosmética** → `structural_changes` com
`symbols_changed:["pkg.Run"]` (falso positivo); mudança **de assinatura** →
relatório **vazio** (falso negativo).

**Correção:** o hash **estrutural** tem de ser insensível à posição e sensível
à assinatura — são **dois hashes** diferentes (o de cache e o estrutural).

### D7 — 🔵 BAIXA · viewer aponta para a ADR errada
`viewer/src/routes/code.tsx:200` e `:321` mandam o usuário para "ADR-047" (a ADR
deferida) quando o code graph é da **ADR-061**.

### D8 — 🔵 BAIXA · precedência env > config contradiz a documentação
`cmd/mem/main.go:1417-1428`: `MY_MEMORY_PG_URL` **vence**
`cfg.Storage.PostgresURL`; o `AGENTS.md` §2 documenta "global > local > env".
Foi a causa do incidente descrito em §5.

---

## 3. Erros da análise (o que os documentos afirmavam e não era verdade)

1. **A spec tem 29 ACs numeradas (1..29), não 28.** O número "28" estava no
   `HANDOFF.md` e no `STATE.md` — erro do gestor, corrigido.
2. **A AC-14 era falsa como escrita** (exigia um backend tree-sitter que não
   existe). **Emendada** na spec em 2026-10-24: o requisito real é a tag
   **não degradar** o grafo em silêncio.
3. **`README.md:64`** — "filtrado por `repository` (`repo_id`), **como o grafo de
   notas**" é **falso**: `graph_nodes(id,type,name)` e
   `graph_edges(source_id,target_id,relation,epistemic_status,weight)` **não têm**
   coluna `repository` em nenhum engine (`internal/db/schema.go:41-61` e
   `:208-228`).
4. **`spec.md:188-189`** — o escopo de código padrão prometia refletir as
   linguagens realmente suportadas; o default indexou `.py` também (via
   `MockParser`). **Emendada**: só Go tem AST precisa; as outras 9 ficam no mock
   heurístico, sem call-site confiável.
5. **AC-10 / CG-09 não fecham** (`mem code-graph` — D5).
6. **AC-20 / CG-19 e AC-07 só valem para `repo_id == "default"`** (D2).
7. **`spec.md:26`** — "2 das 26 tools MCP são decorativas": o total **26**
   confere (medi), o resto está consertado.

---

## 4. O que ninguém olhou

- **Migração em banco existente** → D1, o mais grave. **Nenhum teste do repo
  cobre `code_edges_uncertain.repository`.**
- **Postgres**: tudo foi provado em **SQLite**. Não verificados: escrita real em
  PG, `ON CONFLICT (repository,path)`, o índice único composto
  (`code_migration.go:124-131`), `epistemic_status` em PG, e `mem lesson` com PG.
- **Concorrência**: nenhum teste de dois `mem index` simultâneos no mesmo banco —
  o code stage e o Markdown stage escrevem no mesmo banco.
- **Idempotência**: `mem index` 2× → ok em SQLite (1 nota, 1 aresta de ponte, 0
  duplicatas); `mem code-index` 2× → "0 arquivos, 0 símbolos" (cache).
  **Não** verificado no Postgres.
- **`--json` com arquivo que erra**: `files_errors` foi sempre 0; nunca foi
  forçado um erro de parse. O único "erro" visto foi **silencioso** (D3).
- **Viewer sem o MCP no ar**: existe um ramo `ErrorState "MCP indisponível"`,
  **não exercitado** (o MCP estava no ar).
- **Federação `repo_id`**: `code_files/code_symbols/code_edges/code_edges_uncertain`
  **têm** `repository` ✓; o lado dos símbolos da ponte é filtrado por repo em PG
  (`code_bridge.go:246-251`) ✓; **mas** o nó do grafo é
  `code:<qualified_name>` **global** → dois repositórios com o mesmo qualified
  name colidem no mesmo `graph_nodes`, e a aresta `references_code` não é
  repo-scoped no grafo de notas. **Colisão cross-repo é risco residual real**
  (documentado só em comentário de código, `code_bridge.go:89-92`).

---

## 5. Disclosure (incidente durante a verificação)

O ambiente tem `MY_MEMORY_PG_URL` definido, e `cmd/mem/main.go:1417-1428` dá
precedência à **env var sobre a config global**. Em dois testes de `mem lesson`
(cenário "engine=postgres"), o binário **ignorou** a URL escrita no config de
scratch e conectou no **Postgres real do dono**: 2 notas-teste
(`Licao sem postgres`, `Licao sem postgres 3`) foram indexadas em `repo_central`
do cofre central. Os `.md` ficaram no vault de scratch do verificador.

**Nada foi apagado do Postgres do dono** (mutação em store real = risco alto) e
**nenhum valor de URL foi impresso**.

**Pendência para o dono:** remover os documentos
`.../central/lessons/licao-sem-postgres.md` e `licao-sem-postgres-3.md` do
cofre central.

**Causa raiz:** D8 (precedência env > config contradiz o `AGENTS.md`).

---

## 6. Contradições entre documentos

- **ADR-061 × ADR-047**: coerentes. A ADR-047 **não** é supersedida, deferral
  intacto com "**Gatilhos disparados: 0**" em `047:252` e `047:291` ✓.
- **ADR-054:37** (diz que `code-*` "AST tree-sitter" já é superior) segue no
  corpo imutável; a correção está em `docs/adr/README.md:154-179` — **é o
  mecanismo correto** (ADRs imutáveis), não um defeito.
- **ADR-047 × AC-14**: contradição interna da spec — **emendada** (§3, item 2).
- **`AGENTS.md` §2 (precedência de storage) × `main.go:1417-1428`** → D8.

---

## 7. Não verificável neste ambiente

Por vedação explícita do dono ao Postgres real: AC-05 em Postgres,
`epistemic_status`/`confidence` em PG, upsert multi-tenant em PG, `mem lesson`
com PG, e o gate do CI com `gofmt` (o checkout Windows com `core.autocrlf=true`
 mascara o resultado — o gate real só é válido em Linux).
