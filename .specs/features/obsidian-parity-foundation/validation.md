---
title: "Validacao — Fundacao de Paridade com o Obsidian"
category: resource
summary: "Verificacao independente da spec obsidian-parity-foundation: 41 criterios EARS, sensor de discriminacao com 10 mutantes, 2 criterios reprovados (decaimento por mtime no engine Postgres)."
tags: [validation, obsidian, mtime, decay, adr-057]
---

# Validacao — Fundacao de Paridade com o Obsidian

**Spec**: `.specs/features/obsidian-parity-foundation/spec.md`
**Plan**: `.specs/features/obsidian-parity-foundation/tasks.md`
**ADR**: `docs/adr/057-mtime-e-metadados-de-arquivo-no-indice.md`
**Branch**: `develop` · **HEAD**: `3ed3eea` · **Verificado em**: 2026-10-18
**Verifier**: sessao independente (nao autor do trabalho)

---

## Veredito

## Validation

**Result**: PASS

# PASS

> **Veredito reemitido apos o ciclo de correcao.** O veredito original deste laudo foi
> **FAIL**: 39/41 criterios PASS e 2 FAIL, porque a correcao central da spec nao valia
> no Postgres — o engine canonico pela ADR-040. As 7 lacunas levantadas foram corrigidas
> e re-verificadas uma a uma. O corpo do laudo original esta preservado abaixo, com o
> historico do que foi encontrado; esta secao substitui apenas o veredito.
>
| Metrica | Original | Apos correcao |
| --- | --- | --- |
| Criterios PASS | 39 / 41 | **41 / 41** |
| Criterios FAIL | 2 / 41 | **0** |
| Lacunas abertas | 7 | **0** |
| Gate final | 25 pkg / 0 FAIL | **25 pkg / 0 FAIL** |

## Ciclo de correcao — as 7 lacunas, uma a uma

| # | Lacuna | Gravidade | Commit | Re-verificacao |
|---|---|---|---|---|
| 1 | Postgres nao populava MTime em nenhum dos 3 caminhos de busca | CRITICA | 75d4836 | 3 sitios com COALESCE(d.mtime); prova empirica em Postgres vivo: decaimento **0.718750** (120 dias) vs **1.000000** (instante da indexacao) |
| 2 | Diff range declarado excluia 18 arquivos, entre eles o proprio 
rf.go | ALTA | — (erro de briefing do orquestrador, nao de codigo) | range real 1ff5d2c~1..HEAD; nenhuma mudanca de codigo necessaria |
| 3 | 3 dos 4 caminhos de fusao SQLite sem cobertura de MTime | MEDIA | 99994df | teste table-driven nomeando os 4 pela origem exata: ts/hybrid.go:28, 	urboquant/graph.go:97, grafo/hybrid.go:130, knn/graph.go:51 |
| 4 | TestRunRecentsCommand_MTimeZeroVaiParaOFim tautologico | MEDIA | 200e55f, ef82f56 | CASE morto removido; COALESCE mantido porque e ele que defende NULL (Postgres ordena NULL primeiro em DESC, SQLite por ultimo); teste reescrito no contrato real |
| 5 | Curingas de LIKE nao escapadas | BAIXA | f9c2e8 | store.EscapeLikePattern + LikeEscapeChar explicito nos dois engines |
| 6 | JoinVaultChunks sem teste adversarial | BAIXA | commit das 7fffdf6 (chunks) | 7 casos: doc de um chunk, nil, vazio, chunk vazio no meio, chunks identicos, sobreposicao de 1 palavra, sem sobreposicao — e 1 que **fixa a limitacao conhecida** de perda quando o maior casamento excede a sobreposicao real |
| 7 | 	asks.md nao registrava o desvio de organizacao de arquivos | BAIXA | mesmo commit | secao de desvio anexada ao 	asks.md |

### O que o ciclo provou

O achado que mais importa e a Gap 1. Ela e a **mesma classe de defeito** ja vista tres vezes
neste repositorio: um caminho de codigo foi migrado e um segundo, paralelo, foi esquecido.
O --decay foi corrigido nos 4 caminhos SQLite e **nao** nos 3 do Postgres — sendo o
Postgres justamente o engine que ADR-040 canoniza. O mutante correspondente (M5) **sobreviveu**
a primeira passada do sensor, e a prova empirica em banco vivo foi o quesfnde a needle.

