# ADR-019: Versionamento Semântico Automatizado e Criação de Tags no CI

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: semver, conventional-commits, github-actions, ci-cd, git-tags, releases, ldflags, multiplatform, sqlite-fts5

## Context and Problem Statement

O ecossistema do **My-Memory** atingiu um elevado nível de maturidade técnica, contando com busca híbrida vetorial e full-text com Reciprocal Rank Fusion (RRF), quantização TurboQuant com AVX-512, cache incremental SHA-256, centralidade PageRank ponderada, configuração declarativa de vaults, live watcher contínuo e síntese de notas com escrita bilateral (Compile-not-Retrieve).

Apesar disso, o ciclo de entrega contínua do projeto sofria com lacunas críticas de governança e distribuição:
1. **Ausência de Git Tags:** O repositório não possuía tags de versão publicadas (`git tag -l` retornava vazio), impossibilitando rastrear quais commits compõem uma versão estável.
2. **Binário sem Metadados de Runtime:** O executável CLI (`cmd/mem`) não expunha comando `version` e não recebia flags de injeção em tempo de compilação (`Version`, `GitCommit`, `BuildDate`).
3. **Falta de Automação de Releases no CI:** O pipeline do GitHub Actions limitava-se a validar formatação e testes unitários/integração. Ao fazer push ou merge na branch `main`, não havia mecanismo determinístico para calcular o próximo incremento SemVer (Major, Minor, Patch), criar a tag correspondente e publicar os binários compilados na GitHub Release.

## Decision Drivers

- **SemVer 2.0.0 e Conventional Commits**: Mapear alterações de forma determinística: `BREAKING CHANGE:` ➔ **Major**, `feat:` ➔ **Minor**, `fix:`/`perf:`/`refactor:` ➔ **Patch**, mantendo commits não-releasáveis (`docs:`, `chore:`, `style:`, `test:`, `ci:`) sem bumps vazios.
- **Injeção de Metadados de Compilação no Go**: Injetar `-X main.Version=... -X main.GitCommit=... -X main.BuildDate=...` durante `go build` e expor via subcomando `mem version`, `-v`, `--version` e modo estruturado `--json`.
- **Pipeline de Release Seguro no GitHub Actions**: Disparar exclusivamente em `push` na branch `main` ou disparo manual auditado (`workflow_dispatch`), dependendo de testes prévios aprovados.
- **Distribuição Multiplataforma Automatizada**: Compilar binários nativos para Linux (`amd64`) e Windows (`amd64`) com suporte a CGO/SQLite FTS5, empacotando-os em `.tar.gz` e `.zip` acompanhados do arquivo de somas criptográficas `checksums.txt` (SHA-256).
- **Baseline Inicial Consciente**: Definir `v1.0.0` como marco inicial de versionamento estável devido à completude da arquitetura.

## Decision Outcome

Adotou-se uma arquitetura integrada de versionamento semântico composta pelos seguintes pilares:

1. **Camada de Metadados no Go (`cmd/mem/version.go`):**
   - Variáveis globais `Version`, `GitCommit` e `BuildDate` em `main`.
   - Struct `VersionInfo` com saída estruturada JSON (`mem version --json`).
   - Formatação legível incluindo versão Go e arquitetura de sistema (`formatVersion()`).
   - Roteamento completo para `mem version`, `mem -v` e `mem --version`.

2. **Analisador Determinístico de Commits (`scripts/release/calculate_version.py`):**
   - Inspeciona o histórico Git entre a tag mais recente (`git describe --tags --abbrev=0`) e o `HEAD`.
   - Caso não existam tags anteriores, adota `v1.0.0` como baseline e compila o changelog completo de todos os commits do repositório.
   - Aplica expressões regulares para Conventional Commits e gera notas de release categorizadas em Markdown (`Breaking Changes`, `Features`, `Bug Fixes`, `Documentation & Maintenance`).
   - Integração nativa com `$GITHUB_OUTPUT` do GitHub Actions.
   - Suíte de testes unitários dedicada em `scripts/release/test_calculate_version.py`.

3. **Pipelines de CI/CD no GitHub Actions:**
   - `.github/workflows/ci.yml`: Atualizado para executar os testes unitários do analisador Python e validar a compilação com `-ldflags` e execução do comando `mem version`.
   - `.github/workflows/release.yml`: Workflow acionado em push para `main` e `workflow_dispatch`. Realiza:
     1. Cálculo de versão e geração de release notes.
     2. Matriz de compilação multiplataforma para `linux/amd64` e `windows/amd64` com tags `sqlite_fts5`.
     3. Criação e envio da Git Tag anotada `vX.Y.Z`.
     4. Geração de `checksums.txt` (SHA256).
     5. Publicação da GitHub Release via `softprops/action-gh-release@v2`.

### Positive Consequences

- **Rastreabilidade e Confiabilidade Operacional**: Cada release possui uma tag imutável no Git e uma GitHub Release contendo changelog detalhado e binários pré-compilados.
- **Zero Atrito para Desenvolvedores e IAs**: Apenas seguir a convenção de Conventional Commits (`feat:`, `fix:`) assegura que o CI calcule e incremente a versão correta sem intervenção humana.
- **Auditoria de Integridade**: Downloads contam com somas de verificação SHA-256 (`checksums.txt`).
- **Observabilidade em Tempo de Execução**: `mem version --json` permite que ferramentas de monitoramento e agentes de IA identifiquem a versão exata do executável.

### Negative Consequences / Trade-offs

- **Exigência Estrita de Commits Convencionais**: Commits fora do padrão (`feat:`, `fix:`, etc.) não disparam bumps de versão automaticamente (mas podem ser forçados via `workflow_dispatch`).
- **Tempo de Execução no CI na Main**: A publicação de release e a matriz de build multiplataforma adicionam etapas ao pipeline na branch `main`.

---

## Links

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Conventional Commits 1.0.0](https://www.conventionalcommits.org/)
- [GitHub Actions: softprops/action-gh-release](https://github.com/softprops/action-gh-release)
- [Workflow de Release](file:///C:/repository/my-memory/.github/workflows/release.yml)
- [Script de Cálculo de Versão](file:///C:/repository/my-memory/scripts/release/calculate_version.py)
