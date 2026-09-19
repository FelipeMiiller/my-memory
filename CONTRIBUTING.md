# Contribuindo com o My-Memory

> 📖 [[README|README]] • 🤝 [[CONTRIBUTING|CONTRIBUTING]] • 📜 [[CODE_OF_CONDUCT|CODE_OF_CONDUCT]] • 🛡 [[SECURITY|SECURITY]]

Obrigado por seu interesse em contribuir com o **My-Memory**! Este projeto combina sistemas de alta performance em Go, extensões SQLite (`sqlite-vec`), compressão vetorial de última geração (TurboQuant - Google DeepMind, ICLR 2026), travessia de grafos via SQL recursivo e integração de agentes de IA via Model Context Protocol (MCP).

---

## 🛠 Requisitos de Desenvolvimento

- **Go**: Versão 1.22 ou superior.
- **Compilador C (CGO)**: Necessário para compilar o driver SQLite3 e as extensões `sqlite-vec` (GCC no Linux/macOS ou MinGW-w64 no Windows).
- **Ollama**: Rodando localmente com o modelo `nomic-embed-text`:
  ```bash
  ollama pull nomic-embed-text
  ```
- **Git**: Com suporte a commits no padrão Conventional Commits.

---

## 🚀 Fluxo de Trabalho Recomendado

1. **Fork e Clone:**
   ```bash
   git clone https://github.com/SEU_USUARIO/my-memory.git
   cd my-memory
   ```

2. **Crie uma Branch de Funcionalidade:**
   ```bash
   git checkout -b feat/minha-melhoria
   ```

3. **Verifique os Testes Existentes:**
   ```bash
   go test -v ./internal/...
   ```

4. **Aplique Formatação Oficial Go:**
   ```bash
   gofmt -w .
   ```

5. **Execute a Análise Estática:**
   ```bash
   go vet ./...
   ```

---

## 📌 Padrões de Código e Commits

### 1. Conventional Commits 1.0.0
Todos os commits devem seguir a especificação [Conventional Commits](https://www.conventionalcommits.org/):

- `feat(escopo): descrição da nova funcionalidade`
- `fix(escopo): correção de bug`
- `docs(escopo): alterações em documentação`
- `refactor(escopo): refatoração de código sem alterar comportamento`
- `test(escopo): adição ou correção de testes`
- `chore(escopo): tarefas de manutenção, CI ou build`

Exemplo:
```bash
feat(mcp): implement tool registration and tools/list endpoint
```

### 2. Formatação e Idioma
- O código Go segue a formatação estrita do `gofmt`.
- Nomes de variáveis, tipos e funções são escritos em **inglês**.
- Documentação, comentários e ADRs são mantidos preferencialmente em **português**, com terminologia técnica universal.
- Arquivos de código devem ser salvos em **UTF-8 sem BOM**.

### 3. Decisões Arquiteturais (ADRs)
Qualquer proposta de mudança que envolva bibliotecas estruturais, esquemas do banco ou arquitetura deve ser documentada previamente como um ADR na pasta `docs/adr/`, seguindo o formato **MADR**.

---

## 🧪 Testes e Cobertura

- Toda nova funcionalidade deve vir acompanhada de testes unitários em arquivos `*_test.go`.
- Testes não devem ser enfraquecidos ou removidos para forçar aprovação em pipeline.
- Para rodar testes com relatório de cobertura:
  ```bash
  go test -v -coverprofile=coverage.txt ./internal/...
  go tool cover -func=coverage.txt
  ```

---

## 📬 Submetendo um Pull Request

1. Garanta que todos os testes passem localmente (`go test -v ./internal/...`).
2. Garanta que o binário compile sem erros (`go build -v -o bin/mem.exe ./cmd/mem`).
3. Abra o Pull Request descrevendo a motivação, as mudanças realizadas e os testes executados.
