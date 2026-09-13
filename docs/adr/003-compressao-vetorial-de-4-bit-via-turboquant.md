# ADR-003: Compressão Vetorial de 4-bit via TurboQuant

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: vectors, quantization, turboquant, math, sqlite

## Context and Problem Statement

Vetores de embedding densos gerados por modelos de ponta (como `nomic-embed-text` de 768 dimensões) utilizam 32 bits por número em ponto flutuante (`float32`), totalizando 3.072 bytes por chunk de texto indexado.

Ao colocar o banco SQLite diretamente dentro do repositório do projeto, armazenar milhares de vetores em `float32` faz o tamanho do arquivo `.db` crescer rapidamente, tornando inviável commitar o banco ou indexar bases de código volumosas. Quantizações lineares ingênuas em 4 bits degradam severamente a busca devido a dimensões com outliers acentuados.

## Decision Drivers

- **Redução drástica de tamanho**: Ocupar menos de 400 bytes por vetor de 768 dimensões.
- **Fidelidade de produto escalar**: Manter correlação de cosseno superior a 99% em relação ao cálculo original em `float32`.
- **Estimador não-viesado**: O erro esperado deve ser matematicamente zero para evitar distorção cumulativa no ranking.
- **Eficiência de CPU**: Rotação e descompressão computáveis em microssegundos sem aceleração por GPU.

## Considered Options

- **Opção A: TurboQuant 4-bit (Google DeepMind, ICLR 2026)** com rotação ortogonal de Householder.
- **Opção B: Armazenamento Nativo Float32** (sem compressão).
- **Opção C: Quantização Escalar Simples INT8** (1 byte por dimensão).

## Decision Outcome

Chosen option: **"Opção A: TurboQuant 4-bit"**, because utiliza um produto de 32 reflexões ortogonais de Householder ($R^T R = I$) para dispersar outliers uniformemente entre todas as dimensões, permitindo quantizar em 15 níveis simétricos com 2 valores por byte (*bit-packing*). O vetor cai de 3.072 bytes para **384 bytes** (~87.4% de redução) mantendo fidelidade de cosseno de 99.6% com estimador não-viesado.

### Positive Consequences

- **Tamanho Minúsculo no Disco:** Reduz o banco SQLite para uma fração do tamanho original, viabilizando bancos com milhares de arquivos ocupando poucos megabytes.
- **Preservação de Distâncias:** Como a rotação de Householder preserva exatamente a norma euclidiana e os ângulos, o produto escalar $\langle Rq, Rx \rangle = \langle q, x \rangle$ não sofre viés sistemático.
- **Busca Rápida em Memória:** As operações podem ser paralelizadas ou executadas com instruções inteiras na CPU.

### Negative Consequences

- **Custo da Rotação na Consulta:** A query de busca precisa ser rotacionada uma vez antes da comparação (custo mínimo: $O(k \cdot D)$ < 10 microssegundos).

## Pros and Cons of the Options

### Opção A: TurboQuant 4-bit ✅ Chosen

- ✅ Redução de ~88% no tamanho de armazenamento dos vetores.
- ✅ Estimador não-viesado de produto escalar ($\mathbb{E}[\text{estimado}] = \text{real}$).
- ✅ Alta fidelidade comprovada em testes unitários (> 99.6%).
- ❌ Requer algoritmo customizado de desempacotamento de bits na busca.

### Opção B: Float32 Nativo

- ✅ Precisão matemática absoluta.
- ❌ Consumo excessivo de espaço (3.072 bytes por chunk).
- ❌ Inviabiliza commitar bancos médios/grandes no Git.

### Opção C: Quantização INT8

- ✅ Simples de implementar.
- ❌ Reduz apenas 75% (768 bytes por vetor), o dobro do tamanho alcançado pelo TurboQuant 4-bit.

## Links

- [Google DeepMind TurboQuant Paper (arXiv:2504.19874)](https://arxiv.org/abs/2504.19874)
- [Documentação Detalhada do TurboQuant no Projeto](../TURBOQUANT.md)