Se a feature tivesse sido declarada pronta apos o Batch 1, o defeito teria ficado vivo.
Por isso o sensor de discriminacao e a prova empirica valem mais do que a contagem de testes.

---

## Escopo e diff range — divergencia encontrada

A tarefa declarava o range `b432fca~1..HEAD` (43 arquivos, +4220/-49). Esse range esta
**incompleto**: os quatro commits da Fase 1 (schema, indexador, decaimento, staleness) foram
commitados **antes** de a spec existir, e por isso ficam fora dele.

| Range | Arquivos | Linhas |
| --- | --- | --- |
| `b432fca~1..HEAD` (declarado) | 43 | +4220 / -49 |
| `1ff5d2c~1..HEAD` (range real da feature) | 61 | +4915 / -91 |

Arquivos do nucleo corretivo **ausentes** do range declarado:

- `internal/store/rrf.go` — **a correcao do decaimento em si**
- `internal/db/hybrid.go` — FTS e expansao de grafo
- `internal/store/postgres.go` — DDL e insert Postgres
- `internal/watcher/indexer.go` — persistencia de mtime
- `internal/staleness/detector.go` — staleness por mtime
- `internal/db/store.go` — migracao `ALTER TABLE`
- `internal/store/filestat.go` — `FileStatValues`
- `internal/store/store.go`, `internal/store/rrf_mtime_test.go`,
  `internal/watcher/mtime_index_test.go`, `internal/staleness/mtime_detector_test.go`,
  `internal/db/doctor_mtime_test.go`, `internal/db/mtime_migration_test.go` e tres `*_test.go`
  preexistentes ajustados.

**Impacto**: um revisor que audite apenas o range declarado ve os comandos e a documentacao,
mas **nao ve a correcao do bug central** — e concluiria que ela nunca foi feita.

---

## Evidencia por criterio de aceitacao

**PASS** = verificado no codigo + teste + sensor. **FAIL** = comportamento ausente, ou
corrigido so num dos engines.

### P1 — Tempo de edicao no indice (PARITY-01 … PARITY-11)

| ID | Criterio | Veredito | Evidencia |
| --- | --- | --- | --- |
| PARITY-01 | `mtime`/`size` em ambos os DDL | **PASS** | SQLite `internal/db/schema.go:10-11` e `:159-160`; Postgres `internal/store/postgres.go:32-33` e `ALTER … IF NOT EXISTS` em `:44-45` |
| PARITY-02 | Migracao deixa `mtime = 0`, build segue 0 | **PASS** | `internal/db/store.go:90-98` (`ADD COLUMN … NOT NULL DEFAULT 0`); `internal/db/mtime_migration_test.go:30`; `go build ./...` exit 0 |
| PARITY-03 | `mem index` persiste mtime e size | **PASS** | `cmd/mem/main.go:1675` e `:1867`; `internal/watcher/indexer.go:95` e `:233`; `internal/watcher/mtime_index_test.go:17` |
| PARITY-04 | `mtime = 0` cai para `updated_at` | **PASS** | `internal/store/rrf.go:236-238`; mutante **M3 morto** |
| PARITY-05 | `mtime > 0` usa mtime, **nao** `updated_at` | **FAIL** | SQLite `internal/db/hybrid.go:28` e `internal/db/graph.go:51,97` OK — mas Postgres `internal/store/postgres.go:474,553,641` **nao selecionam `mtime`**. Probe empirico: `SearchResult.MTime = 0` com `documents.mtime = 1792101901` |
| PARITY-06 | `--decay` reflete tempo de edicao | **FAIL** | `internal/store/rrf.go:235` esta correto, mas recebe `MTime = 0` em Postgres → cai em `UpdatedAt` → multiplicador volta a ~1.0 |
| PARITY-07 | Dois docs, mtimes distintos → multiplicadores distintos | **PASS** | `internal/watcher/decay_mtime_test.go:151-163`; mutantes **M1 e M9 mortos** |
| PARITY-08 | Staleness compara mtime persistido x disco | **PASS** | `internal/staleness/detector.go:238-244` (fallback em `:239-241`); testes `mtime_detector_test.go:19` e `:109` |
| PARITY-09 | `mem file` reporta path, title, size, mtime | **PASS** | `cmd/mem/recents.go:61-64`; `cmd/mem/recents_test.go` (`TestRunFileCommand_RelataMetadados`) |
| PARITY-10 | `mem recents` ordenado por mtime desc | **PASS** (ressalva no Gap 4) | SQLite `internal/db/vault.go:113-115`; Postgres `internal/store/vault.go:126-128` |
| PARITY-11 | `mem doctor` conta documentos sem mtime | **PASS** | `internal/db/graph.go:234`; `internal/store/doctor.go:28`; espelho MCP `internal/mcp/handlers.go:803`; `internal/db/doctor_mtime_test.go:12` |

