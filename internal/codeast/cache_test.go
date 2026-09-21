package codeast

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

func newCodeastTestDB(t *testing.T) *dbHandle {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "code_cache.db")
	d, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return &dbHandle{db: d}
}

type dbHandle struct {
	db interface {
		Close() error
	}
}

func (h *dbHandle) close() error { return h.db.Close() }

// --- T3 gates -----------------------------------------------------------------

func TestCacheContentHash_Stable(t *testing.T) {
	// Confirma que ComputeContentHash === HashContent (mesmo código).
	if got := ComputeContentHash([]byte("hello")); got != HashContent([]byte("hello")) {
		t.Errorf("ComputeContentHash deve delegar para HashContent")
	}
	if got := ComputeContentHash([]byte("hello")); len(got) != 64 {
		t.Errorf("ComputeContentHash deve devolver hex de 64 chars; got %d", len(got))
	}
}

func TestCacheAstHash_DeterministicAndOrderIndependent(t *testing.T) {
	symsA := []Symbol{
		{Kind: KindFunction, QualifiedName: "pkg.A", StartLine: 10},
		{Kind: KindStruct, Name: "S", StartLine: 1},
	}
	reversed := []Symbol{symsA[1], symsA[0]}
	if ComputeAstHash(symsA) != ComputeAstHash(reversed) {
		t.Error("ComputeAstHash deve ser determinístico independente da ordem")
	}
	different := append([]Symbol{}, symsA...)
	different[0].StartLine = 99
	if ComputeAstHash(symsA) == ComputeAstHash(different) {
		t.Error("ComputeAstHash deve mudar quando symbols mudam")
	}
}

func TestCacheIncrementalSkip(t *testing.T) {
	h := newCodeastTestDB(t)
	defer h.close()

	ctx := context.Background()
	path := "internal/example/hello.go"
	content := []byte(`package example
func Hello() {}
`)
	symbols := []Symbol{
		{Kind: KindFunction, Name: "Hello", QualifiedName: "example.Hello", StartLine: 2, EndLine: 2},
	}

	cache := NewCodeFileCache(toSQLDB(h))

	// 1. Primeira chamada: ainda não indexado → DecodeFirstIndex.
	res, err := cache.Evaluate(ctx, path, content, symbols)
	if err != nil {
		t.Fatalf("Evaluate (1ª chamada) falhou: %v", err)
	}
	if res.Status != DecodeFirstIndex {
		t.Fatalf("Status = %v; want %v", res.Status, DecodeFirstIndex)
	}

	// Persiste via UpdateAfterIndex.
	if _, err := cache.UpdateAfterIndex(ctx, path, "go", content, symbols, 1700000000); err != nil {
		t.Fatalf("UpdateAfterIndex (1ª) falhou: %v", err)
	}

	// 2. Mesmo content_hash → DecodeSkipParse.
	res2, err := cache.Evaluate(ctx, path, content, symbols)
	if err != nil {
		t.Fatalf("Evaluate (2ª) falhou: %v", err)
	}
	if res2.Status != DecodeSkipParse {
		t.Fatalf("Status = %v; want DecodeSkipParse (idempotência)", res2.Status)
	}

	// 3. content_hash muda, ast_hash igual (cosmético) → DecodeSkipDownstream.
	cosmetic := []byte(`package example

// comentário novo
func Hello() {} // trailing
`)
	// Re-extrair symbols (mesma estrutura, mesma line do Hello)
	symsCosmetic := []Symbol{
		{Kind: KindFunction, Name: "Hello", QualifiedName: "example.Hello", StartLine: 4, EndLine: 4},
	}
	res3, err := cache.Evaluate(ctx, path, cosmetic, symsCosmetic)
	if err != nil {
		t.Fatalf("Evaluate (3ª) falhou: %v", err)
	}
	// ast_hash depende do StartLine — ela mudou (2 → 4), então ast_hash difere.
	// Para validar DecodeSkipDownstream precisamos manter StartLine igual.
	symsCosmeticSameLine := []Symbol{
		{Kind: KindFunction, Name: "Hello", QualifiedName: "example.Hello", StartLine: 2, EndLine: 2},
	}
	res3b, err := cache.Evaluate(ctx, path, cosmetic, symsCosmeticSameLine)
	if err != nil {
		t.Fatalf("Evaluate (3b) falhou: %v", err)
	}
	if res3.Status == DecodeSkipParse {
		t.Errorf("cosmetic change não pode ser skip_parse")
	}
	if res3b.Status != DecodeSkipDownstream {
		t.Fatalf("Status = %v; want DecodeSkipDownstream", res3b.Status)
	}

	// 4. content_hash muda + ast_hash muda (mudança estrutural) → DecodeFullReparse.
	structural := []byte(`package example
type Greeter struct{}
func (g *Greeter) Greet() string { return "" }
`)
	symsStructural := []Symbol{
		{Kind: KindStruct, Name: "Greeter", StartLine: 2, EndLine: 2},
		{Kind: KindMethod, Name: "Greet", QualifiedName: "example.Greeter.Greet", StartLine: 3, EndLine: 3},
	}
	res4, err := cache.Evaluate(ctx, path, structural, symsStructural)
	if err != nil {
		t.Fatalf("Evaluate (4ª) falhou: %v", err)
	}
	if res4.Status != DecodeFullReparse {
		t.Fatalf("Status = %v; want DecodeFullReparse", res4.Status)
	}
}

