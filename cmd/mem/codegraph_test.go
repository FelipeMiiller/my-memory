package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// codeGraphTestFixture cria vault + popula code_symbols + cria edges
// manuais para exercitar o CTE recursivo sem depender de heurística do mock.
func codeGraphTestFixture(t *testing.T) (string, []int64) {
	t.Helper()
	tmpVault := t.TempDir()
	dbDir := t.TempDir()
	dbFile := filepath.Join(dbDir, "codegraph.db")

	files := map[string]string{
		"pkg/foo.go": `package foo
func Hello() {}
func Hello2() {}
`,
		"pkg/bar.go": `package bar
func Bar() {}
`,
		"pkg/baz.go": `package baz
func Baz() {}
`,
	}
	for rel, content := range files {
		full := filepath.Join(tmpVault, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	if err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--lang=go",
		"--include=**/*.go",
		"--exclude=**/.git/**",
		tmpVault,
	}); err != nil {
		t.Fatalf("index runCodeIndexCLI: %v", err)
	}

	// Inserir edges manualmente: foo.Hello → bar.Bar (hop 1),
	// bar.Bar → baz.Baz (hop 2). Permite testar CTE recursivo.
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	idHello, err := resolveCodeSymbol(ctx, database, "foo.Hello")
	if err != nil || idHello <= 0 {
		t.Fatalf("resolveCodeSymbol(foo.Hello) = (%d, %v)", idHello, err)
	}
	idBar, err := resolveCodeSymbol(ctx, database, "bar.Bar")
	if err != nil || idBar <= 0 {
		t.Fatalf("resolveCodeSymbol(bar.Bar) = (%d, %v)", idBar, err)
	}
	idBaz, err := resolveCodeSymbol(ctx, database, "baz.Baz")
	if err != nil || idBaz <= 0 {
		t.Fatalf("resolveCodeSymbol(baz.Baz) = (%d, %v)", idBaz, err)
	}

	// file_id do foo.go (qualquer serve)
	var fooFileID int64
	if err := database.QueryRowContext(ctx, `SELECT id FROM code_files WHERE path = 'pkg/foo.go'`).Scan(&fooFileID); err != nil {
		t.Fatalf("select foo file: %v", err)
	}
	var barFileID int64
	if err := database.QueryRowContext(ctx, `SELECT id FROM code_files WHERE path = 'pkg/bar.go'`).Scan(&barFileID); err != nil {
		t.Fatalf("select bar file: %v", err)
	}
	var bazFileID int64
	if err := database.QueryRowContext(ctx, `SELECT id FROM code_files WHERE path = 'pkg/baz.go'`).Scan(&bazFileID); err != nil {
		t.Fatalf("select baz file: %v", err)
	}

	if err := db.UpsertCodeEdge(ctx, database, idHello, idBar, fooFileID, "calls", 2, 2, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge hello→bar: %v", err)
	}
	if err := db.UpsertCodeEdge(ctx, database, idBar, idBaz, barFileID, "calls", 2, 2, 1.0); err != nil {
		t.Fatalf("UpsertCodeEdge bar→baz: %v", err)
	}

	return dbFile, []int64{idHello, idBar, idBaz}
}

func TestCodeGraph_ResolveSymbol_ExactMatch(t *testing.T) {
	dbFile, ids := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	got, err := resolveCodeSymbol(ctx, database, "foo.Hello")
	if err != nil {
		t.Fatalf("resolveCodeSymbol: %v", err)
	}
	if got != ids[0] {
		t.Errorf("resolveCodeSymbol(foo.Hello) = %d; want %d", got, ids[0])
	}
}

func TestCodeGraph_ResolveSymbol_SuffixFallback(t *testing.T) {
	dbFile, ids := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	got, err := resolveCodeSymbol(ctx, database, "Foo.Hello") // case errado
	if err != nil {
		t.Fatalf("resolveCodeSymbol: %v", err)
	}
	if got != ids[0] {
		t.Errorf("resolveCodeSymbol(Foo.Hello) = %d; want %d (suffix match)", got, ids[0])
	}
}

