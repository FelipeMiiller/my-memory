# Test script for scripts/install.ps1
$ErrorActionPreference = "Stop"

$TestDir = Join-Path ([System.IO.Path]::GetTempPath()) "test_mem_inst_$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"

try {
    Write-Host "Iniciando teste de scripts/install.ps1..." -ForegroundColor Cyan
    $ScriptPath = Join-Path $PSScriptRoot "install.ps1"
    
    # Executa o instalador em diretório isolado sem alterar o PATH do sistema
    & powershell -ExecutionPolicy Bypass -File $ScriptPath -Version "v1.2.0" -InstallDir $TestDir -NoPath -Force
    
    $InstalledBin = Join-Path $TestDir "mem.exe"
    if (-not (Test-Path $InstalledBin)) {
        throw "O binário mem.exe não foi encontrado em: $InstalledBin"
    }

    Write-Host "Verificando execução do binário instalado..." -ForegroundColor Cyan
    $Output = & $InstalledBin version
    Write-Host "Saída do comando version: $Output" -ForegroundColor Gray
    
    if (-not ($Output -match "My-Memory|Version|v1\.")) {
        throw "A saída do executável não continha informações de versão esperadas: $Output"
    }

    Write-Host "[OK] Teste do instalador Windows (scripts/install.ps1) passou com sucesso!" -ForegroundColor Green
    exit 0
} catch {
    Write-Host "[ERROR] Falha no teste de install.ps1: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    if (Test-Path $TestDir) {
        Remove-Item -Path $TestDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
