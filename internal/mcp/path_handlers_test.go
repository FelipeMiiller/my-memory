package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/graph"
)

func TestMemoryFindPathHandler_Direct(t *testing.T) {
	ctx := context.Background()

	mockFunc := func(ctx context.Context, repo, source, target string, opts graph.PathOptions) (*graph.PathResult, error) {
		if source == "missing" || target == "missing" {
			return nil, errors.New("node not found")
		}
		if source == "unreachable" {
			return &graph.PathResult{
				Found:     false,
				Source:    source,
				Target:    target,
				Directed:  opts.Directed,
				CostMode:  opts.CostMode,
				Hops:      0,
				TotalCost: 0,
				Nodes:     []string{},
				Edges:     []graph.PathEdge{},
				Summary:   "Nenhum caminho encontrado",
			}, nil
		}
		return &graph.PathResult{
			Found:     true,
			Source:    source,
			Target:    target,
			Directed:  opts.Directed,
			CostMode:  opts.CostMode,
			Hops:      2,
			TotalCost: 2.0,
			Nodes:     []string{source, "intermediate", target},
			Edges: []graph.PathEdge{
				{
					From:            source,
					To:              "intermediate",
					Relation:        "implements",
					EpistemicStatus: "EXTRACTED",
					Weight:          1.0,
					Cost:            1.0,
					Direction:       "forward",
				},
				{
					From:            "intermediate",
					To:              target,
					Relation:        "depends_on",
					EpistemicStatus: "EXTRACTED",
					Weight:          1.0,
					Cost:            1.0,
					Direction:       "forward",
				},
			},
			Summary: "Caminho encontrado com 2 salto(s)",
		}, nil
	}

	handler := NewMemoryFindPathHandler(mockFunc)

	// 1. Sucesso com caminho encontrado
	args, _ := json.Marshal(map[string]any{
		"source":    "auth",
		"target":    "redis",
		"max_depth": 4,
	})
	res, err := handler(ctx, args)
	if err != nil {
		t.Fatalf("handler falhou: %v", err)
	}

	callRes, ok := res.(CallToolResult)
	if !ok || len(callRes.Content) == 0 {
		t.Fatalf("resposta não é CallToolResult: %v", res)
	}
	content := callRes.Content[0].Text

	if !strings.Contains(content, "### 🧭 Descoberta de Rotas no Grafo de Conhecimento") {
		t.Errorf("cabeçalho não encontrado:\n%s", content)
	}
	if !strings.Contains(content, "2 salto(s)") {
		t.Errorf("contagem de 2 saltos não encontrada:\n%s", content)
	}
	if !strings.Contains(content, "auth") || !strings.Contains(content, "intermediate") || !strings.Contains(content, "redis") {
		t.Errorf("nós não encontrados no resultado:\n%s", content)
	}
	if !strings.Contains(content, "implements") || !strings.Contains(content, "depends_on") {
		t.Errorf("relações das arestas não encontradas:\n%s", content)
	}

	// 2. Caminho não encontrado
	argsUnreach, _ := json.Marshal(map[string]any{
		"source": "unreachable",
		"target": "redis",
	})
	resUnreach, err := handler(ctx, argsUnreach)
	if err != nil {
		t.Fatalf("handler falhou para unreachable: %v", err)
	}
	unreachContent := resUnreach.(CallToolResult).Content[0].Text
	if !strings.Contains(unreachContent, "Nenhum caminho encontrado") {
		t.Errorf("mensagem de não encontrado esperada:\n%s", unreachContent)
	}

	// 3. Erro interno quando função retorna erro
	argsErr, _ := json.Marshal(map[string]any{
		"source": "missing",
		"target": "redis",
	})
	_, err = handler(ctx, argsErr)
	if err == nil {
		t.Fatal("esperava erro quando mock retorna erro")
	}
	mcpErr, ok := err.(*Error)
	if !ok || mcpErr.Code != CodeInternalError {
		t.Errorf("esperava CodeInternalError, obteve %v", err)
	}
}

func TestMemoryFindPathHandler_Validation(t *testing.T) {
	ctx := context.Background()
	handler := NewMemoryFindPathHandler(nil)

	// 1. JSON inválido
	_, err := handler(ctx, json.RawMessage("invalid-json"))
	if err == nil {
		t.Fatal("esperava erro para JSON inválido")
	}

	// 2. Ausência de source
	argsNoSrc, _ := json.Marshal(map[string]any{"target": "auth"})
	_, err = handler(ctx, argsNoSrc)
	if err == nil {
		t.Fatal("esperava erro ao omitir source")
	}
	if !strings.Contains(err.Error(), "source") {
		t.Errorf("mensagem esperava mencionar source: %v", err)
	}

	// 3. Ausência de target
	argsNoTgt, _ := json.Marshal(map[string]any{"source": "auth"})
	_, err = handler(ctx, argsNoTgt)
	if err == nil {
		t.Fatal("esperava erro ao omitir target")
	}
	if !strings.Contains(err.Error(), "target") {
		t.Errorf("mensagem esperava mencionar target: %v", err)
	}
}

func TestMemoryFindPath_ServerRegistration(t *testing.T) {
	server := NewServer("test-server", "1.0", nil, nil, nil)
	tools := server.GetTools()

	found := false
	for _, tool := range tools {
		if tool.Name == "memory_find_path" {
			found = true
			schema := tool.InputSchema
			props, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("properties inválidas no schema: %v", schema)
			}
			if _, ok := props["source"]; !ok {
				t.Errorf("propriedade 'source' não encontrada")
			}
			if _, ok := props["target"]; !ok {
				t.Errorf("propriedade 'target' não encontrada")
			}
			if _, ok := props["max_depth"]; !ok {
				t.Errorf("propriedade 'max_depth' não encontrada")
			}
			if _, ok := props["directed"]; !ok {
				t.Errorf("propriedade 'directed' não encontrada")
			}
			if _, ok := props["mode"]; !ok {
				t.Errorf("propriedade 'mode' não encontrada")
			}
			break
		}
	}

	if !found {
		t.Fatalf("ferramenta memory_find_path não registrada no servidor")
	}
}
