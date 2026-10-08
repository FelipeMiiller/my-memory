# Validation — embedder-builtin-onnx-minilm / T1 + T2

**Data:** 2026-10-08 · **Branch:** `develop` · **Executado por:** Mavis (author) · **Spec:** [`spec.md`](spec.md)

## Verdict

| Task | Verdict | Resumo |
|---|---|---|
| **T1** — Spike de binding ONNX Runtime | ⚠️ **BLOQUEADO — precisa de toolchain** | Não é PIVOT de design; é pré-requisito ausente na máquina |
| **T2** — Auditoria de manutenção do binding | ✅ **PASS** | Binding é ativamente mantido, não está abandonado |

## T1 — Evidência por AC

### AC-1: `go build` compila em Windows amd64

**❌ NÃO SATISFEITO (bloqueio de ambiente, não de compatibilidade)**

Dois erros, em sequência — o segundo só aparece depois de forçar CGO:

```
# com CGO_ENABLED=0 (estado atual)
go build -tags onnx ./...
→ github.com/yalue/onnxruntime_go: build constraints exclude all Go files in
  ...\onnxruntime_go@v1.36.0

# com CGO_ENABLED=1
$env:CGO_ENABLED='1'; go build -tags onnx ./...
→ # runtime/cgo
  cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%
```

### Estado da máquina no momento do spike

| Item | Valor |
|---|---|
| `go env CGO_ENABLED` | `0` |
| `gcc` no PATH | ❌ não encontrado |
| `clang` no PATH | ❌ não encontrado |
| `go env CC` | `gcc` |
| `onnxruntime.dll` na máquina | ✅ existe (`Program Files\Mozilla Firefox`, `Microsoft Office\WinAppSDK`) — **versão provavelmente incompatível** |

### Por que isso NÃO é PIVOT

O binding declara `//go:build windows` e faz `import "C"` — ele **exige CGO por
construção**. Não há tag alternativa que evite isso. É um requisito do binding,
não uma incompatibilidade com o my-memory.

Faltam **dois pré-requisitos**:
1. **Toolchain C** — `gcc` via MinGW-w64 ou MSYS2 (`winget install MSYS2.MSYS2` ou `BrechtSanders.WinLibs.POSIX.UCRT`)
2. **ONNX Runtime 1.29.0 DLL** — o README do binding pinna a versão:
   > *"this library uses version 1.29.0 of the onnxruntime C API headers. So, it will probably only work with version 1.29.0"*

   Os DLLs que já existem na máquina vêm junto de Firefox e Office e
   **certamente não são 1.29.0**. Esse seria o próximo erro depois do gcc.

### AC-2: diagnóstico nomeando arquivo esperado e origem

**⚠️ PARCIAL** — o erro do Go é claro (`gcc not found`), mas o binding exige
também `ort.SetSharedLibraryPath(...)` para a DLL. Essa parte não pôde ser
exercitada sem CGO.

### AC-3: inferência real retorna vetor com a dimensão do modelo

**⛔ NÃO AVALIÁVEL** — bloqueado por AC-1.

### AC-4: verdict T4 GO ou PIVOT

**⚠️ Verdict: `BLOCKED — GO condicional`.**

Não é GO (não compilou) nem PIVOT (não há evidência de que o binding é
incompatível). O binding **parece viável**, mas a pergunta "builda em Windows?"
**não pode ser respondida sem antes instalar um compilador C**.

## T2 — Evidência (supply chain)

**✅ PASS. O binding NÃO está abandonado.**

| Métrica | Valor |
|---|---|
| Versões publicadas | **37** (`v1.0.0` … `v1.36.0`) |
| Última release | `v1.36.0`, **2026-09-04** (4 dias atrás) |
| Cadência | Ativa e recente |
| Licença | permissive (wrapper de API pública do ONNX Runtime, Microsoft) |

**Conclusão T2:** o critério de PIVOT por abandono **não se aplica**. O binding
passa a auditoria.

## Diff range

Nenhum commit — o spike rodou fora da árvore (módulo descartável em `%TEMP%/onnxprobe`).
`go.mod` do my-memory **não foi tocado**: o `go get` aconteceu no módulo de prova.

## Decisão pendente

Instalar toolchain C é software de sistema e exige aprovação explícita. Enquanto
não houver `gcc`, **T1 não fecha** e a feature fica bloqueada na Fase 1.

Sequência se aprovado:
1. Instalar MinGW-w64/MSYS2 e expor `gcc` no PATH
2. Baixar ONNX Runtime 1.29.0 e apontar via `ort.SetSharedLibraryPath`
3. Reexecutar o smoke test → verdict GO ou PIVOT definitivo

## SPEC_DEVIATION

Nenhuma. O spike foi executado exatamente como a spec define (T1 AC-1..4).

**Nota de qualidade:** o spec não previa o segundo pré-requisito (DLL pinada em
1.29.0). Ele só apareceu ao ler o README do binding. Vale registrar na próxima
revisão da spec como AC adicional em T1.