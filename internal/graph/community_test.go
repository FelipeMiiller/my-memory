package graph

import (
	"math"
	"reflect"
	"testing"
)

func TestDetectCommunities_EmptyAndSingleNode(t *testing.T) {
	opts := DefaultCommunityOptions()

	// 1. Grafo vazio
	emptyRes := DetectCommunities(nil, nil, opts)
	if len(emptyRes.Communities) != 0 || emptyRes.Modularity != 0.0 || emptyRes.TotalNodes != 0 {
		t.Errorf("esperava resultado vazio para grafo sem nós, obteve: %+v", emptyRes)
	}

	// 2. Grafo com um único nó
	singleRes := DetectCommunities([]string{"node-1"}, nil, opts)
	if len(singleRes.Communities) != 1 {
		t.Fatalf("esperava 1 comunidade para grafo com 1 nó, obteve: %d", len(singleRes.Communities))
	}
	c := singleRes.Communities[0]
	if c.LeadNode != "node-1" || c.Size != 1 || len(c.Members) != 1 || c.Members[0] != "node-1" {
		t.Errorf("comunidade inesperada para 1 nó: %+v", c)
	}
	if singleRes.Modularity != 0.0 {
		t.Errorf("modularidade para 1 nó deveria ser 0.0, obteve: %f", singleRes.Modularity)
	}
}

func TestDetectCommunities_DisconnectedComponents(t *testing.T) {
	opts := DefaultCommunityOptions()

	// Dois triângulos totalmente desconectados: (A, B, C) e (D, E, F)
	nodes := []string{"A", "B", "C", "D", "E", "F"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
		{Source: "B", Target: "C", Type: "EXTRACTED"},
		{Source: "C", Target: "A", Type: "EXTRACTED"},

		{Source: "D", Target: "E", Type: "EXTRACTED"},
		{Source: "E", Target: "F", Type: "EXTRACTED"},
		{Source: "F", Target: "D", Type: "EXTRACTED"},
	}

	res := DetectCommunities(nodes, edges, opts)
	if len(res.Communities) != 2 {
		t.Fatalf("esperava exatamente 2 comunidades desconectadas, obteve %d", len(res.Communities))
	}

	for _, c := range res.Communities {
		if c.Size != 3 {
			t.Errorf("tamanho de cada comunidade deveria ser 3, obteve %d", c.Size)
		}
	}

	// Modularidade de dois triângulos idênticos desconectados deve ser positiva (0.5 teoricamente)
	if res.Modularity <= 0.0 {
		t.Errorf("modularidade esperada > 0, obteve: %f", res.Modularity)
	}
	if math.Abs(res.Modularity-0.5) > 1e-4 {
		t.Errorf("modularidade esperada ~0.5 para 2 triângulos simétricos, obteve: %f", res.Modularity)
	}
}

func TestDetectCommunities_TwoCliquesWithBridge(t *testing.T) {
	opts := DefaultCommunityOptions()

	// Dois cliques de 3 nós (A, B, C) e (D, E, F) unidos por uma ponte fraca (C - D) com TAG (0.3)
	nodes := []string{"A", "B", "C", "D", "E", "F"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
		{Source: "B", Target: "C", Type: "EXTRACTED"},
		{Source: "C", Target: "A", Type: "EXTRACTED"},

		{Source: "D", Target: "E", Type: "EXTRACTED"},
		{Source: "E", Target: "F", Type: "EXTRACTED"},
		{Source: "F", Target: "D", Type: "EXTRACTED"},

		// Ponte
		{Source: "C", Target: "D", Type: "TAG"},
	}

	res := DetectCommunities(nodes, edges, opts)
	if len(res.Communities) != 2 {
		t.Fatalf("esperava 2 comunidades mesmo com ponte entre cliques, obteve %d: %+v", len(res.Communities), res.Communities)
	}

	if res.Modularity <= 0.3 {
		t.Errorf("modularidade esperada alta (>0.3) para cliques particionados, obteve: %f", res.Modularity)
	}
}

func TestDetectCommunities_IsolatedNodesAndMinClusterSize(t *testing.T) {
	nodes := []string{"A", "B", "C", "Isolated1", "Isolated2"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Type: "EXTRACTED"},
		{Source: "B", Target: "C", Type: "EXTRACTED"},
		{Source: "C", Target: "A", Type: "EXTRACTED"},
	}

	// 1. MinClusterSize = 1 (inclui nós isolados)
	opts1 := CommunityOptions{MinClusterSize: 1}
	res1 := DetectCommunities(nodes, edges, opts1)
	if len(res1.Communities) != 3 { // {A,B,C}, {Isolated1}, {Isolated2}
		t.Fatalf("esperava 3 comunidades com MinClusterSize=1, obteve %d", len(res1.Communities))
	}

	// 2. MinClusterSize = 2 (oculta nós isolados de tamanho 1)
	opts2 := CommunityOptions{MinClusterSize: 2}
	res2 := DetectCommunities(nodes, edges, opts2)
	if len(res2.Communities) != 1 {
		t.Fatalf("esperava 1 comunidade com MinClusterSize=2, obteve %d", len(res2.Communities))
	}
	if res2.Communities[0].Size != 3 {
		t.Errorf("comunidade restante deve ter tamanho 3, obteve %d", res2.Communities[0].Size)
	}
	// A modularidade global deve permanecer inalterada pois reflete todo o grafo
	if math.Abs(res1.Modularity-res2.Modularity) > 1e-9 {
		t.Errorf("modularidade global não deve mudar com filtro de visualização: %f vs %f", res1.Modularity, res2.Modularity)
	}
}

