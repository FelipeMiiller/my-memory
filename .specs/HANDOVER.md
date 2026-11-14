---
title: "Handover — my-memory: onde a rodada do Vault Central parou (2026-11-13)"
category: resource
summary: "Retomada completa do my-memory em 2026-11-13: diagnóstico medido do Cofre Central, as 3 specs de execução escritas (provider ONNX, doctor de links, vetor zero), decisões já tomadas, decisões em aberto, ambiente, comandos e armadilhas."
tags: [my-memory, handoff, vault-central, onnx, doctor, spec, issue-040, adr-062]
---

# Handover — `my-memory`: onde paramos (2026-11-13)

> **Retrato:** 2026-11-13 18:40 (BRT) · **Branch:** `develop` · **HEAD:** `e64bb28` ·
> **Working tree:** limpa · **Origin:** `github.com/FelipeMiiller/my-memory` ·
> **Push:** `988c6c4..e64bb28` feito, CI verde.
>
> **Este documento é a fonte da verdade para retomar.** O `.specs/STATE.md` tem o
> resumo; aqui está o retrato completo (diagnóstico, specs, decisões, ambiente,
> comandos, armadilhas).

---

## 1. TL;DR — em 8 linhas

1. O dono fixou o foco: **"o foco agora total é a gente fazer o Vault Central
   funcionar"**. O lado de repositório (multi-linguagem, `graph.json`) fica para
   depois.
2. O Cofre Central **não está quebrado**: 92 notas destiladas indexadas
   (94 docs / 481 chunks), **os 481 chunks têm vetor**, e a busca semântica
   responde — **desde que o binário tenha a tag `onnx`**.
3. A causa raiz do "não está funcionando" é o **artefato distribuído**: o
   release compila `-tags "sqlite_fts5"` (sem `onnx`), a CI não tem job com a
   tag, e **todo binário oficial grava vetor zero** ao indexar.
4. Achado **novo** nesta rodada (ISSUE-040): sem provider o indexador grava
   **vetor zero** em vez do chunk sem embedding, e o check de dessincronizado
   (`embedding IS NULL`) **não vê nada** — a degradação é invisível.
5. As **3 specs de execução** do Vault Central estão **escritas e validadas**
   (0 erro nos validadores), em `Draft`, prontas para a fase Execute:
   `fix-release-onnx-provider`, `fix-doctor-unindexed-links`,
   `fix-zero-vector-chunks`.
6. O `mem doctor` do cofre marca **138 links que existem em disco** como
   "mortos" (o material bruto excluído do índice por decisão) → 60/100 num
   vault saudável. A spec 2 resolve isso com checagem em disco.
7. **Nada foi implementado** nesta rodada: é só documentação (specs + tracker).
8. **Decisão em aberto que espera o dono:** o que fazer com o peso da
   governança do repo (62 ADRs + 160 specs = 43% das linhas). Ver §7.

---

## 2. O que foi feito na rodada (2026-11-13)

| Commit | O quê |
| :--- | :--- |
| `12fbca0` | **ADR-062** — contrato dos dois cofres, critério de "pronto" por lado, matriz de padrões do `ai-memory` + `graphify`, causa raiz medida, ordem de execução. Índice em `docs/adr/README.md`. |
| `988c6c4` | **ISSUE-038** — a aresta de tag (`tagged_as`) deixou de ser dead link (o alvo passou a ser reconhecido por `document_tags` e por símbolo de código). 4 testes (SQLite + Postgres real) + **mutante provado**. |
| `e64bb28` | **As 3 specs do Vault Central** + ISSUE-040 aberta + tracker/STATE/ADR-062 atualizados + typo da ADR-062 corrigido. |

Antes disso, na mesma sessão: `273330a` (lookups de resolução em lote),
`2f2b55c` e `9d06e31` (correções de registro no tracker: o watcher **não**
aborta a indexação inteira; números medidos no lugar de estimativas).

**Fora do repo, na máquina:** o `bin/mem.exe` local passou a ser build
`CGO_ENABLED=1 go build -tags onnx` (35.634.166 bytes) — com ele a busca
responde `[vector:1]`. O binário roda **sem** o MSYS2 no PATH.

---

## 3. O diagnóstico do Cofre Central (medido, não estimado)

