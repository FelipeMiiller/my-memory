# staleness-detection-and-banners — Validation Report

**Date**: 2026-11-15
**Spec**: `.specs/features/staleness-detection-and-banners/spec.md`
**Diff range**: `9d3e0ee^..9e8cd05` — commits da feature: `9d3e0ee` (implementação) + `3ff46ce` (fix mtime). HEAD `9e8cd05`, branch `develop`, worktree limpo.
**Verifier**: sub-agente independente (author ≠ verifier). Nenhuma linha de código foi alterada pela verificação; a única escrita nesta árvore foi este arquivo.
**Verdict**: ❌ **FAIL** — 4/9 ACs cobertas, 5 gaps, 1 spec-precision gap. Gate 100% verde; sensor 6/6 mutantes mortos.

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 Motor de detecção (`internal/staleness/`) | ✅ Done | `internal/staleness/detector.go` — detector completo, cache TTL 3s, 2 formatadores. 10 testes. |
| T2 Persistência SQLite + PostgreSQL | ✅ Done | `internal/db/store.go:787`, `internal/store/postgres.go:1426`, interface `internal/store/store.go:274`. Teste PG opt-in roda de verdade (ver Gate). |
| T3 `mem status` + warnings CLI | ⚠️ **Partial** | `cmd/mem/status.go` completo e testado; **a integração do alerta nos comandos de consulta (`search`, `path`, `inspect`, `impact`) NÃO existe** — ver Gap #1. |
| T4 Banner no servidor MCP | ✅ Done | `internal/mcp/handlers.go:1047` injeta em 13 ferramentas de consulta. 2 testes. |
| T5 ADR-028 + documentação | ✅ Done | ADR-028, `docs/adr/README.md:38`, `docs/README.md:55`, `docs/CLI_GUIDE.md:1412`, `docs/AGENT_INTEGRATION_GUIDE.md:388`, `.specs/STATE.md:26`. |

---

## Spec-Anchored Acceptance Criteria

| # | Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| - | ------------------------- | -------------------- | ----------------------- | ------ |
| P1-1 | WHEN há arquivos modificados/novos/deletados THEN as respostas MCP incluem o banner `> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (...)` | Prefixo literal **`> ⚠️ **Staleness Warning**`** + menção a `mem index`/`mem watch` | `internal/mcp/staleness_handlers_test.go:94` — `strings.Contains(toolResStale.Content[0].Text, "AVISO: Memória Desatualizada")` | ❌ **GAP** — o banner **é** injetado, mas o literal asseverado não é o do spec. Código produz `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**` (`internal/staleness/detector.go:302`) e não menciona `mem watch`. O ADR-028 (Accepted) também especifica o literal inglês (`docs/adr/028-...md:45`) — o código diverge do spec **e** do ADR. |
| P1-2 | WHEN o índice está sincronizado THEN nenhum banner é injetado | Banner ausente | `internal/mcp/staleness_handlers_test.go:77` — `if strings.Contains(..., "AVISO: Memória Desatualizada") { t.Errorf(...) }`; `internal/staleness/detector_test.go:83` — `if banner := FormatMarkdownBanner(rep); banner != ""` | ✅ **PASS** |
| P1-3 | WHEN a varredura roda THEN ela respeita estritamente `ShouldIndex` (ignora `.git`, `.obsidian`, `node_modules`) | Arquivo elegível fora da regra não é contado | `internal/staleness/detector_test.go:284` — `if rep.DiskCount != 0 { t.Errorf("Arquivo dentro de .git não deveria ser contado...") }`; skip por `config.DefaultExcludedDirs` em `internal/staleness/detector.go:181-192`; `ShouldIndex` para `.obsidian`/`node_modules` em `internal/config/glob_test.go:58-61` | ✅ **PASS** (nota: o teste *do detector* cobre só `.git`; `.obsidian`/`node_modules` estão cobertos no nível de `ShouldIndex`, não do detector) |
| P2-1 | WHEN `mem status` roda THEN exibe contagem de arquivos no disco vs indexados | Duas contagens no output | `cmd/mem/status_test.go:49` — `strings.Contains(out, "Arquivos no Disco:  1") && strings.Contains(out, "Arquivos Indexados: 1")` | ✅ **PASS** |
| P2-2 | WHEN `mem status` roda THEN lista detalhada de arquivos **modificados, novos e deletados** com seus caminhos relativos | Três categorias listadas, caminhos relativos | `cmd/mem/status.go:132-160` (implementação) — **nenhuma asserção de teste** | ❌ **GAP** — (a) nenhum teste assevera a lista no output humano (`cmd/mem/status_test.go:86-92` só verifica a linha de status e a recomendação); (b) o modelo de dados não separa "novo" de "modificado": `StalenessReport` só tem `StaleFiles`/`DeletedFiles` (`internal/staleness/detector.go:31-32`), e o heading CLI é "Arquivos Modificados ou Não-Indexados" (`cmd/mem/status.go:133`) — os "novos" ficam misturados nos "modificados". ⚠️ Spec-precision gap: o spec não define como as três categorias são representadas. |
| P2-3 | WHEN há desatualização THEN exibe status `⚠️ DESATUALIZADO (Stale)` | Literal `⚠️ DESATUALIZADO (Stale)` | `cmd/mem/status_test.go:87` — `strings.Contains(out, "⚠️  Desatualizado (Stale Data)")` | ❌ **GAP** — literal divergente do spec (implementação: `cmd/mem/status.go:129`) |
| P2-4 | WHEN está 100% alinhado THEN exibe status `✅ ATUALIZADO (Fresh)` | Literal `✅ ATUALIZADO (Fresh)` | `cmd/mem/status_test.go:46` — `strings.Contains(out, "✅ Atualizado (In Sync)")` | ❌ **GAP** — literal divergente do spec (implementação: `cmd/mem/status.go:124`) |
| P2-5 | WHEN `--json` é passado THEN retorna o objeto `StalenessReport` completo em JSON | JSON desserializável em `StalenessReport` | `cmd/mem/status_test.go:116` — `json.Unmarshal(buf.Bytes(), &rep)`; `:120` `rep.IsStale`; `:123` `rep.StaleFilesCount`; `:126` `rep.DiskCount`/`rep.IndexedCount` | ✅ **PASS** (parcial — 4 dos 8 campos do struct asseverados; `StaleFiles`, `DeletedFiles`, `LastCheck` não são) |
| P3-1 | WHEN há desatualização THEN `mem search` / `mem path` emitem aviso claro no cabeçalho do stdout antes dos resultados | Aviso no stdout nos comandos de consulta | **sem evidência** — `FormatCLIBanner` existe em `internal/staleness/detector.go:309` mas **não tem nenhum chamador em produção** (grep do repo inteiro: só a definição e `internal/staleness/detector_test.go:86,141`). `cmd/mem/main.go:203` (search), `cmd/mem/path.go`, `cmd/mem/inspect.go`, `cmd/mem/impact.go` não instanciam detector nem imprimem aviso | ❌ **GAP (Blocker)** — história P3 inteira não implementada, embora `tasks.md` marque `[x] Integrar alerta de staleness nos comandos de consulta` |

**Status**: ❌ **4/9 ACs PASS** (P1-2, P1-3, P2-1, P2-5) · **5 GAP** (P1-1, P2-2, P2-3, P2-4, P3-1) · **1 spec-precision gap** (P2-2)

### Goals

