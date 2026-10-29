# Feature: repo-id-alignment

- **Spec ID**: `repo-id-alignment`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-028 — `.specs/ISSUES.md:945`
- **Related**: ADR-033 (identidade federada e catálogo global) · ADR-040 (precedência de storage) · ADR-016 (auto-scoping de vault)

## Problem Statement

O `id` do catálogo global (`~/.memory/config.yaml` → `repositories[].id`) e o `repo_id` do repositório (`<repo>/.memory/config.yaml` → `repo_id`) são **dois identificadores diferentes para a mesma coisa**, e nada no produto verify um contra o outro.

O achado original (`.specs/ISSUES.md:959-962`) era o catálogo com `repo_271444ab0133` contra um vault com `repo_e4c8f3b1a2d5`. **A Parte A (dado) já foi feita pelo dono**: o catálogo global está hoje com `id: repo_e4c8f3b1a2d5`, igual ao `repo_id` do vault deste repositório. O que continua errado é o **código**: o `mem repos` lista as entradas do catálogo sem conferir nada, e o cascateamento de config casa a entrada do catálogo por **ID ou caminho** — de modo que uma divergência futura passa **em silêncio**.

### Pergunta 1 — Qual é exatamente o mecanismo de detecção (ou, mais precisamente, de *não*-detecção) hoje?

O mecanismo é a **Camada 2.5** do cascateamento, `internal/config/config.go:690-712`:

```go
for _, repoEntry := range globalCfg.Repositories {
    entryAbs, _ := filepath.Abs(ExpandPath(repoEntry.Path))
    matchesID := localRepoID != "" && repoEntry.ID == localRepoID   // config.go:695
    matchesPath := absDir != "" && entryAbs == absDir               // config.go:696
    if matchesID || matchesPath {                                  // config.go:697
        // a entrada sobrepõe SOMENTE o storage (engine, postgres_url, sqlite_path)
        break
    }
}
```

Três fatos seguem da leitura do código:

1. `localRepoID` é o `repo_id` lido da **camada 3** — o `.memory/config.yaml` do próprio repositório (`internal/config/config.go:646-673`). É ele que a entrada do catálogo casa via `matchesID`.
2. A entrada do catálogo, quando casa, **só sobrepõe o storage** (`internal/config/config.go:698-708`). Ela **nunca** escreve o `repo_id` — o `repo_id` é sempre o do disco, e só o disco.
3. **Não existe nenhum ponto do código onde o `id` do catálogo seja comparado com o `repo_id` do disco para gerar um erro.** O único lugar onde os dois coexistem é a comparação de `matchesID` — e ali a divergência é *inofensiva*, porque `matchesPath` cobre o caso. A divergência é, portanto, **indetectável pelo código**: ela só muda o resultado *quando o path também muda*.

**O que muda com o id alinhado:** com `id == repo_id`, `matchesID` passa a ser verdadeiro e o casamento deixa de depender do path. O `repo_id` é persistido no `.memory/config.yaml`, que **move junto com o repositório** (`.specs/ISSUES.md:987-989`); o `path` do catálogo é a única parte do registro que é volátil. Alinhar o id transforma o `path` de *chave de casamento* em *metadado*.

### Pergunta 2 — O que acontece hoje se o repositório for movido ou clonado?

**Comportamento atual, observável** (ids divergentes, como estava o catálogo antes da Parte A):

- O repositório é movido de `C:\repository\my-memory` para `D:\repository\my-memory` e o `path` do catálogo **não** é atualizado: `matchesPath` fica falso e, com os ids divergentes, `matchesID` também → a entrada do catálogo **não casa**.
- Consequência: o override de storage por repositório (engine + URL do Postgres) é **perdido em silêncio** (`.specs/ISSUES.md:970-974`). O repositório volta ao storage global (camada 2, `internal/config/config.go:631-643`) e **nenhum comando avisa** — o `mem repos` lista a entrada com um `✅` porque o único critério dele é "o diretório `.memory` existe" (`cmd/mem/setup.go:293-304`).
- Se o mesmo repositório for registrado em **dois paths**, a entrada casa com o primeiro path e o `repo_id` passa a apontar para duas árvores de dados — split-brain de identidade (`.specs/ISSUES.md:975-977`).

**Comportamento novo, com esta feature:**

- Com o id **alinhado** (Parte A), um repositório movido **continua casando pelo id**: o override de storage sobrevive à mudança de pasta. Isso é garantido por teste de caracterização (`internal/config/config_test.go`, T2) e é o benefício concreto do alinhamento.
- Se o id voltar a divergir, `mem repos` **passa a Falhar alto**: a entrada sai com `❌`, o comando imprime o id do catálogo, o `repo_id` do disco, o path e a ação para alinhar, e **sai com código diferente de zero**. O silêncio da ISSUE-028 vira erro explícito.

