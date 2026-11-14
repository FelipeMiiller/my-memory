# ADR-065: O caminho de limpeza do repo é inventário → podar → simplificar

- **Date**: 2026-11-14
- **Status**: Accepted — **primeiro passo pendente** (o inventário read-only)
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: governanca, limpeza, backlog, dead-code, adr-036, adr-062
- **Adenda**: [ADR-036](036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md) (backlog pós-release) · [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) (a matriz de gaps)
- **Origem**: §7 do [HANDOVER](../HANDOVER.md), respondida pelo dono em 2026-11-14

## Context and Problem Statement

O repositório tem **296 arquivos de documentação** (~34.500 linhas: 62 ADRs,
160 specs, 74 docs) contra **383 arquivos Go** (78.942 linhas) — **43% do
"código"**. Há mais ~3.000 linhas de código morto. A pergunta do dono: *deveria
apagar ADRs e specs e recomeçar?*

A medição feita antes de perguntar (e que sustenta esta decisão):

| Medida | Valor | Leitura |
| :--- | :--- | :--- |
| Arquivos de documentação | 296 (~34.500 linhas) | 43% das linhas do repo |
| ADRs | 62 | nem todos implementados — alguns aceitos e não construídos |
| Specs | 160 | boa parte já executada, mas o histórico fica no repo |
| Funções inalcançáveis | 174 (`deadcode` com testes) | indicador, **não** veredito: ele não vê `init()`, reflexão nem interfaces |
| `internal/supervisor` | 1.627 linhas, **sem importador de produção** | código de um runtime que não está ligado |
| `internal/event_runtime` | 1.327 linhas, **sem importador de produção** | idem |
| `internal/embedder/ollama.go` + `tokenizer.go` | 178 linhas | substituídos pelo provider embutido (ADR-035) |

E o segundo bucket: a ADR-062 promete GAP-6 a GAP-10 — **passe semântico de
PDF, captura de sessão, handoff tipado, perfil cross-project, decaimento por
tier, `graph.json` portátil, eval de recuperação** — e nada disso foi
entregue. É o "prometido mas não entregue" que o repo carrega.

## Decision Drivers

- **DR-1 — Apagar prova não apaga gap.** As ADRs e as specs são a **prova**
  de que um gap foi medido e decidido. Sem eles, o próximo ciclo repete a
  medição do zero.
- **DR-2 — É reversível.** O `git log -- docs/adr/` preserva tudo. Podar é
  barato de desfazer; **zerar o HEAD** é irreversível na prática (o histórico
  deixa de ser navegável por quem não tem o clone).
- **DR-3 — O que incomoda é a promessa, não o arquivo.** Um ADR aceito e não
  implementado não é problema: é dívida **declarada**. O problema é a
  promessa sem registro de dívida.
- **DR-4 — A ordem importa.** Podar antes de inventariar apaga por
  palpite; simplificar a governança antes de podar cria dois problemas ao
  mesmo tempo.
- **DR-5 — Cortar por hype corta por idiomaticamente.** O bucket GAP-6..10
  tem itens de custo muito diferentes (um `graph.json` portátil custa um dia;
  um perfil cross-project custa uma arquitetura). A decisão tem de ser **por
  item**, com o custo medido na frente.

## Considered Options

### (a) Caminho de limpeza

- **A — Inventário → podar o morto → simplificar a governança** (escolhido)
- **B — Inventário → zerar o HEAD** (ADRs e specs só no histórico do git)
- **C — Só inventário**

### (b) Bucket "prometido mas não entregue" (GAP-6 a GAP-10)

- **a — Decidir item a item, com medição antes de cada corte** (escolhido)
- **b — Poda agressiva**
- **c — Corte pelo uso diário**

## Decision Outcome

Escolhidas a **Opção A** no caminho de limpeza e a **Opção a** no bucket.

### Caminho de limpeza

**A ordem é a decisão**:

1. **Inventário (read-only)** — medir o que existe: o que é usado, o que é
   morto, o que está prometido e não entregue, o que é duplicado. Nenhuma
   escrita, nenhuma decisão, nenhum corte.
2. **Podar o morto** — cortar o que o inventário provou morto (código sem
   importador, ADR superseded de fato, spec abandonada com o motivo
   registrado). Cada corte com o ID rastreável do que foi removido.
3. **Simplificar a governança** — só depois do inventário, decidir o que
   muda na forma: o que uma ADR precisa ter, o que uma spec precisa ter, e
   o que passa a ser um issue.

A opção **B** (zerar o HEAD) fica **descartada**: ela resolve a métrica de
governança sacrificando a trilha de decisão — e o dono já tem o histórico no
git, o que torna o ganho duplo e o custo também.

### Bucket GAP-6 a GAP-10

Cada item é decidido **separadamente**, e a decisão exige a **medição do custo
antes do corte**: quanto custa manter, quanto custa entregar, e o que o dono
ganha em cada caso. Nada de poda em bloco, nada de corte por hype.

### Positive Consequences

- O repositório para de acumular promessa sem registro, **sem** apagar a
  memória das decisões.
- A ordem protege do erro caro: nada é cortado antes de ser medido.
- O bucket GAP-6..10 deixa de ser uma lista indefinida e vira uma fila de
  decisões com custo conhecido.

### Negative Consequences

- **É mais lento.** O inventário é trabalho sem resultado visível imediato, e
  ele pode revelar que o repo está menos inchado do que a métrica sugere —
  o que é um resultado, mas um resultado que não "diminui" nada.
- A fase de podar exige **IDs rastreáveis** para cada corte (o tracker
  `.specs/ISSUES.md` é o lugar natural), o que é mais trabalho que um
  `rm` — e é o ponto.
- A fase 3 (simplificar a governança) pode exigir mexer em `AGENTS.md`, nos
  templates de spec/ADR e nos validadores: mudanças que afetam todo
  agente que trabalha no repo.

## Pros and Cons of the Options

### Opção A ✅ Chosen (caminho)

- ✅ Preserva a trilha de decisão e ainda limpa.
- ✅ O inventário é reversível por construção (read-only).
- ❌ São três frentes, não uma.

### Opção B — zerar o HEAD

- ✅ Métrica de governança imediata.
- ❌ Apaga a prova do gap (DR-1) e o histórico deixa de ser navegável por
  quem não tem o clone (DR-2).

### Opção C — só inventário

- ✅ Zero risco.
- ❌ Não resolve nada: o repo segue prometendo o que não entrega (DR-3).

### Opção a ✅ Chosen (bucket)

- ✅ Cada corte Justificável pelo custo medido.
- ✅ Preserva o que é barato e descarta o que é caro — na medida certa.
- ❌ Exige disciplina: item por item é mais lento que decidir em bloco.

### Opção b — poda agressiva

- ✅ Rápida.
- ❌ Apaga capacidade planejada com o mesmo critério do que já está morto —
  e capacidade planejada **não** é dívida: é roadmap (DR-3).

### Opção c — corte pelo uso diário

- ✅ Alinha o produto ao hábito.
- ❌ O "uso diário" do dono não é o uso de um agente: cortar por ele apaga
  justamente o que a memória do produto existe para dar.

## Links

- [ADR-036](036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md) — o backlog
  pós-release que a fase 3 vai simplificar.
- [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) — a
  matriz de gaps (GAP-1 a GAP-10) e os padrões de `graphify`/`ai-memory`.
- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — o escopo do produto,
  que a fase 3 precisa respeitar.
- HANDOVER §7 — a pergunta original e as três opções de caminho.
- `.specs/ISSUES.md` — o tracker onde cada corte da fase 2 ganha ID.