| Goal | Result | Evidência |
| ---- | ------ | --------- |
| G1 motor de detecção `internal/staleness/` | ✅ | `internal/staleness/detector.go:131` (`computeStaleness`), modified `:244`, new `:227`, deleted `:272` |
| G2 cache em memória TTL 3s | ✅ | `internal/staleness/detector.go:22` (`DefaultTTL = 3 * time.Second`), cache `:91-123` |
| G3 `mem status` com `--json` | ✅ | `cmd/mem/status.go:19-163` |
| G4 banners MCP **+** alertas CLI | ❌ parcial | MCP ✅ (`internal/mcp/handlers.go:1047`); **CLI ❌** (P3-1) |
| G5 ADR-028 + guias | ✅ | `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md`, `docs/CLI_GUIDE.md:1412`, `docs/AGENT_INTEGRATION_GUIDE.md:388` |

---

## Discrimination Sensor

Execução em worktree isolado `git worktree add D:\.minimax\scratch\subagents\verifier-staleness\wt HEAD`. **`git stash` nunca usado.** Árvore real verificada como `CLEAN` antes e depois (porcelain idêntico ao baseline).

| # | Mutação (comportamento) | Arquivo:linha | Descrição | Morto? |
| - | ------------------------ | ------------- | --------- | ------ |
| M1 | `IsStale` sempre falso | `internal/staleness/detector.go:286` | `len(staleFiles) > 0 \|\| len(deletedFiles) > 0` → `false` (banner nunca injetado quando há desatualização) | ✅ **Morto** — 6 testes caem em 3 pacotes: `TestDetector_ModifiedFile`, `TestDetector_NewUnindexedFile`, `TestDetector_DeletedFile`, `TestDetector_UsesPersistedMTimeNotIndexTime`, `TestMCP_StalenessBannerInjection`, `TestRunStatusCommand_Stale` |
| M2 | Tolerância de mtime absurda | `internal/staleness/detector.go:244` | `mtime > persistedMTime+1` → `mtime > persistedMTime+3600` (arquivo modificado nunca detectado) | ✅ **Morto** — `TestDetector_ModifiedFile`, `TestDetector_UsesPersistedMTimeNotIndexTime`, `TestDetector_LegacyRowWithZeroMTimeFallsBack` |
| M3 | Regra `ShouldIndex` ignorada | `internal/staleness/detector.go:196` | `if !d.cfg.ShouldIndex(relPath)` → `if false && !d.cfg.ShouldIndex(relPath)` (`.git`/`.obsidian`/`node_modules` passam a contar) | ✅ **Morto** — 9 testes, incluindo `TestDetector_ExcludedDirs` |
| M4 | Cache neutralizado | `internal/staleness/detector.go:119` | `d.cacheUntil = time.Now().Add(ttl)` → `d.cacheUntil = time.Now()` (TTL deixa de valer) | ✅ **Morto** — `TestDetector_CacheTTLAndReset` |
| M5 | Saída `--json` desligada | `cmd/mem/status.go:97` | `if *jsonOutput {` → `if *jsonOutput && false {` (relatório JSON nunca emitido) | ✅ **Morto** — `TestRunStatusCommand_JSON` |
| M6 | Banner MCP nunca injetado | `internal/mcp/handlers.go:1047` | `if det != nil && isQueryTool(...)` → `if false && det != nil && isQueryTool(...)` | ✅ **Morto** — `TestMCP_StalenessBannerInjection` |

**Profundidade**: heavyweight (6 mutações de comportamento, ≥5 exigido para P0/caminhos críticos) — o feature toca integridade de dados (índice vs disco), tratado como caminho crítico.
**Resultado**: **6/6 mutantes mortos — 0 sobreviventes.** ✅ PASS

> Nota de método: a primeira tentativa de M5 (`if *jsonOutput {` → `if false {`) produziu **erro de compilação** (`jsonOutput declared and not used`), o que mataria o mutante por build quebrado e não por asserção. A mutação foi reformulada para `if *jsonOutput && false {` — o mutante só é considerado morto quando o **teste** falha, não o compilador.

---

## Interactive UAT

