package store

import (
	"strings"
	"testing"
)

// TestPostgresSchemaTemTabelasCode garante que o code graph tem DDL no
// Postgres: sem estas tabelas o pipeline escrevia num SQLite local que nada
// consultava (CG-05).
func TestPostgresSchemaTemTabelasCode(t *testing.T) {
	for _, table := range []string{"code_files", "code_symbols", "code_edges", "code_edges_uncertain"} {
		want := "CREATE TABLE IF NOT EXISTS " + table
		if !strings.Contains(PostgresSchema, want) {
			t.Errorf("PostgresSchema sem %q", want)
		}
	}
}

// TestPostgresSchemaCodeTemChaveUnicaDePath cobre o conflict target do upsert
// de code_files: o PostgreSQL exige um índice único correspondente. Com a
// federação de repo_id, a identidade do arquivo é (repository, path).
func TestPostgresSchemaCodeTemChaveUnicaDePath(t *testing.T) {
	if !strings.Contains(PostgresSchema, "UNIQUE (repository, path)") &&
		!strings.Contains(PostgresSchema, "CREATE UNIQUE INDEX IF NOT EXISTS idx_code_files_repository_path ON code_files(repository, path)") {
		t.Error("PostgresSchema sem a chave única (repository, path) de code_files")
	}
	if !strings.Contains(PostgresSchema, "UNIQUE (file_id, kind, qualified_name, start_line)") {
		t.Error("PostgresSchema sem a chave única de code_symbols")
	}
	if !strings.Contains(PostgresSchema, "UNIQUE (src_symbol_id, dst_symbol_id, kind)") {
		t.Error("PostgresSchema sem a chave única de code_edges")
	}
}

// TestPostgresSchemaCodeTemRepository cobre CG-06 no Postgres: as três
// tabelas do code graph carregam o repo_id do vault.
func TestPostgresSchemaCodeTemRepository(t *testing.T) {
	for _, table := range []string{"code_files", "code_symbols", "code_edges"} {
		want := "ALTER TABLE " + table + " ADD COLUMN IF NOT EXISTS repository TEXT NOT NULL DEFAULT 'default';"
		if !strings.Contains(PostgresSchema, want) {
			t.Errorf("PostgresSchema sem %q", want)
		}
	}
}
