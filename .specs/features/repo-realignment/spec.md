# Repositorial — Remoção de Código Morto e Realinhamento Documental

## Problem Statement

Duas migrações estruturais ocorreram no `develop` e nunca foram propagadas: o viewer Electron foi revertido em `b477200` (112 arquivos, 27.858 linhas) e o embedder migrou de Ollama/768d para builtin/384d na T7. O resultado é um repositório cujo código está saudável (build, vet e 30/30 pacotes de teste verdes) mas cercado de três camadas de entulho: ~2.600 linhas de código com zero importadores, documentação que afirma um mundo que não existe mais, e uma lacuna de CI que nunca exercita 8 pacotes — três deles ligados a comandos que o usuário executa.

Pior: `mem init` ainda semeia todo vault novo com `provider: "ollama"` e `dimension: 768`, valores que o código ignora silenciosamente (`resolveEmbedder` chama apenas `newBuiltinEmbedder`; `EmbeddingDim = 384` é constante). Todo vault criado hoje nasce com uma mentira.

## Goals

- [ ] Remover todo o código Go com zero importadores, verificado por grafo de imports, sem quebrar build, vet ou testes
- [ ] Fazer `mem init` gerar config coerente com o embedder real (`builtin`, 384d), com teste de regressão
- [ ] Alinhar README, skill, CLI_GUIDE, STATE.md, CHANGELOG, índice de ADRs e ISSUES.md ao código
- [ ] Fechar a lacuna de CI cobrindo os 8 pacotes nunca executados
- [ ] Eliminar os 239 falsos positivos de `gofmt -l .` causedos por CRLF, sem reescrever o repositório
- [ ] Deixar o repositório sem lixo de working tree e com os artefatos de viewer arquivados

## Out of Scope

Documentado para impedir scope creep.

| Feature | Reason |
| ----------- | -------------- |
| Deletar `cmd/mymemoryd` e o lifecycle do supervisor | O usuário decidiu manter. O código tem ~1.400 linhas de teste prontas; falta ligação, não foundations |
| Implementar o wiring dos workers do supervisor | Reconhecido como gap conhecido via ISSUE, não resolvido aqui |
| Deletar as ADRs obsoletas 037/038/048/049/051/053 | Registro histórico é o propósito de uma ADR. `docs/adr/README.md` já as marca obsoletas |
| Deletar `internal/graphview/template.go` | Parece "viewer" mas é o HTML vivo de `mem graph` e da tool `memory_visualize_graph` |
| Deletar `internal/embedder/ollama.go` | Tem testes reais e benchmark próprios. Fica registrado como dívida, não deletado |
| Renumerar o slot ADR-052 | Referenciado por 4 documentos que passam a citar "slot reservado"; renumerar é outra tarefa |
| Migrar de volta para Postgres ou trocar embedder | Nenhuma mudança de persistência ou modelo está em jogo |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | --------------- | --------- | ---------- |
| Escopo da deleção | Apenas código com **zero** importadores não-test, mais o `main.go` da raiz | Critério verificable objetivamente por grafo de imports; evita deletar superfície de API que pode ser religada | y |
| Destino dos pacotes `writer`/`policy`/`asr`/`ipc` | Deletar do código; **preservar** as ADRs 042/044/045 | O registro de por que a arquitetura foi tentada tem valor; só o código ocioso atrapalha | y |
| `mem up` / daemon ocioso | Manter e registrar como ISSUE aberto | O usuário optou explicitamente por manter; transformar em gap rastreável em vez de silêncio | y |
| Relação entre as duas skills | `skills/mymemory/` (raiz do repo) = **canônica pública**; `.agents/skills/my-memory/` = **stub** que apenas aponta para ela | Duas skills com superfície sobreposta e profundidades diferentes fazem agentes escreverem docs em arquivos diferentes. Uma fonte, um lugar. O stub local preserva o ponto de entrada para agentes que procuram `.agents/skills/` | y |
| Versão da skill pública | Autonomia da `.skill-meta.json` (`1.0.0`), independente do SemVer do binário | A skill é distribuível fora do binário; acoplar os dois versionamentos cria release Conjunto | n |
| Tratamento do CRLF | Criar `.gitattributes` com `* text=auto eol=lf`; **não** rodar `gofmt -w .` | O blob no HEAD tem 0 CR; o CRLF vem de `core.autocrlf=true` local. Reescrever 239 arquivos geraria diff massivo sem valor | y |
| Artefatos de viewer em `.specs/` | Arquivar em `.specs/_archive/` em vez de apagar | `release-v1.5.0-notes.md` tem link para `docs/VIEWER.md` já deletado; arquivo mantém a rastreabilidade sem poluir o índice ativo | n |
| Ordem de execução | Deleção de código **antes** do sweep documental | Documento que descreve código morto é pior que documento ausente; a ordem garante que o doc descreve o estado final | y |

