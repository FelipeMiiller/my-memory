package main

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// setupVaultComCodigoNoRepo monta um vault com uma nota (que casa na busca
// textual) e um símbolo de código gravado no repositório pedido.
func setupVaultComCodigoNoRepo(t *testing.T, repository string) *sql.DB {
	t.Helper()

	ctx := context.Background()
	database, err := db.InitDB(filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	now := time.Now().Unix()
	if err := db.InsertDocument(ctx, database, "doc_alpha", "notes/alpha.md", "Alpha", now, "hash_alpha"); err != nil {
		t.Fatalf("InsertDocument: %v", err)
	}
	if _, err := database.ExecContext(ctx,
		`INSERT INTO chunks (id, document_id, chunk_index, content) VALUES ('c_alpha', 'doc_alpha', 0, 'alpha search documento')`,
	); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}
	if _, err := database.ExecContext(ctx,
		`INSERT INTO chunks_fts (chunk_id, content) VALUES ('c_alpha', 'alpha search documento')`,
	); err != nil {
		t.Fatalf("insert chunk FTS: %v", err)
	}

	res, err := database.ExecContext(ctx, `
		INSERT INTO code_files (repository, path, language, size_bytes, content_hash, ast_hash, indexed_at)
		VALUES (?, 'pkg/hybrid.go', 'go', 120, 'code_hash', NULL, 0)
	`, repository)
	if err != nil {
		t.Fatalf("insert code_files: %v", err)
	}
	fileID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO code_symbols
			(repository, file_id, kind, name, qualified_name, signature, doc_comment, start_line, end_line, start_col, end_col)
		VALUES (?, ?, 'function', 'Search', 'pkg.Search', 'func Search(query string) error', 'Search busca.', 42, 60, 1, 2)
	`, repository, fileID); err != nil {
		t.Fatalf("insert code_symbols: %v", err)
	}
	return database
}

// TestBuildSQLiteSearchFunc_IncludeCodeUsaRepoDoVault prova o defeito D2:
// o caminho de busca do servidor MCP não propagava o repo_id do vault para o
// contexto, então `include_code=true` era um NO-OP em qualquer vault cujo
// repo_id não fosse "default" (o RepositoryFromContext devolvia "default").
func TestBuildSQLiteSearchFunc_IncludeCodeUsaRepoDoVault(t *testing.T) {
	const vaultRepo = "acme"
	database := setupVaultComCodigoNoRepo(t, vaultRepo)
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	ctx := context.Background()

	// 1. COMPORTAMENTO ANTES (D2): o servidor MCP não carregava o repo_id do
	//    vault → o fan-in de código não acha nada e a lista é IDÊNTICA à da
	//    busca sem include_code.
	searchBefore := buildSQLiteSearchFunc(database, nil, tq, "default")
	before, err := searchBefore(ctx, mcp.SearchParams{Query: "search", Mode: "hybrid", Limit: 10, IncludeCode: true})
	if err != nil {
		t.Fatalf("busca (antes) falhou: %v", err)
	}
	for _, r := range before {
		if r.DocumentID == "code:pkg.Search" {
			t.Fatalf("o caminho ANTES não devia achar o símbolo de código (repo != vault): %+v", before)
		}
	}

	// 2. COMPORTAMENTO DEPOIS (D2 corrigido): o repo_id do vault viaja no
	//    contexto e o símbolo de código entra no fan-in.
	searchAfter := buildSQLiteSearchFunc(database, nil, tq, vaultRepo)
	after, err := searchAfter(ctx, mcp.SearchParams{Query: "search", Mode: "hybrid", Limit: 10, IncludeCode: true})
	if err != nil {
		t.Fatalf("busca (depois) falhou: %v", err)
	}

	var codeItem *mcp.SearchResult
	noteCount := 0
	for i := range after {
		if after[i].DocumentID == "code:pkg.Search" {
			codeItem = &after[i]
			continue
		}
		noteCount++
	}
	if codeItem == nil {
		t.Fatalf("esperava o símbolo code:pkg.Search na lista fundida, obteve %+v", after)
	}
	if noteCount != 1 {
		t.Errorf("esperava 1 resultado de nota junto do código, obteve %d (%+v)", noteCount, after)
	}
	want := 1.0 / float64(60+1)
	if math.Abs(codeItem.Score-want) > 1e-9 {
		t.Errorf("esperava score %f (1/(k+1)), obteve %f", want, codeItem.Score)
	}
	if len(codeItem.Sources) != 1 || codeItem.Sources[0] != "code:1" {
		t.Errorf("esperava Sources [code:1], obteve %v", codeItem.Sources)
	}
}

// TestBuildSQLiteSearchFunc_SemIncludeCodeMantemRanking prova o AC 21 no
// caminho do servidor MCP: sem include_code (ou com include_code=false) o
// ranking é IDÊNTICO ao que existia antes da feature.
func TestBuildSQLiteSearchFunc_SemIncludeCodeMantemRanking(t *testing.T) {
	const vaultRepo = "acme"
	database := setupVaultComCodigoNoRepo(t, vaultRepo)
	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)
	search := buildSQLiteSearchFunc(database, nil, tq, vaultRepo)
	ctx := context.Background()

	before, err := search(ctx, mcp.SearchParams{Query: "search", Mode: "hybrid", Limit: 10})
	if err != nil {
		t.Fatalf("busca (sem include_code) falhou: %v", err)
	}
	if len(before) != 1 || before[0].DocumentID != "doc_alpha" {
		t.Fatalf("esperava só a nota no ranking sem include_code, obteve %+v", before)
	}
	if len(before[0].Sources) != 1 || before[0].Sources[0] != "fts:1" {
		t.Errorf("esperava Sources [fts:1], obteve %v", before[0].Sources)
	}

	// include_code=false explícito: mesma lista.
	after, err := search(ctx, mcp.SearchParams{Query: "search", Mode: "hybrid", Limit: 10, IncludeCode: false})
	if err != nil {
		t.Fatalf("busca (include_code=false) falhou: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("include_code=false mudou a quantidade de resultados: %d vs %d", len(after), len(before))
	}
	for i := range before {
		if before[i].DocumentID != after[i].DocumentID {
			t.Errorf("posição %d: DocumentID mudou de %q para %q", i, before[i].DocumentID, after[i].DocumentID)
		}
		if before[i].Score != after[i].Score {
			t.Errorf("posição %d: score mudou de %f para %f", i, before[i].Score, after[i].Score)
		}
	}
}

// TestBuildSQLiteSearchFunc_ContextoIncludeCode prova que a flag também
// pode vir pelo CONTEXTO (é assim que o handler MCP a transporta).
func TestBuildSQLiteSearchFunc_ContextoIncludeCode(t *testing.T) {
	const vaultRepo = "acme"
	database := setupVaultComCodigoNoRepo(t, vaultRepo)
	search := buildSQLiteSearchFunc(database, nil, turboquant.NewQuantizer(embedder.EmbeddingDim), vaultRepo)

	results, err := search(store.WithIncludeCode(context.Background(), true),
		mcp.SearchParams{Query: "search", Mode: "hybrid", Limit: 10})
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	found := false
	for _, r := range results {
		if r.DocumentID == "code:pkg.Search" {
			found = true
		}
	}
	if !found {
		t.Fatalf("o símbolo de código não entrou pelo contexto: %+v", results)
	}
}