**Não executado.** Este agente é um sub-agente filho e não pode solicitar interação ao usuário final. O gate automatizado cobriu o comportamento observável, mas **o gap P3-1 (ausência de aviso na CLI) só seria percebido por um UAT humano** rodando `mem search` num vault desatualizado.

---

## Code Quality

| Check | Pass? | Observação |
| ----- | ----- | ---------- |
| No features beyond what was asked | ⚠️ | `FormatCLIBanner` (`internal/staleness/detector.go:309`) foi construído mas nunca ligado a nenhum comando — é exatamente o tipo de função morta que o próprio repo poda no commit `06de7ec` ("remove as 14 funcoes provadamente sem chamada"). O problema é **feature faltando**, não feature sobrando. |
| No abstractions for single-use code | ✅ | `Detector`/`StalenessReport`/`DocumentMeta` coerentes; `SetTTL`/`ForceCheck`/`ResetCache` todos usados. |
| No unnecessary "flexibility" added | ✅ | Flags `--db`/`--postgres`/`--repo`/`--dir`/`--json` todas exercitadas. |
| Only touched files required for task | ✅ | Superfície: `internal/staleness/`, `cmd/mem/status*.go`, `internal/mcp/handlers.go`, `internal/db/store.go`, `internal/store/{store,postgres}.go` + docs/ADR. |
| Didn't "improve" unrelated code | ✅ | Nenhuma refatoração alheia detectada. |
| Matches existing patterns/style | ✅ | MADR em ADR-028, comentários em PT-BR, erros com `fmt.Errorf` e `%w`, alinhadocom `internal/config` e `internal/db`. |
| Would senior engineer approve? | ❌ | Não aprovaria o merge com P3-1 aberto: o `tasks.md` afirma `[x] Integrar alerta de staleness nos comandos de consulta (search, path, inspect, impact)` e isso **não existe em código**. |
| Tests map to acceptance criteria and are non-shallow | ❌ | P2-2 e P3-1 **não têm teste nenhum**; P1-1/P2-3/P2-4 asseveram literais diferentes dos do spec. |
| Spec-anchored outcome check (asserted values match spec) | ❌ | 4 ACs asseveram literais que o spec não define (P1-1, P2-3, P2-4) ou nada (P2-2, P3-1). |
| Per-layer Coverage Expectation (domínio 1:1 AC; rotas happy+edge+error) | ❌ | Domínio `internal/staleness` tem 1:1 bom; camada CLI não cobre a rota de alerta (P3-1) nem a listagem de arquivos (P2-2). |
| Every test maps to a spec AC / edge case / Done-when (no unclaimed tests) | ✅ | `TestDetector_NoDatabase` (erro sem base), `TestDetector_UnchangedFileIsNotStale`, `TestDetector_LegacyRowWithZeroMTimeFallsBack` (fallback mtime=0), `TestMCP_StalenessCacheResetOnWrite` (invalidação reativa) mapeiam a robustez/edge cases do design; `TestSQLite_GetDocumentsMetadata` e o bloco novo em `TestPostgresStore_Integration` mapeiam T2. |
| Documented project quality/testing guidelines followed | ✅ | `.agents/rules/always-test.md` (suíte + gate), `always-quality-gate.md` (build/vet/gofmt), `always-update-docs-and-specs-on-pr.md` (docs/ADR/STATE atualizados). |

---

## Edge Cases

O `spec.md` **não tem seção de edge cases** — a lista abaixo vem do `tasks.md` e do plano de validação substituído.