### Pergunta 3 — A correção é só de diagnóstico, ou também corrige o catálogo?

**Decidido: diagnóstico apenas.** O código **detecta e acusa**; o alinhamento do arquivo `~/.memory/config.yaml` é uma **operação manual do dono**, guided pela instrução que o próprio erro entrega.

Justificativa (a auto-correção foi avaliada e rejeitada):

1. **A aprovação do dono é literalmente "alinhar" + "rejeitar"**, e o "alinhar" já foi executado na Parte A. Um `--fix` reescreveria o arquivo global do usuário — um arquivo **fora do repositório** e fora da propriedade desta execução.
2. **A auto-correção pode piorar o split-brain.** A divergência não tem uma direção única: quando o mesmo repositório está registrado em dois paths (item 2 do risco), o `repo_id` do disco é o mesmo nos dois, e "alinhar o catálogo" faz as duas entradas colidirem no mesmo `id` — o `repo_id` apontando para duas árvores de dados fica **mais** confuso, não menos. Um `--fix` que reescreve o id por via automática não tem como distinguir "catálogo atrasado" de "catálogo duplicado".
3. **Reescrever o catálogo global tem efeito colateral:** `RegisterRepositoryInGlobalConfig` (`internal/config/config.go:605-609`) substitui a entrada inteira e só preserva o bloco `storage:` quando a nova entrada vem sem `storage`. Uma reescrita mal feita perderia o override de Postgres do repositório — justamente o que a ISSUE-028 quer proteger.
4. Se um comando de auto-correção for considerado depois, o default teria de ser **dry-run + backup**, porque o arquivo global é do usuário. Fica como evolução futura, não como parte desta feature.

## Goals

- [ ] Trocar o silêncio por um **erro explícito**: `mem repos` rejeita uma entrada cujo `id` do catálogo não corresponde ao `repo_id` do vault no path registrado.
- [ ] Preservar integralmente os casos que **não** são divergência (vault sem inicializar, vault sem `repo_id` declarado): listados, sem erro.
- [ ] Manter a auditoria **estritamente read-only** (nada é escrito no catálogo global nem no vault).
- [ ] Manter o formato da saída e o código de saída zero quando não há divergência (mudança compatível).
- [ ] Entregar teste de regressão que **falha contra o defeito** (divergência listada como OK, exit 0) e passa com a correção.

## Out of Scope

- **A correção do arquivo `~/.memory/config.yaml`** — já executada pelo dono (Parte A) e, quando necessário, refazível manualmente seguindo a instrução do erro. Nenhum código desta feature escreve no catálogo global.
- **Comando de auto-correção (`mem repos --fix`)** — decisão registrada na Pergunta 3; reavaliar em feature própria, com dry-run e backup.
- **Derivar o `repo_id` do path** — o `repo_id` é persistido no vault e muda de pasta junto com ele (`.specs/ISSUES.md:987-989`).
- **Mudar a semântica do cascateamento** (quem ganha entre global/local/env) — é a ADR-040, documentada e fora de escopo.
- **`internal/pgtest`, `internal/store`, `internal/mcp` e o servidor MCP** — nada disso é tocado.
- **Atualização de `docs/`, `README.md`, `COMO_USAR.md`, `CHANGELOG.md`, `.specs/ISSUES.md` e `.specs/STATE.md`** — fora da propriedade desta execução; pendência registrada para o gestor.
- **ADR nova** — o comportamento de cascateamento não muda e a ADR-033/ADR-040 continuam válidas; a feature adiciona **diagnóstico** a um comando existente.
- **Postgres real** — a auditoria lê arquivo de config; nenhum teste toca banco.

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 — O que é divergência | `id` do catálogo ≠ `repo_id` declarado em `<path>/.memory/config.yaml` | É a única comparação possível: o `repo_id` do disco é a fonte da verdade (ADR-033) e o `id` do catálogo é o que casa com o repositório (`matchesID`, `internal/config/config.go:695`) | y |
| A2 — Escopo da leitura do vault | Somente o path registrado, **sem** subir para os diretórios pais | O `repo_id` procurado é o do repositório registrado; `FindConfigFile` (`internal/config/config.go:278-328`) sobe até a raiz do Git e poderia ler o vault de um repositório ancestral — leitura errada | y |
| A3 — Entrada sem vault | Não é divergência: listada com `⚠️` (vault não inicializado) | Não há `repo_id` para comparar; virar erro puniria o usuário por um registro que ele pode criar depois | y |
| A4 — Entrada com vault sem `repo_id` | Não é divergência: listada sem erro | Não há contra o que comparar; o `repo_id` é gerado por `EnsureRepoID` (`internal/config/config.go:467-478`), então só um vault escrito à mão fica assim | y |
| A5 — Marcador de "vault não inicializado" | `⚠️` quando **não há arquivo de configuração** no path (superset do gatilho atual, que era só "diretório `.memory` ausente") | O aviso precisa ser verdadeiro em todo caso de A3; o diretório `.memory` sem `config.yaml` é um vault não inicializado, e hoje ele aparece como `✅` | y |
| A6 — Divergência na listagem | `❌` na linha da tabela + bloco de erro explícito | Listar a entrada como OK (`✅`) é exatamente o defeito apontado na ISSUE-028 (`.specs/ISSUES.md:978-980`) | y |
| A7 — Código de saída | Diferente de zero **apenas** quando há divergência | O requisito do dono é "trocar o silêncio por um erro explícito" e a compatibilidade fora da divergência é explícita; nada mais muda no contrato do comando | y |
| A8 — Vault ilegível (config quebrado) | Aviso explícito na saída, **sem** mudar o código de saída | A auditoria é sobre divergência de identidade; um YAML inválido é outro problema, reportado alto mas fora do contrato de exit code definido em A7 | y |
| A9 — Escopo da correção | Diagnóstico + instrução; o arquivo do dono é corrigido à mão | Ver Pergunta 3: a auto-correção pode agravar o split-brain e tem efeito colateral no bloco `storage:` | y |
| A10 — Efecto colateral da correção | Nenhum arquivo é escrito (catálogo global e vault intocados) | A auditoria é de leitura; um comando de diagnóstico que escreve no arquivo de config do usuário seria uma surpresa | y |

