package main

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestRunSearchSQLite_SemProviderNaoQuebra cobre o caso em que o binário foi
// compilado sem a tag de build do provider vetorial.
//
// Sem provider, `resolveEmbedder` devolve um embedder nil, e a indexação cai
// para o modo léxico FTS5 — modo suportado e anunciado pelo próprio comando
// `mem index`. A busca, porém, chamava `emb.GenerateEmbedding` sem guarda, e
// o binário sem provider derrubava `mem search` inteiro com um
// nil-pointer-dereference. Nenhum teste pegou isso: os testes de busca sempre
// rodavam com provider, e o caminho sem provider só aparece na máquina de
// quem não tem o ONNX instalado.
func TestRunSearchSQLite_SemProviderNaoQuebra(t *testing.T) {
	dbPath := newVaultFixture(t, "nota.md")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	ctx := context.Background()

	// Híbrido: sem provider a busca é só léxica, e não pode quebrar.
	if code := runSearchSQLite(ctx, database, nil, tq, "nota", "hybrid", false, 5, 60,
		store.DefaultDecayOptions(), store.SearchOptions{}); code != 0 {
		t.Errorf("busca híbrida sem provider deveria sair com código zero, obteve %d", code)
	}

	// O mesmo com o filtro de tag: era o caminho que o smoke test pegou.
	if code := runSearchSQLite(ctx, database, nil, tq, "nota", "hybrid", false, 5, 60,
		store.DefaultDecayOptions(), store.SearchOptions{Tag: "backend"}); code != 0 {
		t.Errorf("busca híbrida com filtro de tag sem provider deveria sair com código zero, obteve %d", code)
	}
}

// TestRunSearchSQLite_ModoVetorialSemProviderNaoQuebra garante que o modo
// vetorial sai com mensagem e código de erro em vez de panic quando não há
// provider.
func TestRunSearchSQLite_ModoVetorialSemProviderNaoQuebra(t *testing.T) {
	dbPath := newVaultFixture(t, "nota.md")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Se este teste voltar, é porque alguém removeu a guarda de `emb == nil`
	// do modo vetorial: sem provider, `emb.GenerateEmbedding` é um
	// nil-pointer-dereference.
	code := runSearchSQLite(context.Background(), database, nil,
		turboquant.NewQuantizer(embedder.EmbeddingDim), "nota", "vector", false, 5, 60,
		store.DefaultDecayOptions(), store.SearchOptions{})

	// O modo vetorial sem provider é erro de uso: a mensagem explica e o
	// comando sai com código diferente de zero.
	if code == 0 {
		t.Errorf("modo vetorial sem provider deveria devolver código de erro")
	}
}

// TestBuildSQLiteSearchFunc_SemProviderNaoQuebra cobre o caminho do servidor
// MCP.
//
// A busca do servidor MCP roda em goroutine (`internal/federation/search.go`),
// e um pânico dentro dela derrubava o **processo inteiro** — não só a
// requisição. Sem provider de embedding, a busca chamava
// `emb.GenerateEmbedding` sem guarda e o servidor MCP morria no primeiro
// `memory_search`.
func TestBuildSQLiteSearchFunc_SemProviderNaoQuebra(t *testing.T) {
	dbPath := newVaultFixture(t, "nota.md")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	search := buildSQLiteSearchFunc(database, nil, turboquant.NewQuantizer(embedder.EmbeddingDim))
	ctx := context.Background()

	// Híbrido sem provider: a busca fica só textual e devolve.
	if _, err := search(ctx, mcp.SearchParams{Query: "nota", Mode: "hybrid", Limit: 5}); err != nil {
		t.Fatalf("a busca híbrida sem provider falhou: %v", err)
	}

	// Vetorial sem provider: erro de uso, não pânico.
	if _, err := search(ctx, mcp.SearchParams{Query: "nota", Mode: "vector", Limit: 5}); err == nil {
		t.Errorf("o modo vetorial sem provider deveria devolver erro")
	}
}
