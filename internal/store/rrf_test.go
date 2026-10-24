package store

import (
	"math"
	"testing"
)

func TestFuseRRF_MathematicalCorrectness(t *testing.T) {
	sources := []RankedSource[string]{
		{
			Name:  "fts",
			Items: []string{"A", "B", "C"},
		},
		{
			Name:  "vector",
			Items: []string{"B", "D", "A"},
		},
	}

	// k = 60
	results := FuseRRF(sources, 60)

	if len(results) != 4 {
		t.Fatalf("Esperava 4 itens únicos, obteve %d", len(results))
	}

	// B deve ser o primeiro colocado: fts:2 (1/62) + vector:1 (1/61)
	expectedB := (1.0 / 62.0) + (1.0 / 61.0)
	if results[0].Item != "B" {
		t.Errorf("Primeiro item deveria ser 'B', obteve '%s'", results[0].Item)
	}
	if math.Abs(results[0].Score-expectedB) > 1e-9 {
		t.Errorf("Score de 'B' incorreto: esperado %f, obteve %f", expectedB, results[0].Score)
	}
	if len(results[0].Sources) != 2 {
		t.Errorf("Esperava 2 fontes para 'B', obteve %d", len(results[0].Sources))
	}

	// A deve ser o segundo colocado: fts:1 (1/61) + vector:3 (1/63)
	expectedA := (1.0 / 61.0) + (1.0 / 63.0)
	if results[1].Item != "A" {
		t.Errorf("Segundo item deveria ser 'A', obteve '%s'", results[1].Item)
	}
	if math.Abs(results[1].Score-expectedA) > 1e-9 {
		t.Errorf("Score de 'A' incorreto: esperado %f, obteve %f", expectedA, results[1].Score)
	}

	// D deve ser o terceiro colocado: vector:2 (1/62)
	expectedD := 1.0 / 62.0
	if results[2].Item != "D" {
		t.Errorf("Terceiro item deveria ser 'D', obteve '%s'", results[2].Item)
	}
	if math.Abs(results[2].Score-expectedD) > 1e-9 {
		t.Errorf("Score de 'D' incorreto: esperado %f, obteve %f", expectedD, results[2].Score)
	}

	// C deve ser o quarto colocado: fts:3 (1/63)
	expectedC := 1.0 / 63.0
	if results[3].Item != "C" {
		t.Errorf("Quarto item deveria ser 'C', obteve '%s'", results[3].Item)
	}
	if math.Abs(results[3].Score-expectedC) > 1e-9 {
		t.Errorf("Score de 'C' incorreto: esperado %f, obteve %f", expectedC, results[3].Score)
	}
}

func TestFuseRRF_DefaultK(t *testing.T) {
	sources := []RankedSource[string]{
		{
			Name:  "fts",
			Items: []string{"X"},
		},
	}

	// k = 0 deve assumir DefaultRRFK = 60
	results := FuseRRF(sources, 0)
	expectedScore := 1.0 / 61.0

	if len(results) != 1 {
		t.Fatalf("Esperava 1 resultado, obteve %d", len(results))
	}
	if math.Abs(results[0].Score-expectedScore) > 1e-9 {
		t.Errorf("k <= 0 não usou default 60: esperado %f, obteve %f", expectedScore, results[0].Score)
	}
}

func TestFuseRRF_Empty(t *testing.T) {
	results := FuseRRF[string](nil, 60)
	if len(results) != 0 {
		t.Errorf("Esperava slice vazio para entrada nula, obteve %d itens", len(results))
	}

	sources := []RankedSource[string]{
		{Name: "empty", Items: []string{}},
	}
	results = FuseRRF(sources, 60)
	if len(results) != 0 {
		t.Errorf("Esperava slice vazio para fontes vazias, obteve %d itens", len(results))
	}
}

