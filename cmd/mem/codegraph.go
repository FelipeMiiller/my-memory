package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// runCodeGraphCLI implementa `mem code-graph <symbol>` (CA-08 / ADR-047).
//
// Estratégia:
//  1. Resolve o símbolo (exact qualified_name match, fallback suffix .name).
//  2. Executa CTE recursivo (ADR-004) sobre code_edges até `depth` hops.
//  3. Imprime tabela Markdown: direction | kind | symbol | file | line.
//
// Flags:
//
//	--depth=N       profundidade máxima (padrão 1, máx 5)
//	--direction     in | out | both (padrão both)
//	--storage       sqlite | postgres (padrão: engine configurado)
//	--db=<arq>      caminho do SQLite (implica --storage=sqlite)
//	--postgres=<url> URL PostgreSQL com pgvector (alternativa ao SQLite)
//	--repo=<slug>   repositório consultado (padrão: repo_id do vault)
//
// Nenhuma flag é aceita e descartada (CG-09): o `--repo` filtra a consulta
// (o code graph é repo-scoped) e o `--storage`/`--db`/`--postgres` escolhem o
// engine — como no `mem code-index` (ADR-061).
func runCodeGraphCLI(ctx context.Context, defaultRepo string, args []string) error {
	codeast.LogTreesitterBootStatus() // CA-14: boot log uniforme em todos os code-*
	fs := flag.NewFlagSet("code-graph", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Caminho do arquivo SQLite (implica --storage=sqlite)")
	pgURL := fs.String("postgres", "", "URL PostgreSQL com pgvector (alternativa ao SQLite)")
	storage := fs.String("storage", "", "Força engine: 'sqlite' ou 'postgres'")
	repoSlug := fs.String("repo", "", "Identificador/slug do repositório")
	depth := fs.Int("depth", 1, "Profundidade máxima de travessia (padrão: 1, máx: 5)")
	direction := fs.String("direction", "both", "Direção: 'in', 'out' ou 'both'")
	if err := fs.Parse(args); err != nil {
		return err
	}

	symbol := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if symbol == "" {
		fmt.Println("Uso: mem code-graph [--depth=N] [--direction=in|out|both] [--storage=sqlite|postgres] [--db=<arq>] [--postgres=<url>] [--repo=<slug>] <qualified_name>")
		return errors.New("symbol vazio")
	}

	if *depth <= 0 {
		*depth = 1
	}
	if *depth > 5 {
		*depth = 5
	}

	// CG-09: o --repo vale de verdade — o code graph é repo-scoped e a
	// consulta é filtrada pelo repositório pedido (padrão: o repo_id do
	// vault).
	repository := strings.TrimSpace(*repoSlug)
	if repository == "" {
		repository = strings.TrimSpace(defaultRepo)
	}
	ctx = db.WithRepository(ctx, repository)

	// O code graph é consultado no engine configurado (ADR-061), como no
	// `mem code-index`: Postgres quando Postgres, SQLite quando SQLite.
	cfg, _, _ := config.LoadCascadingConfig(".")
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}
	engine, err := resolveCodeEngine(cfg, *storage, *pgURL, *dbPath)
	if err != nil {
		return err
	}
	defer engine.Close()

	rows, err := runCodeGraphQuery(ctx, engine.db, symbol, *depth, *direction)
	if err != nil {
		return err
	}
	printCodeGraphTable(symbol, rows)
	return nil
}

// CodeGraphRow é a estrutura tabular mínima retornada pela query e impressa
// pelo CLI; reutilizada nos testes para assertions estruturais.
type CodeGraphRow struct {
	Direction string `json:"direction"` // "out" ou "in"
	Kind      string `json:"kind"`
	Symbol    string `json:"symbol"`
	FilePath  string `json:"file"`
	Line      int    `json:"line"`
	Hop       int    `json:"hop"`
}

