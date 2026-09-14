# Feature: progressive-context-loading

## Problem Statement
No ecossistema atual de agentes autônomos de IA (Cursor, Claude Code, Antigravity, Copilot), o principal gargalo de custo, latência e degradação de raciocínio é a **poluição da janela de contexto** (*context window stuffing*).

Atualmente, tanto a busca híbrida (`mem search`) quanto a ferramenta MCP `memory_search` retornam trechos de texto brutos dos chunks indexados. Quando um agente realiza buscas amplas para planejar tarefas, centenas de linhas de arquivos inteiros são despejadas na janela de contexto antes mesmo de se saber se o documento é relevante.

Inspirado na arquitetura do **OpenViking** (Volcengine/ByteDance) e no princípio *Zero File Reads* do **CodeGraph** (Colby McHenry), o **Progressive Context Loading (Carregamento Progressivo em Camadas)** divide a recuperação de conhecimento em três níveis graduais e cirúrgicos:
1. **L0 (Micro-Abstract)**: Um resumo de 1 a 2 frases (~30-50 tokens) contendo o propósito central, tese ou resumo da nota (`abstract TEXT` na tabela `documents`). Permite que o agente descarte ou selecione nós com custo mínimo de tokens.
2. **L1 (Overview / Tríptico)**: Sumário estrutural com metadados canônicos, PageRank, comunidade LPA, conexões de entrada (*In-links*), conexões de saída (*Out-links*) e tópicos abordados, sem despejar o corpo bruto do arquivo.
3. **L2 (Full Details)**: Conteúdo integral e detalhado do documento ou seção específica, carregado cirurgicamente sob demanda apenas se L0 e L1 confirmarem a necessidade.

Além disso, introduz-se a **Taxonomia de Contexto Tripartida** (`category: resource | memory | skill` no frontmatter) para permitir buscas focadas no tipo de informação necessária.

---

## Goals
- [x] **G1**: Estender o esquema de banco de dados (`documents` em SQLite e PostgreSQL) com as colunas `abstract TEXT` (L0) e `category TEXT` (`resource`, `memory`, `skill`).
- [x] **G2**: Atualizar o parser Markdown (`internal/parser/`) para extrair automaticamente o micro-abstract do frontmatter (`summary:`, `abstract:`) ou sinteticamente do primeiro parágrafo relevante da nota.
- [ ] **G3**: Estender `db.SearchResult` e os motores de busca híbrida (`internal/db/hybrid.go`, `internal/store/rrf.go`) para suportar seleção de nível de detalhe (`l0`, `l1`, `l2`) e filtro por categoria.
- [ ] **G4**: Atualizar o subcomando de terminal `mem search` com as flags `--level [l0|l1|l2]` (padrão adaptativo ou configurável) e `--category [resource|memory|skill]`.
- [ ] **G5**: Atualizar a ferramenta MCP `memory_search` com os novos parâmetros opcionais `detail_level` ("l0" | "l1" | "l2") e `category` ("resource" | "memory" | "skill").
- [ ] **G6**: Registrar as decisões no documento de arquitetura `docs/adr/025-progressive-context-loading-e-taxonomia-de-memoria.md`.

---

## Out of Scope
- Chamadas síncronas a LLMs externos durante a indexação estática para gerar sumários caso o arquivo não possua (a geração de L0 sintético deve ser local e baseada em regras determinísticas / primeiro parágrafo para manter a indexação instantânea sem custos de API).
- Alteração no formato dos arquivos Markdown originais em disco (o L0 é uma projeção no banco de dados e no cache).

---

## Assumptions & Open Questions

| Assumption / Decision | Chosen Default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Extração padrão do L0 | Frontmatter `summary`/`abstract` ou 1º parágrafo da nota (até 160 caracteres) | Garante extração determinística de altíssima velocidade sem chamadas lentas a modelos externos | y |
| Categorias de Conhecimento | `resource` (default), `memory` (regras/hábitos), `skill` (instruções operacionais) | Alinhamento direto com o padrão de tripartição do OpenViking e Antigravity | y |
| Nível padrão no CLI | `l1` (equilíbrio entre resumo, metadados e trecho) com suporte a `--level l0` | Mantém compatibilidade visual enquanto adiciona controle de densidade | y |
| Nível padrão no MCP | `l0` ou `l1` com base no parâmetro `detail_level` | Economiza tokens críticos para agentes de IA | y |

---

## User Stories

### P1: Esquema de Banco de Dados e Extração de L0 / Category ⭐ MVP
**User Story**: Como desenvolvedor ou agente, quero que o indexador extraia o micro-abstract L0 e a categoria do documento e armazene no SQLite e PostgreSQL para permitir consultas econômicas.
**Critérios de Aceite**:
1. Migração do esquema adiciona `abstract` e `category` em `documents`.
2. Parser extrai `category` do YAML frontmatter (default: `resource`).
3. Parser extrai `summary`/`abstract` do frontmatter ou trunca o primeiro parágrafo sem formatação até 160 caracteres.

### P2: Suporte a Níveis e Filtro de Categoria no Motor de Busca
**User Story**: Como motor de busca, quero filtrar resultados por categoria e projetar apenas o nível solicitado (L0, L1 ou L2) para economizar processamento e transferência.
**Critérios de Aceite**:
1. `SearchFTS`, `SearchVector` e `SearchHybrid` aceitam parâmetro opcional `category`.
2. Estrutura de resultado inclui `Abstract`, `Category` e formatação adequada para L0 (apenas título, ID, score e micro-abstract).

### P3: Subcomando CLI mem search com Níveis e Categorias
**User Story**: Como usuário de terminal, quero executar `mem search "mcp" --level l0 --category memory` para ver uma listagem ultracompacta.
**Critérios de Aceite**:
1. Flag `--level l0|l1|l2` e `--category resource|memory|skill`.
2. Formatação visual ergonômica com badges de categoria e nível.

### P4: Ferramenta MCP memory_search com detail_level e category
**User Story**: Como Agente de IA, quero solicitar buscas com `detail_level="l0"` para fazer triagem rápida em grandes bases sem estourar minha janela de contexto.
**Critérios de Aceite**:
1. Schema JSON da tool `memory_search` atualizado com `detail_level` e `category`.
2. Handler MCP serializa Markdown proporcional ao nível solicitado.
