# ADR-016: Configuração Declarativa e Auto-Scoping de Vault

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: config, yaml, cli, auto-scoping, glob, vault, mcp

## Context and Problem Statement

Anteriormente no My-Memory, parâmetros operacionais como localização do banco de dados (`--db memory.db`), slug do repositório (`--repo <slug>`), endpoint PostgreSQL (`--postgres <url>`), modos de busca e filtragem de arquivos dependiam exclusivamente de flags repetitivas na linha de comando ou de variáveis globais de ambiente (`MY_MEMORY_PG_URL`, `MY_MEMORY_REPO`).

Essa abordagem causava fricção em fluxos de trabalho do mundo real:
1. **Fadiga de Comandos**: O usuário ou agente precisava digitar repetidamente argumentos extensos em cada invocação de `mem index`, `mem search`, `mem doctor` ou `mem mcp`.
2. **Poluição na Varredura**: Pastas irrelevantes ou de sistema (`node_modules/`, `.git/`, `.obsidian/`, `vendor/`, `temp_*`) exigiam limpeza manual ou podiam ser lidas indevidamente caso o usuário executasse o indexador na raiz de um projeto monolítico.
3. **Falta de Auto-Scoping**: Ao executar `mem index` em um subdiretório de notas dentro de um projeto, a ferramenta não descobria automaticamente as fronteiras do vault nem as regras declaradas pelo time.

Havia a necessidade de um sistema de **configuração declarativa versionada** por projeto/vault com descoberta ascendente automática e filtragem flexível por padrões glob.

## Decision Drivers

- **Zero Configuração Inicial**: O sistema deve manter funcionamento autônomo imediato via `DefaultConfig()`.
- **Descoberta Ascendente Inteligente**: Capacidade de subir pastas até encontrar `.memory/config.yaml` ou `.mem.yaml`, interrompendo na raiz do repositório Git.
- **Auto-Scoping de Comandos**: Executar `mem index` sem argumentos deve inferir a raiz do vault e parâmetros configurados.
- **Hierarquia Estrita de Precedência**: Flags de CLI > Variáveis de Ambiente > Config do Vault > Defaults de Código.
- **Filtragem Eficiente com Padrões Glob**: Suporte a padrões `**/*.md`, diretórios ignorados e salto acelerado (`filepath.SkipDir`).
- **Template Amigável**: Subcomando `mem init` para criar uma configuração auto-explicativa pronta para uso.

## Decision Outcome

Adotou-se o motor de **Configuração Declarativa e Filtragem Glob** no pacote `internal/config` integrado universalmente ao CLI `cmd/mem`:

1. **Estrutura Declarativa (`internal/config/config.go`):**
   - Formato primário em YAML (`.memory/config.yaml`, `.memory/config.yml`, `.mem.yaml`, `.mem.yml`) com suporte a JSON (`.mem.json`).
   - Seções especializadas: `storage` (engine SQLite/Postgres), `embedding` (provedor Ollama, modelo, dimensão) e `search` (modo padrão, limit, k, decay, meia-vida, TurboQuant).
   - Função `FindConfigFile(startDir)` com busca ascendente recursiva até a raiz do repositório `.git` ou raiz do filesystem.
   - Função `LoadConfig(path)` com validação sintática e preenchimento garantido de defaults seguros.

2. **Motor de Filtragem Glob (`internal/config/glob.go`):**
   - Função `ShouldIndex(relPath)` combinando listas de `Include` e `Exclude`.
   - Normalização automática de separadores (`/` e `\`) garantindo total compatibilidade multiplataforma (Windows e Unix).
   - Diretórios de sistema ignorados por padrão: `.git`, `node_modules`, `vendor`, `.obsidian`, `.trash`, `.memory`.
   - Conversão de padrões glob em expressões regulares com cache thread-safe (`GlobToRegex`).

3. **Subcomando `mem init` (`cmd/mem/main.go`):**
   - `mem init [--repo <slug>] [--db <caminho>] [--force] [<pasta>]`.
   - Cria o diretório `.memory/` e gera o arquivo `config.yaml` amplamente documentado com comentários.
   - Proteção contra sobrescrita acidental, exigindo flag `--force` caso o arquivo já exista.

4. **Integração no CLI (`cmd/mem/main.go`):**
   - `mem index`: Se nenhuma pasta for passada, descobre automaticamente a raiz do vault configurado. Durante a varredura (`filepath.WalkDir`), aplica `filepath.SkipDir` em diretórios excluídos para ganho de performance drástico.
   - `mem search`, `mem mcp`, `mem export`, `mem hubs`, `mem insights`, `doctor`: Aplicam fallbacks transparentes do arquivo de configuração para backend de storage, repositório e parâmetros de ranking caso as flags de linha de comando sejam omitidas.

### Positive Consequences

- **Experiência do Desenvolvedor Superior**: Executar `mem init` e depois apenas `mem index` e `mem search "pergunta"` sem necessidade de decorar dezenas de flags.
- **Indexação Limpa e Veloz**: Diretórios pesados como `node_modules` são ignorados no nível da árvore antes da leitura dos arquivos em disco.
- **Padronização de Equipe**: A configuração pode ser comitada no controle de versão (`.memory/config.yaml`), garantindo que todos os membros da equipe e agentes de IA utilizem os mesmos parâmetros e pesos de busca.

### Negative Consequences / Trade-offs

- **Resolução de Caminho Adicional**: Uma pequena varredura de filesystem (`FindConfigFile`) é realizada ao inicializar comandos da CLI (< 1 ms).
