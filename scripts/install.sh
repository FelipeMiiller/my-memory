#!/bin/sh
# Instalador Universal One-Liner para Linux e macOS do My-Memory CLI (mem).
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.sh | sh
# ou:
#   wget -qO- https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.sh | sh
#
# Opções via variáveis de ambiente ou argumentos:
#   VERSION="v1.2.0"  (ou --version v1.2.0)
#   INSTALL_DIR="/custom/path" (ou --dir /custom/path)
#   NO_PATH=1         (ou --no-path)
#   FORCE=1           (ou --force)

set -e

REPO="FelipeMiiller/my-memory"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"
NO_PATH="${NO_PATH:-0}"
FORCE="${FORCE:-0}"

# Cores ANSI
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

step() {
    printf "${CYAN}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}[OK]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

err() {
    printf "${RED}[ERR]${NC} %s\n" "$1" >&2
}

# Processamento de argumentos
while [ $# -gt 0 ]; do
    case "$1" in
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --dir|-d)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --no-path)
            NO_PATH=1
            shift
            ;;
        --force|-f)
            FORCE=1
            shift
            ;;
        --help|-h)
            echo "Uso: install.sh [opções]"
            echo "Opções:"
            echo "  -v, --version <tag>   Especifica a versão a instalar (ex: v1.2.0)"
            echo "  -d, --dir <caminho>   Diretório de destino da instalação"
            echo "      --no-path         Não exibe nem sugere inclusão no PATH"
            echo "  -f, --force           Sobrescreve binário existente sem perguntar"
            echo "  -h, --help            Exibe esta ajuda"
            exit 0
            ;;
        *)
            err "Argumento desconhecido: $1"
            exit 1
            ;;
    esac
done

printf "\n"
printf "${MAGENTA}  My-Memory - Instalador do Repository Brain para Linux/macOS${NC}\n"
printf "  ===========================================================\n\n"

# 1. Detectar Sistema Operacional
OS="$(uname -s)"
case "$OS" in
    Linux*)     OS="linux" ;;
    Darwin*)    OS="darwin" ;;
    *)
        err "Sistema operacional não suportado diretamente: $OS"
        exit 1
        ;;
esac

# 2. Detectar Arquitetura de CPU
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    arm64|aarch64)  ARCH="arm64" ;;
    *)
        err "Arquitetura de hardware não suportada diretamente: $ARCH"
        exit 1
        ;;
esac

step "Plataforma detectada: ${OS}/${ARCH}"

# 3. Determinar Diretório de Instalação Padrão
if [ -z "$INSTALL_DIR" ]; then
    if [ "$(id -u)" = "0" ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
    fi
fi

# Função auxiliar para download compatível com curl ou wget
download() {
    url="$1"
    output="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$output" "$url"
    else
        err "Necessário 'curl' ou 'wget' instalado para realizar o download."
        exit 1
    fi
}

# 4. Determinar a Versão
if [ -z "$VERSION" ]; then
    step "Consultando a versão mais recente em https://github.com/${REPO}/releases..."
    LATEST_JSON=""
    if command -v curl >/dev/null 2>&1; then
        LATEST_JSON=$(curl -fsSL -H "User-Agent: my-memory-installer" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
    elif command -v wget >/dev/null 2>&1; then
        LATEST_JSON=$(wget -qO- --header="User-Agent: my-memory-installer" "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
    fi

    if [ -n "$LATEST_JSON" ]; then
        VERSION=$(printf "%s" "$LATEST_JSON" | grep -o '"tag_name": *"[^"]*"' | head -n 1 | sed 's/"tag_name": *"//;s/"//' || true)
    fi

    if [ -z "$VERSION" ]; then
        warn "Não foi possível obter a versão via API do GitHub. Usando versão de fallback v1.2.0..."
        VERSION="v1.2.0"
    fi
fi

case "$VERSION" in
    v*) ;;
    *)  VERSION="v${VERSION}" ;;
esac

step "Instalando versão: ${VERSION}"

TAR_NAME="my-memory-${VERSION}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TAR_NAME}"

# 5. Criar diretório temporário isolado
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'mem-install')"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

TAR_PATH="${TMP_DIR}/${TAR_NAME}"

step "Baixando pacote: ${DOWNLOAD_URL}"
if ! download "$DOWNLOAD_URL" "$TAR_PATH"; then
    err "Falha ao baixar ${DOWNLOAD_URL}"
    if command -v go >/dev/null 2>&1; then
        step "Go detectado no sistema! Tentando fallback via 'go install'..."
        if go install "github.com/${REPO}/cmd/mem@latest"; then
            success "Instalado com sucesso via 'go install'!"
            exit 0
        fi
    fi
    exit 1
fi

# 6. Validação de Integridade SHA-256
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
CHECKSUM_PATH="${TMP_DIR}/checksums.txt"

