package mcp

import (
	"bytes"
	"context"
	"encoding/json"
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
