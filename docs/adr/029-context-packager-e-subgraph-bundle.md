# ADR-029: Context Packager e Subgraph Bundle com Orçamento de Tokens

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, context-packager, subgraph, token-budget, mermaid, progressive-context, mcp, cli

## Context and Problem Statement

Agentes autônomos de IA (Antigravity, Claude Code, Cursor, Windsurf) atuando em bases de código e vaults densos precisam de contextos completos e coerentes para planejar features, desenhar arquiteturas ou refatorar subsistemas complexos.

Embora o **My-Memory** disponibilizasse recuperação progressiva granular (ADR-025) e inspeção de nó individual (ADR-024), existia um abismo operacional quando o agente precisava:
1. Obter a visão consolidada de um módulo raiz junto a todos os seus documentos relacionados diretos e indiretos (1º e 2º graus de conexão).
2. Respeitar com rigor um orçamento predefinido de tokens para não estourar a janela de contexto nem incorrer em custos desnecessários.
3. Compreender a topologia relacional do subgrafo extraído sem precisar reconstituí-la através de dezenas de tool calls consecutivas.

Sem um empacotador de contexto nativo, o agente recorria a múltiplas chamadas consecutivas de `memory_get_neighbors` e `memory_inspect_node`, consumindo turnos excessivos de conversação e montando bundles manuais sem garantia de limite de tamanho.

## Decision Drivers

- **Auto-Contenção e Pronto para Injeção**: O resultado deve ser um documento Markdown único, estruturado e legível, contendo metadados, diagrama de topologia e notas agregadas.
- **Controle Rígido de Orçamento de Tokens (`max_tokens`)**: O empacotador deve garantir matematicamente que o bundle final não exceda o limite estipulado pelo chamador.
- **Degradação Graciosa com Carregamento Progressivo (L0/L1/L2)**: Nós que não couberem em texto integral (TierCore) devem ser incluídos como micro-abstracts L0/L1 (TierFringe), e nós totalmente descartados devem ser listados explicitamente (TierOmitted).
- **Priorização Ponderada Estrutural**: A seleção gananciosa de conteúdo deve priorizar distância do nó raiz, autoridade estrutural (PageRank ponderado) e pesos epistêmicos das arestas.
- **Visualização Topológica Mermaid**: Inclusão de bloco `mermaid` nativo detalhando os nós centrais, periféricos e arestas que compõem o subgrafo.
- **Disponibilidade Multimodal**: Exposição via subcomando CLI `mem pack` e ferramenta MCP `memory_pack_context`.

## Decision Outcome

Implementou-se o motor de **Context Packager & Subgraph Bundle** em `internal/graph/pack.go`, integrado às camadas de persistência SQLite e PostgreSQL, CLI e servidor MCP.

### 1. Modelo de Alocação de Orçamento em 3 Tiers
- **TierCore (L2)**: O nó recebe inclusão de seu texto integral.
- **TierFringe (L0/L1)**: Se o texto integral ultrapassar o orçamento restante, mas o micro-abstract/resumo couber, o nó é preservado com nível de detalhe compacto.
- **TierOmitted**: Se nem o micro-abstract couber, o nó é apenas listado como omitido na seção de controle do bundle, mantendo a consciência relacional do modelo sem desperdício de tokens.

### 2. Heurística Determinística de Estimativa de Tokens
$$\text{Tokens}(s) = \left\lceil \frac{\text{len}(\text{runes}(s)) + 3}{4} \right\rceil$$
Garante previsibilidade sem necessidade de bibliotecas neurais pesadas de tokenização em Go.

### 3. Subcomando CLI `mem pack` (`cmd/mem/pack.go`)
```bash
# Empacotar subgrafo até 2 saltos com orçamento de 4000 tokens
mem pack docs/auth.md --depth 2 --max-tokens 4000

# Salvar o bundle Markdown diretamente em arquivo
mem pack "ARQUITETURA" --out bundle.md

# Saída em formato JSON estruturado
mem pack "ARQUITETURA" --json
```

### 4. Ferramenta MCP `memory_pack_context` (`internal/mcp/pack_handlers.go`)
Permite aos agentes solicitar um subgrafo em uma única chamada:
```json
{
  "name": "memory_pack_context",
  "arguments": {
    "root_node": "docs/auth.md",
    "max_depth": 2,
    "max_tokens": 3500,
    "direction": "both"
  }
}
```

## Consequences

### Positive
- Redução de até 80% nos turnos de diálogo entre agentes e o servidor MCP para assimilação de contexto temático.
- Garantia de que a janela de contexto de prompts não sofrerá overflow não planejado.
- Facilidade de consumo humano e de LLM com diagramas Mermaid renderizáveis nativamente no Obsidian e no GitHub.

### Negative
- A heurística $1 \text{ token} \approx 4 \text{ caracteres}$ é uma aproximação e pode variar sutilmente dependendo do tokenizador exato da LLM (compensado com margem de segurança de overhead).

## Links
- [ADR-024: Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)](024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md)
- [ADR-025: Carregamento Progressivo de Contexto e Taxonomia de Memória](025-progressive-context-loading-e-taxonomia-de-memoria.md)
- [ADR-027: Descoberta de Rotas e Caminho Mínimo no Grafo de Conhecimento](027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md)
