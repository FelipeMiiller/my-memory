# ADR-035: Embedder Embutido com Fallback Automático (`builtin` / `all-MiniLM-L6-v2` via ONNX Runtime)

- **Date**: 2026-09-16
- **Status**: Proposed
- **Deciders**: Felipe Miiller
- **Tags**: embedding, semantic-search, fallback, cpu-only, onnx, minilm, frugal, devx

## Context and Problem Statement

Hoje, o `My-Memory` depende exclusivamente do [Ollama](https://ollama.com/) rodando em `http://localhost:11434` com o modelo `nomic-embed-text` (768 dimensões) para gerar embeddings vetoriais dos chunks indexados. O componente semântico é parte crítica do pipeline de busca híbrida (`mem search --mode hybrid`), do detector de *Surprising Connections* (`mem insights`), e do ranqueamento por PageRank semântico.

Esse acoplamento criou três problemas operacionais reais:

1. **Dependência externa obrigatória** — Sem Ollama rodando, todo o pipeline cai no fallback FTS5 léxico descrito em [`33feb1d`](../../commit/33feb1d), perdendo 100% do ranqueamento semântico e da detecção de conexões surpreendentes. Em máquinas simples (notebooks de estudo, desktops de baixo consumo, CI leve) instalar Ollama + puxar `nomic-embed-text` (~274 MB) é desproporcional ao uso real.
2. **Cold start em primeira indexação** — A primeira execução em qualquer vault novo exige que Ollama esteja previamente instalado e o modelo baixado. Se o usuário pula essa etapa, a busca semântica fica silenciosamente degradada sem aviso estruturado além da mensagem por chunk.
3. **Consumo de RAM/CPU do Ollama** — O daemon Ollama + o modelo carregado consome ~500 MB de RAM mesmo em idle, inviável em máquinas com 4–8 GB de RAM ou em cenários de uso frugal (ex: execução em containers pequenos, Chromebooks, Raspberry Pi).

Inspirado em projetos como `sentence-transformers/all-MiniLM-L6-v2` (modelo leve de 22 MB amplamente usado em produção), `fastembed` (bindings Rust para inferência CPU/GPU de modelos compactos) e a maturidade do `ONNX Runtime Go` (`github.com/yalue/onnxruntime_go`), faz-se necessário um caminho **100% in-process, CPU-friendly e frugal em recursos** que sirva como:

- (a) **Provider alternativo primário** para máquinas onde Ollama não cabe; e
- (b) **Fallback automático** quando Ollama está offline, removendo o caminho silencioso de degradação para FTS5 puro.

## Decision Drivers

- **Frugalidade computacional**: solução deve rodar em CPU (sem GPU dedicada) consumindo ≤ 100 MB de RAM adicional em idle.
- **Zero dependência externa opcional**: Ollama continua suportado; o novo provider é uma alternativa, não uma substituição obrigatória.
- **Fallback transparente**: se Ollama estiver configurado mas offline, o sistema tenta automaticamente o provider `builtin` antes de cair em FTS5 puro, maximizando a cobertura semântica.
- **Compatibilidade de schema**: o vetor gerado deve caber no schema SQLite/PostgreSQL existente (`embedding float[768]`), sem migração de banco.
- **Qualidade aceitável**: o modelo embutido não precisa igualar `nomic-embed-text`, mas deve ser claramente superior ao FTS5 puro em testes de sinônimos e paráfrases.
- **Baixo atrito operacional**: distribuição via binário único, sem etapa separada de download obrigatório (mas download lazy opcional para reduzir o tamanho do binário base).

## Considered Options

1. **Manter só Ollama + fallback FTS5 atual** (Descartado: caminho silencioso de degradação, requer instalação externa).
2. **TF-IDF + SVD (LSA) puro Go** (Descartado para esta opção principal: zero dependência mas qualidade semântica limitada — capta coocorrência mas não semântica lexical treinada).
3. **Hashing trick puro Go** (Descartado: pior que FTS5 em muitos cenários; só serviria como placeholder).
4. **`all-MiniLM-L6-v2` quantizado INT8 via ONNX Runtime Go** (*Opção Escolhida* — razão: melhor relação qualidade/esforço para o perfil frugal).
5. **`bge-small-en-v1.5` quantizado via ONNX Runtime** (Alternativa: qualidade marginalmente superior, modelo ligeiramente maior; mantida como upgrade futuro).

## Pros and Cons of the Options

### Manter só Ollama + fallback FTS5 atual

- ✅ Zero código novo; comportamento atual preservado.
- ❌ Caminho silencioso de degradação para FTS5 puro quando Ollama offline.
- ❌ Requer instalação externa (Ollama + modelo `nomic-embed-text` ~274 MB) em todas as máquinas.
- ❌ Inviável em hardware frugal (≤ 8 GB RAM): Ollama daemon consome ~500 MB em idle.

### TF-IDF + SVD (LSA) puro Go

- ✅ Zero dependência externa.
- ✅ Aprende do próprio corpus (vocabulário local), semântica caseira honesta.
- ❌ Sem semântica lexical treinada — capta coocorrência mas não paráfrase profunda nem sinônimos fora do corpus.
- ❌ Precisa de corpus mínimo (~100+ docs) para gerar embeddings úteis.
- ❌ Primeira indexação lenta (treino do SVD exige SVD completo da matriz TF-IDF).

### Hashing trick puro Go

- ✅ Trivial de implementar (~50 linhas).
- ✅ Zero dependência; vetor fixo em memória.
- ❌ Pior que FTS5 puro em vários cenários — não captura semântica alguma; sinônimos sempre distantes.
- ❌ Colisões de hash degradam qualidade progressivamente com vocabulário crescente.
- ❌ Serviria só como placeholder honesto, não como provider real.

### `all-MiniLM-L6-v2` quantizado INT8 via ONNX Runtime Go ✅ Escolhida

- ✅ Modelo leve (~22 MB não-quantizado, ~12 MB quantizado INT8).
- ✅ Inferência CPU-friendly com ONNX Runtime otimizado para SIMD.
- ✅ Qualidade semântica real — modelo treinado em pares (pergunta, resposta) multilíngue.
- ✅ Licença Apache 2.0 — sem restrições.
- ✅ Mantém schema do banco inalterado via projeção determinística 384d → 768d.
- ❌ Binário aumenta ~15-30 MB (runtime ONNX compartilhado).
- ❌ Cold start de ~1-2s na primeira inferência (warm-up na inicialização do CLI mitiga).
- ❌ Qualidade ~85-90% do `nomic-embed-text` em benchmarks MTEB.

### `bge-small-en-v1.5` quantizado via ONNX Runtime

- ✅ Qualidade marginalmente superior ao MiniLM-L6 em MTEB.
- ✅ Tamanho aceitável (~33 MB quantizado).
- ❌ Modelo maior, inferência ligeiramente mais lenta.
- ❌ Requer troca de vocab e re-tuning de thresholds.
- ❌ Mantida como upgrade futuro, não na primeira iteração.

## Decision Outcome

Adota-se a **Opção 4**: incluir `all-MiniLM-L6-v2` quantizado em INT8 como **provider `builtin`** do subsistema de embeddings, com **fallback automático** integrado.

### Topologia

```
                  ┌────────────────────────────────────────────┐
                  │       config.yaml (embedding.provider)     │
                  │                                            │
                  │   "ollama"  ──► OllamaClient (existente)   │
                  │   "builtin" ──► BuiltinClient (NOVO)      │
                  │   (vazio)    ──► auto-detect fallback      │
                  └────────────────────────┬───────────────────┘
                                           │
                              provider selection
                                           │
                  ┌────────────────────────▼───────────────────┐
                  │            EmbedderSelector                │
                  │                                            │
                  │  1. Tenta provider configurado              │
                  │  2. Se falhar, tenta próximo provider       │
                  │  3. Ordem de fallback: ollama → builtin    │
                  │     → FTS5 zero-vector (atual)              │
                  └────────────────────────────────────────────┘
```

### Componentes

#### `internal/embedder/builtin.go` (NOVO)
- Carrega `all-MiniLM-L6-v2` quantizado INT8 do caminho configurado (default: `.memory/models/minilm-l6-int8.onnx`).
- Tokenização BPE usando o vocabulário HuggingFace (shipped como `vocab.txt` no mesmo diretório, ~230 KB).
- Inferência via `github.com/yalue/onnxruntime_go` (binding para ONNX Runtime C API).
- Saída: `[]float32` de **384 dimensões** (nativas do MiniLM-L6), projetadas para 768d via matriz de projeção aleatória fixa aprendida uma vez no treino (ou padding + scaling determinístico).
- Pooling: **mean pooling** sobre tokens válidos com máscara de atenção.
- Normalização L2 no vetor final.

#### `internal/embedder/selector.go` (NOVO)
- Decide qual provider usar com base em `embedding.provider` e health-check ativo.
- Implementa fallback automático em camadas:
  1. Provider explícito (`ollama` ou `builtin`).
  2. Auto-detect: tenta Ollama; se falhar, tenta `builtin`; se ambos falharem, cai no vetor zero (FTS5 puro).
- Loga a decisão via `Aviso: <provedor> indisponível; usando <próximo>`.

#### `internal/embedder/model_downloader.go` (NOVO)
- Opcional: comando `mem embed download --provider builtin` baixa o modelo do HuggingFace Hub (`Xenova/all-MiniLM-L6-v2` ou `sentence-transformers/all-MiniLM-L6-v2`) para `.memory/models/`.
- Sem internet? O usuário pode colocar o modelo manualmente via flag `--model-path`.

#### Configuração

```yaml
# .memory/config.yaml — exemplo com provider explícito
embedding:
  provider: "builtin"        # "ollama" | "builtin" | "" (auto-detect)
  model: "minilm-l6-int8"    # alias do modelo embutido
  url: ""                    # ignorado quando provider != "ollama"
  dimension: 768             # mantido em 768 (compatibilidade de schema)
  builtin:
    model_path: ""           # default: .memory/models/minilm-l6-int8.onnx
    vocab_path: ""           # default: .memory/models/vocab.txt
    num_threads: 0           # 0 = auto-detect (nCPU / 2)
    projection_seed: 42      # semente determinística pra projeção 384→768
```

#### CLI

```bash
# Subcomando novo
mem embed download --provider builtin   # baixa o modelo + vocab
mem embed doctor                         # mostra provider ativo e health
mem embed rebuild                        # força retreino do tokenizer IDF
```

#### MCP

Novo tool opcional (proposto para ADR-036+):
- `memory_get_embedding_provider` — reporta qual provider está ativo (`ollama` / `builtin` / `fts-only`).
- `memory_set_embedding_provider` — troca o provider em runtime (com fallback automático).

### Compatibilidade

- **Schema**: nenhum impacto. `embedding float[768]` permanece. A projeção 384→768 garante que índices vetoriais existentes (`sqlite-vec`, `pgvector`) continuam funcionando sem migração.
- **Compatibilidade de similaridade**: embeddings gerados por `builtin` **NÃO** são comparáveis com embeddings gerados por `ollama` (espaços latentes diferentes). Em setups federados, é recomendado que **todos os repositórios** usem o mesmo provider. Caso misto, o `mem search` exibe aviso `[provider mismatch]` nos resultados.
- **Cadeia de fallback**: o vetor zero (FTS5 puro) continua sendo o último recurso.

## Consequences

### Positive

- **Reduz drasticamente o atrito de adoção** — Usuários em máquinas modestas (≤ 8 GB RAM) podem usar My-Memory sem instalar Ollama + modelo de 274 MB.
- **Fallback transparente** — Pipeline semântico nunca cai silenciosamente em FTS5 puro quando há provider alternativo viável.
- **Compatibilidade de schema** — Nenhuma migração de banco necessária; binários existentes continuam funcionando.
- **Operação determinística** — Sementes fixas garantem embeddings reproduzíveis para um mesmo input.
- **Independência operacional** — `mem` vira uma ferramenta self-contained; o único componente externo opcional passa a ser o storage (PostgreSQL opcional).

### Negative

- **Tamanho do binário aumenta** — `onnxruntime_go` adiciona ~15 MB ao binário compilado (lib onnxruntime compartilhada). O modelo MiniLM-L6 INT8 (~25 MB) é shippable mas opcional (download lazy).
- **Qualidade semântica inferior ao Ollama** — `all-MiniLM-L6-v2` (384d nativo, ~22 MB não-quantizado) tem qualidade menor que `nomic-embed-text` (768d, 274 MB). Em testes de benchmark padrões, MiniLM atinge ~85–90% da performance do nomic-embed-text em tarefas de similaridade semântica.
- **Cold start de inferência** — Primeira chamada a `BuiltinClient.GenerateEmbedding()` pode levar ~1-2s enquanto ONNX Runtime aloca sessão. Cache de warm-up na inicialização do CLI minimiza esse custo.
- **Licença do modelo** — `all-MiniLM-L6-v2` é Apache 2.0, compatível. Mas precisa ser baixado e sua proveniência creditada.
- **Complexidade de empacotamento** — O binário cross-platform passa a ter uma dependência nativa (lib onnxruntime). Build flags adicionais necessárias (`-tags onnx` opcional).

### Neutral

- **Schema do banco inalterado** — Migração zero.
- **CLI surface estende** — `mem embed` é novo namespace, sem impacto em comandos existentes.
- **Provider explícito opcional** — Quem não usa `builtin` ignora o overhead.

## Implementation Plan (futuro, fora do escopo deste ADR)

1. Criar `internal/embedder/builtin.go` com client ONNX + tokenizer BPE.
2. Criar `internal/embedder/selector.go` com lógica de fallback em camadas.
3. Adicionar `mem embed {download, doctor, rebuild}` em `cmd/mem/main.go`.
4. Adicionar testes: `internal/embedder/builtin_test.go` com corpus pequeno.
5. Atualizar `docs/CENTRAL_VAULT.md` (seção "Modos de Embedding") com a nova opção.
6. Atualizar `docs/CLI_GUIDE.md` com o subcomando `mem embed`.
7. Documentar em ADR-036 (futuro) os tools MCP `memory_get_embedding_provider` e `memory_set_embedding_provider`.

## Alternatives Considered

- **`bge-small-en-v1.5`** (BAAI, MIT): 33M parâmetros, ~33 MB quantizado. Qualidade ligeiramente superior ao MiniLM-L6 em benchmarks MTEB. Mantida como upgrade futuro se a qualidade MiniLM for insuficiente em produção.
- **TF-IDF + SVD puro Go**: Zero dependência, ~50 MB RAM. Capta coocorrência lexical mas não semântica profunda. Útil como fallback terciário caso ONNX não esteja disponível.
- **Embedding determinístico via SHA-256 do texto**: Trivial, mas sem noção semântica. Pior que FTS5 puro — descartado.

## References

- [`sentence-transformers/all-MiniLM-L6-v2`](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2) — modelo base, Apache 2.0
- [`yalue/onnxruntime_go`](https://github.com/yalue/onnxruntime_go) — binding Go para ONNX Runtime
- [Commit `33feb1d`](../../commit/33feb1d) — `fix(index): fallback to FTS chunk indexing when embedding generation is offline` (ancestral do conceito)
- [`docs/CENTRAL_VAULT.md` — Seção 4: Modos de Embedding](../CENTRAL_VAULT.md#4-modos-de-embedding-online-vs-fallback)
- [`docs/CLI_GUIDE.md` — Seção Federação](../../CLI_GUIDE.md) — fallback FTS documentado
- ADR-031 — Semantic Drift
- ADR-012 — Benchmarks de Performance e Conexões Inesperadas
