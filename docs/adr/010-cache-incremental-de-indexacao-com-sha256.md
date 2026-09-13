# ADR-010: Cache Incremental de Indexação com SHA-256

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: cache, indexing, sha256, performance, ollama, sqlite, pgvector, graphify

## Context and Problem Statement

A cada execução de mem index, todas as notas Markdown do repositório eram relidas, divididas em chunks e enviadas ao Ollama para geração de embeddings vetoriais (
omic-embed-text), mesmo quando a grande maioria dos arquivos não sofria nenhuma alteração.

Essa abordagem causava três problemas principais:
1. **Latência Desnecessária e Desperdício de Recursos:** Indexar dezenas ou centenas de notas inalteradas consumia tempo significativo e ciclos intensivos de CPU/GPU do modelo de embeddings local.
2. **Resíduos e Chunks Órfãos:** Quando uma nota era editada e seu texto diminuía ou chunks eram reorganizados, trechos anteriores permaneciam no banco relacional (chunks), na busca textual (chunks_fts) e nos índices vetoriais (chunks_vec e chunks_turboquant), poluindo os resultados de busca.
3. **Arestas Obsoletas no Grafo:** Links ([[wikilinks]]) excluídos do corpo do Markdown permaneciam como arestas ativas na tabela graph_edges.

## Decision Drivers

- **Eficiência Computacional:** Pular 100% da geração de embeddings e extração de nós para notas inalteradas (inspirado em [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify)).
- **Detecção Confiável de Mudanças:** Adotar algoritmo de hashing criptograficamente forte, imutável e determinístico.
- **Limpeza Atômica de Resíduos:** Garantir que qualquer nota modificada tenha seus dados anteriores (chunks e arestas) completamente purgados antes da nova inserção.
- **Controle Explícito do Usuário:** Permitir reconstrução completa forçada do índice sob demanda através de --force.

## Considered Options

- **Opção A: Hashing Criptográfico SHA-256 do Conteúdo** (Inspirada no Git e Graphify).
- **Opção B: Verificação por Timestamp de Modificação de Arquivo (mtime)**.
- **Opção C: File Watcher Contínuo com fsnotify/inotify**.

## Decision Outcome

Opção escolhida: **"Opção A: Hashing Criptográfico SHA-256 do Conteúdo"**, complementada pela flag --force e pelo método em cascata DeleteDocumentData:

1. **Cálculo Determinístico de SHA-256:** Função pure Go CalculateContentHash(content []byte) gera hash hexadecimal de 64 caracteres com colisão praticamente nula, imune a problemas de sincronização de relógio ou operações de checkout/clone do Git que alteram o mtime sem modificar o conteúdo real.
2. **Persistência de Hash:** Coluna content_hash TEXT adicionada à tabela documents tanto no SQLite quanto no PostgreSQL, com migrações automáticas retrocompatíveis (ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_hash TEXT).
3. **Invalidação e Limpeza em Cascata (DeleteDocumentData):** Antes de reindexar qualquer nota modificada ou quando acionado com --force, todos os chunks existentes em chunks, chunks_fts, chunks_vec, chunks_turboquant e arestas de saída em graph_edges associados ao document_id são expurgados atomicamente dentro de uma transação.
4. **Flag --force:** Sobrescreve a checagem de cache para permitir reindexação integral caso o usuário decida trocar o modelo de embeddings do Ollama ou reconstruir os índices.
5. **Sumário de Execução:** mem index contabiliza e exibe documentos novos/indexados versus documentos mantidos em cache (⏩ [cached]), oferecendo feedback claro ao usuário.

### Positive Consequences

- **Indexação Quase Instantânea:** Reexecuções de mem index em repositórios sem alterações concluem em milissegundos sem realizar nenhuma chamada ao Ollama.
- **Zero Registros Órfãos:** Remoção de texto ou wikilinks reflete-se com fidelidade imediata no banco vetorial, léxico e no grafo relacional.
- **Paridade Total:** Comportamento incremental idêntico entre o banco local SQLite e o banco PostgreSQL multi-repositório.

### Negative Consequences

- Requer leitura em disco dos bytes do arquivo Markdown para cálculo do SHA-256, overhead desprezível (poucos microssegundos por arquivo) quando comparado aos centenas de milissegundos economizados por chunk em chamadas neurais.

## Links e Referências

- [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify)
- [docs/REFERENCES.md](../REFERENCES.md)
- [docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md](001-uso-de-sqlite-como-camada-unificada-de-dados.md)
- [docs/adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md](007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)
- [docs/adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md](009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md)
