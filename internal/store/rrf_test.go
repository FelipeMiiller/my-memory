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
