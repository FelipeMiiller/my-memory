# fix-release-onnx-provider Validation

**Date**: 2026-11-13
**Spec**: `.specs/features/fix-release-onnx-provider/spec.md`
**Diff range**: `b379271..317ce17` (branch `develop`, HEAD `317ce17`)
**Verifier**: independent sub-agent (author != verifier)
**Verdict**: FAIL (behavior correct; 4 evidence/documentation gaps to close)

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 | Done | Job `embedder-onnx` em `.github/workflows/ci.yml:385-413`. O comando exato do job foi executado localmente: exit 0, 55 testes PASS, 0 skip. |
| T2 | Done | `.github/workflows/release.yml:123` — `CGO_ENABLED=1 go build -tags "sqlite_fts5 onnx"`. |
| T3 | Done | `.github/workflows/release.yml:169-170` (build) + `:128-159` (passo "Verify C Toolchain (Windows)" com `::error::Toolchain C ausente:` e `exit 1`). |
| T4 | Done | `internal/embedder/model_downloader.go` (novo, 547 linhas). Tamanho conferido antes do rename em `:308-311` e `:438-441`. |
| T5 | Done | `cmd/mem/embed.go` (novo) + wiring em `cmd/mem/main.go:840-844`; ajuda em `cmd/mem/main.go:874-877`. |
| T6 | Done | `scripts/install.ps1:192` (`& $TargetBin embed download`), `:203` (diretório do modelo). Script parse sem erro. |
| T7 | Done | `scripts/install.sh:291` (`"$TARGET_BIN" embed download`), `:298` (diretório do modelo). `bash -n` passa no conteúdo commitado. |
| T8 | Done | `cmd/mem/main.go:1492-1506` (`warnProviderUnavailable`), contador em `:1779`/`:1978`/`:2012` e `:2026`/`:2227`/`:2261`. Aviso por documento removido de `:1941-1944` e `:2187-2190`. |
| T9 | Done | `cmd/mem/main.go:1508-1516` (`warnLexicalFallback`), call sites em `:2344`, `:2348`, `:2493`, `:2497`. |

