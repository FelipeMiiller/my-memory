# ADR-062: Contrato dos dois cofres, prontidão por lado e o que o `ai-memory` ensina

- **Date**: 2026-11-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: cofre-central, code-graph, ai-memory, graphify, embeddings, onnx, cgo, prontidao, gaps

## Context and Problem Statement

O dono fixou a direção do `my-memory` em duas frases: o **Cofre Central** segue
Obsidian (tags, wikilinks, propriedades) e ganha a **busca vetorial semântica**;
os **repositórios** seguem "a mesma ideia do
[Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify)".

A ADR-061 já separou os dois conceitos, e a feature `code-graph-real` construiu
o motor do lado de repositório. O que faltava era uma **expectativa operacional
explícita** de cada lado (o que é "pronto"), o **balanço honesto** contra os
dois projetos de referência, e a lista de **gaps** que ainda impedem o dono de
usar o Cofre Central como memória.

Esta ADR registra: (1) o contrato de cada lado, (2) o critério de prontidão de
cada lado, (3) a matriz de padrões dos dois projetos de referência com o
estado real do `my-memory` (medido, não estimado), (4) a **causa raiz** de o
Cofre Central não funcionar como memória, e (5) a ordem de execução.

### Os dois projetos de referência

**`Graphify-Labs/graphify`** (v8): transforma uma code base + docs + SQL +
PDFs num grafo **consultável em vez de greppado**. Código por **tree-sitter
AST** (determinístico, local, sem LLM); docs/PDFs/imagens por um passe semântico
do modelo do assistente. Toda aresta carrega `EXTRACTED`/`INFERRED`/`AMBIGUOUS`
com score numérico. God nodes, comunidades (Leiden), `query`/`path`/`explain`.
Saída: `graph.json` (node-link) + `graph.html` + `GRAPH_REPORT.md`. **Sem
vetores**: "the graph structure is the similarity signal".

**`akitaonrails/ai-memory`** (Rust, binário único): memória de agente com
**wiki Markdown versionada em git como fonte da verdade** e o banco como índice
derivado. Captura automática por hooks de ciclo de vida → observações
sanitizadas → consolidadas em páginas Markdown legíveis. **Zero chamadas de
LLM por padrão** (captura, busca e handoff funcionam sem chave de API; o LLM é
opcional e habilita a busca semântica). Handoff é um protocolo tipado, com dono
e **claim único**. Perfil cross-project. Decaimento por tier com retenção
ponderada por acesso, compactação extrativa, dedup e flag de contradição —
**tudo zero-LLM**, e as partes que reescrevem/apagam ficam **off por padrão**.
Recuperação: FTS5 + entidade + grafo + vetor opcional fundidos em RRF, com
fallback bruto limitado. Nunca apaga: *supersede-not-delete* + `restore-page`.
A lição central está em `docs/prior-art-implementation-findings.md`.

### O estado medido do `my-memory` (2026-11-13)

- **Cofre Central** (`G:\My Drive\central-memory`): 796 arquivos `.md`; **92 notas
  destiladas** + material bruto (704 arquivos de `catalog-skills/skills/**`)
  **excluído do índice** por decisão registrada no próprio `config.yaml`. Índice
  no Postgres (`repo_central`): **94 documentos / 481 chunks / 238 nós / 596
  arestas** — e **os 481 chunks têm vetor** (a coluna `embed_epoch` é só do
  SQLite; no Postgres o vetor está em `chunks.embedding`).
- **Busca semântica**: funciona **se** o binário tiver a tag `onnx` — medido:
  com `-tags onnx` a busca responde `Fontes RRF: [vector:1]`; sem a tag, responde
  `[fts:1]` e avisa "provider builtin indisponível". **O binário oficial nunca
  tem a tag** (ver GAP-1).
