# Feature: prune-dead-code-2026-11

- **Spec ID**: `prune-dead-code-2026-11`
- **Created**: 2026-11-14
- **Owner**: Felipe Miiller
- **Source**: ADR-065 fase 2 (podar o morto) · inventário em [`.specs/INVENTARIO-2026-11-14.md`](../../INVENTARIO-2026-11-14.md)
- **Issue**: ISSUE-042 (cada corte com ID rastreável)

## Problem Statement

O inventário read-only da ADR-065 (fase 1) mediu **14 funções sem nenhuma
chamada** no repositório. Elas não custam runtime — custam **leitura**: quem
for maintainer tem de entender 14 funções que nada usa, e cada uma é uma
superfície de manutenção (e um lugar onde um bug pode morar sem ninguém
perceber).

A lista veio de **duas** medições, e a segunda é o que importa:

| Medida | Resultado |
|:---|:---|
| `deadcode -test` **sem** a tag `onnx` | 19 funções — inclui 2 **falso-positivos** (o provider só existe na build taggeada) |
| `deadcode -test` **com** a tag do release | 17 funções — inclui 3 que são **dispatch por interface** (o `deadcode` não vê) |
| **Prova de chamada** (busca de chamada real em `.go`, fora do arquivo de definição) | **14 funções sem chamada** |

E a prova de chamada não é frágil: o repo **não usa reflexão** (nenhum
`reflect.ValueOf` / `MethodByName`), e uma chamada através de interface
**aparece** como `.Metodo(` no código — foi assim que as 3 do `Subscriber`
foram salvas.

## Goals

- [ ] Remover as **14** funções provadamente sem chamada.
- [ ] Não mudar **nenhum** comportamento: contrato, dado gravado e saída do
      produto ficam idênticos.
- [ ] Registrar **cada corte** com o ID rastreável (ISSUE-042), o
      `arquivo:linha` e a prova.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| As **6 ADRs obsoletas** (949 linhas) | São o **registro das decisões revertidas** — a regra *supersede-not-delete* da ADR-062 as preserva de propósito. Cortá-las apaga a prova. |
| Os **5 `validation.md` que são plano** | Dívida de **verificação** (o código existe, o Verifier nunca rodou), não de corte. |
| As **6 specs planejadas** | Roadmap: a decisão é do dono (ADR-065 §7b), não uma poda. |
| As **3 funções do `Subscriber`** (`Name`, `EventTypes`, `MaxAckPending`) | **Vivas** — chamadas via a interface `Subscriber`. O `deadcode` não vê dispatch por interface. |
| `NewTokenizerFromDir`, `Tokenizer.Close` | **Vivas** — chamadas pelo provider, que só existe na build com a tag `onnx`. |
| O chunk por âncora, a camada de validade, a busca vetorial | São **frentes novas**, não poda. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Como provar que uma função está morta | **chamada real em `.go`**, com a tag do release e a prova de dispatch por interface | `deadcode` sozinho é indicador: erra nas duas pontas (build tag e interface) | y |
| O que acontece com o histórico | **o git preserva** | o corte é reversível com `git revert`; a ADR-042/ISSUE-042 registra o que era | y |
| API pública em `internal/` | pode ser removida | `internal/` não é importável de fora do repo — não há consumidor externo | y |
| Funções que são *wrapper* de conveniência | podem cair | `SearchTurboQuant` é o atalho; o caminho vivo é `SearchTurboQuantWithOptions` | y |

Open questions: none

## User Stories

### P1: O código morto sai, com prova

**User Story**: Como maintainer, quero que o que ninguém chama saia do repo,
com o motivo registrado, para que a leitura do código não pague por
funcionalidade que ninguém exercise.

**Acceptance Criteria**:

1. WHEN o podar é executado THEN as **14** funções listadas na ISSUE-042
   SHALL ter sumido do código.
