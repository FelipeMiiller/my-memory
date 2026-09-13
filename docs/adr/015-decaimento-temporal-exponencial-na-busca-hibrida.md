# ADR-015: Decaimento Temporal Exponencial na Busca Híbrida

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: search, rrf, time-decay, recency, sqlite, postgres, mcp, cli

## Context and Problem Statement

No My-Memory, a busca híbrida com Reciprocal Rank Fusion (ADR-009) consolidou com sucesso as modalidades léxica (FTS5/tsvector), semântica (vetores k-NN / TurboQuant) e estrutural (vizinhos no grafo de conhecimento). No entanto, o ranking resultante tratava todas as notas de forma atemporal, considerando apenas a relevância do conteúdo em relação aos termos da consulta.

Em bases de notas dinâmicas e grafos de conhecimento de longo prazo, essa abordagem gera um problema crônico de **obsolescência**:
1. **Sufocamento de Decisões Recentes:** Documentos antigos e extensos, contendo alta frequência de termos ou múltiplos chunks similares, tendem a pontuar sistematicamente mais alto que notas recentes concisas contendo a arquitetura atualizada.
2. **Falta de Sensibilidade Temporal:** O agente de IA ou desenvolvedor não tinha como priorizar aprendizados recentes sem filtrar manualmente por datas ou reescrever as consultas.

Havia a necessidade de um modelo de decaimento temporal suave que favorecesse notas recentes, sem penalizar destrutivamente conhecimentos perenes ou notas fundamentais sem atualizações recentes.

## Decision Drivers

- **Piso Assintótico Garantido:** Notas antigas altamente relevantes nunca devem ter seu score zerado; deve existir um piso configurável ($1 - w$).
- **Curva Intuitiva de Meia-Vida:** A perda de relevância deve ser parametrizada por meia-vida física em dias ($T_{half}$), facilitando o ajuste empírico pelo usuário.
- **Isolamento e Retrocompatibilidade:** A busca padrão sem decaimento (`decay = false`) deve manter exatamente os mesmos scores e comportamentos definidos no ADR-009.
- **Unificação Multi-Engine:** O decaimento temporal deve funcionar de maneira transparente tanto no armazenamento local SQLite quanto no cluster PostgreSQL com pgvector.
- **Exposição Uniforme:** Disponibilização tanto no servidor MCP (`memory_search`) para agentes autônomos quanto na linha de comando (`mem search`).

## Decision Outcome

Adotou-se a formulação de **Decaimento Temporal Exponencial com Piso Assintótico Ponderado** integrado à fusão RRF:

1. **Formulação Matemática (`internal/store/rrf.go`):**
   - Idade do documento em segundos: $\Delta t = \max(0, t_{ref} - t_{doc})$.
   - Meia-vida em segundos: $T_{half} = \text{halfLifeDays} \times 86400$.
   - Fator de decaimento puro: $\text{Decay}(\Delta t) = 2^{-\Delta t / T_{half}}$.
   - Multiplicador com piso: $\text{Multiplier} = (1.0 - w) + (w \cdot \text{Decay}(\Delta t))$, onde $w \in [0.0, 1.0]$.
   - Score final pós-fusão: $\text{Score}_{final} = \text{Score}_{RRF} \times \text{Multiplier}$.
   - Se $w = 0.3$ e $T_{half} = 30$ dias (padrão): notas de hoje têm multiplicador $1.0$, notas de 30 dias têm $0.85$, e notas infinitamente antigas convergem suavemente para o piso de $0.70$.

2. **Enriquecimento Estrutural e Queries de Armazenamento:**
   - Adicionado o campo `UpdatedAt int64` na struct `SearchResult` dos pacotes `internal/store`, `internal/db` e `internal/mcp`.
   - Atualizadas as queries SQL em `internal/db/hybrid.go` e `internal/db/graph.go` (SQLite) e `internal/store/postgres.go` (PostgreSQL), realizando `LEFT JOIN documents` para capturar `COALESCE(d.updated_at, 0)`.
   - Adicionado `SearchHybridRRFWithDecay` no SQLite e na interface `Store`.

3. **Protocolo MCP (`internal/mcp`):**
   - Schema de `ToolMemorySearch` estendido com as propriedades opcionais: `decay (boolean)`, `half_life (number)` e `decay_weight (number)`.
   - `NewMemorySearchHandler` repassa `DecayOptions` para o executor de busca.
   - Respostas de busca formatam automaticamente o carimbo legível `Atualizado em: YYYY-MM-DD HH:MM:SS` quando disponível.

4. **Interface de Linha de Comando (`cmd/mem/main.go`):**
   - Flags `--decay`, `--half-life` e `--decay-weight` integradas ao subcomando `mem search`.
   - Feedback visual informativo no cabeçalho quando o decaimento está habilitado.

### Positive Consequences

- **Melhor Relevância Temporal:** Decisões recentes e notas emergentes sobressaem sem perder a capacidade de resgatar notas clássicas.
- **Configurabilidade Fina:** O usuário ou agente pode controlar a agressividade do decaimento ajustando peso e meia-vida conforme a dinâmica do repositório.
- **Robustez Matemática:** A garantia do piso $(1 - w)$ impede que notas fundamentais sejam expurgadas do ranking de relevância.

### Negative Consequences / Trade-offs

- **Custo Adicional de JOIN:** As queries de leitura realizam um `LEFT JOIN documents` adicional sobre a chave primária `document_id`, com impacto insignificante em bancos indexados (< 0.2 ms).
