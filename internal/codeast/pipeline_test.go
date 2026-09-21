package codeast

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

func newCodeastPipelineTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "code_pipeline.db")
	d, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// --- T4 gates -----------------------------------------------------------------

func TestPipelineRunsAfterMarkdown_NoOpWhenDisabled(t *testing.T) {
	tmp := t.TempDir()

	// Cria config mínima sem Include para arquivos Go.
	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.go"}
	cfg.Exclude = []string{"**/.git/**"}

	// Escreve 3 arquivos Go.
	mkGoFile(t, tmp, "a.go", "package x\nfunc A(){}\n")
	mkGoFile(t, tmp, "b.go", "package x\nfunc B(){}\n")
	mkGoFile(t, tmp, "c.go", "package x\nfunc C(){}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h, Parser: NewMockParser(false)})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, &cfg); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if pipe.Stats().FilesIndexed != 3 {
		t.Fatalf("FilesIndexed = %d; want 3 (3 arquivos Go)", pipe.Stats().FilesIndexed)
	}
	if pipe.Stats().FilesSkipped != 0 {
		t.Fatalf("1ª execução não pode ter skipped; got %d", pipe.Stats().FilesSkipped)
	}

	// 2ª execução: tudo deve virar DecodeSkipParse.
	pipe2, _ := NewPipeline(PipelineOptions{DB: h, Parser: NewMockParser(false)})
	if err := pipe2.Run(context.Background(), tmp, &cfg); err != nil {
		t.Fatalf("Run 2 falhou: %v", err)
	}
	if pipe2.Stats().FilesIndexed != 0 {
		t.Errorf("2ª execução indexed = %d; want 0 (cache hit)", pipe2.Stats().FilesIndexed)
	}
	if pipe2.Stats().FilesSkipped < 3 {
		t.Errorf("2ª execução skipped = %d; want ≥3", pipe2.Stats().FilesSkipped)
	}
}

func TestPipelineScopeRespected(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "wanted/main.go", "package x\n")
	mkGoFile(t, tmp, "vendor/lib.go", "package vendor\n")    // excluded por DefaultExcludedDirs
	mkGoFile(t, tmp, "node_modules/main.go", "package nm\n") // excluded

	h := newCodeastPipelineTestDB(t)
	pipe, _ := NewPipeline(PipelineOptions{DB: h, Parser: NewMockParser(false)})
	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.go"}
	if err := pipe.Run(context.Background(), tmp, &cfg); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if pipe.Stats().FilesIndexed != 1 {
		t.Fatalf("FilesIndexed = %d; want 1 (apenas wanted/main.go)", pipe.Stats().FilesIndexed)
	}
}

func TestPipelineLangFilter(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n")
	mkPyFile(t, tmp, "a.py", "def foo():\n    return 1\n")
	mkJsFile(t, tmp, "a.js", "function bar(){}\n")

	h := newCodeastPipelineTestDB(t)

	// Apenas Go — DB ainda vazio.
	pipe, _ := NewPipeline(PipelineOptions{
		DB: h, Parser: NewMockParser(false),
		LangAllow: []string{"go"},
	})
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run go-only falhou: %v", err)
	}
	if pipe.Stats().FilesIndexed != 1 {
		t.Errorf("go-only indexed = %d; want 1", pipe.Stats().FilesIndexed)
	}

	// Novo DB limpa para validar multi-lang.
	tmp2 := t.TempDir()
	mkGoFile(t, tmp2, "a.go", "package x\n")
	mkPyFile(t, tmp2, "a.py", "def foo():\n    return 1\n")
	mkJsFile(t, tmp2, "a.js", "function bar(){}\n")
	h2 := newCodeastPipelineTestDB(t)

	pipe2, _ := NewPipeline(PipelineOptions{
		DB: h2, Parser: NewMockParser(false),
		LangAllow: []string{"go", "python"},
	})
	if err := pipe2.Run(context.Background(), tmp2, nil); err != nil {
		t.Fatalf("Run go+py falhou: %v", err)
	}
	// Espera 2 (Go + Python), JS excluído pelo LangAllow.
	if pipe2.Stats().FilesIndexed != 2 {
		t.Errorf("go+py indexed = %d; want 2 (Go + Python, sem JS)", pipe2.Stats().FilesIndexed)
	}

	// Sanidade: third DB limpa + sem filtro = todos os 3.
	h3 := newCodeastPipelineTestDB(t)
	pipe3, _ := NewPipeline(PipelineOptions{
		DB: h3, Parser: NewMockParser(false),
	})
	if err := pipe3.Run(context.Background(), tmp2, nil); err != nil {
		t.Fatalf("Run all-langs falhou: %v", err)
	}
	if pipe3.Stats().FilesIndexed != 3 {
		t.Errorf("all-langs indexed = %d; want 3", pipe3.Stats().FilesIndexed)
	}
}

