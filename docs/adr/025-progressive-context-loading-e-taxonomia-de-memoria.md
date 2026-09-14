# ADR-025: Carregamento Progressivo de Contexto e Taxonomia de Memória

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: progressive-context, context-stuffing, l0-abstract, l1-overview, l2-details, taxonomy, openviking, codegraph, search, mcp, cli

## Context and Problem Statement

Sistemas modernos de RAG e agentes autônomos de IA sofrem frequentemente do fenômeno conhecido como **Context Window Stuffing**. Ao consultar a memória do repositório, os motores tradicionais injetam trechos extensos ou documentos inteiros diretamente no prompt do LLM. Esse comportamento acarreta sérios inconvenientes:
1. **Degradação de Raciocínio (*Lost in the Middle*)**: Grandes volumes de texto bruto diluem a atenção do modelo, dificultando a localização de informações pontuais.
2. **Desperdício de Tokens e Custo Operacional**: Em fases de triagem e descoberta inicial, o agente consome dezenas de milhares de tokens para descartar documentos irrelevantes.
3. **Falta de Estruturação Semântica**: Documentos de naturezas completamente distintas (especificações técnicas, decisões arquiteturais, scripts de tarefas) competem no mesmo espaço vetorial e textual sem distinção de propósito.

Inspirado nas abordagens de carregamento hierárquico (como o ecossistema **OpenViking**) e nas filosofias de exploração cirúrgica do **CodeGraph** (*Zero File Reads*), este ADR estabelece a estratégia de **Carregamento Progressivo de Contexto (*Progressive Context Loading*)** e a **Taxonomia Tripartida de Memória** no **My-Memory**.

## Decision Drivers

- **Eficiência de Tokens (Anti-Stuffing)**: Prover uma camada ultracompacta L0 (micro-abstract de 30 a 50 tokens) que reduz o consumo de tokens em mais de 90% na fase de descoberta.
- **Níveis Progressivos de Densidade**:
  - **L0 (Micro-Abstract)**: Resumo conciso de 1 a 2 sentenças (sem blocos de código ou tabelas), ideal para triagem de candidatos.
  - **L1 (Overview / Tríptico)**: Resumo, trecho representativo, metadados topológicos e conexões no grafo.
  - **L2 (Detalhes Completos)**: Conteúdo integral com documentação exaustiva e blocos brutos.
- **Taxonomia Tripartida de Memória**:
  - `resource`: Documentação técnica de referência, contratos, especificações de APIs e diagramas.
  - `memory`: Contexto histórico, decisões arquiteturais (ADRs), lições aprendidas e notas conceituais.
  - `skill`: Guias operacionais passo a passo, playbooks, procedimentos de depuração e automações.
- **Transparência e Ergonomia**: Suporte nativo em SQLite, PostgreSQL, CLI (`mem search`) e ferramentas MCP (`memory_search`).
- **Retrocompatibilidade Rigorosa**: Consultas legadas continuam funcionando sem alterações, mantendo o nível `l1` e busca irrestrita de categorias como padrão.

## Decision Outcome

Adotou-se o modelo de Carregamento Progressivo e Taxonomia de Memória estruturado nos seguintes componentes:

### 1. Extensão de Esquema e Migrações Idempotentes (`internal/db` e `internal/store`)
- Adicionadas as colunas `abstract TEXT` e `category TEXT` na tabela `documents` (SQLite e PostgreSQL).
- Criação de índices dedicados para aceleração de filtros por categoria:
  - SQLite: `idx_documents_category` e coluna indexada na tabela virtual `documents_fts`.
  - PostgreSQL: `idx_documents_category` e migração condicional sem bloqueio via `ADD COLUMN IF NOT EXISTS`.

### 2. Extração Determinística de L0 e Classificação no Parser (`internal/parser/markdown.go`)
- **Micro-Abstract (L0)**:
  - Prioridade 1: Metadado explícito em YAML frontmatter (`abstract`, `summary` ou `description`).
  - Prioridade 2: Primeiro parágrafo de texto após o título principal (H1), com sanitização de formatação markdown, wikilinks (`[[...]]`) e tags.
  - Truncamento seguro em UTF-8 com limite nominal de ~200 caracteres / 30-50 tokens.
- **Classificação de Categoria**:
  - Prioridade 1: Frontmatter explícito (`category`, `type` ou `taxonomia`).
  - Prioridade 2: Heurística por convenção de caminho (`resources/` $\to$ `resource`, `skills/` $\to$ `skill`, demais $\to$ `memory`).
  - Sanitização com fallback padronizado para `resource`.

### 3. Motores de Busca e Fusão RRF (`internal/db`, `internal/store`)
- Introduzida a estrutura `store.SearchOptions` contendo `Level` ("l0", "l1", "l2") e `Category` ("resource", "memory", "skill").
- Implementação das variantes `...WithOptions`:
  - `SearchFTSWithOptions`, `SearchKNNWithOptions`, `SearchTurboQuantWithOptions`, `SearchHybridRRFWithOptions`.
- Projeção sob demanda: quando `Level == "l0"`, o campo `Content` é limpo do payload e substituído pelo `Abstract`, minimizando a memória trafegada e o tempo de resposta.

### 4. Interface de Linha de Comando (`cmd/mem/main.go`)
- Adicionadas as flags `--level [l0|l1|l2]` e `--category [resource|memory|skill]` ao subcomando `mem search`.
- Função `rearrangeSearchArgs` para permitir posicionamento livre das flags antes ou após a query textual.
- Renderização visual especializada para L0: exibição em formato de cards ultracompactos no terminal (com tags `[RESOURCE]`, `[MEMORY]`, `[SKILL]`), poupando rolagem e tempo de leitura.

### 5. Protocolo MCP (`internal/mcp/`)
- Atualização do schema da ferramenta `memory_search` com as propriedades `detail_level` e `category`.
- Renderização de tabela GitHub Flavored Markdown ultracompacta para chamadas em `detail_level: "l0"`:
  ```markdown
  | # | Categoria | Documento | Score | Micro-Abstract (L0) | Conexões no Grafo |
  ```
- LLMs e agentes de orquestração podem primeiro mapear de 10 a 20 candidatos em L0 consumindo menos de 500 tokens no total, inspecionando depois cirurgicamente apenas os nós pertinentes via `memory_inspect_node` (ADR-024) ou `detail_level: "l2"`.

## Consequences

### Positive
- **Redução Massiva de Tokens**: Permite a agentes recuperar ampla variedade de candidatos para desambiguação com custo de contexto insignificante.
- **Atenção Focada do LLM**: Evita alucinações e perda de precisão causadas por ruído e blocos de código irrelevantes na fase de busca.
- **Classificação Intencional**: Permite que agentes busquem exclusivamente procedimentos operacionais (`--category skill`) ou regras arquiteturais (`--category memory`) com precisão cirúrgica.
- **Zero Quebra de Compatibilidade**: Sistemas legados mantêm comportamento idêntico.

### Negative / Trade-offs
- Documentos que não possuem frontmatter explícito dependem da qualidade do primeiro parágrafo para gerar o micro-abstract L0. Recomenda-se que notas técnicas incluam resumos de abertura diretos e informativos.
