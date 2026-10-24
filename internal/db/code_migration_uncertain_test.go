package db

import (
	"context"
	"path/filepath"
	"testing"
)

// legacyCodeEdgesUncertainDDL é o DDL PRÉ-FEATURE da tabela de arestas
// incertas: a coluna `repository` NÃO existia (ela entrou com a federação de
// repo_id da ADR-061).
//
// Referência: `git show 1bf7e75~1:internal/db/schema.go`.
const legacyCodeEdgesUncertainDDL = `
CREATE TABLE IF NOT EXISTS code_edges_uncertain (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_candidate_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    confidence REAL NOT NULL,
    reason TEXT,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    UNIQUE(src_symbol_id, dst_candidate_name, kind)
);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_src_idx ON code_edges_uncertain(src_symbol_id);
CREATE INDEX IF NOT EXISTS code_edges_uncertain_conf_idx ON code_edges_uncertain(confidence);
`

// TestEnsureCodeTablesMigraCodeEdgesUncertain prova o defeito D1: a migração
// aditiva cobria code_files/code_symbols/code_edges, mas NÃO a tabela de
// arestas incertas — que é repo-scoped desde a ADR-061. Em um banco
// pré-feature, o InsertCodeEdgeUncertain quebrava com
// "table code_edges_uncertain has no column named repository".
func TestEnsureCodeTablesMigraCodeEdgesUncertain(t *testing.T) {
	database, err := InitDB(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	// Downgrade: recria a tabela no formato pré-feature.
	if _, err := database.Exec("DROP TABLE IF EXISTS code_edges_uncertain"); err != nil {
		t.Fatalf("drop code_edges_uncertain: %v", err)
	}
	if _, err := database.Exec(legacyCodeEdgesUncertainDDL); err != nil {
		t.Fatalf("criação do DDL legado: %v", err)
	}

	// A coluna NÃO pode existir antes da migração (o teste prova o defeito).
	var before int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('code_edges_uncertain') WHERE name = 'repository'",
	).Scan(&before); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if before != 0 {
		t.Fatalf("a coluna repository já existia antes da migração (cnt=%d); o teste não prova nada", before)
	}

	// A migração roda (idempotente).
	if err := EnsureCodeTables(context.Background(), database); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	var after int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('code_edges_uncertain') WHERE name = 'repository'",
	).Scan(&after); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if after != 1 {
		t.Fatalf("code_edges_uncertain.repository não foi criada pela migração (cnt=%d)", after)
	}

	// A inserção real (o caminho que quebrava) funciona e grava o repo certo.
	ctx := WithRepository(context.Background(), "acme")
	fileID, err := UpsertCodeFile(ctx, database, "pkg/a.go", "go", "hash", "ast", 10, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile: %v", err)
	}
	symbolID, err := UpsertCodeSymbol(ctx, database, fileID, "function", "Alpha", "pkg.Alpha", "func Alpha()", "", 1, 2, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol: %v", err)
	}
	if err := InsertCodeEdgeUncertain(ctx, database, symbolID, fileID, "desconhecido", "calls",
		"destino não encontrado", 0.3, 7); err != nil {
		t.Fatalf("InsertCodeEdgeUncertain: %v", err)
	}

	var repo string
	if err := database.QueryRow(
		"SELECT repository FROM code_edges_uncertain WHERE src_symbol_id = ?", symbolID,
	).Scan(&repo); err != nil {
		t.Fatalf("leitura da aresta incerta: %v", err)
	}
	if repo != "acme" {
		t.Fatalf("repository da aresta incerta = %q; want %q", repo, "acme")
	}

	// Idempotência: a segunda migração não falha nem duplica coluna.
	if err := EnsureCodeTables(context.Background(), database); err != nil {
		t.Fatalf("EnsureCodeTables (2ª vez) falhou: %v", err)
	}
	var cols int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('code_edges_uncertain') WHERE name = 'repository'",
	).Scan(&cols); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if cols != 1 {
		t.Fatalf("coluna repository duplicada após a 2ª migração (cnt=%d)", cols)
	}
}
