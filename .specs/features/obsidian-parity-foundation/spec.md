# Fundação de Paridade com o Obsidian — mtime no índice e comandos de leitura

## Problem Statement

O objetivo do My-Memory é ser superset das capacidades de vault do Obsidian, com busca semântica por cima (ADR-054). A matriz de gap de 2026-10-18 mapeou 48 comandos de conteúdo de vault: **3 cobertos, 13 parciais, 32 ausentes**.

Duas constatações definem esta primeira entrega.

A primeira é que `documents.updated_at` guarda `time.Now().Unix()` no momento da indexação (`cmd/mem/main.go:1792`, `internal/watcher/indexer.go:93,228`), e não o mtime do arquivo. `CalculateTimeDecay` (`internal/store/rrf.go:110-134`) calcula `2^(-(refTime - updatedAt)/Thalf)` sobre esse campo. Na prática: um `mem index` em vault inteiro dá a todos os documentos timestamps quase idênticos, o delta tende a zero e o multiplicador converge para 1.0 — **o `--decay` vira no-op**. Com indexação incremental via watcher, ele passa a medir "quando eu indexei" em vez de "quando o autor editou". Como `decay: false` é o default, isso não queima ninguém hoje, mas quem liga a flag enxega um sinal errado sem perceber.

A segunda é que a maioria dos comandos de leitura ausentes é SELECT puro sobre dados que já existem. `documents.path` já lista o vault; `chunks.content` já guarda o arquivo bruto; o grafo já tem as arestas para achar deadends. Não é ausência de dado — é ausência de superfície.

## Goals

- [ ] Persistir `mtime` e `size` do arquivo, e fazer o decaimento temporal usar o tempo de edição (ADR-057)
- [ ] Entregar 9 comandos de leitura do Obsidian sem mudar schema além da ADR-057
- [ ] Expor a mesma capacidade via MCP, porque o usuário final do projeto é um agente
- [ ] Manter o `--decay` opt-in e retrocompatível com documentos já indexados

## Out of Scope

| Feature | Reason |
| ----------- | -------------- |
| `properties` / `property:read\|set\|remove` | É a ADR-056 — o maior gap de schema, spec própria |
| `mem tags` / `mem search --tag` | É a ADR-055, ainda `Proposed` com pergunta aberta sobre `resolved_doc_id` |
| `tasks`, `templates`, `bases`, `history` | Real build, sem dado prévio — specs futuras |
| `aliases` como coluna | Nasce dentro da ADR-056; hoje é parseado e descartado |
| Comandos de escrita (`move`, `rename`, `delete`, `prepend`) | Mudam estado; esta entrega é só leitura |
| Comandos de estado do aplicativo | Fora de escopo por ADR-054 (`plugins`, `theme`, `dev:*`, CDP, sync) |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | --------------- | --------- | ---------- |
| Ordem: schema antes dos comandos | `mtime` primeiro, depois os comandos | É a única entrega corretiva; construir por cima de `updated_at` errado fixa o bug no lugar | y |
| Compatibilidade de vault antigo | `mtime = 0` cai para `updated_at` no decay | Nenhum `mem index --force` é exigido para a correção valer | y |
| Comandos consultam o índice, não o disco | Todos leem do banco | `recents` em vault federado pode estar no Postgres sem estar em disco local | y |
| MCP na mesma entrega | Sim, um tool por capacidade | Dividir deixaria a primeira entrega útil só para humano, num projeto cujo usuário é agente | n |
| Ordem de `mem files` | Alfabética por path | Saída estável permite diff entre execuções e teste determinístico | y |
| `mem read` fonte | `chunks.content`, que é o arquivo bruto | Reindexar nunca foi feito; reconstruir de chunk seria lossy | y |

**Open questions:** none — the one open question in the programme (`resolved_doc_id` in ADR-055) is explicitly out of scope and does not block any criterion here.

### Dimensões de requisito implícito (sweep Large/Complex)

| Dimensão | Destino |
|---|---|
| Input validation & bounds | PARITY-13, PARITY-14, PARITY-17 — filtros e caminho inexistente |
| Failure / partial-failure states | PARITY-02 — vault antigo não pode quebrar o build; PARITY-11 — mtime ausente é visível |
| Idempotency / retry | Migrations são `ADD COLUMN` idempotente; reindexar é seguro por natureza |
| Auth boundaries & rate limits | N/A — nenhum comando novo escreve nem acessa rede |
| Concurrency / ordering | PARITY-07 — dois documentos editados em tempos diferentes, indexados na mesma execução, precisam receber decaimentos diferentes |
| Data lifecycle / expiry | N/A — nenhuma purga |
| Observability | PARITY-11 — a fração de documentos sem mtime é reportada, senão a divergência fica invisível |
| External-dependency failure | N/A — nenhum passo depende de serviço externo novo |
| State-transition integrity | PARITY-08 — staleness passa a comparar mtime persistido contra mtime do disco |

