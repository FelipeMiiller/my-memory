package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// codeSearchTestFixture cria um vault com arquivos Go + Python, roda o
// pipeline de indexação (mock) e devolve o caminho do SQLite pronto
// para uso do CLI.
func codeSearchTestFixture(t *testing.T) (string, []string) {
	t.Helper()
	tmpVault := t.TempDir()
	dbDir := t.TempDir()
	dbFile := filepath.Join(dbDir, "codesearch.db")

	// Cria 3 arquivos Go e 1 Python para exercitar o filtro --lang e --kind.
	files := map[string]string{
		"pkg/foo.go": `package foo
func Hello() string { return "hi" }
func Bye() string { return "bye" }
type Greeter struct{}
func (g *Greeter) Greet() string { return "" }
`,
		"pkg/bar.go": `package bar
import "fmt"
func Bar() { fmt.Println("bar") }
type Bar struct{ X int }
`,
		"pkg/baz.py": `def hello():
    return "hi"

class MyClass:
    def method(self):
        return 0
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

	// Roda o pipeline (mock parser) para popular code_symbols.
	if err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--lang=go,python",
		"--include=**/*.go",
		"--exclude=**/.git/**",
		tmpVault,
	}); err != nil {
		// O mock parser não extrai symbols de python por padrão; tentamos
		// de novo com only go e verificamos que o DB existe.
		if err := runCodeIndexCLI(context.Background(), "FelipeMiiller/my-memory", []string{
			"--db=" + dbFile,
			"--lang=go",
			"--include=**/*.go",
			"--exclude=**/.git/**",
			tmpVault,
		}); err != nil {
			t.Fatalf("index runCodeIndexCLI: %v", err)
		}
	}

	return dbFile, []string{"pkg/foo.go", "pkg/bar.go", "pkg/baz.py"}
}

func TestCodeSearch_Query_BasicMatch(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	hits, err := runCodeSearchQuery(context.Background(), database, "Hello", "", "", 10)
	if err != nil {
		t.Fatalf("runCodeSearchQuery falhou: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("esperava pelo menos 1 hit para 'Hello'")
	}
	// A primeira hit deve ser a função Hello em foo.go.
	if !strings.Contains(hits[0].Name, "Hello") {
		t.Errorf("primeira hit = %q; esperava conter 'Hello'", hits[0].Name)
	}
}

func TestCodeSearch_BoostByQualifiedName(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	// Query que é qualified_name exato de um símbolo.
	hits, err := runCodeSearchQuery(context.Background(), database, "foo.Hello", "", "", 10)
	if err != nil {
		t.Fatalf("runCodeSearchQuery falhou: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("esperava hits para query qualified_name")
	}
	// Aplica boost manualmente para validar.
	noBoost := false
	out := codeast.ApplyBoostToHits(hits, "foo.Hello", noBoost)
	boostedCount := 0
	for _, h := range out {
		if h.Boosted {
			boostedCount++
			if h.RRFScore != 2.0 {
				t.Errorf("score boosted=%.2f; want 2.0", h.RRFScore)
			}
		}
	}
	if boostedCount == 0 {
		t.Error("esperava pelo menos 1 hit boosted para 'foo.Hello'")
	}
}

func TestCodeSearch_NoBoostFlag(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	hits, err := runCodeSearchQuery(context.Background(), database, "foo.Hello", "", "", 10)
	if err != nil {
		t.Fatalf("runCodeSearchQuery falhou: %v", err)
	}
	out := codeast.ApplyBoostToHits(hits, "foo.Hello", true)
	for _, h := range out {
		if h.Boosted {
			t.Error("com noBoost=true, nenhuma hit pode ser boosted")
		}
		if h.RRFScore != 1.0 {
			t.Errorf("score sem boost=%.2f; want 1.0", h.RRFScore)
		}
	}
}

func TestCodeSearch_LangFilter(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	hits, err := runCodeSearchQuery(context.Background(), database, "Hello", "go", "", 10)
	if err != nil {
		t.Fatalf("runCodeSearchQuery falhou: %v", err)
	}
	for _, h := range hits {
		if h.Language != "go" {
			t.Errorf("filtro lang=go falhou: hit %s lang=%s", h.Name, h.Language)
		}
	}
}

func TestCodeSearch_KindFilter(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	hits, err := runCodeSearchQuery(context.Background(), database, "", "", "function", 10)
	if err != nil {
		t.Fatalf("runCodeSearchQuery falhou: %v", err)
	}
	for _, h := range hits {
		if h.Kind != "function" {
			t.Errorf("filtro kind=function falhou: hit %s kind=%s", h.Name, h.Kind)
		}
	}
}

func TestCodeSearch_CLI_NoArgs(t *testing.T) {
	err := runCodeSearchCLI(context.Background(), "FelipeMiiller/my-memory", []string{})
	if err == nil {
		t.Fatal("esperava erro para query vazia")
	}
	if !strings.Contains(err.Error(), "query") {
		t.Errorf("erro deve mencionar 'query'; got %v", err)
	}
}

func TestCodeSearch_CLI_EndToEnd(t *testing.T) {
	dbFile, _ := codeSearchTestFixture(t)
	// Executa via CLI pública com --db explícito.
	err := runCodeSearchCLI(context.Background(), "FelipeMiiller/my-memory", []string{
		"--db=" + dbFile,
		"--limit=5",
		"Hello",
	})
	if err != nil {
		t.Fatalf("runCodeSearchCLI falhou: %v", err)
	}
}

func TestCodeSearch_TokenizeQuery(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"hello", []string{"hello"}},
		{"hello world", []string{"hello", "world"}},
		{"  hello   world  ", []string{"hello", "world"}},
		{"foo.HELLO bar", []string{"foo.hello", "bar"}},
		{"a, b; c", []string{"a", "b", "c"}},
		{"", nil},
		{"   ", nil},
	}
	for _, c := range cases {
		got := tokenizeQuery(c.in)
		if !equalStringSlices(got, c.want) {
			t.Errorf("tokenizeQuery(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestCodeSearch_CodeSearchHitJSONStable(t *testing.T) {
	hit := codeast.CodeSearchHit{
		SymbolID: 1, QualifiedName: "pkg.Func", Name: "Func", Kind: "function",
		Language: "go", FilePath: "x.go", StartLine: 1, EndLine: 2, RRFScore: 2.0, Boosted: true,
	}
	b, err := json.Marshal(hit)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(b), `"symbol_id":1`) {
		t.Errorf("JSON deve conter symbol_id; got %s", string(b))
	}
	if !strings.Contains(string(b), `"boosted":true`) {
		t.Errorf("JSON deve conter boosted=true; got %s", string(b))
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