**Open questions:** none — all resolved or logged above.

### Dimensões de requisito implícito (sweep Large/Complex)

| Dimensão | Destino |
|---|---|
| Input validation & bounds | N/A — nenhum input de usuário novo; o único output gerado é o template de config, coberto por CLEAN-08..CLEAN-11 |
| Failure / partial-failure states | CLEAN-06 — `go mod tidy` pode deixar dependência órfã; o gate de build é a checagem |
| Idempotency / retry | CLEAN-12 — o sweep documental deve ser re-executável sem efeito adicional |
| Auth boundaries & rate limits | N/A — nenhuma superfície de rede ou permissão é alterada |
| Concurrency / ordering | CLEAN-07 — ordem de commits imposta: código antes de documento |
| Data lifecycle / expiry | N/A — nenhum estado persistido é migrado ou expurgado |
| Observability | CLEAN-13 — o gate de CI é o mecanismo de observabilidade que fecha a lacuna |
| External-dependency failure | CLEAN-06 — remoção de pacote pode órfanizar dependência de terceiro em `go.mod` |
| State-transition integrity | CLEAN-16 — reescrita de `.specs/STATE.md` e do índice de ADRs precisa convergir para status verdadeiros |

---

## User Stories

### P1: Base sem código morto ⭐ MVP

**User Story**: Como maintainer do `my-memory`, quero que o repositório não contenha código que nada executa, para que eu possa desenvolver sobre a base real sem ler arquitetura que não existe.

**Why P1**: É o que desbloqueia todo o resto. Enquanto 2.600 linhas mortas estiverem lá, qualquer análise de código, grep ou briefing de agente devolve arquitetura que não existe.

**Acceptance Criteria**

1. WHEN `go list ./...` runs THEN the module graph SHALL NOT contain the packages `.`, `internal/writer`, `internal/writer/projection`, `internal/policy`, `internal/asr`, or `internal/ipc/jsonrpc` <!-- event-driven -->
2. IF a recursive grep for `internal/writer`, `internal/asr`, or `internal/ipc` across `*.go` returns any non-comment hit THEN the cleanup gate SHALL fail <!-- unwanted-behavior -->
3. WHEN `go build ./...` runs THEN the compiler SHALL exit with code 0 <!-- event-driven -->
4. WHEN `go vet ./...` runs THEN the tool SHALL exit with code 0 <!-- event-driven -->
5. WHEN `go test -count=1 ./...` runs THEN every package SHALL report `ok` or `[no test files]` with zero `FAIL` lines <!-- event-driven -->
6. IF removing the dead packages orphans a third-party dependency in `go.mod` THEN `go mod tidy` SHALL be run and `go build ./...` SHALL still exit 0 <!-- unwanted-behavior -->
7. The cleanup SHALL NOT delete `internal/graphview/template.go`, `internal/supervisor/profile.go`, `internal/supervisor/lock_*.go`, `internal/supervisor/prctl_unix.go`, `internal/supervisor/rlimit_unix.go`, `internal/supervisor/sandbox_unix.go`, `internal/db/vec_cgo.go`, or `internal/db/vec_fallback.go` <!-- ubiquitous -->
8. WHEN `cmd/mem/replay.go` is removed THEN `cmd/mem/main.go` SHALL NOT contain a `replay` case and `printHelp()` SHALL NOT list a `replay` command <!-- event-driven -->
9. The commit history SHALL record the removal of each dead cluster as a separate Conventional Commit <!-- ubiquitous -->

