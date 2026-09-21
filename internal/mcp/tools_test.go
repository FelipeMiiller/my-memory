package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTools_DefaultSchemas(t *testing.T) {
	// Verificação da ferramenta memory_search
	search := ToolMemorySearch
	if search.Name != "memory_search" {
		t.Errorf("esperava nome 'memory_search', obteve '%s'", search.Name)
	}
	if search.Description == "" {
		t.Errorf("esperava descrição não vazia para memory_search")
	}
	schema, ok := search.InputSchema["type"].(string)
	if !ok || schema != "object" {
		t.Errorf("esperava inputSchema.type == 'object', obteve %v", search.InputSchema["type"])
	}
	props, ok := search.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("esperava properties no inputSchema de memory_search")
	}
	if _, ok := props["query"]; !ok {
		t.Errorf("esperava propriedade 'query' no inputSchema de memory_search")
	}
	if _, ok := props["repository"]; !ok {
		t.Errorf("esperava propriedade 'repository' no inputSchema de memory_search")
	}
	required, ok := search.InputSchema["required"].([]string)
	if !ok || len(required) == 0 || required[0] != "query" {
		t.Errorf("esperava 'query' nos campos required de memory_search, obteve %v", search.InputSchema["required"])
	}

	// Verificação da ferramenta memory_get_neighbors
	neighbors := ToolMemoryGetNeighbors
	if neighbors.Name != "memory_get_neighbors" {
		t.Errorf("esperava nome 'memory_get_neighbors', obteve '%s'", neighbors.Name)
	}
	if neighbors.Description == "" {
		t.Errorf("esperava descrição não vazia para memory_get_neighbors")
	}
	neighborsSchema, ok := neighbors.InputSchema["type"].(string)
	if !ok || neighborsSchema != "object" {
		t.Errorf("esperava inputSchema.type == 'object', obteve %v", neighbors.InputSchema["type"])
	}
	neighborsProps, ok := neighbors.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("esperava properties no inputSchema de memory_get_neighbors")
	}
	if _, ok := neighborsProps["node_id"]; !ok {
		t.Errorf("esperava propriedade 'node_id' no inputSchema de memory_get_neighbors")
	}
	if _, ok := neighborsProps["repository"]; !ok {
		t.Errorf("esperava propriedade 'repository' no inputSchema de memory_get_neighbors")
	}
	neighborsReq, ok := neighbors.InputSchema["required"].([]string)
	if !ok || len(neighborsReq) == 0 || neighborsReq[0] != "node_id" {
		t.Errorf("esperava 'node_id' nos campos required de memory_get_neighbors, obteve %v", neighbors.InputSchema["required"])
	}

	// Verificação da ferramenta memory_export_canvas
	canvasTool := ToolMemoryExportCanvas
	if canvasTool.Name != "memory_export_canvas" {
		t.Errorf("esperava nome 'memory_export_canvas', obteve '%s'", canvasTool.Name)
	}
	if canvasTool.Description == "" {
		t.Errorf("esperava descrição não vazia para memory_export_canvas")
	}
	canvasProps, ok := canvasTool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("esperava properties no inputSchema de memory_export_canvas")
	}
	if _, ok := canvasProps["node_id"]; !ok {
		t.Errorf("esperava propriedade 'node_id' no inputSchema de memory_export_canvas")
	}
	if _, ok := canvasProps["output_path"]; !ok {
		t.Errorf("esperava propriedade 'output_path' no inputSchema de memory_export_canvas")
	}

	// Verificação da ferramenta memory_get_hubs
	hubsTool := ToolMemoryGetHubs
	if hubsTool.Name != "memory_get_hubs" {
		t.Errorf("esperava nome 'memory_get_hubs', obteve '%s'", hubsTool.Name)
	}
	if hubsTool.Description == "" {
		t.Errorf("esperava descrição não vazia para memory_get_hubs")
	}
	hubsProps, ok := hubsTool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("esperava properties no inputSchema de memory_get_hubs")
	}
	if _, ok := hubsProps["top"]; !ok {
		t.Errorf("esperava propriedade 'top' no inputSchema de memory_get_hubs")
	}
	if _, ok := hubsProps["repository"]; !ok {
		t.Errorf("esperava propriedade 'repository' no inputSchema de memory_get_hubs")
	}
}

