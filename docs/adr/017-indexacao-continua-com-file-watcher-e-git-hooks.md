# ADR-017: Indexação Contínua em Tempo Real com File Watcher e Git Hooks

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: watcher, live-indexing, debounce, git-hooks, pre-commit, sqlite, postgres, mcp

## Context and Problem Statement

Antes desta decisão, a sincronização do grafo de conhecimento no My-Memory dependia exclusivamente da execução manual ou periódica em lote de `mem index`.

Embora o cache incremental SHA-256 (ADR-010) tenha tornado as reindexações em lote muito rápidas, essa abordagem gerava limitações no fluxo de trabalho:
1. **Desfasagem Temporal**: Alterações, novas ideias e anotações feitas no Obsidian ou editor de código só ficavam disponíveis para busca (`mem search`) ou ferramentas de IA (`mem mcp`) após o usuário se lembrar de rodar `mem index`.
2. **Rajadas de I/O em Edição Ativa**: Editores gravam arquivos intermediários e temporários em rajadas durante a digitação ou auto-save, exigindo um mecanismo inteligente de amortecimento (debouncing) para não sobrecarregar o modelo de embeddings (Ollama).
3. **Risco de Desincronização em Commits**: Ao realizar commits em repositórios versionados por Git, notas podiam ser enviadas sem terem seus links, grafos e embeddings sincronizados no banco de memória.

Havia a necessidade de uma solução integrada para monitoramento contínuo em segundo plano (`mem watch`) e garantia de consistência pré-commit (`mem hook`).

## Decision Drivers

- **Zero Dependências CGO / Portabilidade Pura**: O monitor de arquivos deve funcionar em Windows, Linux e macOS sem bibliotecas C externas ou drivers de kernel problemáticos.
- **Debouncing Eficiente**: Rajadas rápidas de escrita no mesmo arquivo devem ser consolidadas em um único evento de indexação cirúrgica.
- **Indexação Cirúrgica em Tempo Real**: Atualizar unicamente o arquivo modificado (documento, arestas, chunks e embeddings) ou expurgar seus registros em caso de remoção, sem reprocessar todo o vault.
- **Encerramento Gracioso (Graceful Shutdown)**: O processo `mem watch` deve respeitar `SIGINT` (Ctrl+C) e `SIGTERM`, fechando conexões de banco e canais com segurança.
- **Git Hooks Seguros e Não-Invasivos**: O instalador de hooks deve proteger hooks pré-existentes de terceiros e permitir desinstalação limpa e idempotente.
- **Integração Declarativa**: Herdar parâmetros de debounce, intervalo e exclusão a partir do arquivo `.memory/config.yaml`.

## Decision Outcome

Adotou-se o motor de **Indexação Contínua e Gestão de Hooks** implementado no pacote `internal/watcher` e integrado ao CLI `cmd/mem`:

1. **Motor de File Watcher e Debounce (`internal/watcher/`):**
   - `Watcher`: Mecanismo puro em Go utilizando `time.Ticker` periódico para comparar `ModTime` e `Size` de arquivos contra um snapshot em memória.
   - `Debouncer`: Canal de amortecimento com janela configurável (padrão 500ms) que cancela timers anteriores ao receber novos eventos no mesmo caminho, consolidando rajadas de salvamento em um único disparo.
   - Filtragem automática através de `cfg.ShouldIndex`, ignorando diretórios de sistema (`.git`, `.obsidian`, `node_modules`, `.memory`) antes de emitir eventos.

2. **Reindexação Cirúrgica (`internal/watcher/indexer.go`):**
   - `IndexSingleFileSQLite` e `IndexSingleFilePostgres`: Reindexa pontualmente uma única nota modificada. Valida o hash SHA-256 e, se alterado, atualiza o grafo de arestas tipadas (ADR-011), regera os chunks e grava novos embeddings no SQLite ou PostgreSQL (pgvector).
   - `PurgeSingleFileSQLite` e `PurgeSingleFilePostgres`: Remove cirurgicamente documentos, arestas e embeddings de arquivos deletados, preservando a higiene do grafo.

3. **Subcomando CLI `mem watch` (`cmd/mem/main.go`):**
   - Sintaxe: `mem watch [--debounce <ms>] [--interval <ms>] [--db <arq>] [--postgres <url>] [--repo <slug>] [<pasta>]`.
   - Resolve automaticamente a raiz do vault via auto-scoping (`config.FindConfigFile`).
   - Gerencia cancelamento de contexto limpo via `signal.NotifyContext` (Ctrl+C).

4. **Gerenciador de Git Hooks (`cmd/mem/hook.go`):**
   - `mem hook install [--force] [<pasta>]`: Localiza a pasta `.git/hooks` e instala o script executável `pre-commit` assinado pelo My-Memory (`# My-Memory Pre-Commit Hook`).
   - Se já existir um hook de terceiro, recusa a sobrescrita a menos que a flag `--force` seja fornecida.
   - `mem hook uninstall [<pasta>]`: Remove com segurança o hook de pre-commit pertencente ao My-Memory de forma limpa e idempotente.

5. **Configuração Declarativa (`internal/config/`):**
   - Adicionada struct `WatcherConfig` ao `.memory/config.yaml` com parâmetros `debounce_ms` e `interval_ms`.
   - Precedência estrita: Flags da CLI > `.memory/config.yaml` > Defaults do sistema.

### Positive Consequences

- **Sincronia Imediata**: O grafo de conhecimento e a base vetorial são atualizados em milissegundos após o salvamento no editor.
- **IA e MCP Sempre Atualizados**: Agentes conectados via `mem mcp` consultam dados frescos em tempo real sem requerer reindexação manual do usuário.
- **Consistência Garantida em Repositórios**: Com o pre-commit hook ativado, nenhum commit é concluído com notas desatualizadas ou quebradas.
- **Portabilidade Total**: Código 100% Go sem CGO, executando de forma idêntica em Windows, Linux e macOS.

### Negative Consequences / Trade-offs

- **Uso de CPU em Polling**: O watcher baseado em polling consome uma fração mínima de CPU a cada ciclo configurado (`interval_ms`), o que é amplamente mitigado pelo filtro de diretórios ignorados (`SkipDir`) e checagem rápida de metadata.

---

## Links e Referências

- **[CodeGraph Auto-Sync (colbymchenry/codegraph)](https://github.com/colbymchenry/codegraph)**: Referência em sincronização reativa com debounced file watcher e staleness banners para agentes de IA.
- [ADR-010: Cache Incremental de Indexação com SHA-256](010-cache-incremental-de-indexacao-com-sha256.md)
- [ADR-016: Configuração Declarativa e Auto-Scoping de Vault](016-configuracao-declarativa-e-auto-scoping-de-vault.md)