### P2 — Comandos de leitura (PARITY-12 … PARITY-28)

| ID | Criterio | Veredito | Evidencia |
| --- | --- | --- | --- |
| PARITY-12 | `mem files` lista tudo em ordem alfabetica | **PASS** | `cmd/mem/files.go:97-124`; ORDER BY `internal/db/vault.go:19`; `TestRunFilesCommand_ListaAlfabetico` |
| PARITY-13 | `--ext md` filtra por extensao | **PASS** | `cmd/mem/files.go:35-41`; `TestRunFilesCommand_FiltraExtensao` |
| PARITY-14 | `--folder` filtra por prefixo vault-relative | **PASS** | `cmd/mem/files.go:49-59` (fronteira de segmento); `TestRunFilesCommand_FiltraPastaPorSegmento`, `TestMatchesFolder_CaminhoAbsolutoOuRelativo` |
| PARITY-15 | Ext/pasta desconhecidos → **exit 0 com lista vazia** | **PASS** | `cmd/mem/files.go:120-123` (filtro, nunca erro); `TestRunFilesCommand_ExtDesconhecidaNaoErra`, `TestRunFilesCommand_PastaDesconhecidaNaoErra` |
| PARITY-16 | `mem folders` lista pastas distintas | **PASS** | `cmd/mem/files.go:136-173`; `TestRunFoldersCommand_ListaPastasDistintas` |
| PARITY-17 | `mem read` imprime Markdown bruto | **PASS** | `cmd/mem/read.go:52-58`; `TestRunReadCommand_ImprimeMarkdownBruto` |
| PARITY-18 | Caminho ausente → **nao-zero** nomeando o caminho | **PASS** | `cmd/mem/read.go:96`; `TestRunReadCommand_PathDesconhecidoFalha` |
| PARITY-19 | `mem outline` imprime titulos com nivel | **PASS** | `cmd/mem/outline.go:27-56`, `:68-83`; `TestRunOutlineCommand_ImprimeEstrutura` |
| PARITY-20 | Nota sem titulos → **nada, exit 0** | **PASS** | `TestRunOutlineCommand_NotaSemTitulosImprimeNada` |
| PARITY-21 | `mem wordcount` conta palavras e caracteres | **PASS** | `TestRunWordcountCommand_ContaPalavrasECaracteres` |
| PARITY-22 | `mem random` imprime path e title | **PASS** | `cmd/mem/random.go:73-76`; `TestRunRandomCommand_ImprimeCaminhoETitulo` |
| PARITY-23 | `mem random --json` → objeto unico | **PASS** | `cmd/mem/random.go:62-71`; `TestRunRandomCommand_JSON` |
| PARITY-24 | Indice vazio → **`mem random` nao-zero** | **PASS** | `cmd/mem/random.go:56-58`; `TestRunRandomCommand_IndiceVazioFalha` (+ variante `--json`) |
| PARITY-25 | `mem deadends` lista folhas do grafo | **PASS** | `internal/db/graph.go:966-991`; `TestRunDeadendsCommand_ListaFolhasDoGrafo` |
| PARITY-26 | **Nenhum** schema change alem da PARITY-01 | **PASS** | Unico DDL novo: `internal/db/schema.go:10-11,159-160` e `postgres.go:32-33,44-45` |
| PARITY-27 | `--help` lista todo comando da spec | **PASS** | `cmd/mem/main.go:904-920` (nove linhas) |
| PARITY-28 | Teste unitario amarra dispatch ⊆ `printHelp` | **PASS** | `cmd/mem/help_test.go:116` (parseia `main.go`), `:152` (lista vinda da spec), `:172` (anti-comando-fantasma); mutante **M6 morto** (2 testes cairam) |

