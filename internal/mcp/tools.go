package mcp

import (
	"context"
	"encoding/json"
)

// ListToolsResult representa a resposta da listagem de ferramentas MCP
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// Schemas das ferramentas expostas pelo My-Memory
var (
	ToolMemorySearch = Tool{
		Name:        "memory_search",
		Description: "Busca semântica k-NN na memória de notas usando sqlite-vec e expansão de conexões no grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Texto ou pergunta para busca semântica na base de notas",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Número máximo de resultados a retornar (padrão: 5)",
				},
			},
			"required": []string{"query"},
		},
	}

	ToolMemoryGetNeighbors = Tool{
		Name:        "memory_get_neighbors",
		Description: "Retorna nós e notas vizinhas no grafo relacional a partir de um node_id via CTE recursivo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador do nó (caminho do arquivo ou título da nota)",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de travessia no grafo (padrão: 1)",
				},
			},
			"required": []string{"node_id"},
		},
	}
)

// RegisterTool adiciona ou atualiza uma ferramenta e seu respectivo handler no servidor
func (s *Server) RegisterTool(tool Tool, handler ToolHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, t := range s.tools {
		if t.Name == tool.Name {
			s.tools[i] = tool
			found = true
			break
		}
	}
	if !found {
		s.tools = append(s.tools, tool)
	}

	if handler != nil {
		s.toolHandlers[tool.Name] = handler
	}
}

// RegisterToolHandler registra apenas a função executora de uma ferramenta
func (s *Server) RegisterToolHandler(name string, handler ToolHandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolHandlers[name] = handler
}

// GetTools retorna uma cópia de todas as ferramentas registradas
func (s *Server) GetTools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, len(s.tools))
	copy(tools, s.tools)
	return tools
}

// handleToolsList responde à requisição tools/list retornando o catálogo de ferramentas registradas
func (s *Server) handleToolsList(ctx context.Context, params json.RawMessage) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ListToolsResult{
		Tools: s.tools,
	}, nil
}
