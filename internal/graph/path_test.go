package graph

import (
	"strings"
	"testing"
)

func TestFindPath_IdenticalSourceAndTarget(t *testing.T) {
	nodes := []string{"auth.md", "session.md"}
	edges := []WeightedEdge{
		{Source: "auth.md", Target: "session.md", Weight: 1.0, Type: "EXTRACTED"},
	}

	res, err := FindPath(nodes, edges, "auth.md", "auth.md", DefaultPathOptions())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !res.Found {
		t.Errorf("deveria ter encontrado o caminho")
	}
	if res.Hops != 0 {
		t.Errorf("esperado 0 saltos, obteve %d", res.Hops)
	}
	if res.TotalCost != 0.0 {
		t.Errorf("esperado custo 0.0, obteve %f", res.TotalCost)
	}
	if len(res.Nodes) != 1 || res.Nodes[0] != "auth.md" {
		t.Errorf("nós inesperados: %v", res.Nodes)
	}
	if len(res.Edges) != 0 {
		t.Errorf("arestas deveriam estar vazias: %v", res.Edges)
	}
}

func TestFindPath_NodeNotFound(t *testing.T) {
	nodes := []string{"auth.md"}
	edges := []WeightedEdge{}

	_, err1 := FindPath(nodes, edges, "inexistente.md", "auth.md", DefaultPathOptions())
	if err1 == nil || !strings.Contains(err1.Error(), "origem 'inexistente.md' não encontrado") {
		t.Errorf("deveria falhar com nó de origem inexistente: %v", err1)
	}

	_, err2 := FindPath(nodes, edges, "auth.md", "inexistente.md", DefaultPathOptions())
	if err2 == nil || !strings.Contains(err2.Error(), "destino 'inexistente.md' não encontrado") {
		t.Errorf("deveria falhar com nó de destino inexistente: %v", err2)
	}

	_, err3 := FindPath(nodes, edges, "", "auth.md", DefaultPathOptions())
	if err3 == nil {
		t.Errorf("deveria falhar com nó vazio")
	}
}

func TestFindPath_Unreachable(t *testing.T) {
	nodes := []string{"A", "B", "C", "D"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "C", Target: "D", Weight: 1.0, Type: "EXTRACTED"},
	}

	res, err := FindPath(nodes, edges, "A", "D", DefaultPathOptions())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if res.Found {
		t.Errorf("caminho não deveria ter sido encontrado entre componentes desconexos")
	}
	if len(res.Nodes) != 0 || len(res.Edges) != 0 {
		t.Errorf("nodes e edges deveriam ser vazios")
	}
}

func TestFindPath_LinearChain(t *testing.T) {
	nodes := []string{"A", "B", "C"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "B", Target: "C", Weight: 1.0, Type: "EXTRACTED"},
	}

	res, err := FindPath(nodes, edges, "A", "C", DefaultPathOptions())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !res.Found {
		t.Fatalf("caminho deveria ter sido encontrado")
	}
	if res.Hops != 2 {
		t.Errorf("esperado 2 saltos, obteve %d", res.Hops)
	}
	if res.TotalCost != 2.0 {
		t.Errorf("esperado custo 2.0, obteve %f", res.TotalCost)
	}
	expectedNodes := []string{"A", "B", "C"}
	for i, n := range expectedNodes {
		if res.Nodes[i] != n {
			t.Errorf("nó[%d] esperado %s, obteve %s", i, n, res.Nodes[i])
		}
	}
	if len(res.Edges) != 2 {
		t.Fatalf("esperado 2 arestas, obteve %d", len(res.Edges))
	}
	if res.Edges[0].From != "A" || res.Edges[0].To != "B" || res.Edges[0].Direction != "forward" {
		t.Errorf("aresta 0 inválida: %+v", res.Edges[0])
	}
	if res.Edges[1].From != "B" || res.Edges[1].To != "C" || res.Edges[1].Direction != "forward" {
		t.Errorf("aresta 1 inválida: %+v", res.Edges[1])
	}
}

