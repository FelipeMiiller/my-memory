package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CodeSourceName é o nome da fonte RRF que carrega os símbolos de código
// (code_symbols) no fan-in da busca híbrida (T14/CG-19).
const CodeSourceName = "code"

// CodeDocumentIDPrefix prefixa o DocumentID de um símbolo de código.
//
// O prefixo não colide com o id de uma nota (uuid): um símbolo nunca funde
// com uma nota, ele entra na lista fundida como item próprio.
const CodeDocumentIDPrefix = "code:"

// CodeSymbolHit é um símbolo de código recuperado pela busca lexical em
// code_symbols (name, qualified_name e signature).
type CodeSymbolHit struct {
	SymbolID      int64
	Kind          string
	Name          string
	QualifiedName string
	Signature     string
	DocComment    string
	StartLine     int
	EndLine       int
	File          string
	Language      string
	Score         float64
	Boosted       bool
}

// PlaceholderFunc devolve o marcador do n-ésimo argumento no dialeto do
// engine: "?" no SQLite, "$n" no PostgreSQL (lib/pq).
type PlaceholderFunc func(n int) string

// SQLitePlaceholder devolve "?" — o marcador aceito pelo SQLite.
func SQLitePlaceholder(int) string { return "?" }

// PostgresPlaceholder devolve "$n" — o marcador exigido pelo lib/pq.
func PostgresPlaceholder(n int) string { return "$" + strconv.Itoa(n) }

// BuildCodeSymbolQuery monta o SQL da busca lexical em code_symbols.
//
// A query é escrita uma vez e vale para os dois engines: o marcador é
// parametrizado (SQLite usa "?", lib/pq usa "$n"), o resto do SQL é
// idêntico. O filtro é o mesmo do memory_code_search (mcp/code_handlers.go):
// cada token da query casa em name, qualified_name ou signature.
//
// Quando a query não produz nenhum token, a busca devolve vazio (SQL "") —
// devolver os primeiros símbolos do banco poluiria o ranking com símbolos
// arbitrários.
func BuildCodeSymbolQuery(query, language, kind, repository string, limit int, ph PlaceholderFunc) (string, []any) {
	if ph == nil {
		ph = SQLitePlaceholder
	}

	var where []string
	var args []any

	for _, tok := range TokenizeCodeQuery(query) {
		like := "%" + tok + "%"
		p1, p2, p3 := ph(len(args)+1), ph(len(args)+2), ph(len(args)+3)
		args = append(args, like, like, like)
		where = append(where, "(s.name LIKE "+p1+" OR s.qualified_name LIKE "+p2+" OR s.signature LIKE "+p3+")")
	}

	if len(where) == 0 {
		return "", nil
	}

	if lang := strings.TrimSpace(language); lang != "" {
		p := ph(len(args) + 1)
		args = append(args, lang)
		where = append(where, "f.language = "+p)
	}

	if k := strings.TrimSpace(kind); k != "" {
		p := ph(len(args) + 1)
		args = append(args, k)
		where = append(where, "s.kind = "+p)
	}

	if repo := strings.TrimSpace(repository); repo != "" {
		// Dois marcadores (e dois argumentos): no SQLite cada "?" consome um
		// argumento, enquanto no PostgreSQL o mesmo "$n" pode ser repetido.
		p1, p2 := ph(len(args)+1), ph(len(args)+2)
		args = append(args, repo, repo)
		where = append(where, "(f.repository = "+p1+" AND s.repository = "+p2+")")
	}

	pLimit := ph(len(args) + 1)
	args = append(args, limit)

	sqlText := `
		SELECT s.id, s.kind, s.name, s.qualified_name, s.signature, s.doc_comment,
		       s.start_line, s.end_line, f.path, f.language, 1.0 AS score
		  FROM code_symbols s
		  JOIN code_files f ON f.id = s.file_id
		 WHERE ` + strings.Join(where, " AND ") + `
		 ORDER BY score DESC, s.id ASC
		 LIMIT ` + pLimit

	return sqlText, args
}

