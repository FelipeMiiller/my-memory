package graphview

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
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
	cases := []struct {
		noteType string
		expected string
	}{
		{"decision", "#f43f5e"},
		{"adr", "#f43f5e"},
		{"concept", "#3b82f6"},
		{"guide", "#10b981"},
		{"howto", "#10b981"},
		{"tutorial", "#10b981"},
		{"reference", "#a855f7"},
		{"doc", "#a855f7"},
		{"docs", "#a855f7"},
		{"synthesis", "#ec4899"},
		{"compiled", "#ec4899"},
		{"hub", "#eab308"},
		{"god_node", "#eab308"},
		{"other", "#64748b"},
		{"unknown", "#64748b"},
	}

	for _, c := range cases {
		got := GetColorForType(c.noteType)
		if got != c.expected {
			t.Errorf("GetColorForType(%q) = %s; esperado %s", c.noteType, got, c.expected)
		}
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

func TestBuildGraphView_HubAndDensity(t *testing.T) {
	// Cria um hub com 12 referências apontando para ele (grau > 10)
	hubID := "hub-node.md"
	docs := []RawDoc{{ID: hubID, Title: "Hub Central"}}
	var edges []RawEdge

	for i := 0; i < 12; i++ {
		leafID := fmt.Sprintf("leaf-%d.md", i)
		docs = append(docs, RawDoc{ID: leafID, Title: leafID})
		edges = append(edges, RawEdge{
			Source:          leafID,
			Target:          hubID,
			Relation:        "depends_on",
			EpistemicStatus: "EXTRACTED",
			Weight:          1.0,
		})
	}

	gv := BuildGraphView(docs, edges, "", 0, "test-hub")
	if gv.Stats.HubCount < 1 {
		t.Errorf("esperado ao menos 1 hub, obteve %d", gv.Stats.HubCount)
	}

	for _, n := range gv.Nodes {
		if n.ID == hubID && !n.IsHub {
			t.Errorf("nó %s deveria estar marcado como IsHub=true", hubID)
		}
	}

	// Teste com grafo vazio (0 nós)
	emptyGV := BuildGraphView(nil, nil, "", 0, "")
	if emptyGV.Stats.TotalNodes != 0 || emptyGV.Stats.Density != 0.0 {
		t.Errorf("grafo vazio deve ter 0 nós e densidade 0")
	}

	// Teste com apenas 1 nó isolado
	singleGV := BuildGraphView([]RawDoc{{ID: "solo.md"}}, nil, "", 0, "")
	if singleGV.Stats.TotalNodes != 1 || singleGV.Stats.Density != 0.0 {
		t.Errorf("grafo com 1 nó deve ter densidade 0.0")
	}
}

func TestBuildFromSQLite_Success(t *testing.T) {
	db, err := setupMockDB()
	if err != nil {
		t.Fatalf("setupMockDB falhou: %v", err)
	}
	defer db.Close()

	mockQueryMu.Lock()
	mockQueryFn = func(query string) (driver.Rows, error) {
		if strings.Contains(query, "FROM documents") {
			return &mockRows{
				columns: []string{"id", "title", "updated_at"},
				rows: [][]driver.Value{
					{"doc1.md", "Documento 1", int64(1000)},
					{"doc2.md", "Documento 2", int64(2000)},
				},
			}, nil
		}
		if strings.Contains(query, "FROM graph_edges") {
			return &mockRows{
				columns: []string{"source_id", "target_id", "relation", "epistemic_status", "weight"},
				rows: [][]driver.Value{
					{"doc1.md", "doc2.md", "links_to", "EXTRACTED", 1.0},
				},
			}, nil
		}
		return nil, fmt.Errorf("query desconhecida: %s", query)
	}
	mockQueryMu.Unlock()

	gv, err := BuildFromSQLite(context.Background(), db, "", 0, "mock-repo")
	if err != nil {
		t.Fatalf("BuildFromSQLite falhou: %v", err)
	}

	if gv.Stats.TotalNodes != 2 {
		t.Errorf("esperado 2 nós, obteve %d", gv.Stats.TotalNodes)
	}
	if gv.Stats.TotalEdges != 1 {
		t.Errorf("esperado 1 aresta, obteve %d", gv.Stats.TotalEdges)
	}
}

func TestBuildFromSQLite_DocQueryError(t *testing.T) {
	db, err := setupMockDB()
	if err != nil {
		t.Fatalf("setupMockDB falhou: %v", err)
	}
	defer db.Close()

	mockQueryMu.Lock()
	mockQueryFn = func(query string) (driver.Rows, error) {
		if strings.Contains(query, "FROM documents") {
			return nil, errors.New("falha na tabela documents")
		}
		return nil, nil
	}
	mockQueryMu.Unlock()

	_, err = BuildFromSQLite(context.Background(), db, "", 0, "mock-repo")
	if err == nil || !strings.Contains(err.Error(), "documents") {
		t.Fatalf("esperava erro mencionando 'documents', obteve: %v", err)
	}
}

func TestBuildFromSQLite_DocScanError(t *testing.T) {
	db, err := setupMockDB()
	if err != nil {
		t.Fatalf("setupMockDB falhou: %v", err)
	}
	defer db.Close()

	mockQueryMu.Lock()
	mockQueryFn = func(query string) (driver.Rows, error) {
		if strings.Contains(query, "FROM documents") {
			// Retorna apenas 1 coluna, causando erro no Scan (espera 3 colunas)
			return &mockRows{
				columns: []string{"id"},
				rows: [][]driver.Value{
					{"doc1.md"},
				},
			}, nil
		}
		return nil, nil
	}
	mockQueryMu.Unlock()

	_, err = BuildFromSQLite(context.Background(), db, "", 0, "mock-repo")
	if err == nil {
		t.Fatal("esperava erro de scan em documents, obteve nil")
	}
}

func TestBuildFromSQLite_EdgeQueryError(t *testing.T) {
	db, err := setupMockDB()
	if err != nil {
		t.Fatalf("setupMockDB falhou: %v", err)
	}
	defer db.Close()

	mockQueryMu.Lock()
	mockQueryFn = func(query string) (driver.Rows, error) {
		if strings.Contains(query, "FROM documents") {
			return &mockRows{
				columns: []string{"id", "title", "updated_at"},
				rows: [][]driver.Value{
					{"doc1.md", "Doc 1", int64(1000)},
				},
			}, nil
		}
		if strings.Contains(query, "FROM graph_edges") {
			return nil, errors.New("falha ao consultar graph_edges")
		}
		return nil, nil
	}
	mockQueryMu.Unlock()

	_, err = BuildFromSQLite(context.Background(), db, "", 0, "mock-repo")
	if err == nil || !strings.Contains(err.Error(), "graph_edges") {
		t.Fatalf("esperava erro mencionando 'graph_edges', obteve: %v", err)
	}
}

func TestBuildFromSQLite_EdgeScanError(t *testing.T) {
	db, err := setupMockDB()
	if err != nil {
		t.Fatalf("setupMockDB falhou: %v", err)
	}
	defer db.Close()

	mockQueryMu.Lock()
	mockQueryFn = func(query string) (driver.Rows, error) {
		if strings.Contains(query, "FROM documents") {
			return &mockRows{
				columns: []string{"id", "title", "updated_at"},
				rows: [][]driver.Value{
					{"doc1.md", "Doc 1", int64(1000)},
				},
			}, nil
		}
		if strings.Contains(query, "FROM graph_edges") {
			// Retorna apenas 1 coluna, causando erro no Scan (espera 5)
			return &mockRows{
				columns: []string{"source_id"},
				rows: [][]driver.Value{
					{"doc1.md"},
				},
			}, nil
		}
		return nil, nil
	}
	mockQueryMu.Unlock()

	_, err = BuildFromSQLite(context.Background(), db, "", 0, "mock-repo")
	if err == nil {
		t.Fatal("esperava erro de scan em graph_edges, obteve nil")
	}
}
