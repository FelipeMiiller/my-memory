# Inventário read-only do `my-memory` — ADR-065, fase 1

**Data**: 2026-11-14
**Medido em**: `D:\repository\my-memory` (branch `develop`, HEAD `82f6a9f`)
**Natureza**: **read-only**. Nada foi escrito no repositório, nenhuma decisão foi
tomada, nada foi cortado. Este relatório vive fora do repo (scratch) porque a
fase é de medição; persisti-lo em `.specs/` é um passo separado, a autorizar.

---

## 0. Resumo em uma tela

| Pergunta | Resposta medida |
|:---|:---|
| A documentação cresceu mais rápido que o código? | **Sim, muito.** Documentação/código foi de **43,7%** para **58,6%** (mesma fórmula do handover: linhas de `.md` ÷ linhas de Go). |
| Qual é o peso da camada spec-driven? | **27.795 linhas (50,6% de toda a documentação)** em 59 features. |
| As ADRs dizem a verdade? | **Não.** Só **5 de 65** dizem "Implementado"; 50 dizem "Accepted" — inclusive ADRs que **estão** implementadas. É dívida de **metadado**, não de entrega. |
| O que está morto no código? | **19 funções inalcançáveis** (o handover dizia 174) — indicador, não veredito. |
| O handover está em dia? | **Não, em dois pontos** (ver §6). |
| Existe conteúdo duplicado? | **Não.** Zero `.md` duplicados byte a byte no repo; **zero** grupos com `content_hash` repetido no cofre (108 documentos, todos distintos). |
| O que está prometido e não entregue? | **5 gaps** (GAP-6 a GAP-10) + **1 ADR** aceito e não implementado (ADR-058) + **6 specs** sem execução. |

---

## 1. Documentação — o que existe

| Categoria | Arquivos | Linhas | ~KB |
|:---|---:|---:|---:|
| ADRs (`docs/adr/`) | 65 | 8.492 | 582 |
| Specs — `spec.md` | 59 | 8.836 | 636 |
| Specs — `tasks.md` | 57 | 11.983 | 567 |
| Specs — `validation.md` | 53 | 6.976 | 504 |
| Specs — `design/context.md` | 1 | 217 | 10 |
| Docs técnicos (`docs/`) | 12 | 4.243 | 264 |
| Skills (`.agents/`) | 21 | 4.091 | 196 |
| Skills (pacote público `skills/`) | 16 | 2.805 | 103 |
| Spec-driven (raiz + `.specs/*.md`) | 18 | 3.786 | 297 |
| Raiz (`*.md`) | 8 | 2.256 | 144 |
| Outros `.md` | 8 | 1.222 | 86 |
| **TOTAL** | **318** | **54.907** | **3.394** |

**Comparação com o handover (2026-11-13):** 296 arquivos / ~34.500 linhas →
**318 / 54.907**. Crescimento de **+22 arquivos / +59% de linhas** em um dia de
trabalho — e o `.md` do repo já é **maior** que o código de produção
(54.907 vs 47.094 linhas).

**Onde o peso está:** a camada spec-driven (spec + tasks + validation) são
**27.795 linhas = 50,6%** de toda a documentação. É a maior fatia isolada do
repo, e é a que mais cresce por feature.

## 2. Código — tamanho e composição

| Pacote | Prod | Teste |
|:---|---:|---:|
| `cmd/mem` | 11.257 | 10.952 |
| `internal/mcp` | 6.376 | 6.777 |
| `internal/db` | 5.301 | 6.055 |
| `internal/store` | 5.026 | 4.482 |
| `internal/codeast` | 3.126 | 2.891 |
| `internal/graph` | 2.033 | 1.239 |
| `internal/supervisor` | 1.794 | 2.033 |
| `internal/event_runtime` | 1.449 | 1.923 |
| `internal/embedder` | 1.178 | 1.607 |
| `internal/config` | 1.104 | 1.351 |
| `internal/parser` | 1.001 | 1.139 |
| `internal/federation` | 997 | 662 |
| `internal/drift` | 897 | 627 |
| `internal/watcher` | 743 | 1.034 |
| `internal/compiler` | 717 | 460 |
| `internal/graphview` | 1.610 | 817 |
| demais (autowire, canvas, ci, deeplink, pgtest, repo, staleness, turboquant, vaultfs) | ~3.551 | ~2.831 |
| **TOTAL** | **47.094** | **46.627** |

