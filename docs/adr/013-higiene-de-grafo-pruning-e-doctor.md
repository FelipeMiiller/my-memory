# ADR-013: Higiene de Grafo, Pruning Incremental e Linter Doctor

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: lifecycle, pruning, garbage-collection, graph-doctor, graph-hygiene, health-score, mcp, cli

## Context and Problem Statement

O My-Memory utiliza cache incremental baseado em SHA-256 para indexação de notas Markdown, evitando o reprocessamento de notas inalteradas. No entanto, o ciclo de vida do armazenamento apresentava duas lacunas estruturais de integridade e higiene de dados:

1. **Ausência de Pruning de Arquivos Deletados (Garbage Collection):** Quando uma nota `.md` era apagada do filesystem, o comando `mem index` percorria apenas os arquivos existentes, ignorando a remoção de documentos ausentes no banco de dados. Com isso, registros fantasmas permaneciam indefinidamente nas tabelas `documents`, `chunks`, `chunks_fts`, `chunks_vec`, `chunks_turboquant` e `graph_edges`.
2. **Falta de Ferramenta de Diagnóstico e Linter de Grafo:** Desenvolvedores e agentes de IA não dispunham de um mecanismo consolidado para auditar anomalias topológicas no grafo de conhecimento — como *dead links* (wikilinks apontando para notas que nunca foram criadas ou foram excluídas), *orphan notes* (documentos sem conexões de entrada ou saída), *self-loops* e consistência de vetores em chunks.

## Decision Drivers

- **Projeção Fiel do Filesystem:** O banco de dados (SQLite ou PostgreSQL) deve refletir exatamente o estado dos arquivos presentes no disco, expurgando dados de notas deletadas.
- **Segurança e Preservação Opcional:** Permitir desativar o pruning automático via flag `--no-prune` em indexações parciais de subdiretórios.
- **Auditoria de Conhecimento e Linter:** Fornecer diagnóstico estrutural com cálculo determinístico de pontuação de saúde (*Health Score* de 0 a 100) e capacidade de auto-reparo (`--fix`) para remoção de arestas mortas e self-loops.
- **Operação Híbrida CLI e MCP:** Disponibilizar o subcomando `mem doctor` para inspeção visual em terminal e a ferramenta `memory_doctor` no protocolo MCP para curadoria autônoma por LLMs.

## Decision Outcome

Adotou-se uma estratégia completa de ciclo de vida e higiene de grafo:

1. **Pruning Incremental Automático (`PruneDeletedDocuments`):**
   - Implementado no SQLite (`internal/db/store.go`) e no PostgreSQL (`internal/store/postgres.go`).
   - Durante a varredura (`filepath.WalkDir`), o indexador acumula o conjunto de identificadores ativos (`activeDocIDs`).
   - Ao término, compara os documentos armazenados na raiz com o conjunto ativo e executa expurgo atômico em cascata: remove chunks de busca léxica (FTS5), vetoriais (`sqlite-vec` / `pgvector` / `TurboQuant`), relacionais, arestas incidentes e nós do grafo.
   - Adicionada a flag `--no-prune` no comando `mem index` para desativar a poda quando desejado.
2. **Motor de Diagnóstico no Contrato `Store` (`DiagnoseHealth` e `FixHealthIssues`):**
   - Definidos tipos `DeadLink`, `OrphanNote`, `SelfLoop`, `DesyncedChunk` e `DoctorReport` no pacote `store`.
   - Adicionados métodos `DiagnoseHealth(ctx, repo)` e `FixHealthIssues(ctx, repo)` na interface `Store`.
   - Implementado cálculo de *Health Score* (0-100) com penalidades graduadas para dead links, percentual de notas órfãs, loops e chunks dessincronizados.
   - Implementada rotina de reparo (`FixHealthIssues`) que expurga com segurança arestas apontando para notas inexistentes e referências reflexivas.
3. **Subcomando CLI `mem doctor`:**
   - Adicionado comando `mem doctor [--fix] [--db <arq>] [--postgres <url>] [--repo <slug>]`.
   - Exibe dashboard ASCII com indicador visual de status (🟢/🟡/🔴), contadores estruturais, lista de links quebrados, notas isoladas e anomalias.
4. **Ferramenta MCP `memory_doctor`:**
   - Declarada `ToolMemoryDoctor` no catálogo de ferramentas MCP com argumentos `repository` e `fix`.
   - Implementados `NewMemoryDoctorHandler`, formatador Markdown `FormatDoctorReport` e método `SetDoctorHandler` integrando servidores SQLite e PostgreSQL.

### Positive Consequences

- **Higiene Garantida:** Deletar arquivos `.md` do disco e rodar `mem index` agora limpa completamente todos os índices e grafos, eliminando respostas obsoletas.
- **Grafo Confiável para Agentes:** LLMs não correm o risco de sugerir navegações para notas inexistentes através do grafo.
- **Curadoria Ativa:** O comando `mem doctor` incentiva a manutenção da base de conhecimento, identificando conceitos isolados que deveriam ser linkados.
- **Compatibilidade:** O pruning opera tanto em bases locais SQLite quanto em instâncias corporativas PostgreSQL com isolamento por repositório.

### Negative Consequences / Trade-offs

- **Custo Adicional na Indexação:** A consulta de reconciliação de documentos adiciona uma rodada de checagem ao fim de `mem index`, porém mitigada com verificação em memória através de hash maps $O(1)$.
