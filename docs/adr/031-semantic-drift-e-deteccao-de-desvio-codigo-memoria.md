# ADR-031: Semantic Drift e Detecção de Desvio Código-Memória (`mem drift` e `memory_get_drift`)

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: semantic-drift, knowledge-drift, git-diff, uncovered-code, cli, mcp, governance, ci-cd

## Context and Problem Statement

À medida que o código-fonte de um projeto evolui através de novos commits, refatorações, criação de pacotes e deleção de rotas, as notas de documentação, especificações de requisitos e Decisões de Arquitetura (ADRs) tendem a se distanciar da realidade implementada. Esse fenômeno é denominado **Knowledge Drift** (Desvio de Conhecimento).

No `my-memory`, contávamos com indexação incremental (`SHA-256`) e detecção de obsolescência temporal (`mem staleness`), mas não existia uma camada analítica capaz de correlacionar quais arquivos de código-fonte foram modificados no Git e quais notas correspondentes deixaram de ser atualizadas ou se código novo entrou no repositório sem nenhum registro arquitetural.

## Decision Drivers

- **Correlação Código-Documentação**: Identificar notas e ADRs cujos arquivos de código referenciados foram alterados no Git após a última atualização da nota.
- **Detecção de Código Órfão (*Uncovered Code*)**: Mapear arquivos de código modificados que possuem zero referências ou decisões vinculadas no grafo de memória.
- **Score Unificado de Desvio**: Métrica normalizada de 0 a 100 ponderando volume de commits, PageRank da nota e linhas alteradas no diff.
- **Integração Operacional na CLI (`mem drift`) e MCP (`memory_get_drift`)**: Disponibilizar o diagnóstico tanto para desenvolvedores no terminal quanto para agentes de IA via MCP.
- **Pronto para CI/CD (`--strict`)**: Permitir bloquear PRs e pipelines caso desvios classificados como `CRITICAL` sejam detectados.

## Decision Outcome

Implementou-se o motor de análise de drift no pacote puro `internal/drift`, integrado à CLI (`cmd/mem/drift.go`) e ao servidor MCP (`internal/mcp/drift_handlers.go`).

### 1. Motor `internal/drift`
- `GitRunner`: Abstração para extração de commits (`git log`) e diffs com adições/deleções (`git diff --numstat` e `--name-status`), com suporte a `OSGitRunner` e `MockGitRunner`.
- `AnalyzeDrift(...)`:
  - Cruza arquivos de código modificados com notas e chunks do banco de dados (SQLite e PostgreSQL).
  - Calcula o `DriftScore` por nota:
    $$\text{Score} = \min\left(100.0, \; (\Delta_{\text{commits}} \times 12) + (\text{PageRank} \times 250) + (\ln(1 + \text{LinesChanged}) \times 6)\right)$$
  - Classifica a severidade em `LOW`, `MEDIUM`, `HIGH` ou `CRITICAL` ($\ge 65.0$).
  - Agrupa arquivos de código órfão (`UncoveredCode`) com ações recomendadas baseadas no status da alteração (`A`, `M`, `D`).

### 2. Subcomando CLI `mem drift`
```bash
# Análise padrão sobre os últimos 5 commits
mem drift

# Faixa Git customizada com limite de sensibilidade e saída JSON
mem drift --since HEAD~10..HEAD --threshold 0.30 --json

# Gate de CI/CD: encerra com código 1 caso existam notas em nível CRITICAL
mem drift --strict
```

### 3. Ferramenta MCP `memory_get_drift`
Permite a agentes de IA inspecionarem o desvio código-memória sob demanda com parâmetros `since`, `threshold`, `include_uncovered` e `repository`, retornando tabelas Markdown com diagnósticos e deep links acionáveis (`obsidian://` e `vscode://`).

## Positive Consequences

- **Prevenção de Débito Documental**: Alerta imediatamente quando código crítico é alterado sem atualização correspondente nos ADRs.
- **Governança em CI/CD**: A flag `--strict` viabiliza gates determinísticos em GitHub Actions ou pré-commits.
- **Zero Fricção de Navegação**: Toda nota com desvio inclui deep links para visualização imediata nos editores suportados.

## Negative Consequences

- **Sensibilidade a Repositórios Rasos (*Shallow Clones*)**: Pipelines de CI com `git clone --depth 1` requerem `git fetch --unshallow` para avaliar faixas amplas de commits como `HEAD~5`.
