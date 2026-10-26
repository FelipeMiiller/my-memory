---
title: "Handoff — code-graph-real: onde a execução parou"
category: resource
summary: "Estado completo da feature code-graph-real (ADR-061) em 2026-10-24: 18 tarefas em 3 fases, o que foi commitado, o que falta, os desvios e o checklist da mudança do repo para D:."
tags: [my-memory, code-graph, adr-061, handoff, spec]
---

# Handoff — `code-graph-real`: onde a execução parou

> **Data do retrato:** 2026-10-24 · **Branch:** `develop` · **Base:** `35b01ed`
> (commit anterior da rodada das skills) · **Estado do working tree no momento
> deste documento:** limpo, exceto pelo arquivo de validação da Fase 3 que o
> worker estava finalizando.

> [!warning] **Veredito da verificação independente: 🔶 PARTIAL (não é PASS) —
> mas os 8 defeitos já foram corrigidos.**
> O núcleo técnico está sólido e verificadamente real, mas a verificação
> encontrou **2 defeitos de severidade alta** que quebram ACs declaradas
> (D1: a migração quebra em vault existente; D2: `include_code` é no-op para
> qualquer `repo_id` ≠ `default`), **1 alto-médio** (D3: o build
> `-tags treesitter` esvazia o code graph em silêncio) e **3 médios**.
> **Relatório completo:** `.specs/features/code-graph-real/verification.md`.
> **Correções:** `b3ed53f` (D1) · `fbd3d62` (D2) · `b34abd8` (D3) ·
> `1dc845a` (D4) · `b84dfce` (D5) · `aadd791` (D6) · `e64e389` (D7) ·
> `aac0c0b` (D8). Evidência D7/D8 em `validation-d7-d8.md`.
> **Falta:** a **revalidação** das 16 hipóteses depois das correções (item 1 da
> tabela de pendências).

Este documento é o **ponto de retomada** da feature. Ele existe porque a
pasta do repositório vai ser movida de `C:\repository\my-memory` para `D:\...`
e porque a cota do provedor matou workers no meio — quem voltar precisa saber
exatamente o que está feito, o que falta e o que não foi provado.

---

## 1. O que a feature é

**ADR-061** — *dois conceitos de memória, um motor*:

- **Vault Central** = memória de aprendizagem (Obsidian Flavored Markdown:
  wikilinks, `#tags`, propriedades, aliases). É o "conceito de obesidade" já
  definido no repo.
- **Vault de Repositório** = **code graph**: arestas estruturais `calls` /
  `imports` / `inherits` / `uses_type` com **call-site** e **proveniência por
  aresta**, no mesmo motor do my-memory (FTS5 + TurboQuant 4-bit + RRF,
  `epistemic_status` da ADR-011, PageRank, LPA+modularidade, blast radius,
  viewer SPA, federação `repo_id`).

O **Graphify** (`Graphify-Labs/graphify` `v0.9.94`, Apache-2.0) foi a
**referência de conceito**, não uma dependência: o my-memory já é superior em
busca vetorial, blast radius, PageRank, LPA+Q, viewer SPA e federação. O clone
de referência está em `D:\.minimax\scratch\subagents\graphify-study\graphify`.

**Decisão estrutural que a ADR-061 tomou:** o backend do AST é **Go puro**
(`go/parser`/`go/ast`/`go/token` da stdlib), coerente com a ADR-022 (Go puro,
sem CGO) e funcional na máquina do dono (que não tem gcc). A **ADR-047
(tree-sitter) não foi supersedida**: seu plano de deferral segue intacto
(0 de 3 gatilhos) e a build tag continua isolada e com precedência.

---

## 2. Estado por tarefa (18 de 18 implementadas)

