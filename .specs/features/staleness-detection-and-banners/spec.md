# Feature: staleness-detection-and-banners

## Problem Statement
Em sistemas de memória para IA locais baseados no padrão *Markdown como Fonte da Verdade* (ADR-005), o banco de dados relacional e vetorial (SQLite/PostgreSQL) atua como um índice derivado e descartável (*disposable index*). 

No fluxo de trabalho real de engenharia de software e anotação, o desenvolvedor frequentemente:
1. Edita arquivos Markdown no editor de texto (Obsidian, VS Code, Cursor), modificando links, contratos e decisões arquiteturais.
2. Não executa `mem index` imediatamente por esquecimento ou hábito.
3. Não mantém o daemon `mem watch` rodando em segundo plano, ou a alteração ainda está na janela de amortecimento (debounce de 500ms).

Quando isso ocorre, instala-se o **Context Drift (Desfasagem de Contexto)**: o banco de dados reflete um estado defasado da base de conhecimento. Se um agente de IA executa chamadas MCP (`memory_search`, `memory_inspect_node`, `memory_find_path`, etc.) ou o desenvolvedor executa comandos de consulta no terminal, o sistema responde com alta confiança baseando-se em dados antigos. Isso gera alucinações críticas, planejamento sobre decisões revogadas e sugestões de código obsoletas.

Inspirado na arquitetura de sincronização do **CodeGraph** ([docs/REFERENCES.md#L65](../../../docs/REFERENCES.md#L65)), esta feature introduz a **Detecção Ativa de Desatualização & Banners de Alerta (Staleness Banners)**, que audita em tempo de consulta o alinhamento entre o sistema de arquivos e o banco de dados, injetando alertas explícitos e não-bloqueantes tanto nas respostas das ferramentas MCP quanto na CLI, acompanhado do novo comando de auditoria `mem status`.

---

## Goals
- [ ] **G1**: Criar o pacote de domínio `internal/staleness/` com o motor de detecção comparando `mtime` dos arquivos do vault contra o registro de `updated_at` / `content_hash` da tabela `documents`.
- [ ] **G2**: Implementar cache em memória leve com TTL configurável (padrão: 3s) para garantir custo de I/O insignificante em consultas em lote e sucessivas.
- [ ] **G3**: Criar o subcomando CLI `mem status` (`cmd/mem/status.go`) para exibir o panorama de sincronização do vault (arquivos modificados, novos e deletados) com suporte a saída estruturada `--json`.
- [ ] **G4**: Implementar injeção transparente de banners de desatualização (`FormatMarkdownBanner`) nas ferramentas MCP (`memory_search`, `memory_inspect_node`, `memory_find_path`, etc.) e alertas sutis na CLI (`mem search`, `mem path`, etc.).
- [ ] **G5**: Registrar a decisão arquitetural no documento `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` e atualizar os guias operacionais da CLI e de Agentes.

---

## Out of Scope
- Reindexação automática síncrona forçada em todas as consultas (reindexar modelos de embeddings durante uma query adicionaria latência inaceitável de segundos; o propósito é alertar o agente de que os dados são velhos para que ele ou o usuário decida rodar `mem index` ou iniciar `mem watch`).
- Modificação física de arquivos Markdown no disco.

---

## Assumptions & Open Questions

| Assumption / Decisão | Valor Escolhido | Justificativa | Confirmado? |
| :--- | :--- | :--- | :--- |
| Estratégia de alerta | Não-bloqueante (aviso/banner) | Permite que o agente continue trabalhando, mas com consciência explícita da defasagem | y |
| TTL do Cache em Memória | 3 segundos | Elimina I/O redundante em rajadas de tool calls mantendo a reatividade alta | y |
| Tolerância de mtime | 1 segundo | Evita falsos positivos por imprecisão de nanossegundos de filesystem em diferentes SOs | y |
| Subcomando de auditoria | `mem status` | Nomenclatura intuitiva e universal (similar a `git status`) | y |

---

## User Stories

### P1: Injeção de Staleness Banner nas Respostas MCP ⭐ MVP
**User Story**: Como um agente de IA conectado via MCP, quero receber um aviso explícito no topo do Markdown quando o índice de memória estiver desatualizado em relação ao disco, para que eu não tome decisões equivocadas com base em informações obsoletas.
**Critérios de Aceite**:
1. Se houver arquivos modificados, novos ou deletados desde a última indexação, as respostas de ferramentas como `memory_search`, `memory_find_path` e `memory_inspect_node` incluem o banner: `> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (...)`.
2. Se o índice estiver sincronizado, nenhum banner é injetado.
3. A checagem de arquivos respeita estritamente as regras de `ShouldIndex` (ignora `.git`, `.obsidian`, `node_modules`, etc.).

### P2: Subcomando CLI `mem status`
**User Story**: Como desenvolvedor, quero executar `mem status` para auditar em um instante a saúde da sincronização do meu vault de notas com o banco de dados.
**Critérios de Aceite**:
1. Exibe contagem de arquivos no disco vs indexados no banco.
2. Lista detalhada de arquivos modificados, novos e deletados com seus caminhos relativos.
3. Se houver desatualização, exibe status `⚠️ DESATUALIZADO (Stale)` e recomenda o comando de sincronização.
4. Se estiver 100% alinhado, exibe status `✅ ATUALIZADO (Fresh)`.
5. A flag `--json` retorna o objeto `StalenessReport` completo em JSON.

### P3: Alertas Informativos na CLI
**User Story**: Como usuário executando `mem search` ou `mem path`, quero ver um aviso sutil caso a base consultada não contenha as alterações mais recentes dos meus arquivos.
**Critérios de Aceite**:
1. Se houver desatualização, emite aviso claro no cabeçalho do stdout antes da exibição dos resultados.