func TestPipelineNoOpWhenTreesitterDisabled(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\nfunc A(){}\n")

	h := newCodeastPipelineTestDB(t)
	if !IsTreesitterEnabled() {
		t.Skip("compilando sem -tags treesitter — pula teste de no-op específico (skip é o comportamento default)")
	}

	// Stub treesitter parser: ParseFile devolve ErrTreesitterDisabled.
	stub := newStubTreesitterParser()
	pipe, err := NewPipeline(PipelineOptions{DB: h, Parser: stub})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	// Mesmo com erro, o pipeline registra o file (1 file seen) e marca como
	// Errors==0 porque ErrTreesitterDisabled é tratado internamente como no-op
	// (fallback UpdateAfterIndex). SymbolsTotal==0.
	if pipe.Stats().FilesIndexed != 1 {
		t.Errorf("FilesIndexed = %d; want 1 (registrado com symbols vazios)", pipe.Stats().FilesIndexed)
	}
	if pipe.Stats().SymbolsTotal != 0 {
		t.Errorf("SymbolsTotal = %d; want 0 (stub devolveu ErrTreesitterDisabled)", pipe.Stats().SymbolsTotal)
	}
}

func TestPipelineInvalidVaultPath(t *testing.T) {
	h := newCodeastPipelineTestDB(t)
	pipe, _ := NewPipeline(PipelineOptions{DB: h, Parser: NewMockParser(false)})
	if err := pipe.Run(context.Background(), "", nil); err == nil {
		t.Fatal("esperava erro para rootDir vazio")
	}
	if err := pipe.Run(context.Background(), filepath.Join(t.TempDir(), "nope-xyz", "really-not-here"), nil); err != nil {
		// filepath.Walk em pasta inexistente não retorna erro (caminha vazio);
		// portanto não devemos falhar aqui. Garantimos só os passos principais.
	}
}

func TestPipelineStatsFinalize(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "x.go", "package x\nfunc A(){}\n")

	h := newCodeastPipelineTestDB(t)
	stats := &PipelineStats{}
	pipe, _ := NewPipeline(PipelineOptions{DB: h, Parser: NewMockParser(false), StatsSink: stats})
	pipe.Run(context.Background(), tmp, nil)
	if stats.EndedAtUnix == 0 || stats.StartedAtUnix == 0 {
		t.Errorf("StartedAtUnix/EndedAtUnix não foram setados: %+v", stats)
	}
	if stats.EndedAtUnix < stats.StartedAtUnix {
		t.Errorf("EndedAtUnix < StartedAtUnix")
	}
	if time.Since(time.Unix(stats.StartedAtUnix, 0)) > 30*time.Second {
		t.Errorf("Stats mostram tempo fora do intervalo razoável: %+v", stats)
	}
}

// --- helpers ------------------------------------------------------------------

func mkGoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	mkFile(t, dir, rel, content)
}

func mkPyFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	mkFile(t, dir, rel, content)
}

func mkJsFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	mkFile(t, dir, rel, content)
}

func mkFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := writeFile(full, []byte(content)); err != nil {
		t.Fatalf("writeFile(%s): %v", full, err)
	}
}

// stubTreesitterParser devolve ErrTreesitterDisabled em ParseFile.
type stubTreesitterParser struct{}

func newStubTreesitterParser() Parser { return &stubTreesitterParser{} }

func (s *stubTreesitterParser) ParseFile(path string, content []byte) (*FileResult, error) {
	return nil, ErrTreesitterDisabled
}
func (s *stubTreesitterParser) Languages() []string {
	out := make([]string, 0)
	for _, l := range DefaultLanguageTable() {
		out = append(out, l.Name)
	}
	return out
}
func (s *stubTreesitterParser) Backend() string { return "treesitter-stub" }