// resolveCodeSymbol encontra o ID do símbolo a partir de uma query.
// Política:
//  1. exact match em qualified_name
//  2. fallback: suffix match em name (último segmento do qualified_name == name)
//
// A busca é filtrada pelo repositório do contexto (o code graph é
// repo-scoped — ADR-061): o mesmo qualified_name pode existir em repositórios
// diferentes, e cada um tem o seu próprio símbolo.
//
// Devolve -1 quando não acha (caller deve emitir mensagem amigável).
func resolveCodeSymbol(ctx context.Context, database *sql.DB, symbol string) (int64, error) {
	if database == nil {
		return -1, errors.New("resolveCodeSymbol: database == nil")
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return -1, errors.New("resolveCodeSymbol: symbol vazio")
	}
	dialect := db.DetectDialect(ctx, database)
	repository := db.RepositoryFromContext(ctx)

	// 1. Exact qualified_name match.
	var id int64
	err := database.QueryRowContext(ctx,
		dialect.Bind(`SELECT id FROM code_symbols WHERE qualified_name = ? AND repository = ? ORDER BY id LIMIT 1`),
		symbol, repository,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return -1, err
	}

	// 2. Fallback: suffix match em name (semelhante a `Mem.find` por basename).
	suffix := symbol
	if idx := strings.LastIndex(symbol, "."); idx >= 0 {
		suffix = symbol[idx+1:]
	}
	err = database.QueryRowContext(ctx,
		dialect.Bind(`SELECT id FROM code_symbols WHERE name = ? AND repository = ? ORDER BY id LIMIT 1`),
		suffix, repository,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return -1, err
}

// runCodeGraphQuery executa o CTE recursivo (ADR-004) sobre code_edges e
// devolve a vizinhança do símbolo.
//
// A travessia é filtrada pelo repositório do contexto (repo-scoped).
func runCodeGraphQuery(ctx context.Context, database *sql.DB, symbol string, depth int, direction string) ([]CodeGraphRow, error) {
	if database == nil {
		return nil, errors.New("runCodeGraphQuery: database == nil")
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

	rootID, err := resolveCodeSymbol(ctx, database, symbol)
	if err != nil {
		return nil, err
	}
	if rootID <= 0 {
		return nil, fmt.Errorf("símbolo %q não encontrado em code_symbols", symbol)
	}

	repository := db.RepositoryFromContext(ctx)
	dialect := db.DetectDialect(ctx, database)

	var rows []CodeGraphRow

	appendRows := func(rowsIter *sql.Rows) error {
		defer rowsIter.Close()
		for rowsIter.Next() {
			var r CodeGraphRow
			if err := rowsIter.Scan(&r.Direction, &r.Kind, &r.Symbol, &r.FilePath, &r.Line, &r.Hop); err != nil {
				return err
			}
			rows = append(rows, r)
		}
		return rowsIter.Err()
	}

	// OUT edges: src_symbol_id == rootID (CTE recursivo)
	if direction == "out" || direction == "both" {
		outRows, err := database.QueryContext(ctx, dialect.Bind(outEdgesCTE()),
			rootID, repository, depth, repository)
		if err != nil {
			return nil, fmt.Errorf("query out edges: %w", err)
		}
		if err := appendRows(outRows); err != nil {
			return nil, err
		}
	}

	// IN edges: dst_symbol_id == rootID (CTE recursivo reverso)
	if direction == "in" || direction == "both" {
		inRows, err := database.QueryContext(ctx, dialect.Bind(inEdgesCTE()),
			rootID, repository, depth, repository)
		if err != nil {
			return nil, fmt.Errorf("query in edges: %w", err)
		}
		if err := appendRows(inRows); err != nil {
			return nil, err
		}
	}

	return rows, nil
}

// outEdgesCTE devolve a CTE recursiva de saída (root → dst).
// Esquema: direction | kind | symbol | file_path | start_line | hop
//
// Parâmetros: src_symbol_id, repository, depth, repository.
func outEdgesCTE() string {
	return `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1
      FROM code_edges
     WHERE src_symbol_id = ? AND repository = ?
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1
      FROM code_edges e
      JOIN code_walk w ON e.src_symbol_id = w.dst
     WHERE w.hop < ? AND e.repository = ?
)
SELECT 'out' AS direction,
       w.kind,
       COALESCE(s.qualified_name, s.name) AS symbol,
       f.path AS file_path,
       w.line,
       w.hop
  FROM code_walk w
  JOIN code_symbols s ON s.id = w.dst
  JOIN code_files f   ON f.id = w.file_id
 ORDER BY w.hop ASC, w.kind ASC, symbol ASC
`
}

// inEdgesCTE devolve a CTE recursiva de entrada (src → root).
// Parâmetros: dst_symbol_id, repository, depth, repository.
func inEdgesCTE() string {
	return `
WITH RECURSIVE code_walk(src, dst, kind, file_id, line, hop) AS (
    SELECT src_symbol_id, dst_symbol_id, kind, file_id, start_line, 1
      FROM code_edges
     WHERE dst_symbol_id = ? AND repository = ?
    UNION ALL
    SELECT e.src_symbol_id, e.dst_symbol_id, e.kind, e.file_id, e.start_line, w.hop + 1
      FROM code_edges e
      JOIN code_walk w ON e.dst_symbol_id = w.src
     WHERE w.hop < ? AND e.repository = ?
)
SELECT 'in' AS direction,
       w.kind,
       COALESCE(s.qualified_name, s.name) AS symbol,
       f.path AS file_path,
       w.line,
       w.hop
  FROM code_walk w
  JOIN code_symbols s ON s.id = w.src
  JOIN code_files f   ON f.id = w.file_id
 ORDER BY w.hop ASC, w.kind ASC, symbol ASC
`
}

func printCodeGraphTable(root string, rows []CodeGraphRow) {
	fmt.Printf("# Code Graph: %s\n", root)
	if len(rows) == 0 {
		fmt.Println("_(sem vizinhos)_")
		return
	}
	fmt.Println("| direction | hop | kind | symbol | file | line |")
	fmt.Println("| :--- | :---: | :--- | :--- | :--- | :---: |")
	for _, r := range rows {
		fmt.Printf("| %s | %d | %s | %s | %s | %d |\n",
			r.Direction, r.Hop, r.Kind, r.Symbol, r.FilePath, r.Line)
	}
}
