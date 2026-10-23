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
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runCodeIndexCLI implementa `mem code-index` (CA-06 / ADR-047).
//
// Aceita flags: --lang=csv --include=glob --exclude=glob --ast-hash --no-embed
// --storage=sqlite|postgres --repo=<slug> --dir=<vault> --db=<arquivo>.
// Por padrão usa o vault configurado em .memory/config.yaml (ADR-016).
func runCodeIndexCLI(ctx context.Context, defaultRepo string, args []string) error {
	fs := flag.NewFlagSet("code-index", flag.ContinueOnError)
	dir := fs.String("dir", "", "Diretório do vault (padrão: auto-detect via .memory/config.yaml)")
	dbPath := fs.String("db", "", "Caminho do arquivo SQLite (padrão: memory.db); implica --storage=sqlite")
	pgURL := fs.String("postgres", "", "URL PostgreSQL com pgvector (alternativa ao SQLite)")
	storage := fs.String("storage", "", "Força engine: 'sqlite' ou 'postgres'")
	repoSlug := fs.String("repo", "", "Identificador/slug do repositório (multi-tenant)")
	lang := fs.String("lang", "", "Lista csv de linguagens permitidas (ex: 'go,python'); vazio = todas")
	include := fs.String("include", "", "Glob include adicional (sobrescreve config)")
	exclude := fs.String("exclude", "", "Glob exclude adicional (sobrescreve config)")
	astHash := fs.Bool("ast-hash", true, "Habilita cálculo e armazenamento de ast_hash (CA-04)")
	noEmbed := fs.Bool("no-embed", true, "Desativa geração de embeddings para code_symbols (placeholder; v1 não embute symbols)")
	jsonOut := fs.Bool("json", false, "Reservada para a saída estruturada (sem efeito nesta versão)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_ = repoSlug
	_ = astHash
	_ = noEmbed
	_ = jsonOut

	// Boot log one-time (CA-14)
	codeast.LogTreesitterBootStatus()

	target := *dir
	if target == "" && fs.NArg() > 0 {
		target = fs.Arg(0)
	}

	cascadeDir := target
	if cascadeDir == "" {
		cascadeDir = "."
	}
	cfg, cfgPath, _ := config.LoadCascadingConfig(cascadeDir)
	_ = cfgPath
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}

	if target == "" {
		fmt.Println("Uso: mem code-index [--lang=go,py,...] [--include=glob] [--exclude=glob] [--ast-hash] [--no-embed] [--storage=sqlite|postgres] [--db=<arq>] [--repo=<slug>] [<dir>]")
		return nil
	}

	// Argumentos CLI sobrescrevem o escopo de código (ADR-061 §4): o include
	// do CLI de código é o escopo de código, não o escopo de notas.
	if *include != "" {
		cfg.Code.Include = []string{*include}
	}
	if *exclude != "" {
		cfg.Code.Exclude = []string{*exclude}
	}
	langs := splitCSV(*lang)

	// O code graph nasce no repositório do vault (ADR-061 — CG-06): o
	// repo_id viaja no contexto até as tabelas code_*.
	ctx = db.WithRepository(ctx, defaultRepo)

	// O code graph é gravado no engine configurado (ADR-061): Postgres quando
	// Postgres, SQLite quando SQLite. Nada de SQLite local paralelo.
	engine, err := resolveCodeEngine(cfg, *storage, *pgURL, *dbPath)
	if err != nil {
		return err
	}
	defer engine.Close()

	return runCodeIndexPipeline(ctx, engine.db, target, cfg, langs)
}

// codeEngine é o engine resolvido para o code stage.
type codeEngine struct {
	name  string // "sqlite" | "postgres"
	db    *sql.DB
	close func()
}

// Close fecha a conexão do engine (idempotente para o chamador).
func (e *codeEngine) Close() {
	if e == nil || e.close == nil {
		return
	}
	e.close()
}

// resolveCodeEngine decide onde o code graph é gravado.
//
// Precedência (ADR-040): --storage=sqlite|postgres vence; depois, um --db
// explícito (que é a forma de pedir o SQLite local); sem nenhum dos dois, vale
// o engine configurado. O caminho --storage=sqlite é o usado em teste/CI.
func resolveCodeEngine(cfg *config.Config, storageFlag, pgURLFlag, dbPathFlag string) (*codeEngine, error) {
	engine := ""
	if cfg != nil {
		engine = strings.ToLower(strings.TrimSpace(cfg.Storage.Engine))
	}
	switch strings.ToLower(strings.TrimSpace(storageFlag)) {
	case "sqlite":
		engine = "sqlite"
	case "postgres":
		engine = "postgres"
	default:
		// --db=<arquivo> nomeia um arquivo SQLite: é a escolha explícita do
		// SQLite local (e impede que o code stage caia no Postgres do vault
		// sem o usuário pedir).
		if strings.TrimSpace(dbPathFlag) != "" {
			engine = "sqlite"
		}
	}
	if engine != "postgres" {
		engine = "sqlite"
	}

	if engine == "postgres" {
		pgURL := strings.TrimSpace(pgURLFlag)
		if pgURL == "" && cfg != nil {
			pgURL = strings.TrimSpace(cfg.Storage.PostgresURL)
		}
		if pgURL == "" {
			return nil, errors.New("engine=postgres sem URL: defina storage.postgres_url, use --postgres=<url> ou force --storage=sqlite")
		}
		pgStore, err := store.NewPostgresStore(pgURL)
		if err != nil {
			return nil, fmt.Errorf("postgres: %w", err)
		}
		return &codeEngine{
			name:  "postgres",
			db:    pgStore.DB(),
			close: func() { _ = pgStore.Close() },
		}, nil
	}

	dbPath := strings.TrimSpace(dbPathFlag)
	if dbPath == "" && cfg != nil {
		dbPath = strings.TrimSpace(cfg.Storage.SQLitePath)
	}
	if dbPath == "" {
		dbPath = "memory.db"
	}
	database, err := db.InitDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("InitDB(%s) falhou: %w", dbPath, err)
	}
	return &codeEngine{
		name:  "sqlite",
		db:    database,
		close: func() { _ = database.Close() },
	}, nil
}

// runCodeIndexPipeline roda o code pipeline contra a conexão já resolvida.
func runCodeIndexPipeline(ctx context.Context, database *sql.DB, dir string, cfg *config.Config, langs []string) error {
	if database == nil {
		return errors.New("code index: database == nil")
	}

	pipe, err := codeast.NewPipeline(codeast.PipelineOptions{
		DB:        database,
		LangAllow: langs,
	})
	if err != nil {
		return fmt.Errorf("NewPipeline falhou: %w", err)
	}
	if err := pipe.Run(ctx, dir, cfg); err != nil {
		return fmt.Errorf("Pipeline.Run falhou: %w", err)
	}

	stats := pipe.Stats()
	fmt.Println("[code-index] OK")
	fmt.Printf("  files scanned : %d\n", stats.FilesScanned)
	fmt.Printf("  files indexed : %d\n", stats.FilesIndexed)
	fmt.Printf("  files skipped : %d\n", stats.FilesSkipped)
	fmt.Printf("  files errors  : %d\n", stats.FilesErrors)
	fmt.Printf("  symbols total : %d\n", stats.SymbolsTotal)
	return nil
}

// splitCSV utilitário local: "go,py" → ["go", "py"].
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
