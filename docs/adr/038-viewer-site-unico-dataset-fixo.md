# ADR-038: Viewer com Site Único e Dataset Fixo (B3 do ADR-036 Revertido)

- **Date**: 2026-09-17
- **Status**: Accepted
- **Deciders**: Felipe Miiller
- **Tags**: viewer, graphview, product-decision, scope-reduction, deferred

## Context and Problem Statement

O ADR-036 original (Trilha B3) propunha **multi-repo viewer** com flag `--all-repos` e `?src=<dataset>.json` no URL para trocar entre cofres (my-memory, central-memory). Esta abordagem foi implementada nos commits `a7fe4dd` (extração DATA + ?src=) e `dda9d1b` (repos registry), mas durante a sessão de validação visual o Felipe revisou a UX e mudou de direção:

> *"Só que agora é só pra ter um site, não é pra ter o gráfico central e outro gráfico, é pra ter só um site, usar tem o Inde, né? Apagar os outros."*

Resultado: o viewer virou um monolito de complexidade desnecessária:
- 2 datasets JSON (`data-my-memory.json`, `data-central.json`) servidos lado a lado
- Lógica de URL parsing para `?src=`
- Default arbitrário (qual carregar quando não tem query string)
- Tabs vazios no header (placeholder "central-memory")
- Possibilidade de bug silencioso (DATA=null enquanto fetch ainda não completou, ver commit `2bf79bd`)

A reversão do B3 é uma decisão de produto: **o viewer serve UM único dataset fixo** (`data-central.json`, que é o "índice" do cofre central), sem mecanismo de troca em runtime.

## Decision Drivers

- **UX simples > UX flexível**: o Felipe não quer gerenciar múltiplos sites/abas. Um único site, um único foco.
- **"Inde" como fonte canônica**: o cofre central (`G:\My Drive\central-memory`) é o "índice" do conhecimento. O viewer sempre mostra central-memory.
- **Menos código = menos bugs**: remover `?src=` parsing elimina ~30 linhas de bootstrap que tinham potencial de race condition.
- **Single source of truth**: 1 dataset, 1 tela, 1 decisão de UI. Sem "qual dataset o usuário quis ver?"
- **Migração futura simplificada**: ADR-037 (Vite + Vanilla TS rewrite) já vai reorganizar o viewer inteiro; voltar pra single-dataset agora dá coerência arquitetural.

## Considered Options

### 1. Multi-repo viewer (B3 original) — REJEITADO
Manter `?src=data-<repo>.json` + tabs + auto-detect de default.
- ❌ Complexidade de parsing URL
- ❌ Risco de bug async (race condition entre fetch e render — aconteceu em `2bf79bd`)
- ❌ Overhead cognitivo pro Felipe ("qual gráfico abrir?")
- ✅ Flexibilidade pra power users
- ✅ Roadmap original do ADR-036

### 2. Site único, dataset via config global — REJEITADO
Manter single-dataset mas ler path do `~/.memory/config.yaml` (catálogo global) em vez de hardcodar `data-central.json`.
- ❌ Requer backend HTTP pra ler filesystem (browser não pode)
- ❌ Adiciona complexidade de fetch + parsing YAML
- ❌ Mesmo problema de race condition do option 1
- ✅ Configurável sem editar código

### 3. Site único, dataset hardcoded `data-central.json` — ESCOLHIDO
Hardcodar `const SRC_PARAM = 'data-central.json'` no `graph-v2.html`. Sem URL parsing. Sem tabs. Sem auto-detect.
- ✅ Zero código de configuração
- ✅ Race condition impossível (não tem fetch dinâmico)
- ✅ Coerente com ADR-037 (futuro Vite + Vanilla TS também vai ser single-site)
- ✅ "Inde" sempre disponível
- ❌ Não-flexível (mas o Felipe explicitamente não quer flexibilidade)

## Decision Outcome

**Opção 3 escolhida**: viewer serve um único dataset fixo `data-central.json`.

### Mudanças aplicadas no commit `2bf79bd`

