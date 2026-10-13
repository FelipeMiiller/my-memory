# Tasks — embedder-builtin-onnx-minilm

**Spec:** [`spec.md`](spec.md) · **ADR:** [[ADR-035]] · **Branch:** `develop`

## Execution Protocol (MANDATORY — do not skip)

1. **T1 e T2 estão fechados (veredito GO).** O gate de binding passou — ver [`validation.md`](validation.md). Fase 2+ liberada.
2. **Uma task = um commit atômico.** Marca a task como completa em `tasks.md` **antes** do commit, e inclui essa mudança no mesmo commit.
3. **O gate decide, não a autoavaliação.** Se `Tests:` falha, a task não está pronta. Nunca afrouxe, pule ou apague teste pra fazer passar.
4. **Ambiente obrigatório** (validado no T1): `CGO_ENABLED=1` e `C:\msys64\ucrt64\bin` no PATH, senão `gcc not found`.
5. Sem branch de feature. Commitar direto em `develop`.

## Test Coverage Matrix

| Layer | Test file | Tasks | Tipo |
|---|---|---|---|
| Corpus | `internal/embedder/benchmark_corpus_testdata.txt` + `corpus_test.go` | T3 | fixture + invariantes |
| Tokenizer | `internal/embedder/tokenizer_test.go` | T4 | unit |
| Inferência bruta | `internal/embedder/builtin_test.go` | T5 | integração ONNX |
| Qualidade / projeção | `internal/embedder/projection_bench_test.go` | T6, T7 | benchmark com gate de abort |
| Selector | `internal/embedder/selector_test.go` | T8 | unit (httptest) |
| Provider mismatch | `internal/store/provider_mismatch_test.go` | T9 | unit |
| Config | `internal/config/config_test.go` | T10 | unit |
| Download | `internal/embedder/model_downloader_test.go` | T11 | unit (httptest) |
| CLI | `cmd/mem/embed_test.go` | T12 | golden |
| E2E | `cmd/mem/index_embed_e2e_test.go` | T13 | integração |

**Critério de abort (T6):** se `builtin` não superar FTS5 em par sinônimo vs não-relacionado em PT-BR, o provider é **descartado** e a feature volta para revisão. Registrado como decision em `.specs/STATE.md`.

## Gate Check Commands

```bash
# estrutural (determinístico, antes de revisão humana)
python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/embedder-builtin-onnx-minilm
python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/embedder-builtin-onnx-minilm

# ambiente (obrigatório para qualquer coisa ONNX)
export CGO_ENABLED=1
export PATH="/c/msys64/ucrt64/bin:$PATH"        # ⚠️ só MSYS2 POSIX; no PowerShell:
# $env:CGO_ENABLED='1'; $env:Path="C:\msys64\ucrt64\bin;$env:Path"

# por task
go build ./...
go test -tags onnx -count=1 ./internal/embedder/...
go test -count=1 ./...

# commit
python .agents/skills/tlc-spec-driven/scripts/check_commit.py --message "<msg>"

# fim da feature
python .agents/skills/tlc-spec-driven/scripts/validate_state.py .specs/features/embedder-builtin-onnx-minilm
```

## Execution Plan

### Phase 1: Spike de binding — ✅ FECHADO (GO)

T1 e T2 concluídos. Evidência em [`validation.md`](validation.md).

### Phase 2: Embedder mínimo funcionando

Objetivo: gerar vetor **nativo 384d**, sem projeção, com tokenizer correto.
É o mínimo para poder medir qualquer coisa na fase seguinte.

### Phase 3: Decisão de projeção (com evidência)