func TestCodeGraph_ResolveSymbol_NotFound(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	got, err := resolveCodeSymbol(context.Background(), database, "nonexistent.Symbol")
	if err != nil {
		t.Fatalf("resolveCodeSymbol: %v", err)
	}
	if got != -1 {
		t.Errorf("resolveCodeSymbol deve devolver -1; got %d", got)
	}
}

func TestCodeGraph_Depth1_OutOnly(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	rows, err := runCodeGraphQuery(context.Background(), database, "foo.Hello", 1, "out")
	if err != nil {
		t.Fatalf("runCodeGraphQuery: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("esperava pelo menos 1 out-edge")
	}
	if rows[0].Direction != "out" {
		t.Errorf("direction = %q; want 'out'", rows[0].Direction)
	}
	if rows[0].Symbol != "bar.Bar" {
		t.Errorf("symbol = %q; want 'bar.Bar'", rows[0].Symbol)
	}
	if rows[0].Hop != 1 {
		t.Errorf("hop = %d; want 1", rows[0].Hop)
	}
}

func TestCodeGraph_Depth2_OutRecursive(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	rows, err := runCodeGraphQuery(context.Background(), database, "foo.Hello", 2, "out")
	if err != nil {
		t.Fatalf("runCodeGraphQuery: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("esperava ≥2 hops; got %d", len(rows))
	}
	// hops esperados: bar.Bar (hop 1), baz.Baz (hop 2)
	symbols := map[string]int{}
	for _, r := range rows {
		symbols[r.Symbol] = r.Hop
	}
	if symbols["bar.Bar"] != 1 {
		t.Errorf("bar.Bar hop=%d; want 1", symbols["bar.Bar"])
	}
	if symbols["baz.Baz"] != 2 {
		t.Errorf("baz.Baz hop=%d; want 2", symbols["baz.Baz"])
	}
}

func TestCodeGraph_InDirection(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	// Quem chama bar.Bar?
	rows, err := runCodeGraphQuery(context.Background(), database, "bar.Bar", 1, "in")
	if err != nil {
		t.Fatalf("runCodeGraphQuery: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("esperava pelo menos 1 in-edge (foo.Hello → bar.Bar)")
	}
	if rows[0].Direction != "in" {
		t.Errorf("direction = %q; want 'in'", rows[0].Direction)
	}
	if rows[0].Symbol != "foo.Hello" {
		t.Errorf("symbol = %q; want 'foo.Hello'", rows[0].Symbol)
	}
}

func TestCodeGraph_NotFound_ReturnsError(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	_, err = runCodeGraphQuery(context.Background(), database, "nope.NotFound", 1, "out")
	if err == nil {
		t.Fatal("esperava erro para símbolo inexistente")
	}
	if !strings.Contains(err.Error(), "não encontrado") {
		t.Errorf("erro deve indicar 'não encontrado'; got %v", err)
	}
}

func TestCodeGraph_CLI_NoArgs(t *testing.T) {
	err := runCodeGraphCLI(context.Background(), "FelipeMiiller/my-memory", []string{})
	if err == nil {
		t.Fatal("esperava erro para symbol vazio")
	}
}

func TestCodeGraph_CLI_EndToEnd(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	err := runCodeGraphCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--depth=2",
		"--direction=both",
		"foo.Hello",
	})
	if err != nil {
		t.Fatalf("runCodeGraphCLI falhou: %v", err)
	}
}

func TestCodeGraph_DepthClampedAt5(t *testing.T) {
	dbFile, _ := codeGraphTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer database.Close()

	// depth=10 deve ser clampado para 5; query não deve explodir.
	_, err = runCodeGraphQuery(context.Background(), database, "foo.Hello", 10, "out")
	if err != nil {
		t.Fatalf("runCodeGraphQuery: %v", err)
	}
}
