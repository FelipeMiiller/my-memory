# Feature: fix-release-onnx-provider

- **Spec ID**: `fix-release-onnx-provider`
- **Created**: 2026-11-13
- **Owner**: Felipe Miiller
- **Source**: ISSUE-037 — `.specs/ISSUES.md` (seção ISSUE-037)
- **Related**: ADR-035 (provider embutido ONNX MiniLM) · ADR-032 (instalador one-liner) · ADR-062 §C e §D-1 · ISSUE-040 (o vetor zero que este spec torna visível)

## Problem Statement

O provider de embedding embutido (ONNX MiniLM, ADR-035) mora atrás de
`//go:build onnx` (`internal/embedder/builtin.go:1`). Mas o **release** compila
com `-tags "sqlite_fts5"` (`.github/workflows/release.yml:118,131`) e a **CI**
não tem nenhum job com a tag `onnx` — o provider nunca é compilado nem testado
no pipeline. Todo binário oficial roda em FTS lexical e, quando reindexa uma
nota, grava **vetor zero** no lugar do vetor (`cmd/mem/main.go:1913`). O
sintoma é a busca semântica do cofre central **decai nota a nota, em silêncio**
(sem erro, sem aviso estruturado).

Há um segundo furo, do mesmo tamanho: mesmo com a tag, o provider precisa de
dois artefatos no disco — o modelo
(`~/.memory/models/paraphrase-multilingual-MiniLM-L12-v2/model_int8.onnx`,
118 MB) e a DLL do ONNX Runtime
(`~/.memory/models/onnxruntime/onnxruntime.dll`, 16 MB). Nenhum comando os
baixa: `internal/embedder/model_downloader.go` **nunca foi escrito** (a task
T11/T12 da spec `embedder-builtin-onnx-minilm` ficou `Pending`) e o instalador
(`scripts/install.ps1`, `scripts/install.sh`) só baixa o zip do release. Numa
máquina nova, o provider não funciona nem com o binário certo.

## Goals

- [x] O artefato publicado (Linux e Windows) compila com a tag `onnx`.
- [x] A CI compila e testa o provider com a tag `onnx` — o provider deixa de ser
      código que "ninguém nunca compilou".
- [ ] Existe um comando que baixa o modelo e a DLL do ONNX Runtime para
      `~/.memory/models/`, e o instalador o invoca.
- [ ] Quando o provider não está disponível, a degradação é **explícita e
      consolidada** — um aviso por execução, não uma linha por documento.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Trocar o modelo (MiniLM → outro) | O modelo é fixo pelo schema (`vector(384)`) e pela ADR-035. |
| Provider remoto (Ollama / API HTTP) | Contraria a doutrina "tudo local" da ADR-035 e da ADR-062 (DR-4). |
| Vendorizar a DLL e o modelo **dentro** do binário/release | 134 MB por artefato; o dono já decidiu que o instalador **baixa** os dois. |
| `embedder/ollama.go` e `embedder/tokenizer.go` (morto) | Débito de código morto (ISSUE aparte), não parte do caminho do provider. |
| O conteúdo do cofre central | O conteúdo é do dono; esta spec muda o que o **binário** faz com ele. |
| O Health Score do cofre | É a spec `fix-doctor-unindexed-links`. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| A tag `onnx` exige toolchain C (CGO) | o build **com** a tag só é possível com toolchain C; sem ele o build **falha** em vez de degradar | o release roda em runner Linux com gcc disponível; degradar em silêncio é exatamente o defeito que esta spec fecha (ADR-062, DR-2) | y |
| O runner Windows tem toolchain C | o job **verifica** a toolchain e **falha explicitamente** se ela não existir | não se pode afirmar o que o runner tem; a verificação é feita em vez de assumida | y |
| Onde a DLL e o modelo ficam | `~/.memory/models/onnxruntime/onnxruntime.dll` e `~/.memory/models/paraphrase-multilingual-MiniLM-L12-v2/` | é o que `cmd/mem/builtin_embedder.go:14-35` já procura; o comando de download precisa escrever exatamente nesses caminhos | y |
| Versão da DLL do ONNX Runtime | **1.29.0** (API version 29) | é a versão que o binding `onnxruntime_go` exige (ADR-035, "Requisitos de runtime descobertos no spike") | y |
| Quem baixa: o instalador ou o usuário | o instalador invoca `mem embed download`; o usuário pode rodar o mesmo comando à mão | o instalador é o caminho de máquina nova; o comando é o caminho de máquina que já tem o binário | y |
| O que acontece sem internet | o download falha com a URL tentada e o caminho de destino, e **não** deixa arquivo parcial | sem rede não há provider; a falha tem de ser explícita, não silenciosa | y |

Open questions: none

## User Stories

### P1: O binário publicado carrega o provider semântico ⭐ MVP

**User Story**: Como dono do cofre central, quero que o binário que eu instalo
traga o provider de embedding embutido, para que a busca semântica funcione sem
eu precisar recompilar nada na minha máquina.

**Why P1**: é a causa raiz medida do "a aplicação não está funcionando"
(ADR-062 §C): sem a tag, o vetor nunca é gerado e a busca semântica do cofre
decai nota a nota, em silêncio.

**Acceptance Criteria**:

1. WHEN the release workflow builds the Linux artifact THEN the build SHALL use
   the `onnx` build tag together with `sqlite_fts5` and `CGO_ENABLED=1`.
