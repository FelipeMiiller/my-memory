# ADR-034: Protocolo Canônico Federado e Wikilinks Cross-Vault (`memory://`)

- **Date**: 2026-09-16
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: federation, cross-vault, canonical-uri, wikilinks, graph, parser, deeplink, mcp, devx

## Context and Problem Statement

Com a implementação da arquitetura federada em estrela ([ADR-033](033-federated-central-vault-and-repo-identity.md)), o ecossistema My-Memory passou a operar com dois níveis de vaults: o **Cofre Central de Conhecimento** (*Global Brain* no Google Drive/OneDrive/Obsidian) e os **Cofres de Projetos** (*Local Brains* nos repositórios de código Git).

No desenvolvimento diário, notas de projetos locais frequentemente precisam referenciar padrões corporativos transversais (ex: normas de autenticação OAuth2, diretrizes de OpenAPI, RFCs corporativas) ou notas de outros microsserviços. Entretanto:
1. **Limitação do Wikilink Tradicional (`[[Nota]]`)**: Os wikilinks do Obsidian assumem que todos os arquivos residem sob a mesma pasta raiz do vault. Quando uma nota em `c:/repo/auth` declara `[[standards/oauth2]]`, o parser local não encontra o arquivo no repositório do projeto e o classifica como nó órfão ou inexistente.
2. **Fragilidade de Caminhos Absolutos (`[[C:/Users/.../standards/oauth2]]`)**: Forçar o uso de caminhos absolutos do sistema de arquivos destrói a portabilidade entre máquinas (Windows vs Linux vs macOS), vaza caminhos privados de usuários para o Git e quebra a navegação quando os vaults são clonados em pastas diferentes.
3. **Falta de Semântica Cross-Repository no Grafo e MCP**: Agentes de IA e desenvolvedores que consultam o grafo de conhecimento local não conseguem discernir se uma aresta aponta para um módulo do projeto atual ou para uma diretriz corporativa do Cofre Central.