- [x] Sem base configurada → erro explícito, sem pânico — `internal/staleness/detector.go:140`, `internal/staleness/detector_test.go:36`
- [x] Banco vazio + arquivo novo em disco → stale, `DiskCount=1`, `IndexedCount=0` — `internal/staleness/detector_test.go:171-179`
- [x] Documento no índice mas ausente do disco → deletado, `DeletedFilesCount=1` — `internal/staleness/detector_test.go:208`
- [x] Linha legada com `mtime = 0` → fallback para `updated_at`, sem falso positivo — `internal/staleness/mtime_detector_test.go:109-159`
- [x] Arquivo editado *antes* da indexação e de novo *depois*, ambas anteriores ao `updated_at` → detectado via `mtime` persistido — `internal/staleness/mtime_detector_test.go:19-67`
- [x] Escrita via MCP invalida o cache do detector — `internal/mcp/staleness_handlers_test.go:102-152` + `internal/mcp/handlers.go:1011-1018`
- [x] Vault sincronizado → nenhum banner (MCP e formatadores) — `internal/mcp/staleness_handlers_test.go:77`, `internal/staleness/detector_test.go:83-88`
- [ ] **Detecção de "arquivo novo" vs "arquivo modificado" na saída** → não separável no modelo de dados (P2-2)
- [ ] **Aviso de desatualização nos comandos de consulta da CLI** → ausente (P3-1)

---

## Gate Check

`tasks.md` **não tem seção "Gate Check Commands"** → conforme o procedimento, roda-se a suíte completa.

- **Gate command**: `go test -count=1 ./...`
- **Result**: **28/28 pacotes `ok`, 0 failed, 0 skipped** — exit code **0**
- **Pacotes**: `cmd/mem` (146.8s), `cmd/mymemoryd`, `internal/{autowire,canvas,ci,codeast,compiler,config,db,deeplink,drift,embedder,event_runtime,federation,graph,graphview,mcp,parser,pgtest,repo,staleness,store,supervisor,turboquant,vaultfs,watcher}`
- **Skipped tests**: nenhum (o único `t.Skip` do escopo — `internal/staleness/mtime_detector_test.go:25` — só dispara em ambiente sem CGO; aqui `CGO_ENABLED=0` cai no driver puro Go `modernc.org/sqlite`, então o skip **não** disparou)
- **Testes opt-in de PostgreSQL** (pulados por padrão na suíte hermética): `MY_MEMORY_PG_TEST=1` → `TestPostgresStore_Integration` **RODOU DE VERDADE** (`=== RUN` / `--- PASS: TestPostgresStore_Integration (15.75s)`), exit **0**. Isso cobre a asserção de `GetDocumentsMetadata` no Postgres em `internal/store/postgres_test.go:281-290`. *A URL de conexão foi lida de `~/.memory/config.yaml` e mantida em variável de ambiente; não foi impressa, logada nem gravada.*
- **`go build ./...`** → exit **0**
- **`go vet ./...`** → exit **0**
- **Test count antes da feature**: 15 funções de teste adicionadas por `9d3e0ee`+`3ff46ce` no escopo staleness, + `TestSQLite_GetDocumentsMetadata` (`internal/db/store_test.go:363`, provado por `git log -S`) + 1 bloco de asserção novo dentro de `TestPostgresStore_Integration` → **16 funções de teste novas + 1 bloco**; total do repo 252 → 1136 funções de teste no HEAD
- **Integridade do teste**: nenhuma função de teste foi removida pela feature; nenhuma asserção foi afrouxada (as asserções divergentes do spec são **defeito da feature**, não regressão)
- **Falhas**: nenhuma

---

## Fix Plans

### Fix 1 (Blocker): P3-1 — alerta de staleness ausente nos comandos de consulta da CLI

