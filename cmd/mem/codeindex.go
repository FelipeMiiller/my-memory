package main

import (
	"context"
	"database/sql"
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
// Aceita flags: --lang=csv --include=glob --exclude=glob --ast-hash --no-embed
// --storage=sqlite|postgres --repo=<slug> --dir=<vault> --db=<arquivo>.
// Por padrão usa o vault configurado em .memory/config.yaml (ADR-016).
func runCodeIndexCLI(ctx context.Context, defaultRepo string, args []string) error {
	fs := flag.NewFlagSet("code-index", flag.ContinueOnError)
	dir := fs.String("dir", "", "Diretório do vault (padrão: auto-detect via .memory/config.yaml)")
	dbPath := fs.String("db", "", "Caminho do arquivo SQLite (padrão: memory.db)")
	pgURL := fs.String("postgres", "", "URL PostgreSQL com pgvector (alternativa ao SQLite)")
	storage := fs.String("storage", "", "Força engine: 'sqlite' ou 'postgres'")
	repoSlug := fs.String("repo", "", "Identificador/slug do repositório (multi-tenant)")
	lang := fs.String("lang", "", "Lista csv de linguagens permitidas (ex: 'go,python'); vazio = todas")
	include := fs.String("include", "", "Glob include adicional (sobrescreve config)")
	exclude := fs.String("exclude", "", "Glob exclude adicional (sobrescreve config)")
	astHash := fs.Bool("ast-hash", true, "Habilita cálculo e armazenamento de ast_hash (CA-04)")
	noEmbed := fs.Bool("no-embed", true, "Desativa geração de embeddings para code_symbols (placeholder; v1 não embute symbols)")
	jsonOut := fs.Bool("json", false, "Saída em JSON (placeholder)")
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

	// Argumentos CLI sobrescrevem config (--include / --exclude / --lang)
	if *include != "" {
		cfg.Include = []string{*include}
	}
	if *exclude != "" {
		cfg.Exclude = []string{*exclude}
	}
	langs := splitCSV(*lang)

	resolvedDB := *dbPath
	if resolvedDB == "" {
		resolvedDB = cfg.Storage.SQLitePath
	}

	resolvedPG := *pgURL
	if *storage == "postgres" || (cfg.Storage.Engine == "postgres" && *storage != "sqlite") {
		if resolvedPG == "" {
			resolvedPG = cfg.Storage.PostgresURL
		}
		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				return fmt.Errorf("postgres: %w", err)
			}
			defer pgStore.Close()
			fmt.Println("[code-index] engine=postgres detectado — code_* tabelas via SQLite local (v1)")
		}
	}

	return runCodeIndexPipeline(ctx, resolvedDB, target, cfg, langs)
}

func runCodeIndexPipeline(ctx context.Context, dbPath, dir string, cfg *config.Config, langs []string) error {
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("InitDB(%s) falhou: %w", dbPath, err)
	}
	defer database.Close()

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

// reachableForCI garante compilação de imports usados em tools estáticas.
var _ = (*sql.DB)(nil)
var _ = os.Exit