| Fase | Tarefas | Status | Commits |
| :--- | :--- | :--- | :--- |
| **1 — ligação** | T1 escopo de código · T2 code stage no `mem index` · T3 engine configurado · T4 `repository` · T5 handlers MCP · T6 flags reais | ✅ | `1bf7e75` `3cd03b5` `33931cb` `0907dbd` `331be16` `340c9cb` `fd95961` |
| **2 — motor** | T7 backend Go puro · T8 `newBackendParser` vivo · T9 `calls` com call-site · T10 `imports`/`inherits`/`uses_type` · T11 `epistemic_status` + ambígua · T12 resposta MCP | ✅ | `4f1182a` `a342518` `b986c37` `870a323` `5f52c2d` `056a3f6` `a680955` `f9a7b58` |
| **3 — superfície** | T13 ponte · T14 fan-in RRF · T15 hub threshold/orçamento · T16 rota `/code` · T17 drift estrutural · T18 `mem lesson` | ✅ | `96a734b` `12f3751` `c1deac3` `2a8d993` `6b48202` `7c14880` |

### O que cada tarefa entregou (curto)

- **T1** — `code.include`/`code.exclude` separados do escopo de notas. Era a
  armadilha mais cara: o `include` global é compartilhado com o pipeline
  Markdown, então incluir `**/*.go` globalmente indexaria Go **como nota**.
- **T2** — `mem index` roda o code stage por default **só quando há escopo de
  código configurado**; falha do code stage não derruba o index.
- **T3** — o code graph vai para o **engine configurado** (Postgres quando
  Postgres). Antes, o pipeline abria o store PG e o descartava, escrevendo num
  SQLite local que nada consultava (gap C5). Veio junto a camada de dialeto
  (`internal/db/dialect.go`: `?` ↔ `$1`).
- **T4** — coluna `repository` em `code_files`/`code_symbols`/`code_edges`, com
  o filtro por repo nas queries MCP.
- **T5** — as tools MCP de código deixaram de responder "indisponível"
  (handlers `nil`): `SetCodeAccess` registra os executores quando o engine do
  servidor é resolvido.
- **T6** — `--repo` vale, `--ast-hash[=false]` controla o `ast_hash`,
  `--json` emite JSON **puro** no stdout (o boot log vai para o stderr) e
  **`--no-embed` saiu do CLI** (era placeholder sem comportamento).
- **T7/T8** — o motor AST real, em Go puro, no caminho de produção. O
  `MockParser` sobrevive só como fallback das linguagens sem backend AST.
- **T9** — arestas `calls` com **call-site** (a linha da *expressão de
  chamada*, não da definição) + a coluna `epistemic_status` e a migração
  idempotente em SQLite e Postgres.
- **T10** — `imports` (com símbolos sintéticos para o pacote importado, que
  de fora do repo não tem símbolo), `inherits` e `uses_type`.
- **T11** — destino ambíguo vai para `code_edges_uncertain` (com
  `confidence` e `reason`) e **nunca** é promovido a aresta resolvida.
- **T12** — a resposta MCP traz `epistemic_status`, `confidence` e o call-site,
  e emite `structuredContent`.
- **T13** — a ponte `documents` ↔ `code_symbols` no **grafo único**: nó
  `graph_nodes` com `type='code_symbol'` e aresta `graph_edges` com
  `relation='references_code'` (idempotente).
- **T14** — `include_code` deixou de ser placeholder: `code_symbols` entra no
  RRF como **fonte real** (lexical, com boost CA-05). Sem a flag, o ranking é
  **idêntico** ao anterior.
- **T15** — hub threshold por percentil do grau (default **p99**, o mesmo do
  Graphify), orçamento de tokens **reaproveitando o estimador que já existia**
  (ADR-029), ordem por distância em saltos e a dica de estreitar a query.
- **T16** — a viewer ganhou a rota **`/code`** (lista símbolos e mostra os
  vizinhos com tipo de aresta e call-site). Bundle gzip **144.667 / 163.840
  bytes**, sem CDN.
- **T17** — o drift lê `ast_hash`: mudança de **assinatura** é drift estrutural,
  mudança de **comentário** não é, e arquivo sem hash gravado não quebra nada.