- **Root cause**: `FormatCLIBanner` (`internal/staleness/detector.go:309`) foi implementado e testado em unidade, mas o gancho de integração nos comandos de consulta nunca foi escrito. `cmd/mem/main.go:203` (`search`), `cmd/mem/path.go`, `cmd/mem/inspect.go`, `cmd/mem/impact.go` nunca constroem um `staleness.Detector`; a única construção no CLI é a de `cmd/mem/status.go:75-90`. O `tasks.md` marca a integração como `[x]`, o que mascarou o gap.
- **Fix task**:
  - *What*: instanciar o detector nos comandos `search`, `path`, `inspect`, `impact` e imprimir `staleness.FormatCLIBanner(report)` no topo do stdout **antes** dos resultados, quando `report.IsStale`. Usar o detector já cacheado (TTL 3s) e `CheckStaleness` (não `ForceCheck`) para não pagar I/O a cada invocação; degradar silenciosamente (sem banner) se a checagem falhar, para não quebrar o comando.
  - *Where*: `cmd/mem/main.go` (bloco `case "search"`), `cmd/mem/path.go`, `cmd/mem/inspect.go`, `cmd/mem/impact.go`.
  - *Verify*: teste que injete um detector com relatório stale e afirme que a saída contém o banner **antes** do marcador de resultado; e que a saída **não** contém o banner quando o detector está fresh.
  - *Done when*: `mem search`/`mem path` num vault desatualizado imprimem o aviso no cabeçalho do stdout; `tasks.md` T3 passa a refletir o que existe.
- **Priority**: **Blocker**

### Fix 2 (Major): P1-1 — literal do banner diverge do spec e do ADR-028

- **Root cause**: o spec (`spec.md:48`) e o ADR-028 Accepted (`docs/adr/028-...md:45`) especificam o banner `> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (...)` e a recomendação `Execute mem index ou inicie mem watch`. A implementação em `internal/staleness/detector.go:302` emite `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**` e só cita `mem index`. Em vez de ajustar o código, a documentação foi reescrita para acompanhar a implementação (`docs/AGENT_INTEGRATION_GUIDE.md:397`).
- **Fix task**: decidir **um** literal canônico e fazer spec, ADR, código, docs e teste concordarem. Duas opções válidas: (a) ajustar o código para o literal do ADR-028 (incluindo `mem watch`); ou (b) amending ADR-028 com decisão explícita de tradução do banner para PT-BR e atualizar o spec com `// SPEC_DEVIATION` registrado. Em qualquer opção, o teste MCP (`internal/mcp/staleness_handlers_test.go:94`) deve asseverar o literal **canônico**, não o acidental.
- **Verify*: asserção do literal canônico em `internal/mcp/staleness_handlers_test.go` e `internal/staleness/detector_test.go:137`.
- **Done when**: o literal do banner em `detector.go`, `spec.md`, ADR-028 e `docs/AGENT_INTEGRATION_GUIDE.md` é idêntico.
- **Priority**: **Major**

### Fix 3 (Major): P2-2 — lista de arquivos modificados/novos/deletados não é testada e não separa "novos"

- **Root cause**: `StalenessReport` (`internal/staleness/detector.go:25-34`) tem apenas `StaleFiles` e `DeletedFiles`; arquivos novos e modificados caem na mesma lista (`internal/staleness/detector.go:227` e `:245`). O heading CLI (`cmd/mem/status.go:133`) reflete essa mistura. O spec (`spec.md:56`) pede "modificados, novos e deletados". Além disso, nenhum teste assevera o bloco de listagem do output humano.
- **Fix task*: (a) decidir no spec como "novo" é representado (campo dedicado `NewFiles` no `StalenessReport` + heading próprio no CLI, **ou** remoção explícita do requisito de categoria "novos"); (b) adicionar asserção de teste sobre a listagem em `cmd/mem/status_test.go` (caminho relativo presente, ordenação estável, truncamento em 10 com "e mais N").
- **Priority**: **Major**

### Fix 4 (Minor): P2-3 / P2-4 — literais de status divergem do spec

- **Root cause**: o spec define `⚠️ DESATUALIZADO (Stale)` e `✅ ATUALIZADO (Fresh)`; a implementação emite `⚠️  Desatualizado (Stale Data)` (`cmd/mem/status.go:129`) e `✅ Atualizado (In Sync)` (`cmd/mem/status.go:124`).
- **Fix task*: alinhar código e spec (mesma decisão do Fix 2 — literal canônico único, validado por asserção de teste em `cmd/mem/status_test.go:46` e `:87`).
- **Priority**: **Minor**