func TestServer_ToolsList(t *testing.T) {
	in := `{"jsonrpc": "2.0", "id": 10, "method": "tools/list"}` + "\n"
	var out bytes.Buffer

	srv := NewServer("test-server", "1.0.0", strings.NewReader(in), &out, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = srv.Run(ctx)

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("erro ao decodificar resposta do tools/list: %v\nSaída: %s", err, out.String())
	}

	if resp.Error != nil {
		t.Fatalf("esperava sucesso em tools/list, obteve erro: %+v", resp.Error)
	}

	if string(resp.ID) != "10" {
		t.Errorf("esperava ID 10, obteve %s", string(resp.ID))
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("erro ao serializar result: %v", err)
	}

	var listResult ListToolsResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		t.Fatalf("erro ao decodificar ListToolsResult: %v", err)
	}

	if len(listResult.Tools) < 2 {
		t.Fatalf("esperava pelo menos 2 ferramentas em tools/list, obteve %d", len(listResult.Tools))
	}

	foundSearch := false
	foundNeighbors := false
	foundCanvas := false
	foundHubs := false
	foundCodeSearch := false
	foundCodeNeighbors := false
	for _, tool := range listResult.Tools {
		if tool.Name == "memory_search" {
			foundSearch = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_search na resposta: %+v", tool.InputSchema)
			}
		}
		if tool.Name == "memory_get_neighbors" {
			foundNeighbors = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_get_neighbors na resposta: %+v", tool.InputSchema)
			}
		}
		if tool.Name == "memory_export_canvas" {
			foundCanvas = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_export_canvas na resposta: %+v", tool.InputSchema)
			}
		}
		if tool.Name == "memory_get_hubs" {
			foundHubs = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_get_hubs na resposta: %+v", tool.InputSchema)
			}
		}
		if tool.Name == "memory_code_search" {
			foundCodeSearch = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_code_search na resposta: %+v", tool.InputSchema)
			}
		}
		if tool.Name == "memory_code_neighbors" {
			foundCodeNeighbors = true
			if tool.InputSchema == nil || tool.InputSchema["type"] != "object" {
				t.Errorf("inputSchema inválido para memory_code_neighbors na resposta: %+v", tool.InputSchema)
			}
		}
	}

	if !foundSearch {
		t.Errorf("ferramenta 'memory_search' não encontrada em tools/list")
	}
	if !foundNeighbors {
		t.Errorf("ferramenta 'memory_get_neighbors' não encontrada em tools/list")
	}
	if !foundCanvas {
		t.Errorf("ferramenta 'memory_export_canvas' não encontrada em tools/list")
	}
	if !foundHubs {
		t.Errorf("ferramenta 'memory_get_hubs' não encontrada em tools/list")
	}
	// GAP-5 fix: garantir que memory_code_search e memory_code_neighbors aparecem
	// no catálogo MCP, defendendo o path CA-10/CA-11 contra remoção acidental.
	if !foundCodeSearch {
		t.Errorf("ferramenta 'memory_code_search' não encontrada em tools/list (CA-10/ADR-047)")
	}
	if !foundCodeNeighbors {
		t.Errorf("ferramenta 'memory_code_neighbors' não encontrada em tools/list (CA-11/ADR-047)")
	}
}

func TestServer_RegisterTool_Custom(t *testing.T) {
	var out bytes.Buffer
	srv := NewServer("test-server", "1.0.0", strings.NewReader(""), &out, nil)

	customTool := Tool{
		Name:        "custom_tool",
		Description: "Uma ferramenta de teste",
		InputSchema: map[string]any{
			"type": "object",
		},
	}

	srv.RegisterTool(customTool, func(ctx context.Context, args json.RawMessage) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	tools := srv.GetTools()
	found := false
	for _, tool := range tools {
		if tool.Name == "custom_tool" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("ferramenta custom_tool não encontrada após RegisterTool")
	}
}
