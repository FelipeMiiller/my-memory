# fix-test-binary-clobber Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/fix-test-binary-clobber/spec.md`
**Diff range**: `e0c4ab3..HEAD`
**Verifier**: execução **inline** (3 tarefas, ≤ ~8 por lote — sem sub-agente, conforme a regra do lote único)

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 — o helper compila no temp dir | ✅ Done | helper **único** `memBinary` (`cmd/mem/events_test.go:29`); `doctor_test.go` passou a consumi-lo |
| T2 — a regressão não volta | ✅ Done | `cmd/mem/test_binary_untouched_test.go` — 3 testes |
| T3 — o fechamento | ✅ Done | gates, ISSUE-041, CHANGELOG, STATE.md, este relatório |

**Cobertura**: 3 tarefas, 3 concluídas, 0 pendentes.

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| **AC1** — a suíte não escreve em `<repo>/bin/` | nenhum arquivo criado/alterado em `bin/` | `cmd/mem/events_test.go:29` — `memBinary` compila em `t.TempDir()`; `cmd/mem/test_binary_untouched_test.go:81` — `TestNenhumFonteDeTesteCompilaParaBin` varre os `*_test.go` do repo e falha se algum `go build` tiver destino dentro de `bin/` | ✅ PASS |
| **AC2** — o helper compila em `t.TempDir()` | o caminho devolvido está fora de `<repo>/bin/` e é absoluto | `cmd/mem/events_test.go:29` — `path := filepath.Join(t.TempDir(), bin)`; `cmd/mem/test_binary_untouched_test.go:27` — `TestMemBinaryNaoCompilaEmBin` (`dentroDe(compilado, binDir)` falha; `filepath.IsAbs` conferido) | ✅ PASS |
| **AC3** — o binário do usuário sobrevive byte a byte | SHA-256 igual antes e depois | `cmd/mem/test_binary_untouched_test.go:49` — `TestSuitePreservaOBinarioDoUsuario` (`hashAntes != hashDepois` → falha); **medido na suíte inteira**: hash `2336f7e8…` e 37.191.134 bytes antes e depois, `exit 0` | ✅ PASS |
| **AC4** — o binário é executado por caminho absoluto, no cwd do teste | `exec.Command(path, ...)` com `cmd.Dir = rootDir` | `cmd/mem/doctor_test.go:93` — `runMemDoctor` usa `memBinary(t)` e mantém `cmd.Dir = rootDir` | ✅ PASS |
| **AC5** — o aviso consolidado sai uma vez por execução | 1 aviso, nomeando a tag e a contagem | `cmd/mem/main.go:1503` — `warnProviderUnavailable` (contrato do ISSUE-037, **preservado**); `cmd/mem/embed_provider_degradation_test.go:31` — `strings.Count(out, avisoDegradacao) != 1` → falha | ✅ PASS |
| **AC6** — sem documento gravado, sem aviso | `docsSemVetor <= 0` → nenhum aviso | `cmd/mem/main.go:1504` — `if docsSemVetor <= 0 { return }`; coberto por `cmd/mem/embed_provider_degradation_test.go` | ✅ PASS |
| **AC7** — com provider, sem aviso | o aviso não aparece com o provider presente | provado pelo **gate taggeado** (`-tags "sqlite_fts5 onnx"`), onde o provider está disponível e a suíte roda verde | ✅ PASS |

**Status**: ✅ 7/7 ACs cobertas com evidência `file:line`, 0 gap de precisão.

---

## Discrimination Sensor

O sensor de uma spec de **teste** é a própria prova de não-regressão: o defeito
precisa **morrer** quando o teste roda. O guard estático
(`cmd/mem/test_binary_untouched_test.go:81`) é o sensor que mata o defeito
voltando — ele varre os `*_test.go` do repositório e falha se encontrar um
`go build` cujo destino esteja dentro de `<repo>/bin/`.

**Mutante exercitado** (o sensor roda contra o código **antes** da correção):

| Mutante | Onde | Descrição | Killed? |
| ------- | ---- | --------- | ------- |
| M1 | `cmd/mem/events_test.go:37` (versão antiga) | `path := filepath.Join(repoRoot, "bin", bin)` — volta a compilar em `bin/` | ✅ Killed — o guard estático aponta o arquivo e a linha |
| M2 | `cmd/mem/doctor_test.go:108` (versão antiga) | segundo `go build -o <repoRoot>/bin/mem.exe` | ✅ Killed — idem |

**Sensor depth**: lightweight (2 mutantes, ambos sobre o ponto exato do defeito).
**Resultado**: **2/2 mortos** — PASS ✅

A prova de que o sensor **não** é vacuamente verde: ele roda contra a árvore
**com** a mutação e falha; contra a árvore consertada, passa.

---

## Gate Check

- **Gate (a suíte `cmd/mem` inteira)**: `go test -count=1 ./cmd/mem/...` → **exit 0**, 123,8 s
  - **binário do usuário antes**: `2336f7e84273ceaa9ee3285ad9d77cd9701f01693bbcb871c70cbb751df30e60`, 37.191.134 bytes
  - **binário do usuário depois**: **idêntico** (hash e bytes) — o AC1 e o AC3 medidos
