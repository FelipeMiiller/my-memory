# Feature: compile-not-retrieve-and-bilateral-mcp

## Problem Statement
O My-Memory atualmente funciona como uma ferramenta exclusivamente de consulta (*read-only*) para os Agentes de IA via Model Context Protocol (MCP). Quando um agente adquire novos entendimentos conceituais, toma decisões de arquitetura ou localiza fragmentos espalhados no repositório, ele não possui mecanismo nativo para:
1. **Consolidar o Conhecimento em Notas Atômicas:** A IA precisa reprocessar fragmentos dispersos repetidamente (custo elevado de contexto e propensão a alucinações).
2. **Escrita Bilateral com Segurança:** A IA não possui ferramentas MCP seguras para criar ou atualizar notas respeitando o sandbox do vault, prevenindo sobrescrita acidental e *path traversal*.
3. **Sincronização Contínua Automática:** Notas criadas ou alteradas pela IA devem ser imediatamente gravadas em Markdown no Git e reindexadas cirurgicamente no SQLite/PostgreSQL sem depender de reindexações manuais.

## Goals
- [ ] Implementar pacote `internal/compiler` com resolução segura de caminhos (*safe sandbox*) restrita à raiz do vault.
- [ ] Implementar geração padronizada de frontmatter YAML, tags e arestas tipadas/epistêmicas (`[[rel:relation:Target]]`).
- [ ] Implementar apensamento estruturado de seções sob cabeçalhos Markdown existentes ou novos.
- [ ] Implementar síntese *Compile-not-Retrieve* consolidando fragmentos de busca RRF em notas estruturadas com backlinks.
- [ ] Implementar reindexação cirúrgica imediata da nota após escrita no disco.
- [ ] Expor novas ferramentas MCP: `memory_write_note`, `memory_append_section` e `memory_compile_note`.
- [ ] Adicionar subcomandos CLI `mem note create`, `mem note append` e `mem compile`.
- [ ] Registrar decisão de arquitetura na ADR-018.

## Out of Scope
- Edição colaborativa em tempo real com CRDTs ou locks distribuídos (o modelo adota escrita atômica local em filesystem).
- Geração autônoma de texto via LLMs embutida sem agente externo (o My-Memory orquestra e estrutura os metadados, links e fragmentos; o conteúdo semântico é fornecido pelo agente chamador ou pelos fragmentos indexados).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Sandbox de Escrita | Raiz do vault descoberto ou explicitado | Impede que ferramentas MCP criem ou alterem arquivos fora da base de memória | y |
| Prevenção de Sobrescrita | Rejeitar por padrão se arquivo existir (`overwrite: false`) | Evita perda acidental de notas prévias sem autorização explícita | y |
| Formatação de Markdown | UTF-8 sem BOM com frontmatter YAML | Compatibilidade plena com Obsidian, GitHub e parsers padrão | y |
| Sincronização Imediata | Reindexação cirúrgica síncrona na chamada da ferramenta | Garante consistência imediata do grafo e busca para a próxima ação do agente | y |
| Fallback para Embeddings | Continua gravação mesmo se Ollama estiver indisponível | Não bloqueia persistência de Markdown soberano se o modelo local estiver offline | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Resolução Segura de Caminho e Criação de Notas Atômicas ⭐ MVP

**User Story**: As a agente de IA ou desenvolvedor, I want criar notas Markdown com frontmatter estruturado e conexões tipadas so that novos conceitos sejam persistidos com segurança no vault.

**Why P1**: Fornece o alicerce fundamental de gravação segura e padronizada no repositório.

**Acceptance Criteria**:
1. The system SHALL provide `SafeResolvePath` in `internal/compiler` ensuring target files remain within the vault boundary.
2. IF a requested path attempts directory traversal (`../` outside vault root) THEN the system SHALL reject the operation with a path traversal error.
3. WHEN `WriteAtomicNote` executes with a new file path THEN the system SHALL format frontmatter YAML and write the file in UTF-8 without BOM.
4. IF a file already exists AND `overwrite` is false THEN the system SHALL return an error without modifying the file.
5. The system SHALL support adding epistemic relations (`[[rel:relation:Target]]`) and tags into the generated Markdown document.

**Independent Test**: Testes unitários em `internal/compiler/note_test.go` validando isolamento de diretório, geração de frontmatter e recusa de sobrescrita.

---

### P2: Apensamento Inteligente de Seções e Compilação de Busca

