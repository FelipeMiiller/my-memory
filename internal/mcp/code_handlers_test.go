package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// codeHandlersTestFixture cria vault + indexa via mock + injeta edges manuais.
func codeHandlersTestFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tmpVault := t.TempDir()
	dbDir := t.TempDir()
	dbFile := filepath.Join(dbDir, "mcpcode.db")

	files := map[string]string{
		"pkg/foo.go": `package foo
func Hello() {}
func Hello2() {}
`,
		"pkg/bar.go": `package bar
func Bar() {}
`,
	}
	for rel, content := range files {
		full := filepath.Join(tmpVault, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// Indexa via mem code-index — não podemos importar cmd/mem sem ciclo,
	// então populamos manualmente aqui.
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	ctx := context.Background()
	ids := map[string]int64{}
	for path := range files {
		full := filepath.Join(tmpVault, path)
		lang := "go"
		bytes, _ := os.ReadFile(full)
		fileID, err := db.UpsertCodeFile(ctx, database, path, lang,
			"hash-"+path, "", int64(len(bytes)), 1700000000)
		if err != nil {
			database.Close()
			t.Fatalf("UpsertCodeFile: %v", err)
		}
		// Heurística mínima para popular symbols (mock-like): uma function
		// por linha começando com "func NAME".
		for ln, line := range strings.Split(string(bytes), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "func ") {
				continue
			}
			rest := strings.TrimPrefix(line, "func ")
			rest = strings.TrimSpace(rest)
			for i, r := range rest {
				if r == '(' || r == ' ' {
					rest = rest[:i]
					break
				}
			}
			if rest == "" {
				continue
			}
			pkg := "foo"
			if strings.HasPrefix(path, "pkg/bar") {
				pkg = "bar"
			}
			qn := pkg + "." + rest
			id, err := db.UpsertCodeSymbol(ctx, database, fileID, "function",
				rest, qn, line, "", ln+1, ln+1, 0, 0)
			if err != nil {
				database.Close()
				t.Fatalf("UpsertCodeSymbol: %v", err)
			}
			ids[qn] = id
		}
	}

	// Edge: foo.Hello → bar.Bar
	if idHello, ok := ids["foo.Hello"]; ok {
		if idBar, ok := ids["bar.Bar"]; ok {
			if err := db.UpsertCodeEdge(ctx, database, idHello, idBar,
				1, "calls", 1, 1, 1.0); err != nil {
				database.Close()
				t.Fatalf("UpsertCodeEdge: %v", err)
			}
		}
	}

	return database, dbFile
}

func TestMemoryCodeSearchHandler_NilFn_ReturnsTextMessage(t *testing.T) {
	handler := NewMemoryCodeSearchHandler(nil)
	res, err := handler(context.Background(), json.RawMessage(`{"query":"foo"}`))
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	cr, ok := res.(CallToolResult)
	if !ok {
		t.Fatalf("res não é CallToolResult: %T", res)
	}
	if len(cr.Content) == 0 {
		t.Fatal("content vazio")
	}
	if !strings.Contains(cr.Content[0].Text, "indisponível") {
		t.Errorf("texto deve indicar indisponibilidade; got %q", cr.Content[0].Text)
	}
}

func TestMemoryCodeSearchHandler_MissingQuery(t *testing.T) {
	handler := NewMemoryCodeSearchHandler(nil)
	_, err := handler(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("esperava erro sem query")
	}
	if !strings.Contains(err.Error(), "query") {
		t.Errorf("erro deve mencionar query; got %v", err)
	}
}

func TestMemoryCodeSearchHandler_SearchWithFn(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
		return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
	}
	handler := NewMemoryCodeSearchHandler(fn)

	res, err := handler(context.Background(), json.RawMessage(`{"query":"Hello","limit":5}`))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	cr := res.(CallToolResult)
	if !strings.Contains(cr.Content[0].Text, "Hello") {
		t.Errorf("resultado deve conter 'Hello'; got %s", cr.Content[0].Text)
	}
}

