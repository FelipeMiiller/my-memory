# staleness-detection-and-banners — Validation Report

**Date**: 2026-11-17
**Spec**: `.specs/features/staleness-detection-and-banners/spec.md`
**Correction Spec**: `.specs/features/fix-staleness-cli-banner/spec.md` (ISSUE-043)
**Diff range**: `9d3e0ee^..HEAD` — commits originais (`9d3e0ee`, `3ff46ce`) + commits da correção `fix-staleness-cli-banner` (`bcfb784`..`e090c2b`). HEAD `e090c2b`, branch `develop`, worktree limpo.
**Verifier**: sub-agente independente (author ≠ verifier). Nenhuma linha de código em produção foi alterada pela verificação; a única escrita nesta árvore é este relatório de validação.
**Verdict**: 🟢 **PASS** — 9/9 ACs cobertas com evidência `file:line`, 0 gaps, 0 spec-precision gaps. Gate 100% verde; sensor 10/10 mutantes mortos (M1..M6 originais + 4 gap sites mortos).

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 Motor de detecção (`internal/staleness/`) | ✅ Done | `internal/staleness/detector.go` — detector completo, cache TTL 3s, separação de 3 categorias disjuntas (modificados, novos, deletados), 2 formatadores canônicos. |
| T2 Persistência SQLite + PostgreSQL | ✅ Done | `internal/db/store.go:787`, `internal/store/postgres.go:1426`, interface `internal/store/store.go:274`. `GetDocumentsMetadata` cobrindo ambos os bancos. |
| T3 `mem status` + warnings CLI | ✅ Done | `cmd/mem/status.go` completo (3 categorias, caminhos relativos, ordenados, truncamento em 10) e alertas integrados nos comandos de consulta (`search`, `path`, `inspect`, `impact`) via helper `cmd/mem/staleness_banner.go:91`. Suprimido no `--json`. |
| T4 Banner no servidor MCP | ✅ Done | `internal/mcp/handlers.go:1047` injeta banner canônico em respostas de ferramentas de consulta. |
| T5 ADR-028 + documentação | ✅ Done | ADR-028 intocada (norma preservada), `docs/adr/README.md`, `docs/README.md`, `docs/CLI_GUIDE.md:1412`, `docs/AGENT_INTEGRATION_GUIDE.md:388`, `.specs/STATE.md`. |

---

## Spec-Anchored Acceptance Criteria

| # | Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| - | ------------------------- | -------------------- | ----------------------- | ------ |
| P1-1 | WHEN há arquivos modificados/novos/deletados THEN as respostas MCP incluem o banner `> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (...)` | Prefixo literal **`> ⚠️ **Staleness Warning**`** + contagens + menção a `mem index`/`mem watch` | `internal/staleness/detector.go:338-344`; `internal/mcp/staleness_handlers_test.go:94-106` — `canonicalBanner := "> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (1 arquivo(s) pendente(s): 1 modificado(s), 0 novo(s), 0 deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute \`mem index\` ou inicie \`mem watch\`."`; `strings.Contains(toolResStale.Content[0].Text, canonicalBanner)`; `internal/staleness/detector_test.go:169` `strings.TrimRight(md, "\n") == canonicalMarkdownBanner` | ✅ **PASS** |
| P1-2 | WHEN o índice está sincronizado THEN nenhum banner é injetado | Banner ausente | `internal/mcp/staleness_handlers_test.go:77` — `if strings.Contains(toolResSync.Content[0].Text, "Staleness Warning") { t.Errorf(...) }`; `internal/staleness/detector_test.go:86-91` e `:218-224` — `FormatMarkdownBanner(inSync) == ""` e `FormatCLIBanner(inSync) == ""`; `cmd/mem/staleness_banner_test.go:86-95` — `buf.Len() == 0` | ✅ **PASS** |
| P1-3 | WHEN a varredura roda THEN ela respeita estritamente `ShouldIndex` (ignora `.git`, `.obsidian`, `node_modules`) | Arquivo elegível fora da regra não é contado | `internal/staleness/detector.go:197-208,224-226`; `internal/staleness/detector_test.go:454-480` — `if rep.DiskCount != 0 { t.Errorf("Arquivo dentro de .git não deveria ser contado...") }`; `internal/config/glob_test.go:58-61` | ✅ **PASS** |
| P2-1 | WHEN `mem status` roda THEN exibe contagem de arquivos no disco vs indexados | Duas contagens no output | `cmd/mem/status.go:120-121`; `cmd/mem/status_test.go:50` — `strings.Contains(out, "Arquivos no Disco:  1") && strings.Contains(out, "Arquivos Indexados: 1")` | ✅ **PASS** |
| P2-2 | WHEN `mem status` roda THEN lista detalhada de arquivos **modificados, novos e deletados** com seus caminhos relativos e ordenados | Três categorias disjuntas listadas com contagens, caminhos relativos e ordenação estável | `internal/staleness/detector.go:57-59,312-325`; `cmd/mem/status.go:132-153`; `internal/staleness/detector_test.go:275-337` (`TestDetector_SeparatesNewModifiedAndDeleted`) assevera `StaleFiles`, `NewFiles`, `DeletedFiles` separadas; `cmd/mem/status_test.go:189-217` (`TestRunStatusCommand_ListagemFormatacao`) assevera headings `Arquivos Modificados (1):`, `Arquivos Novos (12):`, `Arquivos Removidos do Disco (1):`, caminhos relativos e truncamento em 10 + `... e mais 2 arquivo(s)` | ✅ **PASS** |
| P2-3 | WHEN há desatualização THEN exibe status `⚠️ DESATUALIZADO (Stale)` | Literal `⚠️ DESATUALIZADO (Stale)` | `cmd/mem/status.go:129`; `cmd/mem/status_test.go:88` — `strings.Contains(out, "⚠️ DESATUALIZADO (Stale)")` | ✅ **PASS** |
| P2-4 | WHEN está 100% alinhado THEN exibe status `✅ ATUALIZADO (Fresh)` | Literal `✅ ATUALIZADO (Fresh)` | `cmd/mem/status.go:124`; `cmd/mem/status_test.go:47` — `strings.Contains(out, "✅ ATUALIZADO (Fresh)")` | ✅ **PASS** |
| P2-5 | WHEN `--json` é passado THEN retorna o objeto `StalenessReport` completo em JSON com `new_files` e `new_files_count` | JSON desserializável com todas as categorias | `cmd/mem/status.go:96-100`; `cmd/mem/status_test.go:119-137` — `json.Unmarshal(buf.Bytes(), &rep)`, `rep.IsStale`, `rep.NewFilesCount == 1`, `rep.StaleFilesCount == 0`; `cmd/mem/status_test.go:220-239` assevera `new_files` (12), `stale_files` (1), `deleted_files` (1) | ✅ **PASS** |
| P3-1 | WHEN há desatualização THEN `mem search`, `mem path`, `mem inspect`, `mem impact` emitem aviso claro no cabeçalho do stdout antes dos resultados; e com `--json` o stdout é puramente JSON | Aviso no stdout nos comandos de consulta antes do 1º resultado, suprimido com `--json` | `cmd/mem/staleness_banner.go:91-112`; `cmd/mem/main.go:2415,2562`; `cmd/mem/path.go:123`; `cmd/mem/inspect.go:101`; `cmd/mem/impact.go:99`. Testes dedicados: `cmd/mem/staleness_banner_test.go:66-95`; `cmd/mem/search_staleness_banner_test.go:86-101` (`bannerIdx < headerIdx`); `cmd/mem/path_staleness_banner_test.go:68-111` (`bannerIdx < headerIdx` e suprimido no `--json`); `cmd/mem/inspect_staleness_banner_test.go:68-111` (`bannerIdx < headerIdx` e suprimido no `--json`); `cmd/mem/impact_staleness_banner_test.go:68-111` (`bannerIdx < headerIdx` e suprimido no `--json`) | ✅ **PASS** |

**Status**: 🟢 **9/9 ACs PASS** · **0 GAP** · **0 spec-precision gaps**

---

### Goals

