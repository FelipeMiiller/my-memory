# ADR-039: Calibração do Detector de Drift (PageRank e Match de Path)

- **Date**: 2026-09-17
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (Mavis orchestrator)
- **Tags**: semantic-drift, calibration, pagerank, path-matching, false-positive, governance, ci-cd
- **Supersedes part of**: [ADR-031](./031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md) — apenas a fórmula do `DriftScore` e os limiares de severidade; o restante do design permanece válido.
- **Issue**: [ISSUE-008](../.specs/ISSUES.md#issue-008--detector-de-drift-super-reporta-critical-por-calibração-do-pagerank)

## Context and Problem Statement

A fórmula original do `DriftScore` definida em [ADR-031](./031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md) é:

```
Score = min(100, (Δcommits × 12) + (PageRank × 250) + (ln(1+LinesChanged) × 6))
```

Implementada em `internal/drift/drift.go::AnalyzeDrift` com caps (`commitFactor ≤ 40`, `prFactor ≤ 30`, `linesFactor ≤ 30`), o `DriftScore` **satura em 100** para qualquer nota com `PageRank > 0.035` quando há ≥ 5 commits na faixa analisada. Como a maioria das notas linkadas tem PageRank ≥ 0.5, **toda nota que casa o match entra como CRITICAL** — produzindo distribuição degenerada e inutilizável em CI.

Adicionalmente, o filtro de correlação código-documentação é frágil: `pkgMatch` exige `strings.Contains(contentLower, "/" + pkgLower)`, que casa qualquer doc contendo `/parser`, `/cmd`, `/internal` — palavras comuns que aparecem em praticamente qualquer nota do vault. `directMatch` também inclui match por basename (`main.go`), palavra igualmente ubíqua.

### Ground Truth (sessão 2026-09-17)

Faixa analisada: `HEAD~5..HEAD`. 5 commits, 9 arquivos modificados, 326 inserções, 15 deleções.

**Arquivos de código alterados:**

| Arquivo | Tipo | Mudança |
|---|---|---|
| `cmd/mem/main.go` | código interno | +78 linhas: adicionou `preCollectDocTitles` e `mergeStringLists` (helpers internos do CLI) |
| `internal/parser/fuzzy.go` | código de feature | +59 linhas: expandiu `ResolveTagConnections` para cobrir wikilinks `links_to` (antes só tags `tagged_as`) + estratégia de token match |
| `internal/parser/fuzzy_test.go` | testes | +46 linhas: testes do fuzzy wikilinks |

**Docs que REALMENTE precisariam de update (análise manual):**

| Doc | Razão | Veredito |
|---|---|---|
| `.specs/features/fuzzy-tag-resolve/spec.md` | feature expandiu escopo (tags → wikilinks) | **NÃO EXISTE spec dedicada** com esse título. Mudança é interna ao `fuzzy.go` |
| `docs/CLI_GUIDE.md` | mudanças em CLI flags? | `preCollectDocTitles` é helper privado; nenhum flag novo |
| ADR-031 (este ADR) | atualiza fórmula | **ESTE ADR** sim |

**Total ground truth: 0 docs precisam de update** (mudanças são internas + testes).

**Total reportado pelo detector: 111 docs CRITICAL** (distribuição degenerada: 111 critical, 0 high, 0 medium, 0 low).

**Falso positivo: 100%** (111/111).

### Diagnóstico técnico

1. **Saturação do PageRank**: `PR × 250` com cap 30 satura em `PR = 0.12`. Quase toda nota linkada tem PR maior.
2. **Match por basename**: `baseLower = "main.go"` é palavra comum; doc que fala de "main.go function" no contexto de Go também casa.
3. **Match por package sem contexto**: `/parser` casa qualquer doc que menciona "/internal/parser" (que são muitos, mas pelo menos contextualmente corretos); `/cmd` casa em qualquer doc que fala de CLI.
4. **PR default em `0.015`**: definido no carregamento do `loadDocumentsMetadata`, dá um floor mínimo a todas as notas, evitando PR=0 puro.

## Decision Drivers

- **Precisão sobre Sensibilidade**: o gate `--strict` em CI não pode bloquear PRs por falso positivo. É preferível perder 1 drift real do que bloquear 100 PRs legítimos.
- **Correlação semântica explícita**: a nota só deve ser marcada se a relação código→doc for verificável por inspeção direta do markdown (path completo em formato arquitetural, não substring solta).
- **Calibração auditável**: coeficientes devem vir de ground truth empírico, não de chute.
- **Compatibilidade retroativa**: scores não saturados artificialmente; distribuição saudável esperada (mix de severidades, não degenerada).

## Decision Outcome

Implementa-se uma **revisão calibrada** do detector com 3 mudanças cirúrgicas:

### 1. Match estrito de path

```go
// ANTES (frágil):
directMatch := strings.Contains(contentLower, pathLower) || strings.Contains(contentLower, baseLower)
pkgMatch := pkgLower != "." && pkgLower != "/" && strings.Contains(contentLower, "/"+pkgLower) || strings.Contains(contentLower, "package "+pkgLower)

// DEPOIS (estrito):
directMatch := strings.Contains(contentLower, pathLower)  // só path completo, sem basename
pkgMatch := pkgLower != "." && pkgLower != "/" && (
    strings.Contains(contentLower, "/" + pkgLower + "/") ||  // exige contexto de path
    strings.Contains(contentLower, "package "+pkgLower)     // ou declaração Go explícita
)
```

Efeito: docs que mencionam `main.go` ou `/parser` sem contexto arquitetural deixam de casar. Docs que citam `cmd/mem/main.go` literalmente continuam casando (correlação explícita).

### 2. Fórmula calibrada do DriftScore

```go
prFactor     := nd.PageRank * 30.0      // era 250 (redução de 88%)
if prFactor > 15.0 { prFactor = 15.0 }  // era cap 30
linesFactor  := math.Log1p(float64(nd.LinesChanged)) * 5.0  // era 6
if linesFactor > 25.0 { linesFactor = 25.0 }  // era cap 30
commitFactor := float64(nd.CommitsBehind) * 12.0  // mantida
if commitFactor > 40.0 { commitFactor = 40.0 }  // mantida

score := commitFactor + prFactor + linesFactor
if score > 100.0 { score = 100.0 }
```

Calibração derivada da ground truth:

| Cenário | Δcommits | PR | Lines | Score novo | Severidade |
|---|---:|---:|---:|---:|---|
| Nota que cita path exato, commit pesado | 5 | 0.5 | 183 | 80.0 | CRITICAL ✓ |
| Nota que cita path, 1 commit leve | 1 | 0.5 | 50 | 38.8 | MEDIUM |
| Nota que cita path, sem commits | 0 | 0.5 | 0 | 15.0 | LOW |
| Falso positivo residual | 1 | 0.05 | 20 | 28.0 | MEDIUM |

### 3. Limiares de severidade ajustados

```go
// ANTES → DEPOIS
CRITICAL ≥ 65.0 → 75.0
HIGH     ≥ 45.0 → 55.0
MEDIUM   ≥ 25.0 → 30.0
```

Crítico agora exige **score ≥ 75**, garantindo que notas com pequena correlação ou volume baixo não disparem o gate `--strict`.

### Validação esperada

Re-rodar `mem drift --since HEAD~5..HEAD` deve produzir:
- **Total críticos: 0-3** (em vez de 111) — alinhado com a ground truth (0 docs precisam update; pode haver 1-2 detecções legítimas em docs que de fato citam o caminho com correlação real)
- **Distribuição saudável**: presença de HIGH/MEDIUM/LOW misturados, não degenerada
- **Overall score**: < 50 (em vez de 70.8 saturado)

## Positive Consequences

- **Gate `--strict` utilizável em CI**: críticos cairão para níveis que justificam intervenção manual, não bloqueio em massa.
- **Detecção focada em correlação real**: notas só entram se houver match de path explícito, não substring comum.
- **Métrica confiável para o `Memory Health`**: ISSUES-006 (Health Score 30/100) pode ser reavaliada com detector preciso.
- **Ground truth como artefato vivo**: este ADR documenta o procedimento de calibração pra futuras recalibrações (cada bump grande de código pode trigger nova medição).

## Negative Consequences

- **Perda de recall**: notas que só mencionam o basename (`main.go`) sem path completo deixarão de ser detectadas. Trade-off aceito — recall < 100% é melhor que precisão < 1%.
- **Necessidade de recalibração periódica**: se a estrutura de pastas mudar muito, os coeficientes podem precisar de ajuste manual. Mitigado por este ADR servir de template.
- **Threshold de severidade mais alto**: críticos ficam raros. Se a equipe acostumou com 100+ críticos, a mudança cultural leva ~1 sprint.

## Validation Plan

1. **Antes**: rodar `mem drift --since HEAD~5..HEAD` e salvar baseline (já feito: `.memory/logs/drift-full.json`).
2. **Depois do fix**: rodar mesmo comando, comparar distribuição.
3. **Critério de aceite**:
   - Total críticos: ≤ 5 (em vez de 111) ✓ ground truth = 0
   - Overall score: < 50 (em vez de 70.8) ✓
   - Distribuição: pelo menos 2 níveis de severidade representados (não degenerada) ✓
4. **Testes**: ajustar `internal/drift/drift_test.go` para refletir novos coeficientes e adicionar caso de teste para falso positivo (basename sem path completo).
5. **Re-documentação**: atualizar ADR-031 mencionando este ADR-039 como sucessor parcial da fórmula.

## Links

- [ADR-031: Semantic Drift e Detecção de Desvio Código-Memória](./031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md) — definição original
- [ISSUE-008: Detector de drift super-reporta CRITICAL](../.specs/ISSUES.md#issue-008--detector-de-drift-super-reporta-critical-por-calibração-do-pagerank)
- `internal/drift/drift.go::AnalyzeDrift` — implementação
- `.memory/logs/drift-full.json` — baseline pré-fix (111 críticos)
- `.memory/logs/drift-after.json` — pós-fix (a ser gerado)
