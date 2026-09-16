# Feature: context-packager-and-subgraph-bundle

## Problem Statement
Agentes autônomos de IA e desenvolvedores frequentemente precisam do contexto completo e estruturado de um módulo, subsistema ou conceito central (uma nota raiz e sua vizinhança direta/indireta de dependências e decisões).

Atualmente, para montar esse contexto, o agente necessita:
1. Disparar chamadas repetitivas de `memory_get_neighbors` ou `memory_inspect_node` para cada nó adjacente.
2. Realizar múltiplas buscas `memory_search`, obtendo trechos desconexos que perdem a topologia das relações.
3. Consumir dezenas de turnos de diálogo e arriscar estourar o limite de tokens da janela de contexto sem aviso prévio.

O **Context Packager & Subgraph Bundle** resolve esse problema ao extrair um subgrafo conectado em torno de uma nota raiz até profundidade $N$, organizando o conteúdo em um documento Markdown auto-contido com **orçamento estrito de tokens** (`max_tokens`), mapa visual topológico em Mermaid, degradação graciosa para micro-abstracts L0/L1 nas notas periféricas e rastreamento explícito de nós omitidos por corte de orçamento.

---

## Goals
- [ ] **G1**: Implementar o motor de empacotamento puro em `internal/graph/pack.go`, com estimativa determinística de tokens, BFS ponderada por PageRank e distâncias, alocação de tiers (`core`, `fringe`, `omitted`) e renderizador Markdown com bloco Mermaid.
- [ ] **G2**: Implementar `PackContext` nas camadas de persistência SQLite (`internal/db/graph.go`) e PostgreSQL (`internal/store/postgres.go`), resolvendo nós por identificador canônico.
- [ ] **G3**: Criar o subcomando CLI `mem pack <nota>` em `cmd/mem/pack.go`, com suporte a `--depth`, `--max-tokens`, `--direction`, `--out` e `--json`.
- [ ] **G4**: Implementar e expor a ferramenta MCP `memory_pack_context` em `internal/mcp/pack_handlers.go`, registrada nos transportes stdio e HTTP/SSE com esquema formal JSON Schema.
- [ ] **G5**: Registrar a decisão arquitetural em `docs/adr/029-context-packager-e-subgraph-bundle.md` (formato MADR), atualizando guias operacionais e `.specs/STATE.md`.

---

## Out of Scope
- Compressão semântica via chamadas neurais de LLM em tempo de execução (o empacotamento deve ser 100% local, instantâneo e determinístico usando os micro-abstracts L0 e sumários L1 já indexados).
- Algoritmos de corte de grafo NP-completos (usa abordagem gananciosa baseada em distância do nó raiz e PageRank).

---

## Assumptions & Decisions

| Decisão / Premissa | Valor Escolhido | Justificativa |
| :--- | :--- | :--- |
| Heurística de Tokens | $1 \text{ token} \approx 4 \text{ caracteres}$ (`len(text)/4`) | Padrão da indústria para modelos em inglês/português, rápido e sem dependência externa. |
| Profundidade padrão | `MaxDepth = 2` | Captura vizinhos imediatos (1º grau) e nós contextuais diretos (2º grau) sem explosão combinatorial. |
| Orçamento de tokens padrão | `MaxTokens = 4000` | Equivale a ~16.000 caracteres, cabendo confortavelmente em qualquer janela de contexto moderna. |
| Direção padrão | `both` | Captura tanto os documentos que a nota referencia quanto os chamadores que dependem dela. |
| Degradação de nós periféricos | L0/L1 micro-abstract | Preserva a presença e contexto conceitual da nota sem gastar o orçamento com o texto bruto completo. |

---

## Requirements (EARS Notation)

### R1: Expansão de Subgrafo Conexo
- **UBIQUITOUS**: O sistema deve iniciar a exploração a partir do nó raiz canônico e coletar todos os nós e arestas conectados até `max_depth` respeitando o filtro de `direction` (`both`, `outbound`, `inbound`).

### R2: Controle Rígido de Orçamento de Tokens
- **WHEN**: Quando o conteúdo bruto de uma nota periférica exceder o saldo restante de tokens, **THEN** o sistema deve tentar incluir seu micro-abstract L0/L1; se ainda assim não couber, o nó deve ser classificado como `omitted`.
- **UBIQUITOUS**: O total de tokens do bundle gerado nunca deve exceder o parâmetro `max_tokens` (com margem de segurança de cabeçalho).

### R3: Estrutura do Bundle Markdown
- **UBIQUITOUS**: O bundle Markdown gerado deve conter:
  1. Metadados e resumo do orçamento (tokens usados, nós principais, periféricos e omitidos).
  2. Diagrama topológico em Mermaid (`graph TD`).
  3. Seção com documentos centrais em texto integral (`## 📄 Documentos Centrais`).
  4. Seção com resumos de nós periféricos (`## 📑 Resumos Periféricos (L0/L1)`), se houver.
  5. Seção com nós omitidos por orçamento (`## ⚠️ Nós Omitidos por Orçamento`), se houver.

### R4: Interface CLI `mem pack`
- **WHEN**: Quando o usuário executar `mem pack <raiz> [--out <arquivo>]`, **THEN** o sistema deve exibir as métricas de empacotamento no terminal e salvar o bundle no arquivo indicado (ou emitir no stdout).
- **WHEN**: Quando a flag `--json` for fornecida, **THEN** o sistema deve retornar a estrutura de nós, métricas e o Markdown serializados em formato JSON válido.

### R5: Integração com Protocolo MCP
- **WHEN**: Quando um agente de IA invocar a ferramenta `memory_pack_context`, **THEN** o servidor MCP deve retornar o Markdown do bundle pronto para assimilação no prompt, acompanhado de avisos de desatualização (*staleness banner*), se aplicável.
