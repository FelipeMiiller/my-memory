# Regra Obrigatória: Quality Gate de Fim de Tarefa

> **Propósito**: garantir que cada entrega (spec, task, feature, fix) seja avaliada de **início a fim** antes de ser declarada pronta. Nenhuma entrega sai sem ter passado por este gate. Erros encontrados são **corrigidos imediatamente** (não acumulados pra resolver depois) ou, quando a decisão é não-trivial, viram ADR.

Esta regra complementa `always-test.md` e `always-update-docs-and-specs-on-pr.md`. Aplica-se em **todo fim de tarefa**, não apenas em PR/push. Quando o usuário diz "terminei", "finalizei", "completou?" ou simplesmente passa para a próxima atividade sem pedir commit, o agente DEVE ter rodado este gate silenciosamente.

---

## 🎯 Gatilhos (quando aplicar)

- Após `tlc-spec-driven` Execute declarar uma task concluída.
- Após qualquer mudança de código em `internal/`, `cmd/`, `.memory/`, scripts ou viewers (`graph-v2.html`, etc).
- Antes de qualquer `git commit` final.
- Antes de declarar "pronto" / "terminado" para o usuário.

---

## ✅ Checklist do Quality Gate

### 1. Testes unitários (cobre `always-test.md`)

```powershell
go test -count=1 ./...
```

- **Suíte completa, não só o pacote modificado.** Erros em outros pacotes significam regressão.
- Se algum teste falhar: **ler o erro, corrigir o código-fonte, re-rodar**. Não comentar `t.Skip` nem marcar como flaky.

### 2. Build do binário

```powershell
go build -v -o bin/mem.exe ./cmd/mem
```

- Build deve sair com exit 0 e sem warnings de compilação novos.
- Se o build falhar: erro de tipo/import/etc → corrigir antes de prosseguir.

### 3. Formatação Go

```powershell
gofmt -l .
```

- Deve retornar **vazio**. Se listar arquivos: `gofmt -w .` e commitar a correção.

### 4. Benchmarks — verificar e completar

```powershell
go test -bench=. -benchmem -run=^$ ./internal/...
```

- **Pergunta crítica**: o pacote modificado tem benchmark?
  - **Não tem** → adicionar benchmark para a lógica crítica do pacote (hot path).
  - **Tem** → rodar e registrar baseline. Se houver regressão > 2x vs baseline anterior, investigar.
- Pacotes **obrigatoriamente** com benchmark: `turboquant`, `parser`, `graph`, `store`, `graphview`, `embedder`, `db`, `config`.
- Padrão de benchmark:

  ```go
  func BenchmarkXxx_SmallInput(t *testing.B) {
      input := setupSmallInput()
      t.ResetTimer()
      for i := 0; i < t.N; i++ {
          _ = Xxx(input)
      }
  }
  ```

### 5. Smoke test do binário

Para mudanças em comandos CLI:

```powershell
.\bin\mem.exe graph --db '<vault-path>\memory.db' --repo <slug> --out '<out>.html' --open=false
```

- Verificar: exit 0, arquivo escrito, stats coerentes (Nós/Edges/Hubs > 0), encoding UTF-8 correto (sem mojibake tipo `MemÃ³ria`).

### 6. Validação visual (mudanças em viewers)

Para mudanças em `graph-v2.html` ou templates:

```powershell
python -m http.server 8765 --bind 127.0.0.1 -d <repo>
npx --yes playwright@1.63.0 screenshot --browser chromium --viewport-size 1600,1000 --wait-for-timeout <ms> "http://127.0.0.1:8765/graph-v2.html" "<out>.png"
```

- Inspecionar o screenshot via `Read` tool.
- Conferir: sem JS error overlay, nós visíveis, hubs rotulados, layout legível, stats populadas.
- Se algo visual estiver ruim → corrigir CSS/layout (text-background, fCoSE, hide-tags etc.) e re-screenshot.