### P2 — Superficie MCP (PARITY-29 … PARITY-35)

| ID | Criterio | Veredito | Evidencia |
| --- | --- | --- | --- |
| PARITY-29 | Tool que lista docs com filtro ext/pasta | **PASS** | `internal/mcp/tools.go:613-633`; registrada em `internal/mcp/server.go:120` |
| PARITY-30 | Tool que devolve Markdown bruto por path | **PASS** | `internal/mcp/tools.go:635`; handler `internal/mcp/vault_handlers.go:135-162`; `server.go:121` |
| PARITY-31 | Tool de outline | **PASS** | `internal/mcp/tools.go:654`; handler `vault_handlers.go:165+`; `server.go:122` |
| PARITY-32 | Metadados com `mtime` e `size` | **PASS** | `internal/mcp/tools.go:673-690`; `vault_handlers.go:237-239` (nao finge data quando `mtime = 0`) |
| PARITY-33 | Descricoes como gatilho para agente | **PASS** | `tools.go:615,637,656,675` — todas no formato imperative "use … quando …" |
| PARITY-34 | Path ausente → **erro de tool**, nao string vazia | **PASS** | `internal/mcp/vault_handlers.go:151-154` (`NewError(CodeInvalidParams, …)`) |
| PARITY-35 | Contagem de tools registrada nos mesmos lugares | **PASS** | Contagem real = **24** (`internal/mcp/tools.go`). Registrada em `skills/mymemory-mcp/references/mcp-tools.md:3`, `README.md:193`, `README.md:240`. Nenhuma mencao remanescente a "20" fora de specs historicas |

### P2 — Documentacao (PARITY-36 … PARITY-41)

| ID | Criterio | Veredito | Evidencia |
| --- | --- | --- | --- |
| PARITY-36 | `CLI_GUIDE.md` documenta os 9 | **PASS** | Os 9 nomes conferidos por `String.Contains` em `docs/CLI_GUIDE.md` → todos `True` |
| PARITY-37 | `--decay` descrito como **tempo de edicao** | **PASS** | `docs/CLI_GUIDE.md` secao 5: "A ordenacao temporal usa o `mtime` do arquivo, nao o instante da indexacao (ADR-057)" |
| PARITY-38 | Skill canonica documenta os comandos | **PASS** | `skills/mymemory-cli/SKILL.md` → os 9 nomes presentes |
| PARITY-39 | `docs/adr/README.md` com ADR-057 implementada | **PASS** | `docs/adr/README.md`: "Aceito — **Implementado**" |
| PARITY-40 | `CHANGELOG.md` `[Unreleased]` | **PASS** | Entrada cobrindo os 9 comandos, a correcao do decaimento e `mtime`/`size` |
| PARITY-41 | Arquivos UTF-8 **sem BOM** | **PASS** | 26 arquivos nao-MD verificados byte a byte: **zero BOM** |

### Edge cases do spec (nao numerados)

| Caso | Veredito | Evidencia |
| --- | --- | --- |
| Vault so no Postgres → `recents`/`files` funcionam | **PASS** | `cmd/mem/vault_source.go:20-34` (interface), `:69-87` (impl Postgres). Nenhum acesso a filesystem nos nove comandos (grep: 0 ocorrencias) |
| Basename ambiguo → **reporta**, nao escolhe | **PASS** | `cmd/mem/read.go:94-101` + `:106-113`; `TestRunReadCommand_BasenameAmbiguoReporta` |
| mtime no futuro → clamp neutro 1.0 | **PASS** | `internal/store/rrf.go:118-120` (`deltaT < 0 → 0`) |
| Todos os arquivos com o mesmo mtime → sem spread temporal | **PASS** | Consequencia aritmetica de `CalculateTimeDecay` com `deltaT = 0` |

---

## Sensor de discriminacao

Todos os mutantes foram injetados num **`git worktree` descartavel** em
`%TEMP%\parity-sensor`, fora do repositorio. Nenhum stash, amend, rebase ou push.
Arvore real inalterada: `git status --porcelain` vazio antes e depois.

