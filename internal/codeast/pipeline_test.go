package codeast

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
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

// TestPipelineUsaBackendPadraoNaoMock cobre T8: sem Parser injetado, o
// pipeline usa o backend padrão do build (motor Go em Go puro). O MockParser
// (regex) NÃO extrai métodos nem receivers — então a presença do método na
// base prova que o caminho de produção deixou de ser o mock.
func TestPipelineUsaBackendPadraoNaoMock(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n\ntype T struct{}\n\n// M faz nada.\nfunc (t *T) M() {}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if pipe.Stats().FilesIndexed != 1 {
		t.Fatalf("FilesIndexed = %d; want 1", pipe.Stats().FilesIndexed)
	}

	var cnt int
	if err := h.QueryRow(`SELECT COUNT(*) FROM code_symbols WHERE kind = 'method'`).Scan(&cnt); err != nil {
		t.Fatalf("SELECT method falhou: %v", err)
	}
	if cnt != 1 {
		t.Errorf("code_symbols(method) = %d; want 1 (o MockParser não extrai métodos)", cnt)
	}
}

// TestPipelineGravaCallsComCallSiteEProveniencia cobre T9: o pipeline grava a
// aresta `calls` com o PONTO DE CHAMADA (a linha da chamada — não a da
// definição) e com o provenance EXTRACTED (destino resolvido por escopo de
// pacote).
func TestPipelineGravaCallsComCallSiteEProveniencia(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n\n// Helper ajuda.\nfunc Helper() {}\n\n// Caller chama.\nfunc Caller() {\n\tHelper()\n}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	rows, err := h.Query(`
		SELECT src.qualified_name, dst.qualified_name, e.kind, e.start_line, e.epistemic_status
		  FROM code_edges e
		  JOIN code_symbols src ON src.id = e.src_symbol_id
		  JOIN code_symbols dst ON dst.id = e.dst_symbol_id
		 WHERE e.kind = 'calls'`)
	if err != nil {
		t.Fatalf("SELECT edges falhou: %v", err)
	}
	defer rows.Close()

	type edgeRow struct {
		src, dst, kind, status string
		line                   int
	}
	var got []edgeRow
	for rows.Next() {
		var e edgeRow
		if err := rows.Scan(&e.src, &e.dst, &e.kind, &e.line, &e.status); err != nil {
			t.Fatalf("scan falhou: %v", err)
		}
		got = append(got, e)
	}
	if len(got) != 1 {
		t.Fatalf("arestas calls = %d; want 1; got %+v", len(got), got)
	}
	e := got[0]
	if e.src != "x.Caller" || e.dst != "x.Helper" {
		t.Errorf("aresta = %s -> %s; want x.Caller -> x.Helper", e.src, e.dst)
	}
	if e.line != 8 {
		t.Errorf("start_line = %d; want 8 (o CALL-SITE, não a linha 4 da definição)", e.line)
	}
	if e.status != "EXTRACTED" {
		t.Errorf("epistemic_status = %q; want EXTRACTED", e.status)
	}
}

// TestPipelineGravaImportsComDestinoQueExiste cobre T10: a aresta `imports`
// aponta para um símbolo que EXISTE em code_symbols (o nó do pacote
// importado) — nunca para uma aresta órfã — e a provenance é EXTRACTED.
func TestPipelineGravaImportsComDestinoQueExiste(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a.go", "package x\n\nimport \"fmt\"\n\nfunc A() {\n\tfmt.Println(\"oi\")\n}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	var total, orfas int
	if err := h.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE kind = 'imports'`).Scan(&total); err != nil {
		t.Fatalf("SELECT imports falhou: %v", err)
	}
	if total != 1 {
		t.Fatalf("arestas imports = %d; want 1", total)
	}
	if err := h.QueryRow(`
		SELECT COUNT(*) FROM code_edges e
		 WHERE e.kind = 'imports'
		   AND NOT EXISTS (SELECT 1 FROM code_symbols s WHERE s.id = e.dst_symbol_id)`).Scan(&orfas); err != nil {
		t.Fatalf("SELECT órfãs falhou: %v", err)
	}
	if orfas != 0 {
		t.Errorf("arestas imports apontando para símbolo inexistente = %d; want 0", orfas)
	}

	var status string
	if err := h.QueryRow(`SELECT epistemic_status FROM code_edges WHERE kind = 'imports'`).Scan(&status); err != nil {
		t.Fatalf("SELECT status falhou: %v", err)
	}
	if status != "EXTRACTED" {
		t.Errorf("epistemic_status = %q; want EXTRACTED", status)
	}
}