Objetivo: fechar a ambiguidade que o ADR-035 deixou ("projeção aleatória **ou**
padding") medindo de verdade, agora que existe embedder para medir.

### Phase 4: Selector e integração

Objetivo: cadeia de fallback com log, aviso de provider mismatch, e o selector
ligado em `index`/`search`.

### Phase 5: CLI, configuração e validação real

Objetivo: `mem embed`, config, reindex do vault central e fechamento do ADR.

## Task Breakdown

### Phase 1 — ✅ concluída

#### T1: Spike de binding ONNX Runtime
**Status:** Done ✅ (2026-10-08 — verdict **GO**, `build_exit=0`, inferência real `x[2,8] → y[2,10,8]`)
**Spec:** EMBED-001..004
**Where:** módulo descartável em `%TEMP%/onnxprobe5` (não versionado)
**Depends on:** —
**Tests:** `go test -tags onnx` equivalente executado como probe; ver `validation.md`.
**Gate:** `go build -tags onnx` + `InitializeEnvironment` + `Run()` com saída real. **PASSOU.**

#### T2: Auditoria de manutenção do binding
**Status:** Done ✅ (2026-10-08 — v1.36.0, 37 versões, release 2026-09-04)
**Spec:** EMBED-002
**Where:** `validation.md`
**Depends on:** T1
**Tests:** sem teste automatizado — evidência é o registro datado com URL.
**Gate:** registro existe e o verdict é coerente. **PASSOU.**

### Phase 2

#### T3: Corpus de benchmark PT-BR
**Status:** Done ✅ (2026-10-08 — 30 sin + 30 dis = 60 pares; 82% com diacríticos; guard de vocabulário compartilhado verde)
**Spec:** EMBED-005, EMBED-007
**Where:** `internal/embedder/benchmark_corpus_testdata.txt`, `internal/embedder/corpus_test.go`
**Depends on:** —
**What:** ~30 pares sinônimo/mesma-intenção e 30 pares não-relacionados, em português com acentos e cedilha. Crítica: o corpus precisa ser de domínio técnico genérico, não do my-memory.
**Tests:** `corpus_test.go` — `TestCorpusLoads` (30+30, sem duplicatas), `TestCorpusExercitaPortugues` (≥40 pares e ≥80% com diacríticos), `TestCorpusDissimilarNaoCompartilhaVocabulario` (par `dis` não pode compartilhar palavra de conteúdo entre os lados).
**Gate:** `go test ./internal/embedder/ -run TestCorpus` verde. **PASSOU.**

> [!note]
> Material já baixado em `~/.memory/models/`:
> `paraphrase-multilingual-MiniLM-L12-v2/` (`model_int8.onnx` 112,6 MB + `unigram.json` 14 MB)
> `onnxruntime/onnxruntime.dll` (1.29.0)

#### T4: Tokenizer SentencePiece Unigram
**Status:** Done ✅ (2026-10-08 — via lib `github.com/rostamlabs/rembed/tokenizer/sentencepiece` v0.3.0)
**Spec:** EMBED-009
**Where:** `internal/embedder/tokenizer.go`, `internal/embedder/tokenizer_test.go`
**Depends on:** T3
**What:** o normalizador do modelo é um **charsmap pré-compilado** do SentencePiece (NMT-NFKC), não "lowercase + strip accents". Implementar isso à mão era o caminho para um tokenizer silenciosamente errado — sem referência para comparar, um caractere errado no charsmap degrada todo embedding sem erro. Delegado a `github.com/rostamlabs/rembed/tokenizer/sentencepiece`, que implementa exatamente a família XLM-R e tem testes pinados contra fixtures de referência do HF (`VocabSize()==250002 == len(tokenizer)` do HF). Wrapper cobre: caminho do modelo, limite 512, erros explícitos.
**Tests:** `VocabSize == 250002`; `"olá mundo" -> [▁o lá ▁mundo]`; acentos/cedilha preservados; determinismo bit-a-bit; truncamento exato em 512 (8600 runes → 512); emoji/caractere raro/zero-width não quebram; texto vazio → 0 tokens sem erro; modelo ausente → erro que nomeia o arquivo e sugere `mem embed download`. Todos verdes.
**Gate:** `go test ./internal/embedder/ -run TestTokenizer` verde. **PASSOU.**

> [!note]
> Dependência: `github.com/rostamlabs/rembed v0.3.0` (Apache-2.0, 11 stars, 0 forks, 1 autor, push em ago/2026). Baixa adoption — aceito por decisão explícita do Felipe em 2026-10-08, com o risco de normalizer errado registrado aqui.

#### T5: BuiltinClient mínimo (vetor nativo 384d, sem projeção)
**Status:** Done ✅ (2026-10-08 — 8 testes verdes, `build -tags onnx` OK, build padrão sem CGO intacto)
**Spec:** EMBED-005, EMBED-006, EMBED-008
**Where:** `internal/embedder/builtin.go`, `internal/embedder/builtin_test.go`
**Depends on:** T4
**What:** carrega o modelo, cria sessão, mean pooling sobre tokens válidos, normalização L2. **Sem projeção** — devolve os 384d nativos.
**Tests:** gera 384 floats; norma L2 = 1.0; determinismo bit-a-bit; **sem rede** (transport HTTP global trocado por um que falha); texto vazio → erro explícito, nunca vetor zero; textos sem relação → vetores diferentes e não zerados; modelo ausente → erro nomeia `mem embed download`; ORT lib ausente → erro nomeia a biblioteca. Todos verdes.
**Gate:** `go test -tags onnx ./internal/embedder/ -run TestBuiltin` verde + build padrão (CGO off) intacto. **PASSOU.**

> [!note]
> **Grounding do grafo, não assumido.** O modelo exportado pelo Xenova tem
> **três** entradas: `input_ids`, `attention_mask` e `token_type_ids`. Sem a
> terceira o runtime falha em `/embeddings/token_type_embeddings/Gather` com
> `Missing Input: token_type_ids`. Descoberto rodando um probe antes de
> escrever o código.
>
> **Sanidade semântica** (4 pares do corpus, antes da T6):
> sin 0.9868 · sin 0.9446 · dis 0.1817 · dis 0.0968.
> Separação ~0.94 vs ~0.10 confirma que a troca para modelo multilíngue
> (ADR-035 revisado) foi correta — um modelo só-ingles não separaria
> "índice" de "Império Romano" assim.
>
> Dependência: `github.com/yalue/onnxruntime_go v1.36.0`.

### Phase 3

#### T6: Benchmark projeção vs nativo vs FTS5
**Status:** Done ✅ (2026-10-08 — **provider SOBREVIVE** ao critério de abort)
**Spec:** EMBED-007, Risco 3, Risco 4
**Where:** `internal/embedder/projection.go`, `internal/embedder/projection_bench_test.go`
**Depends on:** T5
**Tests:** benchmark dos 60 pares + 7 testes de invariante. Green.

**Resultado medido (corpus PT-BR, 30 sin + 30 dis):**

| Estratégia | sin | dis | gap |
|---|---|---|---|
| nativo 384d | 0.8025 | 0.0498 | **0.7528** |
| projeção JL 384→768 | 0.7997 | 0.0524 | 0.7473 |
| padding+zero 384→768 | 0.8025 | 0.0498 | 0.7528 |
| lexical (Jaccard / teto do BM25) | 0.4556 | 0.0000 | 0.4556 |

| Critério de abort | Piso | Medido | |
|---|---|---|---|
| gap sin−dis | 0.35 | 0.7528 | ✅ |
| margem sobre o léxico | 0.15 | +0.2972 | ✅ |
| score sinônimo | 0.50 | 0.8025 | ✅ |

**Dois achados que corrigem o ADR-035:**

1. **"padding+zero degrada o vetor" era falso.** Padding é equivalente ao nativo *em cosseno* — dimensões zeradas não contribuem pro produto interno, e o benchmark devolve números idênticos. O custo do padding é de armazenamento e de risco com índices que não tratam dimensão constante, não de similaridade. Teste dedicado prova a equivalência nos 60 pares.

2. **JL preserva o gap MÉDIO, não cada par.** Pior sinônimo projetado = 0.0546 contra melhor dissímil = 0.3352. JL é projeção aleatória: preserva distância em média, não par a par. A asserção original ("pior sin > melhor dis") era propriedade que JL não garante; corrigida para as médias.

**Conclusão para o T7:** o nativo 384d ganha em todos os eixos e o padding é inútil por construção. Não há razão para projetar — o schema deveria aceitar 384d.

**Gate:** `go test -tags onnx ./internal/embedder/ -run TestProjection` verde. **PASSOU.**

#### T7: Fechar a estratégia de projeção no ADR
**Status:** Pending
**Spec:** EMBED-007, Open Question 2
**Where:** `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`
**Depends on:** T6
**What:** atualizar a seção de projeção (hoje ambígua entre "projeção aleatória **ou** padding") com a decisão e a evidência do T6.
**Tests:** o teste de T6 passa com a estratégia escolhida.
**Gate:** ADR sem indecisão na seção de projeção.

### Phase 4

#### T8: EmbedderSelector com cadeia de fallback
**Status:** Pending
**Spec:** EMBED-010..013
**Where:** `internal/embedder/selector.go`
**Depends on:** T5, T7
**What:** ordem `ollama → builtin → FTS5`; auto-detect quando provider vazio; provider explícito **não** é trocado silenciosamente; log `Aviso: <p> indisponível; usando <q>`.
**Tests:** Ollama em porta morta + builtin disponível ⇒ `builtin` + log capturado; provider vazio ⇒ tenta ollama primeiro; provider explícito inválido ⇒ registra falha sem trocar.
**Gate:** `go test ./internal/embedder/ -run TestSelector` verde.

#### T9: Aviso de provider mismatch
**Status:** Pending
**Spec:** Risco 2
**Where:** `internal/store/provider_mismatch.go`
**Depends on:** T8
**What:** gravar o provider que gerou cada vetor; no `search`, índice misto emite aviso estruturado `[provider mismatch]`.
**Tests:** índice com providers diferentes ⇒ busca avisa; índice homogêneo ⇒ não avisa.
**Gate:** `go test ./internal/store/ -run TestProviderMismatch` verde.

### Phase 5

#### T10: BuiltinConfig no EmbeddingConfig
**Status:** Pending
**Spec:** EMBED-016, EMBED-017
**Where:** `internal/config/config.go`
**Depends on:** T8
**What:** `BuiltinConfig{ModelPath, VocabPath, ORTLibPath, NumThreads, ProjectionSeed}`. **`ORTLibPath` é novo** — sem ele não há como apontar a DLL 1.29.0 (ver T1 AC-2).
**Tests:** YAML parse; defaults; `dimension` divergente do provider é reportado.
**Gate:** `go test ./internal/config/...` verde.

#### T11: mem embed download
**Status:** Pending
**Spec:** EMBED-014, EMBED-015
**Where:** `internal/embedder/model_downloader.go`
**Depends on:** T10
**What:** baixar modelo + vocab do HuggingFace, **e também a DLL do ONNX Runtime 1.29.0** (o T1 provou que é pré-requisito, não opcional). Validar tamanho, atomic rename, `--model-path`/`--ort-lib` como alternativa sem rede.
**Tests:** `httptest` — grava no path; arquivo parcial descartado; flags manuais pulam download.
**Gate:** `go test ./internal/embedder/ -run TestDownload` verde.

#### T12: mem embed doctor
**Status:** Pending
**Spec:** EMBED-016
**Where:** `cmd/mem/embed.go`
**Depends on:** T10
**What:** reporta provider ativo, saúde, dimensão real, **versão do ONNX Runtime carregada**, cadeia de fallback, e se o índice precisa de reindex.
**Tests:** golden test da saída em vault de teste.
**Gate:** `go test ./cmd/mem/ -run TestEmbedDoctor` verde.

#### T13: Integrar selector em index e search
**Status:** Pending
**Spec:** Success Criteria
**Where:** `cmd/mem/main.go`
**Depends on:** T9, T12
**What:** trocar a construção direta do `OllamaClient` pelo `EmbedderSelector` em index e busca; documentar `mem embed` em `docs/CLI_GUIDE.md`, `docs/CENTRAL_VAULT.md`, skill `my-memory` e `README.md`; reindexar o vault central sem Ollama; fechar ADR-035 como `Accepted`.
**Tests:** E2E — indexar vault **sem Ollama** e verificar chunk com embedding **não-nulo**; `search --mode vector` retorna resultado. Mais `go test -count=1 ./...` limpo.
**Gate:** E2E verde **com Ollama desligado**; Postgres do vault central com `chunks WHERE embedding IS NOT NULL > 0`; ADR-035 em `Accepted`; docs sincronizadas.

## Dependências

```
T3 ──► T4 ──► T5 ──► T6 ──► T7 ──► T8 ──► T9 ──► T13
                  │           │      └──► T10 ──► T11
                  │           │                 └──► T12 ──┘
                  └───────────┘
```

**Correção de ordem (2026-10-08):** o benchmark de projeção (antigo T4) exigia um
embedder funcionando, mas o tokenizer estava numa fase posterior — deadlock
lógico. Reordenado: tokenizer e BuiltinClient **antes** do benchmark.