2. WHEN o gate completo roda THEN ele SHALL passar **sem alteração de
   comportamento** (mesma contagem de testes verde, nenhum teste removido).
3. WHEN o gate taggeado (`-tags "sqlite_fts5 onnx"`) roda THEN ele SHALL
   passar — é a build do release, e é ela que revela os falso-positivos da
   tag.
4. WHEN o `deadcode -test -tags "sqlite_fts5 onnx"` roda depois do podar
   THEN ele SHALL reportar **3** funções (as do `Subscriber`, que só o
   `deadcode` não vê) e nenhuma outra.
5. EACH corte SHALL estar registrado na ISSUE-042 com `arquivo:linha`, o nome
   da função e a prova que a deixou morta.
6. WHEN o cofre central é medido depois do podar THEN o `mem doctor` SHALL
   continuar em **100/100**, com 0 link rompido e 0 chunk sem vetor real — o
   podar é de código do produto, não do vault.

**Independent Test**: `go run golang.org/x/tools/cmd/deadcode@latest -test -tags
"sqlite_fts5 onnx" ./...` reporta 3 funções; e `git show --stat` do commit de
poda mostra exatamente os 14 blocos removidos.

---

### P2: O corte é rastreável

**User Story**: Como dono, quero saber o que foi removido e por quê, para que
"podar" não seja um corte às cegas.

**Acceptance Criteria**:

7. A ISSUE-042 SHALL listar, por função removida: o `arquivo:linha`, o nome, e
   a prova (sem chamada em `.go`, sem dispatch por interface, sem reflexão).
8. O `STATE.md` SHALL registrar a fase 2 da ADR-065 como concluída, com o
   número de funções removidas e o que **não** foi podado (e por quê).

## Edge Cases

- IF uma função removida implementa uma interface THEN ela NÃO pode ser
  removida — foi o que pegou as 3 do `Subscriber` (a prova de chamada as
  salvou).
- IF a função só é chamada na build com a tag `onnx` THEN ela NÃO pode ser
  removida — o `deadcode` sem a tag erra (pegou `NewTokenizerFromDir` e
  `Tokenizer.Close`).
- IF o repo usa reflexão por nome de método THEN a prova por busca de texto
  não bastaria — verificado: **não há reflexão** no repo.
- IF a função é o atalho de conveniência de outra que é usada THEN remover o
  atalho é seguro (`SearchTurboQuant` → `SearchTurboQuantWithOptions`).
- IF a remoção quebrar um teste THEN o gate falha — e o gate é o critério.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| PDC-01 | AC1 — as 14 funções sumiram | os 9 arquivos de código |
| PDC-02 | AC2 — gate completo verde, comportamento idêntico | `go test -count=1 ./...` |
| PDC-03 | AC3 — gate taggeado verde | `go test -tags "sqlite_fts5 onnx" ./...` |
| PDC-04 | AC4 — o `deadcode` reporta só as 3 do `Subscriber` | `deadcode -test -tags ...` |
| PDC-05 | AC5 — cada corte registrado com ID | ISSUE-042 |
| PDC-06 | AC6 — o cofre continua 100/100 | `mem doctor --storage=postgres --repo repo_central` |
| PDC-07 | AC7 — a ISSUE-042 lista arquivo:linha e a prova | ISSUE-042 |
| PDC-08 | AC8 — o STATE registra a fase 2 e o que não foi podado | `.specs/STATE.md` |

**Coverage**: 8 total, 8 mapeados, 0 unmapped.

## Success Criteria

- [ ] As **14** funções sem chamada não estão mais no repo.
- [ ] `deadcode -test -tags "sqlite_fts5 onnx"` reporta **3** (as do `Subscriber`).
- [ ] Gate completo e gate taggeado verdes, sem teste removido.
- [ ] O Cofre Central continua em **100/100**.
- [ ] ISSUE-042 e `STATE.md` registram o que saiu e o que ficou.