| Goal | Result | Evidência |
| ---- | ------ | --------- |
| G1 motor de detecção `internal/staleness/` | ✅ Done | `internal/staleness/detector.go:131` (`computeStaleness`), modified `:274`, new `:257`, deleted `:302` |
| G2 cache em memória TTL 3s | ✅ Done | `internal/staleness/detector.go:22` (`DefaultTTL = 3 * time.Second`), cache `:120-146`, compartilhado na CLI em `cmd/mem/staleness_banner.go:66-82` |
| G3 `mem status` com `--json` | ✅ Done | `cmd/mem/status.go:19-156` (3 categorias, literais canônicos, `--json` completo) |
| G4 banners MCP **+** alertas CLI | ✅ Done | MCP: `internal/mcp/handlers.go:1047`; CLI: `cmd/mem/staleness_banner.go:91` acoplado em `search`, `path`, `inspect`, `impact` |
| G5 ADR-028 + guias | ✅ Done | `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` (norma preservada intacta), `docs/CLI_GUIDE.md:1412`, `docs/AGENT_INTEGRATION_GUIDE.md:388` |

---

## Discrimination Sensor

Execução em worktree isolado temporário (`git worktree add`). **`git stash` nunca usado.** Árvore real verificada como `CLEAN` antes e depois (`git status --porcelain` idêntico ao baseline).

| # | Mutação (comportamento) | Arquivo:linha | Descrição | Morto? |
| - | ------------------------ | ------------- | --------- | ------ |
| M1 | `IsStale` sempre falso | `internal/staleness/detector.go:317` | `len(staleFiles) > 0 \|\| len(newFiles) > 0 \|\| len(deletedFiles) > 0` → `false` | ✅ **Morto** — 6 testes caem em `internal/staleness`: `TestDetector_ModifiedFile`, `TestDetector_NewUnindexedFile`, `TestDetector_SeparatesNewModifiedAndDeleted`, `TestDetector_OnlyNewFilesIsStale`, `TestDetector_DeletedFile`, `TestDetector_UsesPersistedMTimeNotIndexTime` |
| M2 | Tolerância de mtime absurda | `internal/staleness/detector.go:274` | `mtime > persistedMTime+1` → `mtime > persistedMTime+3600` | ✅ **Morto** — 4 testes caem em `internal/staleness`: `TestDetector_ModifiedFile`, `TestDetector_SeparatesNewModifiedAndDeleted`, `TestDetector_UsesPersistedMTimeNotIndexTime`, `TestDetector_LegacyRowWithZeroMTimeFallsBack` |
| M3 | Regra `ShouldIndex` ignorada | `internal/staleness/detector.go:224` | `if !d.cfg.ShouldIndex(relPath)` → `if false && !d.cfg.ShouldIndex(relPath)` | ✅ **Morto** — 8 testes caem em `internal/staleness`, incluindo `TestDetector_ExcludedDirs` e `TestDetector_InSync` |
| M4 | Cache neutralizado | `internal/staleness/detector.go:146` | `d.cacheUntil = time.Now().Add(ttl)` → `d.cacheUntil = time.Now()` | ✅ **Morto** — `TestDetector_CacheTTLAndReset` falha na linha 441 |
| M5 | Saída `--json` desligada | `cmd/mem/status.go:97` | `if *jsonOutput {` → `if *jsonOutput && false {` | ✅ **Morto** — `TestRunStatusCommand_JSON` falha na linha 121 |
| M6 | Banner MCP nunca injetado | `internal/mcp/handlers.go:1047` | `if det != nil && isQueryTool(...)` → `if false && det != nil && isQueryTool(...)` | ✅ **Morto** — `TestMCP_StalenessBannerInjection` falha nas linhas 99 e 105 |
| GS1 | Gancho CLI removido | `cmd/mem/main.go:2562` e `cmd/mem/path.go:123` | Remoção de `printStalenessBanner` em `search` e `path` | ✅ **Morto** — `TestSearchStalenessBannerVemAntesDoResultado` falha na linha 93; `TestPathStalenessBannerVemAntesDoResultado` falha na linha 82 |
| GS2 | Literal do banner MCP/CLI alterado | `internal/staleness/detector.go:32` | `StalenessWarningTitle = "Staleness Warning"` → `"Staleness Alert"` | ✅ **Morto** — `TestFormatBanners_CanonicalLiteral` falha na linha 170; `TestMCP_StalenessBannerInjection` falha na linha 99 |
| GS3 | Separação "novo" vs "modificado" misturada | `internal/staleness/detector.go:257` | `newFiles = append(newFiles, displayPath)` → `staleFiles = append(staleFiles, displayPath)` | ✅ **Morto** — `TestDetector_NewUnindexedFile` falha na linha 255; `TestDetector_SeparatesNewModifiedAndDeleted` falha na linha 320; `TestRunStatusCommand_ListagemFormatacao` falha nas linhas 191, 194, 208, 216 e 232 |
| GS4 | Literais de status `mem status` alterados | `cmd/mem/status.go:124,129` | `✅ ATUALIZADO (Fresh)` → `✅ Atualizado (In Sync)` e `⚠️ DESATUALIZADO (Stale)` → `⚠️  Desatualizado (Stale Data)` | ✅ **Morto** — `TestRunStatusCommand_InSync` falha na linha 48; `TestRunStatusCommand_Stale` falha na linha 89 |

