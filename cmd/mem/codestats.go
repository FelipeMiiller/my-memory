package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"sort"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
	"github.com/FelipeMiiller/my-memory/internal/db"
)

// runCodeStatsCLI implementa `mem code-stats` (CA-09 / ADR-047).
//
// Agrega contagens de code_files / code_symbols e imprime:
//   - distribuição por linguagem (top 20)
//   - distribuição por symbol kind
//   - top orphan files (code_files sem symbols extraídos)
//
// Flags:
//
//	--db=<arq>       caminho do SQLite (padrão memory.db)
//	--top=N          limite de linhas por seção (padrão 20)
//	--json           saída em JSON estruturado
//	--repo=<slug>    slug multi-tenant (placeholder)
func runCodeStatsCLI(ctx context.Context, defaultRepo string, args []string) error {
	codeast.LogTreesitterBootStatus() // CA-14: boot log uniforme em todos os code-*
	fs := flag.NewFlagSet("code-stats", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Caminho do arquivo SQLite (padrão: memory.db)")
	repoSlug := fs.String("repo", "", "Identificador/slug do repositório")
	top := fs.Int("top", 20, "Limite de linhas por seção (padrão: 20)")
	jsonOut := fs.Bool("json", false, "Saída em JSON estruturado")
	if err := fs.Parse(args); err != nil {
		return err
	}

	_ = defaultRepo
	_ = repoSlug

	target := *dbPath
	if target == "" {
		target = "memory.db"
	}

	database, err := db.InitDB(target)
	if err != nil {
		return fmt.Errorf("InitDB(%s) falhou: %w", target, err)
	}
	defer database.Close()

	stats, err := runCodeStatsQuery(ctx, database, *top)
	if err != nil {
		return err
	}

	if *jsonOut {
		return printCodeStatsJSON(stats)
	}
	printCodeStatsTable(stats)
	return nil
}

// CodeStats agrupa todas as contagens expostas pelo CLI e reusadas em testes.
type CodeStats struct {
	FilesTotal      int64                `json:"files_total"`
	SymbolsTotal    int64                `json:"symbols_total"`
	EdgesTotal      int64                `json:"edges_total"`
	Languages       []db.LanguageCount   `json:"languages"`
	SymbolKinds     []db.SymbolKindCount `json:"symbol_kinds"`
	OrphanFilesTopN []db.OrphanFile      `json:"orphan_files,omitempty"`
}

// db.OrphanFile is added in code_migration.go to keep coupling minimal.

// runCodeStatsQuery monta CodeStats a partir do DB.
func runCodeStatsQuery(ctx context.Context, database *sql.DB, top int) (*CodeStats, error) {
	if database == nil {
		return nil, errors.New("runCodeStatsQuery: database == nil")
	}
	if top <= 0 {
		top = 20
	}

	stats := &CodeStats{}

	if n, err := db.CountCodeFiles(ctx, database); err != nil {
		return nil, fmt.Errorf("CountCodeFiles: %w", err)
	} else {
		stats.FilesTotal = n
	}
	if n, err := db.CountCodeSymbols(ctx, database); err != nil {
		return nil, fmt.Errorf("CountCodeSymbols: %w", err)
	} else {
		stats.SymbolsTotal = n
	}
	if n, err := db.CountCodeEdges(ctx, database); err != nil {
		return nil, fmt.Errorf("CountCodeEdges: %w", err)
	} else {
		stats.EdgesTotal = n
	}

	langs, err := db.ListLanguageDistribution(ctx, database, top)
	if err != nil {
		return nil, fmt.Errorf("ListLanguageDistribution: %w", err)
	}
	stats.Languages = langs

	kinds, err := db.ListSymbolKindDistribution(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("ListSymbolKindDistribution: %w", err)
	}
	stats.SymbolKinds = kinds

	orphanPaths, err := db.ListOrphanFiles(ctx, database, top)
	if err != nil {
		return nil, fmt.Errorf("ListOrphanFiles: %w", err)
	}
	stats.OrphanFilesTopN = make([]db.OrphanFile, 0, len(orphanPaths))
	for _, p := range orphanPaths {
		stats.OrphanFilesTopN = append(stats.OrphanFilesTopN, db.OrphanFile{Path: p})
	}
	return stats, nil
}

func printCodeStatsTable(s *CodeStats) {
	fmt.Println("# Code Stats")
	fmt.Println()
	fmt.Printf("files_total   : %d\n", s.FilesTotal)
	fmt.Printf("symbols_total : %d\n", s.SymbolsTotal)
	fmt.Printf("edges_total   : %d\n", s.EdgesTotal)
	fmt.Println()

	fmt.Println("## Languages")
	if len(s.Languages) == 0 {
		fmt.Println("_(sem dados)_")
	} else {
		fmt.Println("| language | count |")
		fmt.Println("| :--- | :---: |")
		for _, l := range s.Languages {
			fmt.Printf("| %s | %d |\n", l.Language, l.Count)
		}
	}
	fmt.Println()

	fmt.Println("## Symbol Kinds")
	if len(s.SymbolKinds) == 0 {
		fmt.Println("_(sem dados)_")
	} else {
		fmt.Println("| kind | count |")
		fmt.Println("| :--- | :---: |")
		for _, k := range s.SymbolKinds {
			fmt.Printf("| %s | %d |\n", k.Kind, k.Count)
		}
	}
	fmt.Println()

	fmt.Println("## Orphan files (sem symbols)")
	if len(s.OrphanFilesTopN) == 0 {
		fmt.Println("_(nenhum)_")
	} else {
		// Ordena alfabeticamente para output estável.
		sorted := make([]string, len(s.OrphanFilesTopN))
		for i, o := range s.OrphanFilesTopN {
			sorted[i] = o.Path
		}
		sort.Strings(sorted)
		for _, p := range sorted {
			fmt.Printf("- %s\n", p)
		}
	}
}

// printCodeStatsJSON usa encoding/json via fmt para simplicidade.
func printCodeStatsJSON(s *CodeStats) error {
	// Marshal manual para evitar import extra.
	fmt.Println("{")
	fmt.Printf(`  "files_total": %d,`+"\n", s.FilesTotal)
	fmt.Printf(`  "symbols_total": %d,`+"\n", s.SymbolsTotal)
	fmt.Printf(`  "edges_total": %d,`+"\n", s.EdgesTotal)
	fmt.Println(`  "languages": [`)
	for i, l := range s.Languages {
		if i > 0 {
			fmt.Println(",")
		}
		fmt.Printf(`    {"language": %q, "count": %d}`, l.Language, l.Count)
	}
	fmt.Println("\n  ],")
	fmt.Println(`  "symbol_kinds": [`)
	for i, k := range s.SymbolKinds {
		if i > 0 {
			fmt.Println(",")
		}
		fmt.Printf(`    {"kind": %q, "count": %d}`, k.Kind, k.Count)
	}
	fmt.Println("\n  ],")
	fmt.Println(`  "orphan_files": [`)
	for i, o := range s.OrphanFilesTopN {
		if i > 0 {
			fmt.Println(",")
		}
		fmt.Printf(`    %q`, o.Path)
	}
	fmt.Println("\n  ]")
	fmt.Println("}")
	return nil
}
