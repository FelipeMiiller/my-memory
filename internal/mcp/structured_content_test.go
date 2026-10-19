package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

func TestCallToolResult_StructuredContentIsAdditive(t *testing.T) {
	// Uma tool que não preenche o payload não emite a chave: o JSON das 24
	// tools que não o preenchem continua idêntico.
	plain, err := json.Marshal(NewTextResult("texto"))
	if err != nil {
		t.Fatalf("falha ao serializar: %v", err)
	}
	if strings.Contains(string(plain), "structuredContent") {
		t.Errorf("o resultado sem payload não pode conter structuredContent: %s", plain)
	}

	// NewStructuredResult preenche os dois: o texto humano e o payload.
	structured := NewStructuredResult("texto", map[string]string{"chave": "valor"})
	if len(structured.Content) != 1 || structured.Content[0].Text != "texto" {
		t.Errorf("o texto legível foi alterado: %+v", structured.Content)
	}
	if structured.StructuredContent == nil {
		t.Fatal("o payload estruturado não foi preenchido")
	}

	encoded, err := json.Marshal(structured)
	if err != nil {
		t.Fatalf("falha ao serializar: %v", err)
	}
	if !strings.Contains(string(encoded), `"structuredContent":{"chave":"valor"}`) {
		t.Errorf("o payload não foi serializado: %s", encoded)
	}
}

func TestMemorySearchTool_DoesNotEmitStructuredContent(t *testing.T) {
	handler := NewMemorySearchHandler(nil)

	res, err := handler(context.Background(), json.RawMessage(`{"query":"   "}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result, ok := res.(CallToolResult)
	if !ok {
		t.Fatalf("esperado CallToolResult, obteve %T", res)
	}
	if result.StructuredContent != nil {
		t.Errorf("memory_search não deve preencher structuredContent: %+v", result.StructuredContent)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("falha ao serializar: %v", err)
	}
	if strings.Contains(string(encoded), "structuredContent") {
		t.Errorf("o JSON não pode conter structuredContent: %s", encoded)
	}
}

func TestMemoryDoctorTool_StructuredContentJSONIsStable(t *testing.T) {
	report := &DoctorReport{
		TotalDocuments:    10,
		TotalChunks:       42,
		TotalEdges:        7,
		TotalNodes:        9,
		HealthScore:       88,
		MissingMTimeCount: 1,
		DeadLinks:         []DeadLink{{SourceID: "a.md", TargetID: "b.md", Relation: "relates_to"}},
		OrphanNotes:       []OrphanNote{{ID: "c.md", Title: "C"}},
		SelfLoops:         []SelfLoop{{NodeID: "d.md", Relation: "self"}},
		DesyncedChunks:    []DesyncedChunk{{ChunkID: "ch1", DocumentID: "doc1", Issue: "sem vetor"}},
	}

	handler := NewMemoryDoctorHandler(func(ctx context.Context, repo string) (*DoctorReport, error) {
		return report, nil
	}, nil)

	res, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	if !strings.Contains(result.Content[0].Text, "Health Score: 88/100") {
		t.Errorf("o texto legível mudou: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	expected := `{"total_documents":10,"total_chunks":42,"total_edges":7,"total_nodes":9,"health_score":88,` +
		`"missing_mtime_count":1,"dead_links":[{"source_id":"a.md","target_id":"b.md","relation":"relates_to"}],` +
		`"orphan_notes":[{"id":"c.md","title":"C"}],"self_loops":[{"node_id":"d.md","relation":"self"}],` +
		`"desynced_chunks":[{"chunk_id":"ch1","document_id":"doc1","issue":"sem vetor"}]}`

	if string(payload) != expected {
		t.Errorf("payload inesperado:\n obtive: %s\n esperava: %s", payload, expected)
	}
}

func TestMemoryGetHubsTool_StructuredContentJSONIsStable(t *testing.T) {
	hubs := []GodNode{{ID: "n1", Name: "N1", InDegree: 3, OutDegree: 2, TotalDegree: 5}}

	handler := NewAdvancedMemoryGetHubsHandler(func(ctx context.Context, repo string, limit int) ([]GodNode, error) {
		return hubs, nil
	}, nil)

	res, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	if !strings.Contains(result.Content[0].Text, "N1") {
		t.Errorf("o texto legível mudou: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	expected := `[{"id":"n1","name":"N1","in_degree":3,"out_degree":2,"total_degree":5}]`
	if string(payload) != expected {
		t.Errorf("payload inesperado:\n obtive: %s\n esperava: %s", payload, expected)
	}
}

func TestMemoryGetClustersTool_StructuredContentCarriesModularity(t *testing.T) {
	handler := NewMemoryGetClustersHandler(func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error) {
		return graph.CommunityResult{
			Communities: []graph.Community{
				{ID: 1, Label: "cluster", LeadNode: "a.md", Members: []string{"a.md", "b.md"}, Size: 2, DominantType: "concept"},
				{ID: 2, Label: "pequeno", LeadNode: "c.md", Members: []string{"c.md"}, Size: 1},
			},
			Modularity: 0.42,
			TotalNodes: 3,
			TotalEdges: 2,
		}, nil
	})

	res, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	expected := `{"communities":[{"id":1,"label":"cluster","lead_node":"a.md","members":["a.md","b.md"],"size":2,"dominant_type":"concept"}],` +
		`"modularity":0.42,"total_nodes":3,"total_edges":2}`
	if string(payload) != expected {
		t.Errorf("payload inesperado:\n obtive: %s\n esperava: %s", payload, expected)
	}
}

func TestMemoryListFilesTool_StructuredContentCarriesDocuments(t *testing.T) {
	docs := []store.VaultDocument{
		{ID: "d1", Path: "docs/a.md", Title: "A", MTime: 1700000000, Size: 120},
		{ID: "d2", Path: "docs/b.md", Title: "B", MTime: 0, Size: 0},
	}

	handler := NewMemoryListFilesHandler(func(ctx context.Context, repo, ext, folder string) ([]store.VaultDocument, error) {
		return docs, nil
	})

	res, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	if !strings.Contains(result.Content[0].Text, "docs/a.md") {
		t.Errorf("o texto legível mudou: %s", result.Content[0].Text)
	}

	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	expected := `[{"id":"d1","path":"docs/a.md","title":"A","mtime":1700000000,"size":120},` +
		`{"id":"d2","path":"docs/b.md","title":"B","mtime":0,"size":0}]`
	if string(payload) != expected {
		t.Errorf("payload inesperado:\n obtive: %s\n esperava: %s", payload, expected)
	}
}

func TestMemoryVisualizeGraphTool_StructuredContentHasURLAndPath(t *testing.T) {
	base := t.TempDir()

	handler := NewMemoryVisualizeGraphHandler(nil, base, func() string {
		return "http://127.0.0.1:38400"
	})

	res, err := handler(context.Background(), json.RawMessage(`{"output_path":"graph.html"}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	if !strings.Contains(string(payload), `"url":"http://127.0.0.1:38400/graph/graph.html"`) {
		t.Errorf("o payload não traz a url servida: %s", payload)
	}
	if !strings.Contains(string(payload), `"path":`) {
		t.Errorf("o payload não traz o caminho do artefato: %s", payload)
	}
}