---

## User Stories

### P1: Tempo de edição no índice ⭐ MVP

**User Story**: Como usuário que mantém um vault vivo, quero que a ordenação temporal reflita quando eu editei a nota, para que `--decay` priorize conhecimento novo em vez de conhecimento recém-indexado.

**Why P1**: É a única entrega desta spec que corrige comportamento em vez de acrescentar. Construir os comandos de leitura sobre `updated_at` errado deixaria o bug solidificado.

**Acceptance Criteria**

1. WHEN the schema is created or migrated THEN `documents` SHALL have `mtime INTEGER` and `size INTEGER` columns in both the SQLite DDL and the Postgres DDL
2. IF a vault was indexed before this feature THEN the migration SHALL leave existing rows with `mtime = 0` and `go build ./...` SHALL still exit 0
3. WHEN `mem index` processes a file THEN it SHALL persist that file's `mtime` and `size` into `documents`
4. IF a document has `mtime = 0` THEN time decay SHALL fall back to `updated_at`
5. IF a document has `mtime > 0` THEN time decay SHALL use `mtime` and SHALL NOT use `updated_at` <!-- event-driven -->
6. WHEN `--decay` ranks results THEN the decay SHALL reflect file edit time rather than index time
7. IF two documents were edited at different times but indexed in the same run THEN their decay multipliers SHALL differ
8. WHEN staleness is detected THEN the comparison SHALL use the persisted `mtime` against the file's current mtime
9. WHEN `mem file <path>` runs THEN it SHALL report path, title, size and mtime
10. WHEN `mem recents` runs THEN it SHALL list documents ordered by `mtime` descending
11. IF a document has `mtime = 0` THEN `mem doctor` SHALL report the count of documents missing mtime

**Independent Test**: index a vault with two files whose mtimes differ by months but which are indexed in one run, run `mem search --decay`, and observe the two decay multipliers differ.

---

### P2: Comandos de leitura do vault

**User Story**: Como agente ou humano que trabalha no vault, quero listar, ler e inspecionar notas sem shell script, para cobrir a superfície de leitura que o Obsidian CLI oferece.

**Why P2**: Nove comandos de SELECT puro sobre dados que já existem. É a entrega mais barata da matriz inteira.

**Acceptance Criteria**

1. WHEN `mem files` runs with no filter THEN it SHALL list every indexed document path in alphabetical order
2. WHEN `mem files --ext md` runs THEN it SHALL list only paths with that extension
3. WHEN `mem files --folder <dir>` runs THEN it SHALL list only documents whose path starts with that vault-relative prefix
4. IF `mem files --ext` or `--folder` is given an unknown value THEN it SHALL exit zero with an empty list rather than an error
5. WHEN `mem folders` runs THEN it SHALL list every distinct folder that contains at least one indexed document
6. WHEN `mem read <path>` runs THEN it SHALL print the note's raw Markdown to stdout
7. IF the path is not present in the index THEN `mem read` SHALL exit non-zero with an error naming the path
8. WHEN `mem outline <path>` runs THEN it SHALL print the note's headings with their level depth
9. IF the note has no headings THEN `mem outline` SHALL print nothing and exit zero
10. WHEN `mem wordcount <path>` runs THEN it SHALL print word and character counts
11. WHEN `mem random` runs THEN it SHALL print the path and title of a randomly selected document
12. IF `mem random --json` runs THEN it SHALL print a single JSON object with at least `path` and `title`
13. IF the index has no documents THEN `mem random` SHALL exit non-zero with a message saying the index is empty
14. WHEN `mem deadends` runs THEN it SHALL list documents that have at least one outgoing edge and no incoming edge
15. These commands SHALL NOT require any schema change beyond the columns added by criterion 1 of P1
16. WHEN `mem --help` runs THEN its output SHALL list every command introduced by this spec
17. A unit test SHALL assert that every case in the command dispatch table appears in `printHelp()`

**Independent Test**: create a three-note vault with known content, then run each command and compare against the files on disk.

---

### P2: Mesma capacidade via MCP

**User Story**: Como agente de IA integrando via MCP, quero ler e listar notas com as mesmas capacidades do CLI, para não precisar de shell durante uma sessão.

**Why P2**: O My-Memory existe para ser consultado por agente. Entregar a capacidade só no CLI deixa o produto pela metade.

**Acceptance Criteria**

1. WHEN the MCP server starts THEN it SHALL register a tool that lists documents with optional extension and folder filters
2. WHEN the MCP server starts THEN it SHALL register a tool that returns a note's raw Markdown by path
3. WHEN the MCP server starts THEN it SHALL register a tool that returns a note's heading outline
4. WHEN the MCP server starts THEN it SHALL register a tool that returns file metadata including `mtime` and `size`
5. Every new MCP tool SHALL carry a description written as a trigger for an agent, in the style of the existing twenty
6. IF a requested path is absent from the index THEN the read tool SHALL return a tool error, not an empty string
7. The MCP tool count SHALL be recorded in the same places that already record twenty

