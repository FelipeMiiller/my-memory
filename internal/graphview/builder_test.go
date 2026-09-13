package graphview

import (
	"testing"
)

func TestInferNoteType(t *testing.T) {
	cases := []struct {
		id       string
		expected string
	}{
		{"decisions/001-auth.md", "decision"},
		{"docs/adr/019-versioning.md", "decision"},
		{"concepts/distributed-cache.md", "concept"},
		{"guides/deploy-kubernetes.md", "guide"},
		{"syntheses/auth-summary.md", "synthesis"},
		{"compiled/db-overview.md", "synthesis"},
		{"references/api-spec.md", "reference"},
		{"notes/random-thought.md", "other"},
	}

	for _, c := range cases {
		got := InferNoteType(c.id)
		if got != c.expected {
			t.Errorf("InferNoteType(%q) = %q; esperado %q", c.id, got, c.expected)
		}
	}
}

func TestGetColorForType(t *testing.T) {
	if GetColorForType("decision") != "#f43f5e" {
		t.Errorf("GetColorForType('decision') incorreto: %s", GetColorForType("decision"))
	}
	if GetColorForType("concept") != "#3b82f6" {
		t.Errorf("GetColorForType('concept') incorreto: %s", GetColorForType("concept"))
	}
	if GetColorForType("desconhecido") != "#64748b" {
		t.Errorf("GetColorForType('desconhecido') deve cair no default: %s", GetColorForType("desconhecido"))
	}
}

func TestBuildGraphView_Global(t *testing.T) {
	docs := []RawDoc{
		{ID: "concepts/auth.md", Title: "Autenticação Central"},
		{ID: "decisions/jwt.md", Title: "Adoção de JWT"},
		{ID: "guides/login.md", Title: "Como Implementar Login"},
		{ID: "notes/orphan.md", Title: "Nota Isolada"},
	}

	edges := []RawEdge{
		{Source: "decisions/jwt.md", Target: "concepts/auth.md", Relation: "implements", EpistemicStatus: "EXTRACTED", Weight: 1.0},
		{Source: "guides/login.md", Target: "decisions/jwt.md", Relation: "depends_on", EpistemicStatus: "EXTRACTED", Weight: 1.0},
		{Source: "guides/login.md", Target: "concepts/auth.md", Relation: "references", EpistemicStatus: "INFERRED", Weight: 0.6},
	}

	gv := BuildGraphView(docs, edges, "", 0, "test-repo")

	if gv.Stats.TotalNodes != 4 {
		t.Fatalf("esperado 4 nós, obteve %d", gv.Stats.TotalNodes)
	}
	if gv.Stats.TotalEdges != 3 {
		t.Fatalf("esperado 3 arestas, obteve %d", gv.Stats.TotalEdges)
	}
	if gv.Repository != "test-repo" {
		t.Errorf("esperado repositório 'test-repo', obteve '%s'", gv.Repository)
	}

	// Verifica se concepts/auth.md tem maior in-degree
	nodeMap := make(map[string]Node)
	for _, n := range gv.Nodes {
		nodeMap[n.ID] = n
	}

	authNode, exists := nodeMap["concepts/auth.md"]
	if !exists {
		t.Fatalf("nó concepts/auth.md não encontrado no GraphView")
	}
	if authNode.InDegree != 2 {
		t.Errorf("esperado InDegree 2 para concepts/auth.md, obteve %d", authNode.InDegree)
	}
	if authNode.Type != "concept" {
		t.Errorf("esperado Type 'concept', obteve '%s'", authNode.Type)
	}
	if authNode.Radius < 10.0 {
		t.Errorf("esperado raio maior que 10 para o nó mais referenciado, obteve %f", authNode.Radius)
	}
}

func TestBuildGraphView_FocusedSubgraph(t *testing.T) {
	docs := []RawDoc{
		{ID: "A", Title: "Nó Raiz A"},
		{ID: "B", Title: "Nó B"},
		{ID: "C", Title: "Nó C"},
		{ID: "D", Title: "Nó D Distante"},
		{ID: "E", Title: "Nó E Isolado"},
	}

	edges := []RawEdge{
		{Source: "A", Target: "B", Relation: "links_to", Weight: 1.0},
		{Source: "B", Target: "C", Relation: "links_to", Weight: 1.0},
		{Source: "C", Target: "D", Relation: "links_to", Weight: 1.0},
	}

	// Profundidade 1 a partir de A: deve incluir apenas A e B
	gvDepth1 := BuildGraphView(docs, edges, "A", 1, "")
	if gvDepth1.Stats.TotalNodes != 2 {
		t.Errorf("esperado 2 nós com profundidade 1 (A e B), obteve %d", gvDepth1.Stats.TotalNodes)
	}

	// Profundidade 2 a partir de A: deve incluir A, B e C (mas não D nem E)
	gvDepth2 := BuildGraphView(docs, edges, "A", 2, "")
	if gvDepth2.Stats.TotalNodes != 3 {
		t.Errorf("esperado 3 nós com profundidade 2 (A, B e C), obteve %d", gvDepth2.Stats.TotalNodes)
	}

	nodeMap := make(map[string]Node)
	for _, n := range gvDepth2.Nodes {
		nodeMap[n.ID] = n
	}
	if !nodeMap["A"].IsRoot {
		t.Errorf("esperado IsRoot=true para nó A")
	}
	if _, exists := nodeMap["D"]; exists {
		t.Errorf("nó D não deveria estar presente na profundidade 2")
	}
}
