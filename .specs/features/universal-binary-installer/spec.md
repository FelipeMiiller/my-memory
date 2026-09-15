# Feature: universal-binary-installer

## Problem Statement
Atualmente, para utilizar o `my-memory`, desenvolvedores e agentes precisam compilar o binário manualmente via Go (`go build ./cmd/mem` ou `go install`) ou baixar manualmente os arquivos compactados (`.zip` ou `.tar.gz`) da página de Releases do GitHub.

Esse processo cria atrito significativo na adoção (DevX), especialmente para usuários que não possuem o toolchain do Go instalado em suas máquinas ou que desejam inicializar o `my-memory` com um único comando em seus ambientes de desenvolvimento, containers ou pipelines de CI/CD.

A funcionalidade **Universal Binary Installer** resolve esse problema ao disponibilizar instaladores automatizados de 1 linha (*One-Liner Installers*) para **Windows** (PowerShell) e **Linux/macOS** (POSIX Shell). Os instaladores detectam arquitetura e sistema operacional, resolvem a versão mais recente via GitHub Releases, validam o checksum criptográfico SHA-256, extraem o executável para uma pasta de usuário isolada (sem exigir privilégios de administrador/root), configuram o `PATH` do sistema e validam a instalação executando `mem version`.

---

## Out of Scope
- Criação de instaladores gráficos MSI / PKG com assistentes de interface (GUI wizards).
- Envio imediato para repositórios centrais de terceiros que exigem revisão externa de mantenedores (ex: `winget-pkgs`, `homebrew-core`), ficando o foco nos scripts oficiais do repositório.
- Suporte a sistemas operacionais obsoletos sem suporte oficial a Go (ex: Windows 7, Linux com glibc < 2.17).

---

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Instalação sem Privilégios de Administrador | `$HOME/.mem/bin` (Windows) e `$HOME/.local/bin` (Linux/macOS) | Elimina a necessidade de `sudo` ou elevação de UAC, funcionando em ambientes corporativos e com menor risco de segurança. |
| Resolução de Versão | Consulta à API do GitHub `/releases/latest` com fallback seguro para tag estável | Garante que o usuário sempre receba a versão mais recente, com resiliência a rate limits da API do GitHub. |
| Verificação de Integridade | Validação de hash SHA-256 contra `checksums.txt` da release | Impede a execução de downloads corrompidos ou adulterados em trânsito. |
| Fallback para Go Toolchain | Execução de `go install github.com/FelipeMiiller/my-memory/cmd/mem@latest` caso o download falhe e o Go esteja instalado | Fornece alternativa automática e transparente se o GitHub Releases estiver inacessível. |
| Modificação do PATH | Persistência no escopo de Usuário (Registro Windows ou `~/.bashrc`/`~/.zshrc`) com opção de desativação (`--no-path`) | Garante que o comando `mem` fique acessível no terminal imediatamente e após reinicializações. |

Open questions: none (todas as premissas técnicas e de segurança foram acordadas e detalhadas).

---

## User Stories

- **US1**: Como desenvolvedor no Windows, quero executar `irm https://.../install.ps1 | iex` no PowerShell para ter o `mem.exe` instalado e pronto no meu PATH em segundos, sem precisar instalar Go.
- **US2**: Como desenvolvedor no Linux ou macOS, quero executar `curl -fsSL https://.../install.sh | sh` no terminal para baixar a versão compatível com minha arquitetura e instalá-la em `~/.local/bin`.
- **US3**: Como engenheiro de DevOps ou agente de IA, quero poder especificar uma versão fixa (ex: `-Version v1.2.0` ou `--version v1.2.0`) e desativar alterações de PATH (`--no-path`) para instalações automatizadas e determinísticas.

---

## Requirements (EARS Notation)

### INST-01: Instalador PowerShell para Windows (`scripts/install.ps1`)
- **WHEN**: When executed on Windows via PowerShell, the system SHALL download the appropriate binary archive from GitHub Releases for the detected architecture.
- **UBIQUITOUS**: The system SHALL extract `mem.exe` to `$env:USERPROFILE\.mem\bin` without requiring administrator privileges.
- **WHERE**: Where the `--no-path` switch is not passed, the system SHALL add the installation directory to the User PATH environment variable and current session.
- **UBIQUITOUS**: The system SHALL verify the installation by executing the installed binary's version command.

### INST-02: Instalador POSIX Shell para Linux e macOS (`scripts/install.sh`)
- **WHEN**: When executed on Linux or macOS via POSIX shell, the system SHALL detect the operating system and architecture (`amd64`, `arm64`).
- **UBIQUITOUS**: The system SHALL download and extract the `mem` binary into `$HOME/.local/bin` (or `/usr/local/bin` if running as root) with executable permissions (`chmod +x`).
- **WHERE**: Where `$HOME/.local/bin` is not in `$PATH`, the system SHALL print clear export instructions or offer non-destructive configuration in the user's shell profile.
- **UBIQUITOUS**: The system SHALL verify the installation by executing the installed binary's version command.

### INST-03: Verificação Criptográfica de Integridade (SHA-256)
- **WHEN**: When the release archive is downloaded, the system SHALL download `checksums.txt` from the release assets and verify the computed SHA-256 hash before extraction.
- **WHERE**: Where the computed hash does not match the checksums record, the system SHALL abort installation with an error and delete temporary files.

### INST-04: Fallback Automático para Go Toolchain
- **WHERE**: Where binary download fails and a working `go` compiler is detected in the environment, the system SHALL attempt installation via `go install github.com/FelipeMiiller/my-memory/cmd/mem@latest` as a fallback.

### INST-05: Documentação e Decisão de Arquitetura (ADR-032)
- **UBIQUITOUS**: The system SHALL document the one-liner installation commands in `README.md` and `COMO_USAR.md`.
- **UBIQUITOUS**: The system SHALL register ADR-032 in `docs/adr/` capturing the design decisions and security model.

---

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| INST-01 | Instalador PowerShell para Windows (`scripts/install.ps1`) | in tasks |
| INST-02 | Instalador POSIX Shell para Linux e macOS (`scripts/install.sh`) | in tasks |
| INST-03 | Verificação Criptográfica de Integridade (SHA-256) | in tasks |
| INST-04 | Fallback Automático para Go Toolchain | in tasks |
| INST-05 | Documentação e Decisão de Arquitetura (ADR-032) | in tasks |
