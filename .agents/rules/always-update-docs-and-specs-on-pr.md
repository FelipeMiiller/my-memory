# Regra Obrigatória: Atualização Completa de Documentação, Specs e README ao Subir PR

Toda vez que o agente preparar, finalizar ou submeter alterações para Pull Request (PR) ou envio ao repositório remoto (`git push`), ele DEVE OBRIGATORIAMENTE garantir que todo o ecossistema de documentação e especificações esteja 100% atualizado e sincronizado com o código implementado.

Nenhum PR, push ou entrega de funcionalidade é considerado completo se houver discrepância entre o código-fonte e os documentos da base de conhecimento (*zero knowledge drift*).

---

## 📋 Checklist Obrigatório Pré-PR / Pré-Push

### 1. Raiz do Repositório (`README.md`)
- [ ] Atualizar a lista de capacidades, pilares arquiteturais e visão geral.
- [ ] Incluir novos subcomandos da CLI e novas ferramentas MCP nas tabelas e exemplos de uso.
- [ ] Atualizar referências a novas decisões de arquitetura e recursos integrados.

### 2. Documentação Técnica e Guias (`docs/`)
- [ ] **`docs/README.md`**: Adicionar a nova funcionalidade e link para o novo ADR no índice geral de documentação.
- [ ] **`docs/CLI_GUIDE.md`**: Adicionar a seção completa do novo comando da CLI com descrição detalhada de flags, opções, comportamento padrão, casos de uso e exemplos práticos.
- [ ] **`docs/AGENT_INTEGRATION_GUIDE.md`**: Adicionar novas ferramentas MCP na tabela de ferramentas e novos comandos no manual operacional para agentes.
- [ ] **`docs/ARCHITECTURE.md`**: Atualizar diagramas, fluxo de dados, contratos de persistência ou schemas caso haja novas tabelas, colunas ou estruturas de dados.
- [ ] **`docs/adr/`**: Garantir que o ADR correspondente esteja criado no padrão MADR em `docs/adr/NNN-...md` e devidamente indexado em `docs/adr/README.md`.

### 3. Especificações e Governança (`.specs/`)
- [ ] **`.specs/features/<feature>/spec.md`**: Requisitos em formato EARS (`SHALL`), suposições preenchidas e rastreabilidade marcada como `verified`.
- [ ] **`.specs/features/<feature>/tasks.md`**: Todas as tarefas (T1..Tn) concluídas, com testes e gates documentados.
- [ ] **`.specs/features/<feature>/validation.md`**: Relatório com veredito `PASS` e citações exatas de evidência (`arquivo:linha`).
- [ ] **`.specs/STATE.md`**: Registrar a nova decisão (`AD-NNN`) e atualizar o snapshot de `Handoff`.

### 4. Validação Determinística e Zero Drift
Antes de realizar o commit final e o push:
1. **Executar Validadores do TLC**:
   ```bash
   python .agents/skills/tlc-spec-driven/scripts/validate_spec.py <feature> --strict
   python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py <feature> --strict
   python .agents/skills/tlc-spec-driven/scripts/validate_state.py <feature>
   ```
2. **Reindexar o Cofre de Memória**:
   ```bash
   mem index
   ```
3. **Verificar Ausência de Desvio Semântico (Semantic Drift)**:
   ```bash
   mem drift --strict
   ```
   Garantir que nenhum arquivo de código novo permaneça sem nota/ADR vinculado (`Uncovered Code == 0`) e que não existam notas com desvio em nível crítico.
4. **Executar Testes Unitários e Compilar Binário**:
   ```bash
   go test -count=1 ./...
   go build -v -o bin/mem.exe ./cmd/mem
   ```
