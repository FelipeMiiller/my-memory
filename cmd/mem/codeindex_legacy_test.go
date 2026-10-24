package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// legacyCodeTablesDDL é o DDL PRÉ-FEATURE das tabelas code_* (o schema de
// `git show 1bf7e75~1:internal/db/schema.go`): nenhuma delas tinha a coluna
// `repository` (a federação de repo_id veio com a ADR-061).
const legacyCodeTablesDDL = `
DROP TABLE IF EXISTS code_edges_uncertain;
DROP TABLE IF EXISTS code_edges;
DROP TABLE IF EXISTS code_symbols;
DROP TABLE IF EXISTS code_files;

CREATE TABLE IF NOT EXISTS code_files (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    ast_hash TEXT,
    indexed_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS code_symbols (
    id INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_col INTEGER,
    end_col INTEGER,
    doc_comment TEXT,
    parent_symbol_id INTEGER REFERENCES code_symbols(id) ON DELETE SET NULL,
    UNIQUE(file_id, kind, qualified_name, start_line)
);
CREATE TABLE IF NOT EXISTS code_edges (
    id INTEGER PRIMARY KEY,
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    UNIQUE(src_symbol_id, dst_symbol_id, kind)
);
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
`

// TestCodeIndexCommand_BancoLegacyProva o defeito D1: `mem code-index` em um
// banco criado com o DDL PRÉ-FEATURE falhava com
//
//	Pipeline.Run falhou: ... SQL logic error: table code_edges_uncertain has no column named repository
//
// porque a migração aditiva não cobria a tabela de arestas incertas.
func TestCodeIndexCommand_BancoLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "legacy.db")

	// 1. Cria o banco e rebaixa as tabelas code_* ao formato pré-feature.
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if _, err := database.Exec(legacyCodeTablesDDL); err != nil {
		t.Fatalf("DDL legado: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// 2. Fixture: uma função que chama um símbolo inexistente → a aresta vai
	//    para code_edges_uncertain (o caminho que quebrava).
	if err := mkLocalGoFile(t, tmpDir, "main.go", "package legacy\n\nfunc Hi() { ausente() }\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// 3. `mem code-index` no banco legado: tem que rodar sem erro.
	err = runCodeIndexCLI(context.Background(), "legacy/repo", []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--lang=go",
		"--include=**/*.go",
		"--repo=legacy/repo",
		tmpDir,
	})
	if err != nil {
		t.Fatalf("mem code-index em banco pré-feature falhou: %v", err)
	}

	// 4. A coluna repository existe e a aresta incerta carrega o repo certo.
	check, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB (2): %v", err)
	}
	defer check.Close()

	var cols int
	if err := check.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('code_edges_uncertain') WHERE name = 'repository'",
	).Scan(&cols); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if cols != 1 {
		t.Fatalf("code_edges_uncertain.repository ausente após a migração (cnt=%d)", cols)
	}

	var repo string
	var count int
	if err := check.QueryRow("SELECT COUNT(*) FROM code_edges_uncertain").Scan(&count); err != nil {
		t.Fatalf("count code_edges_uncertain: %v", err)
	}
	if count == 0 {
		t.Fatal("a aresta incerta não foi gravada — o fixture não exercitou o caminho")
	}
	if err := check.QueryRow("SELECT repository FROM code_edges_uncertain LIMIT 1").Scan(&repo); err != nil {
		t.Fatalf("leitura da aresta incerta: %v", err)
	}
	if repo != "legacy/repo" {
		t.Fatalf("repository da aresta incerta = %q; want %q", repo, "legacy/repo")
	}
}

// TestCodeIndexCommand_BancoLegacyIdempotente prova que reindexar o mesmo
// banco legado não quebra na segunda rodada (a migração é idempotente).
func TestCodeIndexCommand_BancoLegacyIdempotente(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(t.TempDir(), "legacy2.db")

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if _, err := database.Exec(legacyCodeTablesDDL); err != nil {
		t.Fatalf("DDL legado: %v", err)
	}
	_ = database.Close()

	if err := mkLocalGoFile(t, tmpDir, "main.go", "package legacy\n\nfunc Hi() { ausente() }\n"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	args := []string{
		"--db=" + dbFile,
		"--storage=sqlite",
		"--lang=go",
		"--include=**/*.go",
		"--repo=legacy/repo",
		tmpDir,
	}
	for i := 1; i <= 2; i++ {
		if err := runCodeIndexCLI(context.Background(), "legacy/repo", args); err != nil {
			t.Fatalf("mem code-index (rodada %d) em banco legado falhou: %v", i, err)
		}
	}

	check, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB (2): %v", err)
	}
	defer check.Close()
	var cols int
	if err := check.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('code_edges_uncertain') WHERE name = 'repository'",
	).Scan(&cols); err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	if cols != 1 {
		t.Fatalf("coluna repository duplicada após duas indexações (cnt=%d)", cols)
	}
}