func TestMemoryCodeSearchHandler_BoostActiveByDefault(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
		return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
	}
	handler := NewMemoryCodeSearchHandler(fn)

	res, err := handler(context.Background(), json.RawMessage(`{"query":"foo.Hello","limit":5}`))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	cr := res.(CallToolResult)
	if !strings.Contains(cr.Content[0].Text, `"boosted":true`) {
		t.Errorf("resultado deve ter boosted=true (default); got %s", cr.Content[0].Text)
	}
}

func TestMemoryCodeSearchHandler_NoBoostFlag(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
		return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
	}
	handler := NewMemoryCodeSearchHandler(fn)

	res, _ := handler(context.Background(), json.RawMessage(`{"query":"foo.Hello","no_code_boost":true}`))
	cr := res.(CallToolResult)
	if strings.Contains(cr.Content[0].Text, `"boosted":true`) {
		t.Errorf("com no_code_boost=true, não deve ter boosted=true; got %s", cr.Content[0].Text)
	}
}

func TestMemoryCodeSearchHandler_LangFilter(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
		return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
	}
	handler := NewMemoryCodeSearchHandler(fn)

	res, _ := handler(context.Background(), json.RawMessage(`{"query":"Hello","language":"go"}`))
	cr := res.(CallToolResult)
	if !strings.Contains(cr.Content[0].Text, `"language":"go"`) {
		t.Errorf("filtro language=go deve estar no resultado; got %s", cr.Content[0].Text)
	}
}

func TestMemoryCodeSearchHandler_NilDatabase_Errors(t *testing.T) {
	fn := func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
		return SQLCodeSearch(ctx, nil, query, language, kind, limit, includeBoost)
	}
	handler := NewMemoryCodeSearchHandler(fn)
	_, err := handler(context.Background(), json.RawMessage(`{"query":"foo"}`))
	if err == nil {
		t.Fatal("esperava erro com database nil")
	}
}

func TestMemoryCodeNeighborsHandler_NilFn_ReturnsTextMessage(t *testing.T) {
	handler := NewMemoryCodeNeighborsHandler(nil)
	res, _ := handler(context.Background(), json.RawMessage(`{"symbol":"foo.Hello"}`))
	cr := res.(CallToolResult)
	if !strings.Contains(cr.Content[0].Text, "indisponível") {
		t.Errorf("texto deve indicar indisponibilidade; got %q", cr.Content[0].Text)
	}
}

func TestMemoryCodeNeighborsHandler_MissingSymbol(t *testing.T) {
	handler := NewMemoryCodeNeighborsHandler(nil)
	_, err := handler(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("esperava erro sem symbol")
	}
}

func TestMemoryCodeNeighborsHandler_SearchWithFn(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
		return SQLCodeNeighbors(ctx, database, symbol, depth, direction)
	}
	handler := NewMemoryCodeNeighborsHandler(fn)

	res, err := handler(context.Background(), json.RawMessage(`{"symbol":"foo.Hello","depth":1,"direction":"out"}`))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	cr := res.(CallToolResult)
	var resp MemoryCodeNeighborsResponse
	if err := json.Unmarshal([]byte(cr.Content[0].Text), &resp); err != nil {
		t.Fatalf("JSON response inválido: %v", err)
	}
	if resp.Root == nil || resp.Root.QualifiedName != "foo.Hello" {
		t.Errorf("root não resolvido: %+v", resp.Root)
	}
	if len(resp.Neighbors) == 0 {
		t.Error("esperava pelo menos 1 neighbor (bar.Bar)")
	}
}

func TestMemoryCodeNeighborsHandler_NotFound_Errors(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
		return SQLCodeNeighbors(ctx, database, symbol, depth, direction)
	}
	handler := NewMemoryCodeNeighborsHandler(fn)
	_, err := handler(context.Background(), json.RawMessage(`{"symbol":"nonexistent.Symbol"}`))
	if err == nil {
		t.Fatal("esperava erro para symbol inexistente")
	}
}

