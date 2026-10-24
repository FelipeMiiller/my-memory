package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestCodeToolsEmitemStructuredContent prova que as tools de código emitem
// structuredContent ADITIVO: o texto JSON continua sendo a saída humana e o
// payload estruturado é o que a rota /code da viewer consome.
func TestCodeToolsEmitemStructuredContent(t *testing.T) {
	ctx := context.Background()
	srv := NewServer("test", "0.0.0", os.Stdin, os.Stdout, nil)
	srv.SetCodeAccess(CodeAccess{
		Search: func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
			return []CodeSymbolResult{{SymbolID: 7, QualifiedName: "pkg.Root", Name: "Root"}}, nil
		},
		Neighbors: func(ctx context.Context, symbol string, depth int, direction string, maxTokens int) (*MemoryCodeNeighborsResponse, error) {
			return &MemoryCodeNeighborsResponse{
				Root:      &CodeSymbolResult{SymbolID: 7, QualifiedName: "pkg.Root"},
				Neighbors: []CodeNeighborResult{{Direction: "out", Kind: "calls", Symbol: "pkg.Other", File: "pkg/other.go", Line: 3, Hop: 1}},
				Depth:     depth,
				Direction: direction,
			}, nil
		},
	})

	// 1. memory_code_search
	searchHandler := srv.toolHandlers[ToolMemoryCodeSearch.Name]
	if searchHandler == nil {
		t.Fatal("memory_code_search sem executor")
	}
	searchResult, err := searchHandler(ctx, json.RawMessage(`{"query":"Root"}`))
	if err != nil {
		t.Fatalf("memory_code_search: %v", err)
	}
	searchText, ok := searchResult.(CallToolResult)
	if !ok {
		t.Fatalf("memory_code_search devolveu %T; want CallToolResult", searchResult)
	}
	if len(searchText.Content) == 0 {
		t.Error("memory_code_search perdeu a saída textual")
	}
	searchPayload, ok := searchText.StructuredContent.(MemoryCodeSearchResponse)
	if !ok {
		t.Fatalf("memory_code_search structuredContent = %T; want MemoryCodeSearchResponse",
			searchText.StructuredContent)
	}
	if searchPayload.Query != "Root" {
		t.Errorf("query = %q; want %q", searchPayload.Query, "Root")
	}
	if len(searchPayload.Results) != 1 {
		t.Errorf("results = %d; want 1", len(searchPayload.Results))
	}

	// 2. memory_code_neighbors
	neighborsHandler := srv.toolHandlers[ToolMemoryCodeNeighbors.Name]
	if neighborsHandler == nil {
		t.Fatal("memory_code_neighbors sem executor")
	}
	neighborsResult, err := neighborsHandler(ctx, json.RawMessage(`{"symbol":"pkg.Root","depth":2}`))
	if err != nil {
		t.Fatalf("memory_code_neighbors: %v", err)
	}
	neighborsText, ok := neighborsResult.(CallToolResult)
	if !ok {
		t.Fatalf("memory_code_neighbors devolveu %T; want CallToolResult", neighborsResult)
	}
	if len(neighborsText.Content) == 0 {
		t.Error("memory_code_neighbors perdeu a saída textual")
	}
	neighborsPayload, ok := neighborsText.StructuredContent.(*MemoryCodeNeighborsResponse)
	if !ok {
		t.Fatalf("memory_code_neighbors structuredContent = %T; want *MemoryCodeNeighborsResponse",
			neighborsText.StructuredContent)
	}
	if len(neighborsPayload.Neighbors) != 1 {
		t.Errorf("neighbors = %d; want 1", len(neighborsPayload.Neighbors))
	}
}

// TestBuscaDeCodigoRespeitaOrcamento prova que a BUSCA também respeita o
// orçamento de tokens e avisa que truncou.
func TestBuscaDeCodigoRespeitaOrcamento(t *testing.T) {
	ctx := context.Background()
	srv := NewServer("test", "0.0.0", os.Stdin, os.Stdout, nil)
	srv.SetCodeAccess(CodeAccess{
		Search: func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
			hits := make([]CodeSymbolResult, 0, 5)
			for i := 0; i < 5; i++ {
				hits = append(hits, CodeSymbolResult{
					SymbolID:      int64(i),
					QualifiedName: "pkg.Sym",
					Name:          "Sym",
					Signature:     strings.Repeat("x", 400),
				})
			}
			return hits, nil
		},
	})

	handler := srv.toolHandlers[ToolMemoryCodeSearch.Name]
	if handler == nil {
		t.Fatal("memory_code_search sem executor")
	}
	result, err := handler(ctx, json.RawMessage(`{"query":"Sym","max_tokens":120}`))
	if err != nil {
		t.Fatalf("memory_code_search: %v", err)
	}
	text, ok := result.(CallToolResult)
	if !ok {
		t.Fatalf("devolveu %T; want CallToolResult", result)
	}
	payload, ok := text.StructuredContent.(MemoryCodeSearchResponse)
	if !ok {
		t.Fatalf("structuredContent = %T; want MemoryCodeSearchResponse", text.StructuredContent)
	}
	if !payload.Truncated {
		t.Errorf("truncated = false com orçamento de 120 tokens; want true")
	}
	if !strings.Contains(strings.ToLower(payload.Hint), "estreite") {
		t.Errorf("hint = %q; want a dica de estreitar a query", payload.Hint)
	}
	if payload.Count >= 5 {
		t.Errorf("count = %d; want < 5 (o orçamento tem que cortar)", payload.Count)
	}
}