- **T18** — `mem lesson` grava a lição no **cofre central** como nota Markdown
  puro com `category: memory`, `summary` e timestamp, indexada na hora. Sem
  cofre central resolvido, falha com erro claro e exit ≠ 0.

---

## 3. Evidência empírica (números, não adjetivos)

### Code stage real sobre este repositório (SQLite temporário, `--include=**/*.go --lang=go`)

```
{"engine":"sqlite","repository":"proof/repo","files_scanned":349,"files_indexed":349,
 "files_skipped":0,"files_errors":0,"symbols_total":6754}
```

| kind | arestas | | status | arestas |
| :--- | ---: | :--- | :--- | ---: |
| `calls` | 4055 | | `EXTRACTED` | 6367 |
| `imports` | 1984 | | `INFERRED` | 581 |
| `uses_type` | 908 | | **`code_edges_uncertain`** | **10593** |
| `references` | 1 | | | |
| `inherits` | 0 | | | |
| **TOTAL** | **6948** | | | |

- `references` = 1: é a varredida legada por nome, que só roda quando o parser
  não devolveu arestas.
- `inherits` = 0 **é o número correto para este repo**: não há embedding de
  struct nem de interface no código (a ocorrência textual está dentro de uma
  string de fixture). A implementação é coberta por teste unitário.
- 10593 linhas em `code_edges_uncertain`: 9350 "destino não encontrado no
  repositório" (chamadas a `testing.T`, `log.Errorf`, … — fora do repo) e 1243
  "ambíguo: N símbolos com o nome X".

### Demais medições

| Item | Número |
| :--- | --- |
| Bundle da viewer (gzip, soma de `dist/assets/*.js`) | **144.667 / 163.840 bytes** |
| Testes da viewer | **95 passando** (21 novos) |
| Testes Go | suíte completa verde no gate de cada tarefa |
| Baseline do code stage | 331 arquivos / ~2595 símbolos → **349 / 6754** (o delta é o motor real: antes o `MockParser` regex extraía menos e não havia arestas) |
| **Ponte** (nós `code_symbol` / arestas `references_code`) | **2827** nós / **34 181** arestas (28 851 EXTRACTED / 5330 INFERRED) sobre 266 docs |
| **Idempotência da ponte** (2º passe) | 34 181 → **34 181** (delta 0) |
| **Impacto** (`.specs/ISSUES.md` → símbolos de código) | **788 símbolos** alcançados |
| **Travessia** — hub threshold | p99 = 1139 nós, p100 = 1284 → **145 nós podados** pelo hub |
| **Travessia** — orçamento (default 4000 tokens) | **33 nós**, 3988 tokens, truncado **com a dica de estreitar** |
| **Drift** — mudança de assinatura | `structural_changes` com `symbols_changed` e `attribution: "bridge"` |
| **Drift** — mudança de comentário | **ausente** de `structural_changes` (não é drift) |

> [!warning] A ponte é **barulhenta** por desenho: 34 181 arestas para 266 docs
> (o match pelo caminho do arquivo marca todos os símbolos do arquivo). Foi a
> decisão D1 da Fase 3 e está dentro do que a spec pede, mas sugere um **limite
> por documento** como evolução futura.

Evidência detalhada por fase:
`.specs/features/code-graph-real/validation.md` (Fase 2) e
`.specs/features/code-graph-real/validation-t13-t15-t17.md` (Fase 3).

---

## 4. O que ainda falta (nesta ordem)

