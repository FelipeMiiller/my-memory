package db

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// setupCodeFanInTest monta um vault mínimo: uma nota com um chunk (que casa
// na busca textual) e um arquivo de código com dois símbolos.
func setupCodeFanInTest(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "code_fanin.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("falha ao inicializar SQLite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	now := time.Now().Unix()
	if err := InsertDocument(ctx, database, "doc_alpha", "notes/alpha.md", "Alpha", now, "hash_alpha"); err != nil {
		t.Fatalf("falha ao inserir documento: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO chunks (id, document_id, chunk_index, content)
		VALUES ('c_alpha', 'doc_alpha', 0, 'alpha search documento')
	`); err != nil {
		t.Fatalf("falha ao inserir chunk: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES ('c_alpha', 'alpha search documento')
	`); err != nil {
		t.Fatalf("falha ao indexar chunk no FTS: %v", err)
	}

	res, err := database.ExecContext(ctx, `
		INSERT INTO code_files (repository, path, language, size_bytes, content_hash, ast_hash, indexed_at)
		VALUES ('default', 'pkg/hybrid.go', 'go', 120, 'code_hash', NULL, 0)
	`)
	if err != nil {
		t.Fatalf("falha ao inserir code_files: %v", err)
	}
	fileID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("falha ao obter o id do code_file: %v", err)
	}

	if _, err := database.ExecContext(ctx, `
		INSERT INTO code_symbols
			(repository, file_id, kind, name, qualified_name, signature, doc_comment, start_line, end_line, start_col, end_col)
		VALUES
			('default', ?, 'function', 'Search', 'pkg.Search', 'func Search(query string) error', 'Search busca.', 42, 60, 1, 2),
			('default', ?, 'function', 'Fuse', 'pkg.Fuse', 'func Fuse(a, b int) int', '', 10, 20, 1, 2)
	`, fileID, fileID); err != nil {
		t.Fatalf("falha ao inserir code_symbols: %v", err)
	}

	return database
}

// TestHybridRRFIncludeCodeEntraNoFanIn prova o AC 20 no caminho SQLite:
// com include_code=true, code_symbols entra no RRF como fonte real de
// fan-in — o símbolo aparece na lista fundida, com a fonte "code:<rank>" e
// o score 1/(k+rank).
func TestHybridRRFIncludeCodeEntraNoFanIn(t *testing.T) {
	database := setupCodeFanInTest(t)
	ctx := store.WithIncludeCode(context.Background(), true)

	results, err := SearchHybridRRFWithOptions(ctx, database, nil, "search", nil, 10, 60, false,
		store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions falhou: %v", err)
	}

	var codeItem *SearchResult
	noteCount := 0
	for i := range results {
		if results[i].DocumentID == "code:pkg.Search" {
			codeItem = &results[i]
			continue
		}
		noteCount++
	}

	if codeItem == nil {
		t.Fatalf("esperava o símbolo code:pkg.Search na lista fundida, obteve %+v", results)
	}
	if noteCount != 1 {
		t.Errorf("esperava 1 resultado de nota junto do código, obteve %d (%+v)", noteCount, results)
	}

	want := 1.0 / float64(60+1)
	if math.Abs(codeItem.Score-want) > 1e-9 {
		t.Errorf("esperava score %f (1/(k+1)), obteve %f", want, codeItem.Score)
	}
	if len(codeItem.Sources) != 1 || codeItem.Sources[0] != "code:1" {
		t.Errorf("esperava Sources [code:1], obteve %v", codeItem.Sources)
	}
	if codeItem.CodeFile != "pkg/hybrid.go" {
		t.Errorf("esperava CodeFile 'pkg/hybrid.go', obteve %q", codeItem.CodeFile)
	}
	if codeItem.CodeLine != 42 {
		t.Errorf("esperava CodeLine 42, obteve %d", codeItem.CodeLine)
	}
}

