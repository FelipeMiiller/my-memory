# Validation: fix-index-tolerancia

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-032 — opção **C** (`.specs/ISSUES.md:1263`, tabela "Opções de correção")
**Branch**: `develop` · **Base**: `adfb8cc` · **Diff**: working tree (sem commit — o git é do gestor)

## Validation: PASS

**Result**: PASS

Escopo verificado: 10 requisitos (TOL-01..10), a prova de que os testes **falham contra o comportamento antigo** (rodada vermelha com o sintoma exato da ISSUE-032), o gate completo (test, build, vet, gofmt) e um smoke test do binário real em SQLite.

**Arquivos tocados** (todos UTF-8 sem BOM; `.gitattributes` normaliza o blob para LF):

| Arquivo | Mudança |
| ------- | ------- |
| `cmd/mem/main.go` | +60 / −7 — interface `postgresIndexStore`, tolerância por documento nos dois engines, erro de varredura que aborta, resumo com contador de falhas |
| `cmd/mem/index_tolerance_test.go` | +238 — 2 testes herméticos (store falso no Postgres, trigger no SQLite) + o store falso |
| `docs/CLI_GUIDE.md` | +1 — o contrato novo na seção `mem index` |
| `.specs/features/fix-index-tolerancia/` | `spec.md`, `tasks.md`, `validation.md` |

---

## Evidência por requisito

| Req | Critério | Evidência | Comando | Saída real |
| --- | -------- | --------- | ------- | ---------- |
| TOL-01 | Falha de gravação de um documento não aborta a varredura | `cmd/mem/main.go:1867` (Postgres: aviso + `return nil`), `cmd/mem/main.go:2108` (SQLite), `cmd/mem/index_tolerance_test.go:45` | go test -count=1 -run Tolerante ./cmd/mem/... | `ok …/cmd/mem 0.302s` — o 3º documento (depois do que falhou) foi gravado |
| TOL-02 | Aviso por documento com nome do arquivo e erro | `cmd/mem/main.go:1867`, `cmd/mem/main.go:2108`, `cmd/mem/index_tolerance_test.go:64` | go test -count=1 -run Tolerante ./cmd/mem/... | `Aviso: falha ao gravar documento "<path>" (pq: unnamed prepared statement does not exist (26000)); documento fica de fora do índice` |
| TOL-03 | Documento que falhou não conta como indexado | `cmd/mem/main.go:1925` (`indexedCount++` só depois da gravação bem-sucedida), `cmd/mem/index_tolerance_test.go:53` | go test -count=1 -run Tolerante ./cmd/mem/... | resumo `2 indexados` com 3 documentos processados |
| TOL-04 | Documento que falhou não é podado | `cmd/mem/main.go:1793` (`activeDocIDs` é montado **antes** da gravação), `cmd/mem/index_tolerance_test.go:59` | go test -count=1 -run Tolerante ./cmd/mem/... | o `activeDocIDs` recebido pelo prune contém o documento que falhou |
| TOL-05 | Varredura terminada → exit 0 e resumo com a contagem | `cmd/mem/main.go:1956`, `cmd/mem/main.go:2200`, `cmd/mem/index_tolerance_test.go:76` | smoke do binário (`mem.exe index --storage sqlite`) | `🎉 Concluído! 2 documentos processados no SQLite (2 indexados, 0 em cache, 0 podados, 0 falhos).` · `exit=0` |
| TOL-06 | Resumo final quantifica as falhas | `cmd/mem/main.go:1956`, `cmd/mem/main.go:2200`, `cmd/mem/index_tolerance_test.go:82` | go test -count=1 -run Tolerante ./cmd/mem/... | resumo contém `1 falhos` |
| TOL-07 | Relatório final nomeia quem ficou de fora | `cmd/mem/main.go:1950`, `cmd/mem/main.go:2194`, `cmd/mem/index_tolerance_test.go:86` | go test -count=1 -run Tolerante ./cmd/mem/... | `⚠️  Documentos que falharam e ficaram de fora do índice (1):` + `   - <path>` |
| TOL-08 | Mesmo contrato no caminho SQLite | `cmd/mem/main.go:2101-2110`, `cmd/mem/index_tolerance_test.go:95` | go test -count=1 -run Tolerante ./cmd/mem/... | o documento posterior à falha entrou no índice (`documents = 1`) e o que falhou não (`documents = 0`) |
| TOL-09 | Erro de nível de varredura aborta (não engole tudo) | `cmd/mem/main.go:1750-1755`, `cmd/mem/main.go:1996-2001` (o `err` do walk volta a ser propagado) | leitura do código + `cmd/mem/main.go:1930` (`Erro durante a indexação: %v` + `return`) | o `return err` antigo do callback (`return nil`) foi substituído por `return err` nos dois caminhos |
| TOL-10 | Teste hermético que falha contra o comportamento antigo | `cmd/mem/index_tolerance_test.go:171` (store falso), `cmd/mem/index_tolerance_test.go:95` (trigger SQLite) | go test -count=1 -run Tolerante ./cmd/mem/... | sem rede, sem Postgres, sem chamar o binário; a rodada vermelha abaixo prova que o teste morre no comportamento antigo |