**Open questions**: none — todas as decisões estão resolvidas e registradas acima (A1 a A10).

## User Stories

### P1: A divergência vira erro, não silêncio

**User Story**: Como pessoa que diagnostica a memória com `mem repos`, quero saber quando o `id` do catálogo não corresponde ao `repo_id` real do repositório, so that eu não confie num `✅` que não significa nada.

**Why P1**: A ISSUE-028 nasceu de um `id` enganoso no `mem repos` (`.specs/ISSUES.md:978-980`) e do override de storage perdido em silêncio (`.specs/ISSUES.md:970-974`).

**Acceptance Criteria**:
1. WHEN uma entrada do catálogo tem `id` diferente do `repo_id` declarado no `.memory/config.yaml` do path registrado THEN o comando SHALL reportar um erro explícito contendo o `id` do catálogo, o `repo_id` do disco, o path registrado e a ação para alinhar o catálogo.
2. WHEN uma entrada do catálogo tem `id` divergente do `repo_id` do disco THEN o comando SHALL NOT listar a entrada com o marcador de OK (`✅`) na tabela.
3. WHEN existe pelo menos uma entrada divergente THEN o comando SHALL sair com código de saída diferente de zero.
4. WHEN uma entrada do catálogo tem path sem arquivo de configuração de vault THEN o comando SHALL listar a entrada com o aviso de vault não inicializado (`⚠️`) e SHALL NOT reportar divergência.
5. WHEN o vault do path registrado existe mas não declara `repo_id` THEN o comando SHALL listar a entrada sem reportar divergência e sem falhar.
6. WHEN nenhuma entrada do catálogo tem `id` divergente do `repo_id` do disco THEN o comando SHALL imprimir a mesma tabela `REPO ID | NOME | CAMINHO NO DISCO` de hoje AND SHALL sair com código de saída zero.
7. WHEN a auditoria de identidade roda THEN o comando SHALL NOT escrever no catálogo global, SHALL NOT criar arquivo e SHALL NOT modificar o vault do path registrado.

**Independent Test**: Montar um catálogo global com uma entrada de `id` `repo_111122223333` apontando para um vault cujo `.memory/config.yaml` declara `repo_id: repo_e4c8f3b1a2d5` e rodar `mem repos`: a saída tem que conter os dois identificadores, o path e a instrução, a linha da tabela não pode trazer `✅`, e o código de saída tem que ser diferente de zero.

---

### P2: A identidade alinhada sobrevive à mudança de pasta

**User Story**: Como dono de um repositório que muda de pasta, quero que o override de storage do meu repositório continue valendo depois da mudança, so that eu não perca o Postgres isolado sem nenhum aviso.

**Why P2**: O `path` do catálogo é volátil (o repositório migrou de `C:\repository\my-memory` para `D:\repository\my-memory` — `.specs/ISSUES.md:949`); o `repo_id` é a identidade estável e é o que casa com o repositório (`internal/config/config.go:695`).

**Acceptance Criteria**:
1. WHEN o `id` do catálogo é igual ao `repo_id` declarado no vault THEN o comando SHALL listar a entrada com o marcador de alinhamento (`✅`).
2. WHEN o `id` do catálogo está alinhado com o `repo_id` do vault AND o path registrado do catálogo é diferente do path atual do repositório THEN o override de storage da entrada do catálogo SHALL continuar valendo (a identidade sobrevive à mudança de pasta).
3. WHEN o repositório é registrado pelo caminho oficial (`mem init` → `RegisterRepositoryInGlobalConfig`) THEN o `id` da entrada do catálogo SHALL ser o `repo_id` do vault AND o bloco `storage:` da entrada SHALL ser preservado.

