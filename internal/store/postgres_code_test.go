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
// de code_files: o PostgreSQL exige um índice único correspondente.
func TestPostgresSchemaCodeTemChaveUnicaDePath(t *testing.T) {
	if !strings.Contains(PostgresSchema, "path TEXT NOT NULL UNIQUE") {
		t.Error("PostgresSchema sem chave única em code_files.path")
	}
	if !strings.Contains(PostgresSchema, "UNIQUE (file_id, kind, qualified_name, start_line)") {
		t.Error("PostgresSchema sem a chave única de code_symbols")
	}
	if !strings.Contains(PostgresSchema, "UNIQUE (src_symbol_id, dst_symbol_id, kind)") {
		t.Error("PostgresSchema sem a chave única de code_edges")
	}
}
