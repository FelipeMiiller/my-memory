---
name: my-memory
description: Skill de uso do My-Memory neste repositório. É um ponteiro para a skill pública e canônica em skills/mymemory/ — leia-a para instalar, configurar, indexar, buscar, analisar o grafo, integrar via MCP e criar notas no vault. Aplique quando o usuário pedir para instalar o mem, rodar mem init / mem index / mem search, configurar Postgres ou pgvector, criar nota dentro do vault, ou quando um agente de IA precisar entender o estado cognitivo do repositório.
---

# My-Memory — ponteiro para a skill canônica

> [!important]
> A skill **canônica, pública e distribuível** é
> [`skills/mymemory/SKILL.md`](../../../skills/mymemory/SKILL.md) — relativo a este arquivo.
> Este arquivo não a duplica: leia aquela, e volte aqui só para o que é específico deste repositório.

## 1. O pacote público é um roteador de seis skills

`skills/mymemory/SKILL.md` é o **roteador**: ele diz qual das seis skills focadas responde a tarefa, e carrega só o modelo transversal (a nota precisa estar *indexada, ligada e descrita*). Cada skill irmã tem o seu próprio `references/`:

| Skill | Assunto |
| --- | --- |
| [`mymemory-cli`](../../../skills/mymemory-cli/SKILL.md) | Rodar `mem` no shell/script: install, init, index, search, watch, hook, flags |
| [`mymemory-mcp`](../../../skills/mymemory-mcp/SKILL.md) | Cliente MCP, escolha entre tool e CLI, os 20 tools com entradas e saídas |
| [`mymemory-notes`](../../../skills/mymemory-notes/SKILL.md) | Escrever nota: taxonomia de frontmatter, wikilinks, arestas tipadas, tags |
| [`mymemory-graph`](../../../skills/mymemory-graph/SKILL.md) | Grafo e saúde do vault: inspect, path, hubs, clusters, impact, drift, doctor |
| [`mymemory-config`](../../../skills/mymemory-config/SKILL.md) | Precedência de config, split global/local, SQLite vs Postgres, cofre central |
| [`mymemory-knowledge-base`](../../../skills/mymemory-knowledge-base/SKILL.md) | Montar base de conhecimento a partir de várias fontes, com proveniência verificada |

Documenta o embedder `builtin` 384d (ADR-035). Não reescreva CLI, MCP ou markup aqui — escreva em `skills/mymemory/` e, se o fato for do repositório, na §2 abaixo.

## 2. Fatos que só existem neste repositório

### 2.1 Build, teste e formatação

`go build -v -o bin/mem.exe ./cmd/mem` · `go vet ./...` · `go test -count=1 ./...` · `gofmt -l .`

O gate completo de fim de tarefa (benchmarks, smoke test do binário, pipeline CI, `git status`) está em [always-quality-gate.md](../../../.agents/rules/always-quality-gate.md); a execução obrigatória de testes, em [always-test.md](../../../.agents/rules/always-test.md); a atualização de docs e specs antes de PR, em [always-update-docs-and-specs-on-pr.md](../../../.agents/rules/always-update-docs-and-specs-on-pr.md).

### 2.2 Skills irmãs (repo-local, fora do pacote público)

- [my-memory-format](../../../.agents/skills/my-memory-format/SKILL.md) — canônica de markup: Markdown Obsidian Flavored + JSON Canvas 1.0
- [create-adr](../../../.agents/skills/create-adr/SKILL.md) — formato MADR
- [tlc-spec-driven](../../../.agents/skills/tlc-spec-driven/SKILL.md) — Specify → Design → Tasks → Execute

### 2.3 ADRs vigentes

ADR-001 SQLite unificado · ADR-002 Go · ADR-003 TurboQuant 4-bit · ADR-004 SQL recursivo (CTE) · ADR-005 `[[wikilinks]]` · ADR-006 MCP · ADR-016 auto-scoping (superseded by ADR-040) · ADR-018 compile-not-retrieve · ADR-024 `mem inspect` · ADR-025 contexto progressivo e taxonomia `category` · ADR-033/034 Federação e Central Vault · ADR-035 embedder builtin 384d · ADR-039 drift calibrado · ADR-040 config global única + Postgres opt-in · ADR-041 tags sem casa removidas do grafo

### 2.4 Onde o resto mora

`docs/CLI_GUIDE.md` · `docs/ARCHITECTURE.md` · `docs/TURBOQUANT.md` · `docs/adr/` · [AGENTS.md](../../../AGENTS.md) · [.specs/ISSUES.md](../../../.specs/ISSUES.md) · `.specs/features/`