- **Health Score do cofre: 60/100**, com **148 "dead links"**: 80 são arestas
  `tagged_as` (o alvo da tag é gravado sem o prefixo `#`, e o check de dead link
  só isenta alvos que começam com `#`) e 68 são `links_to` para notas do
  **material bruto excluído** (resolvem no Obsidian, não no índice).
- **Code graph do repo**: 376 arquivos, 7 418 símbolos, 7 785 arestas, 11 556
  incertas, 39 376 arestas de ponte; drift estrutural funcionando (mudança
  cosmética suprimida, mudança de assinatura detectada e atribuída).

## Decision Drivers

- **DR-1 — Um código, dois conceitos, sem ambiguidade.** O cofre central
  responde "o que eu aprendi"; o vault de repositório responde "como este código
  se conecta". Nenhum herda o modelo do outro.
- **DR-2 — O que o dono promete precisa existir no binário que ele usa.** Uma
  capacidade documentada e ausente do artefato distribuído é pior que ausente:
  degrada em silêncio.
- **DR-3 — Markdown é a fonte da verdade; o índice é derivado.** (Mesma tese do
  `ai-memory`; o `my-memory` já é assim nos dois lados.)
- **DR-4 — Zero-LLM por padrão; LLM como upgrade explícito.** (A mesma doutrina
  do `ai-memory`, e ela cabe no `my-memory`: o provider embutido é local.)
- **DR-5 — O grafo do repositório é a exceção que prova a regra:** código **vira
  grafo**, não vira vetor. A busca semântica fica para a **prosa** (central e
  notas do repo).
- **DR-6 — Nada de "quase funciona": cada gap fica rastreado, com dono e
  critério de pronto.**

## Considered Options

1. **Status quo (só ADR-061)** — descartado: a ADR-061 separa os conceitos mas
   não diz o que é "pronto" nem por que o dono não consegue usar o cofre.
2. **Adotar o `ai-memory` como motor** — descartado como motor: ele não faz
   code graph, e o `my-memory` já é o grafo de código com向量, multi-repo e
   drift. **Adotado como padrão de projeto** (fonte da verdade Markdown, zero-LLM,
   *supersede-not-delete*, handoff tipado, skill por agente).
3. **Trocar o embedder por um serviço HTTP (Ollama/API)** — considerado: resolve
   o CGO sem toolchain. Descartado por ora: `ollama` não está instalado nesta
   máquina, a API custa dinheiro e tira a garantia de "tudo local"; o modelo
   (MiniLM int8, 118 MB) e a DLL **já estão no disco**.
4. **Manter o Go puro e aceitar FTS** — descartado: contradiz a promessa
   explícita do dono ("busca vetorial semântica") e a ADR-035.

## Decision Outcome

**A. Contrato por lado**

| | Cofre Central | Vault de Repositório |
| :--- | :--- | :--- |
| Pergunta | "o que eu aprendi sobre isso?" | "como este código se conecta?" |
| Unidade | a nota (frontmatter + `[[wikilinks]]` + `#tags`) | o símbolo e a aresta |
| Busca | semântica e vetorial (FTS5 + TurboQuant 4-bit + RRF) | travessia do grafo (`code-search`, `code-graph`, `path`, `inspect`, `impact`, `clusters`, `hubs`, `drift`) |
| Fonte da verdade | Markdown no Drive | o código + as notas do repo |
| Identidade | `repo_central` | `repo_id` por repositório, no mesmo Postgres |
| Verificação | `mem doctor` (Health Score) + drift | drift por `ast_hash` |

**B. Critério de "pronto" por lado** (é isto que fecha a discussão de escopo)

- **Cofre Central pronto** ⇔ (1) a busca semântica responde com **fonte
  `vector`** no binário que o dono usa; (2) `mem doctor` do cofre **≥ 90/100**;
  (3) `mem lesson` grava e a nota é recuperável **por sentido**, não só por
  palavra; (4) as notas destiladas são reindexáveis do zero sem perder o
  vetor.
