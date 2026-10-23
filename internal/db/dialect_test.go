package db

import (
	"context"
	"path/filepath"
	"testing"
)

// TestDetectDialectSQLite garante que o dialeto do memory.db local é o do
// SQLite — o code pipeline usa o mesmo SQL nos dois engines.
func TestDetectDialectSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dialect.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	if got := DetectDialect(context.Background(), d); got != DialectSQLite {
		t.Fatalf("DetectDialect = %q; want %q", got, DialectSQLite)
	}
	// Memoizado: a segunda chamada devolve o mesmo dialeto.
	if got := DetectDialect(context.Background(), d); got != DialectSQLite {
		t.Fatalf("DetectDialect (2a chamada) = %q; want %q", got, DialectSQLite)
	}
}

// TestBindPostgresConvertePlaceholders prova a conversão "?" → "$N" que
// permite ao mesmo código de escrita falar com o lib/pq.
func TestBindPostgresConvertePlaceholders(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "um placeholder",
			in:   "SELECT * FROM code_files WHERE path = ?",
			want: "SELECT * FROM code_files WHERE path = $1",
		},
		{
			name: "varios placeholders",
			in:   "INSERT INTO code_files (path, language) VALUES (?, ?)",
			want: "INSERT INTO code_files (path, language) VALUES ($1, $2)",
		},
		{
			name: "placeholder dentro de literal nao e convertido",
			in:   "SELECT * FROM code_files WHERE path = 'a?b' AND language = ?",
			want: "SELECT * FROM code_files WHERE path = 'a?b' AND language = $1",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DialectPostgres.Bind(c.in)
			if got != c.want {
				t.Fatalf("Bind() = %q; want %q", got, c.want)
			}
		})
	}
}

// TestBindSQLitePreservaPlaceholders garante que o SQLite continua recebendo
// "?" (o bind é no-op fora do PostgreSQL).
func TestBindSQLitePreservaPlaceholders(t *testing.T) {
	in := "SELECT * FROM code_files WHERE path = ?"
	if got := DialectSQLite.Bind(in); got != in {
		t.Fatalf("Bind() = %q; want %q (inalterado)", got, in)
	}
}

// TestEnsureCodeTablesIdempotenteNoSQLite cobre a migração aditiva rodando
// duas vezes no mesmo banco (o pipeline chama EnsureCodeTables a cada indexação).
func TestEnsureCodeTablesIdempotenteNoSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ensure.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	for i := 1; i <= 2; i++ {
		if err := EnsureCodeTables(context.Background(), d); err != nil {
			t.Fatalf("EnsureCodeTables (run %d) falhou: %v", i, err)
		}
	}

	var cnt int
	if err := d.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('code_files') WHERE name='ast_hash'`).Scan(&cnt); err != nil {
		t.Fatalf("pragma falhou: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("code_files.ast_hash count=%d; want 1", cnt)
	}
}