Nenhuma tarefa marcada como parcial ou bloqueada. Todos os 9 "Done when" de `tasks.md` conferem no código, com uma ressalva registrada abaixo (T7: `bash -n` passa no blob commitado; a cópia do working tree local está com CRLF obsoleto).

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| AC1 — release Linux | tag `onnx` + `sqlite_fts5` + `CGO_ENABLED=1` | `.github/workflows/release.yml:123` — `CGO_ENABLED=1 go build -tags "sqlite_fts5 onnx"` | PASS (inspeção do artefato; sem teste automatizado — `tasks.md` matrix atribui "none" à camada de pipeline) |
| AC2 — release Windows | tag `onnx`; falha explícita nomeando o toolchain C ausente | `.github/workflows/release.yml:170` — `go build -tags "sqlite_fts5 onnx"`; `release.yml:152-155` — `::error::Toolchain C ausente: ...` + `exit 1` | PASS (inspeção do artefato) |
| AC3 — job de CI com a tag | compila `cmd/mem` com a tag e roda a suíte do provider | `.github/workflows/ci.yml:385-413` — job `embedder-onnx`; `:408` build, `:413` `go test -tags "sqlite_fts5 onnx" -v -count=1 ./internal/embedder/...`. Comando executado localmente: **exit 0, 55 PASS, 0 skip** | PASS (artefato + execução do comando da CI) |
| AC4 — artefato traz o provider; `mem search` reporta a fonte `vector` | artefato contém o provider; a fonte RRF `vector` aparece | `.github/workflows/release.yml:123,170` (tag). Evidência empírica: binário com `-tags "sqlite_fts5 onnx"` → `mem embed doctor` = "Status: ATIVO"; `mem search --mode hybrid` → `📊 Fontes RRF: [fts:1, vector:1]` | PASS (verificado por execução end-to-end; **sem teste de regressão automatizado** — gap 3) |
| AC5 — `mem embed download` baixa ORT 1.29.0 + modelo para `~/.memory/models/`, conferindo o tamanho antes do move | tamanho conferido antes do rename; versão 1.29.0; destino `~/.memory/models/` | `internal/embedder/model_downloader_test.go:65-91` — `Size: int64(len(content)) + 100` ⇒ erro + `assertNoLeftovers`; `model_downloader.go:308-311` (tamanho antes do rename em `:320`); `model_downloader.go:438-441` (tamanho do pacote antes da extração) | PARTIAL — tamanho/rename cobertos; **"1.29.0" (`model_downloader.go:23`) e "`~/.memory/models/`" (`:117-135`) não têm asserção em nenhum `_test.go`** (gap 2) |
| AC6 — falha de download nomeia URL e destino, sem parcial | erro contém URL + destino; diretório de destino sem arquivos | `internal/embedder/model_downloader_test.go:84-90` — `strings.Contains(err.Error(), url)` + `strings.Contains(err.Error(), dest)` + `assertNoLeftovers(t, dir)`; repetido em `:114-120` (sha256) e `:141-147` (HTTP 404) | PASS |
| AC7 — instalador invoca `mem embed download` e imprime o diretório do modelo | invocação + diretório impresso | `scripts/install.ps1:192` — `& $TargetBin embed download`; `scripts/install.ps1:203` — `Write-Host "   Diretório do modelo: ..."`; `scripts/install.sh:291` — `"$TARGET_BIN" embed download`; `scripts/install.sh:298` — `printf "   Diretório do modelo: %s\n"` | PASS com ressalva — o caminho é **hard-coded**, não o que o download usou (gap 5) |
| AC8 — `mem embed doctor` reporta provider, paths, dimensão e existência | 5 itens reportados | `cmd/mem/embed_test.go:66-78` — `strings.Contains(relatorio, "builtin")`, `"384"`, `manifest.Model.Dest`, `manifest.Runtime.Dest`, `"ATIVO"`; `:79-83` — `a+" — presente"`; `embed_test.go:124-126` — `" — ausente"` | PASS |
| AC9 — `mem index` imprime UM aviso consolidado que nomeia a tag e a consequência | 1 aviso por execução (não 1 por documento), nomeando a tag e a consequência | `cmd/mem/embed_provider_degradation_test.go:31-33` — `strings.Count(out, avisoDegradacao) != 1` ⇒ erro; `:34-36` — `strings.Contains(out, "tag `onnx`")`; `:37-39` — `strings.Contains(out, "SEM VETOR")`; `:46-48` — `strings.Contains(out, "sem provider de embedding;")` ⇒ o aviso por documento não existe mais. Empírico (binário sem a tag): 1 aviso para 3 documentos | PASS (⚠️ gap de precisão: o parêntese da spec diz "chunks", a implementação e o teste contam **documentos** — gap 6) |
| AC10 — `mem search --mode hybrid` declara o fallback léxico | a saída declara que o resultado é só do fallback textual | `cmd/mem/embed_provider_degradation_test.go:216-218` — `strings.Contains(out, "LEXICAL")`; `:219-221` — `strings.Contains(out, "NÃO usou similaridade vetorial")`; `:224-226` — non-vacuous, o documento aparece. Empírico: `⚠️ Resultado apenas LEXICAL — ...` + `Fontes RRF: [fts:1]` | PASS (⚠️ gap de precisão: a spec não define o wording — gap 7) |
| AC11 — `mem index` sai com código 0 e diz quantos documentos ficaram sem vetor | exit code 0 **e** contagem de documentos sem vetor | contagem: `cmd/mem/embed_provider_degradation_test.go:40-42` — `strings.Contains(out, "3 documento(s)")`. exit code: `embed_provider_degradation_test.go:124-137` — a closure retorna `nil` hard-coded, logo `if err != nil` (`:135`) **nunca pode falhar** | PARTIAL — contagem PASS; **exit code não é asseverado** (mutante F sobreviveu; gap 1) |

**Status**: 9/11 ACs com resultado PASS, 2/11 parciais (AC5, AC11), 0 FAIL de comportamento.
**Gaps de precisão da spec**: 2 (AC9 — unidade chunk vs documento; AC10 — wording do aviso não especificado).

---

## Discrimination Sensor

Executado em worktree descartável (`git worktree add D:\.minimax\scratch\verifier-onnx-ac 317ce17`), nunca na árvore real. `git stash` **não** foi usado. Baseline da árvore real antes e depois: `git status --porcelain` vazio (idêntico), HEAD `317ce17`, `git stash list` vazio, worktree removido com `git worktree remove --force`.