func TestFindPath_EpistemicCostTieBreaker(t *testing.T) {
	// Caminho 1: A -> B -> D com arestas EXTRACTED (peso 1.0 cada => custo 1.0 + 1.0 = 2.0, hops = 2)
	// Caminho 2: A -> D direto com aresta TAG (peso 0.3 => custo 1.0/0.3 = 3.333, hops = 1)
	nodes := []string{"A", "B", "D"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "B", Target: "D", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "A", Target: "D", Weight: 0.3, Type: "TAG"},
	}

	// 1. Modo Epistêmico: deve preferir o caminho de 2 saltos (custo 2.0 < 3.333)
	optsEpistemic := PathOptions{
		MaxDepth: 6,
		Directed: true,
		CostMode: CostModeEpistemic,
	}
	resEpistemic, err := FindPath(nodes, edges, "A", "D", optsEpistemic)
	if err != nil {
		t.Fatalf("erro no modo epistêmico: %v", err)
	}
	if !resEpistemic.Found {
		t.Fatalf("deveria ter encontrado o caminho epistêmico")
	}
	if resEpistemic.Hops != 2 {
		t.Errorf("modo epistêmico deveria escolher caminho de 2 saltos sólidos, obteve %d", resEpistemic.Hops)
	}
	if resEpistemic.TotalCost != 2.0 {
		t.Errorf("esperado custo 2.0, obteve %f", resEpistemic.TotalCost)
	}
	if len(resEpistemic.Nodes) != 3 || resEpistemic.Nodes[1] != "B" {
		t.Errorf("esperado caminho [A, B, D], obteve %v", resEpistemic.Nodes)
	}

	// 2. Modo Hops: deve preferir o caminho direto de 1 salto (1 salto < 2 saltos)
	optsHops := PathOptions{
		MaxDepth: 6,
		Directed: true,
		CostMode: CostModeHops,
	}
	resHops, err := FindPath(nodes, edges, "A", "D", optsHops)
	if err != nil {
		t.Fatalf("erro no modo hops: %v", err)
	}
	if !resHops.Found {
		t.Fatalf("deveria ter encontrado o caminho hops")
	}
	if resHops.Hops != 1 {
		t.Errorf("modo hops deveria escolher caminho direto de 1 salto, obteve %d", resHops.Hops)
	}
	if resHops.TotalCost != 1.0 {
		t.Errorf("esperado custo 1.0 no modo hops, obteve %f", resHops.TotalCost)
	}
	if len(resHops.Nodes) != 2 || resHops.Nodes[1] != "D" {
		t.Errorf("esperado caminho direto [A, D], obteve %v", resHops.Nodes)
	}
}

func TestFindPath_DirectedVsUndirected(t *testing.T) {
	// A aponta para B, e C aponta para B: A -> B <- C
	nodes := []string{"A", "B", "C"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "C", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
	}

	// Modo direcionado de A para C: não deve encontrar pois C aponta para B (não B para C)
	optsDirected := PathOptions{MaxDepth: 6, Directed: true, CostMode: CostModeEpistemic}
	resDir, err := FindPath(nodes, edges, "A", "C", optsDirected)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if resDir.Found {
		t.Errorf("não deveria encontrar caminho direcionado de A para C")
	}

	// Modo não-direcionado de A para C: deve encontrar A -> B (forward) -> C (reverse)
	optsUndirected := PathOptions{MaxDepth: 6, Directed: false, CostMode: CostModeEpistemic}
	resUndir, err := FindPath(nodes, edges, "A", "C", optsUndirected)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if !resUndir.Found {
		t.Fatalf("deveria encontrar caminho não-direcionado de A para C")
	}
	if resUndir.Hops != 2 {
		t.Errorf("esperado 2 saltos, obteve %d", resUndir.Hops)
	}
	if len(resUndir.Edges) != 2 {
		t.Fatalf("esperado 2 arestas, obteve %d", len(resUndir.Edges))
	}
	if resUndir.Edges[0].Direction != "forward" {
		t.Errorf("primeira aresta deveria ser forward")
	}
	if resUndir.Edges[1].Direction != "reverse" {
		t.Errorf("segunda aresta deveria ser reverse")
	}
}

func TestFindPath_MaxDepthCutoff(t *testing.T) {
	// Cadeia de 4 saltos: A -> B -> C -> D -> E
	nodes := []string{"A", "B", "C", "D", "E"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "B", Target: "C", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "C", Target: "D", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "D", Target: "E", Weight: 1.0, Type: "EXTRACTED"},
	}

	// Com max_depth = 3, não deve encontrar
	optsDepth3 := PathOptions{MaxDepth: 3, Directed: true, CostMode: CostModeHops}
	res3, err := FindPath(nodes, edges, "A", "E", optsDepth3)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if res3.Found {
		t.Errorf("com limite de 3 saltos não deveria alcançar 4 saltos")
	}

	// Com max_depth = 4, deve encontrar
	optsDepth4 := PathOptions{MaxDepth: 4, Directed: true, CostMode: CostModeHops}
	res4, err := FindPath(nodes, edges, "A", "E", optsDepth4)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if !res4.Found {
		t.Fatalf("com limite de 4 saltos deveria encontrar o destino")
	}
	if res4.Hops != 4 {
		t.Errorf("esperado 4 saltos, obteve %d", res4.Hops)
	}
}

func TestFindPath_CyclicGraph(t *testing.T) {
	// Grafo com ciclo: A -> B -> C -> A, e C -> D
	nodes := []string{"A", "B", "C", "D"}
	edges := []WeightedEdge{
		{Source: "A", Target: "B", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "B", Target: "C", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "C", Target: "A", Weight: 1.0, Type: "EXTRACTED"},
		{Source: "C", Target: "D", Weight: 1.0, Type: "EXTRACTED"},
	}

	res, err := FindPath(nodes, edges, "A", "D", DefaultPathOptions())
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if !res.Found {
		t.Fatalf("deveria encontrar caminho até D em grafo com ciclo")
	}
	if res.Hops != 3 {
		t.Errorf("esperado 3 saltos (A -> B -> C -> D), obteve %d", res.Hops)
	}
	expected := []string{"A", "B", "C", "D"}
	for i, n := range expected {
		if res.Nodes[i] != n {
			t.Errorf("nó[%d] esperado %s, obteve %s", i, n, res.Nodes[i])
		}
	}
}
