package mcp

import (
	"context"
	"os"
	"testing"
)

// BenchmarkCodeAccess_Search mede o custo de uma busca de código com o
// executor injetado no servidor (o caminho que o agente usa via MCP).
func BenchmarkCodeAccess_Search(b *testing.B) {
	database, _ := codeHandlersTestFixture(b)
	defer database.Close()

	srv := NewServer("bench", "0.0.0", os.Stdin, os.Stdout, nil)
	srv.SetCodeAccess(CodeAccess{
		Search: func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
			return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
		},
		Neighbors: func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
			return SQLCodeNeighbors(ctx, database, symbol, depth, direction)
		},
	})

	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := srv.handleToolsCall(ctx, []byte(`{"name":"memory_code_search","arguments":{"query":"Hello"}}`)); err != nil {
			b.Fatalf("memory_code_search: %v", err)
		}
	}
}

// BenchmarkCodeAccess_Neighbors mede a travessia do code graph pelo mesmo
// caminho do agente.
func BenchmarkCodeAccess_Neighbors(b *testing.B) {
	database, _ := codeHandlersTestFixture(b)
	defer database.Close()

	srv := NewServer("bench", "0.0.0", os.Stdin, os.Stdout, nil)
	srv.SetCodeAccess(CodeAccess{
		Search: func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
			return SQLCodeSearch(ctx, database, query, language, kind, limit, includeBoost)
		},
		Neighbors: func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
			return SQLCodeNeighbors(ctx, database, symbol, depth, direction)
		},
	})

	ctx := context.Background()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := srv.handleToolsCall(ctx, []byte(`{"name":"memory_code_neighbors","arguments":{"symbol":"foo.Hello"}}`)); err != nil {
			b.Fatalf("memory_code_neighbors: %v", err)
		}
	}
}