**Independent Test**: start the MCP server and assert each new tool appears in the tool list with a resolvable input schema.

---

### P2: Documentação da entrega

**User Story**: Como leitor da documentação, quero saber que os comandos existem e o que o `--decay` passou a medir, para não confiar na descrição antiga.

**Acceptance Criteria**

1. `docs/CLI_GUIDE.md` SHALL document `mem file`, `mem recents`, `mem files`, `mem folders`, `mem read`, `mem outline`, `mem wordcount`, `mem random` and `mem deadends`
2. `docs/CLI_GUIDE.md` SHALL describe `--decay` as ranking by edit time, not by index time
3. The canonical skill at `skills/mymemory-cli/SKILL.md` SHALL document the new commands
4. `docs/adr/README.md` SHALL record ADR-057 as implemented
5. `CHANGELOG.md` `[Unreleased]` SHALL record the new commands and the decay correction
6. Files written in this spec SHALL be UTF-8 without BOM

**Independent Test**: grep each new command name across `docs/CLI_GUIDE.md` and the canonical skill; each must appear.

---

## Edge Cases

- IF a vault lives only in Postgres and not on local disk THEN `mem recents` and `mem files` SHALL still work, because every new command reads the index and never the filesystem
- IF two files share a basename in different folders THEN `mem read` SHALL require enough path to disambiguate and SHALL report the ambiguity rather than picking one
- IF a note's `mtime` is in the future (clock skew, or a file restored from an archive) THEN decay SHALL clamp it to a neutral 1.0 rather than amplifying the score
- IF `mem index --force` runs on a vault whose files all share one mtime THEN every document SHALL receive that mtime and `--decay` SHALL correctly report no temporal spread

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| PARITY-01 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-02 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-03 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-04 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-05 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-06 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-07 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-08 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-09 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-10 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-11 | P1: Tempo de edição no índice | Execute | Pending |
| PARITY-12 | P2: Comandos de leitura | Execute | Pending |
| PARITY-13 | P2: Comandos de leitura | Execute | Pending |
| PARITY-14 | P2: Comandos de leitura | Execute | Pending |
| PARITY-15 | P2: Comandos de leitura | Execute | Pending |
| PARITY-16 | P2: Comandos de leitura | Execute | Pending |
| PARITY-17 | P2: Comandos de leitura | Execute | Pending |
| PARITY-18 | P2: Comandos de leitura | Execute | Pending |
| PARITY-19 | P2: Comandos de leitura | Execute | Pending |
| PARITY-20 | P2: Comandos de leitura | Execute | Pending |
| PARITY-21 | P2: Comandos de leitura | Execute | Pending |
| PARITY-22 | P2: Comandos de leitura | Execute | Pending |
| PARITY-23 | P2: Comandos de leitura | Execute | Pending |
| PARITY-24 | P2: Comandos de leitura | Execute | Pending |
| PARITY-25 | P2: Comandos de leitura | Execute | Pending |
| PARITY-26 | P2: Comandos de leitura | Execute | Pending |
| PARITY-27 | P2: Comandos de leitura | Execute | Pending |
| PARITY-28 | P2: Comandos de leitura | Execute | Pending |
| PARITY-29 | P2: Superfície MCP | Execute | Pending |
| PARITY-30 | P2: Superfície MCP | Execute | Pending |
| PARITY-31 | P2: Superfície MCP | Execute | Pending |
| PARITY-32 | P2: Superfície MCP | Execute | Pending |
| PARITY-33 | P2: Superfície MCP | Execute | Pending |
| PARITY-34 | P2: Superfície MCP | Execute | Pending |
| PARITY-35 | P2: Superfície MCP | Execute | Pending |
| PARITY-36 | P2: Documentação da entrega | Execute | Pending |
| PARITY-37 | P2: Documentação da entrega | Execute | Pending |
| PARITY-38 | P2: Documentação da entrega | Execute | Pending |
| PARITY-39 | P2: Documentação da entrega | Execute | Pending |
| PARITY-40 | P2: Documentação da entrega | Execute | Pending |
| PARITY-41 | P2: Documentação da entrega | Execute | Pending |

**Coverage:** 41 total, 41 mapped to stories, 0 unmapped.

**Consolidation note:** IDs are assigned in document order, one per acceptance criterion. Each ID is verified by a named gate command recorded in `tasks.md`; the Verifier checks per-ID evidence rather than per-story.

---

## Success Criteria

- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` all exit 0
- [ ] `mem search --decay` over a vault indexed in one run produces different multipliers for notes edited at different times
- [ ] All nine new CLI commands run and return output on a three-note fixture vault
- [ ] Four new MCP tools appear in the server's tool list
- [ ] `mem doctor` reports zero documents missing mtime on a freshly indexed vault
- [ ] Every new file is UTF-8 without BOM