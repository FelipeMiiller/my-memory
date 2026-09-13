package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graphview"
)

func TestMemoryVisualizeGraphTool_List(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errLog := &bytes.Buffer{}

	srv := NewServer("test-mcp", "1.0.0", in, out, errLog)

	tools := srv.GetTools()
	found := false
	for _, tool := range tools {
		if tool.Name == "memory_visualize_graph" {
			found = true
			if !strings.Contains(tool.Description, "HTML") {
				t.Errorf("descrição da ferramenta memory_visualize_graph deve mencionar HTML: %s", tool.Description)
			}
			break
		}
	}

	if !found {
		t.Fatalf("ferramenta 'memory_visualize_graph' não encontrada na lista de ferramentas padrão do MCP")
	}
}

func TestMemoryVisualizeGraphTool_Call(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp_graph_test_*")
	if err != nil {
		t.Fatalf("falha ao criar pasta temporária: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockGV := &graphview.GraphView{
		Title:       "Grafo de Teste MCP",
		Repository:  "test/mcp",
		GeneratedAt: time.Now(),
		Stats: graphview.GraphStats{
			TotalNodes: 2,
			TotalEdges: 1,
			HubCount:   1,
			Density:    0.5,
		},
		Nodes: []graphview.Node{
			{ID: "concepts/mcp.md", Title: "MCP Protocol", Type: "concept", Color: "#3b82f6", Radius: 14.0},
			{ID: "decisions/json-rpc.md", Title: "JSON-RPC Specs", Type: "decision", Color: "#f43f5e", Radius: 10.0},
		},
		Edges: []graphview.Edge{
			{Source: "decisions/json-rpc.md", Target: "concepts/mcp.md", Relation: "implements", Weight: 1.0},
		},
	}

	mockFn := func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
		return mockGV, nil
	}

	handler := NewMemoryVisualizeGraphHandler(mockFn, tmpDir)

	outFile := "output_graph.html"
	args := map[string]any{
		"output_path": outFile,
		"repository":  "test/mcp",
	}
	argsBytes, _ := json.Marshal(args)

	res, err := handler(context.Background(), argsBytes)
	if err != nil {
		t.Fatalf("handler retornou erro inesperado: %v", err)
	}

	result, ok := res.(CallToolResult)
	if !ok {
		t.Fatalf("esperado CallToolResult, obteve %T", res)
	}

	if len(result.Content) == 0 {
		t.Fatalf("resultado vazio")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Visualização Interativa do Grafo") {
		t.Errorf("texto retornado não contém cabeçalho esperado: %s", text)
	}
	if !strings.Contains(text, "**Total de Nós**: 2") {
		t.Errorf("texto deve conter métrica de nós: %s", text)
	}

	// Verifica criação física do arquivo
	targetPath := filepath.Join(tmpDir, outFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("arquivo HTML gerado não existe no disco: %v", err)
	}

	if !strings.Contains(string(data), "Grafo de Teste MCP") {
		t.Errorf("HTML gravado não contém os dados do mock: %s", string(data))
	}
}
