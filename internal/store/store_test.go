package store

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1.0, 2.0, 3.0}
	b := []float32{1.0, 2.0, 3.0}
	sim := CosineSimilarity(a, b)
	if math.Abs(sim-1.0) > 1e-5 {
		t.Fatalf("Esperava similaridade 1.0, obteve %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float32{1.0, 0.0}
	b := []float32{0.0, 1.0}
	sim := CosineSimilarity(a, b)
	if math.Abs(sim-0.0) > 1e-5 {
		t.Fatalf("Esperava similaridade 0.0, obteve %f", sim)
	}
}

func TestCosineSimilarity_Opposite(t *testing.T) {
	a := []float32{1.0, 2.0}
	b := []float32{-1.0, -2.0}
	sim := CosineSimilarity(a, b)
	if math.Abs(sim-(-1.0)) > 1e-5 {
		t.Fatalf("Esperava similaridade -1.0, obteve %f", sim)
	}
}

func TestCosineSimilarity_ZeroOrEmpty(t *testing.T) {
	if CosineSimilarity(nil, nil) != 0 {
		t.Errorf("Esperava 0 para nil")
	}
	if CosineSimilarity([]float32{1.0}, []float32{1.0, 2.0}) != 0 {
		t.Errorf("Esperava 0 para tamanhos diferentes")
	}
	if CosineSimilarity([]float32{0.0, 0.0}, []float32{1.0, 1.0}) != 0 {
		t.Errorf("Esperava 0 para vetor de magnitude zero")
	}
}

func TestCalculateJaccardSimilarity_Identical(t *testing.T) {
	text := "arquitetura distribuída resiliente com postgresql e sqlite"
	sim := CalculateJaccardSimilarity(text, text)
	if math.Abs(sim-1.0) > 1e-5 {
		t.Fatalf("Esperava 1.0 para textos idênticos, obteve %f", sim)
	}
}

func TestCalculateJaccardSimilarity_Disjoint(t *testing.T) {
	textA := "arquitetura distribuída microsserviços"
	textB := "culinária italiana massas receitas"
	sim := CalculateJaccardSimilarity(textA, textB)
	if sim != 0 {
		t.Fatalf("Esperava 0.0 para textos disjuntos, obteve %f", sim)
	}
}

func TestCalculateJaccardSimilarity_PartialOverlap(t *testing.T) {
	// A: {banco, dados, postgresql} (3 palavras)
	// B: {banco, dados, sqlite} (3 palavras)
	// Interseção: {banco, dados} = 2
	// União: {banco, dados, postgresql, sqlite} = 4
	// Jaccard = 2/4 = 0.5
	textA := "banco de dados postgresql"
	textB := "banco de dados sqlite"
	sim := CalculateJaccardSimilarity(textA, textB)
	if math.Abs(sim-0.5) > 1e-5 {
		t.Fatalf("Esperava 0.5 para Jaccard, obteve %f", sim)
	}
}

func TestCalculateJaccardSimilarity_Empty(t *testing.T) {
	if CalculateJaccardSimilarity("", "teste") != 0 {
		t.Errorf("Esperava 0 para texto vazio")
	}
	if CalculateJaccardSimilarity("a b c", "d e f") != 0 { // palavras curtas < 3 chars ignoradas
		t.Errorf("Esperava 0 para palavras curtas ignoradas")
	}
}

func TestSurprisingConnection_JSON(t *testing.T) {
	conn := SurprisingConnection{
		SourceID:   "doc-a",
		SourceName: "Arquitetura",
		TargetID:   "doc-b",
		TargetName: "Design Patterns",
		Similarity: 0.88,
		Reason:     "Alta proximidade semântica (88%) sem conexão direta no grafo",
	}

	bytes, err := json.Marshal(conn)
	if err != nil {
		t.Fatalf("Erro ao serializar SurprisingConnection: %v", err)
	}

	var decoded SurprisingConnection
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("Erro ao deserializar SurprisingConnection: %v", err)
	}

	if decoded.SourceID != "doc-a" || decoded.TargetID != "doc-b" || decoded.Similarity != 0.88 {
		t.Fatalf("Dados decodificados incorretos: %+v", decoded)
	}
}