---

## Prova contra o defeito (comportamento antigo)

A ordem de execução foi **costura → teste → implementação**: primeiro a interface `postgresIndexStore` entrou (sem mudar comportamento), depois o teste foi escrito e rodado **contra o comportamento antigo** — foi aí que a prova nasce. Saída real da rodada vermelha (`exit=1`):

```
--- FAIL: TestMemIndexPostgresToleranteAFalhaPorDocumento (0.02s)
    index_tolerance_test.go:45: a varredura abortou na falha do documento "…\b-segundo.md"; documentos gravados = […\a-primeiro.md]
    index_tolerance_test.go:59: o documento que falhou saiu de activeDocIDs e pode ser podado: []
    index_tolerance_test.go:64: não houve aviso de falha por documento:
        🔍 Indexando notas no PostgreSQL (pgvector) para repo [repo_teste] em: …
        ✔ Indexado no Postgres: a-primeiro (0 arestas, 1 chunks)
        Erro durante a indexação: pq: unnamed prepared statement does not exist (26000)
    index_tolerance_test.go:76: o comando não emitiu o resumo final:
--- FAIL: TestMemIndexSQLiteToleranteAFalhaPorDocumento (0.09s)
    index_tolerance_test.go:137: o documento posterior à falha não foi indexado (documents = 0):
        🔍 Indexando notas no SQLite em: …
        ✔ Indexado no SQLite: a-primeiro (0 arestas, 1 chunks comprimidos)
        Erro durante a indexação: constraint failed: falha simulada de gravacao (1811)
    index_tolerance_test.go:155: o comando não emitiu o resumo final:
FAIL	github.com/FelipeMiiller/my-memory/cmd/mem	0.302s
```

O sintoma vermelho é **exatamente** o da ISSUE-032: a varredura morre no primeiro documento que falha (`Erro durante a indexação: pq: unnamed prepared statement does not exist (26000)`), o documento posterior não é processado, não há aviso por documento e não há resumo final.

Depois da implementação, a mesma roda verde:

```
ok  	github.com/FelipeMiiller/my-memory/cmd/mem	0.302s
```

---

## Comportamento no binário real (smoke SQLite)

Binário compilado do estado final (`go build -o …\mem.exe ./cmd/mem`) contra um vault temporário em scratch — **nenhum** banco do dono foi tocado (`--storage=sqlite` sobrepôs o Postgres do ambiente; a própria saída registra `Postgres detectado no ambiente/config foi ignorado nesta sessão`).

```
🔍 Indexando notas no SQLite em: …\smoke-vault (com TurboQuant 4-bit ativado)
✔ Indexado no SQLite: nota1 (0 arestas, 1 chunks comprimidos)
✔ Indexado no SQLite: nota2 (0 arestas, 1 chunks comprimidos)
🎉 Concluído! 2 documentos processados no SQLite (2 indexados, 0 em cache, 0 podados, 0 falhos).
```

`exit=0`.

---

## Gate

