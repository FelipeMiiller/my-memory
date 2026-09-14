# ADR-028: Staleness Banners e Detecção de Desatualização de Conhecimento

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: staleness, drift-detection, banners, watcher, mcp, cli, consistency, codegraph

## Context and Problem Statement

No ecossistema do **My-Memory**, os arquivos Markdown versionados no sistema de arquivos representam a única fonte soberana de verdade (*Single Source of Truth*, ADR-005), enquanto o banco de dados (SQLite ou PostgreSQL) serve como um índice relacional e vetorial descartável (*disposable index*).

Entretanto, esse modelo introduz um desafio crítico de alinhamento temporal (*Context Drift*):
1. **Edições Locais Não Indexadas**: Desenvolvedores frequentemente realizam alterações pontuais em notas no Obsidian ou VS Code e esquecem de executar `mem index`.
2. **Ausência ou Atraso do Watcher**: O processo `mem watch` pode não estar em execução no momento da edição, ou uma alteração recente pode estar retida na janela de debouncing (500ms).
3. **Alucinação de Agentes de IA**: Agentes de IA operando via MCP (`memory_search`, `memory_find_path`, `memory_inspect_node`, etc.) ou comandos da CLI consultam o banco e recebem respostas com alta confiança baseadas em informações desatualizadas.

Inspirado na arquitetura do **CodeGraph** ([docs/REFERENCES.md#L65](../../docs/REFERENCES.md#L65)), havia a necessidade de um mecanismo ativo e ultra-leve que verifique se os arquivos físicos em disco divergiram dos registros indexados no banco, emitindo alertas imediatos (**Staleness Banners**) antes que o agente tome decisões de código incorretas.

## Decision Drivers

- **Prevenção Proativa de Alucinação**: Agentes de IA devem ser informados explicitamente sempre que consultarem dados potencialmente desatualizados.
- **Operação Não-Bloqueante**: A detecção não deve interromper a busca nem forçar reindexação síncrona custosa (que travaria a query por segundos gerando novos embeddings no Ollama).
- **Custo Quase-Zero de I/O**: A auditoria do sistema de arquivos (`os.Stat`) deve ser rápida ($< 5\text{ms}$) e utilizar cache em memória com TTL de curta duração (3s) para amortecer chamadas consecutivas.
- **Respeito ao Escopo Declarativo**: Apenas arquivos elegíveis segundo as regras de `ShouldIndex` (`include` e `exclude` do `.memory/config.yaml`) devem ser auditados.
- **Dual-Interface (CLI e MCP)**: Apresentação ergonômica em Markdown para ferramentas MCP e texto legível com novo subcomando `mem status` na CLI.

## Decision Outcome

Adotou-se o motor de **Detecção de Desatualização e Injeção de Staleness Banners** estruturado no pacote `internal/staleness/` e integrado bilateralmente na CLI e no servidor MCP:

### 1. Mecanismo de Detecção (`internal/staleness/detector.go`)
- Compara o conjunto de arquivos elegíveis em disco (`filepath.WalkDir` filtrado por `cfg.ShouldIndex`) com os metadados dos documentos indexados (`SELECT path, updated_at, content_hash FROM documents`):
  - **Modificado**: Arquivo presente em ambos, mas `ModTime.Unix() > doc.UpdatedAt + 1s`.
  - **Novo**: Arquivo existente em disco, mas ausente na tabela `documents`.
  - **Deletado**: Arquivo registrado no banco, mas não mais encontrado no disco.
- Se qualquer uma das listas for não-vazia, o estado é marcado como `IsStale = true`.

### 2. Cache em Memória com TTL
- Estrutura protegida por `sync.RWMutex` retém o último `StalenessReport` por 3 segundos.
- Queries subsequentes do agente ou da CLI reutilizam o relatório sem re-escanear a árvore de arquivos.

### 3. Injeção de Banners nas Ferramentas MCP (`internal/mcp/`)
- Quando `IsStale = true`, as ferramentas analíticas (`memory_search`, `memory_find_path`, `memory_inspect_node`, `memory_get_impact`) injetam no topo da resposta Markdown:
  ```markdown
  > ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (N arquivo(s) pendente(s): X modificado(s), Y novo(s), Z deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute `mem index` ou inicie `mem watch`.
  ```

### 4. Subcomando CLI `mem status` (`cmd/mem/status.go`)
- Fornece diagnóstico imediato no terminal:
  - Total de arquivos em disco vs indexados.
  - Lista de arquivos modificados, novos e deletados.
  - Sugestão dos comandos de sincronização (`mem index` ou `mem watch`).
  - Suporte à flag `--json` para integrações e scripts CI.

### Positive Consequences

- **Consciência de Contexto para IA**: Modelos de linguagem identificam imediatamente quando o repositório foi alterado recentemente, evitando que confiem cegamente em links revogados.
- **Visibilidade Instantânea para o Desenvolvedor**: O comando `mem status` elimina dúvidas sobre se o índice está atualizado.
- **Desempenho Preservado**: A checagem baseada em metadata e protegida por cache consome frações mínimas de milissegundos.

### Negative Consequences / Trade-offs

- **Tolerância Temporal de 1 Segundo**: Para prevenir discrepâncias de precisão de relógio entre sistemas de arquivos (FAT32/NTFS/ext4), variações inferiores a 1 segundo não disparam alteração, o que é largamente mitigado pelo cache incremental de conteúdo SHA-256 do `mem index`.

---

## Links e Referências

- [docs/REFERENCES.md - CodeGraph Staleness Banners](../../docs/REFERENCES.md#L65)
- [ADR-005: Markdown com Wikilinks como Fonte de Verdade](005-markdown-com-wikilinks-como-fonte-de-verdade.md)
- [ADR-010: Cache Incremental de Indexação com SHA-256](010-cache-incremental-de-indexacao-com-sha256.md)
- [ADR-017: Indexação Contínua em Tempo Real com File Watcher e Git Hooks](017-indexacao-continua-com-file-watcher-e-git-hooks.md)
