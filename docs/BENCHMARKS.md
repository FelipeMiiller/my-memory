# Relatório de Benchmarks de Performance: My-Memory

Este documento consolida as medições empíricas da suíte de micro-benchmarks do **My-Memory**, aferindo throughput, latência por operação e alocação de memória para os componentes centrais da arquitetura: **TurboQuant 4-bit**, **Reciprocal Rank Fusion (RRF)**, **Hashing SHA-256** e **Parsing/Chunking de Markdown**.

---

## 🔬 1. Metodologia e Ambiente de Testes

### 1.1. Hardware e Sistema Operacional
- **Processador:** Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz (28 vCPUs / threads lógicas)
- **Memória RAM:** 64 GB DDR4
- **Sistema Operacional:** Windows Server / Ubuntu Linux (GitHub Actions CI)
- **Linguagem / Compilador:** Go 1.24+ (compilador nativo, flags padrão de otimização)
- **Dimensão Vetorial Padrão:** 768 dimensões (`nomic-embed-text` / compatível com Ollama)

### 1.2. Protocolo de Medição
As medições foram executadas através de duas abordagens complementares e reproduzíveis:
1. **Suíte Padrão Go (`testing.B`)**: `go test -bench="." -benchmem ./internal/...`
2. **CLI Integrado Programático**: `mem bench` (medições de ciclo com aquecimento, isolamento de `runtime.GC` e amostragem estatística)

---

## 📊 2. Resultados Consolidados

### 2.1. TurboQuant 4-Bit vs Float32 (Vetores de 768 Dimensões)

| Operação | Dimensão | Latência | Throughput Estimado | Memória / Op | Alocs / Op | Compressão |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Quantização 4-bit** (`Quantize`) | 768 floats | **56.6 µs/op** | ~17.600 vetores/s | 3.504 B/op | 3 allocs/op | **8x menor** |
| **Desquantização 4-bit** (`Dequantize`) | 768 floats | **52.6 µs/op** | ~19.000 vetores/s | 6.144 B/op | 2 allocs/op | - |
| **Produto Escalar 4-bit** (`DotProduct`) | 768 floats | **1.12 µs/op** | **~890.000 ops/s** | **0 B/op** | **0 allocs/op** | **Zero allocs** |
| **Produto Escalar Float32** (`Exact`) | 768 floats | **301.8 ns/op** | ~3.310.000 ops/s | **0 B/op** | **0 allocs/op** | Linha de base |

#### Destaques de Eficiência:
- **Redução de Memória:** Vetores de 768 floats32 ocupam **3.072 bytes**; após quantização 4-bit ocupam apenas **384 bytes**, gerando **87,5% de economia de espaço em disco e RAM**.
- **Fidelidade Algorítmica:** Erro Médio Absoluto (MAE) no produto escalar de apenas **0.00338** (distribuição normalizada unitária), comprovando a conservação de distância pelas rotações ortogonais de Householder.
- **Zero Alocações na Busca:** Tanto o produto escalar 4-bit quanto o float32 rodam com **0 bytes e 0 alocações por operação**, permitindo saturação máxima de cache L1/L2 de CPU.

---

### 2.2. Fusão Híbrida via Reciprocal Rank Fusion (RRF)

O algoritmo RRF unifica múltiplos fluxos ranqueados (léxico FTS5/tsvector, semântico k-NN e expansão estrutural de grafo) calculando pontuações acumuladas via $RRF(d) = \sum_{s} \frac{1}{k + rank_s(d)}$.

| Cenário de Entrada | Itens Candidatos | Latência | Memória / Op | Alocs / Op | Throughput |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **RRF Médio** | 100 itens (2 fontes) | **153.6 µs/op** | 87.308 B/op | 1.219 allocs/op | **~6.500 queries/s** |
| **RRF Pesado** | 1.000 itens (2 fontes) | **1.87 ms/op** | 926.081 B/op | 14.270 allocs/op | **~530 queries/s** |

#### Conclusão:
A fusão RRF em Go atende com folga o teto de latência interativa (< 5 ms) mesmo em bases com 1.000 itens candidatos por consulta, operando de forma 100% determinística e agnóstica de modelo de embeddings.

