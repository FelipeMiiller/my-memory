# Feature: automated-semver-and-ci-tagging

## Problem Statement
O My-Memory possui uma suíte completa de testes e integração contínua no GitHub Actions, mas atualmente carece de um ciclo automatizado de versionamento e release:
1. **Ausência de Tags Git:** Não existem tags de versão no repositório (`git tag -l` vazio), impossibilitando que usuários e agentes determinem a versão estável em execução.
2. **Ausência de Metadados no Binário:** O binário compilado `cmd/mem` não expõe comando `version` e não possui metadados injetados (`Version`, `GitCommit`, `BuildDate`).
3. **Ausência de Pipeline de Release no CI:** Quando alterações chegam na branch `main`, não há automação para analisar os Conventional Commits (`feat:`, `fix:`, `BREAKING CHANGE:`), calcular a próxima versão SemVer, criar a Git Tag correspondente e publicar a GitHub Release com binários empacotados.

## Goals
- [ ] Implementar variáveis de compilação e comando `mem version` (com flags `-v`, `--version` e opção `--json`).
- [ ] Implementar injeção de metadados de compilação via `-ldflags` em tempo de build.
- [ ] Criar workflow de CI no GitHub Actions para release e tagging automático acionado em push para a branch `main`.
- [ ] Configurar cálculo determinístico de SemVer (Major, Minor, Patch) baseado em Conventional Commits.
- [ ] Configurar publicação automatizada de GitHub Release com changelog gerado e assets binários multiplataforma (Linux e Windows) com somas SHA256.
- [ ] Registrar decisão de arquitetura na ADR-019.

## Out of Scope
- Publicação de pacotes em gerenciadores de terceiros como Homebrew, Chocolatey ou AUR (podem ser adicionados em releases futuras).
- Assinatura criptográfica Cosign de binários (escopo futuro).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Versão Inicial | `v1.0.0` | Repositório possui arquitetura madura com RRF híbrido, TurboQuant, motor declarativo e compilação de notas | y |
| Gatilho de Versão | Push na branch `main` após sucesso de todos os testes | Garante que nenhuma tag seja criada se houver falhas em testes ou compilação | y |
| Mapeamento Conventional Commits | `BREAKING CHANGE:` -> Major, `feat:` -> Minor, `fix:`/`perf:`/`refactor:` -> Patch | Segue a especificação estrita do Semantic Versioning 2.0.0 | y |
| Commits Não-Releasáveis | `docs:`, `chore:`, `style:`, `test:`, `ci:` não geram nova tag por padrão | Evita releases vazias e poluição desnecessária de tags no Git | y |
| Binários da Release | Compilação para `linux/amd64` e `windows/amd64` com tags `sqlite_fts5` | Cobre os principais ambientes de servidores e estações de trabalho do projeto | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Metadados de Compilação e Subcomando mem version ⭐ MVP

**User Story**: As a usuário ou operador do My-Memory, I want consultar a versão exata, commit Git e data de compilação do binário so that eu possa validar compatibilidade e diagnosticar comportamentos em ambientes locais ou de produção.

**Why P1**: Fornece observabilidade direta do binário compilado e base para a injeção via pipeline de CI.

**Acceptance Criteria**:
1. The system SHALL define `Version`, `GitCommit` and `BuildDate` variables in package `main` capable of being overridden via `-ldflags`.
2. WHEN `mem version`, `mem --version` or `mem -v` is invoked THEN the system SHALL output human-readable version details.
3. WHEN `mem version --json` is invoked THEN the system SHALL output valid JSON containing version, commit, build_date, go_version, os and arch.

**Independent Test**: Testes unitários em `cmd/mem/version_test.go` validando a formatação de texto e JSON.

---

### P2: Automação de Tagging SemVer e Release no GitHub Actions

**User Story**: As a mantenedor do repositório, I want que o CI analise os commits ao mesclar na branch `main`, calcule a próxima versão SemVer e crie uma Git Tag e Release no GitHub so that o ciclo de entrega contínua seja 100% automatizado e livre de erros manuais.

