package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestMemoryInspectNodeHandler_Direct(t *testing.T) {
	ctx := context.Background()

	mockFunc := func(ctx context.Context, repo, nodeID string, maxContentLen int) (*graph.TriptychView, error) {
		if nodeID != "core-auth" {
			return nil, errors.New("node not found")
		}
		return &graph.TriptychView{
			Target: graph.NodeSummary{
				ID:             "core-auth",
				Title:          "Core Authentication",
				Path:           "docs/auth.md",
				Type:           "decision",
				Tags:           []string{"#auth", "#security"},
				PageRank:       0.0450,
				CommunityID:    2,
				CommunityLabel: "Security",
				RiskScore:      85.0,
				RiskLevel:      "CRÍTICO",
				ContentPreview: "Esta nota especifica a arquitetura central de autenticação e sessões.",
				ContentLength:  68,
				UpdatedAt:      time.Now(),
			},
			Inbound: []graph.InboundLink{
				{
					SourceID: "api-gateway",
					Title:    "API Gateway",
					Type:     "service",
					Relation: "implements",
					Severity: graph.SeverityCritical,
					PageRank: 0.1200,
				},
				{
					SourceID: "web-client",
					Title:    "Web Client",
					Type:     "client",
					Relation: "links_to",
					Severity: graph.SeverityMedium,
					PageRank: 0.0300,
				},
			},
			Outbound: []graph.OutboundLink{
				{
					TargetID: "jwt-spec",
					Title:    "JWT Specification",
					Type:     "concept",
					Relation: "depends_on",
					PageRank: 0.0500,
					Exists:   true,
				},
				{
					TargetID: "oauth-legacy",
					Title:    "OAuth Legacy",
					Type:     "other",
					Relation: "supersedes",
					PageRank: 0.0,
					Exists:   false, // dead link
				},
			},
			TotalInbound:       2,
			TotalOutbound:      2,
			CriticalDependents: 1,
			GeneratedAt:        time.Now(),
		}, nil
	}

	handler := NewMemoryInspectNodeHandler(mockFunc)

	// 1. Chamada válida com argumentos
	argsValid := json.RawMessage(`{"node_id": "core-auth", "max_content_length": 500, "repository": "test-repo"}`)
	res, err := handler(ctx, argsValid)
	if err != nil {
		t.Fatalf("erro inesperado ao executar handler: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("esperava CallToolResult válido, obteve: %+v", res)
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "Visualização Cirúrgica de Nó (Triptych Node Inspector)") {
		t.Errorf("cabeçalho esperado ausente:\n%s", text)
	}
	if !strings.Contains(text, "Core Authentication") {
		t.Errorf("título do nó central ausente:\n%s", text)
	}
	if !strings.Contains(text, "85.0 / 100") || !strings.Contains(text, "CRÍTICO") {
		t.Errorf("score de risco ausente:\n%s", text)
	}
	if !strings.Contains(text, "api-gateway") {
		t.Errorf("inbound link api-gateway ausente:\n%s", text)
	}
	if !strings.Contains(text, "jwt-spec") {
		t.Errorf("outbound link jwt-spec ausente:\n%s", text)
	}
	if !strings.Contains(text, "DEAD LINK") {
		t.Errorf("aviso de dead link ausente para oauth-legacy:\n%s", text)
	}
	if !strings.Contains(text, "Alerta de Risco") {
		t.Errorf("alerta de dependentes críticos ausente:\n%s", text)
	}

	// 2. Validação de erro para node_id ausente
	argsMissing := json.RawMessage(`{}`)
	_, err = handler(ctx, argsMissing)
	if err == nil {
		t.Fatal("esperava erro para node_id ausente, obteve nil")
	}

	// 3. Validação de nó inexistente
	argsNotFound := json.RawMessage(`{"node_id": "nao-existe"}`)
	_, err = handler(ctx, argsNotFound)
	if err == nil {
		t.Fatal("esperava erro para nó inexistente, obteve nil")
	}
}

func TestMemoryInspectNode_ServerIntegration(t *testing.T) {
	ctx := context.Background()
	inBuf := &bytes.Buffer{}
	outBuf := &bytes.Buffer{}

	server := NewServer("test-server", "1.0.0", inBuf, outBuf, io.Discard)

	// Verifica se a ferramenta memory_inspect_node está no catálogo padrão
	tools := server.GetTools()
	found := false
	for _, tName := range tools {
		if tName.Name == "memory_inspect_node" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ferramenta memory_inspect_node não registrada por padrão no servidor MCP")
	}

	// Configura handler customizado via SetInspectHandler
	server.SetInspectHandler(func(ctx context.Context, repo, nodeID string, maxLen int) (*graph.TriptychView, error) {
		return &graph.TriptychView{
			Target: graph.NodeSummary{
				ID:        nodeID,
				Title:     "Target Title",
				RiskScore: 20.0,
				RiskLevel: "BAIXO",
			},
			TotalInbound:  0,
			TotalOutbound: 0,
		}, nil
	})

	// Testa execução via Dispatch do JSON-RPC
	callArgs, _ := json.Marshal(map[string]any{
		"name": "memory_inspect_node",
		"arguments": map[string]any{
			"node_id": "custom-node",
		},
	})

	callRes, err := server.handleToolsCall(ctx, callArgs)
	if err != nil {
		t.Fatalf("falha ao chamar tools/call memory_inspect_node: %v", err)
	}

	result, ok := callRes.(CallToolResult)
	if !ok || len(result.Content) == 0 {
		t.Fatalf("resultado inválido de tools/call: %+v", callRes)
	}

	if !strings.Contains(result.Content[0].Text, "custom-node") {
		t.Errorf("texto esperado não encontrado no output: %s", result.Content[0].Text)
	}
}

func TestMemoryInspectNodeHandler_NilFuncFallback(t *testing.T) {
	ctx := context.Background()
	// Handler com função nil deve retornar fallback padrão sem entrar em pânico
	handler := NewMemoryInspectNodeHandler(nil)

	args := json.RawMessage(`{"node_id": "fallback-node"}`)
	res, err := handler(ctx, args)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resultado inválido: %+v", res)
	}

	if !strings.Contains(callRes.Content[0].Text, "fallback-node") {
		t.Errorf("texto esperado não encontrado no fallback: %s", callRes.Content[0].Text)
	}
}

