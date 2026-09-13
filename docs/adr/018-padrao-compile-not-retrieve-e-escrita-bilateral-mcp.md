# ADR-018: Padrão Compile-not-Retrieve e Escrita Bilateral na Memória via MCP e CLI

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: compile-not-retrieve, llm-wiki, mcp, bilateral-memory, atomic-notes, frontmatter, sandboxing, cli, sqlite, postgres

## Context and Problem Statement

Até a ADR-017, a interação entre agentes de inteligência artificial (Claude Code, Cursor, Antigravity) e o ecossistema do **My-Memory** era estritamente unidirecional / somente-leitura (*read-only*):
- Os agentes podiam consultar o repositório (`memory_search`), expandir dependências no grafo (`memory_get_neighbors`), inspecionar nós centrais (`memory_get_hubs`) e verificar anomalias (`memory_doctor`).
- Contudo, sempre que o agente sintetizava uma compreensão nova, chegava a uma decisão de arquitetura relevante ou consolidava fragmentos dispersos do código, ele não tinha mecanismo nativo para **gravar e estruturar essa síntese de volta na memória**.

Conforme identificado nas referências fundamentais de arquitetura (`docs/REFERENCES.md`), em particular no motor `akitaonrails/ai-memory` e no padrão *LLM Wiki / Compile-not-Retrieve* (conceitualizado por Andrej Karpathy):
1. **Ineficiência do RAG Passivo:** Fazer buscas repetidas sobre fragmentos brutos desperdiça tokens, aumenta a latência e perde o histórico de raciocínios consolidados.
2. **Soberania do Markdown Git-versionado:** O conhecimento precisa ser salvo como notas atômicas em Markdown humano (`.md`), com YAML frontmatter padronizado (`title`, `type`, `tags`, `aliases`) e arestas explícitas estilo Obsidian (`[[wikilinks]]` e `[[rel:relation:Target]]`).
3. **Escrita Bilateral Segura:** As ferramentas MCP devem garantir *sandboxing* rígido contra *path traversal* (`../`), prevenção de sobrescrita acidental e sincronização cirúrgica instantânea no banco de dados.

## Decision Drivers

- **Padrão Compile-not-Retrieve**: Permitir que agentes e desenvolvedores sintetizem tópicos e resultados de busca em notas conceituais atômicas e permanentes com backlinks automáticos.
- **Escrita Bilateral Segura no MCP**: Expor ferramentas padronizadas no Model Context Protocol (`memory_write_note`, `memory_append_section`, `memory_compile_note`) com isolamento estrito dentro da raiz do vault.
- **Formatação Padronizada de Markdown**: Gerar UTF-8 sem BOM com frontmatter YAML limpo e links semânticos tipados.
- **Sincronização Cirúrgica Síncrona**: Cada operação de escrita deve disparar reindexação cirúrgica imediata no storage ativo (SQLite unificado ou PostgreSQL com pgvector), mantendo a fonte de verdade em perfeita paridade com o índice.
- **Paridade entre CLI e MCP**: Desenvolvedores humanos no terminal devem ter comandos equivalentes (`mem note create`, `mem note append`, `mem compile`) aos utilizados pelos agentes de IA.

## Decision Outcome

Adotou-se o pacote `internal/compiler` e as respectivas extensões no servidor MCP (`internal/mcp/`) e na CLI (`cmd/mem/`):

1. **Núcleo do Compilador e Sandbox de Caminhos (`internal/compiler/`):**
   - `SafeResolvePath(vaultRoot, requestedPath)`: Garante que todo caminho de arquivo permaneça rigorosamente confinado à raiz do vault, rejeitando tentativas de *path traversal* (`../`) e normalizando extensões `.md`.
   - `WriteAtomicNote`: Formata frontmatter YAML padronizado, mescla tags, aliases e arestas tipadas (`[[rel:relation:Target]]`), prevenindo sobrescrita acidental (`overwrite: false` por padrão).
   - `AppendSection`: Localiza cabeçalhos existentes no Markdown e anexa conteúdo cirurgicamente antes da próxima seção de mesmo nível, ou cria uma nova seção no final do documento.
   - `CompileTopicNote`: Executa o padrão *Compile-not-Retrieve*, construindo uma nota atômica com seção de síntese, contextualização do tópico e bloco `## Fontes e Backlinks` referenciando cada documento de origem (`[[rel:derived_from:Doc]]`).
   - `SyncEngine`: Orquestra a sincronização cirúrgica imediata da nota escrita com o SQLite local ou PostgreSQL com pgvector via rotinas de `watcher.IndexSingleFile`.

2. **Novas Ferramentas MCP (`internal/mcp/`):**
   - `memory_write_note`: Cria ou substitui notas atômicas no vault, validando parâmetros e retornando metadados (caminho, SHA-256, arestas extraídas e status de indexação).
   - `memory_append_section`: Anexa seções e atualiza o grafo sem corromper a estrutura existente do documento.
   - `memory_compile_note`: Conecta busca híbrida RRF, extração de fragmentos e síntese atômica em uma única chamada de ferramenta MCP para agentes de IA.

3. **Subcomandos CLI (`cmd/mem/note.go`):**
   - `mem note create [--title "T"] [--tags "t1,t2"] [--type "concept"] [--overwrite] <caminho>`: Cria nota atômica no terminal ou via pipe de stdin.
   - `mem note append --heading "## Seção" [--content "..."] <caminho> [<conteúdo>]`: Anexa seções rapidamente.
   - `mem compile --topic "<termo>" --out "<caminho.md>" [--mode hybrid|vector|fts] [--limit 5]`: Sintetiza conhecimento diretamente pela linha de comando.

### Positive Consequences

- **Evolução Contínua do Cérebro do Repositório**: Agentes de IA agora aprendem e registram ativamente novo conhecimento, consolidando lições, decisões e sínteses diretamente no repositório.
- **Tokens Poupados (Karpathy Pattern)**: Em consultas futuras sobre o mesmo tema, o agente recupera uma nota atômica já consolidada em vez de ler dezenas de trechos desestruturados.
- **Segurança e Confinamento**: O sandbox impede que ferramentas MCP gravem fora da pasta autorizada do vault.
- **Git Versioning Preservado**: Todo conhecimento novo é gerado em Markdown puro, pronto para ser commitado e revisado por humanos.

### Negative Consequences / Trade-offs

- **Dependência de Espaço em Disco**: O crescimento orgânico de notas sintetizadas aumenta o volume de arquivos Markdown no vault (atenuado pelo *pruning* e *higiene de grafo* da ADR-013).