### 3.1 O ambiente real

| Item | Valor |
| :--- | :--- |
| Cofre central | `G:\My Drive\central-memory` (`central_vault.path` no `~/.memory/config.yaml`) |
| `repo_id` | `repo_central` |
| Conteúdo | 796 `.md` = **92 notas destiladas** + 704 de material bruto (`catalog-skills/skills/**`, **excluído do índice** por decisão registrada no `config.yaml` do cofre) |
| Índice (Postgres) | **94 docs / 481 chunks / 238 nós / 596 arestas** — os 481 chunks **têm** vetor |
| Dead links (depois da ISSUE-038) | **138** (68 `links_to` + 70 `tagged_as`) — **todos existem em disco** |
| Health Score | **60/100** (penalidade de dead link tem teto de 40) → **100/100** esperado após a spec 2 |

### 3.2 As três causas raiz

1. **O binário distribuído não tem o provider** (ISSUE-037).
   `internal/embedder/builtin.go:1` está atrás de `//go:build onnx`; o release
   compila `-tags "sqlite_fts5"` (`.github/workflows/release.yml:118,131`) e a
   CI não tem job com a tag. Resultado: **todo binário oficial roda em FTS
   lexical**.
2. **O provider precisa de artefatos que ninguém baixa.** Modelo
   (`~/.memory/models/paraphrase-multilingual-MiniLM-L12-v2/model_int8.onnx`,
   118 MB) e DLL do ONNX Runtime
   (`~/.memory/models/onnxruntime/onnxruntime.dll`, 16 MB) precisam estar no
   disco. `internal/embedder/model_downloader.go` **nunca foi escrito** (a
   T11/T12 da spec `embedder-builtin-onnx-minilm` ficou `Pending`) e o
   instalador só baixa o zip do release.
3. **O chunk indexado sem provider grava vetor zero** (ISSUE-040, achado
   novo). `cmd/mem/main.go:1913` substitui o embedding ausente por
   `make([]float32, 384)`. O check de dessincronizado do doctor procura
   `embedding IS NULL` (`internal/store/doctor.go:176`) → **não vê**. A busca
   vetorial sobre vetor zero devolve ruído com cara de resultado (distância
   de cosseno indefinida). O SQLite **já sabia** gravar chunk sem vetor
   (`internal/db/store.go:357`, `if len(vec) > 0`) — foi o substituto zero que
   estragou os dois engines.

> Por que o gap 3 não apareceu antes: o binário **local** tem a tag `onnx`.
> O oficial não. O mesmo cofre, indexado pelo artefato distribuído, fica com
> 481 chunks de vetor zero — e nada no produto diz isso.

### 3.3 O lado de repositório (medido, para contexto)

`repo_e4c8f3b1a2d5`: 293 docs / 2.615 chunks / 39.533 arestas / 3.339 nós ·
code graph 381 arquivos / 7.418 símbolos / 7.785 arestas / 11.556 arestas
incertas / 39.376 arestas de ponte · **2 dead links reais** (`[[ADR-035]]`) ·
Health Score **90/100**. Drift estrutural funcionando (mudança cosmética
suprimida, mudança de assinatura detectada e atribuída).

---

## 4. As três specs prontas para Execute

Todas em `.specs/features/`, com `spec.md` + `tasks.md`, validadas pelo fluxo
spec-driven (**0 erro** em `validate_spec.py` e `validate_tasks.py`).

| Spec | Gap | ACs | Tarefas | Fases | Issue |
| :--- | :--- | :---: | :---: | :---: | :--- |
| `fix-release-onnx-provider` | o binário oficial carrega o provider | 11 | 9 | 3 | ISSUE-037 |
| `fix-doctor-unindexed-links` | "fora do índice por decisão" ≠ "link rompido" | 9 | 7 | 3 | ISSUE-039 |
| `fix-zero-vector-chunks` | chunk sem provider não grava vetor zero | 7 | 8 | 3 | ISSUE-040 |

**O que cada uma entrega**

