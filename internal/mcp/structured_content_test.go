package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/drift"
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
		HealthScore:       healthScorePtr(88),
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

// TestStructuredContentNeverEmitsNullArrays fixa o contrato do wire: um array
// vazio viaja como `[]`, nunca como `null`.
//
// O `encoding/json` serializa slice nil como `null`, e o schema `zod/mini` da
// viewer aceita `undefined` em `.optional()`, mas NÃO `null`. Um array vazio
// chegando como `null` derrubava a validação do schema, o cliente caía no
// fallback textual e o painel `/drift` renderizava markdown cru em vez da UI
// estruturada. Este teste é o que impede a regressão de voltar.
func TestStructuredContentNeverEmitsNullArrays(t *testing.T) {
	cases := []struct {
		nome    string
		payload any
		arrays  []string
	}{
		{"drift", &drift.DriftReport{RangeStr: "HEAD~5..HEAD"}, []string{`"drifted_notes":[]`, `"uncovered_code":[]`}},
		{"doctor", &DoctorReport{}, []string{`"dead_links":[]`, `"orphan_notes":[]`, `"self_loops":[]`, `"desynced_chunks":[]`}},
		{"list_files", []store.VaultDocument(nil), []string{`[]`}},
		{"note_outline", []OutlineHeading(nil), []string{`[]`}},
		{"hubs_degree", []GodNode(nil), []string{`[]`}},
		{"hubs_pagerank", []PageRankNode(nil), []string{`[]`}},
		{"insights", []SurprisingConnection(nil), []string{`[]`}},
		{"clusters", graph.CommunityResult{}, []string{`"communities":[]`}},
	}

	for _, tc := range cases {
		t.Run(tc.nome, func(t *testing.T) {
			encoded, err := json.Marshal(NewStructuredResult("texto", tc.payload).StructuredContent)
			if err != nil {
				t.Fatalf("falha ao serializar: %v", err)
			}
			if strings.Contains(string(encoded), ":null") {
				t.Errorf("o payload serializou null em um campo array: %s", encoded)
			}
			for _, want := range tc.arrays {
				if !strings.Contains(string(encoded), want) {
					t.Errorf("o payload não contem %s: %s", want, encoded)
				}
			}
		})
	}
}

// TestMemoryGetDriftTool_EmptyReportEmitsEmptyArrays prova o caminho real: o
// handler de memory_get_drift, com um relatório sem drift nenhum, precisa
// serializar as listas como `[]`.
func TestMemoryGetDriftTool_EmptyReportEmitsEmptyArrays(t *testing.T) {
	handler := NewMemoryGetDriftHandler(func(ctx context.Context, rangeStr string, threshold float64, includeUncovered bool, repoSlug string) (*drift.DriftReport, error) {
		return &drift.DriftReport{RangeStr: rangeStr, AnalyzedAt: time.Now()}, nil
	})

	res, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	payload, err := json.Marshal(res.(CallToolResult).StructuredContent)
	if err != nil {
		t.Fatalf("falha ao serializar o payload: %v", err)
	}

	if strings.Contains(string(payload), "null") {
		t.Errorf("o relatório de drift vazio serializou null: %s", payload)
	}
	if !strings.Contains(string(payload), `"drifted_notes":[]`) {
		t.Errorf("drifted_notes deveria ser lista vazia: %s", payload)
	}
	if !strings.Contains(string(payload), `"uncovered_code":[]`) {
		t.Errorf("uncovered_code deveria ser lista vazia: %s", payload)
	}
}

// TestNormalizeStructuredContent_DoesNotMutateTheProducer garante que a
// normalização é pura: o relatório devolvido pelo produtor não pode ser
// alterado por quem o consome.
func TestNormalizeStructuredContent_DoesNotMutateTheProducer(t *testing.T) {
	report := &drift.DriftReport{RangeStr: "HEAD~5..HEAD"}

	normalized, ok := normalizeStructuredContent(report).(*drift.DriftReport)
	if !ok {
		t.Fatal("a normalização devolveu outro tipo")
	}
	if normalized.DriftedNotes == nil || normalized.UncoveredCode == nil {
		t.Error("o payload normalizado deveria ter as listas vazias, não nulas")
	}
	if report.DriftedNotes != nil || report.UncoveredCode != nil {
		t.Error("a normalização alterou o relatório do produtor")
	}
}
