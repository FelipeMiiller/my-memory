package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runCodeIndexCLI implementa `mem code-index` (CA-06 / ADR-047).
//
// Aceita flags: --lang=csv --include=glob --exclude=glob --ast-hash --json
// --storage=sqlite|postgres --repo=<slug> --dir=<vault> --db=<arquivo>.
// Por padrão usa o vault configurado em .memory/config.yaml (ADR-016).
//
// Nenhuma flag é aceita e descartada (CG-09): cada uma muda o comportamento
// ou não existe.
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
	astHash := fs.Bool("ast-hash", true, "Armazena o ast_hash do arquivo (--ast-hash=false não armazena)")
	jsonOut := fs.Bool("json", false, "Saída em JSON: um objeto com as contagens do code stage")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Boot log one-time (CA-14). Vai para o stderr: o stdout do relatório
	// --json tem que ser JSON puro (CG-09).
	bootLogToStderr()

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
		fmt.Println("Uso: mem code-index [--lang=go,py,...] [--include=glob] [--exclude=glob] [--ast-hash[=false]] [--json] [--storage=sqlite|postgres] [--db=<arq>] [--repo=<slug>] [<dir>]")
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

	// CG-09: o --repo vale de verdade — o code graph é gravado no repositório
	// pedido (o repo_id viaja no contexto até as tabelas code_*).
	repository := strings.TrimSpace(*repoSlug)
	if repository == "" {
		repository = strings.TrimSpace(defaultRepo)
	}
	ctx = db.WithRepository(ctx, repository)
	// --ast-hash=false: o code_files guarda ast_hash NULL.
	ctx = db.WithAstHash(ctx, *astHash)

	// O code graph é gravado no engine configurado (ADR-061): Postgres quando
	// Postgres, SQLite quando SQLite. Nada de SQLite local paralelo.
	engine, err := resolveCodeEngine(cfg, *storage, *pgURL, *dbPath)
	if err != nil {
		return err
	}
	defer engine.Close()

	stats, err := runCodeIndexPipeline(ctx, engine.db, target, cfg, langs)
	if err != nil {
		return err
	}

	if *jsonOut {
		report := codeIndexReport{
			Engine:       engine.name,
			Repository:   repository,
			FilesScanned: stats.FilesScanned,
			FilesIndexed: stats.FilesIndexed,
			FilesSkipped: stats.FilesSkipped,
			FilesErrors:  stats.FilesErrors,
			SymbolsTotal: stats.SymbolsTotal,
		}
		out, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("serialização do relatório JSON falhou: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Println("[code-index] OK")
	fmt.Printf("  engine        : %s\n", engine.name)
	fmt.Printf("  repository    : %s\n", repository)
	fmt.Printf("  files scanned : %d\n", stats.FilesScanned)
	fmt.Printf("  files indexed : %d\n", stats.FilesIndexed)
	fmt.Printf("  files skipped : %d\n", stats.FilesSkipped)
	fmt.Printf("  files errors  : %d\n", stats.FilesErrors)
	fmt.Printf("  symbols total : %d\n", stats.SymbolsTotal)
	return nil
}

// bootLogToStderr executa o boot log do code pipeline com o stdout
// apontado para o stderr.
//
// O boot log é um aviso de diagnóstico (CA-14), não parte do relatório: com
// `--json` o stdout precisa ser JSON puro, senão o cliente não consegue
// parsear a saída (CG-09).
func bootLogToStderr() {
	original := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = original }()
	codeast.LogTreesitterBootStatus()
}

// codeIndexReport é a saída estruturada do code stage (--json).
type codeIndexReport struct {
	Engine       string `json:"engine"`
	Repository   string `json:"repository"`
	FilesScanned int    `json:"files_scanned"`
	FilesIndexed int    `json:"files_indexed"`
	FilesSkipped int    `json:"files_skipped"`
	FilesErrors  int    `json:"files_errors"`
	SymbolsTotal int    `json:"symbols_total"`
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

// runCodeIndexPipeline roda o code pipeline contra a conexão já resolvida e
// devolve as contagens do estágio.
func runCodeIndexPipeline(ctx context.Context, database *sql.DB, dir string, cfg *config.Config, langs []string) (*codeast.PipelineStats, error) {
	if database == nil {
		return nil, errors.New("code index: database == nil")
	}

	pipe, err := codeast.NewPipeline(codeast.PipelineOptions{
		DB:        database,
		LangAllow: langs,
	})
	if err != nil {
		return nil, fmt.Errorf("NewPipeline falhou: %w", err)
	}
	if err := pipe.Run(ctx, dir, cfg); err != nil {
		return nil, fmt.Errorf("Pipeline.Run falhou: %w", err)
	}
	return pipe.Stats(), nil
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
