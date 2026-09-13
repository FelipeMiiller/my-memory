# ADR-009: Busca Híbrida com Reciprocal Rank Fusion (RRF)

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: rrf, hybrid-search, fts5, pgvector, sqlite-vec, turboquant, graph, mcp

## Context and Problem Statement

No projeto **My-Memory**, a recuperação de notas era realizada exclusivamente por distância vetorial densa (k-NN via `sqlite-vec`, `TurboQuant` ou `pgvector`), complementada pela expansão de nós vizinhos imediatos no grafo (ADR-005 e ADR-007).

Entretanto, conforme documentado em referências como [`akitaonrails/ai-memory`](https://github.com/akitaonrails/ai-memory) e [`docs/REFERENCES.md`](../REFERENCES.md), a busca puramente vetorial apresenta deficiências estruturais bem conhecidas na literatura de Information Retrieval (IR):
1. **Baixa precisão em termos léxicos exatos:** Nomes específicos de classes, funções (`InitDB`), identificadores de código, siglas técnicas e termos raros sofrem dispersão semântica no espaço de embeddings do Ollama (`nomic-embed-text`).
2. **Dependência e fragilidade de pesos manuais (Linear Fusion):** Métodos de fusão linear que combinam score de cosseno e score BM25 (ex: $\alpha \cdot \text{BM25} + (1-\alpha) \cdot \text{Cosine}$) exigem calibração empírica minuciosa de pesos $\alpha$, os quais variam drasticamente conforme o tamanho dos textos e domínios de conhecimento.
3. **Subutilização da topologia relacional do grafo:** As conexões explícitas escritas pelo usuário (`[[wikilinks]]`) eram apenas exibidas como metadados passivos no resultado, em vez de reforçar a pontuação dos trechos diretamente adjacentes no grafo de conhecimento.

## Decision Drivers

- **Fusão Agnóstica e Robusta:** Utilizar um algoritmo matemático comprovado na literatura de recuperação da informação que não dependa de pontuações absolutas e elimine hiperparâmetros frágeis de ponderação.
- **Integração das Três Dimensões de Memória:** Unificar busca léxica exata (BM25 via SQLite FTS5 e PostgreSQL tsvector), busca vetorial densa (k-NN float32 e TurboQuant 4-bit) e conexões adjacentes do grafo (CTEs recursivos).
- **Paridade entre Backends:** Manter consistência estrita entre o armazenamento local padrão (SQLite) e o corporativo multi-repositório (PostgreSQL com `pgvector`).
- **Resiliência e Fallback:** Operar com elegância mesmo quando o servidor de modelos locais (Ollama) estiver temporariamente desligado ou indisponível.

## Considered Options

- **Opção A: Reciprocal Rank Fusion (RRF) com $k = 60$** (Inspirada no `akitaonrails/ai-memory` e Elasticsearch).
- **Opção B: Combinação Linear com Pesos Fixos** ($\alpha \cdot \text{vetor} + \beta \cdot \text{FTS}$).
- **Opção C: Re-ranking Neural via Cross-Encoder** (ex: modelo `ms-marco-MiniLM-L-6-v2`).

## Decision Outcome

Opção escolhida: **"Opção A: Reciprocal Rank Fusion (RRF) com $k = 60$"**, pelos seguintes fundamentos:

1. **Fórmula Matemática do RRF:**
   $$\text{RRF Score}(d) = \sum_{m \in M} \frac{1}{k + \text{rank}_m(d)}$$
   onde $M = \{\text{FTS5 (BM25)}, \text{Vetorial (k-NN / TurboQuant)}, \text{Grafo (CTE Neighbors)}\}$ e $k = 60$ é a constante de suavização padrão da literatura de Cormack et al. (2009).
2. **Independência de Escala:** RRF avalia apenas as posições relativas dos candidatos em cada modalidade (rank ordinal: 1, 2, 3...), tornando irrelevantes as disparidades de amplitude entre valores de distância cosseno (0 a 2) e BM25 (-∞ a +∞).
3. **Reforço de Nós Vizinhos do Grafo:** Documentos e trechos pertencentes aos nós vizinhos das principais sementes recuperadas recebem ranking estrutural no grafo, promovendo notas conceitualmente conectadas ao topo da resposta.
4. **Fallback Gracioso:** Se o serviço de embeddings estiver inacessível, o sistema automaticamente executa a busca textual FTS5/BM25 sem interromper o fluxo do agente.

### Positive Consequences

- **Melhoria Expressiva na Precisão:** Encontra termos literais exatos ao mesmo tempo em que preserva a analogia semântica dos vetores e o contexto relacional do grafo.
- **Padrão Transparente no MCP e CLI:** `mem search` e a ferramenta MCP `memory_search` executam o modo `hybrid` por padrão, exibindo `Score RRF` e detalhamento das fontes (`[fts #1, vector #2, graph #1]`).
- **Suporte a Modos Opcionais:** Usuários e agentes podem forçar `--mode vector` ou `--mode fts` quando desejarem consulta restrita a uma única modalidade.

### Negative Consequences

- Requer execução de consultas paralelas ou consecutivas (texto + vetor + expansão CTE) antes da ordenação final, com aumento marginal (menos de 5ms) na latência de busca local.

## Links e Referências

- [`akitaonrails/ai-memory`](https://github.com/akitaonrails/ai-memory)
- Cormack, G. V., Clarke, C. L., & Buettcher, S. (2009). *Reciprocal rank fusion outperforms condorcet and individual rank learning methods*. SIGIR '09.
- [Elasticsearch Hybrid Search and RRF](https://www.elastic.co/guide/en/elasticsearch/reference/current/rrf.html)
- [`docs/REFERENCES.md`](../REFERENCES.md)
- [`docs/adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md`](007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)
