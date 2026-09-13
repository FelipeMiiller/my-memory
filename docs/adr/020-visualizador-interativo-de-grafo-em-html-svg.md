# ADR-020: Visualizador Interativo de Grafo em HTML/SVG Standalone

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graphview, interactive-graph, force-directed, svg, zero-cdn, standalone-html, pagerank, cli, mcp, obsidian

## Context and Problem Statement

Desde a ADR-008, o **My-Memory** suporta a exportação de subgrafos locais para a especificação aberta JSON Canvas 1.0 (`.canvas`). Embora o formato seja interoperável com o Obsidian, essa abordagem apresenta limitações operacionais expressivas:
1. **Dependência Exclusiva de Software Desktop:** Usuários e equipes sem o Obsidian instalado ou em ambientes de terminal remoto e servidores não conseguiam inspecionar graficamente o conhecimento do repositório.
2. **Ausência de Visão Global da Topologia:** O `.canvas` exigia a seleção prévia de uma nota central, sem oferecer uma perspectiva holística de todos os nós, comunidades e ilhas do vault.
3. **Falta de Recursos Interativos Dinâmicos:** A exploração não contava com busca em tempo real com atenuação visual de nós não correlacionados, filtragem dinâmica por tipo semântico (`concept`, `decision`, etc.) e visualização proporcional da autoridade estrutural calculada via PageRank.
4. **Necessidade de Autonomia Offline (Zero-CDN):** Muitas organizações e desenvolvedores trabalham em ambientes isolados (*air-gapped*) ou sem conexão contínua com a internet. Depender de CDNs de terceiros (como D3 via unpkg/cdnjs) violaria princípios de soberania e segurança.

## Decision Drivers

- **Zero Dependências Externas (Zero-CDN)**: Todo o CSS e o motor de física gravitacional (Force-Directed Graph) em JavaScript devem estar embutidos em um único arquivo `.html` autocontido.
- **Topologia Holística & Subgrafos Contextuais**: Suporte tanto à renderização do grafo global completo quanto ao isolamento de subgrafos radiais a partir de uma nota raiz com limitação de profundidade (`--root` e `--depth`).
- **Métricas Visuais Orientadas a PageRank**: O raio de cada nó deve refletir sua relevância na rede de conhecimento calculada pelo algoritmo PageRank ponderado (`internal/graph`), destacando visualmente os nós centrais (*God Nodes* / *Hubs*).
- **Interatividade Total no Browser**: Zoom com roda do mouse, pan com arraste de fundo, drag & drop de nós, busca textual instantânea, filtros por tipo de nota e painel lateral retrátil com deep-links para o Obsidian (`obsidian://open?file=...`).
- **Paridade entre CLI e MCP**: Linha de comando com subcomando `mem graph view` (com abertura direta no navegador padrão), `mem graph export`, retrocompatibilidade em `mem export --html` e ferramenta MCP `memory_visualize_graph` para agentes de IA.

## Decision Outcome

Adotou-se o pacote `internal/graphview` e as integrações na CLI (`cmd/mem/`) e no servidor MCP (`internal/mcp/`):

1. **Pacote `internal/graphview`:**
   - `model.go`: Estruturas de dados `GraphView`, `Node`, `Edge` e `GraphStats` (total de nós, arestas, hubs, densidade).
   - `builder.go`: Extrai dados do SQLite ou PostgreSQL, executa travessia BFS para subgrafos focados, calcula PageRank ponderado com escala dinâmica de raios (8px a 32px) e infere tipos e cores semânticas (`concept` ➔ azul, `decision` ➔ coral/rose, `guide` ➔ esmeralda, `synthesis` ➔ fúcsia, `reference` ➔ púrpura).
   - `template.go`: Gera o documento HTML5/SVG autocontido incorporando simulação de forças gravitacionais (Coulomb, Hooke e amortecimento inercial em JS puro).

2. **Comandos CLI (`cmd/mem/graph.go` e `cmd/mem/main.go`):**
   - `mem graph view [--root <nota>] [--depth 2] [--out <saida.html>] [--db <arq>] [--postgres <url>]`: Gera a visualização e abre automaticamente no navegador padrão do SO (`rundll32 url.dll,FileProtocolHandler` no Windows, `xdg-open` no Linux, `open` no macOS).
   - `mem graph export [--root <nota>] [--depth 2] [--out <saida.html>] [--open]`: Gera o arquivo no disco sem abrir o browser por padrão.
   - `mem export --html <saida.html> [--root <nota>]`: Opção alternativa sob o comando de exportação para simetria com `--canvas`.

3. **Nova Ferramenta MCP (`internal/mcp/`):**
   - `memory_visualize_graph`: Permite que agentes de IA (Claude, Cursor, Antigravity) gerem mapas visuais sob demanda, retornando o caminho do arquivo HTML e resumo de métricas topológicas.

### Positive Consequences

- **Acessibilidade Universal**: Qualquer dispositivo com um navegador web moderno pode inspecionar interativamente o cérebro do repositório.
- **Segurança e Privacidade Total**: Zero conexões de rede externas; 100% do processamento de física ocorre localmente na máquina do usuário.
- **Observabilidade Estrutural Aprimorada**: A combinação de PageRank com cores semânticas facilita a identificação imediata de gargalos, dependências circulares e ilhas de conhecimento isoladas.
- **Integração com Obsidian Preservada**: O painel lateral oferece link direto para abrir e editar a nota no Obsidian.

### Negative Consequences / Trade-offs

- **Desempenho com Grafos Muito Grandes**: Para grafos com mais de 5.000 nós simultâneos, simulações de forças em SVG puro no browser podem sofrer com queda de taxa de quadros (mitigável via limitação de subgrafo com `--root` e `--depth`).

---

## Links

- [Graphify Prior Art Reference](https://github.com/Graphify-Labs/graphify)
- [ADR-008: Interoperabilidade Obsidian e JSON Canvas 1.0](008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)
- [ADR-014: Centralidade de Grafo com PageRank Ponderado](014-centralidade-de-grafo-com-pagerank-ponderado.md)
- [Pacote internal/graphview](file:///C:/repository/my-memory/internal/graphview)
