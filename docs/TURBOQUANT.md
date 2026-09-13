# TurboQuant: Teoria e Implementação

O **TurboQuant** é uma técnica de quantização de representações vetoriais de alta dimensão apresentada pelo **Google DeepMind** (ICLR 2026, [arXiv:2504.19874](https://arxiv.org/abs/2504.19874)).

Originalmente desenvolvida para compressão do **KV Cache** de Grandes Modelos de Linguagem (LLMs), a matemática do TurboQuant é perfeitamente aplicável para **índices de bancos vetoriais**, reduzindo drasticamente o consumo de memória sem perda apreciável de acurácia.

---

## 🔬 O Problema da Quantização Convencional

Em vetores de embeddings (ex: 768 dimensões geradas por transformers), os dados apresentam **canais com outliers extremos** — dimensões específicas com magnitudes desproporcionalmente altas. 

Se aplicarmos quantização direta uniforme (como INT4 linear), esses outliers forçam a escala a aumentar, comprimindo 99% das outras dimensões em valores próximos a zero, destruindo a resolução e a similaridade de cosseno.

---

## 🛠 Os Três Pilares da Solução TurboQuant

```
Vetor Original (768 floats, 3072 bytes)
           │
           ▼
[ 1. Rotação Ortogonal Aleatória (Householder) ]
  -> y = R * x (conserva ||x|| e ângulos, dissipa outliers uniformemente)
           │
           ▼
[ 2. Quantização Escalar Simétrica 4-bit ]
  -> 15 níveis simétricos (-7 a +7)
           │
           ▼
[ 3. Empacotamento de Bits (Bit-Packing) ]
  -> 2 dimensões por byte = 384 bytes + 4 bytes scale (388 bytes totais)
```

---

## 📐 Formulação Matemática no Código

### 1. Rotação Ortogonal com Reflexões de Householder (`rotation.go`)
Para evitar armazenar uma matriz densa $768 \times 768$, utilizamos uma sequência de $k = 32$ refletores unitários de Householder:
$$H_i = I - 2 v_i v_i^T, \quad \|v_i\| = 1$$
$$R = H_k \cdot H_{k-1} \dots H_1$$

Propriedades fundamentais:
* $R^T R = I$ (estritamente ortogonal).
* Preserva distâncias euclidianas e produto interno: $\langle R a, R b \rangle = \langle a, b \rangle$.
* Custo computacional: $O(k \cdot D)$, executado em menos de 10 microssegundos.

### 2. Quantização Escalar e Bit-Packing (`quantizer.go`)
Dado o vetor rotacionado $y = R x$:
1. Calcula a escala: $S = \max_i |y_i|$.
2. Quantiza cada componente em 15 níveis inteiros:
   $$q_i = \text{clamp}\left( \text{round}\left( \frac{y_i}{S} \times 7 \right), -7, 7 \right)$$
3. Desloca para faixa sem sinal: $u_i = q_i + 7 \in [0, 14]$ (ocupa 4 bits).
4. Empacota dois valores em um byte:
   $$\text{byte}_j = u_{2j} \mid (u_{2j+1} \ll 4)$$

### 3. Estimador de Produto Escalar Não-Viesado
Para uma query $q$ e um vetor indexado $x$:
1. Rotaciona a query uma única vez: $q_{rot} = R q$.
2. Calcula o produto diretamente com os valores desempacotados:
   $$\widehat{\langle q, x \rangle} = \frac{S}{7} \sum_{i=0}^{D-1} q_{rot}[i] \cdot (u_i - 7)$$
3. Esperança matemática exata:
   $$\mathbb{E}\left[ \widehat{\langle q, x \rangle} \right] = \langle q, x \rangle$$

---

## 📊 Comparativo de Tamanho

| Formato | Dimensões | Tipo | Tamanho por Chunk | Redução |
| :--- | :--- | :--- | :--- | :--- |
| **Float32 Padrão** | 768 | `float32` | 3.072 bytes | Linha de Base |
| **Int8 Clássico** | 768 | `int8` | 768 bytes | ~75.0% |
| **TurboQuant 4-bit** | 768 | `4-bit packed` | **388 bytes** | **~87.4%** |