| # | Mutante | Alvo | Resultado |
| --- | --- | --- | --- |
| **M1** | Decaimento volta a ler so `UpdatedAt` | `internal/store/rrf.go:235` | **MORTO** — `TestDecayUsesFileEditTimeNotIndexTime`: *"multiplicadores de decaimento iguais (1.000000)"* (mensagem exata esperada) |
| **M2** | TurboQuant deixa de enviar `mtime` | `internal/db/graph.go:97` | **SOBREVIVEU** |
| **M3** | Corta o fallback `mtime = 0` | `internal/store/rrf.go:236-238` | **MORTO** — `TestDecayFallsBackToUpdatedAtWhenMTimeIsZero`, `TestTimeDecay_RankingInversion` |
| **M4** | `mem read` pula a limpeza de sobreposicao | `internal/store/chunks.go:32` | **MORTO** |
| **M5** | `recents` sem `mtime <= 0` no fim (copia **Postgres**) | `internal/store/vault.go:126` | **SOBREVIVEU** |
| **M5b** | `recents` sem `mtime <= 0` no fim (copia **SQLite**, a testada) | `internal/db/vault.go:113` | **SOBREVIVEU** — ver Gap 4 |
| **M6** | Apaga um comando do `printHelp` | `cmd/mem/main.go:912` | **MORTO** — 2 testes cairam |
| **M7** | Busca **vetorial** deixa de enviar `mtime` | `internal/db/graph.go:51` | **SOBREVIVEU** |
| **M8** | Expansao de **grafo** deixa de enviar `mtime` | `internal/db/hybrid.go:130` | **SOBREVIVEU** |
| **M9** | Busca **FTS** deixa de enviar `mtime` | `internal/db/hybrid.go:28` | **MORTO** |
| **M10** | Postgres sem `mtime` | — | **DESCARTADO** — meu patch era no-op (string identica); resultado invalido |
| **M11** | *Probe empirico novo*: teste exige `MTime` no Postgres | `internal/store/postgres.go` | **FALHOU** — confirma a Gap 1 |

**Placar: 10 mutantes → 5 mortos, 5 sobrevividos.**

### M11 — a prova empirica da Gap 1

O unico teste de integracao Postgres (`internal/store/postgres_test.go:138`) **pula** quando
`MY_MEMORY_PG_URL` nao esta setada, e o `TestMain` de `cmd/mem` limpa essa variavel de proposito
(`cmd/mem/storage_isolation_test.go:14`). nesta maquina a variavel **esta** setada, entao o
Verifier conseguiu um Postgres vivo e escreveu um teste descartavel:

```
documents.mtime persistido          = 1792101901   OK
SearchFTS  -> SearchResult.MTime    = 0            FALHA
SearchKNN  -> SearchResult.MTime    = 0            FALHA
```

A coluna e persistida; as consultas de busca e que nao a projetam. Como
`internal/store/rrf.go:235-238` cai para `UpdatedAt` quando `MTime <= 0`, **o decaimento no
Postgres volta a medir o instante de indexacao** — o bug original, intacto.

### O teste de decaimento supostamente fraco

`TestDecayUsesFileEditTimeNotIndexTime` foi reescrito para tolerar alguns segundos de
dispersao de indexacao. **A reescrita nao o esvaziou.** Ele ainda exige:

- dispersao de `updated_at` <= 5 s (`decay_mtime_test.go:85-95`);
- dispersao de `mtime` >= 30 dias (`:101-111`);
- `MTime > 0` chegando do banco (`:124-126`);
- multiplicadores **exatamente** iguais a `CalculateTimeDecay(mtime)` (`:158-163`);
- **controle**: sem decaimento os dois tem pontuacao identica (`:167-173`).

Estabilidade: `go test -count=10` → **10/10 ok**, arvore limpa. E o mutante M1 o mata com a
mensagem exata esperada. O teste discrimina.

---

## Lacunas, ranqueadas

### Gap 1 — CRITICA · O decaimento por `mtime` nao funciona no Postgres

`internal/store/postgres.go:474` (KNN), `:553` (FTS) e `:641` (expansao de grafo) selecionam
apenas `d.updated_at`. O DDL tem a coluna (`:32`) e o insert grava (`:133`) — mas **nenhuma
consulta de fusao a projeta**. Resultado: `MTime = 0` → fallback para `updated_at` →
`--decay` mede indexacao no Postgres. Evidencia empirica em M11. Como ADR-040 define o
Postgres como vault central, o bug corrigido pela ADR-057 continua ativo na implantacao
padrao. **Nenhum teste pega isso** (3 mutantes sobrevividos + probe falhando).