**Profundidade**: heavyweight / critical path (10 mutações de comportamento: 6 da rodada original + 4 dos gap sites).
**Resultado**: **10/10 mutantes mortos — 0 sobreviventes.** ✅ **PASS**

---

## Interactive UAT

Comportamento observável completamente validado de forma automatizada por testes de integração de ponta a ponta:
- `mem search` / `mem path` / `mem inspect` / `mem impact` emitem a linha canônica de aviso antes dos resultados em vault stale.
- `mem status` exibe literais canônicos e 3 seções formatadas com caminhos relativos e truncamento.
- `--json` permanece documento JSON estrito e sem contaminação de banner em todos os subcomandos.

---

## Code Quality

| Check | Pass? | Observação |
| ----- | ----- | ---------- |
| No features beyond what was asked | ✅ | Fechou cirurgicamente os 4 gaps da ISSUE-043 sem escopo não solicitado. |
| No abstractions for single-use code | ✅ | Helper compartilhado `printStalenessBanner` reutilizado nos 4 comandos de consulta da CLI. |
| No unnecessary "flexibility" added | ✅ | Sem flags supérfluas; reutilizou opções e conexões já abertas. |
| Only touched files required for task | ✅ | Escopo estrito em `internal/staleness`, `cmd/mem`, `internal/mcp`, guias e specs. ADR-028 intocada. |
| Didn't "improve" unrelated code | ✅ | Nenhuma alteração alheia ao escopo da feature. |
| Matches existing patterns/style | ✅ | Código alinhado com convenções Go, padrões do projeto e MADR. |
| Would senior engineer approve? | ✅ | 100% de conformidade: cobertura 1:1, literais canônicos unificados, sensor 10/10 morto. |
| Tests map to acceptance criteria and are non-shallow | ✅ | Testes asseveram literais exatos, contagens exatas e precedência de impressão no stdout. |
| Spec-anchored outcome check (asserted values match spec) | ✅ | Todas as 9 ACs possuem asserções correspondentes aos valores definidos na especificação. |
| Per-layer Coverage Expectation (domínio 1:1 AC; rotas happy+edge+error) | ✅ | Camada de domínio `internal/staleness`, camada de aplicação `internal/mcp` e CLI `cmd/mem` cobertas. |
| Every test maps to a spec AC / edge case / Done-when | ✅ | Todos os testes associados a critérios de aceitação e edge cases da spec. |
| Documented project quality/testing guidelines followed | ✅ | Conforme `AGENTS.md`, `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`. |

---

## Edge Cases