**Why P2**: Elimina trabalho manual e garante rastreabilidade estrita entre o código na branch `main` e os artefatos publicados.

**Acceptance Criteria**:
1. The CI pipeline SHALL trigger on push to `main` only after linting, unit tests and database tests succeed.
2. The CI pipeline SHALL inspect git commits since the last tag and calculate the appropriate SemVer bump (Major, Minor, Patch).
3. The CI pipeline SHALL create and push an annotated Git Tag `vX.Y.Z`.
4. The CI pipeline SHALL compile binaries for Linux and Windows injecting version metadata.
5. The CI pipeline SHALL publish a GitHub Release containing categorized release notes and binary assets with checksums.

**Independent Test**: Validação sintática do workflow GitHub Actions e testes de compilação com `-ldflags`.

---

## Requirements (EARS)

### TAG-01: Injeção e Exibição de Versão no CLI
The system SHALL provide variables `Version`, `GitCommit` and `BuildDate` in `cmd/mem`, defaulting to `"dev"`, `"none"` and `"unknown"`, and expose them via `mem version`, `mem --version` and `mem -v`.
- User story: P1
- Task: T1

### TAG-02: Saída Estruturada de Versão em JSON
WHEN `mem version --json` is executed THEN the system SHALL print a valid JSON payload with fields `version`, `git_commit`, `build_date`, `go_version`, `os` and `arch`.
- User story: P1
- Task: T1

### TAG-03: Cálculo Determinístico de Bump SemVer no CI
WHEN a push event occurs on branch `main` THEN the CI pipeline SHALL determine the latest tag or fallback to `v1.0.0` as baseline, and calculate the next SemVer version based on Conventional Commits keywords (`BREAKING CHANGE:`, `feat:`, `fix:`).
- User story: P2
- Task: T2

### TAG-04: Criação Automatizada de Git Tag
WHEN commits qualify for a new release THEN the CI pipeline SHALL create and push an annotated Git tag formatted as `v<MAJOR>.<MINOR>.<PATCH>`.
- User story: P2
- Task: T2

### TAG-05: Compilação e Empacotamento de Binários Multiplataforma
WHEN a new release tag is created THEN the CI pipeline SHALL compile the `mem` binary for `linux/amd64` and `windows/amd64` with embedded `-ldflags` and generate SHA256 checksums.
- User story: P2
- Task: T3

### TAG-06: Publicação de GitHub Release com Changelog
WHEN release artifacts are ready THEN the CI pipeline SHALL create a GitHub Release associated with the Git tag containing categorized release notes and the compiled binary archives.
- User story: P2
- Task: T3

---

## Requirement Traceability

| Requirement ID | User Story | Status | Test / Validation |
| -------------- | ---------- | ------ | ----------------- |
| TAG-01 | P1: Metadados de Compilação e Subcomando mem version | Verified | Tasks |
| TAG-02 | P1: Metadados de Compilação e Subcomando mem version | Verified | Tasks |
| TAG-03 | P2: Automação de Tagging SemVer e Release no GitHub Actions | Verified | Tasks |
| TAG-04 | P2: Automação de Tagging SemVer e Release no GitHub Actions | Verified | Tasks |
| TAG-05 | P2: Automação de Tagging SemVer e Release no GitHub Actions | Verified | Tasks |
| TAG-06 | P2: Automação de Tagging SemVer e Release no GitHub Actions | Verified | Tasks |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Subcomando `mem version` e flags `-v` e `--version` funcionais com saída de texto e JSON.
- [ ] Metadados `Version`, `GitCommit` e `BuildDate` injetáveis via `-ldflags` em tempo de compilação.
- [ ] Workflow GitHub Actions (`release.yml`) validado e integrado com Conventional Commits.
- [ ] Compilação de binários multiplataforma e geração de somas de verificação SHA256.
- [ ] Decisão registrada na ADR-019 e documentação atualizada.
