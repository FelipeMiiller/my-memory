# ADR-007: Suporte Opcional a PostgreSQL com pgvector e Referência Multi-Repositório

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: database, postgres, pgvector, multi-repository, architecture

## Context and Problem Statement

O projeto **My-Memory** foi originalmente projetado em torno do SQLite local unificado (ADR-001) para oferecer uma experiência de desenvolvimento local *zero-config*. No entanto, à medida que múltiplos projetos e repositórios precisam ser indexados e consultados de maneira centralizada por equipes e agentes de IA, surgem os seguintes desafios:
1. **Memória Compartilhada / Centralizada:** Agentes de IA que operam em múltiplos serviços e bibliotecas precisam correlacionar conhecimento além das fronteiras de uma única pasta local.
2. **Isolamento por Repositório:** Em um banco compartilhado, é fundamental saber a qual repositório (`repository`) cada documento, chunk de texto, vetor de embedding e aresta de grafo pertence.
3. **Escala e Concorrência:** Ambientes de ingestão contínua em CI/CD demandam múltiplos escritores simultâneos sem contenção de arquivo SQLite.

## Decision Drivers

- **Coexistência com SQLite:** O SQLite continua como padrão local leve; o PostgreSQL entra como motor alternativo para multi-repositório.
- **Isolamento de Dados:** Cada nó, documento e vetor deve carregar a referência explícita do repositório de origem (`repository`), permitindo buscas focadas ou cruzadas.
- **Rastreabilidade Automática:** O repositório deve ser identificado automaticamente via Git (`remote.origin.url`), com possibilidade de sobrescrita manual.
- **Interface Unificada:** O código de aplicação (CLI e servidor MCP) não deve se acoplar diretamente a drivers SQL específicos.

## Considered Options

- **Opção A: Abstração de Armazenamento com Suporte Híbrido (SQLite padrão + PostgreSQL com pgvector opcional e tag de repositório)**.
- **Opção B: Migração Total para PostgreSQL** (descontinuar SQLite).
- **Opção C: Bancos Vetoriais Isolados por Repositório** (criar um arquivo SQLite ou namespace separado por repo sem banco central).

## Decision Outcome

Chosen option: **"Opção A: Abstração de Armazenamento com Suporte Híbrido"**, because preserva a autonomia e simplicidade do SQLite local (sem exigir Docker para rodar um teste ou script local) enquanto viabiliza o PostgreSQL com `pgvector` para equipes e ambientes centralizados multi-repositório.

### Positive Consequences

- **Flexibilidade Total:** O usuário escolhe `--db memory.db` (local) ou `--postgres <url>` (centralizado).
- **Contexto Multi-Repo:** Um único PostgreSQL central pode abrigar dezenas de repositórios, e os agentes de IA podem consultar com filtro `repository: "org/repo"` ou buscar transversalmente.
- **Detecção Zero-Config do Repositório:** A CLI lê o remote git ativo automaticamente para rotular documentos e filtrar consultas.

### Negative Consequences

- **Manutenção de Dois Dialetos SQL:** A camada de dados precisa gerenciar DDLs e sintaxes específicas para `sqlite-vec` e `pgvector`.
- **Dependência de Driver Adicional:** Inclusão de driver PostgreSQL puro em Go (`pgx` ou `lib/pq`), devidamente gerenciado sem afetar compilações puras.

## Pros and Cons of the Options

### Opção A: Suporte Híbrido com Abstração e Tag de Repositório ✅ Chosen

- ✅ Mantém o SQLite local intacto para desenvolvedores sem infraestrutura.
- ✅ Oferece escalabilidade empresarial e memória centralizada via PostgreSQL + `pgvector`.
- ✅ Rastreabilidade precisa da proveniência de cada chunk e conexão de grafo.
- ❌ Requer manter abstração `Store` e schemas para ambos os bancos.

### Opção B: Migração Total para PostgreSQL

- ✅ Uma única implementação de banco de dados.
- ❌ Quebra a premissa de zero dependência e portabilidade do My-Memory (ADR-001).
- ❌ Torna inviável a execução rápida em máquinas sem PostgreSQL ou Docker.

### Opção C: Múltiplos Arquivos SQLite Separados

- ✅ Mantém a simplicidade do SQLite.
- ❌ Dificulta busca cruzada entre repositórios e exige orquestração manual de arquivos `.db`.
- ❌ Não resolve concorrência de escrita para CI/CD corporativo.

## Links

- [ADR-001: Uso de SQLite como Camada Unificada de Dados](001-uso-de-sqlite-como-camada-unificada-de-dados.md)
- [ADR-006: Integração com Agentes de IA via Model Context Protocol (MCP)](006-integracao-com-agentes-de-ia-via-mcp.md)
- [pgvector no GitHub](https://github.com/pgvector/pgvector)
