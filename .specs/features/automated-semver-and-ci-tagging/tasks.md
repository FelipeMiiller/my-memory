# Tasks: automated-semver-and-ci-tagging

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| CLI Version Core | unit | formatVersion, runVersionCLI, JSON output, flag handling | cmd/mem/*_test.go | go test -v ./cmd/mem/... |
| CI Pipeline | integration | GitHub Actions workflow YAML linting, step sequence | .github/workflows/*.yml | python -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" |
| Cross Compilation | build | Linux/Windows binary build with -ldflags | bin/mem* | go build -ldflags "-X main.Version=v1.0.0" -o bin/mem.exe ./cmd/mem |
| Documentation / ADR | none | Documentation, ADR-019, README updates | docs/adr/* | go test -v ./cmd/mem/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Version Unit | After version CLI changes | go test -v -run TestVersion ./cmd/mem/... |
| Build Verification | After ldflags integration | go build -ldflags "-X main.Version=v1.0.0 -X main.GitCommit=testsha -X main.BuildDate=2026-09-13T00:00:00Z" -o bin/mem.exe ./cmd/mem |
| Full Test | Before workflow commits | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Spec Gate | Before completing feature | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/automated-semver-and-ci-tagging/spec.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4
```

### Phase 1: Versão no Go, CI Tagging e Automação de Release

## Task Breakdown

### T1: Metadados de Compilação e Subcomando mem version
**What**: Implementar variáveis de compilação, formatação legível, modo JSON e subcomando mem version em cmd/mem
**Where**: cmd/mem/version.go, cmd/mem/version_test.go, cmd/mem/main.go
**Depends on**: none
**Requirement**: TAG-01, TAG-02
**Tests**: cmd/mem/version_test.go
**Gate**: go test -v -run TestVersion ./cmd/mem/...
**Done when**:
- [x] Declarar Version, GitCommit e BuildDate em cmd/mem
- [x] Implementar formatVersion() gerando texto amigável com arquitetura e versão Go
- [x] Implementar flag --json para emitir saída estruturada
- [x] Conectar subcomando version e flags -v / --version no roteador CLI de cmd/mem/main.go
- [x] Adicionar testes unitários em cmd/mem/version_test.go

### T2: Análise de Conventional Commits e Workflow de Tagging
**What**: Implementar workflow GitHub Actions de tagging automático baseado em Conventional Commits
**Where**: .github/workflows/release.yml
**Depends on**: T1
**Requirement**: TAG-03, TAG-04
**Tests**: .github/workflows/release.yml
**Gate**: python -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"
**Done when**:
- [x] Criar workflow acionado em push para branch main dependente do sucesso de testes
- [x] Implementar inspeção determinística de Conventional Commits (BREAKING CHANGE: -> major, feat: -> minor, fix: -> patch)
- [x] Configurar baseline inicial para v1.0.0 na ausência de tags prévias
- [x] Automatizar criação e push da tag Git anotada vX.Y.Z com permissões adequadas

### T3: Matriz de Build Multiplataforma e Publicação de GitHub Release
**What**: Configurar build cruzado com ldflags e publicação de release com changelog e somas SHA256
**Where**: .github/workflows/release.yml, .github/workflows/ci.yml
**Depends on**: T2
**Requirement**: TAG-05, TAG-06
**Tests**: .github/workflows/release.yml
**Gate**: go test -v ./cmd/mem/...
**Done when**:
- [x] Adicionar etapa de compilação cruzada para linux/amd64 e windows/amd64 injetando -ldflags
- [x] Gerar arquivos compactados (.tar.gz e .zip) e checksums SHA256
- [x] Publicar GitHub Release oficial com release notes geradas a partir dos commits
- [x] Garantir que commits não-releasáveis (ex: docs, chore) não gerem releases redundantes

### T4: ADR-019, Documentação e Fechamento TLC
**What**: Registrar decisão de arquitetura ADR-019, atualizar guias e validar conformidade TLC
**Where**: docs/adr/019-versionamento-semantico-e-tagging-ci.md, docs/CLI_GUIDE.md, README.md, .specs/STATE.md
**Depends on**: T3
**Requirement**: TAG-01, TAG-02, TAG-03, TAG-04, TAG-05, TAG-06
**Tests**: none
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/automated-semver-and-ci-tagging/spec.md && python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/automated-semver-and-ci-tagging/tasks.md
**Done when**:
- [ ] Criar ADR-019 no formato MADR documentando SemVer, regras de commits e pipeline de CI
- [ ] Atualizar docs/CLI_GUIDE.md com comandos de versão e flags
- [ ] Atualizar README.md com badge/seção de versionamento e ciclo de release
- [ ] Atualizar .specs/STATE.md registrando AD-019 e handoff
- [ ] Validar todos os gates TLC com zero erros
