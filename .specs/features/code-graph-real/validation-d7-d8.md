# Validação D7 + D8 — code-graph-real

> Evidência da correção dos **dois defeitos restantes** da verificação
> independente (`verification.md` §D7 e §D8). Arquivo novo: nenhum outro
> arquivo de `.specs/**` foi tocado.
>
> - **D7** — a viewer apontava para a ADR **errada** (ADR-047, deferida) em vez da ADR-061.
> - **D8** — a precedência `env var > config` contradizia a documentação e era **silenciosa**
>   (causa do incidente do verificador com o Postgres real).

Repo: `D:\repository\my-memory` · branch `develop` · base `aadd791`.

---

## D7 — a viewer aponta para a ADR errada

**O que era:** as 4 referências da viewer que falam do code graph apontavam para a
**ADR-047** (a ADR deferida do tree-sitter, 0 de 3 gatilhos). O code graph é da **ADR-061**.

| Arquivo:linha | Antes | Depois |
|---|---|---|
| `viewer/src/routes/code.tsx:200` | `(ADR-047)` | `(ADR-061)` |
| `viewer/src/routes/code.tsx:321` | `(ADR-047)` | `(ADR-061)` |
| `viewer/src/lib/code-view.ts:30` | `(ADR-047)` | `(ADR-061)` |
| `viewer/src/lib/queries.ts:234` | `(ADR-047)` | `(ADR-061)` |

**Números (grep em `viewer/src/**`):**

- `ADR-047`: **4 → 0** ocorrências.
- `ADR-061`: **0 → 4** ocorrências.

**Prova de que a rota `/code` aponta para a ADR-061:** os dois hints da rota
(`code.tsx:200`, erro de MCP na busca de símbolos; `code.tsx:321`, erro de MCP
nos vizinhos) citam `ADR-061` — nenhum texto da rota cita mais a ADR-047.

**Validação:** `npm run typecheck` OK · `npm run test` OK (5 arquivos, **95 testes**) ·
`npm run build` OK.

**O que NÃO mudou:** o corpo da `docs/adr/047-*.md` é imutável — o deferral do
tree-sitter segue intacto (0 de 3 gatilhos). Nenhuma ADR foi editada.

Commit: `e64e389`.

---

## D8 — precedência env > config: documentação e visibilidade

**Decisão aplicada (do gestor, não reaberta):** a **precedência real não muda**
(`MY_MEMORY_PG_URL` continua vencendo o `storage.postgres_url` do config global —
inverter quebraria o setup em produção). O que muda é (1) a documentação, alinhada
numa frase canônica, e (2) a **visibilidade** da fonte que vence.

### Frase canônica

> **Precedência de storage (ADR-040):** flag CLI > variável de ambiente > config
> (global ou local) > SQLite auto-scope; a fonte que vence é registrada em log
> (`storage: <engine> (fonte: <origem>)`).

Onde ela aparece (grep): `AGENTS.md:47` · `docs/CLI_GUIDE.md:1251`, `:1410`, `:1432` ·
`docs/adr/README.md:236` → **3 documentos, 5 ocorrências, uma única redação**.

A contradição sumiu: a linha `AGENTS.md:47` que dizia `(global > local > env)` foi
trocada. As ocorrências restantes de "global > local > env" no repo são **citações
históricas** (`.specs/ISSUES.md`, `.specs/features/code-graph-real/verification.md`,
a nota da ADR-040 e o comentário do código que explica a correção) — nenhuma delas
é uma declaração de precedência.

A skill canônica `skills/mymemory-config/SKILL.md:11-20` **já** declarava
`flag > env > config` corretamente: não precisou de alteração.

### O log da fonte vencedora

`cmd/mem/storage_source.go` (novo) resolve a URL do Postgres **e a fonte que
venceu**; `cmd/mem/main.go` (`resolveStorageAndRepo`) registra; o override
(`cmd/mem/storage_override.go`) também é visível. O log vai para o **stderr**
(o `log` do Go): o stdout do `--json` continua puro. **Sem URL e sem credencial.**

### Prova empírica (binário compilado, scratch do subagente)

Postgres de teste = porta fechada em `127.0.0.1:1` — **o Postgres real do dono não
foi tocado** e nenhum valor de URL foi impresso.

| Cenário | Resultado observado |
|---|---|
| **A** — env var **presente**, sem flag | stderr: `storage: postgres (fonte: MY_MEMORY_PG_URL)` |
| **B** — env var **ausente** | stderr: `storage: postgres (fonte: config global)` |
| **C** — env var presente + `--storage=sqlite --db=<tmp>` | stderr: `storage: postgres (fonte: MY_MEMORY_PG_URL)` → `storage: sqlite (fonte: flag --storage=sqlite)` → `ℹ  Postgres detectado no ambiente/config foi ignorado nesta sessão`; **exit 0**; SQLite criado (**212.992 bytes**); **1** `.md` no cofre central; `mem search` no SQLite indicado **encontrou** a lição |
| **D** — `mem search --storage=sqlite --db=<tmp>` | stdout contém o título da lição (prova de que ela foi indexada no SQLite, não no Postgres) |
| **E** — `mem tags --json --storage=sqlite --db=<tmp>` | **stdout começa com `{` e faz parse como JSON**; as 2 linhas de storage foram para o **stderr** → stdout puro |

O cenário **C** é a prova de isolamento: com a env var apontando para um Postgres
de porta fechada, o comando **terminou com exit 0** — logo, nenhuma conexão
Postgres foi tentada.

### Regressão automatizada

`cmd/mem/lesson_test.go` → **`TestMemLessonStorageFlagVenceEnvVar`**: com
`MY_MEMORY_PG_URL` no ambiente apontando para uma porta fechada,
`mem lesson --storage=sqlite --db=<arquivo>` tem que gravar no SQLite indicado.
Se a flag deixar de vencer a env var, o comando falha ao conectar e o teste falha
junto.

### Gate

`go build ./...` OK · `go vet ./...` OK ·
`go test -count=1 ./cmd/mem/... ./internal/config/... ./internal/mcp/...` OK
(`cmd/mem` 80,1s · `internal/config` 0,8s · `internal/mcp` 4,5s).

`gofmt`: os arquivos tocados foram copiados para LF e `gofmt -l` não apontou
nenhum deles.

---

## O que NÃO foi feito (e por quê)

- **O corpo de `docs/adr/**` não foi tocado** — a ADR-040 é imutável; a
  clarificação da precedência foi para `docs/adr/README.md` (o mesmo mecanismo já
  usado na correção da ADR-054).
- **A precedência do código não foi invertida** — decisão do gestor: o
  `MY_MEMORY_PG_URL` é o override explícito por invocação e é como o ambiente do
  dono está configurado.
- **`mem code-index` / `mem code-graph` não foram mexidos** — eles resolvem o
  engine pelo **config** (`storage.engine`), não pela variável de ambiente: o
  code stage não pode ser redirecionado pelo ambiente. A flag `--storage` continua
  vencendo tudo. A divergência está declarada em `docs/CLI_GUIDE.md`.
- **Nenhum push, branch ou release** — commits locais em `develop`.