- **Vault de Repositório pronto** ⇔ (1) o grafo responde "quem chama isto, o que
  isto chama, qual nota documenta isto, e a nota está desatualizada?"; (2) o
  drift distingue cosmética de estrutural; (3) **para as linguagens que o dono
  usa de verdade** — hoje, isso é Go e TypeScript/JavaScript.

**C. A causa raiz de "o Vault Central não está funcionando"** (medida)

O cofre **não** está quebrado: o índice está completo (94/92 docs), os 481
chunks **têm** vetor, e a busca semântica **responde** — desde que o binário
seja compilado com a tag `onnx`. O que falha é o **binário**:

- `internal/embedder/builtin.go` (o provider ONNX MiniLM) está atrás de
  `//go:build onnx`;
- o **release** compila com `CGO_ENABLED=1 go build -tags "sqlite_fts5"` — **sem**
  a tag `onnx`;
- a **CI** não tem job com a tag `onnx` (tem o do `treesitter`), ou seja, o
  provider **nunca é compilado nem testado** no pipeline;
- sem a tag, todo comando avisa "provider builtin indisponível" e indexa em FTS
  lexical — **em silêncio quanto ao resultado**.

O toolchain **existe** nesta máquina (`C:\msys64\ucrt64\bin\gcc.exe`), apenas
fora do PATH. Medido: `CGO_ENABLED=1 go build -tags onnx` compila (35,6 MB) e a
busca passa a responder `[vector:1]`.

**Consequência que importa:** cada nota reindexada pelo binário oficial perde o
vetor (o chunk é regravado sem embedding) → **a busca semântica do cofre
decai nota a nota, em silêncio**. É a forma exata de "a aplicação não está
funcionando".

**D. Ordem de execução (o foco agora é o Cofre Central)**

1. **GAP-1 — o binário distribuído carrega o provider semântico.** Build com a
   tag `onnx` no release + job de CI com a tag (copiando o padrão do
   `codeast-cgo-on`), e o caminho de instalação documentando o toolchain
   (mingw/MSYS2) quando ele não existir.
2. **GAP-2 — o doctor não conta aresta de tag como link morto.** 80 dos 148
   "dead links" do cofre são `tagged_as` (o alvo de uma aresta de tag é uma tag
   ou uma referência a nota, não um documento indexado) e os outros 68 são
   `links_to` para o material bruto. É bug de **medição**: o Health Score do
   cofre está penalizado por algo que não está quebrado. *(Parcialmente
   corrigido em 2026-11-13: o check passou a conhecer a tag — 148 → 138; o que
   resta é a checagem em disco do GAP-3.)*
3. **GAP-3 — os 68 wikilinks para o material bruto.** Resolvem no Obsidian, não
   no índice. Decisão de política (resolver contra o disco / aceitar como
   "não indexado por decisão" / remover os links).
4. **Depois**: o lado de repositório, com o gap grande (**GAP-4**:
   multi-linguagem — hoje o backend real é só Go; para TypeScript/JavaScript o
   grafo ainda cai no `MockParser`, que não produz assinatura nem aresta
   estrutural) e o **GAP-5** (o artefato portátil `graph.json`, para o grafo
   ser legível por qualquer ferramenta, sem servidor).

**E. Matriz de padrões (o que o `graphify` e o `ai-memory` trazem que o
`my-memory` ainda não tem)**

