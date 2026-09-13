# ADR-006: Integração com Agentes de IA via Model Context Protocol (MCP)

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: ai, mcp, model-context-protocol, integration, tools

## Context and Problem Statement

Para que assistentes e agentes de inteligência artificial (como Claude Code, Cursor, Antigravity, VS Code Copilot e Zed) utilizem a memória do repositório de forma autônoma durante sessões de pair programming ou geração de código, eles precisam de uma interface padronizada de comunicação.

Criar extensões ou plugins proprietários separados para cada IDE ou assistente de IA geraria um custo de manutenção insustentável.

## Decision Drivers

- **Padrão Aberto de Mercado**: Deve funcionar com as principais ferramentas de IA atuais e futuras.
- **Protocolo Baseado em Streams/Stdio**: Sem necessidade de abrir portas de rede locais ou gerenciar certificados TLS.
- **Chamada de Ferramentas Estruturada**: Suporte nativo a *Tool Use* com validação de schemas JSON.

## Considered Options

- **Opção A: Model Context Protocol (MCP)** via comando `mem mcp` (stdio / JSON-RPC).
- **Opção B: Servidor HTTP REST / WebSockets Local** com endpoints customizados.
- **Opção C: CLI Pura** dependendo de scripts de shell invocados pela IA.

## Decision Outcome

Chosen option: **"Opção A: Model Context Protocol (MCP)"**, because é o padrão aberto mantido pela Anthropic e amplamente adotado por toda a indústria de ferramentas de codificação com IA (Claude Code, Cursor, Antigravity, VS Code, Zed). O executável Go pode atuar como um servidor MCP se comunicando via entrada/saída padrão (`stdin`/`stdout`), expondo ferramentas de busca e navegação no grafo com latência mínima.

### Positive Consequences

- **Interoperabilidade Universal:** Uma única implementação em Go funciona imediatamente no Cursor, Claude Code, Antigravity e qualquer outro cliente MCP compatível.
- **Zero Configuração de Rede:** A comunicação via `stdio` elimina conflitos de portas locais (`bind: address already in use`) e riscos de segurança na rede local.
- **Segurança e Controle:** O cliente de IA só executa as ferramentas formalmente declaradas pelo servidor MCP.

### Negative Consequences

- **Debugging de Stdio:** Mensagens acidentais em `stdout` (como logs de debug do Go) podem corromper o fluxo JSON-RPC se não forem direcionadas para `stderr`.

## Pros and Cons of the Options

### Opção A: Model Context Protocol (MCP) ✅ Chosen

- ✅ Padrão oficial adotado pela indústria para agentes de IA.
- ✅ Compatível com múltiplos clientes sem alterar o código do servidor.
- ✅ Execução local segura via `stdio`.
- ❌ Requer rigor na separação de logs (`stderr`) e protocolo (`stdout`).

### Opção B: Servidor HTTP Local REST

- ✅ Fácil de testar manualmente no navegador ou via curl.
- ❌ Requer gerenciar portas, background daemons e firewalls locais.
- ❌ Não é o padrão nativo de ferramentas como Claude Code e Cursor.

### Opção C: CLI Pura via Scripts

- ✅ Implementação imediata.
- ❌ Adiciona overhead de inicializar novo processo a cada consulta.
- ❌ Falta de tipagem rígida de schema para os parâmetros da IA.

## Links

- [Especificação Oficial do Model Context Protocol (modelcontextprotocol.io)](https://modelcontextprotocol.io/)
- [Documentação do Repositório Autoconsciente](../REPOSITORY_BRAIN.md)