---

## Requirement Traceability Update

*Proposta — `spec.md` **não** foi modificado: a regra de isolamento desta verificação permite escrita apenas em `validation.md`.*

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| G1 motor de detecção | Implementing | ✅ Verified |
| G2 cache TTL 3s | Implementing | ✅ Verified |
| G3 `mem status` + `--json` | Implementing | ✅ Verified |
| G4 banners MCP + alertas CLI | Implementing | ❌ Needs Fix (P3-1) |
| G5 ADR-028 + docs | Implementing | ✅ Verified |
| P1-1 banner MCP (literal do spec) | Implementing | ❌ Needs Fix (Fix 2) |
| P1-2 banner omitido quando fresh | Implementing | ✅ Verified |
| P1-3 respeita `ShouldIndex` | Implementing | ✅ Verified |
| P2-1 contagem disco vs índice | Implementing | ✅ Verified |
| P2-2 lista detalhada c/ caminhos relativos | Implementing | ❌ Needs Fix (Fix 3) |
| P2-3 status "DESATUALIZADO (Stale)" | Implementing | ❌ Needs Fix (Fix 4) |
| P2-4 status "ATUALIZADO (Fresh)" | Implementing | ❌ Needs Fix (Fix 4) |
| P2-5 `--json` com `StalenessReport` | Implementing | ✅ Verified |
| P3-1 aviso no stdout da CLI | Implementing | ❌ Needs Fix (Fix 1) |

---

## Summary

**Overall**: ❌ **Not Ready**

**Spec-anchored check**: 4/9 ACs casam com o resultado definido no spec · 1 spec-precision gap
**Sensor**: 6/6 mutantes mortos (0 sobreviventes)
**Gate**: 28/28 pacotes `ok` (exit 0) · `go build` ✅ · `go vet` ✅ · integração Postgres opt-in ✅

**O que funciona**
- O motor de detecção (`internal/staleness/detector.go`) é sólido: modified/new/deleted detectados corretamente, tolerância de 1s, `ShouldIndex` respeitado, cache TTL 3s com invalidação reativa em escrita MCP.
- A banner MCP é injetada em 13 ferramentas de consulta e **omitida** quando o índice está sincronizado — ambos com teste.
- `mem status` funciona ponta a ponta, com `--json` desserializável em `StalenessReport`.
- A persistência extrai metadados em lote nos dois motores (SQLite e Postgres), e o caminho Postgres foi **realmente exercitado** (não pulado).
- O fix `3ff46ce` (comparar `mtime` persistido em vez de `updated_at`) corrigiu um bug real de falso-negativo e tem teste de regressão dedicado.

**Problemas encontrados**
1. **[Blocker] P3-1**: `mem search`/`mem path`/`mem inspect`/`mem impact` **não emitem nenhum aviso** de desatualização. `FormatCLIBanner` é código morto. O `tasks.md` marca essa integração como feita — o checkbox mente.
2. **[Major] P1-1**: o banner emitido não é o literal do spec nem o do ADR-028 Accepted; o teste assevera o literal acidental.
3. **[Major] P2-2**: a listagem de arquivos modificados/novos/deletados não tem asserção de teste, e "novo" não é distinguível de "modificado" no modelo de dados.
4. **[Minor] P2-3/P2-4**: os literais de status da CLI divergem do spec.
5. **[Processo]**: o `validation.md` original era um **plano** não executado — nunca houve verificação de evidência sobre esta feature até esta rodada.

**Próximos passos**: rotear Fix 1 (Blocker) e Fix 2 (Major) para um implementador; re-verificar após o fix (o ciclo máximo do skill é 3 iterações fix→re-verify antes de escalar ao usuário).