| Gate | Comando | Saída |
| ---- | ------- | ----- |
| Testes (suíte completa) | `$env:MY_MEMORY_PG_TEST='1'; go -C D:\repository\my-memory test -count=1 ./...` | `exit=0` — 26 pacotes `ok`, nenhum `FAIL` (`cmd/mem` 102.3s, `internal/db` 41.1s, `internal/store` 40.3s, `internal/pgtest` 1.1s — a integração com Postgres rodou de verdade, `MY_MEMORY_PG_URL` preservada) |
| Teste da feature | `go -C D:\repository\my-memory test -count=1 -run Tolerante ./cmd/mem/...` | `exit=0` — `ok …/cmd/mem 0.302s` |
| Build | `go -C D:\repository\my-memory build ./...` | `exit=0` |
| Vet | `go -C D:\repository\my-memory vet ./...` | `exit=0` |
| Formatação | `gofmt -l` na **cópia LF** dos 2 arquivos `.go` tocados (`D:\.minimax\scratch\subagents\issue-032-c\gofmt-check`) | saída vazia — formatação limpa |
| Smoke | binário real em vault SQLite temporário | `exit=0` — resumo com `0 falhos` |
| Spec | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-index-tolerancia/spec.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Tasks | `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/fix-index-tolerancia/tasks.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Estado | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-index-tolerancia --root D:\repository\my-memory` | `0 error(s)` · `exit=0` |

---

## Decisões que ficaram registradas (e o porquê)

- **Código de saída zero** (A5 da spec). A varredura terminou; o que houve foi a falha de **um** documento, e ela é reportada na saída. Uma saída não-zero mudaria o contrato do `mem index` para todo consumidor (CI, scripts) por causa de um documento — decisão do dono, não minha. Se o dono quiser o contrário, o ponto de mudança é o `return nil` de `cmd/mem/main.go:1867`/`2108` (propagar o erro no fim da varredura).
- **Erro de nível de varredura agora aborta** (A2 da spec). O código anterior **ignorava** o `err` do `filepath.WalkDir` (`return nil`), ou seja, um diretório ilegível era pulado em silêncio. Como o dono pediu explicitamente que esses erros "continuem abortando — não engola tudo", o comportamento foi alinhado ao contrato: `return err` nos dois caminhos (`cmd/mem/main.go:1754` e `:2000`), com o relatório `Erro durante a indexação: …` e **sem** resumo de sucesso. O helper `preCollectDocTitles` (`cmd/mem/main.go:1476`) segue best-effort — ele só monta a lista de títulos para resolução de tags.
- **Nenhum ADR novo** (A10 da spec). Não há mudança de schema, de driver nem de arquitetura: é o contrato de tolerância do comando `mem index`. O driver continua `lib/pq` e `go.mod`/`go.sum` não foram tocados (a opção B está adiada pelo dono).
- **`README.md` não foi alterado.** A mudança é um contrato de robustez do comando, não uma capacidade nova, um comando novo ou uma ferramenta MCP — o lugar certo do contrato é a seção `mem index` do `docs/CLI_GUIDE.md` (que foi atualizado). O AGENTS.md pede README para "capacidades, novos comandos, novas ferramentas MCP e exemplos de uso"; nada disso mudou.
- **`internal/watcher` ficou de fora** (A9 da spec). O watcher tem o próprio indexador (`internal/watcher/indexer.go:259`), fora da propriedade desta execução; a feature entrega o contrato do comando `mem index`.
- **Efeito colateral conhecido e documentado** (A11 da spec): um documento que falhou pode ficar sem os chunks/arestas antigos, porque `DeleteDocumentData` roda antes da gravação. Reindexar o documento devolve os chunks.

## Pendências para o gestor

- Atualizar `.specs/ISSUES.md` (status da ISSUE-032) e `.specs/STATE.md` — fora da propriedade desta execução.
- Se a opção B (trocar `lib/pq` por `pgx`) for retomada, ela precisa de spec própria.
- Decidir se o `internal/watcher` recebe o mesmo contrato de tolerância.
- Nenhum commit e nenhum push — o git é do gestor.