- [x] Sem base configurada → erro explícito, sem pânico — `internal/staleness/detector.go:140`, `internal/staleness/detector_test.go:36`
- [x] Banco vazio + arquivo novo em disco → stale, `DiskCount=1`, `IndexedCount=0`, categorizado em `NewFiles` — `internal/staleness/detector_test.go:227-270`
- [x] Documento no índice mas ausente do disco → deletado, `DeletedFilesCount=1` — `internal/staleness/detector_test.go:380-407`
- [x] Linha legada com `mtime = 0` → fallback para `updated_at`, sem falso positivo — `internal/staleness/mtime_detector_test.go:109-159`
- [x] Arquivo editado *antes* da indexação e de novo *depois* → detectado via `mtime` persistido — `internal/staleness/mtime_detector_test.go:19-67`
- [x] Escrita via MCP invalida o cache do detector — `internal/mcp/staleness_handlers_test.go:109-159`
- [x] Vault sincronizado → nenhum banner (MCP, CLI e formatadores) — `internal/mcp/staleness_handlers_test.go:77`, `internal/staleness/detector_test.go:86-91`, `cmd/mem/staleness_banner_test.go:86-95`
- [x] Detecção de "arquivo novo" vs "arquivo modificado" na saída → categorias disjuntas `NewFiles` e `StaleFiles` — `internal/staleness/detector.go:57-58`, `cmd/mem/status.go:150-151`, `cmd/mem/status_test.go:189-239`
- [x] Aviso de desatualização nos comandos de consulta da CLI antes do resultado — `cmd/mem/main.go:2415,2562`, `cmd/mem/path.go:123`, `cmd/mem/inspect.go:101`, `cmd/mem/impact.go:99`
- [x] Flag `--json` nos comandos de consulta suprime o banner preservando JSON válido — `cmd/mem/path.go:115-118`, `cmd/mem/inspect.go:95-97`, `cmd/mem/impact.go:90-94`, `cmd/mem/*_staleness_banner_test.go`
- [x] Falha na checagem degrada em silêncio sem interromper a consulta — `cmd/mem/staleness_banner.go:102`
- [x] Reutilização de cache em consultas sequenciais dentro do TTL (3s) — `cmd/mem/staleness_banner_test.go:97`

---

## Gate Check

- **Gate command**: `go test -count=1 ./internal/staleness/... ./internal/mcp/... ./cmd/mem/...`
- **Result**: **PASS** — exit code **0**
- **Build check**: `go build ./...` — exit code **0**
- **Vet check**: `go vet ./...` — exit code **0**
- **Worktree integrity**: baseline limpo, final limpo (`git status --porcelain` vazio) — sem `git stash`

---

## Fix Plans

Nenhum fix pendente. Todos os gaps anteriores (P1-1, P2-2, P2-3, P2-4, P3-1) foram completamente corrigidos pela spec de correção `fix-staleness-cli-banner` (ISSUE-043).

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| G1 motor de detecção | Verified | ✅ Verified |
| G2 cache TTL 3s | Verified | ✅ Verified |
| G3 `mem status` + `--json` | Verified | ✅ Verified |
| G4 banners MCP + alertas CLI | ❌ Needs Fix (P3-1) | ✅ Verified |
| G5 ADR-028 + docs | Verified | ✅ Verified |
| P1-1 banner MCP (literal canônico) | ❌ Needs Fix (Fix 2) | ✅ Verified |
| P1-2 banner omitido quando fresh | Verified | ✅ Verified |
| P1-3 respeita `ShouldIndex` | Verified | ✅ Verified |
| P2-1 contagem disco vs índice | Verified | ✅ Verified |
| P2-2 lista detalhada c/ caminhos relativos e 3 categorias | ❌ Needs Fix (Fix 3) | ✅ Verified |
| P2-3 status `⚠️ DESATUALIZADO (Stale)` | ❌ Needs Fix (Fix 4) | ✅ Verified |
| P2-4 status `✅ ATUALIZADO (Fresh)` | ❌ Needs Fix (Fix 4) | ✅ Verified |
| P2-5 `--json` com `StalenessReport` | Verified | ✅ Verified |
| P3-1 aviso no stdout da CLI antes do resultado | ❌ Needs Fix (Fix 1) | ✅ Verified |

---

## Summary

**Overall**: 🟢 **Ready (PASS)**

**Spec-anchored check**: 9/9 ACs casam com o resultado definido no spec · 0 gaps
**Sensor**: 10/10 mutantes mortos (0 sobreviventes)
**Gate**: `internal/staleness`, `internal/mcp`, `cmd/mem` `ok` (exit 0) · `go build` ✅ · `go vet` ✅

**Conclusão da Re-Verificação Independente**
A implementação da feature `staleness-detection-and-banners` e de sua correção `fix-staleness-cli-banner` atinge 100% de conformidade com todos os requisitos funcionais, critérios de aceitação e diretrizes arquiteturais. A ADR-028 permanece como a norma canônica intocada, o gancho da CLI opera de forma não-bloqueante e protegida sob `--json`, os literais de status e banners estão plenamente padronizados, e o sensor de discriminação comprovou a eficácia da suíte de testes em todos os cenários críticos.