**Independent Test**: `go build ./... && go vet ./... && go test -count=1 ./...` all green, and `go list ./...` shows 24 packages with none of the six deleted ones.

---

### P1: `mem init` gera config coerente com o embedder real

**User Story**: Como usuário que cria um vault novo, quero que o `config.yaml` gerado descreva o embedder que realmente roda, para não depurar vetores vazios descubrindo que a config é ignorada.

**Why P1**: É a única correção que é bug de código, não documentation drift. Todo vault nascido hoje carrega uma config falsa.

**Acceptance Criteria**

1. WHEN `mem init` runs in a directory with no existing vault THEN the generated `.memory/config.yaml` SHALL contain `provider: "builtin"` <!-- event-driven -->
2. WHEN `mem init` runs THEN the generated `.memory/config.yaml` SHALL contain `dimension: 384` <!-- event-driven -->
3. The generated `.memory/config.yaml` SHALL NOT contain the substring `nomic-embed-text` <!-- ubiquitous -->
4. The generated `.memory/config.yaml` SHALL NOT contain the substring `localhost:11434` <!-- ubiquitous -->
5. The repository's own tracked `.memory/config.yaml` SHALL declare `provider: "builtin"` and `dimension: 384` <!-- ubiquitous -->
6. IF a config file declares `dimension: 768` THEN the embedder SHALL use the `EmbeddingDim` constant and SHALL NOT use the configured value <!-- unwanted-behavior -->
7. WHEN the regression test for the config template runs THEN it SHALL assert the exact contents of the `embedding:` block, not a substring match <!-- ubiquitous -->

**Independent Test**: Run `mem init` in a temp dir, read the generated `config.yaml`, observe `builtin` and `384`.

---

### P2: Documentação alinhada ao código

**User Story**: Como leitor da documentação e como agente de IA que a consome, quero que cada documento descreva o estado real, para não seguir instruções que apontam para comandos inexistentes.

**Why P2**: Um agente que lê `SKILL.md` hoje é instruído a instalar Ollama, que foi removido do caminho vetorial, e a rodar `mem embed download`, que não existe.

**Acceptance Criteria**