**Independent Test**: Carregar o cascateamento de um vault cujo `repo_id` casa com o `id` de uma entrada de catálogo registrada em outro path e conferir que o `postgres_url` do catálogo foi aplicado; e registrar de novo o mesmo path com o `repo_id` do vault, conferindo que a entrada ficou com o id certo e com o `storage:` intacto.

---

### P3: O arquivo do dono é corrigido por ele

**User Story**: Como dono do `~/.memory/config.yaml`, quero que a ferramenta me diga exatamente o que alinhar, so that eu corrija o arquivo sem adivinhação e sem que a ferramenta reescreva meu config por conta própria.

**Why P3**: O catálogo global é a fonte de verdade de storage do usuário e a correção envolve o override de Postgres de cada repositório — mexer nele automaticamente é risco, não conveniência.

**Acceptance Criteria**:
1. WHEN a auditoria detecta divergência THEN o comando SHALL instruir o usuário a alinhar o `repositories[].id` do catálogo global com o `repo_id` do disco e SHALL NOT modificar o catálogo global.
2. WHEN a auditoria detecta divergência THEN o comando SHALL indicar que o `repo_id` do disco é o valor autoritativo e SHALL apontar o arquivo do vault que o declara.

**Independent Test**: Rodar `mem repos` contra o catálogo divergente e conferir que a saída nomeia o arquivo do catálogo global, o `repo_id` do disco e o arquivo `.memory/config.yaml` do vault, e que o conteúdo do catálogo global está byte-idêntico antes e depois.

## Edge Cases

- IF o path registrado não existe mais (repositório movido e apagado) THEN the system SHALL reportar a entrada como vault não inicializado (`⚠️`) e NOT reportar divergência — não há `repo_id` no disco para comparar.
- IF o `.memory/config.yaml` do path registrado existe mas está com YAML inválido THEN the system SHALL reportar um aviso explícito com o arquivo e o erro de leitura AND SHALL NOT tratar a entrada como divergência.
- IF o `.memory/config.yaml` do path registrado existe mas declara `repo_id: ""` THEN the system SHALL listar a entrada sem divergência (não há valor para comparar).
- IF o catálogo global tem **várias** entradas divergentes THEN the system SHALL reportar **todas** com os respectivos valores e SHALL sair com código diferente de zero uma única vez.
- IF o `path` do catálogo estiver escrito com `~` ou variável de ambiente THEN the system SHALL expandi-lo antes de ler o vault, como o cascateamento faz (`internal/config/config.go:694`).
- IF o `repo_id` do disco for igual ao `id` do catálogo THEN the system SHALL NOT reportar nada — o caminho saudável é silencioso, como antes.

## Requirement Traceability

| Requirement ID | AC | Story | Arquivo | Fase | Status |
| -------------- | -- | ----- | ------- | ---- | ------ |
| REPOID-01 | P1.1 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-02 | P1.2 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-03 | P1.3 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-04 | P1.4 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-05 | P1.5 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-06 | P1.6 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-07 | P1.7 | P1: A divergência vira erro, não silêncio | cmd/mem/setup.go | Execute | Verified |
| REPOID-08 | P2.1 | P2: A identidade alinhada sobrevive à mudança de pasta | cmd/mem/setup.go | Execute | Verified |
| REPOID-09 | P2.2 | P2: A identidade alinhada sobrevive à mudança de pasta | internal/config/config_test.go | Execute | Verified |
| REPOID-10 | P2.3 | P2: A identidade alinhada sobrevive à mudança de pasta | internal/config/config_test.go | Execute | Verified |
| REPOID-11 | P3.1 | P3: O arquivo do dono é corrigido por ele | cmd/mem/setup.go | Execute | Verified |
| REPOID-12 | P3.2 | P3: O arquivo do dono é corrigido por ele | cmd/mem/setup.go | Execute | Verified |

**Coverage:** 12 total, 12 mapeados para tarefas, 0 não mapeados.

## Success Criteria

- [x] `mem repos` rejeita um `id` de catálogo divergente do `repo_id` do disco com erro explícito (dois valores + path + instrução) e código de saída diferente de zero.
- [x] Entradas sem vault e com vault sem `repo_id` continuam listadas sem erro.
- [x] A auditoria não escreve nada (catálogo global e vault byte-idênticos antes e depois).
- [x] Sem divergência, a saída e o código de saída continuam como antes.
- [x] Teste de regressão que falha contra o defeito revertido.
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0.
