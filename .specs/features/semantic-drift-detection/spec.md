# Feature: semantic-drift-detection

## Problem Statement
À medida que o código-fonte de um repositório evolui (novos commits, refatorações de código, modificações em endpoints, novas dependências), o grafo de memória de longo prazo (documentos de arquitetura, especificações técnicas e ADRs) tende a divergir silenciosamente da realidade implementada. Esse fenômeno é conhecido como **Knowledge Drift** (Desvio de Conhecimento).

Atualmente, o `my-memory` possui indexação incremental (`SHA-256`) e detecção de desatualização baseada em tempo (`mem staleness`), mas não correlaciona quais arquivos de código-fonte sofreram modificações no Git e quais notas de documentação/ADR deveriam ter sido atualizadas ou criadas para acompanhá-las.

A funcionalidade **Semantic Drift & Detecção de Desvio Código-Memória** resolve esse problema ao cruzar o histórico de commits e diffs do Git com os nós e chunks do banco de dados relacional (`memory.db`), calculando scores de desvio, alertando sobre notas defasadas e identificando código órfão sem decisões documentadas.

---

## Out of Scope
- Modificação automática do código ou escrita não-supervisionada de novos ADRs (o sistema apenas diagnostica e alerta o usuário/agente).
- Suporte a sistemas de controle de versão legados diferentes do Git (ex: SVN, Mercurial).

---

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Extração de histórico do Git | `os/exec` via CLI do Git local com interface `GitRunner` mockável | Abordagem leve, sem dependência de bindings pesados em CGO (`libgit2`), mantendo portabilidade universal. |
| Faixa de commits padrão | `HEAD~5..HEAD` (ou fallback seguro para commits disponíveis) | Foco em alterações recentes do ciclo de desenvolvimento sem sobrecarregar a análise. |
| Mapeamento Código-Nota | Correspondência por caminhos citados, menções léxicas e clusters de componentes | Captura referências diretas (ex: menção a `internal/deeplink/`) e conceituais nos chunks. |
| Detecção de Código Órfão | Arquivos de código (`*.go`, `*.py`, `*.ts`, `*.rs`, etc.) modificados sem nenhuma nota vinculada | Evita "pontos cegos" onde código novo entra no repositório sem registro arquitetural. |
| Saída para automação / CI | Suporte a flag `--strict` e saída estruturada em JSON | Permite integrar `mem drift` em pipelines de CI como gate de governança. |

Open questions: none (todas as premissas arquiteturais foram detalhadas e aprovadas no plano).

---

## User Stories

- **US1**: Como engenheiro de software ou tech lead, quero executar `mem drift` no terminal para descobrir quais notas e ADRs estão desatualizados em relação aos últimos commits de código.
- **US2**: Como agente de IA ou desenvolvedor em revisão de PR, quero saber se novas funcionalidades de código foram adicionadas sem que nenhuma nota de memória ou ADR correspondente tenha sido criada.
- **US3**: Como agente de IA conectado via MCP, quero invocar `memory_get_drift` para identificar tópicos com alto desvio e sugerir correções contextuais antes de alterar componentes críticos.

---

## Requirements (EARS Notation)

### SD-01: Extração Determinística de Histórico e Diff do Git
- **UBIQUITOUS**: The system SHALL extract commit history, modified files, additions and deletions across a specified git reference range using an abstracted `GitRunner` interface.
- **WHERE**: Where git is not installed or the directory is not a git repository, the system SHALL gracefully return a clear, non-panicking error.

### SD-02: Detecção e Cálculo de Semantic Drift
- **WHEN**: When analyzing a git range, the system SHALL correlate modified source code files with indexed memory notes via path matching and lexical chunk mentions.
- **UBIQUITOUS**: The system SHALL compute a normalized drift score between 0 and 100 for each affected note based on commit count, line changes and node PageRank.
- **WHERE**: Where the computed drift score exceeds the specified threshold, the system SHALL assign an appropriate severity classification (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`).

### SD-03: Detecção de Código Órfão de Decisões (Uncovered Code)
- **WHEN**: When source code files in the git diff have zero associated memory notes or ADRs, the system SHALL flag them as uncovered code changes.
- **UBIQUITOUS**: The system SHALL summarize uncovered code by package or file path, highlighting additions and recommended documentation actions.

### SD-04: Subcomando CLI `mem drift`
- **WHEN**: When the user executes `mem drift`, the system SHALL analyze code-memory divergence and print a tabular summary of drifted notes and uncovered code.
- **WHERE**: Where the `--json` flag is provided, the system SHALL serialize the full `DriftReport` as valid JSON to standard output.
- **WHERE**: Where the `--strict` flag is provided and any note has `CRITICAL` drift or uncovered core code, the system SHALL terminate with non-zero exit code 1.

### SD-05: Ferramenta MCP `memory_get_drift`
- **WHEN**: When an AI agent invokes `memory_get_drift`, the system SHALL return a structured Markdown report containing drift severity banners, affected notes with deep links, and recommended updates.

---

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| SD-01 | Extração Determinística de Histórico e Diff do Git via `GitRunner` | verified |
| SD-02 | Detecção e Cálculo de Semantic Drift com Scores e Severidades | verified |
| SD-03 | Detecção e Agrupamento de Código Órfão de Decisões | verified |
| SD-04 | Subcomando CLI `mem drift` com `--since`, `--threshold`, `--strict` e `--json` | verified |
| SD-05 | Ferramenta MCP `memory_get_drift` com Deep Links e Recomendações | verified |