func TestDetectCommunities_Determinism(t *testing.T) {
	opts := DefaultCommunityOptions()

	nodes := []string{"N1", "N2", "N3", "N4", "N5", "N6", "N7", "N8"}
	edges := []WeightedEdge{
		{Source: "N1", Target: "N2", Type: "EXTRACTED"},
		{Source: "N2", Target: "N3", Type: "EXTRACTED"},
		{Source: "N3", Target: "N1", Type: "EXTRACTED"},
		{Source: "N4", Target: "N5", Type: "INFERRED"},
		{Source: "N5", Target: "N6", Type: "INFERRED"},
		{Source: "N6", Target: "N4", Type: "INFERRED"},
		{Source: "N7", Target: "N8", Type: "TAG"},
	}

	firstRes := DetectCommunities(nodes, edges, opts)

	for i := 0; i < 10; i++ {
		repeatRes := DetectCommunities(nodes, edges, opts)
		if len(firstRes.Communities) != len(repeatRes.Communities) {
			t.Fatalf("execução %d divergiu no número de comunidades", i)
		}
		if math.Abs(firstRes.Modularity-repeatRes.Modularity) > 1e-9 {
			t.Fatalf("execução %d divergiu na modularidade: %f != %f", i, firstRes.Modularity, repeatRes.Modularity)
		}
		for idx := range firstRes.Communities {
			c1 := firstRes.Communities[idx]
			c2 := repeatRes.Communities[idx]
			if c1.ID != c2.ID || c1.LeadNode != c2.LeadNode || !reflect.DeepEqual(c1.Members, c2.Members) {
				t.Fatalf("execução %d divergiu na comunidade %d: %+v vs %+v", i, idx, c1, c2)
			}
		}
	}
}

func TestDetectCommunities_DominantType(t *testing.T) {
	nodes := []string{"auth-1", "auth-2", "auth-doc", "db-1", "db-2"}
	edges := []WeightedEdge{
		{Source: "auth-1", Target: "auth-2", Type: "EXTRACTED"},
		{Source: "auth-2", Target: "auth-doc", Type: "EXTRACTED"},
		{Source: "auth-doc", Target: "auth-1", Type: "EXTRACTED"},

		{Source: "db-1", Target: "db-2", Type: "EXTRACTED"},
	}

	nodeTypes := map[string]string{
		"auth-1":   "concept",
		"auth-2":   "concept",
		"auth-doc": "documentation",
		"db-1":     "decision",
		"db-2":     "decision",
	}

	opts := CommunityOptions{
		NodeTypes: nodeTypes,
	}

	res := DetectCommunities(nodes, edges, opts)
	if len(res.Communities) != 2 {
		t.Fatalf("esperava 2 comunidades, obteve %d", len(res.Communities))
	}

	for _, c := range res.Communities {
		if c.Size == 3 && c.DominantType != "concept" {
			t.Errorf("comunidade de auth esperava dominantType 'concept', obteve '%s'", c.DominantType)
		}
		if c.Size == 2 && c.DominantType != "decision" {
			t.Errorf("comunidade de db esperava dominantType 'decision', obteve '%s'", c.DominantType)
		}
	}
}

func TestDetectCommunities_EpistemicEdgeWeights(t *testing.T) {
	// Nó X tem conexões com dois grupos diferentes:
	// Conexão forte (EXTRACTED peso 1.0) com Grupo A (A1, A2)
	// Conexão fraca (TAG peso 0.3) com Grupo B (B1, B2)
	// X deve migrar para o Grupo A por causa do peso epistêmico maior
	nodes := []string{"A1", "A2", "X", "B1", "B2"}
	edges := []WeightedEdge{
		{Source: "A1", Target: "A2", Type: "EXTRACTED"},
		{Source: "A1", Target: "X", Type: "EXTRACTED"}, // peso 1.0

		{Source: "B1", Target: "B2", Type: "EXTRACTED"},
		{Source: "B1", Target: "X", Type: "TAG"}, // peso 0.3
	}

	opts := DefaultCommunityOptions()
	res := DetectCommunities(nodes, edges, opts)

	// Procurar em qual comunidade X foi alocado
	var xCommunityID int
	var a1CommunityID int
	for _, c := range res.Communities {
		for _, m := range c.Members {
			if m == "X" {
				xCommunityID = c.ID
			}
			if m == "A1" {
				a1CommunityID = c.ID
			}
		}
	}

	if xCommunityID != a1CommunityID {
		t.Errorf("esperava que nó X ficasse no cluster de A1 devido ao peso EXTRACTED (1.0 > 0.3), mas ficou em clusters diferentes: %d vs %d", xCommunityID, a1CommunityID)
	}
}
