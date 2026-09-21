package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
type CodeNeighborsFunc func(ctx context.Context, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error)

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
type CodeNeighborResult struct {
	Direction string `json:"direction"`
	Kind      string `json:"kind"`
	Symbol    string `json:"symbol"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Hop       int    `json:"hop"`
}

// MemoryCodeSearchResponse envelopa os resultados para a tool MCP.
type MemoryCodeSearchResponse struct {
	Query   string             `json:"query"`
	Count   int                `json:"count"`
	Results []CodeSymbolResult `json:"results"`
}

// MemoryCodeNeighborsResponse envelopa root + neighbors.
type MemoryCodeNeighborsResponse struct {
	Root      *CodeSymbolResult    `json:"root,omitempty"`
	Neighbors []CodeNeighborResult `json:"neighbors"`
	Depth     int                  `json:"depth"`
	Direction string               `json:"direction"`
}

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

		resp := MemoryCodeSearchResponse{
			Query:   query,
			Count:   len(hits),
			Results: hits,
		}
		out, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		return NewTextResult(string(out)), nil
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

		neighbors, root, err := codeFn(ctx, symbol, depth, direction)
		if err != nil {
			return nil, NewError(CodeInternalError, "Falha em memory_code_neighbors", err.Error())
		}

		resp := MemoryCodeNeighborsResponse{
			Root:      root,
			Neighbors: neighbors,
			Depth:     depth,
			Direction: direction,
		}
		out, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		return NewTextResult(string(out)), nil
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
func SQLCodeSearch(ctx context.Context, database *sql.DB, query, language, kind string, limit int, includeBoost bool) ([]CodeSymbolResult, error) {
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

	rows, err := database.QueryContext(ctx, sqlQuery, args...)
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

	// Aplica boost CA-05 quando habilitado — mesma lógica de cmd/mem/codesearch.
	if includeBoost && len(hits) > 0 {
		for i := range hits {
			qn := hits[i].QualifiedName
			if qn == "" {
				qn = hits[i].Name
			}
			if strings.TrimSpace(query) != "" &&
				HasQualifiedNameShapeMCP(query) &&
				strings.TrimSpace(query) == qn {
				hits[i].Score *= 2.0
				hits[i].Boosted = true
			}
		}
	}
	return hits, nil
}

// SQLCodeNeighbors é a implementação padrão em SQL puro via CTE recursivo
// de CodeNeighborsFunc (CA-11).
//
// Devolve (neighbors, root, err). root pode ser nil quando o símbolo não existe.
func SQLCodeNeighbors(ctx context.Context, database *sql.DB, symbol string, depth int, direction string) ([]CodeNeighborResult, *CodeSymbolResult, error) {
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
	root, err := resolveCodeSymbolRow(ctx, database, symbol)
	if err != nil {
		return nil, nil, err
	}
	if root == nil {
		return nil, nil, fmt.Errorf("símbolo %q não encontrado em code_symbols", symbol)
	}

	var neighbors []CodeNeighborResult

	if direction == "out" || direction == "both" {
		out, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "out")
		if err != nil {
			return nil, nil, err
		}
		neighbors = append(neighbors, out...)
	}
	if direction == "in" || direction == "both" {
		in, err := walkCodeEdges(ctx, database, root.SymbolID, depth, "in")
		if err != nil {
			return nil, nil, err
		}
		neighbors = append(neighbors, in...)
	}

	return neighbors, root, nil
}

// resolveCodeSymbolRow devolve o code_symbol resolvido (ou nil).
func resolveCodeSymbolRow(ctx context.Context, database *sql.DB, symbol string) (*CodeSymbolResult, error) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil, nil
	}
	rows, err := database.QueryContext(ctx,
		`SELECT s.id, s.kind, s.name, s.qualified_name, s.signature, s.doc_comment,
                s.start_line, s.end_line, f.path, f.language
           FROM code_symbols s
           JOIN code_files f ON f.id = s.file_id
          WHERE s.qualified_name = ? OR s.name = ?
          ORDER BY (CASE WHEN s.qualified_name = ? THEN 0 ELSE 1 END), s.id ASC
          LIMIT 1`,
		symbol, lastSegment(symbol), symbol,
	)
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

// walkCodeEdges executa a CTE recursiva de saída OU entrada.
func walkCodeEdges(ctx context.Context, database *sql.DB, rootID int64, depth int, dir string) ([]CodeNeighborResult, error) {
	var q string
	if dir == "out" {
		q = `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1
      FROM code_edges
     WHERE src_symbol_id = ?
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1
      FROM code_edges e
      JOIN code_walk w ON e.src_symbol_id = w.dst
     WHERE w.hop < ?
)
SELECT 'out' AS direction, w.kind,
       COALESCE(s.qualified_name, s.name) AS symbol,
       f.path AS file_path, w.line, w.hop
  FROM code_walk w
  JOIN code_symbols s ON s.id = w.dst
  JOIN code_files f   ON f.id = w.file_id
 ORDER BY w.hop ASC, w.kind ASC, symbol ASC`
	} else {
		q = `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1
      FROM code_edges
     WHERE dst_symbol_id = ?
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1
      FROM code_edges e
      JOIN code_walk w ON e.dst_symbol_id = w.src
     WHERE w.hop < ?
)
SELECT 'in' AS direction, w.kind,
       COALESCE(s.qualified_name, s.name) AS symbol,
       f.path AS file_path, w.line, w.hop
  FROM code_walk w
  JOIN code_symbols s ON s.id = w.src
  JOIN code_files f   ON f.id = w.file_id
 ORDER BY w.hop ASC, w.kind ASC, symbol ASC`
	}
	rows, err := database.QueryContext(ctx, q, rootID, depth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CodeNeighborResult
	for rows.Next() {
		var n CodeNeighborResult
		if err := rows.Scan(&n.Direction, &n.Kind, &n.Symbol, &n.File, &n.Line, &n.Hop); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
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

// HasQualifiedNameShapeMCP espelha a heurística de codeast.HasQualifiedNameShape
// sem importar codeast (para evitar ciclo: mcp → codeast → db → mcp).
func HasQualifiedNameShapeMCP(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" || !strings.Contains(q, ".") {
		return false
	}
	for _, seg := range strings.Split(q, ".") {
		seg = strings.TrimSpace(seg)
		if !isIdentSegMCP(seg) {
			return false
		}
	}
	return true
}

func isIdentSegMCP(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