2. WHEN the release workflow builds the Windows artifact THEN the build SHALL
   use the `onnx` build tag, and the job SHALL fail with an explicit message
   naming the missing C toolchain when the runner has none, instead of
   publishing a binary without the provider.
3. The CI pipeline SHALL contain a job that compiles the `cmd/mem` package with
   the `onnx` build tag and runs the provider test suite under that tag.
4. The published artifact SHALL contain the builtin embedding provider, and
   `mem search` against a vectorized vault SHALL report the `vector` source in
   the RRF sources.

**Independent Test**: baixar o artefato publicado, rodar `mem search` num vault
com vetor e conferir a fonte `vector` no rodapé — sem provider a resposta é
`[fts:1]`.

---

### P1: Os artefatos do provider chegam na máquina nova ⭐ MVP

**User Story**: Como dono que instala o `my-memory` numa máquina limpa, quero
que a instalação traga o modelo e a DLL do ONNX Runtime, para que a busca
semântica funcione no primeiro comando.

**Why P1**: com a tag, o provider ainda falha sem os dois arquivos no disco —
e hoje não há comando que os traga.

**Acceptance Criteria**:

5. WHEN `mem embed download` runs THEN the system SHALL download the ONNX
   Runtime library (version 1.29.0) and the model files into
   `~/.memory/models/`, verifying the size of each file before moving it into
   its final path.
6. IF the download fails THEN the system SHALL report the attempted URL and the
   destination path, and SHALL leave no partial file in the destination
   directory.
7. WHEN the installer script finishes THEN it SHALL invoke `mem embed download`
   and SHALL print the model directory it used.
8. WHEN `mem embed doctor` runs THEN the system SHALL report the active
   provider, the model path, the ONNX Runtime library path, the dimension in
   use, and whether each artifact exists on disk.

**Independent Test**: numa máquina sem `~/.memory/models/`, rodar o instalador
e depois `mem embed doctor` — os dois artefatos têm de aparecer como presentes
e o provider como ativo.

---

### P2: A degradação é explícita

**User Story**: Como dono que roda o binário oficial sem provider, quero um
aviso claro e único de que a indexação está em modo léxico, para não confundir
"funciona" com "funciona bem".

**Why P2**: o aviso atual existe (`cmd/mem/main.go:1460`, `1897`), mas é uma
linha por documento e passa batido — a busca funciona, então ninguém procura.

**Acceptance Criteria**:

9. WHEN `mem index` runs without an available embedding provider THEN the
   command SHALL print a single consolidated warning naming the missing build
   tag and the consequence (chunks indexed without a vector), instead of one
   warning line per document.
10. IF the provider is unavailable THEN `mem search --mode hybrid` SHALL state
    in its output that the result came from the lexical fallback only.
11. IF the provider is unavailable THEN `mem index` SHALL exit with code 0
    (o FTS lexical é um modo válido) and SHALL report how many documents were
    indexed without a vector.

**Independent Test**: rodar `mem index` com o binário sem a tag e contar as
linhas de aviso: uma, não uma por documento.

## Edge Cases

- IF the C toolchain is missing on the release runner THEN the job SHALL fail
  with a message naming the missing toolchain (nunca publicar binário mudo).
- IF `~/.memory/models/` is not writable THEN the download SHALL report the
  path it could not write and exit with a non-zero code.
- IF the model file exists but the ONNX Runtime library is missing THEN the
  provider SHALL fail with an error naming the missing library — nunca cair em
  "vetor zero silencioso".
- IF the download is interrupted halfway THEN no partial file SHALL remain in
  the destination directory (escrita em arquivo temporário + rename atômico).
- IF the vault has no documents THEN `mem embed doctor` SHALL still report the
  provider status (o diagnóstico é do ambiente, não do conteúdo).

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| ROP-01 | AC1 — release Linux com a tag `onnx` | `.github/workflows/release.yml` |
| ROP-02 | AC2 — release Windows com a tag e toolchain verificada | `.github/workflows/release.yml` |
| ROP-03 | AC3 — job de CI com a tag `onnx` | `.github/workflows/ci.yml` |
| ROP-04 | AC4 — o artefato publicado traz o provider | `.github/workflows/release.yml` · `cmd/mem/builtin_embedder.go` |
| ROP-05 | AC5 — download verificado do modelo e da DLL | `internal/embedder/model_downloader.go` |
| ROP-06 | AC6 — falha de download explícita e sem arquivo parcial | `internal/embedder/model_downloader.go` |
| ROP-07 | AC7 — o instalador invoca o download | `scripts/install.ps1` · `scripts/install.sh` |
| ROP-08 | AC8 — `mem embed doctor` diagnostica o provider | `cmd/mem/embed.go` |
| ROP-09 | AC9 — aviso consolidado no `mem index` | `cmd/mem/main.go` |
| ROP-10 | AC10 — a busca declara o fallback léxico | `cmd/mem/main.go` |
| ROP-11 | AC11 — contagem de documentos indexados sem vetor | `cmd/mem/main.go` |

**Coverage**: 11 total, 11 mapeados, 0 unmapped.

## Success Criteria

- [ ] O release publica artefatos com a tag `onnx` e o `mem search` de um vault
      vetorizado responde `[vector:1]`.
- [ ] A CI tem um job que compila e testa com a tag `onnx`.
- [ ] Numa máquina limpa, `mem embed download` + `mem embed doctor` + `mem
      index` + `mem search` devolvem resultado vetorial.
- [ ] Sem provider, `mem index` imprime **um** aviso e diz quantos documentos
      ficaram sem vetor.
