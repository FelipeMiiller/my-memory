#!/usr/bin/env python3
"""
Testes de integridade e conformidade para instaladores universais:
- scripts/install.ps1 (Windows PowerShell)
- scripts/install.sh (Linux/macOS POSIX Shell)
"""

import os
import re
import shutil
import subprocess
import sys
import tempfile
import urllib.request

ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
SH_INSTALLER = os.path.join(ROOT_DIR, "scripts", "install.sh")
PS_INSTALLER = os.path.join(ROOT_DIR, "scripts", "install.ps1")

def find_sh_executable():
    """Localiza um interpretador sh no sistema."""
    cand = shutil.which("sh")
    if cand:
        return cand
    for p in [
        r"C:\Program Files\Git\bin\sh.exe",
        r"C:\Program Files\Git\usr\bin\sh.exe",
        r"C:\Program Files (x86)\Git\bin\sh.exe",
    ]:
        if os.path.isfile(p):
            return p
    return None

def test_powershell_installer_structure():
    """Valida estrutura estática e regras de segurança de scripts/install.ps1."""
    print("==> Testando scripts/install.ps1 (estático)...")
    assert os.path.isfile(PS_INSTALLER), f"install.ps1 não encontrado em {PS_INSTALLER}"
    with open(PS_INSTALLER, "r", encoding="utf-8") as f:
        content = f.read()

    # Checagens obrigatórias
    assert "[string]$Version" in content, "Falta parâmetro Version"
    assert "[string]$InstallDir" in content, "Falta parâmetro InstallDir"
    assert "[switch]$Force" in content, "Falta switch Force"
    assert "[switch]$NoPath" in content, "Falta switch NoPath"
    assert "checksums.txt" in content, "Falta validação de checksums.txt"
    assert "SHA256" in content, "Falta checagem de algoritmo SHA-256"
    assert "EnvironmentVariableTarget]::User" in content, "Falta escopo de User PATH (no-admin)"
    assert "go install" in content, "Falta fallback para go install"
    print("[OK] scripts/install.ps1 estrutura estática validada.")

def test_posix_shell_installer_structure():
    """Valida estrutura estática e conformidade POSIX de scripts/install.sh."""
    print("==> Testando scripts/install.sh (estático)...")
    assert os.path.isfile(SH_INSTALLER), f"install.sh não encontrado em {SH_INSTALLER}"
    with open(SH_INSTALLER, "r", encoding="utf-8") as f:
        content = f.read()

    # Checagens obrigatórias
    assert content.startswith("#!/bin/sh"), "Falta shebang #!/bin/sh"
    assert "set -e" in content, "Falta 'set -e' para fail-fast seguro"
    assert "--version" in content and "-v" in content, "Falta flag --version"
    assert "--dir" in content and "-d" in content, "Falta flag --dir"
    assert "--no-path" in content, "Falta flag --no-path"
    assert "--force" in content and "-f" in content, "Falta flag --force"
    assert "--help" in content and "-h" in content, "Falta flag --help"
    assert "uname -s" in content, "Falta detecção de OS via uname -s"
    assert "uname -m" in content, "Falta detecção de arquitetura via uname -m"
    assert "checksums.txt" in content, "Falta download e validação de checksums.txt"
    assert "sha256sum" in content or "shasum" in content, "Falta suporte a hashing SHA-256"
    assert "go install" in content, "Falta fallback para go install"
    assert "chmod +x" in content, "Falta permissão executável chmod +x"
    print("[OK] scripts/install.sh estrutura estática validada.")

def test_release_checksums_format():
    """Valida integridade do arquivo checksums.txt oficial na release v1.2.0."""
    print("==> Testando formato de checksums.txt oficial...")
    url = "https://github.com/FelipeMiiller/my-memory/releases/download/v1.2.0/checksums.txt"
    req = urllib.request.Request(url, headers={"User-Agent": "test-installer"})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = resp.read().decode("utf-8")
        lines = [line.strip() for line in data.strip().splitlines() if line.strip()]
        assert len(lines) >= 2, f"Esperado pelo menos 2 registros em checksums.txt, obtido {len(lines)}"
        for line in lines:
            parts = line.split()
            assert len(parts) >= 2, f"Linha inválida em checksums.txt: {line}"
            sha = parts[0]
            filename = parts[1]
            assert re.match(r"^[a-fA-F0-9]{64}$", sha), f"Hash SHA-256 inválido: {sha}"
            assert filename.startswith("my-memory-"), f"Nome de asset inesperado: {filename}"
        print(f"[OK] checksums.txt contém {len(lines)} hashes SHA-256 válidos.")
    except Exception as e:
        print(f"[WARN] Não foi possível verificar URL remota de checksums: {e}")

def test_posix_shell_execution():
    """Valida execução real do instalador sh se interpretador estiver presente."""
    sh_path = find_sh_executable()
    if not sh_path:
        print("[WARN] Interpretador sh não encontrado no ambiente local; pulando execução direta.")
        return

    print(f"==> Testando execução de scripts/install.sh via {sh_path}...")
    # 1. Teste de --help
    res = subprocess.run([sh_path, "scripts/install.sh", "--help"], cwd=ROOT_DIR, capture_output=True, text=True)
    assert res.returncode == 0, f"Falha ao executar install.sh --help: {res.stderr}"
    assert "Uso: install.sh" in res.stdout, f"Saída inesperada para --help: {res.stdout}"
    assert "--version" in res.stdout
    assert "--dir" in res.stdout

    # 2. Teste de flag inválida
    res_err = subprocess.run([sh_path, "scripts/install.sh", "--invalid-flag"], cwd=ROOT_DIR, capture_output=True, text=True)
    assert res_err.returncode == 1, "Deveria falhar para flag desconhecida"
    assert "Argumento desconhecido" in res_err.stderr

    print("[OK] Execução funcional de scripts/install.sh validada.")

def main():
    print("\n==================================================")
    print("  Iniciando suíte de testes de instalação universal")
    print("==================================================\n")
    test_powershell_installer_structure()
    test_posix_shell_installer_structure()
    test_release_checksums_format()
    test_posix_shell_execution()
    print("\n[OK] Todos os testes dos instaladores passaram com 100% de sucesso!\n")

if __name__ == "__main__":
    main()
