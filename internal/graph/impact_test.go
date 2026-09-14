package graph

import (
	"testing"
)

func TestCalculateImpact_NodeNotFound(t *testing.T) {
	nodes := []string{"A", "B"}
	edges := []WeightedEdge{{Source: "A", Target: "B", Weight: 1.0, Type: "links_to"}}

	_, err := CalculateImpact(nodes, edges, "NonExistent", DefaultImpactOptions())
	if err == nil {
		t.Fatal("esperava erro ao consultar nó inexistente, obteve nil")
	}

	_, errEmpty := CalculateImpact(nodes, edges, "", DefaultImpactOptions())
	if errEmpty == nil {
		t.Fatal("esperava erro para nó com ID vazio, obteve nil")
	}
}

func TestCalculateImpact_IsolatedNode(t *testing.T) {
	nodes := []string{"Isolated", "Other"}
	edges := []WeightedEdge{{Source: "Other", Target: "Other", Weight: 1.0, Type: "links_to"}}

	res, err := CalculateImpact(nodes, edges, "Isolated", DefaultImpactOptions())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if res.TotalImpacted != 0 {
		t.Errorf("nó isolado deveria ter 0 impactados, obteve %d", res.TotalImpacted)
	}
	if res.RiskScore != 0.0 {
		t.Errorf("nó isolado deveria ter RiskScore 0.0, obteve %f", res.RiskScore)
	}
	if res.RiskLevel != "BAIXO" {
		t.Errorf("esperava nível BAIXO, obteve %s", res.RiskLevel)
	}
}

func TestCalculateImpact_LinearChainAndDepth(t *testing.T) {
	// A -> B -> C -> Target
	nodes := []string{"A", "B", "C", "Target"}
	edges := []WeightedEdge{
		{Source: "C", Target: "Target", Weight: 1.0, Type: "implements"},
		{Source: "B", Target: "C", Weight: 1.0, Type: "depends_on"},
		{Source: "A", Target: "B", Weight: 1.0, Type: "links_to"},
	}

	// 1. MaxDepth = 1 (apenas dependentes diretos)
	opts1 := ImpactOptions{MaxDepth: 1}
	res1, err := CalculateImpact(nodes, edges, "Target", opts1)
	if err != nil {
		t.Fatalf("falha ao calcular impacto: %v", err)
	}

	if res1.TotalImpacted != 1 {
		t.Fatalf("com max_depth=1 esperava 1 impactado (C), obteve %d", res1.TotalImpacted)
	}
	if res1.Nodes[0].ID != "C" || res1.Nodes[0].Severity != SeverityCritical {
		t.Errorf("esperava C com severidade CRITICAL, obteve %+v", res1.Nodes[0])
	}
	if res1.DirectDependents != 1 || res1.IndirectDependents != 0 {
		t.Errorf("esperava 1 direto e 0 indireto, obteve direct=%d indirect=%d", res1.DirectDependents, res1.IndirectDependents)
	}

	// 2. MaxDepth = 2 (C e B)
	opts2 := ImpactOptions{MaxDepth: 2}
	res2, err := CalculateImpact(nodes, edges, "Target", opts2)
	if err != nil {
		t.Fatalf("falha com max_depth=2: %v", err)
	}

	if res2.TotalImpacted != 2 {
		t.Fatalf("com max_depth=2 esperava 2 impactados (C, B), obteve %d", res2.TotalImpacted)
	}
	if res2.DirectDependents != 1 || res2.IndirectDependents != 1 {
		t.Errorf("esperava 1 direto e 1 indireto, obteve direct=%d indirect=%d", res2.DirectDependents, res2.IndirectDependents)
	}
	// C (depth 1, implements) -> CRITICAL
	// B (depth 2, depends_on) -> HIGH
	if res2.Nodes[1].ID != "B" || res2.Nodes[1].Severity != SeverityHigh {
		t.Errorf("esperava B com severidade HIGH, obteve %+v", res2.Nodes[1])
	}

	// 3. MaxDepth = 3 (C, B e A)
	opts3 := ImpactOptions{MaxDepth: 3}
	res3, err := CalculateImpact(nodes, edges, "Target", opts3)
	if err != nil {
		t.Fatalf("falha com max_depth=3: %v", err)
	}
	if res3.TotalImpacted != 3 {
		t.Fatalf("com max_depth=3 esperava 3 impactados (C, B, A), obteve %d", res3.TotalImpacted)
	}
	// A (depth 3, links_to) -> LOW
	if res3.Nodes[2].ID != "A" || res3.Nodes[2].Severity != SeverityLow {
		t.Errorf("esperava A com severidade LOW, obteve %+v", res3.Nodes[2])
	}
	if res3.RiskScore <= res2.RiskScore {
		t.Errorf("RiskScore com 3 nós (%f) deveria ser maior que com 2 nós (%f)", res3.RiskScore, res2.RiskScore)
	}
}

func TestCalculateImpact_DiamondAndCycles(t *testing.T) {
	// Diamante com ciclo:
	// A -> Target
	// B -> Target
	// C -> A
	// C -> B
	// Target -> A (ciclo reverso, não deve causar loop infinito)
	nodes := []string{"Target", "A", "B", "C"}
	edges := []WeightedEdge{
		{Source: "A", Target: "Target", Weight: 1.0, Type: "implements"},
		{Source: "B", Target: "Target", Weight: 1.0, Type: "depends_on"},
		{Source: "C", Target: "A", Weight: 1.0, Type: "links_to"},
		{Source: "C", Target: "B", Weight: 1.0, Type: "links_to"},
		{Source: "Target", Target: "A", Weight: 1.0, Type: "links_to"}, // Ciclo
	}

	opts := ImpactOptions{
		MaxDepth:    3,
		Communities: map[string]int{"A": 1, "B": 2, "C": 1},
		PageRanks:   map[string]float64{"A": 0.25, "B": 0.15, "C": 0.05},
		NodeTypes:   map[string]string{"A": "decision", "B": "concept", "C": "note"},
	}

	res, err := CalculateImpact(nodes, edges, "Target", opts)
	if err != nil {
		t.Fatalf("falha no cálculo com ciclos e diamante: %v", err)
	}

	if res.TotalImpacted != 3 {
		t.Fatalf("esperava 3 nós únicos impactados (A, B, C), obteve %d", res.TotalImpacted)
	}

	// Verifica se C foi computado uma única vez
	seen := make(map[string]bool)
	for _, n := range res.Nodes {
		if seen[n.ID] {
			t.Errorf("nó duplicado detectado: %s", n.ID)
		}
		seen[n.ID] = true
	}

	// Verifica clusters afetados
	if len(res.AffectedClusters) != 2 || res.AffectedClusters[0] != 1 || res.AffectedClusters[1] != 2 {
		t.Errorf("clusters afetados esperados [1, 2], obteve %v", res.AffectedClusters)
	}

	if res.RiskScore <= 0 || res.RiskScore > 100 {
		t.Errorf("RiskScore fora dos limites: %f", res.RiskScore)
	}
}