| # | Pendência | Como fazer | Quem |
| :--- | :--- | :--- | :--- |
| 0 | **Corrigir os 8 defeitos da verificação** — D1 (migração) e D2 (`include_code` sem repo) são de severidade **alta** e quebram ACs; D3 (build `-tags treesitter` degrada), D4 (hub threshold poda o vizinho direto), D5 (`mem code-graph` descarta `--repo`), D6 (hash estrutural inclui linha), D7 (viewer cita a ADR errada), D8 (precedência env × config) | ✅ **feito** — `b3ed53f` (D1) · `fbd3d62` (D2) · `b34abd8` (D3) · `1dc845a` (D4) · `b84dfce` (D5) · `aadd791` (D6) · `e64e389` (D7) · `aac0c0b` (D8). Evidência D7/D8 em `validation-d7-d8.md` | — |
| 1 | **Revalidar** depois das correções: repetir as 16 hipóteses, com foco em D1/D2/D3 | `D:\.minimax\scratch\briefing-verificador.txt` | agente `verifier`, read-only |
| 2 | **Gate final único**, sem escritores ativos: `go build ./... && go vet ./... && go test -count=1 ./...` + `cd viewer && npm run typecheck && npm run test && npm run build` + medição do gzip | o gate de `internal/ci` só quebra se **aparecer um pacote Go novo** — nenhum worker criou um | gestor |
| 3 | **Ticar os 18 "Done when"** em `tasks.md` e revalidar `validate_tasks.py` | ✅ **feito** (commit `3012f76`) | gestor |
| 4 | **Consolidar a validação** num `validation.md` único (Fase 2 + Fase 3) | hoje são dois arquivos | gestor |
| 5 | **Fechar a ISSUE-027**: marcar C1..C12 como resolvidos, com o commit que fechou cada um | ✅ **feito** (`3012f76`) | gestor |
| 6 | **Atualizar `STATE.md`** com o estado da feature | ✅ **feito** (`3012f76`) | gestor |
| 7 | **Limpar o scratch** dos subagents: `D:\.minimax\scratch\subagents\*` (o clone do Graphify é referência — manter ou remover conforme a decisão do dono) | `mavis-trash.cmd` (deleção permanente é bloqueada) | gestor |
| 8 | **Remover as 2 notas-teste** que o verificador indexou por acidente no Postgres real (`licao-sem-postgres.md` e `licao-sem-postgres-3.md`, em `repo_central` do cofre central) | o verificador **não** apagou nada do store real; a remoção é decisão do dono | dono |
| 9 | **Push** de `develop` — **NÃO** fazer sem aval explícito do dono | 33+ commits à frente de `origin/develop` | dono |
| 10 | **Release v3.0.0 + PR `develop`→`main`** — adiado por decisão do dono ("só o push por enquanto") | `CHANGELOG.md` já tem o bloco `[Unreleased]` | dono |

---

## 5. Desvios declarados (nada escondido)

1. **`--no-embed` saiu do CLI.** Mudança de superfície: quem passava a flag
   agora recebe "flag provided but not defined". Decisão do gestor: uma flag
   que não muda nada não pode continuar sendo aceita. **Registrar no
   CHANGELOG** (já foi pelo worker de docs) e avisar no release.
2. **O boost CA-05 existiu duplicado** (uma cópia em `internal/store`, outra em
   `internal/mcp`) porque `store` não pode importar `mcp` (ciclo `mcp → db →
   store`). A reconciliação B faz o `mcp` reusar a implementação do `store` — a
   direção correta, já que `mcp` **importa** `store`.
3. **`include_code` tinha dois canais** (opções de busca + contexto) que
   podiam divergir. A reconciliação A passa a calcular o valor efetivo **uma
   vez** em `cmd/mem/main.go` e a escrever nos dois canais.
4. **`tasks.md` não foi atualizado pelos workers** (eu, gestor, ticks no fim):
   o índice do git é compartilhado entre workers paralelos e quatro workers
   mexendo no mesmo arquivo é uma condição de corrida. A convenção "atualizar
   o `tasks.md` no mesmo commit" fica suspensa por decisão operacional, não
   por esquecimento.
5. **`internal/mcp/code_handlers.go` e `internal/db/code_migration.go`**
   receberam exatamente duas rondas de escrita concorrentes (T5/T14 na Fase 1-3
   e T12/T15 na Fase 2-3). Verificar no diff final que não houve sobrescrita
   silenciosa.