| Mutation | File:line | Description | Killed? |
| -------- | --------- | ----------- | ------- |
| A | `internal/embedder/model_downloader.go:308` | Verificação de tamanho desativada (`if a.Size > 0 && written != a.Size` → `if false`) | Killed — `model_downloader_test.go:82` "DownloadArtifact deveria falhar quando o tamanho diverge do esperado" |
| B | `internal/embedder/model_downloader.go:281,304,309,315,321` | Os 5 `os.Remove(tmpName)` dos caminhos de erro de `DownloadArtifact` removidos (parcial fica no disco) | Killed — 3 testes: `model_downloader_test.go:90`, `:120`, `:147` ("o diretório de destino deveria estar vazio, tem [.model_int8.onnx.partial...]") |
| C | `cmd/mem/main.go:1514-1516` | `warnLexicalFallback` não imprime | Killed — `embed_provider_degradation_test.go:217` ("a saída não declara que o resultado é apenas léxico") |
| D | `cmd/mem/main.go:1501-1506` | `warnProviderUnavailable` não imprime | Killed — `embed_provider_degradation_test.go:139` ("o aviso de degradação apareceu 0 vezes, deveria aparecer 1") |
| E | `cmd/mem/main.go:1672` | `runIndexCLI` aborta com erro quando `emb == nil` (exit ≠ 0) | Killed — `embed_provider_degradation_test.go:139` e `:142` (nada indexado, aviso ausente, banco não criado) |
| F | `cmd/mem/main.go:1711` | `runIndexCLI` devolve erro **depois** de indexar tudo (exit ≠ 0, sem alterar nenhuma outra saída) | **Survived na superfície de teste da feature** — `go test -run TestProviderDegradation ./cmd/mem/...` → `ok` (exit 0). Só foi morto por testes pré-existentes fora do escopo da feature: `cmd/mem/index_code_scope_test.go:101` e `:169` (`if err := runIndexCLI(...); err != nil { t.Fatalf }`) |

**Sensor depth**: lightweight estendido (6 mutações, ≥ os 1-3 exigidos; a task pedia 2-4).
**Result**: 5/6 mortos na superfície da feature; 1 sobrevivente (F) → **fix task** (gap 1).
Nota: o comportamento do exit code está correto (verificado empiricamente: binário sem a tag, `mem index` → exit 0). O que falta é a **asserção** que o amarre.

---

## Interactive UAT Results

