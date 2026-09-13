# Makefile para o projeto My-Memory
.PHONY: all build test test-coverage lint fmt clean run-index run-search

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
