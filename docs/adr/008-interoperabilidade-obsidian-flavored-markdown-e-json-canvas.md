# ADR-008: Interoperabilidade com Obsidian Flavored Markdown e JSON Canvas 1.0

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: obsidian, markdown, wikilinks, json-canvas, agent-skills, interoperability

## Context and Problem Statement

O projeto **My-Memory** utiliza Markdown como fonte da verdade soberana e wikilinks `[[...]]` para construção do grafo de conhecimento humano e navegação recursiva por CTEs (ADR-004 e ADR-005).

Contudo, na prática com agentes de IA e ecossistemas de notas pessoais (PKMs como Obsidian), notas Markdown do mundo real contêm construções mais ricas:
1. **Frontmatter YAML estruturado:** propriedades como `title`, `tags` e `aliases`.
2. **Wikilinks com âncoras e blocos:** sintaxes como `[[Nota#Cabeçalho|Alias]]`, `[[Nota#^block-id]]` ou links de mesma nota `[[#Subseção]]`. Anteriormente, o parser tratava toda a string como o nome do nó, poluindo o grafo relacional com nós espúrios.
3. **Visualização Espacial de Memória:** Agentes de IA e usuários precisam de uma forma visual e intuitiva para inspecionar subgrafos recuperados sem precisar ler tabelas de texto brutas.

Projetos de referência como [`kepano/obsidian-skills`](https://github.com/kepano/obsidian-skills) (por Steph Ango, CEO do Obsidian) e [`Graphify-Labs/graphify`](https://github.com/Graphify-Labs/graphify) estabelecem especificações abertas consolidadas para notas e mapas conceituais.

## Decision Drivers

- **Fidelidade Semântica do Grafo:** Normalizar wikilinks para que `[[Nota#Seção]]` aponte corretamente para o nó `Nota`, separando a âncora/bloco nos metadados da aresta.
- **Extração Abrangente de Metadados:** Capturar tags de frontmatter e aliases para enriquecer a busca semântica e FTS5.
- **Formato de Exportação Aberto:** Adotar o [JSON Canvas 1.0](https://jsoncanvas.org/spec/1.0/) (`.canvas`) como formato de mapa mental para o My-Memory.
- **Interoperabilidade com Agentes de IA:** Disponibilizar Agent Skills no padrão aberto em `.agents/skills/`.

## Considered Options

- **Opção A: Adoção das Especificações Obsidian Flavored Markdown e JSON Canvas 1.0**.
- **Opção B: Formato Proprietário de Grafo JSON** (manter schemas ad-hoc exclusivos do My-Memory).
- **Opção C: Parser Simples sem Frontmatter** (ignorar YAML e tratar apenas texto plano).

## Decision Outcome

Chosen option: **"Opção A: Adoção das Especificações Obsidian Flavored Markdown e JSON Canvas 1.0"**, because alinha o My-Memory diretamente com o formato aberto adotado por dezenas de ferramentas do ecossistema PKM, permitindo que usuários abram o arquivo `.canvas` gerado diretamente no Obsidian para visualizar seus clusters de memória em telas interativas com nós e arestas.

### Positive Consequences

- **Conexões de Grafo Limpas:** Eliminação de nós espúrios gerados por âncoras (`[[Nota#Seção]]` agora aponta precisamente para `Nota`).
- **Geração de JSON Canvas Nativa:** Novo pacote `internal/canvas` com layout automático espacial radial, garantindo que o comando `mem export --canvas` e a tool MCP `memory_export_canvas` gerem mapas mentais prontos para renderização.
- **Enriquecimento por Metadados:** Aliases e tags de frontmatter integradas ao modelo de busca.
- **Capacitação de Agentes:** Inclusão da skill `my-memory-format` em `.agents/skills/` (absorve as antigas `memory-md` + `json-canvas`).

### Negative Consequences

- **Dependência YAML:** Inclusão de dependência controlada `gopkg.in/yaml.v3` para parsing seguro de metadados.

## Pros and Cons of the Options

### Opção A: Obsidian Flavored Markdown e JSON Canvas 1.0 ✅ Chosen

- ✅ Compatibilidade imediata com Obsidian, Logseq e ferramentas de JSON Canvas.
- ✅ Especificação aberta e padronizada (JSON Canvas Spec 1.0).
- ✅ Melhor experiência para agentes de IA que navegam vaults de notas.
- ❌ Adiciona complexidade na regra de parsing de tags (diferenciar âncoras dentro de wikilinks de tags reais).

### Opção B: Formato Proprietário

- ✅ Nenhuma necessidade de seguir especificações externas.
- ❌ Exige criação de visualizador próprio do zero, sem interoperabilidade com ecossistemas existentes.

## Links e Referências

- [`kepano/obsidian-skills`](https://github.com/kepano/obsidian-skills)
- [JSON Canvas Spec 1.0](https://jsoncanvas.org/spec/1.0/)
- [`Graphify-Labs/graphify`](https://github.com/Graphify-Labs/graphify)
- [`akitaonrails/ai-memory`](https://github.com/akitaonrails/ai-memory)
- [`docs/REFERENCES.md`](../REFERENCES.md)
