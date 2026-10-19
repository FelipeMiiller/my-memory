package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graphview"
)

// graphArtifactsPath devolve o caminho do artefato dentro da raiz servida,
// que é onde a tool grava desde a ADR-059.
func graphArtifactsPath(base, name string) string {
	return filepath.Join(GraphArtifactsDir(base), name)
}

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

	handler := NewMemoryVisualizeGraphHandler(mockFn, tmpDir, nil)

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

	// Verifica criação física do arquivo, dentro da raiz servida
	targetPath := graphArtifactsPath(tmpDir, outFile)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("arquivo HTML gerado não existe no disco: %v", err)
	}

	if !strings.Contains(string(data), "Grafo de Teste MCP") {
		t.Errorf("HTML gravado não contém os dados do mock: %s", string(data))
	}
}

func TestMemoryVisualizeGraphTool_InvalidJSON(t *testing.T) {
	handler := NewMemoryVisualizeGraphHandler(nil, "", nil)
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

	handler := NewMemoryVisualizeGraphHandler(mockFn, "", nil)
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
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir, nil)

	res, err := handler(context.Background(), json.RawMessage(`{"output_path": "fallback.html"}`))
	if err != nil {
		t.Fatalf("handler com builder nil não deveria falhar: %v", err)
	}

	result := res.(CallToolResult)
	if len(result.Content) == 0 {
		t.Fatal("resultado vazio")
	}

	if _, err := os.Stat(graphArtifactsPath(tmpDir, "fallback.html")); err != nil {
		t.Fatalf("arquivo de fallback não foi criado: %v", err)
	}
}

func TestMemoryVisualizeGraphTool_AutoOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir, nil)

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

	expectedFile := graphArtifactsPath(tmpDir, "graph_concepts_deep_learning.html")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Fatalf("arquivo esperado %s não foi encontrado no disco", expectedFile)
	}
}

func TestMemoryVisualizeGraphTool_ExportError(t *testing.T) {
	// O diretório de artefatos é criado pela tool; quando a criação falha, a
	// falha vira mensagem de tool, nunca panic.
	base := t.TempDir()
	blocked := filepath.Join(base, ".mymemory", "www", "graph")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatalf("falha ao preparar o bloqueio: %v", err)
	}
	if err := os.WriteFile(blocked, []byte("isto e um arquivo, nao um diretorio"), 0o644); err != nil {
		t.Fatalf("falha ao criar o arquivo de bloqueio: %v", err)
	}

	handler := NewMemoryVisualizeGraphHandler(nil, base, nil)
	_, err := handler(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("esperava erro quando o diretório de artefatos não pode ser criado")
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

func TestMemoryVisualizeGraphTool_UrlWithActiveTransport(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir, func() string {
		return "http://127.0.0.1:38400"
	})

	res, err := handler(context.Background(), json.RawMessage(`{"output_path": "graph.html"}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	text := result.Content[0].Text
	if !strings.Contains(text, "http://127.0.0.1:38400/graph/graph.html") {
		t.Errorf("a URL servida não está no resultado: %s", text)
	}
	if !strings.Contains(text, "**Arquivo HTML**") {
		t.Errorf("o caminho do arquivo não está no resultado: %s", text)
	}

	structured, ok := result.StructuredContent.(VisualizeGraphResult)
	if !ok {
		t.Fatalf("structuredContent inesperado: %T", result.StructuredContent)
	}
	if structured.URL != "http://127.0.0.1:38400/graph/graph.html" {
		t.Errorf("URL no structuredContent inesperada: %s", structured.URL)
	}
	if structured.Path == "" {
		t.Error("o structuredContent deve trazer o caminho do artefato")
	}
}

func TestMemoryVisualizeGraphTool_NoUrlWithoutTransport(t *testing.T) {
	tmpDir := t.TempDir()
	// Sem transporte HTTP, o provedor devolve string vazia.
	handler := NewMemoryVisualizeGraphHandler(nil, tmpDir, func() string {
		return ""
	})

	res, err := handler(context.Background(), json.RawMessage(`{"output_path": "graph.html"}`))
	if err != nil {
		t.Fatalf("sem transporte HTTP a tool não pode falhar: %v", err)
	}

	result := res.(CallToolResult)
	text := result.Content[0].Text
	if strings.Contains(text, "URL Servida") {
		t.Errorf("sem transporte HTTP não deve haver URL no resultado: %s", text)
	}
	if !strings.Contains(text, "**Arquivo HTML**") {
		t.Errorf("o caminho do arquivo deve continuar no resultado: %s", text)
	}

	structured, ok := result.StructuredContent.(VisualizeGraphResult)
	if !ok {
		t.Fatalf("structuredContent inesperado: %T", result.StructuredContent)
	}
	if structured.URL != "" {
		t.Errorf("sem transporte HTTP o structuredContent não deve ter url: %s", structured.URL)
	}
	if structured.Path == "" {
		t.Error("o structuredContent deve trazer o caminho do artefato")
	}
}

func TestMemoryVisualizeGraphTool_UrlUsesRealListenerPort(t *testing.T) {
	base := t.TempDir()

	srv := newTestServer()
	httpSrv := NewHTTPServer(srv, HTTPServerOptions{ServedRoot: base})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("falha ao abrir listener tcp: %v", err)
	}
	go func() {
		_ = httpSrv.Serve(ln)
	}()
	defer func() { _ = httpSrv.Shutdown(context.Background()) }()

	deadline := time.Now().Add(2 * time.Second)
	for srv.BaseURL() == "" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if srv.BaseURL() == "" {
		t.Fatal("o transporte não publicou o endereço vinculado")
	}

	port := ln.Addr().(*net.TCPAddr).Port
	srv.SetGraphViewHandler(func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
		return &graphview.GraphView{Title: "Grafo"}, nil
	}, base)

	handler := srv.toolHandlers[ToolMemoryVisualizeGraph.Name]
	res, err := handler(context.Background(), json.RawMessage(`{"output_path": "graph.html"}`))
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	result := res.(CallToolResult)
	structured, ok := result.StructuredContent.(VisualizeGraphResult)
	if !ok {
		t.Fatalf("structuredContent inesperado: %T", result.StructuredContent)
	}

	expected := "http://127.0.0.1:" + strconv.Itoa(port) + "/graph/graph.html"
	if structured.URL != expected {
		t.Errorf("URL inesperada: %s (esperado %s)", structured.URL, expected)
	}
}
