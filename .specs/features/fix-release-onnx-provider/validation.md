# fix-release-onnx-provider Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/fix-release-onnx-provider/spec.md`
**Diff range**: `b379271..438bfdc` (branch `develop`, HEAD `438bfdc`)
**Round**: 2 (a rodada 1 devolveu FAIL — 5 gaps; esta rodada fecha os 5)
**Verdict**: PASS (11/11 ACs com evidência; 8/8 mutações mortas; gate verde)

> [!IMPORTANT]
> **A independência do verificador desta rodada foi perdida, e isso está declarado aqui de propósito.**
> A rodada 1 foi verificada por um sub-agente independente. A rodada 2 **não foi**: o
> sub-agente despachado para re-verificar **morreu duas vezes no meio da execução**
> (turno encerrado com `finish_reason: stop` e saída vazia, logo após uma chamada de
> ferramenta — assinatura de limite de uso do provider). A verificação foi concluída
> pelo **orquestrador** (autor do código), usando o *standalone fallback* do skill
> (`tlc-spec-driven`: "Without sub-agents, run `validate.md` as an independent
> fresh-eyes pass"). Consequência honesta: a separação autor ≠ verificador **vale** para
> a superfície da rodada 1 (que o verificador independente examinou), e **não vale** para
> os testes novos da fase 4 — estes foram verificados por quem os escreveu.
> **Recomendação: reexecutar um Verifier independente antes de publicar o release.**

---

## O que mudou em relação à rodada 1

| Gap da rodada 1 | Como foi fechado | Commit |
| --- | --- | --- |
| 1 — AC11: exit code 0 não asseverado (mutante sobreviveu) | o teste captura o retorno **real** de `runIndexCLI`; o mutante F agora morre | `4b41736` |
| 2 — AC5: versão 1.29.0 e destino `~/.memory/models/` sem asserção | teste novo fixa os dois valores e a release `v1.29.0` do manifesto | `ef7bcb7` |
| 3 — AC4: a fonte RRF `vector` não tinha teste de regressão | 3 asserções novas (SQLite determinístico, Postgres, E2E com o provider real) | `d09aa3a`, `fa4fdda` |
| 4 — documentação contradizia o comportamento novo | CLI_GUIDE, CENTRAL_VAULT, ARCHITECTURE, README, docs/README, CHANGELOG, ISSUES, STATE, spec.md | `18a3725` |
| 5 — AC7: o instalador imprimia um caminho fixo | os dois instaladores imprimem o diretório que o download usou | `81aebcf` |
| 5b — AC7 **sem asserção automatizada** (ressalva da rodada 1) | invariante de fonte sobre os dois scripts | `438bfdc` |

Além dos 5 gaps, a fase 4 corrigiu a **precisão da spec** que a rodada 1 apontou: a AC9
dizia "chunks" (e o contrato aprovado da T8 diz **documentos**), e a AC10 não fixava
nenhum marcador — o teste congelava uma string escolhida pelo implementador. As duas ACs
foram ajustadas na spec, e a AC10 passou a exigir os marcadores `LEXICAL` e
`NÃO usou similaridade vetorial`.

---

## Spec-Anchored Acceptance Criteria

| AC | Resultado definido na spec | Evidência (`file:line` + asserção) | Resultado |
| --- | --- | --- | --- |
| AC1 — release Linux | build com as duas tags e `CGO_ENABLED=1` | `.github/workflows/release.yml:123` — `CGO_ENABLED=1 go build -tags "sqlite_fts5 onnx" \` | PASS (artefato; a matriz de `tasks.md` atribui "none" à camada de pipeline — o gate é o próprio CI) |
| AC2 — release Windows | build com a tag; falha nomeando o toolchain C ausente | `release.yml:170` — `go build -tags "sqlite_fts5 onnx"`; `release.yml:153` — `::error::Toolchain C ausente: ...` + `exit 1` | PASS (artefato) |
| AC3 — job de CI com a tag | compila `cmd/mem` com a tag e roda a suíte do provider | `ci.yml:385` — job `embedder-onnx`; `:408` — build; `:413` — `go test -tags "sqlite_fts5 onnx" -v -count=1 ./internal/embedder/...` | PASS (comando do job executado localmente: exit 0) |
| AC4 — o artefato traz o provider; `mem search` reporta a fonte `vector` | a fonte RRF `vector` aparece na busca contra um vault vetorizado | **E2E com o provider real**: `cmd/mem/search_vector_source_test.go:80` — `strings.Contains(out, "vector:")` · **SQLite**: `cmd/mem/embed_provider_degradation_test.go:284` — `strings.Contains(out, "vector:")` · **Postgres**: `internal/store/vector_source_test.go:54` — `strings.HasPrefix(fonte, "vector:")` | PASS (3 asserções, 2 motores, 1 E2E com o provider embutido de verdade) |
| AC5 — download do ORT 1.29.0 e do modelo para `~/.memory/models/`, com tamanho conferido antes do move | versão 1.29.0; destino `~/.memory/models/`; tamanho conferido antes do rename | `internal/embedder/model_downloader_test.go:374` — `ORTVersion != "1.29.0"` ⇒ erro · `:382` — `HasSuffix(dirModelo, sufixo)` · `:398` — `HasSuffix(dirRuntime, ".memory/models/onnxruntime")` · `:410` — `Contains(manifest.Runtime.URL, "v"+ORTVersion+"/")` · tamanho antes do rename: `model_downloader.go:308-311` → `:320` | PASS (era PARTIAL na rodada 1 — os dois valores fixados pela spec não tinham asserção) |
| AC6 — falha de download nomeia URL e destino, sem parcial | erro contém URL + destino; destino sem arquivos | `model_downloader_test.go:84-90` — `Contains(err.Error(), url)` + `Contains(err.Error(), dest)` + `assertNoLeftovers`; repetido em `:114-120` (sha256) e `:141-147` (HTTP 404) | PASS |
| AC7 — o instalador invoca `mem embed download` e imprime o diretório que usou | invocação + o diretório **que ele usou** | `scripts/install.ps1:192` / `scripts/install.sh:291` — `embed download` · `install.ps1:206-211` / `install.sh:301-302` — resolvem `MY_MEMORY_MINILM_MODEL_DIR` e imprimem o resultado · **asserção**: `cmd/mem/install_scripts_test.go:60` (consulta a variável) e `:75` (a linha de impressão não pode ser literal) | PASS (era "PASS com ressalva" na rodada 1 — não havia asserção automatizada) |
| AC8 — `mem embed doctor` reporta provider, paths, dimensão e existência | 5 itens reportados | `cmd/mem/embed_test.go:66-78` — `"builtin"`, `"384"`, `manifest.Model.Dest`, `manifest.Runtime.Dest`, `"ATIVO"`; `:79-83` / `:124-126` — presente / ausente | PASS |
| AC9 — `mem index` imprime **um** aviso consolidado que nomeia a tag e a consequência | 1 aviso por execução (não 1 por documento), nomeando a tag e a consequência, contando **documentos** | `embed_provider_degradation_test.go:31` — `strings.Count(out, avisoDegradacao) != 1` ⇒ erro · `:34` — `Contains(out, "tag \`onnx\`")` · `:37` — `Contains(out, "SEM VETOR")` · `:40` — `Contains(out, "3 documento(s)")` · `:46` — o aviso por documento não pode existir | PASS (a unidade da spec foi corrigida para **documentos**, alinhada ao contrato da T8) |
| AC10 — a busca híbrida declara o fallback léxico | a saída declara que o resultado é só do fallback textual, com os marcadores fixados | `embed_provider_degradation_test.go:218` — `Contains(out, "LEXICAL")` · `:221` — `Contains(out, "NÃO usou similaridade vetorial")` · `:226` — o documento aparece (não-vacuous) · `:248` — com provider, o aviso **não** aparece | PASS (a spec agora fixa os dois marcadores — o gap de precisão da rodada 1 está fechado) |
| AC11 — `mem index` sai com código 0 e diz quantos documentos ficaram sem vetor | exit code 0 **e** a contagem | **exit code**: `embed_provider_degradation_test.go:137` — `if errIndex != nil` sobre o retorno **real** de `runIndexCLI` (o closure com `nil` fixo foi removido) · **contagem**: `:40` — `Contains(out, "3 documento(s)")` | PASS (era PARTIAL na rodada 1 — o mutante F sobrevivia porque a asserção era vacuamente verdadeira) |

**Status**: 11/11 ACs PASS · 0 PARTIAL · 0 FAIL de comportamento · 0 gaps de precisão de spec.

---

## Discrimination Sensor

Executado em worktree descartável (`git worktree add D:\.minimax\scratch\verifier-onnx-r2 18a3725`, detached), **nunca** na árvore real. `git stash` **não** foi usado.

| Mutação | Arquivo | Descrição | Morta? |
| --- | --- | --- | --- |
| M1 | `cmd/mem/main.go` | `runIndexCLI` devolve erro **depois** de indexar tudo | **MORTA** — `TestProviderDegradationIndexTerminaComCodigoZero` falha (`mutante M1: erro devolvido depois de indexar`). *Era o sobrevivente da rodada 1.* |
| M2 | `internal/embedder/model_downloader.go` | `ORTVersion` "1.29.0" → "1.30.0" | **MORTA** — `TestManifestoFixaVersaoDoRuntimeEDestino` falha na URL do runtime |
| M3 | `internal/db/hybrid.go` | fonte RRF `"vector"` → `"embeddings"` | **MORTA** — `TestProviderDegradationBuscaComProviderDeclaraFonteVector` falha |
| M4 | `internal/store/postgres.go` | fonte RRF `"vector"` → `"embeddings"` | **MORTA** — `TestPostgresStore_HybridDeclaraFonteVector` falha (Postgres real, banco descartável) |
| M5 | `cmd/mem/main.go` | a linha TurboQuant é gravada sob outro chunk id | **MORTA** — `TestBuscaComProviderRealDeclaraFonteVector` falha (build com a tag `onnx`) |
| M6a | `scripts/install.ps1` | a impressão volta a ser um caminho literal | **MORTA** — `TestInstaladoresImprimemODiretorioDoModeloUsado` falha |
| M6b | `scripts/install.sh` | idem | **MORTA** — o mesmo teste falha |
| M7 | `cmd/mem/main.go` | o marcador do aviso consolidado muda de texto | **MORTA** — 3 testes de degradação falham ("o aviso de degradação apareceu 0 vezes") |
| M8 | `cmd/mem/main.go` | `Resultado apenas LEXICAL` → minúsculo | **MORTA** — `TestProviderDegradationBuscaHibridaDeclaraFallbackLexico` falha |

**Resultado**: 8/8 mortas (a rodada 1 matou 5/6 — o sobrevivente era o M1 de hoje).
**Sensor depth**: expandido (8 mutações; a task pedia 2-4), porque a rodada 1 deixou um mutante vivo.

### Achado de sensor que a spec 2 precisa saber

Uma mutação adicional (zerar a coluna `chunks.vec` na gravação) **não** mata a busca
vetorial no Windows: `db.HasSqliteVec` é `false` nessa plataforma
(`internal/db/vec_cgo.go:1` restringe o binding a `darwin || linux`), então a busca
vetorial lê as linhas **TurboQuant** (`chunks_turboquant`), não a coluna de vetor bruto.
A correção da ISSUE-040 tem de cobrir **os dois** caminhos de armazenamento, e o KNN do
Postgres (`internal/store/postgres.go:567-582`) também não filtra `embedding IS NULL`.

---

## Gate Check

| Gate | Comando | Resultado |
| --- | --- | --- |
| Full (com Postgres) | `MY_MEMORY_PG_TEST=1` + `MY_MEMORY_PG_URL` do config global (endpoint direto, sem `-pooler`) + `go -C <repo> test -count=1 ./...` | **exit 0** — 26 pacotes `ok`, 0 `FAIL` |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Formato | `gofmt -l` nas cópias LF dos arquivos Go tocados | **vazio** (o working tree mistura CRLF e produziria falso positivo) |
| Tagged build | `CGO_ENABLED=1 go build -tags "sqlite_fts5 onnx" -o <scratch>/mem-verify-final.exe ./cmd/mem` | **exit 0** |
| Tagged suite | `go test -tags "sqlite_fts5 onnx" -count=1 ./internal/embedder/... ./cmd/mem/...` | **exit 0** — `internal/embedder` 10,2 s ok · `cmd/mem` 83,0 s ok |

**Cobertura do provider real**: `cmd/mem/search_vector_source_test.go` só compila sob a
tag `onnx` e **pula** quando os artefatos não estão no disco (`:36` — `t.Skipf` com
motivo explícito). Nesta máquina os artefatos do ONNX Runtime 1.29.0 e o modelo estão
instalados, então o teste **rodou de verdade** no gate taggeado acima (não pulou).

**Contagem de testes**: 1034 antes da feature → **1056** depois (delta **+22**: 9 em
`internal/embedder/model_downloader_test.go`, 4 em `cmd/mem/embed_test.go`, 6 em
`cmd/mem/embed_provider_degradation_test.go`, 1 em `internal/store/vector_source_test.go`,
1 em `cmd/mem/search_vector_source_test.go`, 1 em `cmd/mem/install_scripts_test.go`).
Nenhum teste existente foi alterado, removido ou enfraquecido —
`git diff --name-status b379271..438bfdc -- '*_test.go'` mostra só arquivos **novos**.

**Isolamento**: worktree removido com `git worktree remove --force`;
`git status --porcelain` da árvore real vazio antes e depois; HEAD `438bfdc`;
`git stash list` vazio.

---

## Fix Plans

Nenhum gap de comportamento pendente. As duas ressalvas abaixo não bloqueiam a feature.

1. **Verificação independente pendente.** O sub-agente Verifier morreu duas vezes por
   limite de uso do provider (ver o aviso no topo). Esta rodada foi executada pelo
   orquestrador. **Um Verifier independente deve reexecutar antes do release.**
2. **AC1/AC2/AC3 sem teste automatizado** — por desenho: a matriz de cobertura de
   `tasks.md` atribui "none" à camada de pipeline, e o gate é o próprio CI (o job
   `embedder-onnx` é **duro**, sem `continue-on-error`).

---

## Summary

**Overall**: PASS — 11/11 ACs com evidência `file:line`, 8/8 mutações mortas, gate verde
na suíte full e na suíte taggeada. Os 5 gaps da rodada 1 estão fechados e a spec ganhou
a precisão que faltava.

**O que o produto faz agora**: o binário oficial vem compilado com a tag `onnx`, o
instalador traz o modelo e a biblioteca do ONNX Runtime, `mem embed download`/`doctor`
existem e são diagnosticáveis, e a degradação é **explícita** — um aviso por execução,
exit code 0, e a busca declara que o resultado é apenas lexical. O vetor real é
reportado na fonte RRF (`[vector:1]`), nos dois motores e com o provider de verdade.

**Ressalva honesta**: a independência do verificador desta rodada foi perdida por limite
de uso do provider. O comportamento está verificado; a separação autor ≠ verificador
precisa ser refeita antes de publicar.
