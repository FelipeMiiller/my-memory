# ADR-024: Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, triptych, inspector, surgical-context, zero-file-reads, codegraph, pagerank, blast-radius, cli, mcp, html-svg

## Context and Problem Statement

Ao navegar em grafos de conhecimento, bases de decisão arquitetural (ADRs) ou repositórios de software, agentes autônomos de IA e desenvolvedores humanos frequentemente incorrem em **exploração cega (*blind exploration*)**. Para compreender uma nota central, o agente precisa ler repetidamente arquivos adjacentes para responder a três perguntas elementares:
1. *Quem referencia ou depende diretamente deste nó? (Inbound / Callers)*
2. *Quais são os metadados canônicos, métricas topológicas, risco de quebra e essência do conteúdo deste nó? (Core Node & Risk)*
3. *Para quais notas, contratos ou conceitos este nó aponta? E esses destinos existem ou são links quebrados? (Outbound / Callees & Dead Links)*

A ausência de uma visão unificada força o agente a gastar múltiplos turnos de conversa, centenas de milhares de tokens e chamadas repetitivas de leitura de arquivos em disco (*file reads*).

Inspirado na arquitetura ergonômica do **CodeGraph** (Colby McHenry), este documento formaliza a decisão de implementar a **Visualização Cirúrgica em 3 Colunas (*Triptych Node Inspector*)** no ecossistema do **My-Memory**.

## Decision Drivers

- **Contexto Cirúrgico (*Zero File Reads*)**: Permitir que um agente de IA ou desenvolvedor obtenha a totalidade da vizinhança topológica e a síntese de conteúdo em uma única chamada.
- **Estrutura Espacial em Três Colunas**:
  - Coluna 1 (Esquerda): Chamadores e dependentes de entrada (*Inbound*), classificados por severidade de impacto (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`) e autoridade PageRank.
  - Coluna 2 (Centro): Metadados canônicos, ID, tipo, PageRank, comunidade temática LPA (ADR-022), score de risco de blast radius (ADR-023) e preview seguro de conteúdo com truncamento inteligente.
  - Coluna 3 (Direita): Referências de saída (*Outbound*), com verificação ativa de existência para detectar *dead links* e nós órfãos.
- **Resolução Canônica e Tolerância a Slashes**: Suporte a caminhos no estilo Windows (`docs\adr\...`) e Unix (`docs/adr/...`), títulos parciais, slugs e links no formato `[[...]]`.
- **Acesso Universal**:
  - Linha de comando via `mem inspect <node_id> [--json] [--full] [--max-len N]`.
  - Protocolo MCP via ferramenta `memory_inspect_node`.
  - Visualizador Web interativo standalone (`internal/graphview`) via modal e painel retrátil de 3 colunas.

## Decision Outcome

Adotou-se o modelo de **Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)** implementado nos seguintes módulos:

### 1. Modelo de Domínio e Ordenação Determinística (`internal/graph/inspector.go`)

- **Estruturas de Dados**:
  - `InboundLink`: Origem, título, tipo de nó, relação, severidade (`DetermineSeverity`), peso, PageRank e cluster LPA.
  - `OutboundLink`: Destino, título, tipo de nó, relação, peso, PageRank, status `Exists` (booleano) e cluster.
  - `NodeSummary`: ID canônico, título, caminho do arquivo, tipo, tags, PageRank, comunidade, score e nível de risco, preview de conteúdo e timestamp.
  - `TriptychView`: Agregação estruturada das três colunas, totalizadores e contagem de dependentes críticos.
- **Critérios de Ordenação**:
  - Inbound: Severidade decrescente (`CRITICAL` $\to$ `HIGH` $\to$ `MEDIUM` $\to$ `LOW`), desempate por PageRank decrescente.
  - Outbound: Destinos existentes primeiro (`Exists = true`), ordenados por tipo de relação e ID.
- **Truncamento UTF-8 Seguro**: Função `TruncateContent` com contagem em *runes* para prevenir cortes em bytes inválidos de caracteres acentuados.

### 2. Camada de Persistência Unificada (`internal/db/graph.go` & `internal/store/postgres.go`)

- Implementação do método `InspectNode(ctx, db, query, maxLen)` no SQLite e `(s *PostgresStore) InspectNode(ctx, repo, query, maxLen)` no PostgreSQL.
- Integração da resolução canônica normalizando barras (`/` e `\`) para interoperabilidade transparente entre Windows e Linux.
- Extração de chunks de texto para composição do preview cirúrgico sem necessidade de I/O em arquivos brutos no disco.
- Enriquecimento em tempo de query com PageRank (ADR-014), detecção de comunidades LPA (ADR-022) e cálculo de raio de destruição (ADR-023).

### 3. Interface de Linha de Comando (`cmd/mem/inspect.go`)

- Subcomando:
  ```bash
  mem inspect <node_id> [--json] [--full] [--max-len 500] [--db <arq>] [--postgres <url>] [--repo <slug>]
  ```
- **Reorganização de Argumentos**: Função `rearrangeInspectArgs` permitindo posicionamento livre de flags antes ou depois do identificador do nó (ex: `mem inspect 023.md --json`).
- Renderização visual com blocos formatados para terminais e suporte à serialização JSON padronizada.

### 4. Servidor MCP e Protocolo Inteligente (`internal/mcp/`)

- Registro da ferramenta `memory_inspect_node` nos catálogos stdio e HTTP/SSE (`port 38400`).
- Geração de resposta em GitHub Flavored Markdown estruturada com seções claras, badges e tabelas para fácil digestão por LLMs.

### 5. Visualizador Interativo HTML/SVG (`internal/graphview/template.go`)

- Integração de um modal responsivo em CSS Grid de 3 colunas (`.triptych-modal`):
  - Chamadores à esquerda com badges coloridos de severidade (`CRÍTICO`, `ALTO`, `MÉDIO`, `BAIXO`).
  - Painel central com título, métricas topológicas, risco e atalho para `obsidian://open?file=...`.
  - Referências de saída à direita com validação visual de integridade.
- Suporte a navegação rápida clicando em nós da lista com atualização dinâmica in-place e fechamento por tecla `Escape`.

## Consequences

### Positive
- **Redução Drástica de Tokens**: Assistentes de IA avaliam dependências e contexto do nó em uma única chamada de ferramenta MCP.
- **Segurança de Refatoração**: Desenvolvedores e agentes visualizam instantaneamente o score de risco (Blast Radius) e os dependentes críticos antes de aplicar edições.
- **Detecção Imediata de Dead Links**: Identificação em tempo real de referências a notas inexistentes no painel de outbound.
- **Consistência Multi-Transporte**: Suporte idêntico em terminal CLI, servidor MCP e navegador HTML standalone.

### Negative / Trade-offs
- A montagem completa do tríptico executa computação de PageRank e comunidades sobre o grafo no momento da inspeção (em bases massivas, poderá demandar cache de centralidade pré-calculada).