**User Story**: As a agente de IA, I want anexar seções em notas existentes e compilar resultados de busca em notas consolidadas so that a base evolua organicamente através do padrão Compile-not-Retrieve.

**Why P2**: Elimina duplicidade de notas e implementa a consolidação ativa de conhecimento (*LLM Wiki*).

**Acceptance Criteria**:
1. The system SHALL provide `AppendSection` in `internal/compiler` appending content to an existing heading or creating a new section.
2. WHEN `CompileTopicNote` is invoked with search results THEN the system SHALL format a consolidated note with a synthesis section and backlink wikilinks to source notes.
3. The system SHALL trigger surgical single-file reindexing immediately after writing or appending a note to disk.
4. IF embedding generation fails due to embedder timeout THEN the system SHALL persist the document and record graph edges gracefully.

**Independent Test**: Testes unitários em `internal/compiler/compile_test.go` verificando apensamento sob cabeçalho e montagem de nota compilada.

---

### P3: Ferramentas de Escrita Bilateral no Servidor MCP

**User Story**: As a agente de IA conectado via MCP, I want invocar `memory_write_note`, `memory_append_section` e `memory_compile_note` so that eu possa manter o repositório autoconsciente atualizado durante o trabalho.

**Why P3**: Permite que agentes em Cursor, Claude Code e Antigravity usem a memória como mecanismo ativo de escrita.

**Acceptance Criteria**:
1. The system SHALL declare `memory_write_note`, `memory_append_section` and `memory_compile_note` in `internal/mcp/tools.go`.
2. WHEN `memory_write_note` is called via JSON-RPC THEN the system SHALL write the note, reindex it and return file path, SHA-256 and indexing status.
3. WHEN `memory_append_section` is called via JSON-RPC THEN the system SHALL append content under the specified heading and update the index.
4. IF parameters are invalid or missing required fields THEN the MCP handler SHALL return a JSON-RPC error with descriptive details.

**Independent Test**: Testes de integração JSON-RPC em `internal/mcp/writer_handlers_test.go` validando listagem e chamada das ferramentas.

---

### P4: Interface de Linha de Comando (mem note e mem compile)

**User Story**: As a desenvolvedor no terminal, I want comandos `mem note` e `mem compile` so that eu possa criar notas estruturadas e compilar tópicos diretamente pela CLI.

**Why P4**: Oferece paridade total entre as ferramentas expostas aos agentes e as ferramentas do desenvolvedor humano.

**Acceptance Criteria**:
1. The system SHALL provide `note create` and `note append` subcommands in `cmd/mem`.
2. The system SHALL provide `compile` subcommand in `cmd/mem` compiling search hits into an atomic note.
3. The system SHALL display execution summary including created file path, SHA-256 and graph connections.

**Independent Test**: Testes de execução da CLI em `cmd/mem/note_test.go`.

---

## Edge Cases
- IF the target directory inside the vault does not exist THEN `WriteAtomicNote` SHALL automatically create parent directories.
- IF an empty content string is passed THEN the system SHALL return a validation error.
- IF the vault configuration specifies excluded patterns and the target path matches an excluded pattern THEN the system SHALL reject writing or warn appropriately.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| CNR-01 | P1: Resolução Segura de Caminho e Criação de Notas Atômicas | Tasks | Implementing |
| CNR-02 | P1: Resolução Segura de Caminho e Criação de Notas Atômicas | Tasks | Implementing |
| CNR-03 | P1: Resolução Segura de Caminho e Criação de Notas Atômicas | Tasks | Implementing |
| CNR-04 | P2: Apensamento Inteligente de Seções e Compilação de Busca | Tasks | Pending |
| CNR-05 | P2: Apensamento Inteligente de Seções e Compilação de Busca | Tasks | Pending |
| CNR-06 | P3: Ferramentas de Escrita Bilateral no Servidor MCP | Tasks | Pending |
| CNR-07 | P4: Interface de Linha de Comando (mem note e mem compile) | Tasks | Pending |

**Coverage:** 7 total, 7 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Pacote `internal/compiler` implementado com 100% dos testes unitários passando.
- [ ] Sandbox de escrita estritamente verificado contra path traversal.
- [ ] Ferramentas MCP `memory_write_note`, `memory_append_section` e `memory_compile_note` funcionais e testadas.
- [ ] Comandos CLI `mem note create`, `mem note append` e `mem compile` integrados.
- [ ] Reindexação cirúrgica instantânea mantendo banco de dados em paridade com o Markdown.
- [ ] Decisão registrada na ADR-018 e documentação atualizada.
