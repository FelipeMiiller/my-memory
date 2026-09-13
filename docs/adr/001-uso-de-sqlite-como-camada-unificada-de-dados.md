# ADR-001: Uso de SQLite como Camada Unificada de Dados

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: database, sqlite, sqlite-vec, fts5, architecture

## Context and Problem Statement

Para prover memória de longo prazo e contexto semântico/relacional a agentes de IA diretamente dentro de repositórios de código, precisamos suportar:
1. Armazenamento de metadados relacionais e chunks de documentos.
2. Busca léxica por palavras-chave exatas (código, identificadores, variáveis).
3. Busca vetorial semântica de alta velocidade (embeddings densos).
4. Grafo de conexões entre documentos, conceitos e símbolos.

Tradicionalmente, essas capacidades exigem múltiplos bancos de dados (ex: PostgreSQL + pgvector, Elasticsearch e Neo4j). Isso adiciona containers Docker, processos pesados em background e fricção inaceitável para desenvolvedores locais.

## Decision Drivers

- **Zero dependência de infraestrutura**: Deve funcionar localmente sem containers ou servidores rodando.
- **Portabilidade**: O banco deve residir em um único arquivo portátil no disco.
- **Latência mínima**: Consultas executadas em microssegundos sem overhead de rede (TCP/IP).
- **Consistência ACID**: Transações seguras com integridade relacional.

## Considered Options

- **Opção A: SQLite Unificado** (com extensões `sqlite-vec`, `FTS5` nativo e tabelas de nós/arestas).
- **Opção B: Stack Multi-Banco** (PostgreSQL + pgvector + Neo4j via Docker Compose).
- **Opção C: Bancos Vetoriais Especializados Dedicados** (ChromaDB, Qdrant ou LanceDB).

## Decision Outcome

Chosen option: **"Opção A: SQLite Unificado"**, because concentra toda a modelagem (relacional, texto completo, vetorial e grafo) em um único arquivo de banco de dados leve (`memory.db`), utilizando extensões modernas em C e recursos nativos do SQLite sem qualquer necessidade de servidores externos.

### Positive Consequences

- **Instalação Instantânea:** Nenhuma configuração de porta, usuário, senha ou Docker é exigida do usuário.
- **Modo WAL (`PRAGMA journal_mode = WAL`):** Permite leituras concorrentes ultrarrápidas enquanto a indexação grava novos chunks.
- **Integridade Relacional:** Chaves estrangeiras com `ON DELETE CASCADE` garantem que ao remover um documento, seus chunks, vetores e arestas de grafo sejam limpos automaticamente.

### Negative Consequences

- **Concorrência de Escrita:** O SQLite suporta apenas um escritor por vez, exigindo enfileiramento em processos massivos de escrita simultânea (mitigado pelo modo WAL e por se tratar de um repositório local de projeto).

## Pros and Cons of the Options

### Opção A: SQLite Unificado ✅ Chosen

- ✅ Zero dependências externas; tudo em um único arquivo `.db`.
- ✅ Busca vetorial acelerada por hardware via `sqlite-vec`.
- ✅ Busca léxica BM25 nativa via módulo `FTS5`.
- ✅ Suporte nativo a SQL recursivo para travessia de grafos.
- ❌ Requer compilação de CGO para a extensão `sqlite-vec`.

### Opção B: Stack Multi-Banco (Postgres + Neo4j)

- ✅ Ferramentas corporativas maduras para escala massiva.
- ❌ Exige Docker, portas abertas e consome gigabytes de RAM.
- ❌ Inviável para ser colocado dentro de repositórios locais de desenvolvimento.

### Opção C: Bancos Vetoriais Especializados (Chroma/Qdrant)

- ✅ Otimizados exclusivamente para busca de vetores.
- ❌ Não fornecem recursos relacionais completos nem tabelas para modelagem de grafos e FTS5 unificados.
- ❌ Adicionam dependências adicionais de runtime.

## Links

- [sqlite-vec no GitHub](https://github.com/asg017/sqlite-vec)
- [Documentação do Módulo FTS5 do SQLite](https://www.sqlite.org/fts5.html)