step "Baixando assinaturas de integridade: ${CHECKSUM_URL}"
if download "$CHECKSUM_URL" "$CHECKSUM_PATH" 2>/dev/null; then
    EXPECTED_HASH=""
    if [ -f "$CHECKSUM_PATH" ]; then
        EXPECTED_HASH=$(grep -E "${TAR_NAME}\$" "$CHECKSUM_PATH" | awk '{print $1}' | head -n 1 || true)
        if [ -z "$EXPECTED_HASH" ]; then
            # Tenta sem caminho caso esteja gravado como basename
            EXPECTED_HASH=$(awk -v fname="$TAR_NAME" '$2 ~ fname {print $1}' "$CHECKSUM_PATH" | head -n 1 || true)
        fi
    fi

    if [ -n "$EXPECTED_HASH" ]; then
        step "Verificando integridade SHA-256..."
        ACTUAL_HASH=""
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL_HASH=$(sha256sum "$TAR_PATH" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL_HASH=$(shasum -a 256 "$TAR_PATH" | awk '{print $1}')
        elif command -v openssl >/dev/null 2>&1; then
            ACTUAL_HASH=$(openssl dgst -sha256 "$TAR_PATH" | awk '{print $NF}')
        fi

        if [ -n "$ACTUAL_HASH" ]; then
            # Comparação insensível a maiúsculas
            ACTUAL_LOWER=$(printf "%s" "$ACTUAL_HASH" | tr '[:upper:]' '[:lower:]')
            EXPECTED_LOWER=$(printf "%s" "$EXPECTED_HASH" | tr '[:upper:]' '[:lower:]')
            if [ "$ACTUAL_LOWER" != "$EXPECTED_LOWER" ]; then
                err "Falha na verificação de integridade!"
                err "Hash esperado: $EXPECTED_HASH"
                err "Hash obtido:   $ACTUAL_HASH"
                exit 1
            fi
            success "Assinatura SHA-256 verificada com sucesso!"
        else
            warn "Nenhum utilitário SHA-256 (sha256sum, shasum, openssl) disponível; prosseguindo sem validação de hash."
        fi
    else
        warn "Assinatura para ${TAR_NAME} não encontrada em checksums.txt, prosseguindo com cautela..."
    fi
else
    warn "Não foi possível baixar checksums.txt; prosseguindo sem verificação."
fi

# 7. Extração do Binário
step "Extraindo binário para: ${INSTALL_DIR}"
mkdir -p "$INSTALL_DIR"
tar -xzf "$TAR_PATH" -C "$TMP_DIR"

BIN_SRC=""
if [ -f "${TMP_DIR}/mem" ]; then
    BIN_SRC="${TMP_DIR}/mem"
elif [ -f "${TMP_DIR}/bin/mem" ]; then
    BIN_SRC="${TMP_DIR}/bin/mem"
else
    # Busca recursiva caso esteja em subpasta
    BIN_SRC=$(find "$TMP_DIR" -type f -name "mem" 2>/dev/null | head -n 1 || true)
fi

if [ -z "$BIN_SRC" ] || [ ! -f "$BIN_SRC" ]; then
    err "O executável 'mem' não foi encontrado no arquivo descompactado."
    exit 1
fi

TARGET_BIN="${INSTALL_DIR}/mem"
cp -f "$BIN_SRC" "$TARGET_BIN"
chmod +x "$TARGET_BIN"
success "Binário instalado com sucesso em: ${TARGET_BIN}"

# 8. Verificação de PATH
if [ "$NO_PATH" -eq 0 ]; then
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*)
            success "Diretório já está configurado no PATH."
            ;;
        *)
            warn "O diretório ${INSTALL_DIR} NÃO está no seu PATH atual."
            printf "\n  Para usar o comando 'mem' diretamente no terminal, adicione ao seu shell profile:\n"
            if [ -n "$ZSH_VERSION" ] || [ -f "${HOME}/.zshrc" ]; then
                printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc && source ~/.zshrc\n\n" "$INSTALL_DIR"
            else
                printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc && source ~/.bashrc\n\n" "$INSTALL_DIR"
            fi
            ;;
    esac
fi

# 9. Validar Execução
step "Validando execução do executável..."
if VERSION_OUT=$("$TARGET_BIN" version 2>&1); then
    success "Validação concluída: ${VERSION_OUT}"
else
    warn "Aviso ao testar o binário: ${VERSION_OUT}"
fi

printf "\n"
printf "${GREEN}  Instalação concluída com sucesso!${NC}\n\n"
printf "${CYAN}  Para começar a usar no seu projeto:${NC}\n"
printf "    1. Abra o terminal na pasta do seu repositório de código.\n"
printf "    2. Execute 'mem init' para inicializar o cofre de memória (.memory/).\n"
printf "    3. Execute 'mem install' para auto-configurar seus clientes de IA (Cursor, VS Code, Claude).\n"
printf "    4. Execute 'mem index' para indexar e quantizar suas notas e docs.\n\n"