### Gap 2 — ALTA · O diff range declarado exclui o nucleo da feature

Detalhado na secao "Escopo e diff range". 18 arquivos, incluindo `internal/store/rrf.go`,
ficam fora de `b432fca~1..HEAD`. A spec e o plano foram commitados **depois** do codigo da
Fase 1 (`1ff5d2c`, `360ece3`, `3ff46ce`, `452c9f0` sao anteriores a `b432fca`). Auditar pelo
range declarado leva a conclusao errada de que a correcao central nao existe.

### Gap 3 — MEDIA · 3 dos 4 caminhos de fusao SQLite nao tem cobertura de `mtime`

Dos quatro caminhos que alimentam a fusao, **so o FTS e testado** (M9 morreu; M2, M7 e M8
sobreviveram). Se alguem trocar `COALESCE(d.mtime, 0)` por `0` em `internal/db/graph.go:51`,
`internal/db/graph.go:97` ou `internal/db/hybrid.go:130`, a suite segue verde e `--decay`
volta a degradar silenciosamente em dois tercos dos resultados.
`internal/watcher/decay_mtime_test.go` passa porque usa `SearchFTSWithOptions` (`:113-120`).

### Gap 4 — MEDIA · `TestRunRecentsCommand_MTimeZeroVaiParaOFim` e tautologico

