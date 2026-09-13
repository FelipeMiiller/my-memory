package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// CallToolParams define os parâmetros recebidos em uma requisição tools/call
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolContent representa um bloco de resposta no padrão MCP
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult estrutura retornada pelo resultado de tools/call
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// NewTextResult cria um CallToolResult com conteúdo textual
func NewTextResult(text string) CallToolResult {
	return CallToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: text,
			},
		},
	}
}

// SearchResult representa um trecho relevante recuperado na busca semântica
type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Repository string   `json:"repository,omitempty"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance"`
	Neighbors  []string `json:"neighbors,omitempty"`
}

// SearchFunc assinatura da função que executa a busca vetorial
type SearchFunc func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error)

// NeighborsFunc assinatura da função que realiza a travessia de vizinhos no grafo
type NeighborsFunc func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error)

// FormatSearchResults formata os resultados da busca vetorial em texto legível
func FormatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "Nenhum resultado encontrado."
	}

	var sb strings.Builder
	for i, res := range results {
		if res.Repository != "" {
			fmt.Fprintf(&sb, "--- [%d] Distância: %.4f | Repositório: %s | Documento: %s ---\n", i+1, res.Distance, res.Repository, res.DocumentID)
		} else {
			fmt.Fprintf(&sb, "--- [%d] Distância: %.4f | Documento: %s ---\n", i+1, res.Distance, res.DocumentID)
		}
		sb.WriteString(res.Content)
		sb.WriteString("\n")
		if len(res.Neighbors) > 0 {
			fmt.Fprintf(&sb, "🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		if i < len(results)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// FormatNeighbors formata a lista de conexões do grafo em texto
func FormatNeighbors(nodeID string, neighbors []string) string {
	if len(neighbors) == 0 {
		return fmt.Sprintf("Nenhum vizinho encontrado para o nó '%s'.", nodeID)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Vizinhos conectados a '%s':\n", nodeID)
	for _, n := range neighbors {
		fmt.Fprintf(&sb, "- %s\n", n)
	}
	return strings.TrimSpace(sb.String())
}

// NewMemorySearchHandler cria o handler para a ferramenta memory_search
func NewMemorySearchHandler(searchFn SearchFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		if len(args) == 0 {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'query'", nil)
		}

		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(args, &rawMap); err != nil {
			return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
		}

		rawQuery, hasQuery := rawMap["query"]
		if !hasQuery {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'query'", nil)
		}

		var query string
		if err := json.Unmarshal(rawQuery, &query); err != nil {
			return nil, NewError(CodeInvalidParams, "Tipo inválido para parâmetro 'query', esperava string", err.Error())
		}

		if strings.TrimSpace(query) == "" {
			return NewTextResult("Nenhum resultado encontrado."), nil
		}

		limit := 5
		if rawLimit, hasLimit := rawMap["limit"]; hasLimit {
			var l int
			if err := json.Unmarshal(rawLimit, &l); err == nil && l > 0 {
				limit = l
			}
		}

		var repo string
		if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		if searchFn == nil {
			return nil, NewError(CodeInternalError, "Backend de busca semântica não configurado", nil)
		}

		results, err := searchFn(ctx, repo, query, limit)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro na execução da busca: %v", err), nil)
		}

		return NewTextResult(FormatSearchResults(results)), nil
	}
}

// NewMemoryNeighborsHandler cria o handler para a ferramenta memory_get_neighbors
func NewMemoryNeighborsHandler(neighborsFn NeighborsFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		if len(args) == 0 {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'node_id'", nil)
		}

		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(args, &rawMap); err != nil {
			return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
		}

		rawNodeID, hasNodeID := rawMap["node_id"]
		if !hasNodeID {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'node_id'", nil)
		}

		var nodeID string
		if err := json.Unmarshal(rawNodeID, &nodeID); err != nil {
			return nil, NewError(CodeInvalidParams, "Tipo inválido para parâmetro 'node_id', esperava string", err.Error())
		}

		if strings.TrimSpace(nodeID) == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'node_id'", nil)
		}

		maxDepth := 1
		if rawDepth, hasDepth := rawMap["max_depth"]; hasDepth {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				maxDepth = d
			}
		}

		var repo string
		if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		if neighborsFn == nil {
			return nil, NewError(CodeInternalError, "Backend de grafo não configurado", nil)
		}

		neighbors, err := neighborsFn(ctx, repo, nodeID, maxDepth)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro na travessia de vizinhos: %v", err), nil)
		}

		return NewTextResult(FormatNeighbors(nodeID, neighbors)), nil
	}
}

// NewDBNeighborsHandler cria uma função de busca de vizinhos consultando o banco SQLite via Recursive CTE
func NewDBNeighborsHandler(database *sql.DB) NeighborsFunc {
	return func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
		cteQuery := `
		WITH RECURSIVE traversal AS (
			SELECT target_id, 1 AS depth
			FROM graph_edges
			WHERE source_id = ?
			
			UNION
			
			SELECT e.target_id, t.depth + 1
			FROM graph_edges e
			JOIN traversal t ON e.source_id = t.target_id
			WHERE t.depth < ?
		)
		SELECT DISTINCT target_id FROM traversal;
		`

		rows, err := database.QueryContext(ctx, cteQuery, nodeID, maxDepth)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var neighbors []string
		for rows.Next() {
			var target string
			if err := rows.Scan(&target); err == nil {
				neighbors = append(neighbors, target)
			}
		}
		return neighbors, nil
	}
}

// SetSearchHandler configura a função de busca semântica para o handler memory_search
func (s *Server) SetSearchHandler(fn SearchFunc) {
	s.RegisterToolHandler("memory_search", NewMemorySearchHandler(fn))
}

// SetNeighborsHandler configura a função de travessia do grafo para o handler memory_get_neighbors
func (s *Server) SetNeighborsHandler(fn NeighborsFunc) {
	s.RegisterToolHandler("memory_get_neighbors", NewMemoryNeighborsHandler(fn))
}

// handleToolsCall despacha a execução da ferramenta indicada no campo 'name'
func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (any, error) {
	if len(params) == 0 {
		return nil, NewError(CodeInvalidParams, "Parâmetros ausentes para tools/call", nil)
	}

	var callParams CallToolParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, NewError(CodeInvalidParams, "Payload inválido para tools/call", err.Error())
	}

	if callParams.Name == "" {
		return nil, NewError(CodeInvalidParams, "Parâmetro 'name' é obrigatório em tools/call", nil)
	}

	s.mu.RLock()
	handler, exists := s.toolHandlers[callParams.Name]
	s.mu.RUnlock()

	if !exists || handler == nil {
		return nil, NewError(CodeMethodNotFound, fmt.Sprintf("Ferramenta '%s' não encontrada", callParams.Name), nil)
	}

	res, err := handler(ctx, callParams.Arguments)
	if err != nil {
		return nil, err
	}

	switch v := res.(type) {
	case CallToolResult:
		return v, nil
	case string:
		return NewTextResult(v), nil
	default:
		return res, nil
	}
}