// QueryCodeSymbols executa a query lexical de code_symbols e devolve os hits.
//
// Um erro (por exemplo, tabelas de código ausentes num vault antigo) não é
// fatal: o chamador decide o que fazer com a lista vazia.
func QueryCodeSymbols(ctx context.Context, database *sql.DB, sqlText string, args []any) ([]CodeSymbolHit, error) {
	if database == nil {
		return nil, errors.New("QueryCodeSymbols: database == nil")
	}
	if strings.TrimSpace(sqlText) == "" {
		return nil, nil
	}

	rows, err := database.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryCodeSymbols: %w", err)
	}
	defer rows.Close()

	var hits []CodeSymbolHit
	for rows.Next() {
		var (
			h   CodeSymbolHit
			qn  sql.NullString
			sig sql.NullString
			doc sql.NullString
		)
		if err := rows.Scan(&h.SymbolID, &h.Kind, &h.Name, &qn, &sig, &doc,
			&h.StartLine, &h.EndLine, &h.File, &h.Language, &h.Score); err != nil {
			return nil, err
		}
		h.QualifiedName = qn.String
		h.Signature = sig.String
		h.DocComment = doc.String
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

// ApplyCodeSymbolBoost aplica o boost CA-05: quando a query é exatamente o
// qualified name do símbolo (e tem a forma de qualified name), o score
// dobra (2.0) e o hit é marcado como boosted.
//
// Esta é a MESMA regra do memory_code_search (internal/mcp/code_handlers.go,
// SQLCodeSearchScoped). O espelho existe porque o pacote mcp importa o db,
// que importa o store — o store não pode importar o mcp sem criar um ciclo.
// Regra: score *= 2.0 quando HasQualifiedNameShape(query) e
// query == qualified_name (ou name, quando não há qualified_name).
func ApplyCodeSymbolBoost(hits []CodeSymbolHit, query string) {
	if len(hits) == 0 || strings.TrimSpace(query) == "" {
		return
	}
	if !HasQualifiedNameShape(query) {
		return
	}
	for i := range hits {
		qn := strings.TrimSpace(hits[i].QualifiedName)
		if qn == "" {
			qn = strings.TrimSpace(hits[i].Name)
		}
		if strings.TrimSpace(query) == qn {
			hits[i].Score *= 2.0
			hits[i].Boosted = true
		}
	}
}

// SortCodeSymbolHits ordena os hits por score decrescente (é o boost CA-05
// que entra no score) e, no empate, pelo id.
//
// A ordem define o rank do hit na fonte "code" do RRF — por isso o boost
// precisa reordenar: o RRF usa o rank, não o score.
func SortCodeSymbolHits(hits []CodeSymbolHit) {
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].SymbolID < hits[j].SymbolID
	})
}

// CodeSymbolResults converte os hits em SearchResult prontos para a fusão RRF.
//
// A identidade do símbolo é "code:" + qualified_name (prefixo que não colide
// com o id de uma nota): o símbolo nunca funde com uma nota, ele entra na
// lista fundida como item próprio.
func CodeSymbolResults(hits []CodeSymbolHit) []SearchResult {
	if len(hits) == 0 {
		return nil
	}

	out := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		qname := strings.TrimSpace(h.QualifiedName)
		if qname == "" {
			qname = strings.TrimSpace(h.Name)
		}
		out = append(out, SearchResult{
			DocumentID:   CodeDocumentIDPrefix + qname,
			Content:      CodeSymbolContent(h),
			Abstract:     CodeSymbolAbstract(h),
			Category:     "resource",
			CodeSymbol:   qname,
			CodeFile:     h.File,
			CodeLine:     h.StartLine,
			CodeEndLine:  h.EndLine,
			CodeKind:     h.Kind,
			CodeLanguage: h.Language,
		})
	}
	return out
}

