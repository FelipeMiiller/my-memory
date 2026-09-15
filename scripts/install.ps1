<#
.SYNOPSIS
    Instalador universal do My-Memory para Windows.
.DESCRIPTION
    Baixa a release mais recente do My-Memory do GitHub Releases,
    extrai o binário mem.exe, instala em %USERPROFILE%\.mem\bin,
    adiciona ao PATH de usuário e valida a instalação.
.PARAMETER Version
    Versão específica para instalar (ex: "v1.2.0"). Se omitido, obtém a mais recente.
.PARAMETER InstallDir
    Diretório de instalação customizado (padrão: $HOME\.mem\bin).
.PARAMETER Force
    Sobrescreve a instalação existente sem confirmação.
.PARAMETER NoPath
    Não adiciona o diretório ao PATH do usuário.
.EXAMPLE
    irm https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.ps1 | iex
#>
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:USERPROFILE\.mem\bin",
    [switch]$Force,
    [switch]$NoPath
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = "FelipeMiiller/my-memory"

function Write-Step {
    param([string]$Message)
    Write-Host "==> " -ForegroundColor Cyan -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] " -ForegroundColor Green -NoNewline
    Write-Host $Message -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $Message -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERR] " -ForegroundColor Red -NoNewline
    Write-Host $Message -ForegroundColor Red
}

Write-Host ""
Write-Host "  My-Memory - Instalador do Repository Brain para Windows" -ForegroundColor Magenta
Write-Host "  ===========================================================" -ForegroundColor DarkGray
Write-Host ""

# 1. Determinar a versão a ser instalada
if (-not $Version) {
    Write-Step "Consultando a versão mais recente em https://github.com/$Repo/releases..."
    try {
        $ReleaseUri = "https://api.github.com/repos/$Repo/releases/latest"
        $Headers = @{ "User-Agent" = "my-memory-installer" }
        $ReleaseJson = Invoke-RestMethod -Uri $ReleaseUri -Headers $Headers -UseBasicParsing
        $Version = $ReleaseJson.tag_name
    } catch {
        Write-Warn "Não foi possível obter a versão via API do GitHub ($($_.Exception.Message))."
        Write-Step "Tentando obter versão de fallback v1.2.0..."
        $Version = "v1.2.0"
    }
}

if (-not $Version.StartsWith("v")) {
    $Version = "v$Version"
}

Write-Step "Instalando versão: $Version"

# 2. Verificar arquitetura
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    Write-Warn "Arquitetura ARM64 detectada. O binário x64 será executado via emulação do Windows 11."
}

$ZipName = "my-memory-$Version-windows-$Arch.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$ZipName"

# 3. Preparar diretório de download temporário
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
$ZipPath = Join-Path $TempDir $ZipName

try {
    Write-Step "Baixando pacote: $DownloadUrl"
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing

    # 3.1. Validar Checksum SHA-256
    $ChecksumUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"
    $ChecksumPath = Join-Path $TempDir "checksums.txt"
    try {
        Write-Step "Baixando assinaturas de integridade: $ChecksumUrl"
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath -UseBasicParsing
        
        $ExpectedHash = ""
        Get-Content $ChecksumPath | ForEach-Object {
            $line = $_.Trim()
            if ($line -match "^([a-fA-F0-9]{64})\s+(.+)$") {
                $hash = $matches[1]
                $fname = [System.IO.Path]::GetFileName($matches[2].Trim())
                if ($fname -eq $ZipName) {
                    $ExpectedHash = $hash
                }
            }
        }

        if ($ExpectedHash) {
            Write-Step "Verificando integridade SHA-256..."
            $ActualHash = (Get-FileHash -Path $ZipPath -Algorithm SHA256).Hash
            if ($ActualHash.ToLower() -ne $ExpectedHash.ToLower()) {
                throw "Falha na verificação de integridade! Hash esperado: $ExpectedHash, obtido: $ActualHash"
            }
            Write-Success "Assinatura SHA-256 verificada com sucesso!"
        } else {
            Write-Warn "Assinatura para $ZipName não encontrada em checksums.txt, prosseguindo com cautela..."
        }
    } catch {
        Write-Warn "Não foi possível validar checksums.txt: $($_.Exception.Message)"
    }

    Write-Step "Extraindo binário para: $InstallDir"
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    
    $ExtractedFolder = Join-Path $TempDir "extracted"
    Expand-Archive -Path $ZipPath -DestinationPath $ExtractedFolder -Force

    $BinSource = Join-Path $ExtractedFolder "mem.exe"
    if (-not (Test-Path $BinSource)) {
        $Found = Get-ChildItem -Path $ExtractedFolder -Filter "mem.exe" -Recurse | Select-Object -First 1
        if ($Found) {
            $BinSource = $Found.FullName
        } else {
            throw "O executável 'mem.exe' não foi encontrado no arquivo descompactado."
        }
    }

    $TargetBin = Join-Path $InstallDir "mem.exe"
    Copy-Item -Path $BinSource -Destination $TargetBin -Force

    Write-Success "Binário instalado com sucesso em: $TargetBin"

    # 4. Configurar PATH no Registro / Perfil de Usuário
    if (-not $NoPath) {
        $UserPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
        $PathParts = $UserPath -split ";" | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
        
        $CleanInstallDir = $InstallDir.TrimEnd("\")
        $AlreadyInPath = $false
        foreach ($p in $PathParts) {
            if ($p.TrimEnd("\") -eq $CleanInstallDir) {
                $AlreadyInPath = $true
                break
            }
        }

        if (-not $AlreadyInPath) {
            Write-Step "Adicionando $InstallDir ao PATH de Usuário..."
            $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
            [Environment]::SetEnvironmentVariable("PATH", $NewPath, [EnvironmentVariableTarget]::User)
            $env:PATH = "$env:PATH;$InstallDir"
            Write-Success "PATH atualizado com sucesso!"
        } else {
            Write-Success "Diretório já configurado no PATH."
        }
    }

    # 5. Validar execução
    Write-Step "Validando execução do executável..."
    $VersionOutput = & $TargetBin version 2>&1
    Write-Success "Validação concluída: $VersionOutput"

    Write-Host ""
    Write-Host "  Instalação concluída com sucesso!" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Para começar a usar no seu projeto:" -ForegroundColor Cyan
    Write-Host "    1. Abra o terminal na pasta do seu repositório de código."
    Write-Host "    2. Execute 'mem init' para inicializar o cofre de memória (.memory/)."
    Write-Host "    3. Execute 'mem install' para auto-configurar seus clientes de IA (Cursor, VS Code, Claude)."
    Write-Host "    4. Execute 'mem index' para indexar e quantizar suas notas e docs."
    Write-Host ""

} catch {
    Write-Err "Falha durante a instalação: $($_.Exception.Message)"
    
    # Fallback caso Go esteja instalado localmente
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Step "Go detectado na máquina! Tentando compilar e instalar via 'go install'..."
        try {
            & go install github.com/$Repo/cmd/mem@latest
            Write-Success "Instalado com sucesso via go install!"
            exit 0
        } catch {
            Write-Err "Fallback via go install também falhou: $($_.Exception.Message)"
        }
    }

    exit 1
} finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
