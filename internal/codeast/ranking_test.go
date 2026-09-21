package codeast

import (
	"sort"
	"testing"
)

func TestRanking_HasQualifiedNameShape(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"foo", false},             // sem ponto
		{"foo.bar", true},          // dois segmentos simples
		{"pkg.FuncName", true},     // camelCase
		{"pkg.Sub.Type", true},     // três segmentos
		{"my_pkg.helper_fn", true}, // snake_case
		{"foo. bar", true},         // trim ao redor dos segmentos é tolerado
		{"foo.123bar", false},      // identificador começa com dígito
		{"foo bar", false},         // espaço no lugar do ponto
		{".foo", false},            // ponto inicial
		{"foo.", false},            // segmento vazio
		{"foo..bar", false},        // segmento vazio no meio
		{"foo-bar.baz", false},     // hífen
	}
	for _, c := range cases {
		if got := HasQualifiedNameShape(c.in); got != c.want {
			t.Errorf("HasQualifiedNameShape(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}

func TestRanking_IsQualifiedNameMatchExact(t *testing.T) {
	if !IsQualifiedNameMatchExact("pkg.Func", "pkg.Func") {
		t.Error("exact match deve retornar true")
	}
	if !IsQualifiedNameMatchExact("  pkg.Func  ", "pkg.Func") {
		t.Error("trim não deve impedir match")
	}
	if IsQualifiedNameMatchExact("pkg.func", "pkg.Func") {
		t.Error("case-diferente não deve casar (case-sensitive)")
	}
	if IsQualifiedNameMatchExact("Foo", "pkg.Foo") {
		t.Error("substring não deve casar (deve ser exato)")
	}
	if IsQualifiedNameMatchExact("", "pkg.Foo") {
		t.Error("query vazio não deve casar")
	}
	if IsQualifiedNameMatchExact("pkg.Foo", "") {
		t.Error("qualified_name vazio não deve casar")
	}
}

func TestRanking_ShouldApplyBoost(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		qualified string
		noBoost   bool
		wantApply bool
	}{
		{"exact match no flag", "pkg.Func", "pkg.Func", false, true},
		{"exact match with no-boost flag", "pkg.Func", "pkg.Func", true, false},
		{"query sem dot", "Func", "pkg.Func", false, false},
		{"query difere", "pkg.Other", "pkg.Func", false, false},
		{"query com dot mas sem match", "pkg", "pkg.Func", false, false},
		{"case-diferente", "pkg.func", "pkg.Func", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			apply, _ := ShouldApplyBoost(c.query, c.qualified, c.noBoost)
			if apply != c.wantApply {
				t.Errorf("ShouldApplyBoost(%q,%q,%v) = %v; want %v", c.query, c.qualified, c.noBoost, apply, c.wantApply)
			}
		})
	}
}

func TestRanking_ApplyBoost(t *testing.T) {
	// Com match: 1.0 * 2 = 2.0
	score, boosted := ApplyBoost(1.0, "pkg.Func", "pkg.Func", false)
	if !boosted || score != 2.0 {
		t.Errorf("ApplyBoost com match: got (%.2f,%v); want (2.0,true)", score, boosted)
	}
	// Sem match: 1.0 inalterado
	score, boosted = ApplyBoost(1.0, "pkg.Func", "pkg.Other", false)
	if boosted || score != 1.0 {
		t.Errorf("ApplyBoost sem match: got (%.2f,%v); want (1.0,false)", score, boosted)
	}
	// Com noBoost: 1.0 inalterado
	score, boosted = ApplyBoost(1.0, "pkg.Func", "pkg.Func", true)
	if boosted || score != 1.0 {
		t.Errorf("ApplyBoost com no-boost: got (%.2f,%v); want (1.0,false)", score, boosted)
	}
	// Score negativo é normalizado para 0 (defensivo)
	score, boosted = ApplyBoost(-1.0, "pkg.Func", "pkg.Func", false)
	if !boosted || score != 0.0 {
		t.Errorf("ApplyBoost(-1) deve normalizar: got (%.2f,%v); want (0.0,true)", score, boosted)
	}
}

func TestRanking_ApplyBoostToHits_OrderingAndBoost(t *testing.T) {
	hits := []CodeSearchHit{
		{SymbolID: 10, QualifiedName: "pkg.Other", RRFScore: 1.0},
		{SymbolID: 20, QualifiedName: "pkg.Func", RRFScore: 1.0},
		{SymbolID: 30, QualifiedName: "a.b.c", RRFScore: 0.5},
	}
	out := ApplyBoostToHits(hits, "pkg.Func", false)
	if len(out) != 3 {
		t.Fatalf("len(out)=%d; want 3", len(out))
	}
	// Esperado ordem: SymbolID 20 (boosted 2.0) > 10 (1.0) > 30 (0.5)
	if out[0].SymbolID != 20 || !out[0].Boosted || out[0].RRFScore != 2.0 {
		t.Errorf("hit[0]=%+v; want SymbolID=20 boosted=true score=2.0", out[0])
	}
	if out[1].SymbolID != 10 || out[1].Boosted {
		t.Errorf("hit[1]=%+v; want SymbolID=10 boosted=false", out[1])
	}
	if out[2].SymbolID != 30 || out[2].Boosted {
		t.Errorf("hit[2]=%+v; want SymbolID=30 boosted=false", out[2])
	}
}

func TestRanking_ApplyBoostToHits_NoBoostFlag(t *testing.T) {
	hits := []CodeSearchHit{
		{SymbolID: 1, QualifiedName: "pkg.Func", RRFScore: 1.0},
	}
	out := ApplyBoostToHits(hits, "pkg.Func", true)
	if out[0].Boosted {
		t.Error("com noBoost=true, hit não pode ser boosted")
	}
	if out[0].RRFScore != 1.0 {
		t.Errorf("score=%.2f; want 1.0", out[0].RRFScore)
	}
}

func TestRanking_ApplyBoostToHits_Empty(t *testing.T) {
	out := ApplyBoostToHits(nil, "pkg.Func", false)
	if out != nil {
		t.Errorf("ApplyBoostToHits(nil) deve devolver nil; got %v", out)
	}
	out2 := ApplyBoostToHits([]CodeSearchHit{}, "x", false)
	if len(out2) != 0 {
		t.Errorf("ApplyBoostToHits([]) deve devolver slice vazio")
	}
}

func TestRanking_ComputeRRFScore(t *testing.T) {
	ranks := map[int64]int{1: 1, 2: 2, 3: 5}
	score := ComputeRRFScore(ranks, 60)
	expected := 1.0/61.0 + 1.0/62.0 + 1.0/65.0
	if absDiff(score, expected) > 1e-9 {
		t.Errorf("ComputeRRFScore = %.6f; want %.6f", score, expected)
	}
	// k<=0 cai para default 60
	if ComputeRRFScore(ranks, 0) != score {
		t.Error("ComputeRRFScore com k=0 deve usar default 60")
	}
	// rank<=0 ignorado
	ranksZero := map[int64]int{1: 1, 2: 0, 3: -1}
	score2 := ComputeRRFScore(ranksZero, 60)
	if score2 != 1.0/61.0 {
		t.Errorf("rank<=0 deve ser ignorado: got %.6f, want %.6f", score2, 1.0/61.0)
	}
	// Ordenação estável: ties preservam ordem original
	// Original: [5 (0.1), 3 (0.1), 1 (0.2)] → após sort desc:
	// [1 (0.2), 5 (0.1), 3 (0.1)] — 5 antes de 3 preservando entrada.
	hits := []CodeSearchHit{
		{SymbolID: 5, RRFScore: 0.1},
		{SymbolID: 3, RRFScore: 0.1},
		{SymbolID: 1, RRFScore: 0.2},
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].RRFScore > hits[j].RRFScore })
	if hits[0].SymbolID != 1 {
		t.Errorf("hits[0].SymbolID = %d; want 1 (maior score)", hits[0].SymbolID)
	}
	if hits[1].SymbolID != 5 || hits[2].SymbolID != 3 {
		t.Errorf("ordenação estável violada: got [%d,%d]; want [5,3]", hits[1].SymbolID, hits[2].SymbolID)
	}
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