Não executado (feature de backend/infraestrutura; sem comportamento de interface para UAT humano). A verificação empírica substituiu o UAT: binários com e sem a tag foram executados contra um vault temporário em `D:\.minimax\scratch\`.

---

## Code Quality

| Principle | Status |
| ---------------- | ------ |
| Minimum code | OK — `warnProviderUnavailable` e `warnLexicalFallback` são 5 linhas cada; o downloader não tem abstração para uso único |
| Surgical changes | OK — `cmd/mem/builtin_embedder.go` só delega para `embedder.DefaultModelsDir()`/`DefaultORTLibPath()`; o resto do arquivo não foi tocado |
| No scope creep | OK — nenhum commit fora dos 9 arquivos de implementação + 3 de teste + 2 de workflow + 2 de instalador |
| Matches patterns | OK — o padrão de aviso único por comando segue o estilo de `fmt.Printf` do CLI existente |
| Spec-anchored outcome check (asserted values match spec) | **NO** — AC5 (versão 1.29.0 e destino `~/.memory/models/`) e AC11 (exit code 0) não têm asserção que aponte para o resultado definido na spec |
| Per-layer Coverage Expectation met | **NO** — a camada "Provider com a tag `onnx`" da matrix de `tasks.md` declara cobrir AC4 em `internal/embedder/builtin_test.go`, mas esse arquivo é pré-existente, não foi alterado pela feature e não assereva a fonte RRF `vector`; e ele dá `t.Skip` quando o modelo/DLL não estão no disco (caso da CI) |
| Every test maps to a spec requirement - no unclaimed tests | OK — os 18 testes novos mapeiam: 9 → AC5/AC6, 4 → AC8/AC6, 5 → AC9/AC10/AC11 |
| Documented guidelines followed | **NO** — `AGENTS.md` regra 7 exige atualizar `README.md`, `docs/CLI_GUIDE.md`, `docs/ARCHITECTURE.md` e registrar ADR; `docs/CENTRAL_VAULT.md`/`docs/CLI_GUIDE.md`/`docs/ARCHITECTURE.md` ainda descrevem o aviso por documento **removido** por esta feature; `README.md` não lista `mem embed download`/`mem embed doctor`; `CHANGELOG.md` e `.specs/STATE.md` não foram tocados |

Testes não frágeis (spot-check, AC9/AC10): o teste de degradação tem âncora non-vacuous (`embed_provider_degradation_test.go:224-226` exige que o documento apareça, para que um fallback quebrado não passe) e o par "com provider não declara fallback" (`:232-253`) cobre o outro lado do contrato.

---

## Edge Cases

| # | Edge case (spec) | Verificado | Evidência |
| - | ---------------- | ---------- | --------- |
| 1 | Toolchain C ausente no runner do release ⇒ falha com mensagem que nomeia o toolchain (nunca publicar binário mudo) | OK | `release.yml:128-159` — passo "Verify C Toolchain (Windows)": procura `gcc`, tenta `C:\msys64\ucrt64\bin\gcc.exe` e `C:\msys64\mingw64\bin\gcc.exe`, e se não achar emite `::error::Toolchain C ausente: ...` e `exit 1` (`release.yml:153-154`). Linux: `release.yml:99-101` instala `gcc` + `libsqlite3-dev` |
| 2 | `~/.memory/models/` não gravável ⇒ reporta o caminho que não conseguiu gravar e sai com código ≠ 0 | OK (sem teste automatizado) | Empírico: `mem embed download` com diretório sob um arquivo → `Erro: download falhou para https://huggingface.co/... (destino D:\...\models\model_int8.onnx): não foi possível criar o diretório: ...`, **exit 1**. Código: `model_downloader.go:269-271` (MkdirAll) e `:273-276` (CreateTemp) |
| 3 | Modelo existe mas a biblioteca do ONNX Runtime falta ⇒ erro nomeando a biblioteca, nunca "vetor zero silencioso" | OK | `internal/embedder/builtin.go:58-60` — `builtin: onnxruntime não encontrado em %q — use `+"`mem embed download`"+``. Empírico: `mem embed doctor` → `Status: INDISPONÍVEL — builtin: onnxruntime não encontrado em "...onnxruntime.dll"`; `mem index` → aviso consolidado `2 documento(s) indexado(s) SEM VETOR`. Teste pré-existente `internal/embedder/builtin_test.go:189` é fraco (aceita os dois desfechos) e não faz parte da feature |
| 4 | Download interrompido no meio ⇒ nenhum arquivo parcial no diretório de destino (temp + rename atômico) | OK | `model_downloader.go:273` (temp no mesmo diretório) + `:320` (`os.Rename`); mutante B matado por `assertNoLeftovers` (`model_downloader_test.go:22-35`) em 3 testes |
| 5 | Vault sem documentos ⇒ `mem embed doctor` ainda reporta o status do provider | OK | `cmd/mem/embed.go:99-121` — o doctor não toca o vault (diagnóstico é do ambiente). `cmd/mem/embed_test.go:47-84` monta um manifesto sem vault e ainda exige "ATIVO" |

---

## Gate Check

- **Gate command** (Full, de `tasks.md`): `$env:MY_MEMORY_PG_TEST='1'` + `MY_MEMORY_PG_URL` do config global (endpoint direto, sem `-pooler`) + `go -C D:\repository\my-memory test -count=1 ./...`
- **Result**: **1052 testes executados — 1051 passaram, 0 falharam, 1 pulado** (213 subtestes também passaram). Exit code 0. 27 pacotes `ok`, 0 `FAIL`.
- **Skipped**: 1 — `TestAuditSubscriber_ReadOnlyFilesystem_FailsClosed` (`internal/.../audit_subscriber_test.go:108`: "skipping read-only FS test on Windows (chmod semantics differ)"). Limitação de plataforma, teste pré-existente, fora do escopo da feature.
- **Test count before feature**: 1034 · **after**: 1052 · **Delta**: **+18** (9 em `internal/embedder/model_downloader_test.go`, 4 em `cmd/mem/embed_test.go`, 5 em `cmd/mem/embed_provider_degradation_test.go`). Nenhum teste existente foi alterado, removido ou enfraquecido — `git diff --name-status b379271..317ce17 -- '*_test.go'` mostra só 3 arquivos **novos** (A/A/A).
- **Build gate**: `go build ./...` exit 0 · `go vet ./...` exit 0 · `gofmt -l` nos 7 arquivos Go tocados → **vazio** (saída limpa).
- **Tagged gate**: `CGO_ENABLED=1 go build -tags "sqlite_fts5 onnx" -o bin/mem-verify.exe ./cmd/mem` → exit 0 (37.127.881 bytes).
- **Tagged test gate**: `go test -tags "sqlite_fts5 onnx" -v -count=1 ./internal/embedder/...` → **exit 0, 55 PASS, 0 skip** (os artefatos do provider estão instalados nesta máquina, então os testes reais do ONNX rodaram, não pularam). `go test -tags "sqlite_fts5 onnx" -run 'TestProviderDegradation|TestEmbed' ./cmd/mem/...` → exit 0.
- **Failures**: nenhuma.