func TestFuseSearchResults(t *testing.T) {
	sources := []RankedResultSource{
		{
			Name: "fts",
			Results: []SearchResult{
				{ChunkID: "c1", DocumentID: "docA", Content: "Texto A", Neighbors: []string{"docB"}},
				{ChunkID: "c2", DocumentID: "docB", Content: "Texto B"},
			},
		},
		{
			Name: "vector",
			Results: []SearchResult{
				{ChunkID: "c2", DocumentID: "docB", Content: "Texto B", Distance: 0.12},
				{ChunkID: "c3", DocumentID: "docC", Content: "Texto C"},
			},
		},
		{
			Name: "graph",
			Results: []SearchResult{
				{ChunkID: "c1", DocumentID: "docA", Neighbors: []string{"docC"}},
			},
		},
	}

	fused := FuseSearchResults(sources, 60, 2)

	if len(fused) != 2 {
		t.Fatalf("Esperava limite de 2 resultados, obteve %d", len(fused))
	}

	// c1 está em fts #1 (1/61) e graph #1 (1/61) -> score = 2/61 ≈ 0.03278688
	// c2 está em fts #2 (1/62) e vector #1 (1/61) -> score = 1/62 + 1/61 ≈ 0.03252247
	if fused[0].ChunkID != "c1" {
		t.Errorf("Primeiro colocado esperado 'c1', obteve '%s'", fused[0].ChunkID)
	}
	if fused[1].ChunkID != "c2" {
		t.Errorf("Segundo colocado esperado 'c2', obteve '%s'", fused[1].ChunkID)
	}

	// Verifica fusão de vizinhos em c1: docB e docC
	if len(fused[0].Neighbors) != 2 {
		t.Errorf("Esperava 2 vizinhos mesclados em c1, obteve %d", len(fused[0].Neighbors))
	}

	// Verifica detalhes de sources
	if len(fused[0].Sources) != 2 {
		t.Errorf("Esperava 2 fontes registradas em c1, obteve %d (%v)", len(fused[0].Sources), fused[0].Sources)
	}
}

func TestFuseRRF_DuplicateItemsInSameSource(t *testing.T) {
	sources := []RankedSource[string]{
		{
			Name:  "fts",
			Items: []string{"A", "A", "B"}, // "A" duplicado na mesma lista não deve acumular score duas vezes da mesma fonte
		},
	}

	results := FuseRRF(sources, 60)
	if len(results) != 2 {
		t.Fatalf("Esperava 2 itens únicos após desduplicação na fonte, obteve %d", len(results))
	}

	expectedScoreA := 1.0 / 61.0
	if math.Abs(results[0].Score-expectedScoreA) > 1e-9 {
		t.Errorf("Score de 'A' acumulou indevidamente da mesma fonte: esperado %f, obteve %f", expectedScoreA, results[0].Score)
	}
}

func TestFuseSearchResults_FallbackToDocumentID(t *testing.T) {
	sources := []RankedResultSource{
		{
			Name: "fts",
			Results: []SearchResult{
				{DocumentID: "doc-sem-chunk", Content: "Conteudo apenas com doc ID"},
			},
		},
	}

	fused := FuseSearchResults(sources, 60, 5)
	if len(fused) != 1 {
		t.Fatalf("Esperava 1 resultado usando fallback de DocumentID, obteve %d", len(fused))
	}
	if fused[0].DocumentID != "doc-sem-chunk" {
		t.Errorf("DocumentID incorreto: obteve '%s'", fused[0].DocumentID)
	}
}