### 7. Pipeline / CI (se aplicável)

- Se o projeto tem CI (GitHub Actions, etc), verificar última execução antes do commit:
  ```powershell
  gh run list --limit 1 --json status,conclusion,name,headBranch
  ```
- Se houver erro em CI relacionado às mudanças: priorizar correção **antes** de merge.

### 8. Estado git

```powershell
git status --short
```

- Tudo que vai ser commitado deve estar **intencionalmente** staged.
- Untracked de smoke test (`screenshot-*.png`, `graph-smoke-*.html`, `debug-*.cjs`, `test-*.cjs`) não vão pro commit. Se forem arquivos de longa duração, mover pra `.gitignore`.
- **Não acumular untracked**: limpar artefatos de teste depois de validar.

### 9. Specs e ADRs (cobre `always-update-docs-and-specs-on-pr.md`)

- Specs em `.specs/features/<feature>/` atualizadas com `verified` em todas as tarefas.
- ADR novo (se aplicável) criado em `docs/adr/NNN-...md` no padrão MADR.
- README raiz e `docs/README.md` refletem o estado real do código.
- `mem drift --strict` retorna 0 uncovered code.

### 10. Cross-reference com backlog

- Se a entrega corresponde a um item do ADR-036 (backlog post-release v1.3.0): marcar mentalmente como done, mover pro log de sessão, ou referenciar no commit message.

---

## 🚨 Política de Erros

### Encontrou erro de TESTE → corrigir imediatamente

Exemplos:
- `FAIL: TestXxx` → ler stack trace, corrigir, re-rodar.
- Build error → corrigir import/tipo/etc.
- Moijibake no HTML → checar template/charset.
- JS error no viewer → `console.error` no Playwright, corrigir.

**Não fazer**: comentar teste, marcar `t.Skip`, "deixar pra depois", abrir issue.

### Erro é decisão arquitetural não-trivial → ADR

Quando o erro expõe uma decisão de design:
- Criar ADR em `docs/adr/NNN-<decision>.md` (MADR).
- Conter: contexto, opções consideradas, decisão, consequências.
- Commitar ADR junto com o fix quando fizer sentido.

### Erro é cosmético trivial → fix direto, sem ADR

Para coisas pequenas (formatação, typo, label mal nomeado), corrigir direto sem ADR. **ADR não é overkill pra coisas simples.**

---

## 🎯 Critério de Aceitação Final

A entrega só é declarada "pronta" quando:

1. ✅ `go test -count=1 ./...` passa 100%
2. ✅ `go build -v -o bin/mem.exe ./cmd/mem` exit 0
3. ✅ `gofmt -l .` retorna vazio
4. ✅ Benchmarks rodados, sem regressões > 2x
5. ✅ Smoke test do binário exit 0 com output coerente
6. ✅ Visual do viewer (se aplicável) confirmado via Playwright screenshot
7. ✅ CI verde (se aplicável)
8. ✅ `git status` sem untracked de teste
9. ✅ Docs/specs/ADRs atualizados conforme `always-update-docs-and-specs-on-pr.md`
10. ✅ Cross-ref com ADR-036 (se aplicável)

Se **qualquer item** falhar: **parar**, corrigir, re-rodar este gate do início.

---

## 🔗 Integração com Skills

- **`tlc-spec-driven`** → esta regra é invocada automaticamente ao final da fase Execute.
- **`create-adr`** → usada quando o erro expõe decisão arquitetural (não pra fix trivial).
- **`post-task-validation`** → complementar para apps Next.js/TypeScript; este gate é a versão Go.

---

## 📌 Histórico

- **2026-09-17**: criada. Complementa `always-test.md` (que cobre só test+build+MCP) e `always-update-docs-and-specs-on-pr.md` (que cobre só docs). Adiciona benchmarks, visual, pipeline, política explícita de erros.

---

← [[AGENTS]]