// TestHybridRRFSemIncludeCodeNaoMudaRanking é o AC 21: sem include_code a
// lista de resultados (ids, scores e ordem) é exatamente a de antes da
// mudança — e continua igual quando a flag está ligada mas nenhum símbolo
// casa com a query.
func TestHybridRRFSemIncludeCodeNaoMudaRanking(t *testing.T) {
	database := setupCodeFanInTest(t)

	// 1. Sem include_code: a lista fundida é a de antes da mudança.
	before, err := SearchHybridRRFWithOptions(context.Background(), database, nil, "search", nil, 10, 60, false,
		store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions falhou: %v", err)
	}

	expected := []struct {
		documentID string
		score      float64
		sources    []string
	}{
		{documentID: "doc_alpha", score: 1.0 / float64(60+1), sources: []string{"fts:1"}},
	}
	if len(before) != len(expected) {
		t.Fatalf("esperava %d resultado(s) sem include_code, obteve %d (%+v)", len(expected), len(before), before)
	}
	for i, want := range expected {
		if before[i].DocumentID != want.documentID {
			t.Errorf("posição %d: esperava DocumentID %q, obteve %q", i, want.documentID, before[i].DocumentID)
		}
		if math.Abs(before[i].Score-want.score) > 1e-9 {
			t.Errorf("posição %d: esperava score %f, obteve %f", i, want.score, before[i].Score)
		}
		if len(before[i].Sources) != len(want.sources) || before[i].Sources[0] != want.sources[0] {
			t.Errorf("posição %d: esperava Sources %v, obteve %v", i, want.sources, before[i].Sources)
		}
	}

	// 2. Com include_code, mas nenhum símbolo casando: a lista é idêntica.
	ctx := store.WithIncludeCode(context.Background(), true)
	after, err := SearchHybridRRFWithOptions(ctx, database, nil, "alpha", nil, 10, 60, false,
		store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions falhou: %v", err)
	}

	if len(after) != len(before) {
		t.Fatalf("esperava a mesma quantidade de resultados, obteve %d e %d", len(before), len(after))
	}
	for i := range before {
		if before[i].DocumentID != after[i].DocumentID {
			t.Errorf("posição %d: DocumentID mudou de %q para %q", i, before[i].DocumentID, after[i].DocumentID)
		}
		if before[i].Score != after[i].Score {
			t.Errorf("posição %d: score mudou de %f para %f", i, before[i].Score, after[i].Score)
		}
		if len(before[i].Sources) != len(after[i].Sources) {
			t.Errorf("posição %d: quantidade de fontes mudou", i)
			continue
		}
		for j := range before[i].Sources {
			if before[i].Sources[j] != after[i].Sources[j] {
				t.Errorf("posição %d: fonte mudou de %q para %q", i, before[i].Sources[j], after[i].Sources[j])
			}
		}
	}
}

// TestHybridRRFIncludeCodeNaoAlteraNotaComScoreIgual prova que a fonte
// "code" não altera a contribuição da nota: a nota continua com 1/(k+1).
func TestHybridRRFIncludeCodeNaoAlteraNotaComScoreIgual(t *testing.T) {
	database := setupCodeFanInTest(t)

	withCode, err := SearchHybridRRFWithOptions(store.WithIncludeCode(context.Background(), true), database, nil, "search", nil, 10, 60, false,
		store.DefaultDecayOptions(), store.SearchOptions{Level: "l1"})
	if err != nil {
		t.Fatalf("SearchHybridRRFWithOptions falhou: %v", err)
	}

	for _, res := range withCode {
		if res.DocumentID == "doc_alpha" {
			want := 1.0 / float64(60+1)
			if math.Abs(res.Score-want) > 1e-9 {
				t.Errorf("a nota não pode mudar de score com a fonte code: esperava %f, obteve %f", want, res.Score)
			}
			return
		}
	}
	t.Fatalf("a nota doc_alpha não apareceu na lista: %+v", withCode)
}
