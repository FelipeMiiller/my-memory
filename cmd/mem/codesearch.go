package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// runCodeSearchCLI implementa `mem code-search <query>` (CA-07 / ADR-047).
//
// Estratégia: consulta SQL direta sobre code_symbols aplicando
// match por substring em name/qualified_name/signature e aplica o boost
// CA-05 por qualified_name match exato (codeast.ApplyBoostToHits).
//
// Flags:
//
//	--lang=csv       filtra por linguagens
//	--kind=kind      filtra por symbol kind (function, method, class, ...)
//	--limit=N        máximo de resultados (padrão 10)
//	--no-code-boost  desativa o boost 2x por qualified_name match
//	--db=<arq>       caminho do SQLite (padrão memory.db)
//	--repo=<slug>    slug multi-tenant (placeholder)
//
// Nota (CA-07): a especificação prevê delegação a `mem search --kind=code`.
// A busca Markdown existente (`runSearchSQLite`) não indexa code_symbols
// ainda — a integração RRF unificada é ADR-040 (storage) + backlog ADR-047.
// Este comando implementa o sub-grafo de busca de código enquanto isso.
func runCodeSearchCLI(ctx context.Context, defaultRepo string, args []string) error {
	fs := flag.NewFlagSet("code-search", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Caminho do arquivo SQLite (padrão: memory.db)")
	repoSlug := fs.String("repo", "", "Identificador/slug do repositório")
	lang := fs.String("lang", "", "Lista csv de linguagens permitidas (ex: 'go,python'); vazio = todas")
	kind := fs.String("kind", "", "Filtra por symbol kind (function, method, class, ...)")
	limit := fs.Int("limit", 10, "Número máximo de resultados (padrão: 10)")
	noBoost := fs.Bool("no-code-boost", false, "Desativa o boost 2x por qualified_name match (CA-05)")
	showLinks := fs.Bool("links", false, "Exibe deep links (VS Code / File) ao lado de cada resultado")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_ = defaultRepo
	_ = repoSlug

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		fmt.Println("Uso: mem code-search [--lang=go,py,...] [--kind=function] [--limit=10] [--no-code-boost] [--links] [--db=<arq>] [--repo=<slug>] <query>")
		return errors.New("query vazia")
	}

	target := *dbPath
	if target == "" {
		target = "memory.db"
	}

	database, err := db.InitDB(target)
	if err != nil {
		return fmt.Errorf("InitDB(%s) falhou: %w", target, err)
	}
	defer database.Close()

	hits, err := runCodeSearchQuery(ctx, database, query, *lang, *kind, *limit)
	if err != nil {
		return err
	}

	// Aplica boost CA-05 in-place (ordenando por score decrescente)
	noBoostVal := *noBoost
	codeast.ApplyBoostToHits(hits, query, noBoostVal)

	// Output tabular
	printCodeSearchResults(query, hits, *showLinks)
	return nil
}

// runCodeSearchQuery é a função pura (sem fmt) usada tanto pela CLI quanto
// por testes que validam o pipeline isoladamente.
func runCodeSearchQuery(ctx context.Context, database *sql.DB, query, langCSV, kind string, limit int) ([]codeast.CodeSearchHit, error) {
	if database == nil {
		return nil, errors.New("runCodeSearchQuery: database == nil")
	}
	query = strings.TrimSpace(query)
	if limit <= 0 {
		limit = 10
	}

	// Match simples: LIKE '%query%' em name, qualified_name e signature.
	// Para query com espaços, quebramos em tokens e exigimos TODOS os tokens
	// (AND semantics) — uma implementação mais sofisticada (FTS5) é ADR futuro.
	tokens := tokenizeQuery(query)

	// Constrói SQL com placeholders; cada token vira um LIKE em cada coluna.
	var (
		where []string
		args  []any
	)
	for _, tok := range tokens {
		like := "%" + tok + "%"
		where = append(where, "(s.name LIKE ? OR s.qualified_name LIKE ? OR s.signature LIKE ?)")
		args = append(args, like, like, like)
	}

	if strings.TrimSpace(langCSV) != "" {
		langs := splitCSV(langCSV)
		if len(langs) > 0 {
			placeholders := make([]string, len(langs))
			for i, l := range langs {
				placeholders[i] = "?"
				args = append(args, l)
			}
			where = append(where, "f.language IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if strings.TrimSpace(kind) != "" {
		where = append(where, "s.kind = ?")
		args = append(args, strings.TrimSpace(kind))
	}

	args = append(args, limit)
	sqlQuery := `
SELECT s.id, s.kind, s.name, s.qualified_name, s.signature, s.doc_comment,
       s.start_line, s.end_line, f.path, f.language, 1.0 AS rrf_score
FROM code_symbols s
JOIN code_files f ON f.id = s.file_id
WHERE ` + strings.Join(where, " AND ") + `
LIMIT ?`

	rows, err := database.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query code_symbols falhou: %w", err)
	}
	defer rows.Close()

	var hits []codeast.CodeSearchHit
	for rows.Next() {
		var (
			h     codeast.CodeSearchHit
			qn    sql.NullString
			sig   sql.NullString
			doc   sql.NullString
			score float64
		)
		if err := rows.Scan(&h.SymbolID, &h.Kind, &h.Name, &qn, &sig, &doc,
			&h.StartLine, &h.EndLine, &h.FilePath, &h.Language, &score); err != nil {
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
		h.RRFScore = score
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

// tokenizeQuery quebra uma query em tokens lowercased não-vazios.
// Separadores: whitespace e '.', mas '.' vira token próprio apenas quando
// é usado em qualified name; mantemos a granularidade padrão
// (palavras) e usamos HasQualifiedNameShape do codeast para detectar
// quando o query inteiro é um qualified_name.
func tokenizeQuery(query string) []string {
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

// printCodeSearchResults imprime a tabela de resultados com boost flag.
func printCodeSearchResults(query string, hits []codeast.CodeSearchHit, showLinks bool) {
	fmt.Printf("query: %q\n", query)
	if len(hits) == 0 {
		fmt.Println("nenhum resultado.")
		return
	}
	fmt.Printf("%-5s %-8s %-22s %-12s %-6s  %s\n", "rank", "kind", "qualified_name", "lang", "boost", "file:line")
	fmt.Println(strings.Repeat("-", 90))
	for i, h := range hits {
		flag := " "
		if h.Boosted {
			flag = "*2x"
		}
		qn := h.QualifiedName
		if qn == "" {
			qn = h.Name
		}
		loc := fmt.Sprintf("%s:%d", h.FilePath, h.StartLine)
		if showLinks {
			loc = fmt.Sprintf("%s:%d  (vscode://file/%s:%d)", h.FilePath, h.StartLine, h.FilePath, h.StartLine)
		}
		fmt.Printf("%-5d %-8s %-22s %-12s %-6s  %s\n", i+1, h.Kind, truncate(qn, 22), h.Language, flag, loc)
	}
	fmt.Println(strings.Repeat("-", 90))
	fmt.Printf("%d resultado(s) — '%s' boosted=%d\n", len(hits), query, countBoosted(hits))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func countBoosted(hits []codeast.CodeSearchHit) int {
	n := 0
	for _, h := range hits {
		if h.Boosted {
			n++
		}
	}
	return n
}

// touchOsStdout garante compilação caso o import seja removido por refactor.
var _ = os.Stdout