1. `README.md` SHALL state the MCP tool count as 20 in every location where a count appears <!-- ubiquitous -->
2. `README.md` and `COMO_FUNCIONA.md` SHALL describe the embedder as `paraphrase-multilingual-MiniLM-L12-v2` at 384 dimensions <!-- ubiquitous -->
3. No skill file SHALL present Ollama as a prerequisite, as a default provider, or as a fallback path for embeddings <!-- ubiquitous -->
4. The canonical skill at `skills/mymemory/SKILL.md` SHALL document `mem up`, `mem down`, `mem logs`, `mem profiles`, `mem events`, and the four `mem code-*` commands <!-- ubiquitous -->
5. The canonical skill SHALL list all 20 MCP tools, including `memory_reconcile` <!-- ubiquitous -->
6. `docs/CLI_GUIDE.md` SHALL document `mem up`, `mem down`, `mem logs`, `mem profiles`, and `mem events` <!-- ubiquitous -->
7. `docs/CLI_GUIDE.md` SHALL document `mem doctor --events` and `mem search --links` <!-- ubiquitous -->
8. IF the documentation states a flag for `mem code-graph` or `mem code-stats` THEN the stated flag SHALL exist in the corresponding `cmd/mem/*.go` flag registration <!-- unwanted-behavior -->
9. `.specs/STATE.md` SHALL NOT describe the Electron viewer as shipped, in draft, or as an active decision <!-- ubiquitous -->
10. `.specs/STATE.md` SHALL list ADR-035 exactly once, with a status consistent with the file `035-*.md` <!-- ubiquitous -->
11. `docs/adr/README.md` SHALL index ADR-035, 036, 039, 040, 041, 042, 043, 044, and 050 <!-- ubiquitous -->
12. IF the ADR index lists ADR-045 THEN the index SHALL mark it `Deferred`, matching `045-*.md` <!-- unwanted-behavior -->
13. `docs/adr/035-*.md` SHALL declare status `Accepted`, matching its implementation on 2026-10-13 <!-- ubiquitous -->
14. `CHANGELOG.md` `[Unreleased]` SHALL document `memory_reconcile`, the Ortografia section of `mem doctor`, the 768→384 embedder migration, and the `mem events` / `mem up` / `mem down` / `mem logs` / `mem profiles` / `mem code-*` commands <!-- ubiquitous -->
15. `.specs/ISSUES.md` metrics table SHALL include a row for ISSUE-012, ISSUE-013, ISSUE-015, and ISSUE-017 <!-- ubiquitous -->
16. `.specs/ISSUES.md` SHALL mark ISSUE-007 as moot with the reason that Playwright was removed with the viewer <!-- ubiquitous -->
17. `cmd/mem/main.go` `printHelp()` SHALL list every subcommand registered in the dispatch table <!-- ubiquitous -->

**Independent Test**: `grep -rn "nomic-embed-text" README.md COMO_USAR.md COMO_FUNCIONA.md CONTRIBUTING.md docs/` returns no hit; `printHelp()` output matches the dispatch table.

---

### P2: CI cobrando o que existe

**User Story**: Como maintainer, quero que o CI rode os pacotes que os comandos de produção usam, para não descobrir regressão só depois do merge.

**Why P2**: `supervisor`, `event_runtime` e `federation` estão ligados a comandos que o usuário executa e nunca entram no CI.

**Acceptance Criteria**

1. `.github/workflows/ci.yml` SHALL execute `internal/supervisor`, `internal/event_runtime`, and `internal/federation` test suites <!-- ubiquitous -->
2. `.github/workflows/ci.yml` SHALL execute `internal/policy`, `internal/asr`, `internal/ipc/jsonrpc`, `internal/writer`, `internal/writer/projection`, and `bench/codeast` <!-- ubiquitous -->
3. IF a package listed in `go list ./...` is neither in an existing CI test step nor in the documented exclusion list THEN the CI gap check SHALL fail <!-- unwanted-behavior -->
4. `.github/workflows/ci.yml` SHALL NOT reintroduce any Electron, Node, Vite, or Playwright setup step <!-- ubiquitous -->

**Independent Test**: For every package from `go list ./...`, assert it appears in ci.yml or in a documented exclusion list.

---

### P2: Gate de formatação utilizável no Windows

**User Story**: Como developer no Windows, quero que `gofmt -l .` retorne vazio sem precisar reescrever o repositório, para que o gate do `AGENTS.md` seja obedecível.

**Why P2**: O gate é exigido pelo `AGENTS.md` mas é impossível de passar localmente, o que treina o agente a ignorar o gate.

**Acceptance Criteria**

1. The repository SHALL contain a `.gitattributes` declaring `* text=auto eol=lf` <!-- ubiquitous -->
2. WHEN a fresh checkout is materialized with `core.autocrlf=true` THEN every tracked `*.go` file SHALL contain LF line endings and zero CR bytes <!-- event-driven -->
3. WHEN `gofmt -l .` runs in that checkout THEN it SHALL print nothing <!-- event-driven -->
4. IF applying `.gitattributes` would rewrite more than zero existing tracked files THEN the change SHALL be limited to `.gitattributes` alone and SHALL NOT reformat source files <!-- unwanted-behavior -->

