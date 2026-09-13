package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/canvas"
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
	Distance   float64  `json:"distance,omitempty"`
	Score      float64  `json:"score,omitempty"`
	Sources    []string `json:"sources,omitempty"`
	Neighbors  []string `json:"neighbors,omitempty"`
}

// SearchFunc assinatura da função que executa a busca vetorial legada
type SearchFunc func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error)

// SearchParams agrupa os parâmetros da busca para flexibilidade de múltiplos modos
type SearchParams struct {
	Repo  string
	Query string
	Mode  string // "hybrid" (default), "vector", "fts"
	Limit int
	K     int // constante RRF, default 60
}

// AdvancedSearchFunc assinatura da função que executa busca avançada suportando modo híbrido e RRF
type AdvancedSearchFunc func(ctx context.Context, params SearchParams) ([]SearchResult, error)

// NeighborsFunc assinatura da função que realiza a travessia de vizinhos no grafo
type NeighborsFunc func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error)

// GodNode representa um nó central com alta centralidade de conexões retornado no MCP
type GodNode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	InDegree    int    `json:"in_degree"`
	OutDegree   int    `json:"out_degree"`
	TotalDegree int    `json:"total_degree"`
}

// HubsFunc assinatura da função que calcula nós centrais (God Nodes / Hubs)
type HubsFunc func(ctx context.Context, repo string, limit int) ([]GodNode, error)

// SurprisingConnection representa uma conexão latente sem link direto no grafo
type SurprisingConnection struct {
	SourceID   string  `json:"source_id"`
	SourceName string  `json:"source_name"`
	TargetID   string  `json:"target_id"`
	TargetName string  `json:"target_name"`
	Similarity float64 `json:"similarity"`
	Reason     string  `json:"reason"`
}

// InsightsFunc assinatura da função que calcula conexões latentes/inesperadas
type InsightsFunc func(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error)

