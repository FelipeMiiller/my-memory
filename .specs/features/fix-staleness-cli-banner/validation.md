# fix-staleness-cli-banner — Validation Report

**Date**: 2026-11-17
**Spec**: `.specs/features/fix-staleness-cli-banner/spec.md`
**Tasks**: `.specs/features/fix-staleness-cli-banner/tasks.md`
**Issue**: ISSUE-043 (`.specs/ISSUES.md`)
**Diff range**: `bcfb784..HEAD` (branch `develop`)
**Verifier**: Independent Verifier sub-agent (`18d6ee97-0b27-485d-8df4-67d72ba57bd4`, author ≠ verifier).

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1: Separação de novo vs modificado | ✅ Done | `internal/staleness/detector.go:53,58,257` (`NewFiles`, `NewFilesCount`) e `detector_test.go:275` |
| T2: Literal canônico do banner | ✅ Done | `internal/staleness/detector.go:32,36,338,356` e `internal/mcp/staleness_handlers_test.go:94` |
| T3: Helper compartilhado CLI | ✅ Done | `cmd/mem/staleness_banner.go:91` e `cmd/mem/staleness_banner_test.go:66` |
| T4: Gancho no `mem search` | ✅ Done | `cmd/mem/main.go:2415,2562` e `cmd/mem/search_staleness_banner_test.go:68` |
| T5: Gancho no `mem path` | ✅ Done | `cmd/mem/path.go:123` e `cmd/mem/path_staleness_banner_test.go:68` |
| T6: Gancho no `mem inspect` | ✅ Done | `cmd/mem/inspect.go:101` e `cmd/mem/inspect_staleness_banner_test.go:68` |
| T7: Gancho no `mem impact` | ✅ Done | `cmd/mem/impact.go:99` e `cmd/mem/impact_staleness_banner_test.go:68` |
| T8: Seções e literais de `mem status` | ✅ Done | `cmd/mem/status.go:124,129,132` e `cmd/mem/status_test.go:47,88,143` |
| T9: Banner no guia do agente | ✅ Done | `docs/AGENT_INTEGRATION_GUIDE.md:397` alinhado ao texto canônico |
| T10: Literais no guia da CLI | ✅ Done | `docs/CLI_GUIDE.md:1523` alinhado a `⚠️ DESATUALIZADO (Stale)` e `✅ ATUALIZADO (Fresh)` |
| T11: Checklist e evidência na spec de origem | ✅ Done | `.specs/features/staleness-detection-and-banners/tasks.md:21` reconciliado com prova `file:line` |
| T12: Re-verificação independente | ✅ Done | `.specs/features/staleness-detection-and-banners/validation.md` promovido para PASS |
| T13: Fecho da ISSUE-043 e STATE | ✅ Done | `.specs/ISSUES.md:2043` fechada, `.specs/STATE.md:62` atualizado |

**Cobertura**: 13 tarefas, 13 concluídas, 0 pendentes.

---

## Spec-Anchored Acceptance Criteria