// CodeSymbolAbstract monta o micro-abstract do símbolo: a assinatura (ou o
// nome) e a origem (arquivo:linha).
//
// A origem é obrigatória: sem ela o agente recebe um resultado que não dá
// para abrir.
func CodeSymbolAbstract(h CodeSymbolHit) string {
	label := strings.TrimSpace(h.Signature)
	if label == "" {
		label = strings.TrimSpace(h.QualifiedName)
	}
	if label == "" {
		label = strings.TrimSpace(h.Name)
	}
	loc := CodeSymbolLocation(h)
	switch {
	case label == "":
		return loc
	case loc == "":
		return label
	default:
		return label + " — " + loc
	}
}

// CodeSymbolContent monta o corpo do resultado (L1/L2): assinatura, doc
// comment e a origem do símbolo.
func CodeSymbolContent(h CodeSymbolHit) string {
	var sb strings.Builder
	if sig := strings.TrimSpace(h.Signature); sig != "" {
		sb.WriteString(sig)
		sb.WriteString("\n")
	}
	if doc := strings.TrimSpace(h.DocComment); doc != "" {
		sb.WriteString(doc)
		sb.WriteString("\n")
	}
	if loc := CodeSymbolLocation(h); loc != "" {
		sb.WriteString("📄 ")
		sb.WriteString(loc)
	}
	return strings.TrimSpace(sb.String())
}

// CodeSymbolLocation devolve "arquivo:linha" do símbolo (vazio se não há
// arquivo).
func CodeSymbolLocation(h CodeSymbolHit) string {
	file := strings.TrimSpace(h.File)
	if file == "" {
		return ""
	}
	if h.StartLine > 0 {
		return fmt.Sprintf("%s:%d", file, h.StartLine)
	}
	return file
}

// TokenizeCodeQuery quebra a query em tokens únicos (o mesmo corte do
// memory_code_search: espaço, tab, quebra de linha, vírgula e ponto-e-vírgula).
func TokenizeCodeQuery(query string) []string {
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

// HasQualifiedNameShape devolve true quando a query tem a forma de um
// qualified name (segmentos separados por ".", cada um um identificador).
//
// Espelho de HasQualifiedNameShapeMCP (internal/mcp/code_handlers.go) pelo
// mesmo motivo do boost: o store não pode importar o mcp (ciclo).
func HasQualifiedNameShape(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" || !strings.Contains(q, ".") {
		return false
	}
	for _, seg := range strings.Split(q, ".") {
		seg = strings.TrimSpace(seg)
		if !isIdentSeg(seg) {
			return false
		}
	}
	return true
}

func isIdentSeg(s string) bool {
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

// includeCodeCtxKey é a chave (não exportada) da flag include_code no contexto.
type includeCodeCtxKey struct{}

// WithIncludeCode devolve um contexto que carrega a flag include_code.
//
// O contexto é o canal que atravessa a ligação da tool MCP com as
// implementações de busca (SQLite e Postgres) sem depender de assinatura nova
// — o mesmo mecanismo do repo_id (ADR-061).
func WithIncludeCode(ctx context.Context, enabled bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, includeCodeCtxKey{}, enabled)
}

// IncludeCodeFromContext devolve true quando o contexto pede o fan-in de
// code_symbols no RRF.
func IncludeCodeFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	if enabled, ok := ctx.Value(includeCodeCtxKey{}).(bool); ok {
		return enabled
	}
	return false
}

// IncludeCodeRequested devolve true quando o fan-in de código foi pedido —
// pela flag das opções de busca ou pela flag do contexto.
//
// As duas fontes são OR (e a fonte é montada uma única vez), de modo que
// ligar as duas não duplica a contribuição no score.
func IncludeCodeRequested(ctx context.Context, searchOpts SearchOptions) bool {
	return searchOpts.IncludeCode || IncludeCodeFromContext(ctx)
}
