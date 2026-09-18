# Tasks: obsidian-integration

- [x] **T1**: Parser de Obsidian Flavored Markdown e YAML Frontmatter (`internal/parser`)
  - Extração de YAML frontmatter (`tags`, `aliases`, propriedades)
  - Parser robusto de wikilinks (âncoras `#`, blocos `^`, aliases `|`, embeds `!`, links locais `[[#...]]`)
  - Testes unitários com casos reais do Obsidian

- [x] **T2**: Módulo e Especificação JSON Canvas 1.0 (`internal/canvas`)
  - Structs para Canvas, Nodes (`file`, `text`, `group`) e Edges
  - Gerador de IDs hex 16 chars
  - Algoritmo de layout automático espacial (radial/grid sem sobreposição)
  - Construtor `FromNeighbors(centerNode string, neighbors []string)`
  - Testes de conformidade e integridade referencial

- [x] **T3**: Tool e Handler MCP `memory_export_canvas` (`internal/mcp`)
  - Schema da tool em `internal/mcp/tools.go`
  - Handler e conversor de vizinhos em `internal/mcp/handlers.go`
  - Testes unitários no servidor MCP

- [x] **T4**: Comando CLI `mem export --canvas` (`cmd/mem/main.go`)
  - Flag `--canvas`, `--depth`, `--out`, `--db`, `--postgres`, `--repo`
  - Ajuda CLI atualizada

- [x] **T5**: Agent Skills Locais (`.agents/skills/`)
  - Criar `.agents/skills/my-memory-format/SKILL.md` (absorve as antigas `memory-md` + `json-canvas`)

- [x] **T6**: Documentação e Decisão de Arquitetura (ADR-008)
  - Criar `docs/adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md`
  - Atualizar `.specs/STATE.md` e gerar `validation.md`
