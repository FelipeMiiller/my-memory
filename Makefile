# Makefile para o projeto My-Memory
.PHONY: all build test test-all test-coverage lint fmt clean run-index run-search viewer

BINARY_NAME=mem
BUILD_DIR=bin

all: lint test build

## Compila o executável principal
build:
	@echo "==> Compilando $(BINARY_NAME)..."
	@go build -v -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mem

## Executa os testes unitários principais
test:
	@echo "==> Executando testes unitários..."
	@go test -v ./internal/parser/... ./internal/turboquant/... ./internal/mcp/...

## Executa todos os testes incluindo pacotes com CGO (sqlite-vec)
test-all:
	@echo "==> Executando todos os testes (requer CGO/GCC)..."
	@go test -v ./internal/...

## Executa testes com relatório de cobertura
test-coverage:
	@echo "==> Gerando relatório de cobertura..."
	@go test -v -coverprofile=coverage.txt -covermode=atomic ./internal/parser/... ./internal/turboquant/... ./internal/mcp/...
	@go tool cover -func=coverage.txt

## Aplica formatação automática Go
fmt:
	@echo "==> Formatando arquivos Go com gofmt..."
	@gofmt -w .

## Executa análise estática
lint:
	@echo "==> Executando verificação de formatação e go vet..."
	@test -z "$$(gofmt -l .)" || (echo "Arquivos fora do padrão gofmt:" && gofmt -l . && exit 1)
	@go vet ./...

## Limpa binários e relatórios gerados
clean:
	@echo "==> Limpando diretórios de build..."
	@rm -rf $(BUILD_DIR) coverage.txt

## Compila a viewer e publica o build na raiz servida pelo MCP
## (ADR-059 D-2, ADR-060). A raiz servida é .mymemory/www/, com app/ para o
## build da SPA e graph/ para os artefatos de grafo gerados.
viewer:
	@echo "==> Instalando dependências da viewer..."
	cd viewer && npm install --no-audit --no-fund
	@echo "==> Compilando a viewer..."
	cd viewer && npm run build
	@echo "==> Publicando o build em .mymemory/www/app/..."
	@rm -rf .mymemory/www/app
	@mkdir -p .mymemory/www/app
	@cp -R viewer/dist/. .mymemory/www/app/
	@echo "==> Viewer publicada. Suba com 'mem mcp --port 38400' e abra http://127.0.0.1:38400/"
