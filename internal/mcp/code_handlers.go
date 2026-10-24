package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// CodeSearchFunc é a assinatura da função que executa busca em code_symbols
// com suporte a filtros language/kind e boost CA-05.
//
// O MCP server aceita uma função com esta assinatura via
// SetCodeSearchHandler. Quando nil, o handler devolve uma resposta
// textual amigável indicando que o code pipeline não está habilitado
// (CA-14 — comportamento idêntico a outras tools opcionais).
type CodeSearchFunc func(ctx context.Context, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error)

// CodeNeighborsFunc é a assinatura da função que retorna o sub-grafo
// de um code_symbol via CTE recursivo (ADR-004).
//
// O retorno é a resposta COMPLETA da travessia: vizinhos, raiz e os
// metadados de truncamento (hub threshold, orçamento de tokens, dica para
// estreitar a query — CG-21/CG-22).
type CodeNeighborsFunc func(ctx context.Context, symbol string, depth int, direction string, maxTokens int) (*MemoryCodeNeighborsResponse, error)

// CodeSymbolResult é o payload JSON de cada hit de memory_code_search.
type CodeSymbolResult struct {
	SymbolID      int64   `json:"symbol_id"`
	QualifiedName string  `json:"qualified_name"`
	Name          string  `json:"name"`
	Kind          string  `json:"kind"`
	Language      string  `json:"language"`
	File          string  `json:"file"`
	StartLine     int     `json:"start_line"`
	EndLine       int     `json:"end_line"`
	Signature     string  `json:"signature,omitempty"`
	DocComment    string  `json:"doc_comment,omitempty"`
	Score         float64 `json:"score"`
	Boosted       bool    `json:"boosted"`
}

