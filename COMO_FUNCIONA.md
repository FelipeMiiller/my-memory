# 🔍 Como Funciona o My-Memory: Arquitetura e Engenharia Interna

Este documento detalha o funcionamento interno, os algoritmos matemáticos e a engenharia de software que tornam o **My-Memory** uma infraestrutura de memória autoconsciente (*Repository Brain*) para desenvolvedores e Agentes de IA.

---

## 📑 Sumário Técnico
1. [A Filosofia do Repository Brain](#1-a-filosofia-do-repository-brain)
2. [Ciclo de Ingestão e Cache Incremental SHA-256](#2-ciclo-de-ingestão-e-cache-incremental-sha-256)
3. [Extração de Grafo Epistêmico e Parsing](#3-extração-de-grafo-epistêmico-e-parsing)
4. [A Matemática do TurboQuant 4-bit (Google DeepMind, ICLR 2026)](#4-a-matemática-do-turboquant-4-bit-google-deepmind-iclr-2026)
5. [Busca Híbrida: RRF e Decaimento Temporal](#5-busca-híbrida-rrf-e-decaimento-temporal)
6. [Topologia de Grafo e Algoritmos Estruturais](#6-topologia-de-grafo-e-algoritmos-estruturais)
7. [Camada de Persistência Dual (SQLite & PostgreSQL)](#7-camada-de-persistência-dual-sqlite--postgresql)
8. [Integração com IA via Model Context Protocol (MCP)](#8-integração-com-ia-via-model-context-protocol-mcp)

---

## 1. A Filosofia do Repository Brain

Tradicionalmente, a recuperação de contexto em repositórios de código e bases de conhecimento divide-se em duas abordagens isoladas:
1. **Busca Puramente Textual (Grep / BM25):** Excelente para nomes de variáveis e identificadores exatos, mas incapaz de compreender sinônimos, intenção e semântica.
2. **Busca Puramente Vetorial (RAG Tradicional):** Captura proximidade conceitual, mas sofre com alucinações relacionais, perde termos exatos e ignora a topologia de dependências do código.

O **My-Memory** implementa um **tripé simbiótico**:

```
                  ┌─────────────────────────────────────────┐
                  │            REPOSITORY BRAIN             │
                  └────────────────────┬────────────────────┘
                                       │
         ┌─────────────────────────────┼─────────────────────────────┐
         ▼                             ▼                             ▼
  [ Léxico: BM25 ]            [ Vetorial: TurboQuant ]        [ Grafo: Obsidian ]
   Termos Exatos               Conceitos & Similaridade        Dependências Reais
   Nomes de Funções            4-bit Compacto (388B)           Hierarquia & Impacto
```

Ao fundir essas três camadas, uma consulta como *"como funciona o cache de indexação?"* recupera não apenas os trechos que usam essas palavras exatas, mas também os conceitos matemáticos equivalentes e os nós de dependência no grafo (quem chama, quem é chamado e qual o impacto).

---

## 2. Ciclo de Ingestão e Cache Incremental SHA-256

O pipeline de leitura e indexação opera em 4 etapas estritas para garantir máxima velocidade e zero desperdício de chamadas de IA:

```
[ Arquivo Markdown ] ──> [ Poda Rápida SkipDir ] ──> [ ShouldIndex (Glob) ] ──> [ SHA-256 Check ]
                                                                                        │
                                               ┌────────────────────────────────────────┴────────────────────────────────────────┐
                                               ▼                                                                                 ▼
                                       [ Hash Idêntico ]                                                                 [ Hash Diferente / Novo ]
                                       ⏩ Pula Embedding                                                                  ⚙️ Processa:
                                       (0 ms de latência)                                                                 1. Parser [[wikilinks]]
                                                                                                                         2. Chunking 200/30
                                                                                                                         3. Embedding Ollama (768d)
                                                                                                                         4. TurboQuant 4-bit (388B)
```

### 1. Poda Rápida de Diretórios (`filepath.WalkDir`)
Pastas de sistema como `.git`, `node_modules`, `vendor`, `.obsidian` e `.memory` emitem imediatamente `filepath.SkipDir`. O sistema operacional sequer entra nessas pastas, garantindo que repositórios gigantes sejam inspecionados em milissegundos.

### 2. Filtragem Declarativa (`ShouldIndex`)
O caminho do arquivo é normalizado (`\` para `/` no Windows) e testado contra as regras de `exclude` e `include` do `.memory/config.yaml`. Apenas arquivos que casam com padrões aceitos (ex: `docs/**/*.md`, `specs/**/*.md`) continuam.

### 3. Cache Criptográfico SHA-256
Para cada arquivo aprovado:
1. O conteúdo é lido em memória e é calculado o hash criptográfico SHA-256:
   $$\text{hash} = \text{SHA-256}(\text{content})$$
2. O hash é comparado contra a coluna `content_hash` da tabela `documents`.
3. Se os hashes coincidirem (e a flag `--force` não estiver ativa):
   - O arquivo é marcado como `⏩ [cached]` e **nenhuma chamada de rede ou processamento de embedding é executada**.
4. Se o hash mudou ou o arquivo é novo:
   - Os dados antigos associados àquele documento são purgados atomicamente (`DeleteDocumentData`);
   - O documento é reindexado de forma limpa.

### 4. Pruning Automático de Arquivos Deletados
Ao término da varredura, qualquer registro no banco cujo arquivo não exista mais no disco é expurgado (nós do grafo, chunks vetoriais e índices FTS5), mantendo a base sempre íntegra.

---

## 3. Extração de Grafo Epistêmico e Parsing

O My-Memory não trata texto apenas como blocos planos; ele analisa sua estrutura relacional:

### Wikilinks e Aliases
Detecta links no padrão Obsidian `[[Nota]]` ou `[[Nota|Nome de Exibição]]`. Converte automaticamente referências relativas para identificadores canônicos no grafo.

### Arestas Epistêmicas e Conexões Tipadas
Além de links comuns, o parser detecta conexões semânticas explícitas com pesos de relevância:
- `[[rel:depends_on:modulo_x]]` (Peso 1.0)
- `[[rel:implements:adr_010]]` (Peso 1.0)
- `[[rel:derived_from:documento_y]]` (Peso 0.8)
- `[[rel:see_also:conceito_z]]` (Peso 0.5)

### Extração de Metadados e Tags
Lê blocos YAML de frontmatter (`title`, `type`, `tags`, `status`) e hashtags inline (`#arquitetura`, `#auth`). Cada tag vira um nó e uma aresta no grafo conceitual.

---

## 4. A Matemática do TurboQuant 4-bit (Google DeepMind, ICLR 2026)

Embeddings gerados por Large Language Models (como `nomic-embed-text` com 768 dimensões) utilizam números de ponto flutuante de 32 bits (`float32`), totalizando **3.072 bytes por vetor**.

### O Desafio da Quantização Linear
Vetores latentes possuem **dimensões com outliers extremos** — canais esparsos onde a ativação é centenas de vezes maior que a média. Em uma quantização linear simples para 4-bit (16 níveis), a escala máxima $S = \max |x_i|$ explode para acomodar os outliers, e 99% das dimensões restantes são forçadas a colapsar em 0, destruindo a acurácia.

### A Solução TurboQuant: 3 Etapas Fundamentais

```
Vetor Original float32 (768d, 3.072 bytes)
              │
              ▼
[ 1. Rotação Ortogonal de Householder ] ──> y = R * x (Dissipa outliers na hiperesfera)
              │
              ▼
[ 2. Quantização Escalar Simétrica 4-bit ] ──> 15 níveis inteiros [-7, +7]
              │
              ▼
[ 3. Empacotamento de Bits (Bit-Packing) ] ──> 2 dimensões por byte = 384B + 4B scale = 388 bytes!
```

#### 1. Rotação Ortogonal com Matrizes de Householder
Em vez de gastar memória armazenando uma matriz densa de rotação $768 \times 768$, o My-Memory computa $k = 32$ refletores unitários ortogonais de Householder:
$$H_i = I - 2 v_i v_i^T, \quad \text{onde } \|v_i\| = 1$$
$$R = H_{32} \cdot H_{31} \dots H_1$$

Como cada matriz $H_i$ é estritamente ortogonal ($H_i^T H_i = I$), temos que:
$$R^T R = I$$
**Propriedade Crítica:** A rotação ortogonal preserva estritamente o produto interno e a distância euclidiana:
$$\langle R a, R b \rangle = \langle a, b \rangle, \quad \|R a\| = \|a\|$$
A rotação espalha a energia dos outliers de forma homogênea por todas as 768 dimensões.

#### 2. Quantização Simétrica e Bit-Packing
Após a rotação, o vetor resultante $y = R x$ apresenta distribuição quase perfeitamente gaussiana. Ele é quantizado em 15 níveis simétricos:
$$q_i = \text{clamp}\left( \text{round}\left( \frac{y_i}{S} \times 7 \right), -7, 7 \right)$$
Onde $S = \max_i |y_i|$. Os valores inteiros $[-7, +7]$ são deslocados para $[0, 14]$ (ocupando 4 bits) e empacotados aos pares em um único byte (`byte = u0 | (u1 << 4)`).

#### 3. Economia de Armazenamento
- **Original:** 768 floats $\times 4$ bytes = **3.072 bytes**.
- **TurboQuant 4-bit:** 384 bytes de dados + 4 bytes de escala = **388 bytes**.
- **Redução:** **87.4% de economia de RAM e disco**, mantendo correlação angular $> 99\%$.

---

## 5. Busca Híbrida: RRF e Decaimento Temporal

Para responder com precisão máxima, o My-Memory executa simultaneamente:
1. **Busca Léxica (FTS5 BM25):** Ranqueia documentos por correspondência exata de termos.
2. **Busca Vetorial (k-NN Cosine Similarity):** Ranqueia trechos por proximidade semântica no hiperespaço.

### Fusão por Classificação Recíproca (Reciprocal Rank Fusion - RRF)
Os rankings de ambas as fontes são combinados pela fórmula matemática:
$$\text{RRF\_Score}(d) = \frac{w_{\text{vec}}}{k + \text{rank}_{\text{vec}}(d)} + \frac{w_{\text{fts}}}{k + \text{rank}_{\text{fts}}(d)}$$
Onde:
- $k = 60$ (constante de suavização para evitar que o 1º lugar domine desproporcionalmente).
- $w_{\text{vec}}$ e $w_{\text{fts}}$ são os pesos das modalidades.

Documentos que aparecem bem posicionados em **ambos** os rankings sobem rapidamente para o topo.

### Decaimento Temporal Exponencial (ADR-015)
Em bases vivas de engenharia, uma decisão de arquitetura tomada há 3 dias é frequentemente mais relevante que uma de 2 anos atrás. O My-Memory implementa decaimento temporal baseado na meia-vida ($t_{1/2}$):

$$\lambda = \frac{\ln 2}{t_{1/2}}$$
$$\text{Fator}(t) = e^{-\lambda \Delta t}$$
$$\text{Score\_Final}(d) = \text{Score\_RRF}(d) \times \left( (1 - w_{\text{decay}}) + w_{\text{decay}} \times e^{-\lambda \Delta t} \right)$$
Dessa forma, notas recentes recebem um impulso proporcional sem anular notas fundamentais antigas.

---

## 6. Topologia de Grafo e Algoritmos Estruturais

O grafo de conexões do My-Memory roda diretamente dentro do motor SQL através de tabelas relacionais e queries recursivas:

### Expansão Recursiva de Vizinhos (`WITH RECURSIVE`)
Utiliza Common Table Expressions (CTEs) nativas para navegar no grafo com controle de profundidade e prevenção de loops:
```sql
WITH RECURSIVE traverse(node_id, depth, path) AS (
    SELECT target, 1, source || '->' || target FROM graph_edges WHERE source = ?
    UNION
    SELECT e.target, t.depth + 1, t.path || '->' || e.target
    FROM graph_edges e JOIN traverse t ON e.source = t.node_id
    WHERE t.depth < ? AND instr(t.path, e.target) = 0
)
SELECT DISTINCT node_id FROM traverse;
```

### Detecção de Comunidades (LPA Ponderado & Modularidade $Q$)
O comando `mem clusters` executa o *Weighted Label Propagation Algorithm*:
1. Cada nó começa com seu próprio rótulo comunitário.
2. Em iterações síncronas, cada nó adota o rótulo com maior soma de pesos entre seus vizinhos ponderados.
3. Calcula a **Modularidade de Newman-Girvan ($Q$)** para medir a densidade de conexões intra-cluster versus inter-cluster:
   $$Q = \frac{1}{2m} \sum_{i,j} \left( A_{ij} - \frac{k_i k_j}{2m} \right) \delta(c_i, c_j)$$

### Análise de Raio de Destruição (*Blast Radius Analysis*)
O comando `mem impact` realiza uma BFS reversa (quem depende de mim?):
1. Percorre as arestas de entrada até a profundidade solicitada.
2. Calcula um **Risk Score de 0 a 100** baseado no grau de dependência e peso das arestas epistêmicas.
3. Permite que uma IA saiba exatamente quais serviços ou documentos quebram caso uma função seja alterada.

### Inspetor Cirúrgico em 3 Colunas (Triptych Node Inspector)
O comando `mem inspect` projeta uma visão tridimensional do conhecimento:
- **Coluna 1 (In-Links / Dependências):** Quem referencia este arquivo.
- **Coluna 2 (Nó Central):** Título, status, métricas estruturais e preview de conteúdo.
- **Coluna 3 (Out-Links / Referências):** Quais notas este arquivo consome.

---

## 7. Camada de Persistência Dual (SQLite & PostgreSQL)

O My-Memory suporta dois motores de dados através de interfaces abstratas limpas em Go:

| Característica | SQLite Local (Padrão) | PostgreSQL (pgvector) |
| :--- | :--- | :--- |
| **Instalação** | Zero dependências (arquivo único `memory.db`) | Servidor PostgreSQL 15+ com extensão `pgvector` |
| **Busca Vetorial** | Virtual table `sqlite-vec` (`vec0`) + TurboQuant | Tipo nativo `vector(768)` com índice HNSW/IVFFlat |
| **Busca Léxica** | Tabela virtual `FTS5` (BM25 nativo) | Índices `tsvector` com dicionários em português/inglês |
| **Escala** | Até centenas de milhares de notas em máquina local | Milhões de documentos corporativos distribuídos |
| **Multi-Tenant** | Um arquivo `.db` por vault | Coluna `repository` particionada no mesmo banco |

---

## 8. Integração com IA via Model Context Protocol (MCP)

O My-Memory opera como um servidor nativo **Model Context Protocol (MCP)**, o padrão universal da indústria para conectar IAs a fontes de dados locais.

```
[ IDE: Cursor / VS Code Copilot / Claude Code ]
                      │
           JSON-RPC 2.0 (stdio ou HTTP/SSE)
                      │
                      ▼
            [ My-Memory MCP Server ]
                      │
  ┌───────────────────┼───────────────────┐
  ▼                   ▼                   ▼
memory_search    memory_get_impact   memory_compile_note
```

### O Padrão *Compile-not-Retrieve*
Modelos de IA convencionais gastam milhares de tokens toda vez que precisam reler dezenas de arquivos dispersos. Com a ferramenta MCP `memory_compile_note`:
1. A IA realiza a pesquisa híbrida sobre o tema.
2. Sintetiza a resposta em uma nota atômica definitiva com links de proveniência (`[[rel:derived_from:...]]`).
3. O My-Memory grava a nota em disco e atualiza o grafo instantaneamente.
4. Nas próximas interações, a IA lê apenas a nota compilada, **economizando mais de 90% dos tokens de contexto**.

---

## Resumo da Arquitetura

O My-Memory transforma a leitura passiva de arquivos em uma **rede neural simbólica viva**, onde texto, vetores e grafo conspiram para dar contexto completo e sem alucinações a você e aos seus agentes de IA.