| Padrão (origem) | Estado no `my-memory` | Gap |
| :--- | :--- | :--- |
| Código como grafo, toda aresta explicada (graphify) | ✅ code graph (Go), `EXTRACTED`/`INFERRED`, call-site | **GAP-4** multi-linguagem |
| Artefato portátil `graph.json` (graphify) | ❌ | **GAP-5** |
| Passe semântico de PDF/imagem (graphify) | ⚠️ só `.md` com FTS+TurboQuant | GAP-6 |
| Captura automática de prompt/tool-call (ai-memory) | ⚠️ `mem watch`/`mem hook` indexam **arquivos**, não a sessão | GAP-7 |
| Consolidação em página legível (ai-memory) | ❌ (`mem lesson` grava 1 nota) | GAP-7 |
| Handoff tipado, claim-once (ai-memory) | ❌ | GAP-8 |
| Perfil cross-project (ai-memory) | ❌ | GAP-8 |
| Decaimento por tier + compactação + dedup (ai-memory) | ⚠️ decaimento temporal (ADR-012) | GAP-9 |
| Zero-LLM por padrão (ai-memory) | ✅ | — |
| Markdown como fonte da verdade, índice derivado (ai-memory) | ✅ | — |
| FTS + entidade + grafo + vetor em RRF (ai-memory) | ✅ FTS+vetor+RRF; o grafo entra por `include_code` lexical | — |
| Diagnóstico com contagens (ai-memory) | ✅ `mem doctor` | — |
| Eval de qualidade de recuperação (ai-memory) | ⚠️ benchmark de throughput | GAP-10 |
| Multi-usuário + auth (ai-memory) | ❌ | fora de escopo (dono único) |
| Slots `_slots/` (ai-memory) | ❌ | GAP-9 |

**F. O que não muda**

- O `my-memory` **não** vira o `ai-memory`: os têm escopos complementares
  (`ai-memory` não faz code graph nem vetores por padrão; o `my-memory` não
  captura sessão nem tem handoff). Adota-se o **padrão de projeto**, não o
  produto.
- O `my-memory` **não** vira o graphify: ele continua sendo o grafo de código
  com vetor, multi-repo e drift — o graphify é referência de **modelo**, não
  dependência (a ADR-061 já diz isso; a auditoria de 2026-10-23 mediu 12 gaps
  favoráveis ao `my-memory`).

## Compatibility

- Nenhuma mudança de schema, tabela ou API nesta ADR.
- GAP-1 muda **o que o release entrega** (o binário passa a ter o provider) —
  é aditivo: sem toolchain C, o build com a tag falha, então a política é
  "compila com a tag quando houver toolchain; senão, sem a tag" e o binário
  avisa. Hmm: a política tem de ser explícita, senão a degradação volta a ser
  silenciosa.
- GAP-2 muda o número do Health Score (o do cofre sobe de 60 para ~90) — nenhum
  dado muda.
- GAP-3 é decisão de política do conteúdo do cofre.

## Implementation Plan

1. `fix(embedder): o binário de release carrega o provider semântico` — release
   com a tag `onnx` quando houver toolchain; job de CI com a tag; aviso
   **explícito** quando o provider não está disponível.
2. `fix(doctor): aresta de tag não é link morto` — o check de dead link passa a
   reconhecer o alvo de `tagged_as`.
3. Decisão do dono sobre os 68 wikilinks do material bruto →(spec ou ajuste do
   vault).
4. `feat(codeast): backend multi-linguagem` (GAP-4) e `feat(export): graph.json
   por repositório` (GAP-5), na ordem que o dono escolher.

### As specs escritas (2026-11-13)

Os passos 1 e 3 viraram specs fechadas, e um **terceiro** gap apareceu no
diagnóstico:

| Spec | Gap | Issue |
| :--- | :--- | :--- |
| `.specs/features/fix-release-onnx-provider/` | GAP-1 — o binário distribuído carrega o provider semântico | ISSUE-037 |
| `.specs/features/fix-doctor-unindexed-links/` | GAP-2/GAP-3 — "fora do índice por decisão" ≠ "link rompido" | ISSUE-039 |
| `.specs/features/fix-zero-vector-chunks/` | **novo** — o chunk sem provider grava vetor zero, e ninguém detecta | ISSUE-040 |

O GAP-4 (multi-linguagem) e o GAP-5 (`graph.json`) continuam sem spec: são do
lado de repositório e dependem da ordem que o dono escolher.