**Teste ≈ produção (46.627 vs 47.094 — 99%)**: a massa de teste está nos
mesmos pacotes grandes (`cmd/mem`, `internal/mcp`, `internal/db`,
`internal/store`). Isso é bom para a qualidade e caro para a manutenção.

## 3. O que está morto (indicador, não veredito)

`deadcode -test ./...` (x/tools v0.52.0): **19 funções inalcançáveis**.

| Pacote | N | Funções |
|:---|---:|---|
| `internal/event_runtime` | 6 | `AuditSubscriber.Name`, `.EventTypes`, `.MaxAckPending`, `auditLogPathForTesting`, `Dispatcher.IsReady`, `Log.ReadRange` |
| `internal/embedder` | 3 | `DownloadProviderArtifacts`, `NewTokenizerFromDir`, `Tokenizer.Close` |
| `internal/mcp` | 2 | `NewDBNeighborsHandler`, `Server.StalenessDetector` |
| `internal/supervisor` | 2 | `AuditLogger.CurrentSize`, `.CurrentPath` |
| `internal/codeast` | 2 | `ValidKinds`, `IsCamelOrPascalSegment` |
| `internal/db` | 1 | `SearchTurboQuant` |
| `internal/federation` | 1 | `ReplaceDatabaseInPostgresURL` |
| `internal/graph` | 1 | `DefaultInspectorOptions` |
| **`internal/vaultfs`** | **1** | **`Resolver.Root`** — método que eu mesmo criei na spec 3 (2026-11-14) e **nada chama**. É o meu: está aqui, measured, sem desculpa. |

Ressalva (a mesma do handover): `deadcode` não vê `init()`, reflexão nem
chamada por interface — é **indicador, não veredito**.

**Pacotes de produção sem nenhum importador de produção: 4** — e nenhum é
"morto": `cmd/mem` e `cmd/mymemoryd` são os **entrypoints**; `internal/ci` e
`internal/pgtest` são **pacotes de teste** por natureza.

## 4. O que está prometido e não entregue

### 4.1 O bucket do §7(b) — GAP-6 a GAP-10 (5 itens)

| GAP | O que é | Estado medido na ADR-062 |
|:---|:---|:---|
| **GAP-6** | Passe semântico de PDF/imagem | ⚠️ só `.md` com FTS + TurboQuant |
| **GAP-7** | Captura automática de prompt/tool-call + consolidação em página legível | ⚠️ `mem watch`/`mem hook` indexam **arquivos**, não a sessão |
| **GAP-8** | Handoff tipado claim-once + perfil cross-project | ❌ não existe |
| **GAP-9** | Decaimento por tier + compactação + dedup + slots `_slots/` | ⚠️ só decaimento temporal (ADR-012) |
| **GAP-10** | Eval de qualidade de recuperação | ⚠️ só benchmark de throughput |

**GAP-1, GAP-2 e GAP-3 já foram fechados** por esta mesma rodada (specs
`fix-release-onnx-provider`, `fix-doctor-tag-edges`, `fix-doctor-unindexed-links`).
**GAP-4** (multi-linguagem) e **GAP-5** (`graph.json`) são do **lado do
repositório** e continuam **sem spec**.

### 4.2 ADRs

| Status declarado | N | Leitura |
|:---|---:|---|
| "Implementado" no rótulo | 5 | o único grupo em que se pode confiar no rótulo |
| "Accepted"/"Aceito" **sem** marcar implementado | 50 | **majoritariamente enganoso**: inclui ADR-001 (SQLite), ADR-002 (Go), ADR-013 (doctor)… que **estão** implementados. É dívida de **metadado**, não de entrega. |
| Obsoleto | 5 | corte 후보, mas cada um precisa de verificação individual (o supersede-not-delete é a regra do ADR-062) |
| Proposed | 2 | `ADR-042` (mymemoryd), `ADR-050` (threat model) |
| Deferred | 1 | `ADR-045` (ASR/Nemotron) |
| Outro | 1 | ADR-052 (slot reservado, sem arquivo) |

**O único caso documentado de "aceito e não implementado" é a ADR-058**
(renderizador vis-network vendorizado — o asset e o `go:embed` não existem),
registrada na ISSUE-020.

### 4.3 Specs