func TestCacheNilSafeAndEdge(t *testing.T) {
	// Path vazio → erro explícito.
	h := newCodeastTestDB(t)
	defer h.close()
	cache := NewCodeFileCache(toSQLDB(h))
	if _, err := cache.Evaluate(context.Background(), "", []byte("x"), nil); err == nil {
		t.Fatal("esperava erro para path vazio")
	}

	// ComputeContentHash para content vazio
	if h := ComputeContentHash(nil); h == "" {
		t.Fatal("ComputeContentHash não pode ser vazio para []byte{} válido")
	}
	if h := ComputeAstHash(nil); h != "" {
		t.Errorf("ComputeAstHash(nil) deve devolver string vazia; got %q", h)
	}
}

func TestCacheUpdateAfterIndex_Roundtrip(t *testing.T) {
	h := newCodeastTestDB(t)
	defer h.close()
	cache := NewCodeFileCache(toSQLDB(h))
	ctx := context.Background()

	path := "lib/foo.py"
	content := []byte("def foo():\n    return 1\n")
	syms := []Symbol{
		{Kind: KindFunction, Name: "foo", StartLine: 1, EndLine: 2},
	}
	id, err := cache.UpdateAfterIndex(ctx, path, "python", content, syms, 1700001234)
	if err != nil {
		t.Fatalf("UpdateAfterIndex falhou: %v", err)
	}
	if id == 0 {
		t.Fatal("id retornado = 0")
	}

	// PersistSymbols
	ids, err := cache.PersistSymbols(ctx, id, syms)
	if err != nil {
		t.Fatalf("PersistSymbols falhou: %v", err)
	}
	if len(ids) != 1 || ids[0] == 0 {
		t.Errorf("IDs inesperados: %+v", ids)
	}

	// Update novamente (idempotente)
	id2, err := cache.UpdateAfterIndex(ctx, path, "python", content, syms, 1700001234)
	if err != nil {
		t.Fatalf("UpdateAfterIndex idempotente falhou: %v", err)
	}
	if id != id2 {
		t.Errorf("id mudou entre updates: %d → %d", id, id2)
	}
}

func TestCacheReadContent(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "x.go")
	if err := writeFile(p, []byte("package x\n")); err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	data, err := ReadContent(p)
	if err != nil {
		t.Fatalf("ReadContent falhou: %v", err)
	}
	if string(data) != "package x\n" {
		t.Errorf("ReadContent = %q; want 'package x\\n'", string(data))
	}
	if _, err := ReadContent(filepath.Join(tmp, "nope.go")); err == nil {
		t.Error("esperava erro para arquivo inexistente")
	}
}
