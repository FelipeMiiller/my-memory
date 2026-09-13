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
		Description: "Busca híbrida com Reciprocal Rank Fusion (RRF) combinando texto exato FTS5/tsvector, vetores k-NN e expansão de grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Texto ou pergunta para busca na base de memória de notas",
				},
				"mode": map[string]any{
					"type":        "string",
					"enum":        []string{"hybrid", "vector", "fts"},
					"description": "Modo de busca: 'hybrid' (padrão, RRF unificando FTS + vetores + grafo), 'vector' (apenas semântico k-NN) ou 'fts' (apenas texto exato)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Número máximo de resultados a retornar (padrão: 5)",
				},
				"k": map[string]any{
					"type":        "integer",
					"description": "Constante de suavização do algoritmo RRF (padrão: 60)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para filtrar a busca (opcional). Se omitido, busca no repositório padrão ou global.",
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
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a travessia de vizinhos (opcional).",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryExportCanvas = Tool{
		Name:        "memory_export_canvas",
		Description: "Exporta um subgrafo centrado em node_id no formato aberto JSON Canvas 1.0 (.canvas) para visualização espacial no Obsidian",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"node_id": map[string]any{
					"type":        "string",
					"description": "Identificador ou nome da nota central do subgrafo",
				},
				"max_depth": map[string]any{
					"type":        "integer",
					"description": "Profundidade máxima de travessia no grafo (padrão: 1)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a busca de conexões (opcional)",
				},
				"output_path": map[string]any{
					"type":        "string",
					"description": "Caminho de arquivo para salvar o .canvas (opcional). Se omitido, retorna a string JSON do Canvas.",
				},
			},
			"required": []string{"node_id"},
		},
	}

	ToolMemoryGetHubs = Tool{
		Name:        "memory_get_hubs",
		Description: "Retorna os nós com maior centralidade de conexões (God Nodes / Hubs de conhecimento) no grafo de notas",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"top": map[string]any{
					"type":        "integer",
					"description": "Número máximo de nós centrais a retornar (padrão: 10)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar a busca de hubs (opcional)",
				},
			},
		},
	}

	ToolMemoryGetInsights = Tool{
		Name:        "memory_get_insights",
		Description: "Retorna conexões latentes e surpreendentes (Surprising Connections) entre notas com alta similaridade sem links diretos no grafo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "integer",
					"description": "Número máximo de conexões latentes a retornar (padrão: 10)",
				},
				"min_similarity": map[string]any{
					"type":        "number",
					"description": "Limiar mínimo de similaridade semântica entre as notas (padrão: 0.70)",
				},
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para filtrar as conexões inesperadas (opcional)",
				},
			},
		},
	}

	ToolMemoryDoctor = Tool{
		Name:        "memory_doctor",
		Description: "Audita a integridade do grafo e tabelas de notas, identificando dead links, notas órfãs, self-loops e calculando o Health Score",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"repository": map[string]any{
					"type":        "string",
					"description": "Slug ou nome do repositório para contextualizar o diagnóstico (opcional)",
				},
				"fix": map[string]any{
					"type":        "boolean",
					"description": "Se verdadeiro, remove automaticamente anomalias conhecidas (self-loops e dead links) (padrão: false)",
				},
			},
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