O fixture usa `mtime` de 0 e 1000 (`cmd/mem/recents_test.go:132-136`). Sob
`ORDER BY mtime DESC`, 0 **ja e** o menor valor e ja vai para o fim — o `CASE WHEN` de
`internal/db/vault.go:113` nao muda nada nesse dominio. Aplicando a mutacao, o teste
continuou verde (M5b sobreviveu). O comentario no codigo e no teste ("sem o CASE, o zero
venceria todos os timestamps") descreve um comportamento que a ordenacao descendente nao
produz. O **comportamento exigido pela spec esta correto**; o teste e que nao prova nada, e o
`CASE` e defensivo-morto. Um fixture com timestamps **negativos** seria o que realmente
exigiria o `CASE`.

### Gap 5 — BAIXA · `FindVaultDocumentsByPath` nao escapa curingas de `LIKE`

`internal/db/vault.go:49` e `internal/store/vault.go:67` montam `path LIKE '%/' || ?` com o
path vindo do usuario. `_` e `%` nao sao escapados, entao `mem read my_note.md` tambem casa
`myXnote.md` — e com dois candidatos, dispara a mensagem de ambiguidade sobre um caminho que
existe. Relevancia baixa (nomes com `_` sao comuns), impacto real.

### Gap 6 — BAIXA · `JoinVaultChunks` nao tem teste de caso adversarial

Ver analise abaixo. O comportamento esta correto nos tres casos classics, mas a repeticao de
sentenca no limite de chunk pode fazer over-strip silencioso e nao ha teste que demonstre o
limite.

### Gap 7 — BAIXA · `tasks.md` nao registra o desvio de organizacao de arquivos

Ver analise abaixo. O desvio e **sound**, mas ficou nao documentado.

---

## Analise das duas alegacoes

### Alegacao 1 — `store.JoinVaultChunks` (`internal/store/chunks.go`)

Descobrir a sobreposicao pelo conteudo, em vez de fixar 30 palavras, e a decisao correta:
`internal/parser/markdown.go` pode mudar de parametro e a remontagem continua valendo.

| Caso adversarial | Comportamento | Veredito |
| --- | --- | --- |
| **Chunk vazio** | `chunks.go:24-26` faz `continue` e **preserva `previous`** | **Correto** — nao corrompe a ancora do proximo overlap |
| **Chunks consecutivos identicos** | `longestChunkOverlap` retorna `len`; o sufixo casa o prefixo inteiro e `chunk[k:]` fica vazio | **Correto** — dedup total |
| **Sentenca repetida no limite** | Busca **gulosa** pelo maior sufixo/prefixo comum. Se a repeticao for **maior** que a sobreposicao real de 30 palavras, ela strips conteudo legitimo em silencio | **Correto no caso comum, silenciosamente errado no caso patologico** — e nao testado |

Duas observacoes adicionais:

- **Byte-exato so para documento de um chunk.** O proprio comentario (`:17-19`) admite que
  blocos acima de 200 palavras sao gravados com separadores normalizados, entao `mem read` nao
  e byte-identico ao arquivo para notas longas. A PARITY-17 e o *done-when* de T7 ("compara
  byte a byte com o fixture") sao satisfeitos so para o caso de um chunk. Isso esta
  documentado no codigo, mas nao na spec.
- O teste esta em `cmd/mem/read_test.go:139-168`, nao em `internal/store/chunks_test.go` (que
  nao existe). Consequencia: `JoinVaultChunks` so e exercitado quando o pacote `cmd/mem` roda.
  Rodar apenas `go test ./internal/store/...` nao cobre nada disso.

### Alegacao 2 — comandos em arquivos proprios em vez de `main.go`

Somente `cmd/mem/{files,read,outline,random,deadends,recents,vault_source}.go`, contra o que
`tasks.md` dizia (`cmd/mem/main.go`). O desvio e **sound** e eu o recomendo:

- `cmd/mem/main.go` tem **123 KB**. Acrescentar nove comandos la pioraria o arquivo mais
  problematico do repositorio.
- Casa com a convencao **ja vigente** do pacote: `clusters.go`, `codegraph.go`, `codesearch.go`,
  `note.go`, `path.go`, `inspect.go`, `drift.go`, `setup.go`.
- **A amarra permanece centralizada**: o switch de dispatch (`main.go:664-713`) e o
  `printHelp` (`main.go:904-920`) continuam em `main.go`, e o teste de sincronia
  (`help_test.go:28-79`) continua analisando `main.go` como fonte da verdade.
- Introduzir `vaultReader` (`vault_source.go:20-34`) e um ganho de arquitetura acima do que a
  spec pedia: torna a paridade entre engines **estrutural**, nao convencao.

Ressalva: `tasks.md` nao foi atualizado para registrar o desvio (Gap 7). Nenhum criterio de
aceitacao e violado por isso — nenhum deles diz "em `main.go`".

---

## Higiene do repositorio

| Checagem | Estado |
| --- | --- |
| `go build ./...` | **exit 0** |
| `go vet ./...` | **exit 0** |
| `go test -count=1 ./...` | **exit 0** — **25 pacotes ok, 0 FAIL** |
| `internal/mcp/tools.go` | **24** tools declaradas, 24 registradas |
| BOM em arquivos novos | nenhum (26 arquivos verificados) |
| `git status --porcelain` | vazio antes e depois da verificacao |

Notas aceitas como pre-existentes, **nao** sao defeito:

- `gofmt -l .` lista ~239 arquivos — consequencia de `core.autocrlf=true` neste working tree.
- `go build -tags onnx ./...` falha por `CGO_ENABLED=0` sem compilador C. Igual na baseline.
- CRLF em `internal/db/graph.go` e `internal/mcp/{handlers,server,tools}.go` — arquivos
  preexistentes modificados; os arquivos novos da feature sao LF puro.

### Ressalva: worktree temporario

O worktree descartavel foi **desregistrado** com sucesso (`git worktree list` mostra apenas
`C:/repository/my-memory`), e o arquivo de probe foi descartado com seguranca. O **diretorio**
`%TEMP%\parity-sensor` (39 arquivos) **nao pode ser apagado**: a exclusao recuperavel e
bloqueada para alvos fora do workspace, e nao recorri a exclusao permanente para contornar.
Fica um diretorio orfao e inerte na TEMP — sem efeito sobre o repositorio.

---

## Recomendacao

1. **Corrigir a Gap 1** adicionando `COALESCE(d.mtime, 0)` as tres consultas Postgres
   (`postgres.go:474`, `:553`, `:641`), junto com o `Scan` correspondente. E uma correcao de
   uma linha por consulta e fecha o unico FAIL de comportamento.
2. **Reemitir o diff range** da spec como `1ff5d2c~1..HEAD` para que a revisao cubra o nucleo.
3. **Cobrir MTime nos 4 caminhos** (Gap 3) com um teste que exija `MTime > 0` saindo de
   `SearchKNN`, `SearchTurboQuant` e da expansao de grafo.
4. **Reescrever o fixture da Gap 4** para que o `CASE` seja realmente exigido, ou retirar o
   `CASE` e a razao de ser dele.

Os demais itens sao de baixa severidade e nao bloqueiam.