- **`fix-release-onnx-provider`** — release Linux e Windows com a tag `onnx`
  (o job Windows **verifica** a toolchain C e falha explicitamente em vez de
  publicar binário mudo); job de CI com a tag; `mem embed download` (modelo +
  DLL do ONNX Runtime 1.29.0) com download verificado e rename atômico;
  `mem embed doctor`; o instalador (`scripts/install.ps1` + `scripts/install.sh`)
  invoca o download; e a degradação vira **um** aviso consolidado em vez de uma
  linha por documento.
- **`fix-doctor-unindexed-links`** — o doctor classifica cada link em
  **rompido** / **fora do índice por decisão** / **não verificável**. Raiz do
  vault: `--vault` > `central_vault.path` do config global > cwd. Alvo
  resolvido em disco com a semântica do Obsidian (basename em qualquer
  profundidade). Categoria nova **não** penaliza o Health Score; a não
  verificável penaliza e marca o score como **provisório**; o `--fix` apaga
  só o que o relatório provou quebrado.
- **`fix-zero-vector-chunks`** — o chunk sem provider é gravado **sem
  embedding** (NULL no Postgres; nenhuma linha em `chunks_vec` no SQLite); o
  doctor trata vetor zero como dessincronizado (além do NULL); a busca
  vetorial não devolve chunk sem vetor real; e um E2E (tag `onnx`) prova que a
  nota volta **por sentido** e que o reindex do zero não perde vetor — os
  critérios §B-3 e §B-4 da ADR-062, que nenhum spec cobria.

**Ordem sugerida de Execute**: `fix-release-onnx-provider` → (o binário oficial
precisa do provider antes de o cofre poder ser reindexado com vetor) →
`fix-zero-vector-chunks` → `fix-doctor-unindexed-links` (o doctor só tem o que
medir depois que os vetores são reais). Cada uma cabe em **um lote** (≤ ~8
tarefas) → execução inline, sem sub-agente.

**Como executar**: ativar a skill `tlc-spec-driven` **pelo nome** e seguir o
fluxo Execute dela (ciclo por tarefa, commit atômico, Verifier com sensor de
discriminação ao final). As tasks já trazem `Depends on`, `Tests`, `Gate` e a
matriz de cobertura.

**Gotcha do validador** (custou uma rodada): o `validate_tasks.py` **cobra a
seta entre fases** no diagrama do Execution Plan, porque ele não infere a fase
a partir do plano (só do `### Tn:` no Task Breakdown). As 3 specs desenham
`T2 -> T3 -> T4 -> T5` na fase 2, por exemplo, e não `T3 -> T4 -> T5`. É uma
correção possível no `validate_tasks.py`/na referência `tasks.md` do skill —
**ainda não propus nenhuma alteração** (proposta, não ação).

---

## 5. Decisões do dono — o que JÁ está decidido (não reabra)

- **Direção dos dois cofres** (ADR-061/062): o central responde *"o que eu
  aprendi"* (Obsidian + busca semântica e vetorial); o de repositório responde
  *"como este código se conecta"* (code graph).
- **Release**: compilar com a tag `onnx` **e** o instalador baixar a DLL do
  ONNX Runtime.
- **Wikilinks**: distinguir "fora do índice por decisão" de "link rompido" —
  **não** remover os links do material bruto.
- **Multi-linguagem**: o escopo de código do vault de repositório fica **só em
  Go**, de propósito (o backend real do build padrão é Go puro; as outras
  linguagens cairiam no `MockParser` e envenenariam o `ast_hash`).
- **Não implementar**: ISSUE-032 B (pgx), ISSUE-033 C (os 5 relatórios de
  `# Validation Plan`), `mem repos --fix`. ISSUE-030 (fallback multi-linguagem)
  fica como está.
- **Promoção `develop` → `main`**: só via "alertar versão estável". A v3.0.0
  segue preparada e **não** promovida; o que é novo está em `[Unreleased]`.
- **Correções também pedem spec** (sem patch solto); **achado errado → ID
  rastreável** (issue), nunca corrigido em silêncio.

---

## 6. O que a ADR-062 §B define como "Cofre Central pronto"

> (1) a busca semântica responde com **fonte `vector`** no binário que o dono
> usa; (2) `mem doctor` do cofre **≥ 90/100**; (3) `mem lesson` grava e a nota
> é recuperável **por sentido**, não só por palavra; (4) as notas destiladas
> são **reindexáveis do zero sem perder o vetor**.

