# Feature: incremental-indexing-cache

## Problem Statement
A cada execução do comando `mem index`, todas as notas Markdown do diretório são lidas, divididas em chunks e enviadas para geração de embeddings vetoriais via Ollama/OpenAI, mesmo quando a vasta maioria dos arquivos não sofreu qualquer alteração. Além de gerar latência desnecessária e consumo excessivo de CPU/GPU, notas que tiveram trechos removidos acumulam chunks órfãos e arestas obsoletas no banco relacional e vetorial. Um sistema de cache incremental baseado em hashing criptográfico SHA-256 do conteúdo dos arquivos (inspirado em `Graphify-Labs/graphify`) resolve este problema, pulando notas inalteradas e garantindo invalidação limpa para notas modificadas.

## Goals
- [ ] Implementar gerador determinístico de hash de conteúdo SHA-256 em pure Go.
- [ ] Adicionar coluna `content_hash` na tabela `documents` do SQLite e PostgreSQL.
- [ ] Implementar consulta de hash e método de limpeza em cascata de dados obsoletos (`DeleteDocumentData`) para SQLite e PostgreSQL.
- [ ] Integrar verificação de cache no comando CLI `mem index` pulando notas inalteradas e suportando a flag `--force`.
- [ ] Exibir sumário detalhado de indexação (quantidade de notas indexadas vs mantidas em cache).

## Out of Scope
- File watcher contínuo em background (`--watch`) com inotify/fsnotify (planejado para etapa posterior).
- Hashing de imagens ou anexos binários além do conteúdo textual das notas.

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Algoritmo de Hashing | SHA-256 | Criptograficamente seguro, padrão no Git/Graphify, com zero probabilidade de colisões práticas | y |
| Comportamento de Nota Inalterada | Pular geração de embeddings e manter chunks existentes | Poupa 100% das requisições ao Ollama para arquivos sem mudanças | y |
| Comportamento de Nota Modificada | Limpar chunks e arestas antigas antes de reindexar | Evita chunks órfãos e arestas fantasmas quando texto ou links forem excluídos | y |
| Sobrescrita Forçada | Flag --force | Permite ao usuário reconstruir o índice vetorial completo quando trocar de modelo de embeddings | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Algoritmo de Hashing e Modelo de Cache no Store ⭐ MVP

**User Story**: As a desenvolvedor ou agente, I want calcular o SHA-256 do conteúdo de arquivos e persistir esse hash na tabela de documentos so that o sistema saiba com precisão quando um arquivo foi alterado.

**Why P1**: É a base de dados necessária para qualquer decisão de cache incremental.

**Acceptance Criteria**:
1. The system SHALL compute a 64-character hexadecimal SHA-256 string for any byte slice representing document content.
2. WHEN a document is saved via InsertDocument THEN the system SHALL persist its content_hash in the documents table.
3. WHEN GetDocumentHash is called with an existing document ID THEN the system SHALL return its stored content_hash.
4. IF a document ID does not exist in the store THEN the system SHALL return an empty hash without error.

**Independent Test**: Testes unitários com slices de bytes conhecidos asserindo os hashes hexadecimais esperados e consultas de hash no store.

---

### P2: Limpeza em Cascata de Documentos Modificados

**User Story**: As a sistema de indexação, I want remover chunks e arestas antigos de uma nota modificada antes de reinseri-la so that o banco não acumule registros órfãos ou conexões inválidas.

**Why P2**: Garante a integridade referencial do grafo e a precisão das buscas semântica e léxica.

**Acceptance Criteria**:
1. WHEN DeleteDocumentData is executed for a document ID THEN the system SHALL delete all associated chunks from relational, FTS5, vector, and TurboQuant tables.
2. WHEN DeleteDocumentData is executed THEN the system SHALL delete all outgoing edges in graph_edges where source_id matches the document ID.
3. IF the document ID has no existing chunks or edges THEN the system SHALL complete DeleteDocumentData successfully without error.

**Independent Test**: Testes unitários de inserção seguidos de DeleteDocumentData verificando que a contagem de chunks e arestas associadas zera.

---

### P3: Indexação Incremental no CLI com Flag --force

**User Story**: As a usuário executando mem index, I want que o comando pule automaticamente notas inalteradas e informe quantas foram reaproveitadas do cache so that a indexação seja instantânea.

**Why P3**: É a experiência final do usuário no terminal que poupa tempo e recursos computacionais.

**Acceptance Criteria**:
1. WHEN mem index runs without --force and file content matches stored content_hash THEN the system SHALL skip chunking, graph extraction, and Ollama embedding generation for that file.
2. WHEN an unchanged file is skipped THEN the system SHALL display a cached status indicator and increment the cached counter.
3. WHERE --force flag is passed to mem index the system SHALL reindex all files regardless of their content_hash.
4. WHEN indexing finishes THEN the system SHALL display a summary containing total processed, indexed, and cached document counts.

**Independent Test**: Teste executando indexação sucessiva em diretório temporário verificando que a segunda execução pula 100% dos arquivos inalterados.

---

## Edge Cases
- IF a file is completely empty (0 bytes) THEN the system SHALL compute the standard empty SHA-256 hash and handle it gracefully.
- IF database already contains records without content_hash (legado) THEN the system SHALL treat them as modified and compute their hash na próxima indexação.
- IF file read fails due to permission error THEN the system SHALL skip that file and continue indexando os demais.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| CACHE-01 | P1: Algoritmo de Hashing e Modelo de Cache no Store | Tasks | Pending |
| CACHE-02 | P1: Algoritmo de Hashing e Modelo de Cache no Store | Tasks | Pending |
| CACHE-03 | P2: Limpeza em Cascata de Documentos Modificados | Tasks | Pending |
| CACHE-04 | P3: Indexação Incremental no CLI com Flag --force | Tasks | Pending |
| CACHE-05 | P3: Indexação Incremental no CLI com Flag --force | Tasks | Pending |

**Coverage:** 5 total, 5 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Testes unitários do cálculo de SHA-256 e operações de hash passando com 100% de sucesso.
- [ ] Coluna `content_hash` presente e funcional nos schemas SQLite e PostgreSQL.
- [ ] `mem index` pulando notas inalteradas na segunda execução com zero requisições ao Ollama.
- [ ] Flag `--force` reindexando todos os arquivos sob demanda.
