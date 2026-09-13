# ADR-002: Adoção de Go como Linguagem Principal de Implementação

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: language, go, cli, performance, developer-experience

## Context and Problem Statement

O projeto `my-memory` precisa operar como uma ferramenta CLI ultrarrápida e leve, rodando na máquina de desenvolvedores e em pipelines de CI/CD para indexação e consulta de memória semântica e relacional.

A escolha da linguagem afeta diretamente a portabilidade do binário, tempo de inicialização (*cold start*), consumo de memória RAM e facilidade de distribuição entre sistemas operacionais (Windows, Linux, macOS).

## Decision Drivers

- **Binário Único Autocontido**: O usuário não deve precisar instalar runtimes ou gerenciar ambientes virtuais.
- **Tempo de Inicialização Rápido**: O comando CLI deve iniciar em menos de 15ms para não adicionar latência em hooks do Git ou chamadas de IA.
- **Suporte a CGO / SQLite**: Integração simples com extensões C (`sqlite-vec`).
- **Concorrência Nativa**: Facilidade para indexar múltiplos arquivos e processar chunks em paralelo.

## Considered Options

- **Opção A: Go (Golang)**
- **Opção B: Python**
- **Opção C: Rust**

## Decision Outcome

Chosen option: **"Opção A: Go (Golang)"**, because gera um binário estático único com excelente suporte a concorrência via Goroutines, inicialização quase instantânea (<10ms), compilação veloz e integração limpa com drivers SQLite e bibliotecas CGO.

### Positive Consequences

- **Distribuição Simples:** Basta disponibilizar o executável `mem.exe` ou `mem` compilado, sem necessidade de Python, `pip`, `venv` ou Node.js.
- **Consumo Mínimo de Memória:** Opera com apenas alguns megabytes de memória RAM.
- **Cross-Compilation Nativa:** Facilidade de compilar para Windows, Linux e macOS diretamente via GitHub Actions.

### Negative Consequences

- **Ecossistema de IA Tradicional:** A maioria das bibliotecas de Machine Learning de ponta são feitas primeiro em Python, exigindo que implementemos algoritmos matemáticos (como TurboQuant) ou usemos clientes HTTP/ONNX para inferência.

## Pros and Cons of the Options

### Opção A: Go (Golang) ✅ Chosen

- ✅ Binário compilado único sem dependências de runtime.
- ✅ Inicialização e execução extremamente rápidas.
- ✅ Excelente suporte a concorrência nativa (goroutines e channels).
- ❌ Requer bindings em CGO para extensões em C do SQLite.

### Opção B: Python

- ✅ Ecossistema dominante de IA, transformers e Hugging Face.
- ❌ Exige interpretador Python, gerenciar `venv`/`pip` e consome muita memória.
- ❌ Tempo de inicialização lento para comandos frequentes de CLI.

### Opção C: Rust

- ✅ Máxima performance e segurança de memória sem garbage collector.
- ❌ Curva de aprendizado e tempo de compilação significativamente maiores para desenvolvimento ágil.

## Links

- [Go Official Site](https://go.dev/)
- [ADR-001: Uso de SQLite como Camada Unificada de Dados](001-uso-de-sqlite-como-camada-unificada-de-dados.md)