func TestMemoryCodeNeighborsHandler_DepthClamped(t *testing.T) {
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	fn := func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
		return SQLCodeNeighbors(ctx, database, symbol, depth, direction)
	}
	handler := NewMemoryCodeNeighborsHandler(fn)
	res, _ := handler(context.Background(), json.RawMessage(`{"symbol":"foo.Hello","depth":99}`))
	cr := res.(CallToolResult)
	var resp MemoryCodeNeighborsResponse
	_ = json.Unmarshal([]byte(cr.Content[0].Text), &resp)
	if resp.Depth > 5 {
		t.Errorf("depth deve ser clampado a 5; got %d", resp.Depth)
	}
}

func TestMemorySearch_IncludeCodeFlag_AcceptsBoolean(t *testing.T) {
	// handler com searchFn nil vai falhar depois do parsing, mas queremos
	// apenas verificar que o parâmetro include_code não causa erro de parse.
	database, _ := codeHandlersTestFixture(t)
	defer database.Close()

	handler := NewMemorySearchHandler(AdvancedSearchFunc(func(ctx context.Context, p SearchParams) ([]SearchResult, error) {
		// Devolve vazio; só precisamos garantir que o flag chegou até aqui.
		if p.Query != "foo" {
			return nil, errors.New("query errada")
		}
		return nil, nil
	}))
	_, err := handler(context.Background(), json.RawMessage(`{"query":"foo","include_code":true}`))
	// Erro pode ocorrer no backend (sem results) mas não no parsing.
	if err != nil && strings.Contains(err.Error(), "Tipo inválido") {
		t.Errorf("include_code deve aceitar boolean; got %v", err)
	}
}

func TestSetCodeSearchHandler_Registers(t *testing.T) {
	srv := NewServer("test", "0.0.0", os.Stdin, os.Stdout, nil)
	called := false
	srv.SetCodeSearchHandler(func(ctx context.Context, q, l, k string, lim int, ib bool) ([]CodeSymbolResult, error) {
		called = true
		return nil, nil
	})

	tools := srv.GetTools()
	found := false
	for _, tool := range tools {
		if tool.Name == "memory_code_search" {
			found = true
		}
	}
	if !found {
		t.Fatal("memory_code_search não foi registrado em tools/list")
	}
	_ = called // só queremos verificar o registro
}

func TestSetCodeNeighborsHandler_Registers(t *testing.T) {
	srv := NewServer("test", "0.0.0", os.Stdin, os.Stdout, nil)
	srv.SetCodeNeighborsHandler(func(ctx context.Context, s string, d int, dir string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
		return nil, nil, nil
	})
	tools := srv.GetTools()
	found := false
	for _, tool := range tools {
		if tool.Name == "memory_code_neighbors" {
			found = true
		}
	}
	if !found {
		t.Fatal("memory_code_neighbors não foi registrado em tools/list")
	}
}

func TestSQLCodeSearch_NilDatabase_Errors(t *testing.T) {
	_, err := SQLCodeSearch(context.Background(), nil, "foo", "", "", 10, true)
	if err == nil {
		t.Fatal("esperava erro com database nil")
	}
}

func TestSQLCodeNeighbors_NilDatabase_Errors(t *testing.T) {
	_, _, err := SQLCodeNeighbors(context.Background(), nil, "foo.Hello", 1, "both")
	if err == nil {
		t.Fatal("esperava erro com database nil")
	}
}

func TestMCPTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"foo.bar", []string{"foo.bar"}},
		{"  HELLO   World  ", []string{"hello", "world"}},
		{"", nil},
	}
	for _, c := range cases {
		got := mcpTokenize(c.in)
		if len(got) != len(c.want) {
			t.Errorf("mcpTokenize(%q) = %v; want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("mcpTokenize(%q)[%d] = %q; want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestHasQualifiedNameShapeMCP(t *testing.T) {
	if !HasQualifiedNameShapeMCP("foo.bar") {
		t.Error("foo.bar deve ser valid")
	}
	if HasQualifiedNameShapeMCP("foobar") {
		t.Error("foobar não tem ponto")
	}
	if HasQualifiedNameShapeMCP("") {
		t.Error("vazio não é valid")
	}
}
