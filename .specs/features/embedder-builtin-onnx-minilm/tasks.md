# Tasks — embedder-builtin-onnx-minilm

**Spec:** [`spec.md`](spec.md) · **ADR:** [[ADR-035]] · **Branch:** `develop`

## Execution Protocol (MANDATORY — do not skip)

1. **T1 é gate de bloqueio absoluto.** Nenhuma task de Phase 2+ começa antes de `validation.md` do spike registrar verdict **T4 GO**. Se der PIVOT, para e decide o alternativo com o usuário — não implemente provider sobre binding quebrado.
2. **Uma task = um commit atômico.** Marca a task como completa em `tasks.md` **antes** do commit, e inclui essa mudança no mesmo commit.
3. **O gate decide, não a autoavaliação.** Se `Tests:` falha, a task não está pronta. Nunca afrouxe, pule ou apague teste pra fazer passar.
4. **Sem branch de feature.** Commitar direto em `develop`.
5. Conventional Commits, validados por `check_commit.py`.

## Test Coverage Matrix

| Layer | Test file | Tasks cobertas | Tipo |
|---|---|---|---|
| Spike / binding | `internal/embedder/binding_smoke_test.go` | T1, T2 | integração (build tag `onnx`) |
| Tokenizer | `internal/embedder/tokenizer_test.go` | T6 | unit |
| Inferência | `internal/embedder/builtin_test.go` | T7 | unit + integração ONNX |
| Qualidade | `internal/embedder/builtin_bench_test.go` | T4 | benchmark com gate de abort |
| Selector | `internal/embedder/selector_test.go` | T8 | unit (httptest) |
| Provider mismatch | `internal/store/provider_mismatch_test.go` | T9 | unit |
| Config | `internal/config/config_test.go` | T10 | unit |
| Download | `internal/embedder/model_downloader_test.go` | T11 | unit (httptest) |
| CLI | `cmd/mem/embed_test.go` | T12 | golden |
| E2E | `cmd/mem/index_embed_e2e_test.go` | T13 | integração |

**Critério de abort (T4):** se `builtin` não superar FTS5 em par sinônimo vs não-relacionado em PT-BR, o provider é **descartado** e a feature volta para revisão. Registrado como decision em `.specs/STATE.md`.

## Gate Check Commands

```bash
# estrutural (determinístico, antes de revisão humana)
python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/embedder-builtin-onnx-minilm
python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/embedder-builtin-onnx-minilm

# por task
go build ./...
go test -count=1 ./internal/embedder/...
go test -tags onnx -v ./internal/embedder/...   # só T1/T2
go test -count=1 ./...

# commit
python .agents/skills/tlc-spec-driven/scripts/check_commit.py --message "<msg>"

# fim da feature
python .agents/skills/tlc-spec-driven/scripts/validate_state.py .specs/features/embedder-builtin-onnx-minilm
```

## Execution Plan

### Phase 1: Spike de binding — GATE

Objetivo: decidir GO ou PIVOT antes de qualquer código de produção.

### Phase 2: Decisão de projeção

Objetivo: fechar a ambiguidade que o ADR-035 deixou ("projeção aleatória **ou** padding") com benchmark, não com palpite.

### Phase 3: BuiltinClient

Objetivo: inferência ONNX sem rede, determinística, com tokenizer correto.

### Phase 4: Selector e integração

Objetivo: cadeia de fallback com log, aviso de provider mismatch, e o selector ligado em `index`/`search`.

### Phase 5: CLI, configuração e validação real

Objetivo: `mem embed`, config, reindex do vault central e fechamento do ADR.

## Task Breakdown

### Phase 1

#### T1: Spike de binding ONNX Runtime
**Spec:** EMBED-001..004
**Where:** `go.mod`, `internal/embedder/binding_smoke_test.go`
**Depends on:** —
**What:** adicionar `github.com/yalue/onnxruntime_go`; smoke test sob build tag `onnx` que compila, carrega o modelo e roda uma inferência real; registrar verdict T4 GO / PIVOT com erro literal em `.specs/features/embedder-builtin-onnx-minilm/validation.md`.
**Tests:** `go test -tags onnx -v ./internal/embedder/ -run TestBinding` — binding builda, sessão carrega, vetor sai com a dimensão do modelo.
**Gate:** `go build -tags onnx ./...` + teste verdes + `validation.md` escrito. **PIVOT aqui encerra a feature.**

#### T2: Auditoria de manutenção do binding
**Spec:** EMBED-002
**Where:** `.specs/features/embedder-builtin-onnx-minilm/validation.md`
**Depends on:** T1
**What:** `go list -m -versions github.com/yalue/onnxruntime_go`; verificar última release, issues abertas, atividade, licença. **Binding abandonado sem fork mantido ⇒ PIVOT por default.**
**Tests:** sem teste automatizado — a evidência é o registro datado com URL em `validation.md`.
**Gate:** registro existe e o verdict está coerente com ele.

### Phase 2

#### T3: Corpus de benchmark PT-BR
**Spec:** EMBED-005, EMBED-007
**Where:** `internal/embedder/benchmark_corpus_testdata.txt`
**Depends on:** T1
**What:** ~30 pares sinônimo/mesma-intenção e 30 pares não-relacionados, em português com acentos.
**Tests:** o arquivo carrega e tem as duas classes balanceadas.
**Gate:** `go test ./internal/embedder/ -run TestCorpusLoads` verde.

