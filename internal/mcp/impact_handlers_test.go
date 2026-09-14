package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestMemoryGetImpactHandler_Direct(t *testing.T) {
	ctx := context.Background()

	mockFunc := func(ctx context.Context, repo, nodeID string, maxDepth int) (*graph.ImpactResult, error) {
		if nodeID != "core-auth" {
			return nil, errors.New("node not found")
		}
		return &graph.ImpactResult{
			TargetNode:         "core-auth",
			MaxDepthReached:    2,
			TotalImpacted:      3,
			DirectDependents:   2,
			IndirectDependents: 1,
			RiskScore:          78.5,
			RiskLevel:          "ALTO",
			AffectedClusters:   []int{1, 2},
			SeverityCounts: map[graph.ImpactSeverity]int{
				graph.SeverityCritical: 1,
				graph.SeverityHigh:     1,
				graph.SeverityMedium:   1,
				graph.SeverityLow:      0,
			},
			Nodes: []graph.ImpactedNode{
				{
					ID:       "api-gateway",
					NodeType: "service",
					Depth:    1,
					Relation: "depends_on",
					Severity: graph.SeverityCritical,
					ViaNode:  "core-auth",
				},
				{
					ID:       "user-dashboard",
					NodeType: "ui",
					Depth:    1,
					Relation: "implements",
					Severity: graph.SeverityHigh,
					ViaNode:  "core-auth",
				},
				{
					ID:       "mobile-app",
					NodeType: "client",
					Depth:    2,
					Relation: "calls",
					Severity: graph.SeverityMedium,
					ViaNode:  "api-gateway",
				},
			},
		}, nil
	}

	handler := NewMemoryGetImpactHandler(mockFunc)

	// 1. Chamada válida com argumentos
	argsValid := json.RawMessage(`{"node_id": "core-auth", "max_depth": 2, "repository": "test-repo"}`)
	res, err := handler(ctx, argsValid)
	if err != nil {
		t.Fatalf("erro inesperado ao executar handler: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("esperava CallToolResult válido, obteve: %+v", res)
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "Raio de Destruição (Blast Radius)") {
		t.Errorf("título esperado ausente no markdown:\n%s", text)
	}
	if !strings.Contains(text, "core-auth") {
		t.Errorf("nó alvo ausente:\n%s", text)
	}
	if !strings.Contains(text, "78.5 / 100") || !strings.Contains(text, "ALTO") {
		t.Errorf("score de risco ou nível incorreto:\n%s", text)
	}
	if !strings.Contains(text, "api-gateway") || !strings.Contains(text, "user-dashboard") || !strings.Contains(text, "mobile-app") {
		t.Errorf("nós dependentes ausentes na tabela:\n%s", text)
	}
	if !strings.Contains(text, "Cluster 1, Cluster 2") {
		t.Errorf("clusters afetados ausentes no resumo:\n%s", text)
	}
	if !strings.Contains(text, "(direto)") {
		t.Errorf("marcação de dependência direta ausente na tabela:\n%s", text)
	}
}

func TestMemoryGetImpactHandler_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryGetImpactHandler(nil)

	// 1. node_id ausente
	_, err := handler(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("esperava erro para node_id ausente, mas não ocorreu")
	}
	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInvalidParams {
		t.Fatalf("esperava CodeInvalidParams (%d), obteve: %v", CodeInvalidParams, err)
	}

	// 2. node_id em branco
	_, err = handler(ctx, json.RawMessage(`{"node_id": "   "}`))
	if err == nil {
		t.Fatal("esperava erro para node_id em branco, mas não ocorreu")
	}

	// 3. JSON malformado
	_, err = handler(ctx, json.RawMessage(`{malformed}`))
	if err == nil {
		t.Fatal("esperava erro para JSON inválido")
	}
}

func TestMemoryGetImpactHandler_NilFuncDefault(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryGetImpactHandler(nil)

	res, err := handler(ctx, json.RawMessage(`{"node_id": "isolated-node"}`))
	if err != nil {
		t.Fatalf("erro inesperado com nil impactFunc: %v", err)
	}

	callRes := res.(CallToolResult)
	text := callRes.Content[0].Text
	if !strings.Contains(text, "Nenhum nó dependente identificado") {
		t.Errorf("esperava aviso de zero dependentes no markdown:\n%s", text)
	}
	if !strings.Contains(text, "0.0 / 100") {
		t.Errorf("esperava score 0.0 para nó sem dependentes:\n%s", text)
	}
}

func TestServer_MemoryGetImpact_RegistrationAndCall(t *testing.T) {
	in := bytes.NewBuffer(nil)
	out := io.Discard
	errLog := io.Discard

	srv := NewServer("test-mcp", "1.0.0", in, out, errLog)

	// Verificar se a ferramenta está na lista de ferramentas padrão
	tools := srv.GetTools()
	found := false
	for _, tName := range tools {
		if tName.Name == "memory_get_impact" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ferramenta memory_get_impact não registrada por padrão no servidor MCP")
	}

	// Configurar mock via SetImpactHandler
	srv.SetImpactHandler(func(ctx context.Context, repo, nodeID string, maxDepth int) (*graph.ImpactResult, error) {
		return &graph.ImpactResult{
			TargetNode:       nodeID,
			MaxDepthReached:  1,
			TotalImpacted:    1,
			DirectDependents: 1,
			RiskScore:        30.0,
			RiskLevel:        "MODERADO",
			Nodes: []graph.ImpactedNode{
				{
					ID:       "dependent-service",
					NodeType: "concept",
					Depth:    1,
					Relation: "depends_on",
					Severity: graph.SeverityHigh,
					ViaNode:  nodeID,
				},
			},
			SeverityCounts: map[graph.ImpactSeverity]int{
				graph.SeverityHigh: 1,
			},
		}, nil
	})

	callParams, _ := json.Marshal(CallToolParams{
		Name:      "memory_get_impact",
		Arguments: json.RawMessage(`{"node_id": "core-module", "max_depth": 3}`),
	})

	res, err := srv.handleToolsCall(context.Background(), callParams)
	if err != nil {
		t.Fatalf("falha ao chamar tools/call memory_get_impact: %v", err)
	}

	callRes := res.(CallToolResult)
	if len(callRes.Content) == 0 || !strings.Contains(callRes.Content[0].Text, "dependent-service") {
		t.Errorf("conteúdo da chamada tools/call incorreto: %+v", callRes)
	}
}