**Independent Test**: `git ls-files --eol "*.go"` reports `i/lf w/lf` for every Go file and `gofmt -l .` prints nothing.

---

### P3: Working tree limpo e artefatos de viewer arquivados

**User Story**: Como maintainer, quero que o repositório não carregue rascunhos e artefatos de uma arquitetura removida, para que a limpeza final deixe um ponto de partida claro.

**Why P3**: É o requisito explícito de "no final, uma limpa no repositório", e é o que diferencia um repositório atualizado de um apenas documentado.

**Acceptance Criteria**

1. The files `config-global.txt`, `config-local.txt`, `mem.exe`, and `bin/scratch_ops/faults.py` SHALL NOT exist in the working tree <!-- ubiquitous -->
2. IF a file named in the preceding criterion is untracked and gitignored THEN removing it SHALL require no git command <!-- unwanted-behavior -->
3. The viewer-era artifacts `ROADMAP-v3-workspace-architecture.md`, `pr-feat-v3-workspace-architecture.md`, `pr-feat-chat-language-models.md`, and `release-v1.5.0-notes.md` SHALL be moved under `.specs/_archive/` <!-- ubiquitous -->
4. IF an archived artifact contains a link to `docs/VIEWER.md` THEN it SHALL NOT be reachable as a live link from any active document <!-- unwanted-behavior -->
5. WHEN the cleanup finishes THEN `git status --porcelain` SHALL show no untracked file other than intentionally staged work <!-- event-driven -->

**Independent Test**: `Test-Path` on each junk file returns false; `git status --porcelain` is clean.

---

### P3: Skill pública canônica com stub local, e issue do daemon registrada

**User Story**: Como agente de IA e como mantenedor, quero uma única fonte da verdade para a skill e quero o daemon ocioso rastreado, para não documentar nem promover a superfície errada.

**Why P3**: Duas skills descrevem a mesma superfície com profundidades diferentes; sem uma fonte única, agentes escrevem docs em arquivos diferentes e divergem.

**Acceptance Criteria**

1. `skills/mymemory/` SHALL be committed to git and SHALL NOT be reported as untracked <!-- event-driven -->
2. `.agents/skills/my-memory/SKILL.md` SHALL contain no duplicated CLI or MCP reference content and SHALL instead point to `skills/mymemory/SKILL.md` <!-- ubiquitous -->
3. `.agents/skills/my-memory/SKILL.md` SHALL remain loadable by an agent that discovers skills under `.agents/skills/` <!-- ubiquitous -->
4. IF an agent reads `.agents/skills/my-memory/SKILL.md` THEN it SHALL be directed to the canonical public skill by an explicit relative path <!-- unwanted-behavior -->
5. `.specs/ISSUES.md` SHALL contain an open issue describing that `mem up` starts a daemon whose `Run` blocks on context cancellation without starting workers <!-- ubiquitous -->
6. The daemon issue SHALL cite `cmd/mymemoryd/main.go` and `internal/supervisor/manager.go` as evidence <!-- ubiquitous -->

**Independent Test**: `git status` shows `skills/` tracked; `grep -rn "mymemoryd" .specs/ISSUES.md` returns a hit.

---

## Edge Cases