- `SRC_PARAM = 'data-central.json'` (hardcoded)
- Removido `new URLSearchParams(location.search).get('src') || ...`
- Removido tabs de repo vazios no header
- Removido `data-my-memory.json` (era cópia do mesmo conteúdo com label errado)
- Bootstrap async simplificado: fetch direto sem fallback

### O que NÃO muda

- `mem graph --db <outro> --out graph-central.html` continua gerando HTML standalone (template.go). Usado pra outros fins (não viewer).
- O template standalone (`internal/graphview/template.go`) **NÃO** foi alterado — ele ainda produz HTML completo auto-contido via `go run`, não tem `?src=`.
- Central-memory DB continua sendo o índice (`G:\My Drive\central-memory\memory.db`).
- `data-central.json` continua sendo extraído de `graph-central.html` (gerado por `mem graph --db central-memory`).

### Quando reconsiderar

Se no futuro o Felipe trabalhar com múltiplos cofres distintos em paralelo (ex: 1 por cliente/empresa), e a fricção de "abrir 2 abas" virar bottleneck real, reverter esta decisão criando ADR-039 com justificativa concreta baseada em uso real (não hipotético).

## Consequences

### Positive

- **Viewer mais simples**: 1 arquivo JS, 1 dataset, 0 URL parsing, 0 race condition.
- **Coerência com ADR-037**: o rewrite pra Vite pode assumir single-dataset como invariante, simplificando o data layer.
- **Menos código a manter**: ~30 linhas de bootstrap removidas. Menos superficie de bug.
- **Foco**: Felipe abre 1 site, vê o índice do cofre central, trabalha.

### Negative

- **Perda de flexibilidade**: se precisar ver o grafo de outro vault, tem que regenerar `data-central.json` apontando pro DB correspondente (manual).
- **Bug potencial futuro**: trocar de dataset significa editar `graph-v2.html` (string `data-central.json`) e rebuild. Não é hot-swap.
- **ADR-036 B3 fica em status "Reverted"**: precisa de update no ADR-036 pra documentar a reversão.

### Neutral

- **Não impacta CLI standalone**: `mem graph --out` continua gerando HTML completo auto-contido.
- **Não impacta MCP**: ferramentas MCP não dependem do viewer.
- **Não impacta Postgres**: `MY_MEMORY_PG_URL` continua funcionando independente.

## Implementation Plan

### Já feito (commit `2bf79bd`)
1. ✅ Hardcoded `SRC_PARAM = 'data-central.json'`
2. ✅ Removido `?src=` parsing
3. ✅ Removido `data-my-memory.json` (era duplicado)
4. ✅ Adicionada `<meta http-equiv>` defense (A1 do ADR-036)
5. ✅ Migrado layout pra fCoSE + text-background labels + hide-tags

### Próxima sessão (quando começar migração Vite)
1. Replicar single-dataset em `graph-v3/` sem reintroduzir multi-repo
2. Documentar a invariante em `src/data/loader.ts` ("sempre `data-central.json`")
3. Se múltiplos vaults forem necessários, criar ADR-039 com use case concreto

### A fazer AGORA
1. Atualizar ADR-036 marcando B3 como "Reverted per ADR-038"
2. Criar spec `.specs/features/post-release-v1.3.0-bugfixes/` consolidando tudo desta sessão

## References

- [ADR-036: Consolidação Pós-Release v1.3.0](./036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md) — Trilha B3 original, agora revertida
- [ADR-037: Rewrite do Viewer com Vite + Vanilla TS](./037-rewrite-viewer-com-vite-vanilla-ts.md) — migração futura, herda single-dataset como invariante
- [Commit `2bf79bd`](../../commit/2bf79bd) — feat(viewer): single-site + fCoSE layout + text-background labels + hide-tags toggle
- [Commit `a7fe4dd`](../../commit/a7fe4dd) — feat(viewer): B3 multi-repo via ?src= URL param (AGORA REVERTIDO)
- Feedback do Felipe em sessão 2026-09-17: "Só que agora é só pra ter um site"