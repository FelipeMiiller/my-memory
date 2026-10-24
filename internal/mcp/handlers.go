package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/canvas"
	"github.com/FelipeMiiller/my-memory/internal/federation"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
	"github.com/FelipeMiiller/my-memory/internal/store"
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
	// StructuredContent é o payload estruturado da tool, conforme o MCP spec
	// (2025-06-18). Ele é aditivo: as tools que não o preenchem continuam
	// emitindo exatamente o mesmo JSON, porque o campo é omitido quando vazio.
	StructuredContent any `json:"structuredContent,omitempty"`
}

// NewTextResult constrói uma resposta textual compatível com CallToolResult
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

// NewStructuredResult constrói uma resposta com o texto legível e o payload
// estruturado correspondente. O texto continua sendo a saída humana; o
// structuredContent é o que um cliente de máquina (a SPA) valida.
//
// O payload passa por `normalizeStructuredContent`: array vazio viaja como
// `[]`, nunca como `null` — ver o comentário de lá para o porquê.
func NewStructuredResult(text string, structured any) CallToolResult {
	result := NewTextResult(text)
	result.StructuredContent = normalizeStructuredContent(structured)
	return result
}

// SearchResult representa um trecho relevante recuperado na busca semântica
type SearchResult = federation.SearchResult

// SearchFunc assinatura da função que executa a busca vetorial legada
type SearchFunc func(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error)

// SearchParams agrupa os parâmetros da busca para flexibilidade de múltiplos modos
type SearchParams = federation.SearchParams

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

// PageRankNode representa um nó com autoridade calculada via PageRank retornado no MCP
type PageRankNode struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
	InDegree  int     `json:"in_degree"`
	OutDegree int     `json:"out_degree"`
}

// PageRankFunc assinatura da função que calcula autoridade de nós via PageRank
type PageRankFunc func(ctx context.Context, repo string, limit int) ([]PageRankNode, error)

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

// DeadLink representa um link no grafo para uma nota inexistente
type DeadLink struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Relation string `json:"relation"`
}

// OrphanNote representa uma nota isolada com zero conexões
type OrphanNote struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SelfLoop representa uma aresta circular onde a nota aponta para si mesma
type SelfLoop struct {
	NodeID   string `json:"node_id"`
	Relation string `json:"relation"`
}

// DesyncedChunk representa um chunk sem correspondência vetorial
type DesyncedChunk struct {
	ChunkID    string `json:"chunk_id"`
	DocumentID string `json:"document_id"`
	Issue      string `json:"issue"`
}

// DoctorReport agrega as métricas estruturais e anomalias de saúde do grafo
//
// Espelho parcial de store.DoctorReport: espelha o que o agente precisa para
// decidir se reindexa, e não os avisos de ortografia, que são detalhe de
// tokenização e não de integridade do grafo.
type DoctorReport struct {
	TotalDocuments int `json:"total_documents"`
	TotalChunks    int `json:"total_chunks"`
	TotalEdges     int `json:"total_edges"`
	TotalNodes     int `json:"total_nodes"`
	HealthScore    int `json:"health_score"` // 0 a 100
	// ADR-057: documentos com mtime = 0 foram indexados antes de a coluna
	// existir. Sem esta contagem o agente não tem como saber que o índice
	// conhece a nota mas não sabe quando ela foi editada.
	MissingMTimeCount int             `json:"missing_mtime_count"`
	DeadLinks         []DeadLink      `json:"dead_links"`
	OrphanNotes       []OrphanNote    `json:"orphan_notes"`
	SelfLoops         []SelfLoop      `json:"self_loops"`
	DesyncedChunks    []DesyncedChunk `json:"desynced_chunks"`
}

// DoctorDiagnoseFunc assinatura da função que diagnostica a saúde do grafo
type DoctorDiagnoseFunc func(ctx context.Context, repo string) (*DoctorReport, error)

// DoctorFixFunc assinatura da função que repara anomalias conhecidas
type DoctorFixFunc func(ctx context.Context, repo string) (int, error)