- IF `go mod tidy` removes a dependency still needed by a build-tagged file such as `internal/codeast/treesitter_enabled.go` THEN the gate SHALL re-run `go build -tags treesitter ./...` before declaring success
- IF the machine has `CGO_ENABLED=0` and no C compiler THEN `go build -tags onnx ./...` SHALL fail on `onnxruntime_go: build constraints exclude all Go files`, and this failure SHALL be recorded as environmental rather than treated as a regression — it reproduces at the pre-cleanup baseline and is unrelated to this feature
- IF the tracked `.memory/config.yaml` is regenerated by `mem init --force` during cleanup THEN it SHALL be committed with `builtin`/384, never reverted to the old values
- IF a documentation sweep finds that an ADR is referenced by another ADR for methodology anchoring THEN the referencing text SHALL be updated to a still-valid ADR rather than the reference being deleted
- IF archiving a viewer-era artifact leaves `.specs/` with no roadmap document THEN the index SHALL explicitly state that no active roadmap exists <!-- unwanted-behavior -->

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| CLEAN-01 | P1: Base sem código morto | Execute | Pending |
| CLEAN-02 | P1: Base sem código morto | Execute | Pending |
| CLEAN-03 | P1: Base sem código morto | Execute | Pending |
| CLEAN-04 | P1: Base sem código morto | Execute | Pending |
| CLEAN-05 | P1: Base sem código morto | Execute | Pending |
| CLEAN-06 | P1: Base sem código morto | Execute | Pending |
| CLEAN-07 | P1: Base sem código morto | Execute | Pending |
| CLEAN-08 | P1: Base sem código morto | Execute | Pending |
| CLEAN-09 | P1: Base sem código morto | Execute | Pending |
| CLEAN-10 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-11 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-12 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-13 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-14 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-15 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-16 | P1: `mem init` gera config coerente | Execute | Pending |
| CLEAN-17 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-18 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-19 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-20 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-21 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-22 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-23 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-24 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-25 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-26 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-27 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-28 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-29 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-30 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-31 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-32 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-33 | P2: Documentação alinhada | Execute | Pending |
| CLEAN-34 | P2: CI cobrando o que existe | Execute | Pending |
| CLEAN-35 | P2: CI cobrando o que existe | Execute | Pending |
| CLEAN-36 | P2: CI cobrando o que existe | Execute | Pending |
| CLEAN-37 | P2: CI cobrando o que existe | Execute | Pending |
| CLEAN-38 | P2: Gate de formatação utilizável | Execute | Pending |
| CLEAN-39 | P2: Gate de formatação utilizável | Execute | Pending |
| CLEAN-40 | P2: Gate de formatação utilizável | Execute | Pending |
| CLEAN-41 | P2: Gate de formatação utilizável | Execute | Pending |
| CLEAN-42 | P3: Working tree limpo | Execute | Pending |
| CLEAN-43 | P3: Working tree limpo | Execute | Pending |
| CLEAN-44 | P3: Working tree limpo | Execute | Pending |
| CLEAN-45 | P3: Working tree limpo | Execute | Pending |
| CLEAN-46 | P3: Working tree limpo | Execute | Pending |
| CLEAN-47 | P3: Skill canônica + stub + daemon | Execute | Pending |
| CLEAN-48 | P3: Skill canônica + stub + daemon | Execute | Pending |
| CLEAN-49 | P3: Skill canônica + stub + daemon | Execute | Pending |
| CLEAN-50 | P3: Skill canônica + stub + daemon | Execute | Pending |
| CLEAN-51 | P3: Skill canônica + stub + daemon | Execute | Pending |
| CLEAN-52 | P3: Skill canônica + stub + daemon | Execute | Pending |

**Coverage:** 52 total, 52 mapped to stories, 0 unmapped.

**Consolidation note:** IDs are assigned in document order, one per acceptance criterion. Each ID is verified by a named gate command recorded in `tasks.md`; the Verifier checks per-ID evidence rather than per-story.

---

## Success Criteria

- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` all exit 0 after the cleanup
- [ ] `go list ./...` reports 24 packages, none of them the six deleted ones
- [ ] `grep -rn "nomic-embed-text" README.md COMO_USAR.md COMO_FUNCIONA.md CONTRIBUTING.md docs/` returns zero hits
- [ ] `gofmt -l .` prints nothing on a fresh Windows checkout
- [ ] Every package from `go list ./...` is executed by CI or listed in a documented exclusion table
- [ ] `git status --porcelain` is clean at the end of the last commit