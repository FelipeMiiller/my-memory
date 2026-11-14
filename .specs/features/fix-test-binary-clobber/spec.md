# Feature: fix-test-binary-clobber

- **Spec ID**: `fix-test-binary-clobber`
- **Created**: 2026-11-14
- **Owner**: Felipe Miiller
- **Source**: ISSUE-041 — a suíte de `cmd/mem` sobrescreve `bin/mem.exe` com um build sem a tag `onnx`
- **Related**: ADR-035 (o provider atrás da tag) · ADR-013 (o doctor) · ISSUE-037 (o aviso consolidado) · ISSUE-040 (chunk sem vetor real)

## Problem Statement

Dois helpers de teste do pacote `cmd/mem` compilam o binário **por cima do
binário do usuário**, sem a tag `onnx`:

| Onde | O que faz |
| :--- | :--- |
| `cmd/mem/doctor_test.go:108-111` | `go build -o <repoRoot>/bin/mem.exe ./cmd/mem` |
| `cmd/mem/events_test.go:37-40` | idem |

O efeito colateral é medido, não suposto (Cofre Central, 2026-11-14):

| # | Fato medido |
| :--- | :--- |
| 1 | `bin/mem.exe` compilado com `-tags "sqlite_fts5 onnx"` tem **37.191.134 bytes**; o `mem embed doctor` reporta `Status: ATIVO` |
| 2 | a suíte completa roda e sobrescreve o binário por um build **sem** a tag (**26.481.664 bytes**); o `mem embed doctor` passa a dizer `INDISPONÍVEL — este binário foi compilado sem a tag 'onnx'` |
| 3 | o `mem index` seguinte roda em **modo léxico**: os chunks do documento alterado são gravados **sem embedding** (ISSUE-040) |
| 4 | o `mem doctor` reporta **96/100** (2 chunks sem vetor real = 4 pontos) e **0 link rompido** |

O estrago é **anunciado** — o `mem index` imprime o aviso consolidado do
ISSUE-037 (`cmd/mem/main.go:1503`, que nomeia a tag ausente e a causa) e o
doctor lista os chunks sem vetor. O defeito **não** é o silêncio: é que a
**suíte de testes derruba o artefato que o dono usa**, trocando o binário
taggeado por um mudo, sem que a suíte tenha qualquer obrigação sobre isso.

> [!warning]
> O risco real não é o aviso que passa batido — é o **artefato trocado**. O
> dono compila `bin/mem.exe` com a tag (o `mem embed doctor` confirma
> `ATIVO`), roda a suíte por qualquer motivo, e o próximo `mem index` degrada
> o índice do cofre. O reparo é não tocar no binário do usuário.

## Goals

- [ ] A suíte de `cmd/mem` **não escreve** em `<repoRoot>/bin/`.
- [ ] O binário do usuário sobrevive à suíte **byte a byte**.
- [ ] O contrato do aviso consolidado (ISSUE-037) **não regride**.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| O `mem index` avisar que grava chunk sem vetor | **Já existe** (ISSUE-037): `warnProviderUnavailable` (`cmd/mem/main.go:1503`) imprime uma vez por execução, nomeia a tag, a causa e o diagnóstico. Verificado por `cmd/mem/embed_provider_degradation_test.go`. |
| O `mem embed doctor` distinguir "sem a tag" de "artefato faltando" | **Já existe**: o stub sem tag diz *"este binário foi compilado sem a tag `onnx`"*; o provider real diz que o modelo não foi encontrado. |
| Compilar o binário da suíte **com** a tag `onnx` | O `go test` padrão roda sem CGO e sem a toolchain C; exigir a tag quebraria a CI. E o binário da suíte **não precisa** da tag: os testes não dependem do provider vetorial. |
| Hash por chunk / chunking por âncora | É a **granularidade de update** (a pergunta do dono sobre "atualizar só aquele conteúdo"): pede âncora de conteúdo no chunk + `chunks.content_hash`. Trabalho de produto, em spec própria. |
| O `embed_epoch` e o auto-cura | **Já funciona** e foi provado neste caso: documento indexado em modo léxico fica com epoch `""`; quando o provider volta, `"" != epoch corrente` e o gate reprocessa **só** ele. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Onde o binário da suíte deve ficar | **`t.TempDir()`**, por teste | o comentário dos helpers diz *"Always rebuild to avoid stale-binary issues"* — o problema de binário velho se resolve compilando **novo** num lugar descartável, não sobrescrevendo o artefato do usuário | y |
| O binário do usuário pode não existir | o helper compila no temp e segue | um repo recém-clonado não tem `bin/`; o teste não pode depender de artefato externo | y |
| Os dois helpers devem virar um | **um** helper compartilhado (`memBinary(t)`), usado pelos dois | duplicar a lógica do build em dois lugares é o que criou a divergência; um helper só tem um caminho a consertar | y |
| O binário precisa estar no PATH do processo | não — é executado por caminho absoluto (`exec.Command(path, ...)`) | o helper já faz isso; compilar no temp não muda o contrato | y |

