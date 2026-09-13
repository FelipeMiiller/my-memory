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
)

func TestServer_ToolsCall_MemorySearch_Success(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 20, "method": "tools/call", "params": {"name": "memory_search", "arguments": {"repository": "my-org/my-repo", "query": "arquitetura", "limit": 2}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetSearchHandler(func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error) {
		if repo != "my-org/my-repo" {
			t.Errorf("esperava repo 'my-org/my-repo', obteve '%s'", repo)
		}
		if query != "arquitetura" {
			t.Errorf("esperava query 'arquitetura', obteve '%s'", query)
		}
		if limit != 2 {
			t.Errorf("esperava limit 2, obteve %d", limit)
		}
		return []SearchResult{
			{
				ChunkID:    "doc1#0",
				DocumentID: "docs/arch.md",
				Repository: repo,
				Content:    "Este documento detalha a arquitetura do sistema.",
				Distance:   0.1234,
				Neighbors:  []string{"docs/intro.md", "docs/spec.md"},
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	if string(resp.ID) != "20" {
		t.Errorf("esperava ID 20, obteve %s", string(resp.ID))
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resBytes, &callResult); err != nil {
		t.Fatalf("erro ao decodificar CallToolResult: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava blocos de conteúdo no resultado")
	}

	text := callResult.Content[0].Text
	if !strings.Contains(text, "docs/arch.md") {
		t.Errorf("esperava 'docs/arch.md' no texto, obteve: %s", text)
	}
	if !strings.Contains(text, "0.1234") {
		t.Errorf("esperava distância '0.1234' no texto, obteve: %s", text)
	}
	if !strings.Contains(text, "docs/intro.md") {
		t.Errorf("esperava vizinhos no grafo, obteve: %s", text)
	}
}

func TestServer_ToolsCall_MemorySearch_EmptyQuery(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 21, "method": "tools/call", "params": {"name": "memory_search", "arguments": {"query": "   "}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	called := false
	srv.SetSearchHandler(func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error) {
		called = true
		return nil, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	if called {
		t.Errorf("searchHandler não deveria ser chamado para query vazia")
	}

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("query vazia não deve retornar erro rpc, obteve: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)

	if len(callResult.Content) == 0 || !strings.Contains(callResult.Content[0].Text, "Nenhum resultado") {
		t.Errorf("esperava mensagem de 'Nenhum resultado', obteve: %+v", callResult)
	}
}

func TestServer_ToolsCall_MemorySearch_MissingQuery(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 22, "method": "tools/call", "params": {"name": "memory_search", "arguments": {"limit": 10}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("esperava erro para query ausente, obteve nil")
	}

	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("esperava CodeInvalidParams (%d), obteve %d", CodeInvalidParams, resp.Error.Code)
	}

	if !strings.Contains(resp.Error.Message, "query") {
		t.Errorf("esperava mensagem citando 'query', obteve: %s", resp.Error.Message)
	}
}

func TestServer_ToolsCall_MemoryGetNeighbors_Success(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 23, "method": "tools/call", "params": {"name": "memory_get_neighbors", "arguments": {"repository": "my-org/my-repo", "node_id": "docs/architecture.md", "max_depth": 2}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetNeighborsHandler(func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
		if repo != "my-org/my-repo" {
			t.Errorf("esperava repo 'my-org/my-repo', obteve '%s'", repo)
		}
		if nodeID != "docs/architecture.md" {
			t.Errorf("esperava nodeID 'docs/architecture.md', obteve '%s'", nodeID)
		}
		if maxDepth != 2 {
			t.Errorf("esperava maxDepth 2, obteve %d", maxDepth)
		}
		return []string{"docs/note1.md", "docs/note2.md"}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava conteúdo na resposta")
	}

	text := callResult.Content[0].Text
	if !strings.Contains(text, "docs/note1.md") || !strings.Contains(text, "docs/note2.md") {
		t.Errorf("esperava nós vizinhos no texto, obteve: %s", text)
	}
}

func TestServer_ToolsCall_MemoryGetNeighbors_MissingNodeID(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 24, "method": "tools/call", "params": {"name": "memory_get_neighbors", "arguments": {}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("esperava erro para node_id ausente, obteve nil")
	}

	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("esperava CodeInvalidParams (%d), obteve %d", CodeInvalidParams, resp.Error.Code)
	}
}

func TestServer_ToolsCall_UnknownTool(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 25, "method": "tools/call", "params": {"name": "unknown_tool", "arguments": {}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error == nil {
		t.Fatalf("esperava erro para ferramenta desconhecida, obteve nil")
	}

	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("esperava CodeMethodNotFound (-32601), obteve %d", resp.Error.Code)
	}
}

func TestServer_ToolsCall_InvalidParamsJSON(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 26, "method": "tools/call", "params": "not-an-object"}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error == nil || resp.Error.Code != CodeInvalidParams {
		t.Errorf("esperava CodeInvalidParams (-32602), obteve: %+v", resp.Error)
	}
}

func TestServer_ToolsCall_MemoryExportCanvas_StringOutput(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 30, "method": "tools/call", "params": {"name": "memory_export_canvas", "arguments": {"node_id": "Arquitetura"}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)
	srv.SetNeighborsHandler(func(ctx context.Context, repo, nodeID string, maxDepth int) ([]string, error) {
		return []string{"Banco de Dados", "Autenticacao"}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava conteúdo retornado")
	}

	rawJSON := callResult.Content[0].Text
	if !strings.Contains(rawJSON, "Arquitetura.md") {
		t.Errorf("esperava 'Arquitetura.md' no canvas retornado, obteve: %s", rawJSON)
	}
	if !strings.Contains(rawJSON, "nodes") || !strings.Contains(rawJSON, "edges") {
		t.Errorf("esperava JSON Canvas válido com nodes e edges, obteve: %s", rawJSON)
	}
}

func TestServer_ToolsCall_MemoryExportCanvas_SaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	canvasPath := filepath.Join(tmpDir, "exported.canvas")

	// Usamos formato JSON escapado para o caminho
	escapedPath := strings.ReplaceAll(canvasPath, `\`, `\\`)
	in := `{"jsonrpc": "2.0", "id": 31, "method": "tools/call", "params": {"name": "memory_export_canvas", "arguments": {"node_id": "Arquitetura", "output_path": "` + escapedPath + `"}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)
	srv.SetNeighborsHandler(func(ctx context.Context, repo, nodeID string, maxDepth int) ([]string, error) {
		return []string{"Banco de Dados"}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	// Verifica se o arquivo foi realmente gerado no disco
	data, err := os.ReadFile(canvasPath)
	if err != nil {
		t.Fatalf("esperava arquivo salvo em disco: %v", err)
	}

	if !strings.Contains(string(data), "Banco de Dados.md") {
		t.Errorf("conteúdo salvo não contém nó esperado: %s", string(data))
	}
}

func TestServer_ToolsCall_MemorySearch_AdvancedHybrid(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 40, "method": "tools/call", "params": {"name": "memory_search", "arguments": {"query": "turboquant", "mode": "hybrid", "limit": 3, "k": 50}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetAdvancedSearchHandler(func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		if params.Query != "turboquant" {
			t.Errorf("esperava query 'turboquant', obteve '%s'", params.Query)
		}
		if params.Mode != "hybrid" {
			t.Errorf("esperava mode 'hybrid', obteve '%s'", params.Mode)
		}
		if params.Limit != 3 {
			t.Errorf("esperava limit 3, obteve %d", params.Limit)
		}
		if params.K != 50 {
			t.Errorf("esperava k 50, obteve %d", params.K)
		}

		return []SearchResult{
			{
				ChunkID:    "chunk-tq#1",
				DocumentID: "docs/tq.md",
				Content:    "TurboQuant comprime vetores em 4-bits mantendo alta fidelidade.",
				Score:      0.0325,
				Sources:    []string{"fts:1", "vector:1"},
				Neighbors:  []string{"docs/embeddings.md"},
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callRes CallToolResult
	if err := json.Unmarshal(resBytes, &callRes); err != nil {
		t.Fatalf("falha ao converter CallToolResult: %v", err)
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "Score RRF: 0.0325") {
		t.Errorf("resposta esperada com Score RRF, obteve: %s", text)
	}
	if !strings.Contains(text, "Fontes RRF: [fts:1, vector:1]") {
		t.Errorf("resposta esperada com Fontes RRF, obteve: %s", text)
	}
}

func TestFormatSearchResults_Empty(t *testing.T) {
	got := FormatSearchResults(nil)
	if got != "Nenhum resultado encontrado." {
		t.Errorf("esperava 'Nenhum resultado encontrado.', obteve: %s", got)
	}
}

func TestServer_ToolsCall_MemoryGetHubs_Success(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 50, "method": "tools/call", "params": {"name": "memory_get_hubs", "arguments": {"repository": "my-org/my-repo", "top": 5}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetHubsHandler(func(ctx context.Context, repo string, limit int) ([]GodNode, error) {
		if repo != "my-org/my-repo" {
			t.Errorf("esperava repo 'my-org/my-repo', obteve '%s'", repo)
		}
		if limit != 5 {
			t.Errorf("esperava limit 5, obteve %d", limit)
		}
		return []GodNode{
			{
				ID:          "Arquitetura",
				Name:        "Arquitetura do Sistema",
				InDegree:    10,
				OutDegree:   4,
				TotalDegree: 14,
			},
			{
				ID:          "BancoDados",
				Name:        "Banco de Dados",
				InDegree:    8,
				OutDegree:   2,
				TotalDegree: 10,
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resBytes, &callResult); err != nil {
		t.Fatalf("erro ao decodificar CallToolResult: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava conteúdo na resposta")
	}

	text := callResult.Content[0].Text
	if !strings.Contains(text, "Arquitetura do Sistema") {
		t.Errorf("resposta deve conter 'Arquitetura do Sistema', obteve: %s", text)
	}
	if !strings.Contains(text, "Grau Total: 14") {
		t.Errorf("resposta deve conter 'Grau Total: 14', obteve: %s", text)
	}
}

func TestFormatHubs_Empty(t *testing.T) {
	got := FormatHubs(nil)
	if !strings.Contains(got, "Nenhum nó central") {
		t.Errorf("esperava mensagem de vazio, obteve: %s", got)
	}
}

func TestServer_ToolsCall_MemoryGetHubs_PageRank(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 45, "method": "tools/call", "params": {"name": "memory_get_hubs", "arguments": {"repository": "my-org/my-repo", "top": 3, "algorithm": "pagerank"}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetAdvancedHubsHandler(nil, func(ctx context.Context, repo string, limit int) ([]PageRankNode, error) {
		if repo != "my-org/my-repo" {
			t.Errorf("esperava repo 'my-org/my-repo', obteve '%s'", repo)
		}
		if limit != 3 {
			t.Errorf("esperava limit 3, obteve %d", limit)
		}
		return []PageRankNode{
			{
				ID:        "doc-core",
				Name:      "Core Arquitetural",
				Score:     0.3540,
				Rank:      1,
				InDegree:  15,
				OutDegree: 3,
			},
			{
				ID:        "doc-db",
				Name:      "Camada de Banco",
				Score:     0.2810,
				Rank:      2,
				InDegree:  10,
				OutDegree: 2,
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resBytes, &callResult); err != nil {
		t.Fatalf("erro ao decodificar CallToolResult: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava conteúdo na resposta")
	}

	text := callResult.Content[0].Text
	if !strings.Contains(text, "Core Arquitetural") {
		t.Errorf("resposta deve conter 'Core Arquitetural', obteve: %s", text)
	}
	if !strings.Contains(text, "Score: 0.3540 (35.40%)") {
		t.Errorf("resposta deve conter 'Score: 0.3540 (35.40%%)', obteve: %s", text)
	}
}

func TestFormatPageRankHubs_Empty(t *testing.T) {
	got := FormatPageRankHubs(nil)
	if !strings.Contains(got, "Nenhum nó encontrado") {
		t.Errorf("esperava mensagem de vazio, obteve: %s", got)
	}
}

func TestServer_ToolsCall_MemoryGetInsights_Success(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 50, "method": "tools/call", "params": {"name": "memory_get_insights", "arguments": {"repository": "my-org/my-repo", "limit": 3, "min_similarity": 0.80}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	srv.SetInsightsHandler(func(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error) {
		if repo != "my-org/my-repo" {
			t.Errorf("esperava repo 'my-org/my-repo', obteve '%s'", repo)
		}
		if limit != 3 {
			t.Errorf("esperava limit 3, obteve %d", limit)
		}
		if minSimilarity != 0.80 {
			t.Errorf("esperava minSimilarity 0.80, obteve %f", minSimilarity)
		}
		return []SurprisingConnection{
			{
				SourceID:   "doc-a",
				SourceName: "Arquitetura Híbrida",
				TargetID:   "doc-b",
				TargetName: "Design Patterns Go",
				Similarity: 0.875,
				Reason:     "Alta proximidade semântica (88%) sem conexão direta no grafo",
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resBytes, &callResult); err != nil {
		t.Fatalf("erro ao decodificar CallToolResult: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatalf("esperava conteúdo na resposta")
	}

	text := callResult.Content[0].Text
	if !strings.Contains(text, "Arquitetura Híbrida") {
		t.Errorf("resposta deve conter 'Arquitetura Híbrida', obteve: %s", text)
	}
	if !strings.Contains(text, "Design Patterns Go") {
		t.Errorf("resposta deve conter 'Design Patterns Go', obteve: %s", text)
	}
	if !strings.Contains(text, "87.5%") {
		t.Errorf("resposta deve conter '87.5%%', obteve: %s", text)
	}
}

func TestServer_ToolsCall_MemoryGetInsights_Defaults(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 51, "method": "tools/call", "params": {"name": "memory_get_insights"}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	called := false
	srv.SetInsightsHandler(func(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error) {
		called = true
		if limit != 10 {
			t.Errorf("esperava limit default 10, obteve %d", limit)
		}
		if minSimilarity != 0.70 {
			t.Errorf("esperava minSimilarity default 0.70, obteve %f", minSimilarity)
		}
		return []SurprisingConnection{}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	if !called {
		t.Fatalf("handler de insights deveria ter sido chamado")
	}

	var resp Response
	_ = json.Unmarshal(out.Bytes(), &resp)
	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)
	if len(callResult.Content) == 0 || !strings.Contains(callResult.Content[0].Text, "Nenhuma conexão inesperada encontrada") {
		t.Errorf("esperava mensagem vazia de retorno, obteve: %+v", callResult)
	}
}

func TestFormatInsights_Empty(t *testing.T) {
	got := FormatInsights(nil)
	if !strings.Contains(got, "Nenhuma conexão inesperada") {
		t.Errorf("esperava mensagem de vazio, obteve: %s", got)
	}
}

func TestServer_ToolsCall_MemoryDoctor_Success(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 60, "method": "tools/call", "params": {"name": "memory_doctor", "arguments": {"repository": "my-repo"}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	diagnoseCalled := false
	srv.SetDoctorHandler(
		func(ctx context.Context, repo string) (*DoctorReport, error) {
			diagnoseCalled = true
			if repo != "my-repo" {
				t.Errorf("esperava repo 'my-repo', obteve '%s'", repo)
			}
			return &DoctorReport{
				TotalDocuments: 5,
				TotalChunks:    15,
				TotalEdges:     10,
				TotalNodes:     8,
				HealthScore:    90,
				DeadLinks: []DeadLink{
					{SourceID: "notes/a.md", TargetID: "Ghost", Relation: "links_to"},
				},
				OrphanNotes: []OrphanNote{},
				SelfLoops:   []SelfLoop{},
			}, nil
		},
		nil,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	if !diagnoseCalled {
		t.Fatalf("diagnose handler deveria ter sido chamado")
	}

	var resp Response
	_ = json.Unmarshal(out.Bytes(), &resp)
	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)

	if len(callResult.Content) == 0 {
		t.Fatalf("resultado vazio de tools/call")
	}
	text := callResult.Content[0].Text
	if !strings.Contains(text, "Health Score: 90/100") || !strings.Contains(text, "Ghost") {
		t.Errorf("resposta inesperada do doctor: %s", text)
	}
}

func TestServer_ToolsCall_MemoryDoctor_WithFix(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 61, "method": "tools/call", "params": {"name": "memory_doctor", "arguments": {"fix": true}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	fixCalled := false
	srv.SetDoctorHandler(
		func(ctx context.Context, repo string) (*DoctorReport, error) {
			return &DoctorReport{
				TotalDocuments: 3,
				HealthScore:    100,
			}, nil
		},
		func(ctx context.Context, repo string) (int, error) {
			fixCalled = true
			return 2, nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	if !fixCalled {
		t.Fatalf("fix handler deveria ter sido chamado")
	}

	var resp Response
	_ = json.Unmarshal(out.Bytes(), &resp)
	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	_ = json.Unmarshal(resBytes, &callResult)

	if len(callResult.Content) == 0 || !strings.Contains(callResult.Content[0].Text, "2 arestas problemáticas removidas") {
		t.Errorf("esperava menção ao reparo de 2 arestas, obteve: %+v", callResult)
	}
}

func TestToolMemorySearch_SchemaDecayProperties(t *testing.T) {
	props, ok := ToolMemorySearch.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("InputSchema properties inválido")
	}
	if _, exists := props["decay"]; !exists {
		t.Errorf("esperava propriedade 'decay' no schema de ToolMemorySearch")
	}
	if _, exists := props["half_life"]; !exists {
		t.Errorf("esperava propriedade 'half_life' no schema de ToolMemorySearch")
	}
	if _, exists := props["decay_weight"]; !exists {
		t.Errorf("esperava propriedade 'decay_weight' no schema de ToolMemorySearch")
	}
}

func TestServer_ToolsCall_MemorySearch_WithDecay(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 70, "method": "tools/call", "params": {"name": "memory_search", "arguments": {"query": "deep learning", "decay": true, "half_life": 15.0, "decay_weight": 0.5}}}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	called := false
	srv.SetAdvancedSearchHandler(func(ctx context.Context, params SearchParams) ([]SearchResult, error) {
		called = true
		if !params.Decay {
			t.Errorf("esperava Decay true, obteve %v", params.Decay)
		}
		if params.HalfLife != 15.0 {
			t.Errorf("esperava HalfLife 15.0, obteve %f", params.HalfLife)
		}
		if params.DecayWeight != 0.5 {
			t.Errorf("esperava DecayWeight 0.5, obteve %f", params.DecayWeight)
		}
		return []SearchResult{
			{
				ChunkID:    "c_1",
				DocumentID: "notes/ai.md",
				Content:    "pesquisa recente sobre transformers",
				Score:      0.035,
				UpdatedAt:  1700000000,
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	if !called {
		t.Fatalf("SetAdvancedSearchHandler deveria ter sido invocado")
	}

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta JSON-RPC: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso, obteve erro: %+v", resp.Error)
	}

	resBytes, _ := json.Marshal(resp.Result)
	var callResult CallToolResult
	if err := json.Unmarshal(resBytes, &callResult); err != nil {
		t.Fatalf("erro ao decodificar CallToolResult: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatalf("resultado vazio de tools/call")
	}
	text := callResult.Content[0].Text
	if !strings.Contains(text, "Atualizado em:") {
		t.Errorf("esperava 'Atualizado em:' no texto formatado, obteve: %s", text)
	}
	if !strings.Contains(text, "notes/ai.md") {
		t.Errorf("esperava 'notes/ai.md' no texto, obteve: %s", text)
	}
}