func TestFuseSearchResults_CategoryFilter(t *testing.T) {
	sources := []RankedResultSource{
		{
			Name: "fts",
			Results: []SearchResult{
				{ChunkID: "c1", DocumentID: "doc_mem", Content: "Memória de regras", Category: "memory", Abstract: "Resumo regras"},
				{ChunkID: "c2", DocumentID: "doc_skill", Content: "Skill operacional", Category: "skill", Abstract: "Resumo skill"},
				{ChunkID: "c3", DocumentID: "doc_res", Content: "Recurso arquitetural", Category: "resource", Abstract: "Resumo recurso"},
				{ChunkID: "c4", DocumentID: "doc_def", Content: "Recurso padrão sem categoria", Category: "", Abstract: "Resumo padrão"},
			},
		},
	}

	// 1. Filtro "memory": deve retornar apenas doc_mem
	memResults := FuseSearchResultsWithOptions(sources, 60, 10, DecayOptions{}, SearchOptions{Category: "memory"})
	if len(memResults) != 1 {
		t.Fatalf("Esperava 1 resultado para categoria 'memory', obteve %d", len(memResults))
	}
	if memResults[0].DocumentID != "doc_mem" {
		t.Errorf("Esperava doc_mem, obteve %s", memResults[0].DocumentID)
	}

	// 2. Filtro "skill": deve retornar apenas doc_skill
	skillResults := FuseSearchResultsWithOptions(sources, 60, 10, DecayOptions{}, SearchOptions{Category: "skill"})
	if len(skillResults) != 1 {
		t.Fatalf("Esperava 1 resultado para categoria 'skill', obteve %d", len(skillResults))
	}
	if skillResults[0].DocumentID != "doc_skill" {
		t.Errorf("Esperava doc_skill, obteve %s", skillResults[0].DocumentID)
	}

	// 3. Filtro "resource": deve incluir doc_res e doc_def (pois "" assume padrão resource)
	resResults := FuseSearchResultsWithOptions(sources, 60, 10, DecayOptions{}, SearchOptions{Category: "resource"})
	if len(resResults) != 2 {
		t.Fatalf("Esperava 2 resultados para categoria 'resource', obteve %d", len(resResults))
	}

	// 4. Sem filtro: retorna todos os 4
	allResults := FuseSearchResultsWithOptions(sources, 60, 10, DecayOptions{}, SearchOptions{})
	if len(allResults) != 4 {
		t.Fatalf("Esperava 4 resultados sem filtro de categoria, obteve %d", len(allResults))
	}
}

func TestFuseSearchResults_LevelL0(t *testing.T) {
	sources := []RankedResultSource{
		{
			Name: "fts",
			Results: []SearchResult{
				{ChunkID: "c1", DocumentID: "docA", Content: "Texto longo do chunk 1", Abstract: "Resumo A"},
				{ChunkID: "c2", DocumentID: "docA", Content: "Texto longo do chunk 2", Abstract: "Resumo A"},
				{ChunkID: "c3", DocumentID: "docB", Content: "Texto do chunk 3 sem abstract"},
			},
		},
		{
			Name: "vector",
			Results: []SearchResult{
				{ChunkID: "c1", DocumentID: "docA", Content: "Texto longo do chunk 1", Abstract: "Resumo A"},
			},
		},
	}

	// Nível L0: deve deduplicar chunks do mesmo docA e limpar Content
	l0Results := FuseSearchResultsWithOptions(sources, 60, 10, DecayOptions{}, SearchOptions{Level: "l0"})
	if len(l0Results) != 2 {
		t.Fatalf("Esperava 2 documentos únicos no modo L0 (docA e docB), obteve %d", len(l0Results))
	}

	for _, res := range l0Results {
		if res.Content != "" {
			t.Errorf("No nível L0, Content deve ser vazio para economizar contexto, obteve '%s'", res.Content)
		}
		if res.DocumentID == "docA" && res.Abstract != "Resumo A" {
			t.Errorf("Esperava Abstract 'Resumo A', obteve '%s'", res.Abstract)
		}
		if res.DocumentID == "docB" && res.Abstract == "" {
			t.Errorf("Esperava abstract extraído por fallback para docB, obteve vazio")
		}
	}
}