func TestMemoryInspectNodeHandler_FederatedOutbound(t *testing.T) {
	ctx := context.Background()

	mockFunc := func(ctx context.Context, repo, nodeID string, maxContentLen int) (*graph.TriptychView, error) {
		return &graph.TriptychView{
			Target: graph.NodeSummary{
				ID:    "service-impl",
				Title: "Service Implementation",
				Type:  "note",
			},
			Inbound: []graph.InboundLink{},
			Outbound: []graph.OutboundLink{
				{
					TargetID:     "memory://central/standards/oauth2",
					Title:        "memory://central/standards/oauth2",
					Type:         "note",
					Relation:     "implements",
					IsFederated:  true,
					CanonicalURI: "memory://central/standards/oauth2",
					Exists:       true,
				},
			},
			TotalInbound:  0,
			TotalOutbound: 1,
			GeneratedAt:   time.Now(),
		}, nil
	}

	handler := NewMemoryInspectNodeHandler(mockFunc)
	args := json.RawMessage(`{"node_id": "service-impl"}`)
	res, err := handler(ctx, args)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resultado inválido: %+v", res)
	}

	text := callRes.Content[0].Text
	if !strings.Contains(text, "Federado (is_federated: true)") {
		t.Errorf("esperava indicador 'Federado (is_federated: true)' no outbound, obteve:\n%s", text)
	}
	if !strings.Contains(text, "memory://central/standards/oauth2") {
		t.Errorf("esperava targetID federado, obteve:\n%s", text)
	}
}

