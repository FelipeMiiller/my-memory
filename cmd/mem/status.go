package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runStatusCLI executa o subcomando mem status direcionando para stdout
func runStatusCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runStatusCommand(ctx, defaultRepo, args, os.Stdout)
}

// runStatusCommand executa a verificação de integridade e sincronização permitindo injeção de out para testes
func runStatusCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	statusCmd := flag.NewFlagSet("status", flag.ContinueOnError)
	statusCmd.SetOutput(out)

	dirPath := statusCmd.String("dir", "", "Caminho da pasta do vault de notas")
	jsonOutput := statusCmd.Bool("json", false, "Exibe o relatório em formato JSON estruturado")
	dbPath := statusCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := statusCmd.String("postgres", "", "URL de conexão PostgreSQL")
	targetRepo := statusCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := statusCmd.Parse(args); err != nil {
		return err
	}

	targetDir := *dirPath
	if targetDir == "" && statusCmd.NArg() > 0 {
		targetDir = statusCmd.Arg(0)
	}

	searchDir := targetDir
	if searchDir == "" {
		searchDir = "."
	}

	var cfg *config.Config
	cfgPath, err := config.FindConfigFile(searchDir)
	if err == nil {
		_, _ = config.FindAndLoadDotEnv(searchDir)
		if loaded, loadErr := config.LoadConfig(cfgPath); loadErr == nil {
			cfg = loaded
			if targetDir == "" {
				dirOfCfg := filepath.Dir(cfgPath)
				if filepath.Base(dirOfCfg) == ".memory" {
					targetDir = filepath.Dir(dirOfCfg)
				} else {
					targetDir = dirOfCfg
				}
			}
		}
	}

	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}
	if targetDir == "" {
		targetDir = "."
	}

	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

	var detector *staleness.Detector
	if resolvedPG != "" && (*pgURL != "" || (cfg != nil && cfg.Storage.Engine == "postgres")) {
		pgStore, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()
		detector = staleness.NewDetector(targetDir, cfg, nil, pgStore, resolvedRepo)
	} else {
		database, err := db.InitDB(resolvedDB)
		if err != nil {
			return fmt.Errorf("falha ao abrir banco SQLite '%s': %w", resolvedDB, err)
		}
		defer database.Close()
		detector = staleness.NewDetector(targetDir, cfg, database, nil, "")
	}

	report, err := detector.ForceCheck(ctx)
	if err != nil {
		return fmt.Errorf("falha ao verificar integridade do vault: %w", err)
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	absDir, _ := filepath.Abs(targetDir)
	fmt.Fprintf(out, "\n=== Status de Integridade e Sincronização do Vault ===\n")
	fmt.Fprintf(out, "Diretório do Vault: %s\n", absDir)
	if resolvedRepo != "" {
		fmt.Fprintf(out, "Repositório:        %s\n", resolvedRepo)
	}
	fmt.Fprintf(out, "Arquivos no Disco:  %d\n", report.DiskCount)
	fmt.Fprintf(out, "Arquivos Indexados: %d\n", report.IndexedCount)

	if !report.IsStale {
		fmt.Fprintf(out, "Status:             ✅ Atualizado (In Sync)\n")
		fmt.Fprintf(out, "Todos os arquivos elegíveis estão devidamente sincronizados com o grafo e embeddings.\n\n")
		return nil
	}

	fmt.Fprintf(out, "Status:             ⚠️  Desatualizado (Stale Data)\n")
	fmt.Fprintf(out, "Diferenças:         %d modificado(s)/novo(s), %d removido(s)\n\n", report.StaleFilesCount, report.DeletedFilesCount)

	if len(report.StaleFiles) > 0 {
		fmt.Fprintf(out, "Arquivos Modificados ou Não-Indexados (%d):\n", len(report.StaleFiles))
		limit := 10
		if len(report.StaleFiles) < limit {
			limit = len(report.StaleFiles)
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(out, "  • %s\n", report.StaleFiles[i])
		}
		if len(report.StaleFiles) > limit {
			fmt.Fprintf(out, "  ... e mais %d arquivo(s)\n", len(report.StaleFiles)-limit)
		}
		fmt.Fprintln(out)
	}

	if len(report.DeletedFiles) > 0 {
		fmt.Fprintf(out, "Arquivos Removidos do Disco (%d):\n", len(report.DeletedFiles))
		limit := 10
		if len(report.DeletedFiles) < limit {
			limit = len(report.DeletedFiles)
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(out, "  • %s\n", report.DeletedFiles[i])
		}
		if len(report.DeletedFiles) > limit {
			fmt.Fprintf(out, "  ... e mais %d arquivo(s)\n", len(report.DeletedFiles)-limit)
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintf(out, "💡 Recomendação: Execute 'mem index' para sincronizar o grafo e embeddings.\n\n")
	return nil
}