---

## 6. Mapa dos gaps (ISSUE-027) → commit que fechou cada um

| Gap | O que era | Fechado por |
| :--- | :--- | :--- |
| **C1** | Motor AST nunca implementado (`newBackendParser` devolvia `ErrTreesitterDisabled` e não tinha call site) | `4f1182a` + `a342518` |
| **C2** | Não existia call graph; só `references` por nome, `imports` pulado | `b986c37` + `870a323` |
| **C3** | As 2 tools MCP de código mortas (handler `nil`) | `331be16` |
| **C4** | `mem code-index` indexava 0 arquivos; `mem index` nunca chamava o code pipeline | `1bf7e75` + `3cd03b5` |
| **C5** | Em Postgres o code graph ficava vazio (store PG aberto e descartado) | `33931cb` |
| **C6** | `code_*` sem coluna `repository` (fora da federação) | `0907dbd` |
| **C7** | Dois grafos disjuntos (`graph_edges` × `code_edges`) | `96a734b` |
| **C8** | `code_edges` sem `epistemic_status`; resposta sem confiança nem call-site | `5f52c2d` + `056a3f6` |
| **C9** | `include_code` era placeholder | `12f3751` |
| **C10** | A viewer não tinha superfície de código | `2a8d993` |
| **C11** | `mem drift` ↔ `ast_hash` desligados | `c1deac3` |
| **C12** | Flags mortas no CLI de código | `340c9cb` + `fd95961` |

---

## 7. Ambiente (o que muda com a mudança de pasta)

### Disco

O **C: encheu** durante a noite de builds: o cache de build do Go chegou a
**1,6 GB** e o `npm-cache` a **3,8 GB**, e com 80 MB livres um worker foi
morto no meio ("C: drive is full"). **Já corrigido:**

- `go clean -cache` (cache antigo freed) e `npm cache clean --force`;
- `GOCACHE` e `GOTMPDIR` gravados com `go env -w` para
  `D:\.minimax\scratch\gocache` e `D:\.minimax\scratch\gotmp` — **persistem
  para qualquer processo**, inclusive os workers;
- `GOMODCACHE` (570 MB) **ficou no C:** de propósito (re-download custoso);
- `%TEMP%`: só se limpa com `mavis-trash.cmd` — a deleção permanente é
  bloqueada pela policy.

C: com ~3,3 GB livres no fim; D: com ~153 GB livres.

### Checklist da mudança do repositório para D:

1. **O catálogo global de repositórios aponta para o caminho antigo.**
   `~/.memory/config.yaml` tem `repositories: - id: repo_271444ab0133 /
   path: C:\repository\my-memory`. **Atualizar o `path` para o caminho novo** —
   sem isso a entrada do catálogo deixa de casar com o repositório.
   O cascateamento casa por **ID ou caminho**
   (`internal/config/config.go:695-697`: `matchesID || matchesPath`), e a
   entrada do catálogo só sobrepõe o **storage** (engine + URL do Postgres),
   nunca o `repo_id`.
2. **Inconsistência pré-existente a conferir:** o `repo_id` do próprio repo
   (`.memory/config.yaml`) é `repo_e4c8f3b1a2d5`, e o `id` da entrada do
   catálogo global é `repo_271444ab0133` — **são diferentes**. Hoje isso
   funciona porque a entrada casa pelo **caminho**. O `repo_id` é gerado
   criptograficamente e **persistido** (`internal/config/config.go:458-465`,
   `GenerateRepoID`), não derivado do remoto nem do caminho — então ele **não
   muda** com a mudança de pasta. Ainda assim, o `id` divergente do catálogo
   é um risco latente de *split-brain* de identidade: se o repo for clonado
   em outro lugar, as duas entradas podem casar por caminho em vez de por ID.
   **Conferir e alinhar** o `id` do catálogo com o `repo_id` do repo.
