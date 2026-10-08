---
title: embedder-builtin-onnx-minilm — provider de embedding embutido
category: resource
summary: Implementa o ADR-035 (Status: Proposed): provider `builtin` de embeddings via all-MiniLM-L6-v2 quantizado INT8 em ONNX Runtime, com fallback automático quando o Ollama está offline. Elimina a dependência de instalar Ollama + 274 MB de modelo para ter busca semântica. O spike de binding é o gate de entrada — sem ele o plano inteiro descansa numa premissa não validada.
status: draft
tags: [embedding, onnx, minilm, cgo, fallback, adr-035, search]
---

# embedder-builtin-onnx-minilm — provider de embedding embutido

> **Tipo:** Feature. Executa o plano de implementação do [[ADR-035]] (que está com `Status: Proposed` desde 2026-09-16 e nunca foi executado).

> [!warning] O plano inteiro depende de uma premissa não validada
> O ADR-035 assume que `github.com/yalue/onnxruntime_go` compila em **Windows com CGO** e produz inferência utilizável. Isso **nunca foi testado** neste repositório: não há `onnxruntime` no `go.mod`, e nada nunca rodou uma inferência ONNX aqui.
>
> A primeira task desta spec é o spike de binding. Se o binding não buildar em Windows, tudo o resto cai e o verdict é **PIVOT** (ver [Open Question 1](#open-questions)).
>
> *Nota: o spike `.specs/features/spike-onnxruntime-binding/` existia para o mesmo binding mas no contexto ASR/Nemotron, que foi adiado em 2026-10-08 (ADR-045 → `Deferred`). A validação é feita aqui, no pacote do embedder.*

## Problem Statement

Hoje o My-Memory tem **uma única forma** de gerar embedding: `internal/embedder/ollama.go`, que faz `POST /api/embed` para um Ollama rodando em `localhost:11434`. Sem ele, o pipeline cai no FTS5 léxico — e o fazem **em silêncio**, sem aviso estruturado além de uma linha por chunk.

O estado atual da máquina do Felipe é exatamente esse pior caso:

```
Aviso: Ollama indisponível (…README.md#0); indexando em modo léxico FTS
🎉 Concluído! 16 documentos processados no PostgreSQL
```

O ADR-035 mapeia as três consequências: dependência externa obrigatória, cold start de ~1-2s, e ~500 MB de RAM do daemon Ollama em idle — inviável em máquina de 4-8 GB.

**Impacto concreto do silêncio:** o `mem search --mode hybrid` (RRF sobre BM25 + vetor + grafo) degrada para BM25+grafo sem o usuário perceber. O detector de conexões surpreendentes (`mem insights`) e o PageRank semântico ficam inertes. O componente semântico — que é a razão de existir um "cérebro" — está fora.

## Goals

- [ ] **G0 — spike de binding executado e decided** (T4 GO ou PIVOT), antes de qualquer código de produção
- [ ] `provider: builtin` gerando vetores de 768 dimensões **sem nenhum processo externo**
- [ ] Fallback automático `ollama → builtin → FTS5` com log explícito da decisão
- [ ] Schema de banco **inalterado** (nenhuma migração)
- [ ] `mem embed download` para buscar modelo + vocab, com override por flag

## Out of Scope

| Item | Reason |
|---|---|
| Trocar `nomic-embed-text` por `bge-small-en-v1.5` | ADR-035 §Alternatives: upgrade futuro, mantido como opção 5 |
| Tools MCP `memory_*_embedding_provider` | ADR-035 sugere "para ADR-036+" |
| Reindexar/migrar vetores já gravados por outro provider | Espaços latentes diferentes — ver [Risco 2](#risco-2--espaços-latentes-incompatíveis) |
| Suporte a GPU | ADR-035 exige CPU-only, ≤100 MB RAM |
| Empacotar o modelo no binário release | ADR-035: download lazy, opcional |

## Assumptions & Open Questions

| Assumption | Status | Impacto se falsa |
|---|---|---|
| `yalue/onnxruntime_go` builda em Windows com gcc/MinGW | ❌ **não validada** | Feature inteira cai → PIVOT |
| `onnxruntime_go` empacota a lib nativa por `go build` ou exige `.dll` no PATH | ❌ não validada | Precisa de passo de setup ou ship da lib |
| Projeção determinística 384d→768d preserva similaridade o suficiente | ⚠️ plausível (Johnson-Lindenstrauss), **não medida** | Precisa reavaliar o vetor ou mudar o schema |
| O modelo quantizado INT8 mantém ≥85% da qualidade do `nomic-embed-text` | ❌ não medida (afirmado pelo ADR sem benchmark) | Provedor builtin pode ser pior que FTS5 |
| HuggingFace `Xenova/all-MiniLM-L6-v2` tem ONNX INT8 disponível | ⚠️ a verificar | Trocar por `sentence-transformers` ou outro host |
| `embedding float[768]` é o schema nos dois backends | ✅ **verificado** (`internal/db/schema.go:31`, `internal/store/postgres.go:47`) | Nenhum — sem impacto |

**Open questions (bloqueiam decisions):**

1. **O binding builda?** → G0. Sem resposta, sem P2+.
2. **384→768: projeção ou mudança de schema?** O ADR sugere "projeção aleatória fixa **ou** padding+escalonamento determinístico" — são abordagens muito diferentes. Projeção preserva distâncias aproximadamente; padding+zeros **não preserva nada** e degrada o vetor. Precisa de escolha explícita e benchmark.
3. **Qual threshold de qualidade?** Como saber que `builtin` é melhor que FTS5 puro sem benchmark? Precisa de um corpus de teste com sinônimos/paráfrases em português.
4. **Qual o custo de cold start real?** O ADR estima 1-2s; isso é por processo ou amortizável?

## User Stories

### P0: Spike de binding (gate de entrada) ✅ MVP do spike

**User Story**: Como dev, quero saber se o ONNX Runtime compila e roda em Windows antes de investir no provider, para não construir 5 fases em cima de premissa não verificada.

**Acceptance Criteria:**

1. WHEN `go build` de um pacote mínimo usando `github.com/yalue/onnxruntime_go` roda em Windows amd64 THEN o sistema SHALL compilar sem erro. (event-driven)
2. WHEN a lib ONNX Runtime não está disponível THEN o sistema SHALL emitir diagnóstico nomeando o arquivo esperado e onde obtê-lo. (unwanted-behavior)
3. WHEN o modelo de teste carrega e roda uma inferência THEN o sistema SHALL retornar vetor com a dimensão declarada pelo modelo. (event-driven)
4. WHEN o spike conclui THEN o sistema SHALL produzir verdict **T4 GO** (binding viável) ou **PIVOT** (binding inviável), com o erro literal como evidência. (ubiquitous)

**Independent Test:** `internal/embedder/binding_smoke_test.go` sob build tag `onnx`, com `go test -tags onnx -v`.

> [!note]
> Reaproveitar o smoke test existente em vez de escrever um novo. O spec do spike já está revisado e corrigido.

### P1: `BuiltinClient` — inferência embutida

**User Story**: Como dev, quero gerar embedding sem Ollama rodando, para que o componente semântico funcione numa máquina sem serviço externo.

**Acceptance Criteria:**

1. WHEN `embedding.provider: builtin` está configurado THEN o sistema SHALL gerar vetores **sem abrir nenhuma conexão de rede**. (event-driven)
2. WHEN a inferência falha THEN o sistema SHALL retornar erro nomeando a causa (modelo ausente, tokenizer ausente, sessão ONNX) — nunca um vetor zero silencioso. (unwanted-behavior)
3. WHEN a dimensão nativa do modelo é 384 e o schema exige 768 THEN o sistema SHALL aplicar a projeção definida em [Open Question 2] e normalizar em L2. (event-driven)
4. WHEN o mesmo texto é embedado duas vezes THEN o sistema SHALL produzir vetores idênticos bit a bit. (ubiquitous)
5. WHEN o texto contém português com acento e cedilha THEN o sistema SHALL tokenizar corretamente (BPE com vocab do modelo). (ubiquitous)

**Independent Test:** `internal/embedder/builtin_test.go` — corpus de sentenças com sinônimos, comparando `cosine(embedding(A), embedding(B))` para par sinônimo vs par não-relacionado. **Se o par sinônimo não tiver score maior que o não-relacionado, o provider não serve.**

### P2: `EmbedderSelector` — cadeia de fallback

**User Story**: Como dev, quero que o provider caia sozinho quando o primário estiver indisponível, para que a busca semântica nunca degrade em silêncio.

**Acceptance Criteria:**

1. WHEN o provider configurado falha na inicialização THEN o sistema SHALL tentar o próximo da cadeia `ollama → builtin → FTS5`. (event-driven)
2. WHILE qualquer provider for escolhido por fallback THEN o sistema SHALL logar `Aviso: <provider> indisponível; usando <próximo>`. (unwanted-behavior)
3. WHEN `embedding.provider` está vazio THEN o sistema SHALL auto-detectar começando por `ollama`. (event-driven)
4. WHEN um provider está configurado explicitamente THEN o sistema SHALL **não** trocar por outro semwarn — respeitar a escolha explícita, mas registrar a falha. (unwanted-behavior)

**Independent Test:** `internal/embedder/selector_test.go` com servidor Ollama apontando pra porta morta + `builtin` disponível → o resultado deve ser `builtin`, com log capturado.

### P3: CLI `mem embed`

**User Story**: Como dev, quero baixar o modelo e diagnosticar o provider ativo sem editar arquivo manualmente, para colocar de pé em uma máquina nova sem passo manual.

**Acceptance Criteria:**

1. WHEN `mem embed download --provider builtin` roda com internet THEN o sistema SHALL baixar modelo + vocab para `.memory/models/` e reportar o caminho final. (event-driven)
2. IF o download falhar (sem rede / 404) THEN o sistema SHALL emitir erro com a URL tentada e aceitar `--model-path` como alternativa manual. (unwanted-behavior)
3. WHEN `mem embed doctor` roda THEN o sistema SHALL reportar provider ativo, sua saúde, a dimensão em uso e a cadeia completa de fallback. (ubiquitous)
4. IF o `embedding.dimension` do config divergir da dimensão real do provider THEN o sistema SHALL avisar a divergência no `mem embed doctor`. (unwanted-behavior)

**Independent Test:** `mem embed doctor` num vault de teste, verificando que reporta `builtin` ativo e a dimensão correta.

## Edge Cases

- **IF** `onnxruntime.dll` some do PATH depois de instalada → o provider builtin quebra em runtime; precisa de erro que diga exatamente qual arquivo.
- **IF** o usuário tem Ollama ativo E muda pra `builtin` → vetores antigos ficam obsoletos. `mem embed doctor` deve avisar que o índice precisa ser reindexado.
- **IF** vetores de `builtin` (384d projetado) e `ollama` (768d nativo) se misturam no mesmo índice → similaridade fica sem sentido. **Precisa de aviso estruturado**, não só documentação.
- **IF** o download do modelo é interrompido no meio → arquivo `.onnx` parcial. Verificar checksum/tamanho antes de usar.
- **IF** `num_threads` não é setado numCI com 2 cores → ONNX pode estourar thread. Default `nCPU/2` conforme ADR.
- **IF** o texto é muito longo (>512 tokens do MiniLM) → truncamento silencioso degrada a qualidade. Precisa de chunking compatível.

## Requirement Traceability

| Req ID | Story | AC | Status |
|---|---|---|---|
| EMBED-001 | P0 | 1 | Draft |
| EMBED-002 | P0 | 2 | Draft |
| EMBED-003 | P0 | 3 | Draft |
| EMBED-004 | P0 | 4 | Draft |
| EMBED-005 | P1 | 1 | Draft |
| EMBED-006 | P1 | 2 | Draft |
| EMBED-007 | P1 | 3 | Draft |
| EMBED-008 | P1 | 4 | Draft |
| EMBED-009 | P1 | 5 | Draft |
| EMBED-010 | P2 | 1 | Draft |
| EMBED-011 | P2 | 2 | Draft |
| EMBED-012 | P2 | 3 | Draft |
| EMBED-013 | P2 | 4 | Draft |
| EMBED-014 | P3 | 1 | Draft |
| EMBED-015 | P3 | 2 | Draft |
| EMBED-016 | P3 | 3 | Draft |
| EMBED-017 | P3 | 4 | Draft |

## Riscos

### Risco 1 — Binding não compila (blocker)
Sem CGO funcionando no Windows, a feature morre. É por isso que P0 é gate, não formality.

### Risco 2 — Espaços latentes incompatíveis
Embedding de `builtin` **não é comparável** com embedding de `ollama`. Num vault único misturado, a similaridade vira ruído. Como o vault central do Felipe é **um só** cobrindo todos os repos, esse risco é maior aqui do que num setup federado.
Mitigação: gravar o provider usado no documento/chunk e avisar no `search` quando o índice mistura providers.

### Risco 3 — Qualidade pior que FTS5
Se o benchmark (P1 AC5) mostrar que `builtin` não separa sinônimo de não-relacionado, o provider é pior que não ter nada. Precisa de **critério de abort explícito**, não "implementamos e confia".

### Risco 4 — Projeção 384→768 degrada
Se a projeção for padding+zero, o vetor final tem metade das dimensões zeradas e o cosine perde resolução. Precisa de benchmark comparando **projeção vs vetor nativo 384d** — se o schema pudesse aceitar 384d, seria melhor que projetar.

## Success Criteria

- [ ] G0 decidido: **T4 GO** ou **PIVOT**, com evidência em `.specs/features/embedder-builtin-onnx-minilm/validation.md`
- [ ] Benchmark em PT-BR decide se `builtin` supera FTS5 (critério de abort definido)
- [ ] `mem index` num vault sem Ollama produz **vetores não-nulos** nos chunks
- [ ] `mem search --mode vector` retorna resultados **sem** Ollama instalado
- [ ] Schema inalterado: nenhum `.sql` de migração, nenhum `ALTER TABLE`
- [ ] Cadeia de fallback registrada em log, sem degradação silenciosa
- [ ] `mem embed doctor` reporta provider ativo, dimensão e cadeia

## Cross-references

- [[ADR-035]] — a decisão que esta spec executa (Status: Proposed)
- [`.specs/features/spike-onnxruntime-binding/`](../spike-onnxruntime-binding/spec.md) — **adiado** (ADR-045 → `Deferred`, 2026-10-08); o gate de binding migrou para T1 desta spec
- [`internal/embedder/ollama.go`](../../../internal/embedder/ollama.go) — o provider atual, que precisa ganhar um irmão
- [`internal/config/config.go`](../../../internal/config/config.go) — `EmbeddingConfig` (hoje só conhece `ollama`)
- [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml) — job `codeast-cgo-on`, precedente de CGO no CI
- [`docs/CLI_GUIDE.md`](../../../docs/CLI_GUIDE.md) — precisa de `mem embed`

## Outputs

1. `internal/embedder/binding_smoke_test.go` + `validation.md` com verdict (P0)
2. `internal/embedder/builtin.go` + `builtin_test.go` (P1)
3. `internal/embedder/selector.go` + `selector_test.go` (P2)
4. `internal/embedder/model_downloader.go` (P3)
5. `cmd/mem/embed.go` — subcomando `mem embed {download,doctor}`
6. `embedding.builtin.*` no `EmbeddingConfig` + default em `.memory/.env.example`
7. Benchmark de qualidade PT-BR com critério de abort documentado
8. ADR-035 atualizado para `Status: Accepted` com a evidência do G0