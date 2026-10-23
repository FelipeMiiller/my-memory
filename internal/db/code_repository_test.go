package db

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

// TestCodeTablesTemRepository cobre CG-06: code_files, code_symbols e
// code_edges carregam o repo_id do vault (a federação de repo_id).
func TestCodeTablesTemRepository(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_repo.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	for _, table := range []string{"code_files", "code_symbols", "code_edges"} {
		var cnt int
		q := fmt.Sprintf(`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = 'repository'`, table)
		if err := d.QueryRow(q).Scan(&cnt); err != nil {
			t.Fatalf("pragma_table_info(%s) falhou: %v", table, err)
		}
		if cnt != 1 {
			t.Errorf("%s.repository ausente (count=%d)", table, cnt)
		}
	}
}

// TestCodeRepositoryGravadoÉOResolvido prova que o valor gravado é o
// repo_id carregado pelo contexto — e não um default fixo.
func TestCodeRepositoryGravadoÉOResolvido(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_repo_value.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	ctx := WithRepository(context.Background(), "repo-felipe")
	fid, err := UpsertCodeFile(ctx, d, "a/b.go", "go", "h", "ah", 10, 1)
	if err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	sid, err := UpsertCodeSymbol(ctx, d, fid, "function", "Hi", "pkg.Hi", "", "", 1, 2, 0, 0)
	if err != nil {
		t.Fatalf("UpsertCodeSymbol falhou: %v", err)
	}
	if err := UpsertCodeEdge(ctx, d, sid, sid, fid, "references", 1, 2, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge falhou: %v", err)
	}

	for _, table := range []string{"code_files", "code_symbols", "code_edges"} {
		var repo string
		q := fmt.Sprintf(`SELECT repository FROM %s LIMIT 1`, table)
		if err := d.QueryRow(q).Scan(&repo); err != nil {
			t.Fatalf("SELECT repository FROM %s falhou: %v", table, err)
		}
		if repo != "repo-felipe" {
			t.Errorf("%s.repository = %q; want %q", table, repo, "repo-felipe")
		}
	}
}

// TestCodeRepositoryDefaultSemContexto documenta o fallback: sem repo_id no
// contexto, o code graph usa o repositório "default".
func TestCodeRepositoryDefaultSemContexto(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "code_repo_default.db")
	d, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer d.Close()

	if err := EnsureCodeTables(context.Background(), d); err != nil {
		t.Fatalf("EnsureCodeTables falhou: %v", err)
	}

	ctx := context.Background()
	if _, err := UpsertCodeFile(ctx, d, "a/b.go", "go", "h", "", 10, 1); err != nil {
		t.Fatalf("UpsertCodeFile falhou: %v", err)
	}
	var repo string
	if err := d.QueryRow(`SELECT repository FROM code_files LIMIT 1`).Scan(&repo); err != nil {
		t.Fatalf("SELECT repository falhou: %v", err)
	}
	if repo != DefaultRepository {
		t.Errorf("repository = %q; want %q", repo, DefaultRepository)
	}
}