| Critério | Hoje | Quem fecha |
| :--- | :--- | :--- |
| (1) fonte `vector` | ✅ **localmente** (binário com a tag), ❌ **no oficial** | spec 1 |
| (2) doctor ≥ 90 | ❌ 60/100 | spec 2 |
| (3) recuperável por sentido | sem prova | spec 3 (E2E) |
| (4) reindex sem perder vetor | sem prova | spec 3 (E2E) |

---

## 7. Decisão em aberto que espera o dono: o peso da governança

> **✅ DECIDIDO em 2026-11-14** — o dono escolheu **(a) caminho A**: *inventário →
> podar o morto → simplificar a governança*; e **(b) opção a**: o bucket
> GAP-6 a GAP-10 é decidido **item a item, com medição antes de cada corte**.
> Registrado na [ADR-065](docs/adr/065-o-caminho-de-limpeza-do-repo-e-inventario-podar-simplificar.md).
> **Próximo passo: o inventário read-only** (nenhuma escrita, nenhuma decisão,
> nenhum corte) — ele mede o que existe e produz o inventário que a fase de
> podar vai usar. As seções abaixo ficam como **registro da medição** que
> sustentou a decisão.

**Pergunta:** o repo tem 296 arquivos de documentação (~34.500 linhas: 62 ADRs,
160 specs, 74 docs) contra 383 arquivos Go (78.942 linhas) — **43% do código**.
Mais ~3.000 linhas de código morto. O dono perguntou se deveria apagar ADRs e
specs e recomeçar.

**Meu parecer (medido, não opinado):** o repo **não está bagunçado** — está
**prometendo mais do que entrega**. Apagar ADRs/specs remove a **prova** do
gap, não o gap. E é **reversível**: `git log -- docs/adr/` preserva tudo.

**Ordem que recomendo:** inventário (read-only) → podar o **morto** →
simplificar a governança → *(opcional)* zerar o HEAD.

**Opções que foram colocadas ao dono** (a pergunta está em aberto, sem
resposta até agora):

- *(a) caminho de limpeza* — **A)** inventário → podar → simplificar
  *(recomendado)* · **B)** inventário → zerar o HEAD (ADRs/specs só no
  histórico do git) · **C)** só inventário.
- *(b) o bucket "prometido mas não entregue"* (GAP-6 a GAP-10 da matriz da
  ADR-062: passe semântico de PDF, captura de sessão, handoff tipado, perfil
  cross-project, decaimento por tier, `graph.json` portátil, eval de
  recuperação) — **a)** eu decido item a item · **b)** poda agressiva ·
  **c)** corte pelo uso diário.

**Medições que apoiam o parecer** (repetidas aqui para o novo chat não
recalcular): 174 funções inalcançáveis (`deadcode` com testes — usar como
indicador, não veredito: ele não vê `init()`, reflexão nem interfaces);
`internal/supervisor` (1.627 linhas) e `internal/event_runtime` (1.327 linhas)
**sem nenhum importador de produção**; `embedder/tokenizer.go` (119) e
`embedder/ollama.go` (59) mortos.

---

## 8. Ambiente (medido nesta rodada)

| Item | Valor / como |
| :--- | :--- |
| Go | 1.27.0 (`go.mod` manda) · `CGO_ENABLED=0` por default **nesta máquina** |
| Toolchain C | `C:\msys64\ucrt64\bin\gcc.exe` — **fora do PATH** |
| Postgres | Neon, endpoint **direto** (sem `-pooler`) no `~/.memory/config.yaml` |
| Config global | `C:\Users\Felipe\.memory\config.yaml` (`storage.engine: postgres`, `postgres_url`, `central_vault.path: G:\My Drive\central-memory`) |
| Binário local | `D:\repository\my-memory\bin\mem.exe` = build **com** `-tags onnx` (gitignored) |
| Modelo / DLL | `~/.memory/models/paraphrase-multilingual-MiniLM-L12-v2/model_int8.onnx` · `~/.memory/models/onnxruntime/onnxruntime.dll` |
| Relógio do sistema | saltou de 29/10 para 13/11 durante a sessão — commits datados conforme |
| Arquivo-âncora | `C:\repository\my-memory\_ANCORA_WORKSPACE.txt` — **precisa continuar existindo** (sessão ancorada) |