// CodeNeighborResult é cada vizinho devolvido por memory_code_neighbors.
//
// Os campos de provenance (epistemic_status, confidence) e de call-site
// (call_site_file/call_site_line) são ADITIVOS: as tools existentes não
// quebram. `file`/`line` continuam sendo o arquivo e a linha da ARESTA
// (para uma aresta `calls`, o ponto de chamada); `dst_file`/`dst_line` são a
// posição do SÍMBOLO de destino.
type CodeNeighborResult struct {
	Direction string `json:"direction"`
	Kind      string `json:"kind"`
	Symbol    string `json:"symbol"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Hop       int    `json:"hop"`
	// EpistemicStatus é a procedência da aresta: EXTRACTED (extraída da
	// estrutura do código) ou INFERRED (inferida por nome).
	EpistemicStatus string `json:"epistemic_status,omitempty"`
	// Confidence é a confiança da aresta (1.0 = extraída por escopo de
	// pacote; menor = inferida por nome).
	Confidence float64 `json:"confidence,omitempty"`
	// CallSiteFile/CallSiteLine são o ponto de chamada da relação — a
	// posição da EXPRESSÃO que criou a aresta (a linha/coluna onde a
	// chamada acontece), que é diferente da linha da definição.
	CallSiteFile string `json:"call_site_file,omitempty"`
	CallSiteLine int    `json:"call_site_line,omitempty"`
	// DstFile/DstLine são a posição do símbolo de DESTINO da aresta.
	DstFile string `json:"dst_file,omitempty"`
	DstLine int    `json:"dst_line,omitempty"`
	// DocumentID/DocumentPath só existem nos vizinhos que são DOCUMENTOS
	// (a ponte notas↔código, relação `references_code` — CG-18).
	DocumentID   string `json:"document_id,omitempty"`
	DocumentPath string `json:"document_path,omitempty"`
}

// MemoryCodeSearchResponse envelopa os resultados para a tool MCP.
type MemoryCodeSearchResponse struct {
	Query   string             `json:"query"`
	Count   int                `json:"count"`
	Results []CodeSymbolResult `json:"results"`
	// Truncated sinaliza que a lista foi cortada pelo orçamento de tokens.
	Truncated bool `json:"truncated"`
	// Hint é a dica para estreitar a query quando a resposta foi truncada.
	Hint string `json:"hint,omitempty"`
	// TokensEstimated é o custo estimado da resposta em tokens.
	TokensEstimated int `json:"tokens_estimated"`
}

// MemoryCodeNeighborsResponse envelopa root + neighbors.
type MemoryCodeNeighborsResponse struct {
	Root      *CodeSymbolResult    `json:"root,omitempty"`
	Neighbors []CodeNeighborResult `json:"neighbors"`
	Depth     int                  `json:"depth"`
	Direction string               `json:"direction"`
	// Truncated sinaliza que o subgrafo devolvido foi cortado — pelo hub
	// threshold, pelo orçamento de tokens, ou pelos dois.
	Truncated bool `json:"truncated"`
	// Hint é a dica para estreitar a query (AC 23).
	Hint string `json:"hint,omitempty"`
	// TokensEstimated é o custo estimado da resposta em tokens.
	TokensEstimated int `json:"tokens_estimated"`
	// HubThreshold é o grau de corte aplicado (0 = nenhum corte).
	HubThreshold int `json:"hub_threshold"`
	// NodesPruned é o número de nós removidos pelo hub threshold.
	NodesPruned int `json:"nodes_pruned"`
}

const (
	// DefaultHubThresholdPercentile é o PERCENTIL de grau que define o que
	// é um nó-hub na travessia de código (CG-22).
	//
	// O default é p99 — o mesmo do Graphify (`graphify/analyze.py`,
	// `god_nodes(exclude_hubs_percentile=...)`): o nó-hub é o que está
	// ACIMA do percentil 99 da distribuição de grau do subgrafo. Com p99,
	// ~1% dos nós mais conectados é removido: o suficiente para o hub não
	// dominar o subgrafo devolvido, sem perder vizinho legítimo.
	//
	// Configurável pela variável de ambiente
	// `MY_MEMORY_CODE_HUB_PERCENTILE` (0 < valor <= 100).
	DefaultHubThresholdPercentile = 99.0

	// HubThresholdPercentileEnv é a variável de ambiente que sobrescreve o
	// percentil do hub threshold.
	HubThresholdPercentileEnv = "MY_MEMORY_CODE_HUB_PERCENTILE"

	// DefaultCodeTokenBudget é o orçamento de tokens default da resposta
	// das tools de código (ADR-029: o estimador é o mesmo do
	// memory_pack_context).
	DefaultCodeTokenBudget = 4000

	// MinHubPruneNodes é o número MÍNIMO de nós distintos no subgrafo para
	// que o corte por hub (percentil do grau) seja aplicado.
	//
	// Motivo (defeito D4): o p99 é um percentil de GRAFO GRANDE. Sobre
	// poucos nós, o índice do percentil cai no penúltimo grau da
	// distribuição e o corte remove o vizinho de hop 1 (o `calls` mais
	// próximo da semente) em vez do hub — invertendo a relevância. O número
	// 50 é o piso: abaixo dele a travessia devolve o subgrafo inteiro.
	//
	// Acima de 50 nós, o corte volta a valer (o p99 passa a distinguir o
	// topo da distribuição do resto).
	MinHubPruneNodes = 50
)

// NewMemoryCodeSearchHandler constrói o handler MCP para memory_code_search.
//
// Quando codeFn == nil, devolve CallToolResult com texto explicativo
// (não erro fatal) para que clientes MCP não quebrem em ambientes onde
// o code pipeline está desabilitado (CA-14).
func NewMemoryCodeSearchHandler(codeFn CodeSearchFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		rawQuery, hasQuery := rawMap["query"]
		if !hasQuery {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'query'", nil)
		}
		var query string
		if err := json.Unmarshal(rawQuery, &query); err != nil {
			return nil, NewError(CodeInvalidParams, "Tipo inválido para 'query', esperava string", err.Error())
		}
		query = strings.TrimSpace(query)
		if query == "" {
			return NewTextResult("Nenhum resultado encontrado."), nil
		}

		limit := 10
		if rawLimit, ok := rawMap["limit"]; ok {
			var l int
			if err := json.Unmarshal(rawLimit, &l); err == nil && l > 0 {
				limit = l
			}
		}

		var language string
		if rawLang, ok := rawMap["language"]; ok {
			_ = json.Unmarshal(rawLang, &language)
		}

		var kind string
		if rawKind, ok := rawMap["kind"]; ok {
			_ = json.Unmarshal(rawKind, &kind)
		}

		noBoost := false
		if rawNB, ok := rawMap["no_code_boost"]; ok {
			_ = json.Unmarshal(rawNB, &noBoost)
		}

		if codeFn == nil {
			return NewTextResult("memory_code_search indisponível: code pipeline não configurado (CGO/tree-sitter desabilitado — CA-14)."), nil
		}

		hits, err := codeFn(ctx, query, strings.TrimSpace(language), strings.TrimSpace(kind), limit, !noBoost)
		if err != nil {
			return nil, NewError(CodeInternalError, "Falha em memory_code_search", err.Error())
		}

		maxTokens := codeTokenBudget(rawMap)

		// Orçamento de tokens (AC 23): a lista de hits é cortada quando o
		// custo estimado passa do orçamento, e o corte vem com a dica de
		// estreitar a query.
		kept, tokens, truncated := truncateCodeResults(hits, maxTokens)

		resp := MemoryCodeSearchResponse{
			Query:           query,
			Count:           len(kept),
			Results:         kept,
			Truncated:       truncated,
			TokensEstimated: tokens,
		}
		if truncated {
			resp.Hint = codeNarrowHint(maxTokens, len(hits)-len(kept), 0)
		}
		out, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		// structuredContent (aditivo): o texto continua sendo a saída
		// humana; o payload estruturado é o que a viewer consome.
		return NewStructuredResult(string(out), resp), nil
	}
}

// NewMemoryCodeNeighborsHandler constrói o handler MCP para memory_code_neighbors.
func NewMemoryCodeNeighborsHandler(codeFn CodeNeighborsFunc) ToolHandlerFunc {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var rawMap map[string]json.RawMessage
		if len(args) > 0 {
			if err := json.Unmarshal(args, &rawMap); err != nil {
				return nil, NewError(CodeInvalidParams, "Formato JSON de argumentos inválido", err.Error())
			}
		}

		rawSymbol, hasSymbol := rawMap["symbol"]
		if !hasSymbol {
			return nil, NewError(CodeInvalidParams, "Parâmetro obrigatório ausente: 'symbol'", nil)
		}
		var symbol string
		if err := json.Unmarshal(rawSymbol, &symbol); err != nil {
			return nil, NewError(CodeInvalidParams, "Tipo inválido para 'symbol', esperava string", err.Error())
		}
		symbol = strings.TrimSpace(symbol)
		if symbol == "" {
			return nil, NewError(CodeInvalidParams, "Parâmetro 'symbol' vazio", nil)
		}

		depth := 1
		if rawDepth, ok := rawMap["depth"]; ok {
			var d int
			if err := json.Unmarshal(rawDepth, &d); err == nil && d > 0 {
				depth = d
			}
		}
		if depth > 5 {
			depth = 5
		}

		direction := "both"
		if rawDir, ok := rawMap["direction"]; ok {
			_ = json.Unmarshal(rawDir, &direction)
			direction = strings.ToLower(strings.TrimSpace(direction))
			if direction == "" {
				direction = "both"
			}
		}

		if codeFn == nil {
			return NewTextResult("memory_code_neighbors indisponível: code pipeline não configurado (CGO/tree-sitter desabilitado — CA-14)."), nil
		}

		maxTokens := codeTokenBudget(rawMap)

		resp, err := codeFn(ctx, symbol, depth, direction, maxTokens)
		if err != nil {
			return nil, NewError(CodeInternalError, "Falha em memory_code_neighbors", err.Error())
		}
		if resp == nil {
			resp = &MemoryCodeNeighborsResponse{Neighbors: []CodeNeighborResult{}, Depth: depth, Direction: direction}
		}
		if resp.Neighbors == nil {
			resp.Neighbors = []CodeNeighborResult{}
		}

		out, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		// structuredContent (aditivo): a rota /code da viewer consome o
		// payload estruturado; o texto continua sendo a saída humana.
		return NewStructuredResult(string(out), resp), nil
	}
}

// SetCodeSearchHandler registra o handler de code_search no servidor MCP.
func (s *Server) SetCodeSearchHandler(fn CodeSearchFunc) {
	s.RegisterTool(ToolMemoryCodeSearch, NewMemoryCodeSearchHandler(fn))
}

// SetCodeNeighborsHandler registra o handler de code_neighbors no servidor MCP.
func (s *Server) SetCodeNeighborsHandler(fn CodeNeighborsFunc) {
	s.RegisterTool(ToolMemoryCodeNeighbors, NewMemoryCodeNeighborsHandler(fn))
}

// --- Helpers expostos para o cliente embutir (in-process) -------------------

// SQLCodeSearch é a implementação padrão em SQL puro de CodeSearchFunc
// usando LIKE matching nos campos name/qualified_name/signature + boost CA-05.
//
// Esta função é exportada para permitir que outros pontos do código
// (ex: mem code-search CLI, testes MCP) reutilizem a mesma lógica.
//
// Sem filtro de repositório: use SQLCodeSearchScoped para isolar o vault.
func SQLCodeSearch(ctx context.Context, database *sql.DB, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
	return SQLCodeSearchScoped(ctx, database, query, language, kind, limit, includeBoost, "")
}

// SQLCodeSearchScoped é o SQLCodeSearch com filtro de repositório (CG-07).
//
// repository vazio = sem filtro (comportamento legado). Com repository
// preenchido, só as linhas do mesmo repo_id entram no resultado.
func SQLCodeSearchScoped(ctx context.Context, database *sql.DB, query, language, kind string, limit int, includeBoost bool, repository string) ([]CodeSymbolResult, error) {
	if database == nil {
		return nil, errors.New("SQLCodeSearch: database == nil")
	}
	if limit <= 0 {
		limit = 10
	}

	tokens := mcpTokenize(query)
	var where []string
	var args []any
	for _, tok := range tokens {
		like := "%" + tok + "%"
		where = append(where, "(s.name LIKE ? OR s.qualified_name LIKE ? OR s.signature LIKE ?)")
		args = append(args, like, like, like)
	}
	if strings.TrimSpace(language) != "" {
		where = append(where, "f.language = ?")
		args = append(args, strings.TrimSpace(language))
	}
	if strings.TrimSpace(kind) != "" {
		where = append(where, "s.kind = ?")
		args = append(args, strings.TrimSpace(kind))
	}
	if repo := strings.TrimSpace(repository); repo != "" {
		where = append(where, "f.repository = ? AND s.repository = ?")
		args = append(args, repo, repo)
	}

	if len(where) == 0 {
		where = append(where, "1=1")
	}
	args = append(args, limit)

	sqlQuery := `
SELECT s.id, s.kind, s.name, s.qualified_name, s.signature, s.doc_comment,
       s.start_line, s.end_line, f.path, f.language, 1.0 AS score
FROM code_symbols s
JOIN code_files f ON f.id = s.file_id
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY score DESC, s.id ASC
LIMIT ?`

	rows, err := database.QueryContext(ctx, bindCodeQuery(ctx, database, sqlQuery), args...)
	if err != nil {
		return nil, fmt.Errorf("SQLCodeSearch: %w", err)
	}
	defer rows.Close()

	var hits []CodeSymbolResult
	for rows.Next() {
		var (
			h   CodeSymbolResult
			qn  sql.NullString
			sig sql.NullString
			doc sql.NullString
		)
		if err := rows.Scan(&h.SymbolID, &h.Kind, &h.Name, &qn, &sig, &doc,
			&h.StartLine, &h.EndLine, &h.File, &h.Language, &h.Score); err != nil {
			return nil, err
		}
		if qn.Valid {
			h.QualifiedName = qn.String
		}
		if sig.Valid {
			h.Signature = sig.String
		}
		if doc.Valid {
			h.DocComment = doc.String
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Boost CA-05: a implementação é a ÚNICA do repositório, a do
	// internal/store. O pacote mcp IMPORTA o store (o store não pode
	// importar o mcp sem criar o ciclo mcp → db → store), então a direção
	// correta é o mcp REUSAR a do store — nunca o contrário.
	if includeBoost {
		ApplyCodeSymbolBoostMCP(hits, query)
	}
	return hits, nil
}

// ApplyCodeSymbolBoostMCP aplica o boost CA-05 nos hits da tool MCP.
//
// Regra (idêntica à de internal/store.ApplyCodeSymbolBoost): quando a query
// é exatamente o qualified name do símbolo (e tem a forma de qualified name),
// o score dobra (2.0) e o hit é marcado como boosted.
func ApplyCodeSymbolBoostMCP(hits []CodeSymbolResult, query string) {
	if len(hits) == 0 {
		return
	}
	storeHits := make([]store.CodeSymbolHit, len(hits))
	for i, h := range hits {
		storeHits[i] = store.CodeSymbolHit{
			SymbolID:      h.SymbolID,
			Kind:          h.Kind,
			Name:          h.Name,
			QualifiedName: h.QualifiedName,
			Signature:     h.Signature,
			DocComment:    h.DocComment,
			StartLine:     h.StartLine,
			EndLine:       h.EndLine,
			File:          h.File,
			Language:      h.Language,
			Score:         h.Score,
			Boosted:       h.Boosted,
		}
	}
	store.ApplyCodeSymbolBoost(storeHits, query)
	for i := range hits {
		hits[i].Score = storeHits[i].Score
		hits[i].Boosted = storeHits[i].Boosted
	}
}

// SQLCodeNeighbors é a implementação padrão em SQL puro via CTE recursivo
// de CodeNeighborsFunc (CA-11).
//
// Devolve (neighbors, root, err). root pode ser nil quando o símbolo não existe.
//
// A travessia COMPLETA (com hub threshold, orçamento de tokens e dica para
// estreitar a query) é `SQLCodeNeighborsTraverse` — esta é a forma crua,
// sem poda.
func SQLCodeNeighbors(ctx context.Context, database *sql.DB, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
	return SQLCodeNeighborsScoped(ctx, database, symbol, depth, direction, "")
}

// SQLCodeNeighborsScoped é o SQLCodeNeighbors com filtro de repositório
// (CG-07): a raiz e as arestaspertencecem só ao repositório pedido.
func SQLCodeNeighborsScoped(ctx context.Context, database *sql.DB, symbol string, depth int, direction string, repository string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
	if database == nil {
		return nil, nil, errors.New("SQLCodeNeighbors: database == nil")
	}
	if depth <= 0 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		direction = "both"
	}

	// Resolve root symbol (mesma política do CLI: exact qualified_name → suffix fallback).
	root, err := resolveCodeSymbolRow(ctx, database, symbol, repository)
	if err != nil {
		return nil, nil, err
	}
	if root == nil {
		return nil, nil, fmt.Errorf("símbolo %q não encontrado em code_symbols", symbol)
	}

	var neighbors []CodeNeighborResult

	if direction == "out" || direction == "both" {
		out, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "out", repository)
		if err != nil {
			return nil, nil, err
		}
		neighbors = append(neighbors, neighborRowsToResults(out)...)
	}
	if direction == "in" || direction == "both" {
		in, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "in", repository)
		if err != nil {
			return nil, nil, err
		}
		neighbors = append(neighbors, neighborRowsToResults(in)...)
	}

	return neighbors, root, nil
}

// neighborRowsToResults converte as linhas da travessia nos vizinhos
// expostos pela tool.
func neighborRowsToResults(rows []codeWalkRow) []CodeNeighborResult {
	if len(rows) == 0 {
		return nil
	}
	out := make([]CodeNeighborResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.neighbor)
	}
	return out
}

// resolveCodeSymbolRow devolve o code_symbol resolvido (ou nil).
func resolveCodeSymbolRow(ctx context.Context, database *sql.DB, symbol, repository string) (*CodeSymbolResult, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, nil
	}
	query := `
		SELECT s.id, s.kind, s.name, s.qualified_name, s.signature, s.doc_comment,
                s.start_line, s.end_line, f.path, f.language
           FROM code_symbols s
           JOIN code_files f ON f.id = s.file_id
          WHERE (s.qualified_name = ? OR s.name = ?)`
	args := []any{symbol, lastSegment(symbol)}
	if repo := strings.TrimSpace(repository); repo != "" {
		query += ` AND f.repository = ? AND s.repository = ?`
		args = append(args, repo, repo)
	}
	query += ` ORDER BY (CASE WHEN s.qualified_name = ? THEN 0 ELSE 1 END), s.id ASC
          LIMIT 1`
	args = append(args, symbol)

	rows, err := database.QueryContext(ctx, bindCodeQuery(ctx, database, query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	var (
		r   CodeSymbolResult
		qn  sql.NullString
		sig sql.NullString
		doc sql.NullString
	)
	if err := rows.Scan(&r.SymbolID, &r.Kind, &r.Name, &qn, &sig, &doc,
		&r.StartLine, &r.EndLine, &r.File, &r.Language); err != nil {
		return nil, err
	}
	if qn.Valid {
		r.QualifiedName = qn.String
	}
	if sig.Valid {
		r.Signature = sig.String
	}
	if doc.Valid {
		r.DocComment = doc.String
	}
	r.Score = 1.0
	return &r, nil
}

// codeWalkRow é uma linha da travessia: o vizinho (o payload) mais os ids
// dos símbolos de origem e destino — o grau do nó (usado pelo hub
// threshold) é calculado sobre estes ids.
type codeWalkRow struct {
	neighbor   CodeNeighborResult
	srcID      int64
	dstID      int64
	neighborID int64
}

// walkCodeEdges executa a CTE recursiva de saída OU entrada.
//
// A CTE carrega a provenance da aresta (epistemic_status, confidence) e o
// resultado expoe o call-site (arquivo/linha da aresta) e a posição do
// símbolo de destino (dst_file/dst_line).
func walkCodeEdges(ctx context.Context, database *sql.DB, rootID int64, depth int, dir, repository string) ([]codeWalkRow, error) {
	var q string
	var args []any
	if dir == "out" {
		q = `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop, epistemic_status, confidence) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1, epistemic_status, confidence
      FROM code_edges
     WHERE src_symbol_id = ?`
		args = append(args, rootID)
		if repo := strings.TrimSpace(repository); repo != "" {
			q += ` AND repository = ?`
			args = append(args, repo)
		}
		q += `
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1, e.epistemic_status, e.confidence
      FROM code_edges e
      JOIN code_walk w ON e.src_symbol_id = w.dst
     WHERE w.hop < ?`
		args = append(args, depth)
	} else {
		q = `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop, epistemic_status, confidence) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1, epistemic_status, confidence
      FROM code_edges
     WHERE dst_symbol_id = ?`
		args = append(args, rootID)
		if repo := strings.TrimSpace(repository); repo != "" {
			q += ` AND repository = ?`
			args = append(args, repo)
		}
		q += `
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1, e.epistemic_status, e.confidence
      FROM code_edges e
      JOIN code_walk w ON e.dst_symbol_id = w.src
     WHERE w.hop < ?`
		args = append(args, depth)
	}
	q += `
)
SELECT '` + dir + `' AS direction, w.kind,
       COALESCE(s.qualified_name, s.name) AS symbol,
       f.path AS file_path, w.line, w.hop,
       w.epistemic_status, w.confidence,
       f.path AS call_site_file, w.line AS call_site_line,
       df.path AS dst_file, ds.start_line AS dst_line,
       w.src, w.dst
  FROM code_walk w
  JOIN code_symbols s  ON s.id = w.` + walkTarget(dir) + `
  JOIN code_files f    ON f.id = w.file_id
  JOIN code_symbols ds ON ds.id = w.` + walkTarget(dir) + `
  JOIN code_files df   ON df.id = ds.file_id
 ORDER BY w.hop ASC, w.kind ASC, symbol ASC`

	rows, err := database.QueryContext(ctx, bindCodeQuery(ctx, database, q), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []codeWalkRow
	for rows.Next() {
		var n CodeNeighborResult
		var status sql.NullString
		var srcID, dstID int64
		if err := rows.Scan(&n.Direction, &n.Kind, &n.Symbol, &n.File, &n.Line, &n.Hop,
			&status, &n.Confidence, &n.CallSiteFile, &n.CallSiteLine,
			&n.DstFile, &n.DstLine, &srcID, &dstID); err != nil {
			return nil, err
		}
		if status.Valid {
			n.EpistemicStatus = status.String
		}
		neighborID := dstID
		if dir == "in" {
			neighborID = srcID
		}
		out = append(out, codeWalkRow{
			neighbor:   n,
			srcID:      srcID,
			dstID:      dstID,
			neighborID: neighborID,
		})
	}
	return out, rows.Err()
}

// SQLCodeNeighborsTraverse executa a travessia COMPLETA de um símbolo:
//
//   - vizinho estrutural (CTE recursiva, ADR-004);
//   - DOCUMENTOS que referenciam o símbolo (a ponte notas↔código, CG-18);
//   - hub threshold (percentil do grau, CG-22);
//   - ordem por DISTÂNCIA EM SALTOS a partir da semente;
//   - orçamento de tokens na resposta (ADR-029), com a dica de estreitar a
//     query quando o corte acontece (AC 23).
//
// maxTokens <= 0 usa o orçamento default (DefaultCodeTokenBudget).
func SQLCodeNeighborsTraverse(ctx context.Context, database *sql.DB, symbol string, depth int, direction string, repository string, maxTokens int) (*MemoryCodeNeighborsResponse, error) {
	if database == nil {
		return nil, errors.New("SQLCodeNeighborsTraverse: database == nil")
	}
	if depth <= 0 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}
	if maxTokens <= 0 {
		maxTokens = DefaultCodeTokenBudget
	}
	direction = strings.ToLower(strings.TrimSpace(direction))
	if direction == "" {
		direction = "both"
	}

	root, err := resolveCodeSymbolRow(ctx, database, symbol, repository)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("símbolo %q não encontrado em code_symbols", symbol)
	}

	resp := &MemoryCodeNeighborsResponse{
		Root:      root,
		Neighbors: []CodeNeighborResult{},
		Depth:     depth,
		Direction: direction,
	}

	// 1) Vizinho estrutural.
	var rows []codeWalkRow
	if direction == "out" || direction == "both" {
		out, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "out", repository)
		if err != nil {
			return nil, err
		}
		rows = append(rows, out...)
	}
	if direction == "in" || direction == "both" {
		in, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "in", repository)
		if err != nil {
			return nil, err
		}
		rows = append(rows, in...)
	}

	// 2) Hub threshold (percentil do grau): os nós-hub saem do subgrafo.
	rows, threshold, pruned := applyHubThreshold(rows, hubThresholdPercentile())

	// 3) Documentos que referenciam o símbolo (a ponte). Não entram no
	// cálculo de grau: eles não são nós do grafo de código.
	docs, err := db.CodeBridgeDocumentsForSymbol(ctx, database, root.QualifiedName)
	if err == nil {
		for _, d := range docs {
			rows = append(rows, codeWalkRow{
				neighbor: CodeNeighborResult{
					Direction:       "document",
					Kind:            db.BridgeRelation,
					Symbol:          d.Path,
					File:            d.Path,
					Hop:             1,
					EpistemicStatus: d.Status,
					DocumentID:      d.DocumentID,
					DocumentPath:    d.Path,
				},
			})
		}
	}

	// 4) Ordem por distância em saltos a partir da semente.
	sortCodeNeighbors(rows)

	// 5) Orçamento de tokens (ADR-029 — o estimador é o do memory_pack_context).
	kept, tokens, truncated := truncateCodeNeighbors(rows, maxTokens)

	resp.Neighbors = kept
	resp.TokensEstimated = tokens
	resp.HubThreshold = threshold
	resp.NodesPruned = pruned
	resp.Truncated = truncated || pruned > 0
	if resp.Truncated {
		resp.Hint = codeNarrowHint(maxTokens, len(rows)-len(kept), pruned)
	}
	return resp, nil
}

// CodeImpactSymbol é um símbolo de código alcançado pelo memory_get_impact.
//
// A ponte notas↔código (CG-18) faz o impacto atravessar a fronteira entre
// o grafo de notas e o de código: a partir de um documento, o impacto
// alcança os símbolos que o documento referencia; a partir de um símbolo,
// alcança os documentos que o referenciam.
type CodeImpactSymbol struct {
	// NodeID é o id do nó no grafo único ("code:<qualified_name>").
	NodeID string `json:"node_id"`
	// QualifiedName é a identidade do símbolo.
	QualifiedName string `json:"qualified_name"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Language      string `json:"language"`
	File          string `json:"file"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	// Depth é a profundidade a que o símbolo foi alcançado (0 = semente).
	Depth int `json:"depth"`
	// Relation é a relação que ligou o nó ao impacto.
	Relation string `json:"relation,omitempty"`
	// ViaNode é o nó por onde o símbolo foi alcançado.
	ViaNode string `json:"via_node,omitempty"`
}

// ImpactCodeSymbolsFunc resolve os símbolos de código alcançados a partir
// de uma semente do grafo.
type ImpactCodeSymbolsFunc func(ctx context.Context, seedNodeID string, nodeIDs []string) ([]CodeImpactSymbol, error)

// ResolveCodeImpactSymbols resolve os símbolos de código alcançados pelo
// impacto a partir de seedNodeID.
//
// Duas direções:
//   - os nós de código que já estão no resultado do impacto (dependentes da
//     semente);
//   - os símbolos que a SEMENTE referencia (a ponte `references_code`): é o
//     que faz o impacto a partir de um DOCUMENTO alcançar o código.
func ResolveCodeImpactSymbols(ctx context.Context, database *sql.DB, seedNodeID string, nodeIDs []string) ([]CodeImpactSymbol, error) {
	if database == nil {
		return nil, errors.New("ResolveCodeImpactSymbols: database == nil")
	}

	var out []CodeImpactSymbol
	seen := make(map[string]bool)

	appendSymbol := func(nodeID string, depth int, relation, viaNode string) {
		if !strings.HasPrefix(nodeID, db.CodeSymbolNodePrefix) {
			return
		}
		qn := strings.TrimPrefix(nodeID, db.CodeSymbolNodePrefix)
		if qn == "" || seen[qn] {
			return
		}
		ref, err := db.CodeSymbolByNodeID(ctx, database, qn)
		if err != nil || ref == nil {
			return
		}
		seen[qn] = true
		out = append(out, CodeImpactSymbol{
			NodeID:        nodeID,
			QualifiedName: ref.QualifiedName,
			Name:          ref.Name,
			Kind:          ref.Kind,
			Language:      ref.Language,
			File:          ref.File,
			StartLine:     ref.StartLine,
			EndLine:       ref.EndLine,
			Depth:         depth,
			Relation:      relation,
			ViaNode:       viaNode,
		})
	}

	// 1) nós de código já presentes no resultado do impacto.
	for _, id := range nodeIDs {
		appendSymbol(id, 1, db.BridgeRelation, seedNodeID)
	}

	// 2) símbolos que a semente (um documento) referencia.
	symbols, err := db.CodeBridgeSymbolsForDocument(ctx, database, seedNodeID)
	if err == nil {
		for _, id := range symbols {
			appendSymbol(id, 1, db.BridgeRelation, seedNodeID)
		}
	}

	return out, nil
}

// hubThresholdPercentile devolve o percentil de grau em uso.
//
// Default: p99 (DefaultHubThresholdPercentile). Configurável pela variável
// de ambiente `MY_MEMORY_CODE_HUB_PERCENTILE`.
func hubThresholdPercentile() float64 {
	raw := strings.TrimSpace(os.Getenv(HubThresholdPercentileEnv))
	if raw == "" {
		return DefaultHubThresholdPercentile
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 || value > 100 {
		return DefaultHubThresholdPercentile
	}
	return value
}

// applyHubThreshold remove do subgrafo os nós cujo grau supera o percentil
// da distribuição de grau do próprio subgrafo.
//
// A regra é a do Graphify (`graphify/analyze.py`, `god_nodes`): o limiar é
// o grau no índice `int(len(degrees) * percentile / 100) - 1` da lista
// ordenada de graus, e o nó é removido quando `degree > limiar`.
//
// DOIS LIMITES (defeito D4):
//
//  1. O corte só vale para grafos GRANDES. O p99 é um percentil de grafo
//     grande: sobre poucos nós o índice do percentil cai no penúltimo grau
//     e o corte remove o vizinho mais relevante (o `calls` de hop 1) em vez
//     do hub — invertendo a relevância. Abaixo de MinHubPruneNodes nós a
//     travessia NÃO poda nada.
//  2. Com o índice inteiro, o limiar cai no TOPO da distribuição: o nó
//     removido é sempre o de MAIOR grau (nunca o penúltimo por erro de
//     índice).
//
// Devolve (linhas mantidas, limiar aplicado, número de linhas removidas).
func applyHubThreshold(rows []codeWalkRow, percentile float64) ([]codeWalkRow, int, int) {
	if len(rows) == 0 {
		return rows, 0, 0
	}
	degree := make(map[int64]int)
	for _, r := range rows {
		if r.neighborID == 0 {
			continue
		}
		degree[r.srcID]++
		degree[r.dstID]++
	}
	if len(degree) == 0 {
		return rows, 0, 0
	}
	// Subgrafo pequeno: o percentil não distingue hub de vizinho comum.
	if len(degree) < MinHubPruneNodes {
		return rows, 0, 0
	}

	degrees := make([]int, 0, len(degree))
	for _, d := range degree {
		degrees = append(degrees, d)
	}
	sort.Ints(degrees)

	idx := int(float64(len(degrees))*percentile/100) - 1
	if idx < 0 {
		idx = 0
	}
	if idx > len(degrees)-1 {
		idx = len(degrees) - 1
	}
	threshold := degrees[idx]

	pruned := 0
	kept := rows[:0]
	for _, r := range rows {
		if r.neighborID != 0 && degree[r.neighborID] > threshold {
			pruned++
			continue
		}
		kept = append(kept, r)
	}
	return kept, threshold, pruned
}

// sortCodeNeighbors ordena as linhas da travessia por DISTÂNCIA EM SALTOS
// (hop) a partir da semente. O desempate é estável e determinístico.
func sortCodeNeighbors(rows []codeWalkRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].neighbor.Hop != rows[j].neighbor.Hop {
			return rows[i].neighbor.Hop < rows[j].neighbor.Hop
		}
		if rows[i].neighbor.Direction != rows[j].neighbor.Direction {
			return rows[i].neighbor.Direction < rows[j].neighbor.Direction
		}
		if rows[i].neighbor.Kind != rows[j].neighbor.Kind {
			return rows[i].neighbor.Kind < rows[j].neighbor.Kind
		}
		return rows[i].neighbor.Symbol < rows[j].neighbor.Symbol
	})
}

// truncateCodeNeighbors corta a resposta quando o orçamento de tokens é
// estourado. O estimador é o JÁ EXISTO no repo (ADR-029):
// `graph.EstimateTokens` — o mesmo do memory_pack_context.
//
// Devolve (vizinhos mantidos, tokens estimados, truncado).
func truncateCodeNeighbors(rows []codeWalkRow, maxTokens int) ([]CodeNeighborResult, int, bool) {
	kept := make([]CodeNeighborResult, 0, len(rows))
	used := 0
	truncated := false
	for _, r := range rows {
		cost := estimateCodeTokens(r.neighbor)
		if used+cost > maxTokens {
			truncated = true
			// A resposta nunca fica vazia por causa do orçamento: o primeiro
			// vizinho sempre entra (o corte é sinalizado na dica).
			if len(kept) == 0 {
				kept = append(kept, r.neighbor)
				used += cost
			}
			break
		}
		kept = append(kept, r.neighbor)
		used += cost
	}
	return kept, used, truncated
}

// truncateCodeResults corta a lista de hits da busca pelo orçamento de
// tokens (mesmo estimador e mesma regra de truncateCodeNeighbors).
func truncateCodeResults(hits []CodeSymbolResult, maxTokens int) ([]CodeSymbolResult, int, bool) {
	kept := make([]CodeSymbolResult, 0, len(hits))
	used := 0
	truncated := false
	for _, h := range hits {
		cost := estimateCodeTokens(h)
		if used+cost > maxTokens {
			truncated = true
			if len(kept) == 0 {
				kept = append(kept, h)
				used += cost
			}
			break
		}
		kept = append(kept, h)
		used += cost
	}
	return kept, used, truncated
}

// estimateCodeTokens estima o custo de um item da resposta em tokens.
//
// O estimador é o do repo (ADR-029 — `graph.EstimateTokens`, 1 token ~ 4
// caracteres): nenhum estimador novo foi criado.
func estimateCodeTokens(payload any) int {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return 0
	}
	return graph.EstimateTokens(string(encoded))
}

// codeTokenBudget lê o orçamento de tokens dos argumentos da tool.
func codeTokenBudget(rawMap map[string]json.RawMessage) int {
	budget := DefaultCodeTokenBudget
	if raw, ok := rawMap["max_tokens"]; ok {
		var t int
		if err := json.Unmarshal(raw, &t); err == nil && t > 0 {
			budget = t
		}
	}
	return budget
}

// codeNarrowHint monta a dica de estreitar a query (AC 23).
//
// A dica diz o QUE cortou a resposta: o hub threshold (nós de grau alto
// removidos) ou o orçamento de tokens (vizinhos que ficaram de fora).
func codeNarrowHint(maxTokens, dropped, pruned int) string {
	hint := "Resposta truncada. Estreite a query: use o qualified name completo, reduza a profundidade (depth) ou filtre por language/kind."
	var reasons []string
	if pruned > 0 {
		reasons = append(reasons, fmt.Sprintf("%d no(s)-hub removido(s) (grau acima do percentil)", pruned))
	}
	if dropped > 0 {
		reasons = append(reasons, fmt.Sprintf("%d vizinho(s) fora do orcamento de %d tokens", dropped, maxTokens))
	}
	if len(reasons) > 0 {
		hint = fmt.Sprintf("%s (%s).", hint, strings.Join(reasons, "; "))
	}
	return hint
}

// walkTarget devolve a coluna da CTE que é o símbolo destination da
// travessia: "out" olha o destino da aresta, "in" olha a origem.
func walkTarget(dir string) string {
	if dir == "in" {
		return "src"
	}
	return "dst"
}

// bindCodeQuery reescreve os placeholders "?" para o dialeto do engine ativo:
// o SQLite aceita "?", o lib/pq exige "$1" (o mesmo code graph nos dois
// engines).
func bindCodeQuery(ctx context.Context, database *sql.DB, query string) string {
	return db.DetectDialect(ctx, database).Bind(query)
}

func lastSegment(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func mcpTokenize(query string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == ';'
	}) {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	return out
}

// HasQualifiedNameShapeMCP devolve true quando a query tem a forma de um
// qualified name (segmentos separados por ".", cada um um identificador).
//
// A implementação é a de internal/store (HasQualifiedNameShape) — o mcp
// importa o store, então é o mcp que reusa a do store.
func HasQualifiedNameShapeMCP(query string) bool {
	return store.HasQualifiedNameShape(query)
}