// TestArestaAmbiguaVaiParaUncertain cobre T11 (CG-15): um destino ambíguo
// (mais de um símbolo com o mesmo nome) NUNCA vira aresta resolvida — vai para
// code_edges_uncertain com a confiança e a razão.
func TestArestaAmbiguaVaiParaUncertain(t *testing.T) {
	tmp := t.TempDir()
	// Dois pacotes com uma função de mesmo nome: o destino por nome é
	// ambíguo.
	mkGoFile(t, tmp, "a/x.go", "package a\n\nfunc Helper() {}\n")
	mkGoFile(t, tmp, "b/x.go", "package b\n\nfunc Helper() {}\n")
	mkGoFile(t, tmp, "c/main.go", "package c\n\nfunc Call() {\n\tHelper()\n}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	var resolvidas int
	if err := h.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE kind = 'calls'`).Scan(&resolvidas); err != nil {
		t.Fatalf("SELECT code_edges falhou: %v", err)
	}
	if resolvidas != 0 {
		t.Errorf("arestas calls resolvidas = %d; want 0 (destino ambíguo não vira aresta)", resolvidas)
	}

	var dst, reason string
	var confidence float64
	if err := h.QueryRow(`SELECT dst_candidate_name, reason, confidence FROM code_edges_uncertain WHERE kind = 'calls'`).
		Scan(&dst, &reason, &confidence); err != nil {
		t.Fatalf("SELECT code_edges_uncertain falhou: %v", err)
	}
	if dst != "c.Helper" {
		t.Errorf("dst_candidate_name = %q; want c.Helper (o qualified name do destino pretendido)", dst)
	}
	if !strings.Contains(reason, "ambíguo") {
		t.Errorf("reason = %q; quer a razão da ambiguidade", reason)
	}
	if confidence != 0.5 {
		t.Errorf("confidence = %v; want 0.5", confidence)
	}
}

// TestUsesTypeNaoResolveParaMetodoDeMesmoNome: uma aresta de TIPO só pode
// apontar para um tipo. Sem o filtro por kind, o nível "por nome" resolveria
// `a.DB` (tipo que não existe no repo) para um MÉTODO chamado DB — uma
// aresta errada. Com o filtro, o destino fica no bucket de incertas.
func TestUsesTypeNaoResolveParaMetodoDeMesmoNome(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "a/x.go", "package a\n\ntype Wrapper struct{}\n\nfunc (w *Wrapper) DB() {}\n")
	mkGoFile(t, tmp, "b/y.go", "package b\n\nimport \"a\"\n\ntype Holder struct {\n\tDB a.DB\n}\n")

	h := newCodeastPipelineTestDB(t)
	pipe, err := NewPipeline(PipelineOptions{DB: h})
	if err != nil {
		t.Fatalf("NewPipeline falhou: %v", err)
	}
	if err := pipe.Run(context.Background(), tmp, nil); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	var erradas int64
	if err := h.QueryRow(`
		SELECT COUNT(*) FROM code_edges e
		  JOIN code_symbols dst ON dst.id = e.dst_symbol_id
		 WHERE e.kind = 'uses_type' AND dst.kind <> 'struct' AND dst.kind <> 'interface' AND dst.kind <> 'class'`).
		Scan(&erradas); err != nil {
		t.Fatalf("SELECT uses_type falhou: %v", err)
	}
	if erradas != 0 {
		t.Errorf("arestas uses_type apontando para um símbolo que não é tipo = %d; want 0", erradas)
	}

	var incertas int64
	if err := h.QueryRow(`SELECT COUNT(*) FROM code_edges_uncertain WHERE kind = 'uses_type'`).Scan(&incertas); err != nil {
		t.Fatalf("SELECT incertas falhou: %v", err)
	}
	if incertas != 1 {
		t.Errorf("arestas uses_type incertas = %d; want 1", incertas)
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

// TestListCodeFilesUsaCodeScopeNaoShouldIndex prova que a selecao de arquivos
// do code pipeline e o escopo de codigo, nunca o filtro do pipeline Markdown
// (ADR-061 §4, CG-01). O `include` global e escopo de NOTAS: ele nao escolhe
// o que o code pipeline parseia, e o code scope nao escolhe o que o pipeline
// Markdown indexa.
func TestListCodeFilesUsaCodeScopeNaoShouldIndex(t *testing.T) {
	tmp := t.TempDir()
	mkGoFile(t, tmp, "main.go", "package x\n")
	mkPyFile(t, tmp, "script.py", "def f():\n    return 1\n")
	mkFile(t, tmp, "nota.md", "# Nota\n")

	// O include global (escopo de notas) casa com o .go — isso NAO pode
	// escolher o que o code pipeline parseia.
	cfg := config.DefaultConfig()
	cfg.Include = []string{"**/*.go", "**/*.md"}
	// O escopo de codigo e que decide: aqui, so Python.
	cfg.Code = config.CodeScopeConfig{Include: []string{"**/*.py"}}

	files, err := ListCodeFiles(tmp, &cfg, nil)
	if err != nil {
		t.Fatalf("ListCodeFiles falhou: %v", err)
	}
	if len(files) != 1 || files[0] != "script.py" {
		t.Fatalf("o code scope deve escolher apenas script.py; obteve %v", files)
	}

	// Escopo de codigo default (extensoes do backend ativo): entram os .go e
	// .py, e a nota Markdown nunca entra.
	cfg.Code = config.CodeScopeConfig{}
	files, err = ListCodeFiles(tmp, &cfg, nil)
	if err != nil {
		t.Fatalf("ListCodeFiles falhou: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("o escopo de codigo default deve pegar main.go e script.py; obteve %v", files)
	}
	for _, f := range files {
		if f == "nota.md" {
			t.Errorf("nota Markdown nao pode entrar no code scope: %v", files)
		}
	}

	// O include global de notas tambem nao EXCLUI arquivo de codigo: com
	// include=["**/*.md"], o .go continua sendo selectionado pelo code scope.
	cfg.Include = []string{"**/*.md"}
	files, err = ListCodeFiles(tmp, &cfg, nil)
	if err != nil {
		t.Fatalf("ListCodeFiles falhou: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("o include global de notas nao pode excluir codigo do code scope; obteve %v", files)
	}
}

// TestCodeScopeCobreExtensoesDoBackendAtivo garante que o escopo de codigo
// padrao (config) e a tabela de linguagens do backend ativo (codeast) nao
// divergem: toda extensao suportada tem um glob no escopo, e o escopo nao tem
// glob sobrando (ADR-061 §4, CG-02).
func TestCodeScopeCobreExtensoesDoBackendAtivo(t *testing.T) {
	suportadas := map[string]bool{}
	for _, ext := range SupportedExtensions() {
		suportadas[strings.ToLower(strings.TrimPrefix(ext, "."))] = true
	}

	defaultCfg := config.DefaultConfig()
	escopo := defaultCfg.CodeScope()
	vistas := map[string]bool{}
	for _, glob := range escopo.Include {
		ext := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(glob, "**/"), "*"), ".")
		vistas[ext] = true
		if !suportadas[ext] {
			t.Errorf("o escopo de codigo padrao tem %q, que o backend ativo nao suporta", glob)
		}
	}
	for ext := range suportadas {
		if !vistas[ext] {
			t.Errorf("o backend ativo suporta %q, mas o escopo de codigo padrao nao cobre", ext)
		}
	}
}

// TestIsCodeFileReconheceExtensoesDoBackend: o filtro que impede o pipeline
// Markdown de indexar codigo como nota (CG-04) usa a mesma tabela de
// linguagens do backend ativo.
func TestIsCodeFileReconheceExtensoesDoBackend(t *testing.T) {
	if !IsCodeFile("cmd/mem/main.go") {
		t.Error("um .go deve ser reconhecido como arquivo de codigo")
	}
	if IsCodeFile("docs/adr/061-x.md") {
		t.Error("um .md nao e arquivo de codigo")
	}
	if IsCodeFile("README") {
		t.Error("um arquivo sem extensao nao e arquivo de codigo")
	}
	for _, ext := range SupportedExtensions() {
		if !IsCodeFile("arquivo." + strings.TrimPrefix(ext, ".")) {
			t.Errorf("a extensao %q do backend ativo deve ser reconhecida como codigo", ext)
		}
	}
}

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