// FormatSearchResults formata os resultados da busca em texto legível
func FormatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "Nenhum resultado encontrado."
	}

	var sb strings.Builder
	for i, res := range results {
		var headerParts []string
		if res.Score > 0 {
			headerParts = append(headerParts, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headerParts = append(headerParts, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		if res.Repository != "" {
			headerParts = append(headerParts, fmt.Sprintf("Repositório: %s", res.Repository))
		}
		headerParts = append(headerParts, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Fprintf(&sb, "--- [%d] %s ---\n", i+1, strings.Join(headerParts, " | "))
		if len(res.Sources) > 0 {
			fmt.Fprintf(&sb, "📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
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
func NewMemorySearchHandler(searchFn any) ToolHandlerFunc {
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

		mode := "hybrid"
		if rawMode, hasMode := rawMap["mode"]; hasMode {
			var m string
			if err := json.Unmarshal(rawMode, &m); err == nil && strings.TrimSpace(m) != "" {
				mode = strings.ToLower(strings.TrimSpace(m))
			}
		}

		k := 60
		if rawK, hasK := rawMap["k"]; hasK {
			var val int
			if err := json.Unmarshal(rawK, &val); err == nil && val > 0 {
				k = val
			}
		}

		var repo string
		if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		if searchFn == nil {
			return nil, NewError(CodeInternalError, "Backend de busca semântica não configurado", nil)
		}

		var results []SearchResult
		var err error

		switch fn := searchFn.(type) {
		case AdvancedSearchFunc:
			results, err = fn(ctx, SearchParams{
				Repo:  repo,
				Query: query,
				Mode:  mode,
				Limit: limit,
				K:     k,
			})
		case SearchFunc:
			results, err = fn(ctx, repo, query, limit)
		case func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error):
			results, err = fn(ctx, repo, query, limit)
		default:
			return nil, NewError(CodeInternalError, "Tipo de função de busca não suportado", nil)
		}

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

// NewMemoryExportCanvasHandler cria o handler para a ferramenta memory_export_canvas
func NewMemoryExportCanvasHandler(neighborsFn NeighborsFunc) ToolHandlerFunc {
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

		var outputPath string
		if rawOut, hasOut := rawMap["output_path"]; hasOut {
			_ = json.Unmarshal(rawOut, &outputPath)
		}

		if neighborsFn == nil {
			return nil, NewError(CodeInternalError, "Backend de grafo não configurado", nil)
		}

		neighbors, err := neighborsFn(ctx, repo, nodeID, maxDepth)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro na travessia de vizinhos: %v", err), nil)
		}

		c := canvas.FromNeighbors(nodeID, neighbors)

		if outputPath != "" {
			if err := c.SaveToFile(outputPath); err != nil {
				return nil, NewError(CodeInternalError, fmt.Sprintf("Erro salvando arquivo canvas: %v", err), nil)
			}
			return NewTextResult(fmt.Sprintf("JSON Canvas salvo com sucesso em '%s' (%d nós, %d arestas).", outputPath, len(c.Nodes), len(c.Edges))), nil
		}

		data, err := c.ToJSON()
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro serializando JSON canvas: %v", err), nil)
		}

		return NewTextResult(string(data)), nil
	}
}

// SetSearchHandler configura a função de busca semântica para o handler memory_search
func (s *Server) SetSearchHandler(fn SearchFunc) {
	s.RegisterToolHandler("memory_search", NewMemorySearchHandler(fn))
}

// SetAdvancedSearchHandler configura a função de busca avançada com suporte a RRF e modos
func (s *Server) SetAdvancedSearchHandler(fn AdvancedSearchFunc) {
	s.RegisterToolHandler("memory_search", NewMemorySearchHandler(fn))
}

// SetNeighborsHandler configura a função de travessia do grafo para memory_get_neighbors e memory_export_canvas
func (s *Server) SetNeighborsHandler(fn NeighborsFunc) {
	s.RegisterToolHandler("memory_get_neighbors", NewMemoryNeighborsHandler(fn))
	s.RegisterToolHandler("memory_export_canvas", NewMemoryExportCanvasHandler(fn))
}

// FormatHubs formata a lista de God Nodes / Hubs de conhecimento em texto legível
func FormatHubs(hubs []GodNode) string {
	if len(hubs) == 0 {
		return "Nenhum nó central (God Node) encontrado no grafo."
	}
	var sb strings.Builder
	sb.WriteString("🌟 Principais Nós Centrais (God Nodes / Hubs de Conhecimento):\n\n")
	for i, h := range hubs {
		displayName := h.Name
		if displayName == "" {
			displayName = h.ID
		}
		fmt.Fprintf(&sb, "[%d] %s (Grau Total: %d | Entrada: %d | Saída: %d)\n",
			i+1, displayName, h.TotalDegree, h.InDegree, h.OutDegree)
		if h.ID != displayName {
			fmt.Fprintf(&sb, "    ID: %s\n", h.ID)
		}
	}
	return sb.String()
}

// NewMemoryGetHubsHandler cria o handler para a ferramenta memory_get_hubs
func NewMemoryGetHubsHandler(hubsFn HubsFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		top := 10
		var repo string

		if len(args) > 0 {
			var rawMap map[string]json.RawMessage
			if err := json.Unmarshal(args, &rawMap); err == nil {
				if rawTop, hasTop := rawMap["top"]; hasTop {
					var t int
					if err := json.Unmarshal(rawTop, &t); err == nil && t > 0 {
						top = t
					}
				}
				if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
					_ = json.Unmarshal(rawRepo, &repo)
				}
			}
		}

		if hubsFn == nil {
			return nil, NewError(CodeInternalError, "Backend de cálculo de hubs não configurado", nil)
		}

		hubs, err := hubsFn(ctx, repo, top)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao buscar nós centrais: %v", err), nil)
		}

		return NewTextResult(FormatHubs(hubs)), nil
	}
}

// SetHubsHandler configura a função de cálculo de hubs para memory_get_hubs
func (s *Server) SetHubsHandler(fn HubsFunc) {
	s.RegisterToolHandler("memory_get_hubs", NewMemoryGetHubsHandler(fn))
}

// FormatInsights formata a lista de conexões inesperadas em texto legível para LLMs
func FormatInsights(connections []SurprisingConnection) string {
	if len(connections) == 0 {
		return "Nenhuma conexão inesperada encontrada com os critérios especificados."
	}
	var sb strings.Builder
	sb.WriteString("💡 Conexões Inesperadas e Pontes Conceituais Latentes:\n\n")
	for i, c := range connections {
		src := c.SourceName
		if src == "" {
			src = c.SourceID
		}
		tgt := c.TargetName
		if tgt == "" {
			tgt = c.TargetID
		}
		fmt.Fprintf(&sb, "[%d] %s <--> %s (Similaridade: %.1f%%)\n",
			i+1, src, tgt, c.Similarity*100)
		if c.Reason != "" {
			fmt.Fprintf(&sb, "    Razão: %s\n", c.Reason)
		}
		if c.SourceID != src || c.TargetID != tgt {
			fmt.Fprintf(&sb, "    IDs: %s <--> %s\n", c.SourceID, c.TargetID)
		}
	}
	return sb.String()
}

// NewMemoryGetInsightsHandler cria o handler para a ferramenta memory_get_insights
func NewMemoryGetInsightsHandler(insightsFn InsightsFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		limit := 10
		minSimilarity := 0.70
		var repo string

		if len(args) > 0 {
			var rawMap map[string]json.RawMessage
			if err := json.Unmarshal(args, &rawMap); err == nil {
				if rawLimit, hasLimit := rawMap["limit"]; hasLimit {
					var l int
					if err := json.Unmarshal(rawLimit, &l); err == nil && l > 0 {
						limit = l
					}
				}
				if rawSim, hasSim := rawMap["min_similarity"]; hasSim {
					var s float64
					if err := json.Unmarshal(rawSim, &s); err == nil && s > 0 {
						minSimilarity = s
					}
				}
				if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
					_ = json.Unmarshal(rawRepo, &repo)
				}
			}
		}

		if insightsFn == nil {
			return nil, NewError(CodeInternalError, "Backend de cálculo de insights não configurado", nil)
		}

		connections, err := insightsFn(ctx, repo, limit, minSimilarity)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao buscar insights conceituais: %v", err), nil)
		}

		return NewTextResult(FormatInsights(connections)), nil
	}
}

// SetInsightsHandler configura a função de cálculo de conexões inesperadas para memory_get_insights
func (s *Server) SetInsightsHandler(fn InsightsFunc) {
	s.RegisterToolHandler("memory_get_insights", NewMemoryGetInsightsHandler(fn))
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