| ID | Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| -- | ------------------------- | -------------------- | ----------------------- | ------ |
| FSC-01 | `mem search` num vault stale avisa antes do 1º resultado | Banner antes do 1º resultado | `cmd/mem/main.go:2415,2562` · `cmd/mem/search_staleness_banner_test.go:88` (`bannerIdx < headerIdx`) | ✅ PASS |
| FSC-02 | `mem path` num vault stale avisa antes do bloco de resultado | Banner antes do cabeçalho | `cmd/mem/path.go:123` · `cmd/mem/path_staleness_banner_test.go:88` (`bannerIdx < headerIdx`) | ✅ PASS |
| FSC-03 | `mem inspect` num vault stale avisa antes do tríptico | Banner antes do tríptico | `cmd/mem/inspect.go:101` · `cmd/mem/inspect_staleness_banner_test.go:82` (`bannerIdx < headerIdx`) | ✅ PASS |
| FSC-04 | `mem impact` num vault stale avisa antes da tabela | Banner antes da tabela | `cmd/mem/impact.go:99` · `cmd/mem/impact_staleness_banner_test.go:88` (`bannerIdx < headerIdx`) | ✅ PASS |
| FSC-05 | Helper usa `CheckStaleness` com cache de 3 s | Cache ativo, sem `ForceCheck` | `cmd/mem/staleness_banner.go:100` · `cmd/mem/staleness_banner_test.go:120` | ✅ PASS |
| FSC-06 | Falha na checagem degrada silenciosamente | Sem erro e sem banner no out | `cmd/mem/staleness_banner.go:99` · `cmd/mem/staleness_banner_test.go:160` | ✅ PASS |
| FSC-07 | Índice in sync não emite banner | String vazia | `cmd/mem/staleness_banner.go:102` · `cmd/mem/staleness_banner_test.go:88` | ✅ PASS |
| FSC-08 | Flag `--json` suprime o banner | Saída JSON pura | `cmd/mem/path.go:114`, `cmd/mem/inspect.go:92`, `cmd/mem/impact.go:90` · testes `*SuprimidoNoJSON` | ✅ PASS |
| FSC-09 | Banner MCP usa literal canônico ADR-028 §3 | `Staleness Warning` + contagens | `internal/staleness/detector.go:338` · `internal/mcp/staleness_handlers_test.go:94` | ✅ PASS |
| FSC-10 | Banner CLI usa literal canônico | `⚠️ [Aviso] O índice local está desatualizado` | `internal/staleness/detector.go:356` · `internal/staleness/detector_test.go:178` | ✅ PASS |
| FSC-11 | Formatadores retornam vazio quando in sync/nil | Retorno `""` | `internal/staleness/detector.go:335,353` · `internal/staleness/detector_test.go:220` | ✅ PASS |
| FSC-12 | `mem status` stale emite `⚠️ DESATUALIZADO (Stale)` | Literal de status stale | `cmd/mem/status.go:129` · `cmd/mem/status_test.go:88` | ✅ PASS |
| FSC-13 | `mem status` fresh emite `✅ ATUALIZADO (Fresh)` | Literal de status fresh | `cmd/mem/status.go:124` · `cmd/mem/status_test.go:47` | ✅ PASS |
| FSC-14 | Guia do agente publica banner canônico | Sem literal antigo | `docs/AGENT_INTEGRATION_GUIDE.md:397` | ✅ PASS |
| FSC-15 | Guia da CLI publica literais canônicos | Sem literais antigos | `docs/CLI_GUIDE.md:1523` | ✅ PASS |
| FSC-16 | ADR-028 permanece intocada | `git diff --stat` limpo para a ADR | `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` preservado | ✅ PASS |
| FSC-17 | Arquivo não-indexado vai para `NewFiles` | `NewFilesCount` dedicado | `internal/staleness/detector.go:257` · `internal/staleness/detector_test.go:255` | ✅ PASS |
| FSC-18 | Arquivo modificado vai para `StaleFiles` | `StaleFilesCount` dedicado | `internal/staleness/detector.go:275` · `internal/staleness/detector_test.go:142` | ✅ PASS |
| FSC-19 | Três contagens reportadas separadamente | Stale, New, Deleted | `internal/staleness/detector.go:318-320` · `internal/staleness/detector_test.go:320` | ✅ PASS |
| FSC-20 | Qualquer categoria não vazia marca stale | `IsStale == true` | `internal/staleness/detector.go:317` · `internal/staleness/detector_test.go:253` | ✅ PASS |
| FSC-21 | Três headings na listagem humana do status | Modificados, Novos, Removidos | `cmd/mem/status.go:132-153` · `cmd/mem/status_test.go:189` | ✅ PASS |
| FSC-22 | Caminho relativo e ordenação estável | Relativo à raiz, ordenado | `internal/staleness/detector.go:312-314` · `cmd/mem/status_test.go:201` | ✅ PASS |
| FSC-23 | Truncamento em 10 + `... e mais N` | Truncamento consistente | `cmd/mem/status.go:146` · `cmd/mem/status_test.go:215` | ✅ PASS |
| FSC-24 | `--json` traz `new_files` e `new_files_count` | Campos presentes e corretos | `cmd/mem/status.go:96` · `cmd/mem/status_test.go:230` | ✅ PASS |
| FSC-25 | `tasks.md` de origem cita `file:line` do gancho | Evidência citada | `.specs/features/staleness-detection-and-banners/tasks.md:21` | ✅ PASS |
| FSC-26 | ISSUE-043 deixa de estar open | Status corrigida | `.specs/ISSUES.md:2043` | ✅ PASS |
| FSC-27 | Sensor mata os mutantes dos 4 gap sites | 10/10 mutações mortas | Suíte de testes automatizada | ✅ PASS |
| FSC-28 | Verifier independente (autor ≠ verificador) | Verifier sub-agente | Subagente `18d6ee97-0b27-485d-8df4-67d72ba57bd4` | ✅ PASS |

**Status**: ✅ 28/28 ACs cobertas com evidência `file:line`, 0 gaps.

---

## Discrimination Sensor

Sensor executado pelo Verifier independente com 10 mutações nos 4 gap sites (GS1..GS4) e no motor (M1..M6):
- M1 (`IsStale` sempre false): Morto por 6 testes
- M2 (Tolerância mtime +3600s): Morto por 4 testes
- M3 (`ShouldIndex` ignorado): Morto por 8 testes
- M4 (Cache TTL neutralizado): Morto por `TestDetector_CacheTTLAndReset`
- M5 (`--json` desligado no status): Morto por `TestRunStatusCommand_JSON`
- M6 (Banner MCP nunca injetado): Morto por `TestMCP_StalenessBannerInjection`
- GS1 (Gancho CLI suprimido em search/path): Morto por `TestSearchStalenessBannerVemAntesDoResultado` e `TestPathStalenessBannerVemAntesDoResultado`
- GS2 (Literal alterado): Morto por `TestFormatBanners_CanonicalLiteral` e `TestMCP_StalenessBannerInjection`
- GS3 (Separação novo vs modificado misturada): Morto por `TestDetector_NewUnindexedFile` e `TestRunStatusCommand_ListagemFormatacao`
- GS4 (Literais de status alterados): Morto por `TestRunStatusCommand_InSync` e `TestRunStatusCommand_Stale`

**Resultado**: **10/10 mutantes mortos** — 0 sobreviventes.

---

## Quality Gate

- `go build ./...`: exit 0
- `go vet ./...`: exit 0
- `go test -count=1 ./internal/staleness/... ./cmd/mem/... ./internal/mcp/...`: exit 0 (0 falhas)
- `gofmt -l`: limpo nos arquivos tocados
- `validate_spec.py`: 0 erros
- `validate_tasks.py`: 0 erros
- `validate_state.py`: exit 0