#### T4: Benchmark projeção vs nativo vs FTS
**Spec:** EMBED-007, Risco 3, Risco 4
**Where:** `internal/embedder/builtin_bench_test.go`
**Depends on:** T3
**What:** comparar coseno médio dos dois grupos para (a) nativo 384d, (b) projeção JL 384→768, (c) padding+zero. Registrar qual vence e o critério de abort.
**Tests:** asserção **score sinônimo > score não-relacionado**, com margem mínima. Padding+zero que não superar o baseline é descartado.
**Gate:** benchmark roda, resultado registrado, critério de abort explícito.

#### T5: Fechar a estratégia de projeção no ADR
**Spec:** EMBED-007, Open Question 2
**Where:** `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`
**Depends on:** T4
**What:** atualizar a seção de projeção, hoje ambígua, com a decisão e a evidência do T4.
**Tests:** o teste de T4 passa com a estratégia escolhida.
**Gate:** ADR sem "ou" indeciso na seção de projeção.

### Phase 3

#### T6: Tokenizer BPE wordpiece
**Spec:** EMBED-009
**Where:** `internal/embedder/tokenizer.go`
**Depends on:** T5
**What:** carregar `vocab.txt`, BPE wordpiece, normalização (lowercase, acentos), máscara de atenção, truncamento em 512 tokens.
**Tests:** vetores de entrada/saída do tokenizer; frase PT-BR com acento e cedilha; truncamento de texto longo não quebra nem retorna token vazio.
**Gate:** `go test ./internal/embedder/ -run TestTokenizer` verde.

#### T7: BuiltinClient com inferência ONNX
**Spec:** EMBED-005, EMBED-006, EMBED-008
**Where:** `internal/embedder/builtin.go`
**Depends on:** T6
**What:** carregar modelo do path configurado, criar sessão, mean pooling sobre tokens válidos, projeção (T5), normalização L2.
**Tests:** inferência real retorna 768 floats; **nenhuma conexão de rede** (assert com listener fechado); determinismo bit-a-bit entre duas chamadas; erro nomeado quando o modelo falta.
**Gate:** `go test ./internal/embedder/ -run TestBuiltin` verde + `go build ./...`.

### Phase 4

#### T8: EmbedderSelector com cadeia de fallback
**Spec:** EMBED-010..013
**Where:** `internal/embedder/selector.go`
**Depends on:** T7
**What:** ordem `ollama → builtin → FTS5`; auto-detect quando provider vazio; provider explícito **não** é trocado silenciosamente.
**Tests:** Ollama em porta morta + builtin disponível ⇒ resultado `builtin` com log contendo `indisponível; usando`; provider vazio ⇒ tenta ollama primeiro; provider explícito inválido ⇒ registra falha sem trocar.
**Gate:** `go test ./internal/embedder/ -run TestSelector` verde.

#### T9: Aviso de provider mismatch
**Spec:** Risco 2
**Where:** `internal/store/provider_mismatch.go`
**Depends on:** T8
**What:** gravar o provider que gerou cada vetor; no `search`, índice misto emite aviso estruturado `[provider mismatch]`.
**Tests:** índice com chunks de providers diferentes ⇒ busca avisa; índice homogêneo ⇒ não avisa.
**Gate:** `go test ./internal/store/ -run TestProviderMismatch` verde.

### Phase 5

#### T10: BuiltinConfig no EmbeddingConfig
**Spec:** EMBED-016, EMBED-017
**Where:** `internal/config/config.go`
**Depends on:** T8
**What:** `BuiltinConfig{ModelPath, VocabPath, NumThreads, ProjectionSeed}`; default `.memory/models/`; `num_threads: 0` = nCPU/2.
**Tests:** YAML parse; defaults aplicados; `dimension` divergente do provider é reportado.
**Gate:** `go test ./internal/config/...` verde.

#### T11: mem embed download
**Spec:** EMBED-014, EMBED-015
**Where:** `internal/embedder/model_downloader.go`
**Depends on:** T10
**What:** baixar modelo + vocab do HuggingFace, validar tamanho, atomic rename, `--model-path` como alternativa sem rede.
**Tests:** `httptest` servindo modelo fake — grava no path; arquivo parcial é descartado; `--model-path` pula o download.
**Gate:** `go test ./internal/embedder/ -run TestDownload` verde.

#### T12: mem embed doctor
**Spec:** EMBED-016
**Where:** `cmd/mem/embed.go`
**Depends on:** T10
**What:** subcomando que reporta provider ativo, saúde, dimensão real, cadeia de fallback e se o índice precisa de reindex.
**Tests:** golden test da saída em vault de teste.
**Gate:** `go test ./cmd/mem/ -run TestEmbedDoctor` verde.

#### T13: Integrar selector em index e search
**Spec:** Success Criteria
**Where:** `cmd/mem/main.go`
**Depends on:** T9, T12
**What:** trocar a construção direta do `OllamaClient` pelo `EmbedderSelector` nos pontos de index e busca; documentar `mem embed` em `docs/CLI_GUIDE.md`, `docs/CENTRAL_VAULT.md`, skill `my-memory` e `README.md`; reindexar o vault central sem Ollama; fechar ADR-035 como `Accepted` com a evidência.
**Tests:** E2E — indexar vault de teste **sem Ollama** e verificar chunk com embedding **não-nulo**; `search --mode vector` retorna resultado. Mais `go test -count=1 ./...` limpo.
**Gate:** E2E verde **rodando com Ollama desligado**; Postgres do vault central com `chunks WHERE embedding IS NOT NULL > 0`; ADR-035 em `Accepted`; docs sincronizadas.