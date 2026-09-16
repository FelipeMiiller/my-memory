# Validation: universal-binary-installer

Result: PASS

## Evidence Summary

| Requisito | Status | Evidência de Código / Teste |
| :--- | :--- | :--- |
| **INST-01** (Instalador PowerShell Windows) | PASS | `scripts/install.ps1:20` e `scripts/test_install.ps1:1` |
| **INST-02** (Instalador POSIX Shell Linux/macOS) | PASS | `scripts/install.sh:1` e `scripts/test_install.py:44` |
| **INST-03** (Verificação Criptográfica SHA-256) | PASS | `scripts/install.ps1:100`, `scripts/install.sh:140` e `scripts/test_install.py:65` |
| **INST-04** (Fallback Automático Go Toolchain) | PASS | `scripts/install.ps1:197`, `scripts/install.sh:128` e `scripts/test_install.py:40` |
| **INST-05** (Documentação e Decisão ADR-032) | PASS | `docs/adr/032-instalador-universal-one-liner.md:1`, `README.md:78` e `COMO_USAR.md:40` |

---

## 1. Testes Automatizados Executados

### Instalador Windows PowerShell (`scripts/install.ps1`)
- [x] **Download e Extração Real**: Execução isolada em diretório temporário baixando a release oficial `v1.2.0` (`scripts/test_install.ps1`).
- [x] **Integridade Criptográfica SHA-256**: Validação rigorosa do hash do arquivo `.zip` contra `checksums.txt` emitido na release do GitHub Actions.
- [x] **Execução Sem Privilégios de Administrador**: Destino padrão configurado em `$HOME\.mem\bin` (`[EnvironmentVariableTarget]::User`).
- [x] **Execução Funcional**: Validação da saída do binário extraído (`mem version`).

### Instalador POSIX Shell Linux & macOS (`scripts/install.sh`)
- [x] **Análise Estática e Portabilidade**: Shebang `#!/bin/sh`, flag fail-fast `set -e`, flags `--version`, `--dir`, `--no-path`, `--force`, `--help` (`scripts/test_install.py:44`).
- [x] **Detecção de Sistema Operacional e CPU**: Mapeamento de `uname -s` (`Linux` -> `linux`, `Darwin` -> `darwin`) e `uname -m` (`x86_64` -> `amd64`, `arm64` -> `arm64`).
- [x] **Hashing Criptográfico**: Suporte resiliente com `sha256sum`, `shasum -a 256` ou `openssl dgst -sha256`.
- [x] **Execução Funcional**: Validação de flags via interpretador POSIX `sh` com retorno de código de erro determinístico (`scripts/test_install.py:80`).

### Validação de Assets e Checksums Remotos
- [x] **Formato do Manifesto Oficial**: Verificação remota de integridade do arquivo `checksums.txt` da release `v1.2.0`, comprovando 2 hashes SHA-256 hexadecimais de 64 caracteres válidos (`scripts/test_install.py:65`).

---

## 2. Documentação e Governança

- [x] **ADR-032 Aceito**: Registrado em `docs/adr/032-instalador-universal-one-liner.md` no padrão MADR, indexado em `docs/adr/README.md` e `docs/README.md`.
- [x] **Quickstart Atualizado**: Seção "⚡ Instalação Rápida (1 Comando)" adicionada em destaque no `README.md`.
- [x] **Guia Prático Atualizado**: Seção de instalação rápida e pré-requisitos atualizada em `COMO_USAR.md`.
- [x] **Guia da CLI Atualizado**: Detalhamento dos parâmetros e opções dos instaladores documentados em `docs/CLI_GUIDE.md`.
- [x] **Conformidade de Estado**: Registrado `AD-032` e snapshot de handoff em `.specs/STATE.md`.
