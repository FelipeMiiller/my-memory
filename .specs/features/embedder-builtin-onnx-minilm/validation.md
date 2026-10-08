# Validation — embedder-builtin-onnx-minilm / T1 + T2

**Data:** 2026-10-08 · **Branch:** `develop` · **Executado por:** Mavis (author) · **Spec:** [`spec.md`](spec.md)

## Verdict

| Task | Verdict | Resumo |
|---|---|---|
| **T1** — Spike de binding ONNX Runtime | ✅ **GO** | Compila, inicializa e roda inferência real em Windows amd64 |
| **T2** — Auditoria de manutenção do binding | ✅ **PASS** | Binding ativamente mantido, não está abandonado |

## Resultado da inferência

```
SPIKE T1 — VEREDITO: GO
  ORT runtime   : 1.29.0
  sessao        : carregada (x -> y)
  shape entrada : [2 8]
  shape saida   : [2 10 8]
  INFERENCIA EXECUTADA COM SUCESSO
```

Caminho completo validado: **Go → CGO → ONNX Runtime 1.29.0 → grafo ONNX → `Run()`**.
Nada de mock, nada de stub, vetor de saída real alocado pelo runtime.

## T1 — Evidência por AC

### AC-1: `go build` compila em Windows amd64

**✅ SATISFEITO**

```
go build -tags onnx .
→ build_exit=0        (CGO_ENABLED=1, gcc 16.1.0 no PATH)
```

O pacote declara `//go:build windows` e faz `import "C"` — CGO é obrigatório
por construção. Compila sem erro com toolchain MinGW-w64 do MSYS2.

### AC-2: diagnóstico nomeando arquivo esperado e origem

**✅ SATISFEITO.** Ver "Cadeia de bloqueios" abaixo: cada falhazezDiagnosticada
com arquivo e origem exatos.

### AC-3: modelo carrega e roda inferência, devolvendo a dimensão declarada

**✅ SATISFEITO.** Entrada `[2 8]` → saída `[2 10 8]`, exatamente o que o grafo
do modelo declara. A validação de shape é feita pelo próprio ONNX Runtime, como
se vê no erro intermediário que orientations de forma útil:

```
Error running network: Got invalid dimensions for input: x
 index: 0 Got: 1 Expected: 2
 index: 1 Got: 1 Expected: 8
```

Esse erro **prova que o runtime está de fato executando o grafo** — ele leu a
assinatura do tensor, comparou com o modelo e rejeitou.

### AC-4: verdict T4 GO ou PIVOT

**✅ GO.** Binding viável em Windows amd64.

## Cadeia de bloqueios (do mais fundamental ao menos)

O spike encontrou **dois** pré-requisitos que a spec não previa:

| # | Erro literal | Causa | Resolução |
|---|---|---|---|
| 1 | `cgo: C compiler "gcc" not found: exec: "gcc": executable file not found in %PATH%` | Máquina sem compilador C | `pacman -S --needed mingw-w64-ucrt-x86_64-gcc` → gcc 16.1.0 em `C:\msys64\ucrt64\bin` |
| 2 | `The requested API version [29] is not available, only API versions [1, 22] are supported in this build. Current ORT Version is: 1.22.0` | As `onnxruntime.dll` da máquina vêm com Firefox (1.22) e Office (1.22), e o binding exige **API 29 = ONNX Runtime 1.29.0** | Baixar `onnxruntime-win-x64-1.29.0.zip` do GitHub Releases (76 MB), DLL em `~/.memory/models/onnxruntime/` |

> [!warning] Armadilha registrada
> A máquina **já tinha** `onnxruntime.dll` em dois lugares
> (`Program Files/Mozilla Firefox` e `Microsoft Office/.../WinAppSDK`). Elas
> parecem resolver o problema e **não servem** — versão errada. Sem o download
> explícito da 1.29.0, o T1 para com um erro de versão que parece incompatibilidade
> de binding quando na verdade é de runtime.

## T2 — Evidência (supply chain)

**✅ PASS. O binding NÃO está abandonado.**

| Métrica | Valor |
|---|---|
| Versões publicadas | **37** (`v1.0.0` … `v1.36.0`) |
| Última release | `v1.36.0`, **2026-09-04** |
| Cadência | Ativa e recente |
| Licença | permissive (wrapper da API pública do ONNX Runtime, Microsoft) |

Critério de PIVOT por abandono **não se aplica**.

## Ambiente exigido (para a Fase 2+)

| Pré-requisito | Valor | Como foi resolvido |
|---|---|---|
| Compilador C | gcc 16.1.0 | MSYS2 + `mingw-w64-ucrt-x86_64-gcc` |
| `CGO_ENABLED` | `1` | precisa ser explícito por processo |
| PATH | inclui `C:\msys64\ucrt64\bin` | **não está no PATH do sistema** — passar por env |
| ONNX Runtime | 1.29.0 DLL | `~/.memory/models/onnxruntime/onnxruntime.dll` |
| Binding | `github.com/yalue/onnxruntime_go` v1.36.0 | ainda **não** está no `go.mod` do my-memory |

## SPEC_DEVIATION

**Duas**, ambas registradas para correção da spec:

1. **Pré-requisito de runtime não previsto.** A spec (T1 AC-1) só cobria *build*.
   Houve um segundo bloqueio, em runtime: a DLL do ONNX Runtime precisa ser a
   1.29.0 e precisa ser passada via `ort.SetSharedLibraryPath`. Nova AC deveria
   existir em T1 para "carregar a DLL correta e reportar a versão".

2. **API do binding mudou em relação a documentações antigas.**
   `ort.SetSharedLibraryPath(...)` **não retorna erro** em v1.36.0 (assinatura
   `func(path string)`); `NewSessionOptions()` retorna `(*SessionOptions, error)`;
   `NewAdvancedSession` tem 6 parâmetros; `NewDynamicAdvancedSession` é a
   variante com nomes de entrada/saída. Escrever a implementação contra
   assinatura antiga quebra a compilação. Deve-se sempre ler a assinatura real
   no módulo.

## Diff range

Nenhum commit no my-memory: o spike rodou em módulos descartáveis em `%TEMP%`
(`onnxprobe2..5`). `go.mod` **não foi tocado**.

## Próximo passo

**Fase 2 (T3–T5)** — liberada. Mas ver correção de ordenação registrada em
`tasks.md`: o benchmark de projeção (T4) exige um embedder funcionando, então
o tokenizer precisa vir **antes** dele.