---

### 2.3. Cache Incremental e Hashing de Conteúdo (SHA-256)

O sistema de cache incremental do `my-memory` calcula hashes SHA-256 do conteúdo dos arquivos Markdown para ignorar notas inalteradas durante a indexação (`mem index`).

| Tamanho do Payload | Latência | Throughput | Memória / Op | Alocs / Op |
| :--- | :--- | :--- | :--- | :--- |
| **1 KB** (Nota curta) | **3.67 µs/op** | **292.90 MB/s** | 128 B/op | 2 allocs/op |
| **64 KB** (Nota média / longa) | **230.4 µs/op** | **298.60 MB/s** | 128 B/op | 2 allocs/op |
| **1 MB** (Documento consolidado) | **3.41 ms/op** | **314.45 MB/s** | 128 B/op | 2 allocs/op |

#### Conclusão:
O throughput consistente acima de **290 MB/s** garante que repositórios com centenas de notas sejam verificados em milissegundos, evitando re-geração redundante de embeddings e chamadas caras a LLMs/Ollama.

---

### 2.4. Parsing e Chunking de Markdown

| Tarefa | Entrada | Latência | Throughput | Memória / Op | Alocs / Op |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Extração de Conexões** (`[[links]]`, `#tags`, relações tipadas) | Documento Markdown complexo | **785.1 µs/op** | **5.46 MB/s** | 45.541 B/op | 286 allocs/op |
| **Chunking de Texto** (janela de 200 palavras, 30 overlap) | Documento Markdown complexo | **111.5 µs/op** | **76.80 MB/s** | 78.849 B/op | 16 allocs/op |

---

## 🛠 3. Como Reproduzir os Benchmarks

### Via Go Test Nativo
Para executar a suíte formal de benchmarks com relatórios detalhados de memória:
```bash
# Executa todos os micro-benchmarks do repositório
go test -bench="." -benchmem ./internal/turboquant ./internal/store ./internal/parser

# Apenas quantização TurboQuant
go test -bench="BenchmarkQuantize" -benchmem ./internal/turboquant

# Apenas produto escalar 4-bit vs float32
go test -bench="BenchmarkDotProduct" -benchmem ./internal/turboquant
```

### Via CLI Integrada
Para executar a suíte programática diretamente no terminal:
```bash
mem bench
```
Saída esperada:
```
🚀 Executando suíte de micro-benchmarks do My-Memory...
Avaliando: TurboQuant 4-bit, Reciprocal Rank Fusion (RRF), SHA-256 Hashing e Markdown Parsing

==================================================================================================
 Micro-Benchmark                | Iterações  | Latência      | Memória     | Alocações | Throughput
--------------------------------------------------------------------------------------------------
 TurboQuant Quantize (4-bit)    | 2653       | 56.69 µs/op   | 3.4 KB/op   | 3/op      | -
 TurboQuant Dequantize (4-bit)  | 2861       | 52.59 µs/op   | 6.0 KB/op   | 2/op      | -
 TurboQuant Dot Product (4-bit) | 134110     | 1.12 µs/op    | 0 B/op      | 0/op      | -
 Float32 Dot Product (768-dim)  | 348052     | 432 ns/op     | 0 B/op      | 0/op      | -
 RRF Fusão (100 itens)          | 1193       | 125.95 µs/op  | 76.7 KB/op  | 919/op    | -
 RRF Fusão (1.000 itens)        | 107        | 1.42 ms/op    | 805.1 KB/op | 10524/op  | -
 SHA-256 Hashing (64 KB)        | 727        | 207.00 µs/op  | 128 B/op    | 2/op      | 301.93 MB/s
 SHA-256 Hashing (1 MB)         | 46         | 3.26 ms/op    | 128 B/op    | 2/op      | 306.31 MB/s
 Markdown Conexões (Wikilinks)  | 2683       | 56.15 µs/op   | 6.4 KB/op   | 53/op     | 6.47 MB/s
 Markdown Chunking (200w/30o)   | 39066      | 3.85 µs/op    | 1.9 KB/op   | 3/op      | 94.37 MB/s
==================================================================================================
```
