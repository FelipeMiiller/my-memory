# ADR-036: Consolidação Pós-Release v1.3.0 — Backlog Técnico e Roadmap para v1.4.0

- **Date**: 2026-09-16
- **Status**: Accepted
- **Deciders**: Felipe Miiller
- **Tags**: backlog, technical-debt, release-planning, graph-viewer, embedder, viewer, multi-repo, encoding

## Context and Problem Statement

A sessão de release da v1.3.0 (PR #4 merged em `4c2e204`, tag `v1.3.0` publicada) foi acompanhada de trabalho substantivo no **Cofre Central de Conhecimento** (`G:\My Drive\central-memory`), na configuração federada (`~/.memory/config.yaml`), na **documentação** (`docs/CENTRAL_VAULT.md` + ADR-035) e principalmente no **Visualizador Interativo de Grafo** (`graph-v2.html` 60 KB, Tailwind via CDN + Cytoscape.js + 13 componentes isolados).

Durante esse trabalho foram identificados **bugs, gaps de feature e melhorias de DX** que não cabiam no escopo da release v1.3.0 e portanto ficaram pendentes. Sem uma consolidação formal, esses itens correm o risco de se perder entre sessões — especialmente os que vivem só em screenshots e no histórico de conversa, sem rastro no repositório.

Este ADR **documenta, prioriza e atribui** esses itens como backlog explícito para a próxima sprint (v1.4.0), separando-os em três trilhas: bugs críticos, gaps funcionais do viewer, e itens diferidos do roadmap maior.

## Decision Drivers

- **Memória institucional**: itens discutidos oralmente ou via screenshot sem rastro no repositório desaparecem entre sessões.
- **Escopo da release**: v1.3.0 já estava congelada (CI verde, PR mergeado, tag publicada) — incluir novos itens quebraria o fluxo.
- **Granularidade útil**: preferir muitos ADRs curtos para decisões futuras; este ADR serve como **inventário**, não como decisão técnica por item.
- **Acionabilidade imediata**: cada item deve ter descrição, repro e caminho de fix sugeridos para começar a próxima sessão com lista pronta.
- **Coerência com ADRs anteriores**: ADR-033/034 fecharam o ciclo do cofre central; ADR-035 já é Proposed (embedder embutido); este ADR-036 fecha a lacuna operacional.
- **Severidade**: bugs que comprometem a função principal do viewer têm prioridade sobre polish visual.

## Considered Options

1. **Criar um ADR-036 por item** (rejeitado: overhead desnecessário, fragmenta o contexto da sessão).
2. **Adicionar tudo como issues GitHub e pular ADR** (rejeitado: o usuário não usa GitHub Issues como fonte primária; o vault `.memory/` e os ADRs são a fonte da verdade local).
3. **Criar um único ADR-036 de consolidação** (escolhido — preserva contexto, é acionável, cabe numa página).
4. **Não documentar e seguir memória** (rejeitado: viola a regra 6 dos Princípios do Agente — "sempre conferir MD da prova", análoga para qualquer decisão).

## Decision Outcome

Adota-se a **Opção 3**: criar este ADR-036 como inventário único e priorizado, vinculando cada item à trilha (Bugs / Viewer / Roadmap) e ao seu caminho de fix. Itens serão promovidos a ADRs dedicados ou specs `tlc-spec-driven` quando entrarem em execução.

### Topologia do backlog

```
ADR-036 (consolidação)
├── Trilha A — Bugs críticos (bloqueiam uso)
│   ├── A1: Encoding UTF-8 quebrado no template antigo do `mem graph`
│   ├── A2: `mem graph --db <outro>` ignora DB e usa repo do git
│   └── A3: Cluster labels do v2 viewer sumiram (CSS ausente)
├── Trilha B — Gaps funcionais do viewer
│   ├── B1: Tag filter chips (chips de tipo) removidos na migração v2
│   ├── B2: ConfigModal read-only — precisa ser editável com lista de repos
│   ├── B3: Multi-repo viewer — flag `--all-repos` ainda não implementada
│   ├── B4: `mem open` no CLI só suporta Obsidian
│   └── B5: OpenWith modal no viewer (parcial — falta persistir reader escolhido)
└── Trilha C — Itens diferidos do roadmap maior
    ├── C1: ADR-035 — implementação do embedder embutido (ONNX MiniLM)
    ├── C2: Backend --all-repos com datasets reais do catálogo global
    ├── C3: Mini-mapa interativo (drag-to-navigate)
    ├── C4: Animações de entrada dos nodes
    └── C5: Edge labels no hover (já implementado mas precisa polish)
```

## Detalhamento dos itens

### Trilha A — Bugs críticos

#### A1. Encoding UTF-8 quebrado no `mem graph` (template antigo)

- **Sintoma**: `mem graph --repo central-memory --out graph-central.html` gera HTML com caracteres acentuados trocados (`MemÃ³ria` em vez de `Memória`, `SÃ­ntese` em vez de `Síntese`, `UsuÃ¡rio` em vez de `Usuário`). Reproduzido em 2026-09-16.
- **Causa provável**: `internal/graphview/template.go` escreve bytes UTF-8 sem declarar `<meta charset="UTF-8">` corretamente OU o template Go embed usa encoding Latin-1 no `WriteString`.
- **Repro**:
  ```bash
  cd C:\repository\my-memory
  .\bin\mem.exe graph --db 'G:\My Drive\central-memory\memory.db' --repo central-memory --out 'C:\repository\my-memory\graph-central.html' --open=false
  # Abrir o HTML gerado: caracteres acentuados aparecem trocados.
  ```
- **Fix sugerido**:
  - Conferir `internal/graphview/template.go` linha ~50–80 (onde o `<head>` é escrito).
  - Garantir `template.New(...).Parse(src)` com `src` UTF-8.
  - Adicionar `<meta charset="UTF-8">` no `<head>` (pode já estar; conferir).
  - Re-rodar teste manual e abrir no Chrome para validar.
- **Severidade**: Média (afeta readability mas não funcionalidade; usuário pode usar o viewer v2 que já está em UTF-8).
- **Workaround atual**: usar `graph-v2.html` (v2 viewer, codificado em UTF-8 sem BOM) para ambos os repos via `?src=data-central.json`.

#### A2. `mem graph --db <outro>` ignora DB não registrado

- **Sintoma**: ao passar `--db 'G:\My Drive\central-memory\memory.db'`, o CLI gera o grafo mas o slug do repo continua sendo o do git-detected (`repo_e4c8f3b1a2d5`), não o `--repo central-memory` nem um slug derivado do path do DB.
- **Repro**:
  ```bash
  cd C:\repository\my-memory
  .\bin\mem.exe graph --db 'G:\My Drive\central-memory\memory.db' --repo central-memory --out 'C:\repository\my-memory\graph-central.html' --open=false
  # O HTML gerado tem title="Grafo de Memória [central-memory]" (com --repo)
  # mas sem --repo, o title sai como "Grafo de Memória [repo_e4c8f3b1a2d5]".
  ```
- **Causa provável**: `cmd/mem/graph.go::runGraphCLI` faz auto-detect via git antes de cair em fallback, e ignora `--db` quando este aponta para vault não registrado no catálogo global.
- **Fix sugerido**:
  - Aceitar `--db` como override forte: se `--db` aponta para arquivo válido, usar path-derived slug (`basename(filepath.Dir(db))`).
  - Adicionar log estruturado quando override é aplicado: `"using db-derived repo slug: <slug>"`.
- **Severidade**: Alta (bloqueia geração de grafo para cofres não registrados, que é o caso do central-memory).
- **Workaround atual**: passar `--repo central-memory` explicitamente (resolve parcialmente — title fica correto mas stats ainda refletem 52 nodes que parecem ser fallback).

#### A3. Cluster labels do v2 viewer sumiram (CSS ausente)

- **Sintoma**: durante a sessão, removi os cluster labels (`../README`, `MY-MEMORY (13)`, etc.) que apareciam como overlays HTML. Decisão correta pelo usuário ("não quero cluster centrais, faça um gráfico"). Mas os divs `.cluster-label` ainda eram criados — a remoção só cortou o `setTimeout` que os chamava. Não há mais CSS nem uso, então este item é só de limpeza.
- **Fix sugerido**:
  - Remover a função `renderClusterLabels` (linhas ~327–360) do `graph-v2.html` se ainda estiver lá.
  - Remover `.cluster-label { ... }` do bloco CSS (foi injetado pelo patch anterior).
  - Confirmar via `grep -c cluster-label graph-v2.html` → deve retornar 0.
- **Severidade**: Baixa (código morto, não afeta usuário).
- **Discovery**: confirmar com `grep` no estado atual do arquivo.

### Trilha B — Gaps funcionais do viewer

#### B1. Tag filter chips removidos na migração v2

- **Sintoma**: o viewer antigo (`graph-central.html` legado) tinha chips de tipo no header (`Guia`, `Decisão`, `Referência`, `Síntese`, `Outro`) que filtravam nodes por tipo via clique. O v2 viewer (`graph-v2.html`) só tem o input de texto livre.
- **User feedback**: "Eu acho que tem que ficar espalhado mais de um jeito organizado e dê pra entender o gráfico" + "tinha a opção de eu clicar num ponto central aqui, eu conseguia abrir o Markdown com o Obsidian" — o usuário prefere clique em chip a digitar.
- **Fix sugerido**:
  - Extrair tipos únicos de `DATA.nodes` (`type: 'decision' | 'guide' | 'synthesis' | 'reference' | 'other'`) no init do `GraphRenderer`.
  - Renderizar chips no header após a stats row, com cores iguais às da paleta de nodes (já existe).
  - Handler: click no chip → filtra nodes (`cy.nodes().show()` / `.hide()` por tipo), estilo "active" no chip clicado.
  - Adicionar chip "Todos" para reset.
- **Severidade**: Média (DX — usuário prefere).
- **Local**: `graph-v2.html` linhas ~88–100 (após stats row).

#### B2. ConfigModal read-only — precisa ser editável com lista de repos

- **Sintoma**: o ConfigModal atual mostra `~/.memory/config.yaml` e `.memory/config.yaml` em `<pre>` read-only, com botão "Copiar". Não permite editar, remover entradas stale, ou ver quais repos estão registrados.
- **User feedback**: "a parte de configuração deixa poder editar. Eu acho que fica mais simples poder editar. E visualiza lá no repositório que tem informação que não é mais utilizada, né? Deixa a configuração o que pode, né? O que tiver a mais, a ideia é você remover, atualizar, corrigir."
- **Fix sugerido**:
  - Trocar `<pre>` por `<textarea contenteditable>` OU área editável com syntax highlight mínimo.
  - Adicionar seção "Repositórios Registrados" abaixo, com lista de `repositories[]` do config global:
    - Cada item: nome, path no disco, status (In Sync / Out of Sync / Stale / Missing).
    - Ações: "Reindexar", "Remover", "Editar path".
  - Botão "Salvar" persiste via backend HTTP novo (precisa endpoint `PUT /api/config`).
  - Indicador visual de stale: badge laranja "stale (>7d)" ou vermelho "missing" se o path não existe.
- **Severidade**: Média (DX — usuário pediu explicitamente).
- **Local**: `graph-v2.html` linhas ~144–162 (ConfigModal HTML) + JS componente.
- **Dependência**: requer novo endpoint HTTP no CLI `mem serve` (ou um mini-server `mem config serve`).

#### B3. Multi-repo viewer — flag `--all-repos` ainda não implementada

- **Sintoma**: o viewer v2 só carrega 1 dataset por vez (DATA const inline). Para ver o cofre central e o repo my-memory lado a lado, o usuário precisa abrir 2 abas manualmente.
- **User feedback**: "Eu ainda não descobri como eu acesso cada gráfico" — explícito que não descobriu como alternar.
- **Fix sugerido** (curto prazo):
  - Mover `const DATA = {...}` para um arquivo externo `data-my-memory.json` servido pelo `python -m http.server`.
  - Adicionar parâmetro `?src=data-central.json` no URL para trocar dataset sem mudar o HTML.
  - Gerar `data-central.json` extraindo DATA de `graph-central.html` (script PowerShell já existe em `C:\Users\Felipe\AppData\Local\Temp\extract_data.ps1`, mas precisa virar comando `mem graph --export-json`).
- **Fix sugerido** (longo prazo):
  - Backend Go: flag `--all-repos` em `mem graph` que carrega datasets de TODOS os repos do catálogo global, gera um único HTML com tabs funcionais que tropam o dataset client-side (lazy load via fetch).
  - Cada tab tem indicador de saúde (In Sync / Stale / Orphan).
- **Severidade**: Alta (UX bloqueante — usuário não consegue acessar central-memory no v2 viewer).
- **Local**: `cmd/mem/graph.go` + `internal/graphview/template.go` + `graph-v2.html`.

#### B4. `mem open` no CLI só suporta Obsidian

- **Sintoma**: `mem open <nota> --app obsidian|vscode|system` aceita 3 apps hardcoded. Falta Typora, Mark Text, Joplin, Zettlr.
- **User feedback**: "Eu quero poder abrir com qualquer leitor de Markdown. Pode ser Obsidian, pode ser qualquer outro. Precisa especificar não. Quando eu clicar, ele vai abrir uma opção de eu escolher."
- **Fix sugerido**:
  - Adicionar reader registry em `internal/openers/` com tabela de apps (nome, URI scheme, validador de presença opcional).
  - Lista inicial: Obsidian, VS Code, Typora, Mark Text, Joplin, Zettlr, System Default.
  - CLI: `mem open <nota>` sem `--app` → abre chooser interativo (ou lista texto se não-interativo).
  - `mem open --app list` → mostra readers disponíveis.
  - `--dry-run` já existe, mantém.
- **Severidade**: Média (parcialmente mitigado pelo OpenWith modal no v2 viewer, mas CLI ainda é gap).
- **Local**: `cmd/mem/open.go` + `internal/openers/registry.go` (novo).

#### B5. OpenWith modal — falta persistir reader escolhido

- **Sintoma**: o modal "📂 Abrir com..." no v2 viewer foi implementado (linhas ~600–650) com 7 readers e ações genéricas (Copiar path, Mostrar no Explorer). Não persiste a escolha — toda vez o usuário clica e tem que escolher de novo.
- **Fix sugerido**:
  - `localStorage.setItem('mem-preferred-reader', 'obsidian')` no `OpenWith.launch()`.
  - Próxima vez que `Sidebar.show(node)` for chamado, mostrar badge "✓ Obsidian" no botão "Abrir com..." indicando o reader default.
  - Botão "Tornar padrão" no modal.
- **Severidade**: Baixa (nice-to-have, não bloqueia).
- **Local**: `graph-v2.html` linhas ~600–650 (OpenWith component).

### Trilha C — Itens diferidos do roadmap maior

#### C1. ADR-035 — Implementação do embedder embutido (ONNX MiniLM)

- **Status atual**: ADR-035 Proposed (documentado, não implementado).
- **Próximo passo**: rodar a `tlc-spec-driven` para Specify → Design → Tasks → Execute.
- **Componentes previstos**:
  - `internal/embedder/builtin.go` (ONNX Runtime + MiniLM-L6 INT8)
  - `internal/embedder/selector.go` (fallback em camadas: ollama → builtin → FTS5)
  - `internal/embedder/model_downloader.go` (lazy download do modelo)
  - `mem embed {download, doctor, rebuild}` subcomandos
- **Severidade**: Alta (impacto direto na qualidade do cofre central sem Ollama).
- **Local**: `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` já tem o plano completo.

#### C2. Backend `--all-repos` com datasets reais

- Ver B3 (longo prazo).

#### C3. Mini-mapa interativo (drag-to-navigate)

- **Status atual**: mini-mapa existe mas é read-only (canvas com nodes estáticos).
- **Fix sugerido**: drag no mini-mapa → atualiza pan do canvas principal. Requer sincronização de eventos cytoscape ↔ canvas.
- **Severidade**: Baixa (polish).

#### C4. Animações de entrada dos nodes

- **Status atual**: layout do cytoscape já tem `animate: true`, mas os nodes aparecem todos juntos após o layout settle.
- **Fix sugerido**: stagger de entrada (100ms entre nodes), fade-in de edges após nodes.
- **Severidade**: Baixa (polish).

#### C5. Edge labels no hover (polish)

- **Status atual**: implementado parcialmente (linha 274–282 do v2) — mostra label da relação no hover.
- **Polish pendente**: aumentar contraste do background do label, fade-out mais suave, evitar piscar quando mouse passa pelo edge.
- **Severidade**: Baixa.

## Consequences

### Positive

- **Continuidade entre sessões**: ao abrir o repo amanhã, o próximo agente lê ADR-036 e tem contexto completo do que ficou pendente.
- **Priorização explícita**: usuário pode re-priorizar a qualquer momento editando este ADR (até mover de Trilha).
- **Rastro de decisões intermediárias**: itens rejeitados (ex: "criar ADR por item") ficam registrados como `Considered Options` para evitar re-discussão.
- **Descoberta de bugs latentes**: A1 (encoding) e A2 (DB override) só foram achados durante o trabalho desta sessão — sem este ADR, voltariam a se esconder.
- **Acoplamento fraco com ADRs futuros**: cada item Trilha C aponta pro ADR/spec que vai consumi-lo (ex: C1 → ADR-035).

### Negative

- **ADR-036 fica longo** (~250 linhas) — risco de ser ignorado por未来的 leitor por ser "só inventário". Mitigação: estrutura clara em Trilhas + bullets curtos.
- **Sem código de produção** — este ADR não move nenhum ponteiro de feature; é puramente organizacional. Pode parecer "trabalho burocrático" mas é investimento em memória institucional.
- **Itens podem ficar obsoletos**: se a próxima sessão resolver A1, este ADR-036 fica desatualizado até alguém remover o item. Mitigação: regra ADR-006 do projeto diz para criar ADR novo em vez de editar.

### Neutral

- **Não tem ADR de decisão técnica** — é ADR de organização de backlog, não de escolha arquitetural. Vai contra o uso "puro" de MADR mas cabe como extensão dentro do mesmo framework.
- **Itens em Trilha C continuam onde estavam** — ADR-035 não muda de status por causa deste ADR.
- **Geração automática de próximo ADR** — este ADR não força a criação de ADR-037 etc., apenas recomenda.

## Implementation Plan (próxima sessão)

### Sessão 1 (amanhã) — ordem sugerida

1. **A1 + A2 juntos** (~30 min): corrigir encoding do `mem graph` e o override de `--db`. Ambos tocam `internal/graphview/template.go` + `cmd/mem/graph.go`.
2. **B3 curto prazo** (~20 min): extrair DATA do v2 para `data-my-memory.json`, adicionar `?src=data-central.json` no URL. Gera dois JSONs servidos via `python -m http.server`.
3. **B1 + B2 parciais** (~45 min): renderizar chips de tipo no header + adicionar lista de repos no ConfigModal (read-only por enquanto, editável depois).
4. **A3 limpeza** (~5 min): `grep -c cluster-label graph-v2.html` → remover se > 0.
5. **Rodar `go test -count=1 ./...`** + smoke test do `mem graph` com encoding corrigido.

### Sessão 2 (depois)

- **B4** completo: registry de openers no CLI.
- **B5**: persistência do reader preferido no localStorage.
- **C3 + C4 + C5**: polish visual.

### Sessão 3+

- **C1 / ADR-035 implementação** via `tlc-spec-driven` (4 fases, escopo amplo).
- **B3 longo prazo**: `--all-repos` backend com tabs funcionais.

## References

- [ADR-033: Federated Central Vault and Repo Identity](./033-federated-central-vault-and-repo-identity.md) — central-memory origin
- [ADR-034: Protocolo Canônico Federado e Wikilinks Cross-Vault](./034-protocolo-canonico-federado-e-wikilinks-cross-vault.md) — release v1.3.0
- [ADR-035: Embedder Embutido com Fallback ONNX MiniLM](./035-embedder-embutido-com-fallback-onnx-minilm.md) — Trilha C1
- [docs/CENTRAL_VAULT.md](../CENTRAL_VAULT.md) — doc do cofre central criada nesta sessão
- [PR #4 — v1.3.0 release](https://github.com/FelipeMiiller/my-memory/pull/4) — release merged
- [Commit `33feb1d`](../../commit/33feb1d) — fallback FTS5 ancestor
- Skill [`tlc-spec-driven`](../../.agents/skills/tlc-spec-driven/SKILL.md) — para C1 (embedder embutido)
- Skill [`create-adr`](../../.agents/skills/create-adr/SKILL.md) — formato MADR usado aqui