// FormatSearchResultsWithOptions formata os resultados da busca com base no nível de densidade
func FormatSearchResultsWithOptions(results []SearchResult, level string) string {
	if len(results) == 0 {
		return "Nenhum resultado encontrado."
	}

	if strings.EqualFold(level, "l0") {
		var sb strings.Builder
		sb.WriteString("### 🎯 Resultados da Busca (L0: Micro-Abstracts)\n\n")
		sb.WriteString("| # | Categoria | Documento | Score | Micro-Abstract (L0) | Conexões no Grafo |\n")
		sb.WriteString("| :- | :--- | :--- | :--- | :--- | :--- |\n")
		for i, res := range results {
			cat := "resource"
			if res.Category != "" {
				cat = strings.ToLower(res.Category)
			}
			scoreStr := "-"
			if res.Score > 0 {
				scoreStr = fmt.Sprintf("%.4f", res.Score)
			} else if res.Distance > 0 {
				scoreStr = fmt.Sprintf("dist: %.4f", res.Distance)
			}
			abstract := res.Abstract
			if abstract == "" {
				abstract = "-"
			}
			abstract = strings.ReplaceAll(abstract, "|", "/")
			neighbors := "-"
			if len(res.Neighbors) > 0 {
				neighbors = fmt.Sprintf("`%s`", strings.Join(res.Neighbors, ", "))
			}
			fmt.Fprintf(&sb, "| %d | `%s` | `%s` | %s | %s | %s |\n", i+1, cat, res.DocumentID, scoreStr, abstract, neighbors)
		}
		return strings.TrimSpace(sb.String())
	}

	var sb strings.Builder
	for i, res := range results {
		var headerParts []string
		if res.Category != "" {
			headerParts = append(headerParts, fmt.Sprintf("[%s]", strings.ToUpper(res.Category)))
		} else {
			headerParts = append(headerParts, "[RESOURCE]")
		}
		if res.Score > 0 {
			headerParts = append(headerParts, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headerParts = append(headerParts, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		if res.Repository != "" {
			headerParts = append(headerParts, fmt.Sprintf("Repositório: %s", res.Repository))
		}
		if res.UpdatedAt > 0 {
			headerParts = append(headerParts, fmt.Sprintf("Atualizado em: %s", time.Unix(res.UpdatedAt, 0).UTC().Format("2006-01-02 15:04:05")))
		}
		headerParts = append(headerParts, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Fprintf(&sb, "--- [%d] %s ---\n", i+1, strings.Join(headerParts, " | "))
		if res.Abstract != "" {
			fmt.Fprintf(&sb, "💡 **Resumo (L0)**: %s\n", res.Abstract)
		}
		if len(res.Sources) > 0 {
			fmt.Fprintf(&sb, "📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
		}
		if res.Content != "" {
			sb.WriteString(res.Content)
			sb.WriteString("\n")
		}
		if len(res.Neighbors) > 0 {
			fmt.Fprintf(&sb, "🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		if i < len(results)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// FormatSearchResults formata os resultados da busca em texto legível (retrocompatibilidade)
func FormatSearchResults(results []SearchResult) string {
	return FormatSearchResultsWithOptions(results, "l1")
}

// FormatNeighbors formata a lista de conexões do grafo em texto
func FormatNeighbors(nodeID string, neighbors []string) string {
	if len(neighbors) == 0 {
		return fmt.Sprintf("Nenhum vizinho encontrado para o nó '%s'.", nodeID)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Vizinhos conectados a '%s':\n", nodeID)
	for _, n := range neighbors {
		if strings.HasPrefix(n, "memory://") {
			fmt.Fprintf(&sb, "- %s (is_federated: true, canonical_uri: %s)\n", n, n)
		} else {
			fmt.Fprintf(&sb, "- %s\n", n)
		}
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

		decay := false
		if rawDecay, hasDecay := rawMap["decay"]; hasDecay {
			var d bool
			if err := json.Unmarshal(rawDecay, &d); err == nil {
				decay = d
			}
		}

		halfLife := 30.0
		if rawHalfLife, hasHalfLife := rawMap["half_life"]; hasHalfLife {
			var hl float64
			if err := json.Unmarshal(rawHalfLife, &hl); err == nil && hl > 0 {
				halfLife = hl
			}
		}

		decayWeight := 0.3
		if rawDecayWeight, hasDecayWeight := rawMap["decay_weight"]; hasDecayWeight {
			var dw float64
			if err := json.Unmarshal(rawDecayWeight, &dw); err == nil && dw >= 0.0 && dw <= 1.0 {
				decayWeight = dw
			}
		}

		var repo string
		if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
			_ = json.Unmarshal(rawRepo, &repo)
		}

		detailLevel := "l1"
		if rawLevel, hasLevel := rawMap["detail_level"]; hasLevel {
			var dl string
			if err := json.Unmarshal(rawLevel, &dl); err == nil && strings.TrimSpace(dl) != "" {
				detailLevel = strings.ToLower(strings.TrimSpace(dl))
			}
		}
		if detailLevel != "l0" && detailLevel != "l1" && detailLevel != "l2" {
			detailLevel = "l1"
		}

		var category string
		if rawCategory, hasCategory := rawMap["category"]; hasCategory {
			var c string
			if err := json.Unmarshal(rawCategory, &c); err == nil && strings.TrimSpace(c) != "" {
				category = strings.ToLower(strings.TrimSpace(c))
			}
		}

		// O filtro por tag (paridade com `obsidian search tag:X`): a tag
		// canônica viaja para o push-down, que casa a tag e as descendentes.
		// Tag fora do formato do Obsidian é erro de parâmetro — um filtro que
		// não casa nada seria lido pelo agente como "o vault não tem a tag".
		var tag string
		if rawTag, hasTag := rawMap["tag"]; hasTag {
			var t string
			if err := json.Unmarshal(rawTag, &t); err == nil && strings.TrimSpace(t) != "" {
				canonical, ok := parser.NormalizeTag(t)
				if !ok {
					return nil, NewError(CodeInvalidParams, fmt.Sprintf("Tag inválida: '%s'", strings.TrimSpace(t)), nil)
				}
				tag = canonical
			}
		}

		// CG-19/T14: include_code é flag aditiva (default false). Ligada, ela
		// coloca code_symbols no RRF como fonte real de fan-in (a fonte
		// "code"); desligada, o ranking atual é preservado byte-a-byte.
		includeCode := false
		if rawIncludeCode, hasIncludeCode := rawMap["include_code"]; hasIncludeCode {
			_ = json.Unmarshal(rawIncludeCode, &includeCode)
		}
		if includeCode {
			// A flag viaja no contexto porque as implementações de busca
			// (SQLite e Postgres) são ligadas no CLI e recebem o ctx — é o
			// mesmo canal usado pelo repo_id (ADR-061).
			ctx = store.WithIncludeCode(ctx, true)
		}

		if searchFn == nil {
			return nil, NewError(CodeInternalError, "Backend de busca semântica não configurado", nil)
		}

		var results []SearchResult
		var err error

		switch fn := searchFn.(type) {
		case AdvancedSearchFunc:
			results, err = fn(ctx, SearchParams{
				Repo:        repo,
				Query:       query,
				Mode:        mode,
				Limit:       limit,
				K:           k,
				Decay:       decay,
				HalfLife:    halfLife,
				DecayWeight: decayWeight,
				DetailLevel: detailLevel,
				Category:    category,
				Tag:         tag,
				IncludeCode: includeCode,
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

		return NewTextResult(FormatSearchResultsWithOptions(results, detailLevel)), nil
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

		var format string
		if rawFmt, hasFmt := rawMap["format"]; hasFmt {
			_ = json.Unmarshal(rawFmt, &format)
		}

		if strings.ToLower(strings.TrimSpace(format)) == "json" {
			type NeighborEdge struct {
				TargetID     string `json:"target_id"`
				IsFederated  bool   `json:"is_federated"`
				CanonicalURI string `json:"canonical_uri,omitempty"`
			}
			items := make([]NeighborEdge, len(neighbors))
			for i, n := range neighbors {
				isFed := strings.HasPrefix(n, "memory://")
				items[i] = NeighborEdge{
					TargetID:    n,
					IsFederated: isFed,
				}
				if isFed {
					items[i].CanonicalURI = n
				}
			}
			jsonBytes, _ := json.MarshalIndent(items, "", "  ")
			return NewTextResult(string(jsonBytes)), nil
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

// FormatPageRankHubs formata a lista de nós centrais calculados via PageRank em texto legível para LLMs
func FormatPageRankHubs(nodes []PageRankNode) string {
	if len(nodes) == 0 {
		return "Nenhum nó encontrado para o cálculo de PageRank no grafo."
	}
	var sb strings.Builder
	sb.WriteString("🌐 Principais Nós por Autoridade PageRank (Top Hubs):\n\n")
	for i, n := range nodes {
		displayName := n.Name
		if displayName == "" {
			displayName = n.ID
		}
		percentage := n.Score * 100.0
		fmt.Fprintf(&sb, "[%d] %s — Score: %.4f (%.2f%%) | Entrada: %d | Saída: %d\n",
			i+1, displayName, n.Score, percentage, n.InDegree, n.OutDegree)
		if n.ID != displayName {
			fmt.Fprintf(&sb, "    ID: %s\n", n.ID)
		}
	}
	return sb.String()
}

// NewAdvancedMemoryGetHubsHandler cria o handler para a ferramenta memory_get_hubs com suporte a grau ou PageRank
func NewAdvancedMemoryGetHubsHandler(hubsFn HubsFunc, prFn PageRankFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		top := 10
		algorithm := "degree"
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
				if rawAlgo, hasAlgo := rawMap["algorithm"]; hasAlgo {
					var algoStr string
					if err := json.Unmarshal(rawAlgo, &algoStr); err == nil && algoStr != "" {
						algorithm = strings.ToLower(strings.TrimSpace(algoStr))
					}
				}
			}
		}

		if algorithm == "pagerank" {
			if prFn == nil {
				return nil, NewError(CodeInternalError, "Backend de cálculo de PageRank não configurado", nil)
			}
			nodes, err := prFn(ctx, repo, top)
			if err != nil {
				return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao calcular PageRank dos nós: %v", err), nil)
			}
			// structuredContent (algorithm=pagerank): [{id: string, name: string, score: number, rank: number, in_degree: number, out_degree: number}]
			return NewStructuredResult(FormatPageRankHubs(nodes), nodes), nil
		}

		if hubsFn == nil {
			return nil, NewError(CodeInternalError, "Backend de cálculo de hubs não configurado", nil)
		}

		hubs, err := hubsFn(ctx, repo, top)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao buscar nós centrais: %v", err), nil)
		}

		// structuredContent (algorithm=degree): [{id: string, name: string, in_degree: number, out_degree: number, total_degree: number}]
		return NewStructuredResult(FormatHubs(hubs), hubs), nil
	}
}

// NewMemoryGetHubsHandler cria o handler para a ferramenta memory_get_hubs
func NewMemoryGetHubsHandler(hubsFn HubsFunc) ToolHandlerFunc {
	return NewAdvancedMemoryGetHubsHandler(hubsFn, nil)
}

// SetHubsHandler configura a função de cálculo de hubs padrão para memory_get_hubs
func (s *Server) SetHubsHandler(fn HubsFunc) {
	s.SetAdvancedHubsHandler(fn, nil)
}

// SetAdvancedHubsHandler configura as funções de cálculo de hubs por grau e por PageRank
func (s *Server) SetAdvancedHubsHandler(hubsFn HubsFunc, prFn PageRankFunc) {
	s.RegisterToolHandler("memory_get_hubs", NewAdvancedMemoryGetHubsHandler(hubsFn, prFn))
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

		// structuredContent: [{source_id: string, source_name: string, target_id: string, target_name: string, similarity: number, reason: string}]
		return NewStructuredResult(FormatInsights(connections), connections), nil
	}
}

// SetInsightsHandler configura a função de cálculo de conexões inesperadas para memory_get_insights
func (s *Server) SetInsightsHandler(fn InsightsFunc) {
	s.RegisterToolHandler("memory_get_insights", NewMemoryGetInsightsHandler(fn))
}

// FormatDoctorReport formata o relatório de diagnóstico em Markdown rico para LLMs
func FormatDoctorReport(report *DoctorReport, fixedCount int) string {
	if report == nil {
		return "Nenhum relatório de diagnóstico disponível."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# 🩺 Diagnóstico de Memória & Grafo (Health Score: %d/100)\n\n", report.HealthScore))
	sb.WriteString(fmt.Sprintf("- **Documentos**: %d | **Chunks**: %d | **Arestas**: %d | **Nós**: %d\n",
		report.TotalDocuments, report.TotalChunks, report.TotalEdges, report.TotalNodes))

	// ADR-057: só aparece quando há contagem, para não poluir o relatório de um
	// vault saudável com uma seção que sempre diria zero.
	if report.MissingMTimeCount > 0 {
		sb.WriteString(fmt.Sprintf("- ⏳ **Documentos sem mtime** (indexados antes do metadado existir): %d — rode 'mem index --force' para capturar\n",
			report.MissingMTimeCount))
	}

	if fixedCount > 0 {
		sb.WriteString(fmt.Sprintf("- 🛠 **Anomalias Reparadas**: %d arestas problemáticas removidas\n", fixedCount))
	}
	sb.WriteString("\n")

	if len(report.DeadLinks) > 0 {
		sb.WriteString(fmt.Sprintf("## 🔗 Links Quebrados (Dead Links - %d)\n", len(report.DeadLinks)))
		for _, dl := range report.DeadLinks {
			sb.WriteString(fmt.Sprintf("- `[[%s]]` -> `[[%s]]` (relação: `%s`)\n", dl.SourceID, dl.TargetID, dl.Relation))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("## 🔗 Links Quebrados\n- Nenhum dead link detectado. Todas as conexões apontam para notas existentes.\n\n")
	}

	if len(report.OrphanNotes) > 0 {
		sb.WriteString(fmt.Sprintf("## 🏝️ Notas Órfãs (Sem Conexões - %d)\n", len(report.OrphanNotes)))
		for _, on := range report.OrphanNotes {
			displayName := on.Title
			if displayName == "" {
				displayName = on.ID
			}
			sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", displayName, on.ID))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("## 🏝️ Notas Órfãs\n- Nenhuma nota órfã detectada. Todo o conhecimento está interligado no grafo.\n\n")
	}

	if len(report.SelfLoops) > 0 {
		sb.WriteString(fmt.Sprintf("## 🔄 Loops Reflexivos (Self-Loops - %d)\n", len(report.SelfLoops)))
		for _, sl := range report.SelfLoops {
			sb.WriteString(fmt.Sprintf("- `%s` (relação: `%s`)\n", sl.NodeID, sl.Relation))
		}
		sb.WriteString("\n")
	}

	if len(report.DesyncedChunks) > 0 {
		sb.WriteString(fmt.Sprintf("## ⚡ Chunks Dessincronizados (%d)\n", len(report.DesyncedChunks)))
		for _, dc := range report.DesyncedChunks {
			sb.WriteString(fmt.Sprintf("- Chunk `%s`: %s\n", dc.ChunkID, dc.Issue))
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

// NewMemoryDoctorHandler cria o executor da ferramenta memory_doctor
func NewMemoryDoctorHandler(diagnoseFn DoctorDiagnoseFunc, fixFn DoctorFixFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var repo string
		fix := false

		if len(args) > 0 {
			var rawMap map[string]json.RawMessage
			if err := json.Unmarshal(args, &rawMap); err == nil {
				if rawRepo, hasRepo := rawMap["repository"]; hasRepo {
					_ = json.Unmarshal(rawRepo, &repo)
				}
				if rawFix, hasFix := rawMap["fix"]; hasFix {
					_ = json.Unmarshal(rawFix, &fix)
				}
			}
		}

		fixedCount := 0
		if fix && fixFn != nil {
			n, err := fixFn(ctx, repo)
			if err != nil {
				return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao reparar anomalias: %v", err), nil)
			}
			fixedCount = n
		}

		if diagnoseFn == nil {
			return nil, NewError(CodeInternalError, "Backend de diagnóstico não configurado", nil)
		}

		report, err := diagnoseFn(ctx, repo)
		if err != nil {
			return nil, NewError(CodeInternalError, fmt.Sprintf("Erro ao auditar integridade da memória: %v", err), nil)
		}

		// structuredContent: {total_documents: number, total_chunks: number, total_edges: number, total_nodes: number, health_score: number, missing_mtime_count: number, dead_links: [{source_id: string, target_id: string, relation: string}], orphan_notes: [{id: string, title: string}], self_loops: [{node_id: string, relation: string}], desynced_chunks: [{chunk_id: string, document_id: string, issue: string}]}
		if report == nil {
			report = &DoctorReport{}
		}

		return NewStructuredResult(FormatDoctorReport(report, fixedCount), report), nil
	}
}

// SetDoctorHandler configura as funções de diagnóstico e reparo para memory_doctor
func (s *Server) SetDoctorHandler(diagnoseFn DoctorDiagnoseFunc, fixFn DoctorFixFunc) {
	s.RegisterToolHandler("memory_doctor", NewMemoryDoctorHandler(diagnoseFn, fixFn))
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

	// Invalidação reativa do detector quando novas notas ou seções forem escritas
	if callParams.Name == ToolMemoryWriteNote.Name || callParams.Name == ToolMemoryAppendSection.Name {
		s.mu.RLock()
		det := s.stalenessDetector
		s.mu.RUnlock()
		if det != nil {
			det.ResetCache()
		}
	}

	// Injeção não-bloqueante de Staleness Banner em ferramentas de leitura/consulta
	isQueryTool := func(name string) bool {
		switch name {
		case ToolMemorySearch.Name,
			ToolMemoryGetNeighbors.Name,
			ToolMemoryExportCanvas.Name,
			ToolMemoryGetHubs.Name,
			ToolMemoryGetInsights.Name,
			ToolMemoryDoctor.Name,
			ToolMemoryVisualizeGraph.Name,
			ToolMemoryGetClusters.Name,
			ToolMemoryGetImpact.Name,
			ToolMemoryInspectNode.Name,
			ToolMemoryFindPath.Name,
			ToolMemoryPackContext.Name,
			ToolMemoryOpenNode.Name:
			return true

		default:
			return false
		}
	}

	var banner string
	s.mu.RLock()
	det := s.stalenessDetector
	s.mu.RUnlock()
	if det != nil && isQueryTool(callParams.Name) {
		if rep, checkErr := det.CheckStaleness(ctx); checkErr == nil && rep != nil && rep.IsStale {
			banner = staleness.FormatMarkdownBanner(rep)
		}
	}

	switch v := res.(type) {
	case CallToolResult:
		if banner != "" {
			if len(v.Content) > 0 && v.Content[0].Type == "text" {
				v.Content[0].Text = banner + v.Content[0].Text
			} else {
				v.Content = append([]ToolContent{{Type: "text", Text: banner}}, v.Content...)
			}
		}
		return v, nil
	case string:
		if banner != "" {
			return NewTextResult(banner + v), nil
		}
		return NewTextResult(v), nil
	default:
		return res, nil
	}
}