3. `.memory/config.yaml` **do repo** — não tem `db:`/`path:` absoluto (só o
   `repo_id` e o `repository`), então ele move junto sem ajuste.
3. Os scratchs dos subagents (`D:\.minimax\scratch\subagents\*`) e o clone do
   Graphify **já estão no D:** e não se movem.
4. O `workspaceDir` das sessões de worker é o caminho **antigo** — sessões
   antigas não conseguem mais escrever no repo. Isso é desejado: as duas
   sessões zumbi (`mvs_79dfcc68…` e a da docs) já foram mortas por cota e
   receberam ordem de parada.
5. `GOCACHE`/`GOTMPDIR` já apontam para D: — o build continua funcionando no
   caminho novo sem ajuste.

### Armadilhas do ambiente já mapeadas

- **PowerShell 5.1**: sem `&&`; `2>$null` bloqueia (use `2>&1`);
  `cd <dir>; <comando que escreve>` é bloqueado; `rm` exige caminho literal
  absoluto; **`git commit -m` com aspas duplas quebra o commit** (o `\` não é
  escape) → usar `git commit -F <arquivo> -- <caminhos>`.
- **`gofmt -l .` lista ~200 arquivos** por CRLF do checkout local — **não é
  defeito**. O `.gitattributes` do repo é `* text=auto eol=lf` e **todo blob
  no índice é LF**. A prova válida de formatação é copiar os arquivos tocados
  para LF e rodar `gofmt -l` na cópia. **Nunca** `gofmt -w .`.
- **Cota do provedor (429/529) mata workers.** O runtime *retenta* o turno: um
  worker marcado como `failed` pode voltar a escrever no repo. Antes de
  commitar, confira `git status` — modificação em arquivo de worker é trabalho
  dele, não resíduo.
- **Índice do git compartilhado**: workers em paralelo commitaram com
  `git commit -F <msg> -- <caminhos explícitos>` para não levar o arquivo de
  outro worker junto no commit.
- `internal/ci` (`TestWorkflowCoversEveryPackage`) só quebra se **aparecer um
  pacote Go novo** sem step de teste na CI — workers paralelos **não** o
  quebram.

---

## 8. Itens herdados (pendentes, não pedidos novamente)

1. **Release v3.0.0 + PR `develop`→`main`** — adiado por decisão do dono.
   Tag atual: `v2.0.0`. O `CHANGELOG.md` tem um bloco `[Unreleased]` grande.
2. **ISSUE-026** — 5 decisões de produto (D1–D5) da matriz de paridade com o
   Obsidian, aguardando o dono.
3. **ISSUE-020** (ADR-058 / vis-network vendorizado: a ADR está `Accepted` mas
   o asset e o `go:embed` não existem) e as demais issues de produto.
4. **A ADR-047 segue com 0 de 3 gatilhos** — nada a fazer até o dono decidir
   sobre toolchain C.

---

## 9. Referências

| Artefato | Caminho |
| :--- | :--- |
| Spec (**29** ACs em EARS) | `.specs/features/code-graph-real/spec.md` |
| Tarefas (18, 3 fases) | `.specs/features/code-graph-real/tasks.md` |
| Evidência Fase 2 | `.specs/features/code-graph-real/validation.md` |
| Evidência Fase 3 | `.specs/features/code-graph-real/validation-t13-t15-t17.md` |
| **Verificação independente** (veredito PARTIAL + 8 defeitos) | `.specs/features/code-graph-real/verification.md` |
| ADR da decisão | `docs/adr/061-dois-conceitos-de-memoria-vault-central-e-code-graph.md` |
| Tracker dos 12 gaps | `.specs/ISSUES.md` (ISSUE-027) |
| Índice de ADRs | `docs/adr/README.md` |
| Estado do projeto | `.specs/STATE.md` |

> [!info] Este documento é um retrato de 2026-10-24. Ao retomar, confira
> `git log --oneline -25` e `git status` antes de trusting os números acima.
