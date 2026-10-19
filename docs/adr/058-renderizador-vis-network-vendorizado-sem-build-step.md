# ADR-058: Renderizador vis-network vendorizado, sem build step e sem CDN

- **Data**: 2026-10-19
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: graphview, vis-network, vendorizado, go-embed, offline
- **Relacionada**: [ADR-020](020-visualizador-interativo-de-grafo-em-html-svg.md), [ADR-022](022-deteccao-de-comunidades-e-clusters-no-grafo.md), [ADR-054](054-escopo-de-paridade-com-o-obsidian.md)

## Contexto e Problema

`mem graph view` já gera um grafo interativo real. `internal/graphview/template.go` são 38 KB de HTML inline com simulação de forças, busca, filtro por PageRank e export SVG — e `internal/graphview/model.go` já carrega o que um visualizador profissional precisa: `Node.Color`, `Node.Radius`, `Node.PageRank`, `Node.CommunityID`, `Node.CommunityColor`, `Edge.Weight`, `GraphStats.Modularity`.

O problema não é ausência de dado. É que o renderer é **feito à mão**, sem contribuições externas, com física aproximada e sem clustering nativo.

vis-network cobre exatamente isso e respeita a restrição que importa: **build standalone UMD**, carregado por `<script>`, **sem bundler**. v10.1.2 (2026-08-19), dual-license Apache-2.0 OU MIT, ~862k downloads semana, mantida ativamente (último push 2026-09-20, repositório não arquivado).

O precedente que restringe a escolha é `b477200`: a lição não é "não use bibliotecas", é **não introduza build step**.

## Drivers da Decisão

- Física, zoom, pan e seleção de vizinhança de qualidade, sem manter o layout à mão
- Zero build step e zero CDN — o HTML tem que abrir offline, sempre
- Preservar o renderizador atual como padrão, sem alterar a saída de quem já usa
- Manter PageRank e modularidade Q calculados em Go

## Opções Consideradas

- **A — vis-network vendorizado com `go:embed`** (escolhida)
- **B — vis-network via CDN**
- **C — SVG puro gerado em Go, sem JS de terceiros**

## Resultado da Decisão

Escolhida a **Opção A**.

```
internal/graphview/assets/vis-network.min.js   652 kB, standalone UMD, 10.1.2
                                   LICENSE-vis-network.txt
                                   PROVENANCE.md   URL + versão + SHA-256 + data
```

Embutido com `go:embed`, então o binário distribuído já sabe renderizar sem instalação extra. `mem graph view --engine vis --out g.html` escreve **dois arquivos** lado a lado — `g.html` e `vis-network.min.js` — extraídos dos bytes embutidos. O par é portátil e abre sem rede.

O renderizador atual **permanece o padrão** e sua saída é byte-idêntica. `--engine vis` é opt-in.

### Por que não as outras

**B — CDN** foi rejeitada porque contradiz o propósito declarado do projeto. O artefato deixa de abrir offline, e o pin de versão vira pin de URL cujo conteúdo pode mudar. É a dependência de rede que o revert de `b477200` eliminou.

**C — SVG puro** foi rejeitada porque reimplementa física de forças, zoom e pan. O histórico do repo mostra que código de viewer nosso não se sustenta: foram 27.858 linhas antes do revert.

### Decisões de execução

| Decisão | Escolha | Racionale |
|---|---|---|
| Distribuição | `go:embed` + escrita ao lado do HTML | Binário funciona sozinho; o par emitido é portátil |
| Pin de versão | SHA-256 em `PROVENANCE.md`, verificado por teste | Detecta upgrade não auditado **antes** de um viewer ser gerado |
| Origem da versão | dist-tag do npm | O `package.json` do vis-network tem `0.0.0-no-version` (semantic-release); ler do repo dá placeholder |
| Clustering | Continua o LPA de `internal/graph` | A vis agrupa mas não produz modularidade Q (ADR-022) |
| Física | Desligada abaixo de 30 nós | Para 8 nós a simulação é ruído; posições determinísticas tornam o viewer reproduzível |
| Source map | **Não** vendorizado | 1,74 MB que só o devtools usa |

### Mapeamento

| `graphview.Node` | vis-network |
|---|---|
| `ID` / `Title` | `id` / `label` |
| `Color` → `CommunityColor` → `GetColorForCommunity` → `#64748b` | `color` |
| `Radius` | `value` |
| `PageRank` | painel de detalhe |

| `graphview.Edge` | vis-network |
|---|---|
| `Source` / `Target` | `from` / `to` |
| `Weight` | `width` |

`builder.go` **não muda**: o modelo existente já é o contrato.

### Consequências Positivas

- Física de verdade, zoom/pan e seleção sem código nosso
- `builder.go` intocado — mapping quase 1:1
- HTML sem origem remota (AC do spec)
- Upgrade só por bump explícito de hash

### Consequências Negativas

- **+652 kB no binário.** É o preço de não depender de rede; a alternativa (SVG próprio) custa manutenção
- Primeira dependência de runtime de terceiros no repositório, num projeto que não tinha nenhuma
- `template.go` continua sendo 38 KB de HTML inline — **não escalado**, ver ADR-059 para a resposta no frontend

## Links

- Spec: `.specs/features/graph-view-vis-engine/spec.md`
- Design: `.specs/features/graph-view-vis-engine/design.md`
- [ADR-020](020-visualizador-interativo-de-grafo-em-html-svg.md) — o visualizador atual
- [ADR-022](022-deteccao-de-comunidades-e-clusters-no-grafo.md) — modularidade Q que a vis não tem
- [ADR-059](059-spa-react-vite-em-viewer-servida-pelo-mcp.md) — a SPA, que não usa esta lib