---

## 9. Comandos que funcionam (copiar e usar)

```powershell
# --- estado do repo
git -C D:\repository\my-memory status --short
git -C D:\repository\my-memory log --oneline -5

# --- suíte completa (leva ~4 min; internal/db ~114s, internal/store ~76s, cmd/mem ~134s)
$env:MY_MEMORY_PG_TEST='1'
$env:MY_MEMORY_PG_URL = ([regex]::Match((Get-Content "$env:USERPROFILE\.memory\config.yaml" -Raw), 'postgres_url:\s*(\S+)').Groups[1].Value -replace '-pooler\.', '.')
go -C D:\repository\my-memory test -count=1 ./...
go -C D:\repository\my-memory build ./...
go -C D:\repository\my-memory vet ./...

# --- build COM o provider semântico (o gcc do MSYS2 só precisa estar no PATH no build)
$env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"; $env:CGO_ENABLED='1'
go -C D:\repository\my-memory build -tags onnx -o D:\repository\my-memory\bin\mem.exe ./cmd/mem

# --- index / doctor / drift (Postgres, endpoint direto)
& 'D:\repository\my-memory\bin\mem.exe' index
& 'D:\repository\my-memory\bin\mem.exe' doctor --storage=postgres --repo repo_central
& 'D:\repository\my-memory\bin\mem.exe' doctor --storage=postgres --repo repo_e4c8f3b1a2d5
& 'D:\repository\my-memory\bin\mem.exe' drift --storage=postgres --repo repo_e4c8f3b1a2d5 --uncovered --since 'adfb8cc~5..adfb8cc' --json

# --- validadores do fluxo spec-driven
python .agents\skills\tlc-spec-driven\scripts\validate_spec.py .specs\features\<slug>\spec.md
python .agents\skills\tlc-spec-driven\scripts\validate_tasks.py .specs\features\<slug>\tasks.md
python .agents\skills\tlc-spec-driven\scripts\validate_state.py <slug> --root D:\repository\my-memory
python .agents\skills\tlc-spec-driven\scripts\validate_state.py --root D:\repository\my-memory   # 6 erros pré-existentes

# --- código morto (indicador, não veredito; não escreve no repo)
$env:GOPATH='D:\.minimax\scratch\gopath'
go run golang.org/x/tools/cmd/deadcode@latest D:\repository\my-memory\...

# --- commit sem BOM (PS 5.1 Out-File -Encoding utf8 grava BOM)
[System.IO.File]::WriteAllText('D:\repository\my-memory\.git\COMMIT_MSG_TMP.txt', $msg, (New-Object System.Text.UTF8Encoding($false)))
git -C D:\repository\my-memory commit -F 'D:\repository\my-memory\.git\COMMIT_MSG_TMP.txt'
```

---

## 10. Armadilhas do ambiente (NÃO repetir)