Open questions: none

## User Stories

### P1: A suíte não toca no binário do usuário ⭐ MVP

**User Story**: Como dono do cofre, quero que rodar os testes não substitua o
binário que eu uso, para que o próximo `mem index` não degrade o índice sem
que eu perceba.

**Why P1**: é a causa raiz medida — sem isso, o resto é sintoma.

**Acceptance Criteria**:

1. WHEN a suíte de `cmd/mem` roda THEN ela SHALL NOT escrever nenhum arquivo
   em `<repoRoot>/bin/`.
2. WHEN um teste precisa de um binário THEN o helper SHALL compará-lo num
   diretório temporário próprio (`t.TempDir()`), descartado no fim do teste.
3. WHEN a suíte roda THEN o arquivo `<repoRoot>/bin/mem.exe` (se existir) SHALL
   permanecer **byte a byte** igual (mesmo SHA-256 antes e depois).
4. WHEN um teste executa o binário THEN ele SHALL continuar executando-o por
   caminho absoluto, a partir do diretório de trabalho que o teste definir.

**Independent Test**: rodar `go test ./cmd/mem/...` e comparar o SHA-256 de
`bin/mem.exe` antes e depois — tem que ser o mesmo.

---

### P2: O contrato do aviso continua valendo

**User Story**: Como dono, quero que a degradação vetorial continue anunciada
por execução, para que eu saiba que o índice ficou sem vetor.

**Acceptance Criteria**:

5. WHEN o `mem index` roda sem provider e grava documento THEN ele SHALL
   imprimir o aviso consolidado **uma** vez por execução, nomeando a tag
   ausente e a contagem (contrato do ISSUE-037, preservado).
6. WHEN o `mem index` roda sem provider e **nada** é gravado THEN ele SHALL NOT
   imprimir o aviso.
7. WHEN o `mem index` roda **com** provider THEN ele SHALL NOT imprimir o
   aviso de degradação.

**Independent Test**: a suíte existente
(`cmd/mem/embed_provider_degradation_test.go`) cobre 5 e 6; a 7 é provada pelo
gate taggeado (`-tags "sqlite_fts5 onnx"`), onde o provider está disponível.

## Edge Cases

- IF `<repoRoot>/bin/` não existir THEN o helper compila no temp e segue — a
  suíte não cria diretório no repo.
- IF dois testes do pacote rodarem em paralelo THEN cada um tem seu
  `t.TempDir()` — nenhum binário é compartilhado.
- IF o `go build` falhar (sem toolchain C, sem toolchain Go) THEN o helper
  falha **o teste**, com a saída do build — nunca deixa um binário parcial no
  lugar de outro.
- IF o Windows exigir `.exe` THEN o helper continua resolvendo o sufixo pelo
  `runtime.GOOS` (o contrato atual).
- IF o `cmd/mymemoryd` tiver o mesmo defeito THEN ele precisa da mesma
  correção — **verificado: não tem** (só `cmd/mem` escreve em `bin/`).
- IF um teste depender do binário **taggeado** (com provider) THEN ele não
  pode depender do binário da suíte — a suíte compila sem tag por definição.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| TBC-01 | AC1 — a suíte não escreve em `bin/` | `cmd/mem/*_test.go` |
| TBC-02 | AC2 — o helper compila em `t.TempDir()` | `cmd/mem/events_test.go` (helper único) |
| TBC-03 | AC3 — o binário do usuário sobrevive byte a byte | teste de regressão em `cmd/mem` |
| TBC-04 | AC4 — o binário é executado por caminho absoluto, no cwd do teste | `cmd/mem/events_test.go` · `cmd/mem/doctor_test.go` |
| TBC-05 | AC5 — o aviso consolidado sai uma vez por execução | `cmd/mem/main.go:1503` (preservado) |
| TBC-06 | AC6 — sem documento gravado, sem aviso | `cmd/mem/embed_provider_degradation_test.go` (preservado) |
| TBC-07 | AC7 — com provider, sem aviso | gate taggeado |

**Coverage**: 7 total, 7 mapeados, 0 unmapped.

## Success Criteria

- [ ] `go test ./cmd/mem/...` deixa `bin/mem.exe` **byte a byte** igual.
- [ ] Nenhum arquivo de teste do repo escreve em `<repoRoot>/bin/`.
- [ ] A suíte de `cmd/mem` continua verde, e o gate completo (full + taggeado)
      também.
- [ ] O contrato do aviso consolidado do ISSUE-037 continua verde.
