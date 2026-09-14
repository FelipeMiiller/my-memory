package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestMemoryGetClustersHandler_Direct(t *testing.T) {
	ctx := context.Background()

	mockFunc := func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error) {
		return graph.CommunityResult{
			Communities: []graph.Community{
				{
					ID:           1,
					Label:        "concept/auth",
					LeadNode:     "concept/auth",
					Members:      []string{"concept/auth", "decision/jwt", "guide/auth"},
					Size:         3,
					DominantType: "concept",
				},
				{
					ID:           2,
					Label:        "concept/db",
					LeadNode:     "concept/db",
					Members:      []string{"concept/db", "decision/postgres"},
					Size:         2,
					DominantType: "decision",
				},
				{
					ID:           3,
					Label:        "other/isolated",
					LeadNode:     "other/isolated",
					Members:      []string{"other/isolated"},
					Size:         1,
					DominantType: "other",
				},
			},
			Modularity: 0.4321,
			TotalNodes: 6,
			TotalEdges: 8,
		}, nil
	}

	handler := NewMemoryGetClustersHandler(mockFunc)

	// 1. Chamada padrão (min_size = 2 por padrão no handler)
	argsDefault := json.RawMessage(`{}`)
	res, err := handler(ctx, argsDefault)
	if err != nil {
		t.Fatalf("erro ao executar handler padrão: %v", err)
	}

	textRes, ok := res.(CallToolResult)
	if !ok || len(textRes.Content) == 0 {
		t.Fatalf("esperava CallToolResult com conteúdo, obteve: %+v", res)
	}

	text := textRes.Content[0].Text
	if !strings.Contains(text, "0.4321") {
		t.Errorf("modularidade ausente na saída MCP:\n%s", text)
	}
	if !strings.Contains(text, "concept/auth") || !strings.Contains(text, "concept/db") {
		t.Errorf("clusters esperados ausentes na tabela:\n%s", text)
	}
	// O cluster isolado de tamanho 1 deve ter sido filtrado pelo min_size=2
	if strings.Contains(text, "other/isolated") {
		t.Errorf("cluster de tamanho 1 não deveria aparecer com min_size padrão:\n%s", text)
	}

	// 2. Chamada com min_size = 1 (deve incluir cluster isolado)
	argsMin1 := json.RawMessage(`{"min_size": 1}`)
	res1, err := handler(ctx, argsMin1)
	if err != nil {
		t.Fatalf("erro com min_size=1: %v", err)
	}
	text1 := res1.(CallToolResult).Content[0].Text
	if !strings.Contains(text1, "other/isolated") {
		t.Errorf("cluster isolado deveria aparecer com min_size=1:\n%s", text1)
	}
}

func TestMemoryGetClustersHandler_Empty(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryGetClustersHandler(nil)

	res, err := handler(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("erro com handler nil: %v", err)
	}
	text := res.(CallToolResult).Content[0].Text
	if !strings.Contains(text, "Nenhum cluster com o tamanho mínimo solicitado foi encontrado") {
		t.Errorf("esperava mensagem de lista vazia:\n%s", text)
	}
}

func TestServer_MemoryGetClusters_RegistrationAndCall(t *testing.T) {
	in := bytes.NewBuffer(nil)
	out := io.Discard
	errLog := io.Discard

	srv := NewServer("test-mcp", "1.0.0", in, out, errLog)

	// Verificar se a ferramenta está na lista de ferramentas
	tools := srv.GetTools()
	found := false
	for _, tName := range tools {
		if tName.Name == "memory_get_clusters" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ferramenta memory_get_clusters não registrada por padrão no servidor MCP")
	}

	// Registrar mock e chamar via SetClustersHandler
	srv.SetClustersHandler(func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error) {
		return graph.CommunityResult{
			Communities: []graph.Community{
				{
					ID:       1,
					LeadNode: "hub-1",
					Members:  []string{"hub-1", "leaf-1"},
					Size:     2,
				},
			},
			Modularity: 0.5,
			TotalNodes: 2,
			TotalEdges: 1,
		}, nil
	})

	callParams, _ := json.Marshal(CallToolParams{
		Name:      "memory_get_clusters",
		Arguments: json.RawMessage(`{"min_size": 2}`),
	})

	res, err := srv.handleToolsCall(context.Background(), callParams)
	if err != nil {
		t.Fatalf("falha ao chamar tools/call memory_get_clusters: %v", err)
	}

	callRes := res.(CallToolResult)
	if len(callRes.Content) == 0 || !strings.Contains(callRes.Content[0].Text, "hub-1") {
		t.Errorf("conteúdo da chamada tools/call incorreto: %+v", callRes)
	}
}