// TestFanInDeCodeEntraNoRRF prova que a fonte "code" participa do RRF como
// fonte real de fan-in (CG-19/T14): o item de codigo aparece na lista
// fundida com o score 1/(k+rank) e a origem "code:<rank>" em Sources.
func TestFanInDeCodeEntraNoRRF(t *testing.T) {
	k := DefaultRRFK

	noteResults := []SearchResult{
		{ChunkID: "doc1#0", DocumentID: "doc1", Content: "nota sobre a busca hibrida"},
	}
	codeResults := []SearchResult{
		{
			DocumentID: "code:pkg.Search",
			CodeSymbol: "pkg.Search",
			CodeFile:   "pkg/search.go",
			CodeLine:   42,
			Abstract:   "func Search(query string) error - pkg/search.go:42",
			Category:   "resource",
		},
		{
			DocumentID: "code:pkg.Fuse",
			CodeSymbol: "pkg.Fuse",
			CodeFile:   "pkg/fuse.go",
			CodeLine:   10,
			Abstract:   "func Fuse(a, b int) int - pkg/fuse.go:10",
			Category:   "resource",
		},
	}

	base := []RankedResultSource{
		{Name: "fts", Results: noteResults},
		{Name: "vector"},
		{Name: "graph"},
	}
	withCode := append(append([]RankedResultSource{}, base...), RankedResultSource{Name: CodeSourceName, Results: codeResults})

	fused := FuseSearchResultsWithOptions(withCode, k, 0, DefaultDecayOptions(), SearchOptions{Level: "l1"})

	var code1, code2 *SearchResult
	for i := range fused {
		switch fused[i].DocumentID {
		case "code:pkg.Search":
			code1 = &fused[i]
		case "code:pkg.Fuse":
			code2 = &fused[i]
		}
	}

	if code1 == nil || code2 == nil {
		t.Fatalf("esperava os dois simbolos de codigo na lista fundida, obteve %+v", fused)
	}

	want1 := 1.0 / float64(k+1)
	if math.Abs(code1.Score-want1) > 1e-9 {
		t.Errorf("score de code:pkg.Search: esperado %f (1/(k+1)), obteve %f", want1, code1.Score)
	}
	if len(code1.Sources) != 1 || code1.Sources[0] != "code:1" {
		t.Errorf("esperava Sources [code:1] em code:pkg.Search, obteve %v", code1.Sources)
	}

	want2 := 1.0 / float64(k+2)
	if math.Abs(code2.Score-want2) > 1e-9 {
		t.Errorf("score de code:pkg.Fuse: esperado %f (1/(k+2)), obteve %f", want2, code2.Score)
	}
	if len(code2.Sources) != 1 || code2.Sources[0] != "code:2" {
		t.Errorf("esperava Sources [code:2] em code:pkg.Fuse, obteve %v", code2.Sources)
	}

	// A nota mantem a contribuicao 1/(k+1) da fonte fts: a fonte "code" nao
	// altera o score das notas.
	var note *SearchResult
	for i := range fused {
		if fused[i].DocumentID == "doc1" {
			note = &fused[i]
		}
	}
	if note == nil {
		t.Fatalf("a nota doc1 sumiu da lista fundida: %+v", fused)
	}
	wantNote := 1.0 / float64(k+1)
	if math.Abs(note.Score-wantNote) > 1e-9 {
		t.Errorf("score da nota: esperado %f, obteve %f", wantNote, note.Score)
	}

	// Sem a fonte "code", a lista e exatamente a de antes (AC 21).
	before := FuseSearchResultsWithOptions(base, k, 0, DefaultDecayOptions(), SearchOptions{Level: "l1"})
	if len(before) != 1 || before[0].DocumentID != "doc1" {
		t.Fatalf("sem a fonte code a lista deve conter apenas a nota, obteve %+v", before)
	}
	if math.Abs(before[0].Score-wantNote) > 1e-9 {
		t.Errorf("sem a fonte code a nota deveria manter o score %f, obteve %f", wantNote, before[0].Score)
	}
}