---

## Fix Plans

### Fix 1 — AC11: asseverar o exit code 0 do `mem index` sem provider

- **Root cause**: `cmd/mem/embed_provider_degradation_test.go:124-137` — a closure aninhada devolve `nil` literal, então `err` é sempre `nil` e `if err != nil` (`:135`) é uma asserção vacuamente verdadeira. O valor de retorno de `runIndexCLI` é descartado.
- **Evidência**: mutante F (`cmd/mem/main.go:1711` → devolve erro após indexar tudo) **sobreviveu** a `go test -run TestProviderDegradation ./cmd/mem/...` (exit 0). O `main` traduz erro em `os.Exit(1)` em `cmd/mem/main.go:91-95`, então a violação passaria sem o teste.
- **Fix task**: capturar o valor de retorno real de `runIndexCLI` e asseverar `err == nil`; opcionalmente rodar o binário construído e asseverar o exit code do processo.
- **Priority**: Major (integridade de teste; o comportamento em si está correto).

### Fix 2 — AC5: asseverar a versão 1.29.0 e o destino `~/.memory/models/`

- **Root cause**: `internal/embedder/model_downloader.go:23` (`ORTVersion = "1.29.0"`), `:45-46` (URLs base), `:69-98` (assets por GOOS/GOARCH), `:117-135` (`DefaultModelsDir`, `DefaultORTDir`) — nenhum `_test.go` do repositório referencia `DefaultManifest`, `DefaultModelsDir`, `DefaultORTDir`, `ORTVersion` ou `1.29.0`.
- **Evidência**: grep em `**/*_test.go` → 0 ocorrências. O risco residual é baixo (o downloader e o provider compartilham as funções de path — `cmd/mem/builtin_embedder.go:12-26`), mas a spec fixa os dois valores com precisão e nada os fixa em teste.
- **Fix task**: teste que assereve `ORTVersion == "1.29.0"`, que `DefaultModelsDir()` termina em `.memory/models/<ModelID>` sob o home, que `DefaultORTDir()` é `.memory/models/onnxruntime`, e que as URLs de `DefaultManifest` contêm `v1.29.0`.
- **Priority**: Major (evidence-or-zero em dois pontos precisos da spec).

### Fix 3 — AC4: teste de regressão para a fonte RRF `vector`

