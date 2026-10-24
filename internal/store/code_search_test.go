package store

import (
	"context"
	"strings"
	"testing"
)

func TestBuildCodeSymbolQuery_SQLiteUsaInterrogacao(t *testing.T) {
	sqlText, args := BuildCodeSymbolQuery("Search hybrid", "", "", "default", 5, SQLitePlaceholder)

	if sqlText == "" {
		t.Fatal("esperava query não vazia para uma query com tokens")
	}
	if strings.Contains(sqlText, "$1") {
		t.Errorf("o dialeto SQLite não pode usar marcador ordinal: %s", sqlText)
	}
	if !strings.Contains(sqlText, "s.name LIKE ?") {
		t.Errorf("esperava casar o token em s.name: %s", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") {
		t.Errorf("esperava LIMIT com marcador do SQLite: %s", sqlText)
	}
	if !strings.Contains(sqlText, "FROM code_symbols s") {
		t.Errorf("esperava a tabela code_symbols: %s", sqlText)
	}

	// 2 tokens × 3 colunas + repositório (2) + limite = 9 argumentos
	if len(args) != 9 {
		t.Fatalf("esperava 9 argumentos, obteve %d (%v)", len(args), args)
	}
	if args[0] != "%search%" || args[3] != "%hybrid%" {
		t.Errorf("esperava os tokens em minúsculas nos argumentos, obteve %v", args[:6])
	}
	if args[6] != "default" || args[7] != "default" {
		t.Errorf("esperava o repositório nos argumentos 7 e 8, obteve %v", args[6:8])
	}
	if args[8] != 5 {
		t.Errorf("esperava o limite 5 no último argumento, obteve %v", args[8])
	}
}

func TestBuildCodeSymbolQuery_PostgresUsaOrdinal(t *testing.T) {
	sqlText, args := BuildCodeSymbolQuery("Search", "", "", "default", 5, PostgresPlaceholder)

	if sqlText == "" {
		t.Fatal("esperava query não vazia para uma query com tokens")
	}
	if strings.Contains(sqlText, "?") {
		t.Errorf("o dialeto Postgres não pode usar interrogação: %s", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT $6") {
		t.Errorf("esperava LIMIT com marcador ordinal $6: %s", sqlText)
	}
	// 1 token × 3 colunas + repositório (2) + limite = 6 argumentos
	if len(args) != 6 {
		t.Fatalf("esperava 6 argumentos, obteve %d (%v)", len(args), args)
	}
	if args[5] != 5 {
		t.Errorf("esperava o limite 5 no último argumento, obteve %v", args[5])
	}
}

func TestBuildCodeSymbolQuery_SemTokenNaoBusca(t *testing.T) {
	sqlText, args := BuildCodeSymbolQuery("   ", "", "", "default", 5, SQLitePlaceholder)

	if sqlText != "" {
		t.Errorf("esperava query vazia sem tokens, obteve: %s", sqlText)
	}
	if args != nil {
		t.Errorf("esperava argumentos nulos sem tokens, obteve: %v", args)
	}
}

func TestTokenizeCodeQuery(t *testing.T) {
	got := TokenizeCodeQuery("Search, Hybrid; search")
	want := []string{"search", "hybrid"}

	if len(got) != len(want) {
		t.Fatalf("esperava %v, obteve %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token %d: esperava %q, obteve %q", i, want[i], got[i])
		}
	}
}

func TestApplyCodeSymbolBoost_CA05(t *testing.T) {
	hits := []CodeSymbolHit{
		{SymbolID: 1, Name: "Search", QualifiedName: "pkg.Search", Score: 1.0},
		{SymbolID: 2, Name: "Other", QualifiedName: "pkg.Other", Score: 1.0},
	}

	ApplyCodeSymbolBoost(hits, "pkg.Search")

	if !hits[0].Boosted {
		t.Error("esperava o símbolo com qualified name exato como boosted")
	}
	if hits[0].Score != 2.0 {
		t.Errorf("esperava score 2.0 no símbolo boosted, obteve %v", hits[0].Score)
	}
	if hits[1].Boosted {
		t.Error("símbolo sem correspondência exata não pode ser boosted")
	}
	if hits[1].Score != 1.0 {
		t.Errorf("esperava score 1.0 no símbolo não boosted, obteve %v", hits[1].Score)
	}
}

func TestApplyCodeSymbolBoost_SemFormaDeQualifiedName(t *testing.T) {
	hits := []CodeSymbolHit{
		{SymbolID: 1, Name: "Search", QualifiedName: "pkg.Search", Score: 1.0},
	}

	ApplyCodeSymbolBoost(hits, "search")

	if hits[0].Boosted || hits[0].Score != 1.0 {
		t.Errorf("esperava o símbolo intacto (query sem forma de qualified name), obteve %+v", hits[0])
	}
}

func TestSortCodeSymbolHits_BoostReordena(t *testing.T) {
	hits := []CodeSymbolHit{
		{SymbolID: 1, Score: 1.0},
		{SymbolID: 2, Score: 2.0},
	}

	SortCodeSymbolHits(hits)

	if hits[0].SymbolID != 2 {
		t.Errorf("esperava o símbolo boosted em primeiro, obteve id %d", hits[0].SymbolID)
	}
}

func TestCodeSymbolResults_IdentidadeEOrigem(t *testing.T) {
	hits := []CodeSymbolHit{
		{
			SymbolID:      7,
			Kind:          "function",
			Name:          "Search",
			QualifiedName: "pkg.Search",
			Signature:     "func Search(query string) error",
			DocComment:    "Search busca o símbolo.",
			StartLine:     42,
			EndLine:       60,
			File:          "pkg/search.go",
			Language:      "go",
		},
	}

	results := CodeSymbolResults(hits)
	if len(results) != 1 {
		t.Fatalf("esperava 1 resultado, obteve %d", len(results))
	}

	res := results[0]
	if res.DocumentID != "code:pkg.Search" {
		t.Errorf("esperava DocumentID 'code:pkg.Search', obteve %q", res.DocumentID)
	}
	if res.ChunkID != "" {
		t.Errorf("símbolo de código não tem chunk, obteve %q", res.ChunkID)
	}
	if res.CodeSymbol != "pkg.Search" {
		t.Errorf("esperava CodeSymbol 'pkg.Search', obteve %q", res.CodeSymbol)
	}
	if res.CodeFile != "pkg/search.go" {
		t.Errorf("esperava CodeFile 'pkg/search.go', obteve %q", res.CodeFile)
	}
	if res.CodeLine != 42 {
		t.Errorf("esperava CodeLine 42, obteve %d", res.CodeLine)
	}
	if res.CodeKind != "function" {
		t.Errorf("esperava CodeKind 'function', obteve %q", res.CodeKind)
	}
	if !strings.Contains(res.Abstract, "pkg/search.go:42") {
		t.Errorf("esperava a origem no abstract, obteve %q", res.Abstract)
	}
	if !strings.Contains(res.Content, "pkg/search.go:42") {
		t.Errorf("esperava a origem no conteúdo, obteve %q", res.Content)
	}
	if !strings.Contains(res.Abstract, "func Search(query string) error") {
		t.Errorf("esperava a assinatura no abstract, obteve %q", res.Abstract)
	}
}

func TestIncludeCodeRequested_ContextoEOpcoes(t *testing.T) {
	if IncludeCodeRequested(context.Background(), SearchOptions{}) {
		t.Error("sem include_code o fan-in de código não pode ser pedido")
	}
	if !IncludeCodeRequested(WithIncludeCode(context.Background(), true), SearchOptions{}) {
		t.Error("a flag do contexto deve pedir o fan-in de código")
	}
	if !IncludeCodeRequested(context.Background(), SearchOptions{IncludeCode: true}) {
		t.Error("a flag das opções deve pedir o fan-in de código")
	}
}
