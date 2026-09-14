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

func TestMemoryVisualizeGraphTool_InvalidJSON(t *testing.T) {
	handler := NewMemoryVisualizeGraphHandler(nil, "")
	_, err := handler(context.Background(), json.RawMessage(`{invalid-json`))
	if err == nil {
		t.Fatal("esperava erro para JSON inválido, obteve nil")
	}

	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInvalidParams {
		t.Errorf("esperado erro com código CodeInvalidParams (%d), obteve: %v", CodeInvalidParams, err)
	}
}

func TestMemoryVisualizeGraphTool_BuilderError(t *testing.T) {
	mockFn := func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
		return nil, os.ErrPermission
	}

	handler := NewMemoryVisualizeGraphHandler(mockFn, "")
	_, err := handler(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("esperava erro quando builder falha, obteve nil")
	}

	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInternalError {
		t.Errorf("esperado erro CodeInternalError (%d), obteve: %v", CodeInternalError, err)
	}
}

func TestMemoryVisualizeGraphTool_NilBuilderFallback(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir)

	res, err := handler(context.Background(), json.RawMessage(`{"output_path": "fallback.html"}`))
	if err != nil {
		t.Fatalf("handler com builder nil não deveria falhar: %v", err)
	}

	result := res.(CallToolResult)
	if len(result.Content) == 0 {
		t.Fatal("resultado vazio")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "fallback.html")); err != nil {
		t.Fatalf("arquivo de fallback não foi criado: %v", err)
	}
}

func TestMemoryVisualizeGraphTool_AutoOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir)

	args := json.RawMessage(`{"root_node": "concepts/deep/learning.md", "max_depth": 3}`)
	res, err := handler(context.Background(), args)
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	text := result.Content[0].Text
	if !strings.Contains(text, "graph_concepts_deep_learning.html") {
		t.Errorf("esperado nome sanitizado 'graph_concepts_deep_learning.html' no resultado, obteve: %s", text)
	}

	expectedFile := filepath.Join(tmpDir, "graph_concepts_deep_learning.html")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("arquivo esperado %s não foi encontrado no disco", expectedFile)
	}
}

func TestMemoryVisualizeGraphTool_ExportError(t *testing.T) {
	handler := NewMemoryVisualizeGraphHandler(nil, "")

	// Tenta gravar em subdiretório inexistente para induzir falha de exportação
	invalidOut := filepath.Join(os.TempDir(), "non_existent_folder_abc123", "sub", "graph.html")
	args, _ := json.Marshal(map[string]any{"output_path": invalidOut})

	_, err := handler(context.Background(), args)
	if err == nil {
		t.Fatal("esperava erro ao tentar exportar em caminho de diretório inexistente")
	}

	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInternalError {
		t.Errorf("esperado erro CodeInternalError (%d), obteve: %v", CodeInternalError, err)
	}
}

func TestServer_SetGraphViewHandler(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errLog := &bytes.Buffer{}

	srv := NewServer("test-mcp", "1.0.0", in, out, errLog)
	called := false
	mockFn := func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
		called = true
		return &graphview.GraphView{Title: "Registered"}, nil
	}

	srv.SetGraphViewHandler(mockFn, t.TempDir())

	handler := srv.toolHandlers[ToolMemoryVisualizeGraph.Name]
	if handler == nil {
		t.Fatalf("ferramenta %s não registrada no servidor", ToolMemoryVisualizeGraph.Name)
	}

	_, err := handler(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execução do handler registrado falhou: %v", err)
	}
	if !called {
		t.Error("esperava que mockFn tivesse sido invocado")
	}
}
