package codeast

import (
	"context"
	"testing"
)

// TestBuildNaoDegradaOcodeGraph prova o defeito D3: o build com a tag
// `-tags treesitter` NÃO pode esvaziar o code graph em silêncio.
//
// Antes da correção, o stub de treesitter_enabled.go devolvia
// ErrTreesitterDisabled em todo ParseFile e o pipeline tratava isso como
// no-op: o build com a tag produzia 1 símbolo e 0 arestas (files_indexed=4,
// files_errors=0, exit 0) enquanto o build padrão produzia 8 símbolos e
// 4 arestas.
//
// O teste roda nos DOIS builds e exige o mesmo resultado.
func TestBuildNaoDegradaOcodeGraph(t *testing.T) {
	// 1. O backend do build em execução não pode ser o stub tree-sitter.
	parser, err := newBackendParser()
	if err != nil {
		t.Fatalf("newBackendParser falhou: %v", err)
	}
	if parser == nil {
		t.Fatal("newBackendParser devolveu nil")
	}
	if parser.Backend() != "go-stdlib" {
		t.Fatalf("Backend() = %q; want go-stdlib (o build com a tag nao pode sombrear o motor real)", parser.Backend())
	}

	// 2. O parser extrai símbolos de verdade.
	res, err := parser.ParseFile("pkg/exemplo.go", []byte("package x\n\n// F faz nada.\nfunc F() {}\n"))
	if err != nil {
		t.Fatalf("ParseFile falhou: %v", err)
	}
	if res == nil || len(res.Symbols) == 0 {
		t.Fatalf("o backend do build não extraiu símbolos: %+v", res)
	}

	// 3. O code stage (pipeline completo) grava símbolos E arestas.
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n\nfunc A() { B() }\n\nfunc B() {}\n")
	mkGoFile(t, tmp, "b.go", "package x\n\nfunc C() { A() }\n")

	database := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: database})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	stats := pipe.Stats()
	if stats.SymbolsTotal == 0 {
		t.Fatalf("o code stage não extraiu símbolos (build degradado): %+v", stats)
	}
	if stats.FilesErrors != 0 {
		t.Errorf("o code stage acusou erros de arquivo: %+v", stats)
	}
	t.Logf("code stage (build com tag=%v): files_indexed=%d symbols_total=%d",
		IsTreesitterEnabled(), stats.FilesIndexed, stats.SymbolsTotal)
}

// TestBuildComTagNaoDivergeDoPadrao fixa o número esperado de símbolos e
// arestas do code stage. O MESMO teste roda no build padrão e no build com
// `-tags treesitter`: os números têm que ser idênticos (é a prova
// empírica de que a tag não degrada o grafo).
func TestBuildComTagNaoDivergeDoPadrao(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n\nfunc A() { B() }\n\nfunc B() {}\n")
	mkGoFile(t, tmp, "b.go", "package x\n\nfunc C() { A() }\n")

	database := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: database})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	stats := pipe.Stats()

	// 3 funções (A, B, C) + 2 arestas de chamada (A→B, C→A).
	if stats.SymbolsTotal != 3 {
		t.Errorf("SymbolsTotal = %d; want 3 (o build nao pode degradar o code graph)", stats.SymbolsTotal)
	}

	var edges int
	if err := database.QueryRow("SELECT COUNT(*) FROM code_edges").Scan(&edges); err != nil {
		t.Fatalf("count code_edges: %v", err)
	}
	if edges != 2 {
		t.Errorf("code_edges = %d; want 2 (o build nao pode degradar o code graph)", edges)
	}
	t.Logf("RESULTADO build: symbols=%d edges=%d files_indexed=%d files_errors=%d",
		stats.SymbolsTotal, edges, stats.FilesIndexed, stats.FilesErrors)
}