1. `Set-Location` + escrita/apagamento no mesmo comando → **bloqueado** pela
   policy ("write/delete targets cannot be verified safely after a command
   working directory change"). Use `go -C` / `git -C`.
2. `Copy-Item` de/para dentro do workspace → bloqueado ("recursive
   write/delete target"). Para mutação temporária num arquivo do repo, use
   `[System.IO.File]::WriteAllText` e restaure o texto original na mesma chamada.
3. `Remove-Item` bloqueado; usar `mavis-trash.cmd` (aceita vários caminhos;
   arquivo já ausente = erro "missing", ignorar). O `rm --` direto é bloqueado
   ("requires the targets-only trash marker").
4. `Out-File -Encoding utf8` / `Set-Content -Encoding UTF8` gravam **BOM** em
   PS 5.1 → usar `WriteAllText(..., UTF8Encoding($false))`.
5. `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...` devolve **imports
   de produção** (testes não contam) → um pacote só importado pelos testes
   aparece como "sem importador de produção" ✓.
6. `deadcode` sem `-test=false` inclui testes ✓; com `-test=false`
   super-reporta (o `pgtest.Capture` é chamado em `init()`).
7. `gofmt -l .` no worktree local mistura CRLF → copiar para o scratch,
   converter para LF e rodar `gofmt -l` na cópia.
8. `embed_epoch` é coluna **só do SQLite**; no Postgres o vetor está em
   `chunks.embedding` (pgvector, `vector(384)` **sem** NOT NULL).
9. `mem code-stats` **não** aceita `--storage`; `mem drift` aceita.
10. Python: usar `python` (não `python3`); `PYTHONIOENCODING=utf-8` para saída
    com acento.
11. **Subagente pode morrer com 429** (`BYOK provider minimax_api upstream
    error: 429 Token Plan usage limit reached`). Quando isso acontecer:
    **não redispatchar** — retomar no pai (ler o output do filho antes, para
    não duplicar).

---

## 11. Arquivos-chave

| Arquivo | O que é |
| :--- | :--- |
| `docs/adr/062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md` | contrato dos dois cofres, critério de pronto, matriz de gaps, causa raiz |
| `docs/adr/061-dois-conceitos-de-memoria-vault-central-e-code-graph.md` | separação dos dois conceitos |
| `.specs/features/fix-release-onnx-provider/` | spec 1 (ISSUE-037) |
| `.specs/features/fix-doctor-unindexed-links/` | spec 2 (ISSUE-039) |
| `.specs/features/fix-zero-vector-chunks/` | spec 3 (ISSUE-040) |
| `.specs/ISSUES.md` | tracker; ISSUE-037 a ISSUE-040 são as abertas desta rodada |
| `.specs/STATE.md` | log de decisões + handoff (este documento é o retrato completo) |
| `.specs/features/fix-doctor-tag-edges/` | a correção da ISSUE-038 já entregue (`988c6c4`) |
| `internal/embedder/builtin.go` · `cmd/mem/builtin_embedder.go` | o provider ONNX e o wiring dos paths dos artefatos |
| `cmd/mem/main.go` | `resolveEmbedder` (`:1455`), indexação (`:1888` Postgres, `:2129` SQLite), busca |
| `internal/store/doctor.go` · `internal/db/graph.go` | o doctor nos dois engines |
| `internal/db/store.go:357` | `InsertChunk` do SQLite — **já sabe** gravar chunk sem vetor |
| `docs/CENTRAL_VAULT.md` | doc do cofre central (seção 4.2 trata do fallback léxico) |
| `.github/workflows/release.yml` · `ci.yml` | onde a tag `onnx` precisa entrar |
| `scripts/install.ps1` · `scripts/install.sh` | o instalador (ADR-032) |

---

## 12. Pendências do lado de repositório (depois do cofre central)

- **GAP-4 — multi-linguagem**: hoje o backend real do code AST é **só Go**
  (`internal/codeast/go_parser.go`); TypeScript/JavaScript/Python caem no
  `MockParser` (símbolos sintéticos, sem assinatura nem aresta estrutural).
  Decisão pendente: tree-sitter (ADR-047) × Go stdlib por linguagem. O dono já
  fixou o **escopo em Go** por ora — o que falta é o backend.
- **GAP-5 — artefato portátil `graph.json`**: o grafo do `graphify` é legível
  por qualquer ferramenta, sem servidor. Hoje não existe.
- **GAP-6 a GAP-10** — o bucket "prometido mas não entregue" (§7).
- **ISSUE-030** (fallback multi-linguagem) fica como está.

---

## 13. Como retomar (protocolo)

1. Ler este documento **e** `.specs/STATE.md` (decisões + handoff).
2. Reconciliar com o git: `git -C D:\repository\my-memory status --short` e
   `git log --oneline -5` — **a evidência ganha do snapshot**.
3. Se a tarefa for o Vault Central: abrir
   `.specs/features/fix-release-onnx-provider/` e seguir o Execution Protocol do
   `tasks.md` (ativar a skill `tlc-spec-driven` **pelo nome**).
4. Se a tarefa for a limpeza do repo: responder a pergunta de §7 **antes** de
   mexer — a ordem é inventário (read-only) → podar → simplificar.
5. Regras que continuam valendo: sem patch solto (correção pede spec), achado
   errado → issue com ID, "menos invenção" (implementar o que foi pedido; se
   faltar algo, **perguntar em uma frase**), commits atômicos em Conventional
   Commits, e o quality gate antes de dizer "pronto".