- **Root cause**: a matrix de `tasks.md` (linha 29) afirma que AC4 é coberto por `internal/embedder/builtin_test.go`, mas esse arquivo é pré-existente (não aparece no diff de `b379271..317ce17`), só assereva vetores 384d do provider e dá `t.Skip` quando o modelo/DLL não estão no disco — o caso da CI.
- **Evidência**: `grep '"vector:' **`/*_test.go`** → 1 ocorrência, em `internal/mcp/handlers_test.go:433` (mock). Nenhum teste do fluxo real de `mem search` verifica a fonte `vector`. Verificado só por execução manual: `Fontes RRF: [fts:1, vector:1]`.
- **Fix task**: teste de integração que indexa um vault com vetor e assereva a fonte `vector` nas `Sources` do resultado da busca (o caminho sem provider já está coberto por `embed_provider_degradation_test.go:202-227`).
- **Priority**: Major (evidence-or-zero).

### Fix 4 — documentação: reconciliar o aviso por documento removido e listar os comandos novos

- **Root cause**: a feature removeu o aviso `sem provider de embedding; <docID> indexado em modo léxico FTS` (`cmd/mem/main.go:1941-1944`, `:2187-2190`) e o substituiu por um aviso consolidado (`cmd/mem/main.go:1505`), mas a documentação continua descrevendo o comportamento antigo.
- **Evidência**: `docs/CLI_GUIDE.md:1647`, `docs/CENTRAL_VAULT.md:122`, `:132`, `:296`, `docs/ARCHITECTURE.md:475` descrevem o aviso por documento; `README.md` não lista `mem embed download`/`mem embed doctor` (`cmd/mem/main.go:874-877`); `CHANGELOG.md` e `.specs/STATE.md` não foram tocados em `b379271..317ce17`. Viola `AGENTS.md` regras 7 e 8.
- **Fix task**: atualizar os quatro documentos, o `README.md`, o `CHANGELOG.md` e o `.specs/STATE.md`; marcar os 2 Goals restantes e os 4 Success Criteria em `spec.md` (hoje ainda `[ ]`, embora implementados).
- **Priority**: Major (regra 7/8 do `AGENTS.md`).

### Fix 5 — AC7: o instalador imprime o diretório que o download usou, não um caminho fixo

- **Root cause**: `scripts/install.ps1:203` e `scripts/install.sh:298` imprimem o caminho default hard-coded. Com `MY_MEMORY_MINILM_MODEL_DIR` definido (`internal/embedder/model_downloader.go:118-120`), o download vai para outro lugar e o instalador ainda imprime o default — o oposto de "SHALL print the model directory it used".
- **Fix task**: derivar o caminho de `mem embed download` (que já imprime o destino real em `cmd/mem/embed.go:87-89`) ou de `MY_MEMORY_MINILM_MODEL_DIR`.
- **Priority**: Minor.

---

## Requirement Traceability Update

`spec.md` não foi editado por este Verifier (read-only). Status a aplicar pelo implementador:

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| ROP-01 (AC1) | Implementando | Verified |
| ROP-02 (AC2) | Implementando | Verified |
| ROP-03 (AC3) | Implementando | Verified |
| ROP-04 (AC4) | Implementando | Verified (sem teste de regressão — gap 3) |
| ROP-05 (AC5) | Implementando | Needs Fix (gap 2) |
| ROP-06 (AC6) | Implementando | Verified |
| ROP-07 (AC7) | Implementando | Needs Fix (gap 5, Minor) |
| ROP-08 (AC8) | Implementando | Verified |
| ROP-09 (AC9) | Implementando | Verified |
| ROP-10 (AC10) | Implementando | Verified |
| ROP-11 (AC11) | Implementando | Needs Fix (gap 1) |

---

## Summary

**Overall**: Issues — o comportamento está correto e comprovado, mas 4 lacunas de evidência/documentação impedem marcar a feature como concluída.

**Spec-anchored check**: 9/11 ACs PASS (2 PARTIAL: AC5, AC11) · 2 gaps de precisão da spec (AC9, AC10)
**Sensor**: 6 mutações injetadas, 5 mortas, **1 sobreviveu** na superfície da feature (mutante F)
**Gate**: 1052 testes — 1051 passaram, 0 falharam, 1 pulado (justificado) · build/vet/gofmt/build-tagged/test-tagged todos verdes

**O que funciona**
- O binário com a tag `onnx` embute o provider de verdade: `mem embed doctor` → "Status: ATIVO", `mem search` → `Fontes RRF: [fts:1, vector:1]`.
- Sem a tag, a degradação é explícita e consolidada: 1 aviso para 3 documentos, com a tag ausente nomeada e a contagem; `mem search --mode hybrid` declara "Resultado apenas LEXICAL"; `mem search --mode vector` sai com código 1; `mem index` sai com código 0.
- O downloader é verificado por tamanho e digest antes do rename atômico, não deixa parcial, e nomeia URL + destino no erro.
- Os instaladores invocam `mem embed download` sem abortar a instalação quando o download falha.

**Issues encontrados**
1. AC11 exit code não asseverado (asserção vacuamente verdadeira) — mutante F sobreviveu. Fix 1.
2. AC5 sem teste para "1.29.0" e "`~/.memory/models/`". Fix 2.
3. AC4 sem teste de regressão para a fonte RRF `vector`; a matrix de `tasks.md` superdeclara a cobertura. Fix 3.
4. Documentação contradiz o comportamento novo (`docs/CLI_GUIDE.md:1647`, `docs/CENTRAL_VAULT.md:122,132,296`, `docs/ARCHITECTURE.md:475`); `README.md`/`CHANGELOG.md`/`.specs/STATE.md` não atualizados. Fix 4.
5. AC7 imprime caminho hard-coded em vez do diretório usado. Fix 5.
6. AC9 diz "chunks" e AC11 diz "documentos"; a implementação conta documentos. Gap de precisão da spec, a favor da feature.
7. AC10 não fixa o wording do aviso; o teste assereve strings escolhidas pelo implementador. Gap de precisão da spec.

**Next steps**: despachar Fix 1-4 como tarefas (fixes de teste/documentação; nenhuma mudança de comportamento), re-verificar. As lacunas 5-7 podem ser fechadas no mesmo lote ou aceitas explicitamente.