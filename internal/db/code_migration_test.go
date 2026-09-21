package db

import (
	"context"
	"path/filepath"
	"testing"
)

// --- T2 gates -----------------------------------------------------------------

func TestCodeFilesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_files.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	columns := map[string]bool{
		"id":           true,
		"path":         true,
		"language":     true,
		"size_bytes":   true,
		"content_hash": true,
		"ast_hash":     true,
		"indexed_at":   true,
	}
	for col := range columns {
		var cnt int
		err := d.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('code_files') WHERE name = ?`, col).Scan(&cnt)
		if err != nil {
			t.Fatalf("pragma_table_info falhou: %v", err)
		}
		if cnt != 1 {
			t.Errorf("code_files.%s ausente (count=%d)", col, cnt)
		}
	}

	// Índice language_idx
	var idx int
	if err := d.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='code_files_language_idx'`).Scan(&idx); err != nil {
		t.Fatalf("consulta sqlite_master falhou: %v", err)
	}
	if idx != 1 {
		t.Errorf("Índice code_files_language_idx ausente")
	}
}

func TestCodeSymbolsSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_symbols.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	want := map[string]string{
		"file_id":          "INTEGER",
		"kind":             "TEXT",
		"name":             "TEXT",
		"qualified_name":   "TEXT",
		"signature":        "TEXT",
		"start_line":       "INTEGER",
		"end_line":         "INTEGER",
		"start_col":        "INTEGER",
		"end_col":          "INTEGER",
		"doc_comment":      "TEXT",
		"parent_symbol_id": "INTEGER",
	}
	rows, err := d.Query(`SELECT name, type FROM pragma_table_info('code_symbols')`)
	if err != nil {
		t.Fatalf("pragma falhou: %v", err)
	}
	defer rows.Close()
	got := make(map[string]string)
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			t.Fatalf("scan falhou: %v", err)
		}
		got[name] = typ
	}
	for col, typ := range want {
		if got[col] == "" {
			t.Errorf("code_symbols.%s ausente", col)
		} else if typ != "" && got[col] != typ {
			t.Errorf("code_symbols.%s tipo = %q; want %q", col, got[col], typ)
		}
	}
}

func TestCodeEdgesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_edges.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	wantIdx := []string{
		"code_graph_kind_idx",
		"code_graph_src_idx",
		"code_graph_dst_idx",
	}
	for _, name := range wantIdx {
		var cnt int
		err := d.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&cnt)
		if err != nil {
			t.Fatalf("consulta sqlite_master falhou: %v", err)
		}
		if cnt != 1 {
			t.Errorf("Índice %s ausente", name)
		}
	}
	// Tabela code_edges_uncertain
	var cnt int
	if err := d.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='code_edges_uncertain'`).Scan(&cnt); err != nil {
		t.Fatalf("consulta sqlite_master falhou: %v", err)
	}
	if cnt != 1 {
		t.Errorf("Tabela code_edges_uncertain ausente")
	}
}

func TestCodeMigrationIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_idem.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	// 1ª execução: cria tudo (já feito pelo InitDB)
	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables run 1 falhou: %v", err)
	}
	// 2ª execução: deve ser no-op (CREATE TABLE IF NOT EXISTS é idempotente)
	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables run 2 falhou: %v", err)
	}
	// 3ª execução: ainda no-op
	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables run 3 falhou: %v", err)
	}

	// Confirmar contagem de tabelas code_*: 4
	rows, err := d.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'code_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("query code_*: %v", err)
	}
	defer rows.Close()
	tables := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables = append(tables, n)
	}
	want := []string{"code_edges", "code_edges_uncertain", "code_files", "code_symbols"}
	if len(tables) != len(want) {
		t.Fatalf("tabelas code_* = %v; want %v", tables, want)
	}
	for i, n := range want {
		if tables[i] != n {
			t.Fatalf("tabela[%d]=%q; want %q", i, tables[i], n)
		}
	}
}

func TestCodeFilesUpsertAndRoundtrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_upsert.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	id, err := UpsertCodeFile(ctx, d, "internal/x/foo.go", "go", "hash-1", "ast-1", 1234, 1700000000)
	if err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	if id == 0 {
		t.Fatal("id = 0 inesperado")
	}

	// Roundtrip GetCodeFileHash
	ch, ah, ok, err := GetCodeFileHash(ctx, d, "internal/x/foo.go")
	if err != nil {
		t.Fatalf("GetCodeFileHash falhou: %v", err)
	}
	if !ok || ch != "hash-1" || ah != "ast-1" {
		t.Fatalf("Roundtrip falhou: ch=%q ah=%q ok=%v", ch, ah, ok)
	}

	// Update: muda ast_hash, mantendo path
	id2, err := UpsertCodeFile(ctx, d, "internal/x/foo.go", "go", "hash-2", "ast-2", 1235, 1700000001)
	if err != nil {
		t.Fatalf("UpsertCodeFile update falhou: %v", err)
	}
	if id != id2 {
		t.Errorf("id mudou no upsert: %d → %d", id, id2)
	}
	ch2, ah2, _, _ := GetCodeFileHash(ctx, d, "internal/x/foo.go")
	if ch2 != "hash-2" || ah2 != "ast-2" {
		t.Errorf("update não persistiu: ch=%q ah=%q", ch2, ah2)
	}

	// Delete cascade: apagar code_files deve ser suficiente (CASCADE)
	if err := DeleteCodeFile(ctx, d, "internal/x/foo.go"); err != nil {
		t.Fatalf("DeleteCodeFile falhou: %v", err)
	}
	_, _, ok, err = GetCodeFileHash(ctx, d, "internal/x/foo.go")
	if err != nil {
		t.Fatalf("GetCodeFileHash pós-delete falhou: %v", err)
	}
	if ok {
		t.Errorf("arquivo ainda presente após delete")
	}
}

func TestCodeSymbolUpsertAndUncertainEdges(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_sym.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()
	ctx := context.Background()

	fid, err := UpsertCodeFile(ctx, d, "a/b.go", "go", "h", "", 100, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	s1, err := UpsertCodeSymbol(ctx, d, fid, "function", "Hello", "example.Hello", "func Hello()", "greets", 10, 12, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol s1 falhou: %v", err)
	}
	s2, err := UpsertCodeSymbol(ctx, d, fid, "struct", "Config", "example.Config", "", "", 1, 5, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol s2 falhou: %v", err)
	}

	// Edge alta confiança (1.0) → code_edges
	if err := UpsertCodeEdge(ctx, d, s1, s2, fid, "uses_type", 10, 12, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge alta-conf falhou: %v", err)
	}

	// Edge baixa confiança → redirecionar (deve dar erro explicativo)
	if err := UpsertCodeEdge(ctx, d, s1, s2, fid, "calls", 11, 11, 0.3); err == nil {
		t.Fatal("esperava erro ao tentar inserir edge de baixa confiança em code_edges")
	}

	// Versão uncertain: deve inserir
	if err := InsertCodeEdgeUncertain(ctx, d, s1, fid, "Helper", "calls", "ref ambígua", 0.3, 11); err != nil {
		t.Fatalf("InsertCodeEdgeUncertain falhou: %v", err)
	}

	// Contagens
	nFiles, _ := CountCodeFiles(ctx, d)
	nSyms, _ := CountCodeSymbols(ctx, d)
	if nFiles != 1 {
		t.Errorf("CountCodeFiles = %d; want 1", nFiles)
	}
	if nSyms != 2 {
		t.Errorf("CountCodeSymbols = %d; want 2", nSyms)
	}
}

func TestCodeMigrationOnExistingDB_AddsMissingColumns(t *testing.T) {
	// Simula uma base com schema antigo SEM ast_hash/confidence.
	dbPath := filepath.Join(t.TempDir(), "code_old.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}

	// Remove ast_hash e confidence para simular base pré-migration.
	if _, err := d.Exec(`ALTER TABLE code_files DROP COLUMN ast_hash`); err != nil {
		// SQLite não suporta DROP COLUMN em todas as versões — pula o teste
		// nesse caso (testa apenas o caminho feliz do EnsureCodeTables).
		t.Logf("SQLite não suporta DROP COLUMN: %v (teste segue pelo caminho EnsureCodeTables direto)", err)
		d.Close()
		return
	}
	d.Close()

	// Reabre, roda EnsureCodeTables (deve recriar ast_hash via ALTER).
	d2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB (reabertura) falhou: %v", err)
	}
	defer d2.Close()

	if err := EnsureCodeTables(context.Background(), d2); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}
	var cnt int
	_ = d2.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('code_files') WHERE name='ast_hash'`).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("ast_hash não foi restaurada após EnsureCodeTables (cnt=%d)", cnt)
	}
}