Inspirado na especificação **OKF v0.2** (*Open Knowledge Format*) e no projeto [Atlas](https://github.com/sergio-sisternes-epam/atlas) ([docs/REFERENCES.md](../REFERENCES.md)), faz-se necessária uma notação canônica, universal e independente de sistema operacional para endereçamento e navegação federada entre vaults.

## Decision Drivers

- **Portabilidade Absoluta e Zero-Vazamento de SO**: A notação de link deve ser puramente lógica e canônica, sem referenciar caminhos de disco (`C:/`, `/home/...`).
- **Resolução Transparente de Vaults**: O sistema deve resolver dinamicamente se o destino é o Cofre Central (`repo_central` ou `central`) ou um repositório satélite registrado no catálogo global `~/.memory/config.yaml`.
- **Compatibilidade com Obsidian e Wikilinks Ricos**: Suporte a aliases (`[[uri|Texto de Exibição]]`), âncoras de cabeçalho (`[[uri#Seção]]`), block IDs (`[[uri#^id]]`) e relações epistêmicas (`[[implements:uri]]`).
- **Descoberta e Navegação em Editores (`mem open` e Deep Links)**: Permitir abrir instantaneamente notas federadas no Obsidian ou VS Code com cursor posicionado na seção correspondente.
- **Suporte Nativo no Servidor MCP**: Ferramentas como `memory_open_node`, `memory_get_neighbors` e `memory_inspect_node` devem compreender e retornar a proveniência de arestas federadas.

## Considered Options

1. **Caminhos Relativos com Subida de Diretório (`[[../../Google Drive/...]]`)** (Descartado: extremamente frágil, dependente do layout relativo das pastas no disco do desenvolvedor e incompatível no Obsidian Mobile).
2. **Submódulos Git de Links Simbólicos** (Descartado: atrito de manutenção e incompatível com nuvem Google Drive/OneDrive).
3. **Protocolo Canônico Federado `memory://<repo>/<path>` com Resolução via Catálogo Global** (*Opção Escolhida*).

## Decision Outcome

Adotou-se a **Opção 3**:

```
                              [ Nota Local (Markdown) ]
                         [[memory://central/standards/oauth2#JWT]]
                                        │
                                        ▼
                           [ Parser de Wikilinks ]
               - Detecta esquema canônico 'memory://'
               - Extrai: repo='central', path='standards/oauth2', anchor='JWT'
               - Registra aresta federada: repo_local ──links_to──► memory://central/...
                                        │
                    ┌───────────────────┴───────────────────┐
                    ▼                                       ▼
       [ Resolução no Grafo / Busca ]          [ Abertura no Editor (mem open) ]
       - Arestas marcadas como federadas       - Consulta ~/.memory/config.yaml
       - Proveniência visual no terminal       - Obtém caminho absoluto em disco
       - Expansão de vizinhos no MCP           - Monta URI obsidian:// ou vscode://
```

### 1. Especificação da Sintaxe Canônica (`memory://`)

A URI federada adota a seguinte estrutura:

```
memory://<repo_identifier>/<document_path>[#<anchor_or_block>]
```

Onde:
- `<repo_identifier>`: Identificador do vault alvo:
  - `repo_central` ou `central`: Aponta para o Cofre Central de Conhecimento (definido em `central_vault.path`).
  - `repo_<12-hex-chars>`: Identificador imutável de um repositório satélite registrado no catálogo.
  - `<slug>` (ex: `my-org/auth-service` ou nome amigável): Slug de repositório registrado no catálogo `repositories:` de `~/.memory/config.yaml`.
- `<document_path>`: Caminho relativo do documento sem necessidade da extensão `.md` (ex: `standards/oauth2` ou `docs/adr/001-auth`).
- `[#anchor]`: Cabeçalho opcional da nota (ex: `#Diretrizes de Segurança`).
- `[^block_id]`: Identificador de bloco opcional (ex: `^c182`).

### 2. Exemplos Válidos em Notas Markdown

```markdown
<!-- Link simples para o cofre central -->
Consulte as regras em [[memory://central/standards/code-review]].

<!-- Link com alias amigável para leitura humana -->
Implementamos o [[memory://central/standards/oauth2|Padrão Corporativo de OAuth2]].

<!-- Link com âncora de seção -->
Conforme detalhado em [[memory://central/architecture/database-standards#PostgreSQL e pgvector]].

<!-- Link com aresta epistêmica tipada -->
Este serviço [[implements:memory://central/standards/microservices-contract]].

<!-- Link para outro repositório satélite via catálogo -->
Integração via API com [[memory://repo_a1b2c3d4e5f6/docs/api-contracts#Rotas de Pagamento]].
```

### 3. Anatomia do Parser (`internal/parser/wikilinks.go`)

O analisador de wikilinks é aprimorado para blindar o prefixo `memory://` contra falsos positivos de relação semântica (`colonIdx`):
- A estrutura `LinkTarget` é estendida com os campos:
  - `IsFederated bool`
  - `FederatedRepo string`
  - `FederatedPath string`
- O `Target` é mantido como a URI canônica completa (`memory://<repo>/<path>`), garantindo unicidade no banco de dados e no grafo de arestas.

### 4. Motor de Resolução Federada (`internal/federation/resolver.go`)

Um novo motor de resolução converte a URI canônica no caminho absoluto do arquivo no disco host:
1. Se o alvo for `central` ou `repo_central`: resolve a partir de `central_vault.path` (do arquivo local ou global `~/.memory/config.yaml`).
2. Se o alvo for um `repo_id` ou slug: localiza a entrada correspondente em `globalCfg.Repositories` e concatena o caminho do arquivo.
3. Verifica a existência de `<path>.md` ou `<path>`.
4. Se o vault alvo não estiver montado ou o arquivo não existir, retorna erro explicativo amigável sem interromper outras operações.

### 5. Navegação em Editores (`cmd/mem/open.go` e `internal/deeplink`)

Ao executar:
```bash
mem open "memory://central/standards/oauth2" --app obsidian
```
1. O CLI resolve a URI para o arquivo físico no Cofre Central (ex: `C:/Google Drive/Vault/standards/oauth2.md`).
2. Gera a URI nativa do editor (`obsidian://open?vault=CentralVault&file=standards/oauth2.md` ou `vscode://file/...`).
3. Dispara a abertura imediata.

### 6. Integração MCP (`internal/mcp`)

- `memory_open_node`: Aceita nós no formato `memory://<repo>/<doc>`.
- `memory_get_neighbors` e `memory_inspect_node`: Indicam se uma aresta de entrada/saída é federada (`is_federated: true`).

## Positive Consequences

- **Grafo Semântico Unificado**: O grafo agora conecta naturalmente decisões táticas locais a padrões corporativos centrais.
- **Portabilidade no Git**: Nenhuma referência a caminhos de máquina local vaza para commits de código.
- **Interoperabilidade Total**: Compatível com Obsidian Desktop, Obsidian Mobile e VS Code.
- **Resolução Automática via MCP**: Agentes de IA podem seguir referências de normas centrais diretamente a partir de notas locais.

## Negative Consequences

- **Resolução em Disco Dependente de Montagem**: Para que o `mem open` abra a nota no editor, a pasta do vault central ou do repositório alvo precisa estar montada/acessível no host da máquina local.

## Implementation References

- `internal/parser/wikilinks.go`: Suporte ao parser de URIs canônicas `memory://`.
- `internal/federation/resolver.go`: Resolução de URIs canônicas contra vaults físicos.
- `internal/federation/resolver_test.go`: Testes unitários de resolução, aliases e âncoras.
- `internal/deeplink/deeplink.go`: Formatação de URIs canônicas federadas.
- `cmd/mem/open.go`: Suporte a URIs canônicas no comando CLI `mem open`.
- `internal/mcp/handlers.go`: Suporte a navegação federada nas ferramentas MCP.