| Estado | N |
|:---|---:|
| Validadas com **PASS** | 37 |
| Validada com **PARTIAL** | 1 (`feat-code-ast`) |
| `validation.md` que é **plano, não relatório** (`# Validation Plan:`) | 5 |
| Planejadas (têm `spec.md`, sem `validation.md`) | 6 |
| Rascunho (só `spec.md`) | 2 |

As **5 com plano em vez de relatório**: `ai-tool-auto-wiring`,
`context-packager-and-subgraph-bundle`, `graph-path-discovery`,
`progressive-context-loading`, `staleness-detection-and-banners` — **nunca
foram verificadas por um Verifier**, embora o código exista.

As **6 planejadas**: `deferred-until-application`, `graph-view-vis-engine`,
`mymemoryd-supervisor`, `nemotron-asr-streaming`, `spike-onnxruntime-binding`,
`threat-model-owasp-controls`.

## 5. Duplicados

| Onde | Resultado |
|:---|:---|
| Repo (`.md`, fora de `skills/`) | **0** arquivos duplicados byte a byte (SHA-256) |
| Cofre central (108 documentos) | **0** grupos com `content_hash` repetido |

A base de conhecimento não tem conteúdo repetido — nem por accidento de cópia.

## 6. Correções ao handover (o snapshot de 2026-11-13 ficou velho)

| Afirmação do handover | Estado medido em 2026-11-14 |
|:---|:---|
| "174 funções inalcançáveis" | **19** (a versão da ferramenta ou o código mudaram; o número de hoje é 19) |
| "`internal/supervisor` (1.627 linhas) e `internal/event_runtime` (1.327 linhas) **sem nenhum importador de produção**" | **FALSO hoje.** `internal/supervisor` é importado por `cmd/mem/up.go`, `cmd/mem/down.go`, `cmd/mem/profiles.go` e `cmd/mymemoryd/main.go`; `internal/event_runtime` por `cmd/mem/events.go` e `internal/supervisor/{lifecycle,manager}.go`. O wiring veio de `feat(supervisor)` + `feat(cli): mem up/down/logs/profiles` — a afirmação era verdadeira quando escrita e envelheceu. |
| "296 arquivos de documentação / ~34.500 linhas / 43% do código" | **318 / 54.907 / 58,6%** — o repo cresceu 59% em documentação num dia, contra 19% em Go. |

## 7. O que este inventário **não** diz (limites honestos)

1. **"50 ADRs aceitos sem implementar"** não é 50 entregas faltando. É 50
   **rótulos desatualizados**. Podar ADRs por esse número seria destructive e
   errado.
2. **`deadcode` é indicador, não veredito**: não vê `init()`, reflexão nem
   interface. As 19 funções precisam de leitura humana antes de qualquer corte.
3. **Nenhuma medição de "o que é usado pelo dono"**: uso real exige o histórico
   de uso (o cofre tem `content_hash` e `mtime`, mas o repo não tem telemetria).
4. **A Documentation do vault central** (108 notas, 510 chunks) está em
   `G:\My Drive\central-memory` e **não** foi auditada aqui — o inventário cobre
   o repositório.
5. **Nada foi cortado, decidido ou escrito.** Este relatório é o insumo da
   fase de podar; a decisão de cada corte é do dono.

## 8. Como este inventário se liga nas fases seguintes (ADR-065)

| Fase | O que este inventário entrega |
|:---|:---|
| **1 — inventário** (esta) | §1–§6: o que existe, o que está morto, o que está prometido, o que é duplicado |
| **2 — podar o morto** | candidatos: 5 ADRs obsoletas · 19 funções inalcançáveis · 6 specs planejadas · 1 ADR (058) sem implementar · 5 relatórios-plano. **Cada corte com ID rastreável no `ISSUES.md`.** |
| **3 — simplificar a governança** | candidatos: 50 ADRs com rótulo de status desatualizado · o peso da camada spec-driven (50,6% da documentação) · a decisão sobre os 5 GAPs (item a item, com medição) |

---

**Reprodução**: as medições vêm de `deadcode -test ./...` (x/tools v0.52.0),
`validate_state.py` (o validador autoritativo do fluxo spec-driven), contagem
por SHA-256 dos `.md` do repo, e consulta ao `content_hash` do cofre central
(Postgres `repo_central`). Nenhum comando desta fase escreve no repositório.