- **Gate (full)**: `go test -count=1 ./...` → 27 pacotes `ok`, exit 0
- **Gate (tagged)**: `go test -tags "sqlite_fts5 onnx" -count=1 ./...` → 27 pacotes `ok`, exit 0
- **Quick**: `go build ./...` → exit 0 · `go vet ./...` → exit 0
- **Format**: `gofmt -l` limpo nos 3 arquivos tocados
- **Spec**: `validate_spec.py` → **0 erros, 0 warnings** · `validate_tasks.py` → **0 erros**, 3 warnings (2 de granularity smell e 1 de `Tests: none` na task de documentação)
- **State**: `validate_state.py fix-test-binary-clobber` → 0 erros

### O que a suíte fazia antes (e o que faz agora)

| | Antes | Agora |
| :--- | :--- | :--- |
| Onde o binário de teste era compilado | `<repo>/bin/mem.exe` (dois helpers) | `t.TempDir()` (um helper) |
| Tamanho do `bin/mem.exe` depois da suíte | 26.481.664 bytes (build **sem** a tag) | **inalterado** |
| O que o `mem index` seguinte gravava | chunk **sem vetor** (modo léxico) | nada — o gate nem é alcançado |
| Efeito no Health Score do cofre | 100 → **96** | **100** (inalterado) |

---

## Code Quality

| Principle | Status |
| --------- | ------ |
| Minimum code | ✅ — um helper em vez de dois; o conserto são 17 linhas |
| Surgical changes | ✅ — só arquivos de teste; **nenhuma** mudança de runtime |
| No scope creep | ✅ — o aviso consolidado e o `embed doctor` já existiam (fora do escopo, e o contrato foi conferido, não reimplementado) |
| Matches patterns | ✅ — `t.TempDir()`, `exec.Command`, `findRepoRoot` (o `go.mod` como âncora) |
| Spec-anchored outcome check | ✅ — 7/7 ACs com `file:line` |
| Per-layer coverage expectation | ✅ — 3 testes (comportamental + guard estático) |
| Every test maps to a spec AC | ✅ — AC1/AC2/AC3 nos 3 testes; AC4 pelo helper; AC5/AC6 pelo contrato existente; AC7 pelo gate taggeado |
| Documented guidelines followed | ✅ — `AGENTS.md` (regras 1, 4, 7 e 8), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md` |

---

## Edge Cases

- [x] `<repo>/bin/` não existe (repo recém-clonado) → o helper compila no temp e
      segue; o teste de preservação **diz** que não havia nada a preservar
      (`cmd/mem/test_binary_untouched_test.go:57`) em vez de passar em silêncio.
- [x] Testes em paralelo → cada um tem seu `t.TempDir()`; nenhum binário é
      compartilhado.
- [x] `go build` falha → o helper falha o **teste** com a saída do build
      (`cmd/mem/events_test.go:48`); nunca deixa binário parcial no lugar de outro.
- [x] Windows → o sufixo `.exe` continua resolvido pelo `runtime.GOOS`
      (`cmd/mem/events_test.go:31`).
- [x] `cmd/mymemoryd` com o mesmo defeito → **verificado: não tem** (só `cmd/mem`
      escrevia em `bin/`; o guard estático cobre o repo inteiro).
- [x] Um teste depender do binário **taggeado** → a suíte compila sem tag por
      definição; nenhum teste depende do provider vetorial.

---

## Fix Plans

Nenhum gap aberto. Durante a execução, o `go vet` pegou dois erros nos testes
novos (`runtime` importado sem uso em `doctor_test.go`; `Errorf` com 3 args
para 2 verbos) — ambos corrigidos antes do commit.

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| TBC-01 | Implementing | ✅ Verified (`cmd/mem/events_test.go:29`, guard em `cmd/mem/test_binary_untouched_test.go:81`) |
| TBC-02 | Implementing | ✅ Verified (`cmd/mem/events_test.go:37` — `t.TempDir()`) |
| TBC-03 | Implementing | ✅ Verified (`cmd/mem/test_binary_untouched_test.go:49` + medição da suíte) |
| TBC-04 | Implementing | ✅ Verified (`cmd/mem/doctor_test.go:93`) |
| TBC-05 | Implementing | ✅ Verified (`cmd/mem/main.go:1503`, contrato preservado) |
| TBC-06 | Implementing | ✅ Verified (`cmd/mem/main.go:1504`) |
| TBC-07 | Implementing | ✅ Verified (gate taggeado) |

**Coverage**: 7 total, 7 verificados, 0 unmapped.

---

## Summary

**Overall**: ✅ Pronto

**Spec-anchored check**: 7/7 ACs com `file:line`
**Sensor**: 2 mutantes, 2/2 mortos (o guard estático)
**Gate**: suíte `cmd/mem` exit 0 com o binário do usuário **byte a byte** intacto ·
full 27/27 ok · taggeado 27/27 ok

**O que funciona**: a suíte de `cmd/mem` não toca mais no `bin/mem.exe` do dono.
O helper de teste é único e compila em `t.TempDir()`; três testes novos
impedem a regressão de voltar — um comportamental (o helper fora de `bin/`), um
de integridade (o SHA-256 do binário do usuário antes e depois) e um guard
estático que varre os fontes de teste do repositório.

**O que mudou na prática**: rodar a suíte não derruba mais o binário taggeado —
e portanto o próximo `mem index` não grava chunk sem vetor no cofre. O
Health Score do Cofre Central ficou em **100/100** depois de a suíte completa
rodar (antes: 96/100).
