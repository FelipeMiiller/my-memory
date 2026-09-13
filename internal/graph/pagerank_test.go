package graph

import (
	"math"
	"testing"
)

func TestPageRank_EmptyAndSingleNode(t *testing.T) {
	opts := DefaultPageRankOptions()

	// Grafo vazio
	emptyScores := ComputePageRank(nil, nil, opts)
	if len(emptyScores) != 0 {
		t.Errorf("esperava mapa vazio para grafo sem nós, obteve %d elementos", len(emptyScores))
	}

	// Grafo com 1 nó
	singleScores := ComputePageRank([]string{"A"}, nil, opts)
	if len(singleScores) != 1 || singleScores["A"] != 1.0 {
		t.Errorf("esperava score 1.0 para único nó, obteve: %+v", singleScores)
	}
}

func TestPageRank_StarGraph(t *testing.T) {
	opts := DefaultPageRankOptions()

	// Grafo estrela: 4 folhas (L1, L2, L3, L4) apontam para o centro (C)
	nodes := []string{"C", "L1", "L2", "L3", "L4"}
	edges := []WeightedEdge{
		{Source: "L1", Target: "C", Type: "EXTRACTED"},
		{Source: "L2", Target: "C", Type: "EXTRACTED"},
		{Source: "L3", Target: "C", Type: "EXTRACTED"},
		{Source: "L4", Target: "C", Type: "EXTRACTED"},
	}

	scores := ComputePageRank(nodes, edges, opts)

	// Invariante 1: soma = 1.0
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("soma dos scores do PageRank deve ser 1.0, obteve: %f", sum)
	}

	// Invariante 2: Centro C deve ter score estritamente maior que qualquer folha
	for _, leaf := range []string{"L1", "L2", "L3", "L4"} {
		if scores["C"] <= scores[leaf] {
			t.Errorf("nó central C (%f) deveria ter score maior que folha %s (%f)", scores["C"], leaf, scores[leaf])
		}
	}

	// Invariante 3: Simetria das folhas
	if math.Abs(scores["L1"]-scores["L2"]) > 1e-6 || math.Abs(scores["L2"]-scores["L3"]) > 1e-6 {
		t.Errorf("folhas simétricas deveriam ter scores idênticos: %+v", scores)
	}
}

func TestPageRank_CycleGraph(t *testing.T) {
	opts := DefaultPageRankOptions()

	// Ciclo simétrico A -> B -> C -> A
	nodes := []string{"A", "B", "C"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
		{Source: "B", Target: "C", Type: "EXTRACTED"},
		{Source: "C", Target: "A", Type: "EXTRACTED"},
	}

	scores := ComputePageRank(nodes, edges, opts)

	expected := 1.0 / 3.0
	for _, n := range nodes {
		if math.Abs(scores[n]-expected) > 1e-5 {
			t.Errorf("nó %s no anel simétrico deveria ter score %f, obteve %f", n, expected, scores[n])
		}
	}
}

func TestPageRank_WeightedEdges(t *testing.T) {
	opts := DefaultPageRankOptions()

	// Nó A aponta para B com EXTRACTED (peso 1.0) e para C com INFERRED (peso 0.6)
	// B deve receber mais autoridade que C a partir de A
	nodes := []string{"A", "B", "C"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
		{Source: "A", Target: "C", Type: "INFERRED"},
	}

	scores := ComputePageRank(nodes, edges, opts)

	if scores["B"] <= scores["C"] {
		t.Errorf("score de B (%f com EXTRACTED) deveria ser maior que C (%f com INFERRED)", scores["B"], scores["C"])
	}

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("soma dos scores deve ser 1.0, obteve %f", sum)
	}
}

func TestPageRank_DanglingNodes(t *testing.T) {
	opts := DefaultPageRankOptions()

	// A -> B; B não tem arestas de saída (dangling node)
	nodes := []string{"A", "B"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
	}

	scores := ComputePageRank(nodes, edges, opts)

	// B recebe o link de A mais sua fatia da redistribuição, logo score(B) > score(A)
	if scores["B"] <= scores["A"] {
		t.Errorf("nó B (%f) apontado por A deveria ter score maior que A (%f)", scores["B"], scores["A"])
	}

	sum := scores["A"] + scores["B"]
	if math.Abs(sum-1.0) > 1e-5 {
		t.Errorf("soma deve ser 1.0, obteve %f", sum)
	}
}
