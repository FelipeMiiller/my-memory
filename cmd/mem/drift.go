package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/drift"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runDriftCLI executa o subcomando mem drift direcionando a saída para stdout
func runDriftCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runDriftCommand(ctx, defaultRepo, args, os.Stdout, drift.NewOSGitRunner())
}

// runDriftCommand executa o subcomando 'mem drift'
func runDriftCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer, runner drift.GitRunner) error {
	driftCmd := flag.NewFlagSet("drift", flag.ContinueOnError)
	driftCmd.SetOutput(out)

	sinceFlag := driftCmd.String("since", "HEAD~5..HEAD", "Faixa de commits do Git a analisar (ex: HEAD~5..HEAD ou commit..HEAD)")
	threshFlag := driftCmd.Float64("threshold", 0.20, "Limite mínimo de desvio semântico para exibir (0.0 a 1.0, padrão: 0.20)")
	uncoveredFlag := driftCmd.Bool("uncovered", true, "Exibe arquivos de código modificados sem documentação/ADR correspondente")
	strictFlag := driftCmd.Bool("strict", false, "Falha com código de saída 1 caso existam notas em estado CRÍTICO")
	jsonFlag := driftCmd.Bool("json", false, "Exporta o relatório consolidado em formato JSON")

	dbPath := driftCmd.String("db", "", "Caminho para o banco de dados SQLite")
	pgURL := driftCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
	targetRepo := driftCmd.String("repo", "", "Slug ou identificador do repositório")

	reordered := rearrangeDriftArgs(args)
	if err := driftCmd.Parse(reordered); err != nil {
		return err
	}

	cfg := resolveConfig()
	_, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

	repoRoot, err := os.Getwd()
	if err != nil {
		repoRoot = "."
	}
	vaultName := cfg.ResolveObsidianVault(repoRoot)

	// Conecta ao banco de dados SQLite ou PostgreSQL
	var database *sql.DB
	if resolvedPG != "" {
		pgStore, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()
		database = pgStore.DB()
	} else {
		sqldb, err := db.InitDB(resolvedDB)
		if err != nil {
			return fmt.Errorf("falha ao conectar no SQLite: %w", err)
		}
		defer sqldb.Close()
		database = sqldb
	}

	if runner == nil {
		runner = drift.NewOSGitRunner()
	}

	// Executa análise de desvio de conhecimento
	report, err := drift.AnalyzeDrift(ctx, runner, database, repoRoot, vaultName, *sinceFlag, *threshFlag, *uncoveredFlag)
	if err != nil {
		return fmt.Errorf("falha na análise de desvio semântico: %w", err)
	}

	// 1. Saída em formato JSON
	if *jsonFlag {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("falha ao serializar JSON: %w", err)
		}
		if *strictFlag && report.CriticalCount > 0 {
			return fmt.Errorf("desvio crítico detectado em modo --strict (%d nota(s) em nível CRITICAL)", report.CriticalCount)
		}
		return nil
	}

	// 2. Saída em formato tabular para terminal
	renderDriftTerminal(out, report)

	// 3. Verificação do modo strict
	if *strictFlag && report.CriticalCount > 0 {
		return fmt.Errorf("desvio crítico detectado em modo --strict (%d nota(s) em nível CRITICAL)", report.CriticalCount)
	}

	return nil
}

// renderDriftTerminal renderiza o relatório human-readable no terminal
func renderDriftTerminal(out io.Writer, report *drift.DriftReport) {
	fmt.Fprintf(out, "\n=== 🧭 Análise de Desvio Código-Memória (Semantic Drift) ===\n")
	fmt.Fprintf(out, "Faixa Git Analisada:   %s\n", report.RangeStr)
	fmt.Fprintf(out, "Commits no Intervalo:  %d\n", report.TotalCommits)
	fmt.Fprintf(out, "Arquivos Modificados:  %d\n", report.TotalChanged)
	fmt.Fprintf(out, "Índice Geral de Drift: %.1f / 100\n", report.OverallScore)
	fmt.Fprintf(out, "Severidades:           CRÍTICO: %d | ALTO: %d | MÉDIO: %d | BAIXO: %d\n",
		report.CriticalCount, report.HighCount, report.MediumCount, report.LowCount)
	fmt.Fprintln(out)

	if len(report.DriftedNotes) == 0 {
		fmt.Fprintf(out, "✅ Nenhuma nota com desvio significativo detectada (documentação sincronizada com o código)!\n")
	} else {
		fmt.Fprintf(out, "--- [📑 NOTAS & ADRS DEFASADOS EM RELAÇÃO AO CÓDIGO] (%d) ---\n", len(report.DriftedNotes))
		for i, n := range report.DriftedNotes {
			var icon string
			switch n.Severity {
			case drift.SeverityCritical:
				icon = "🔴 [CRITICAL]"
			case drift.SeverityHigh:
				icon = "🟠 [HIGH]"
			case drift.SeverityMedium:
				icon = "🟡 [MEDIUM]"
			default:
				icon = "🟢 [LOW]"
			}

			fmt.Fprintf(out, "\n[%d] %s Score: %.1f/100 | %s\n", i+1, icon, n.DriftScore, n.Title)
			fmt.Fprintf(out, "    Caminho:     %s\n", n.Path)
			fmt.Fprintf(out, "    Diagnóstico: %s\n", n.Reason)
			if len(n.AffectedFiles) > 0 {
				fmt.Fprintf(out, "    Código(s):   %s\n", strings.Join(n.AffectedFiles, ", "))
			}
			if n.Links != nil {
				fmt.Fprintf(out, "    Obsidian:    %s\n", n.Links.Obsidian)
				fmt.Fprintf(out, "    VS Code:     %s\n", n.Links.VSCode)
			}
		}
		fmt.Fprintln(out)
	}

	if len(report.UncoveredCode) > 0 {
		fmt.Fprintf(out, "--- [⚠️  CÓDIGO ÓRFÃO DE DECISÕES / SEM NOTAS VINCULADAS] (%d) ---\n", len(report.UncoveredCode))
		fmt.Fprintf(out, "%-35s | %-6s | %-12s | %s\n", "ARQUIVO", "STATUS", "MODIFICAÇÃO", "AÇÃO SUGERIDA")
		fmt.Fprintf(out, "%s\n", strings.Repeat("-", 100))
		for _, uc := range report.UncoveredCode {
			diffStr := fmt.Sprintf("+%d/-%d", uc.Additions, uc.Deletions)
			fmt.Fprintf(out, "%-35s | %-6s | %-12s | %s\n", truncateString(uc.FilePath, 35), uc.Status, diffStr, uc.SuggestedAction)
		}
		fmt.Fprintln(out)
	}
}

// rearrangeDriftArgs permite posicionar flags antes ou depois do comando
func rearrangeDriftArgs(args []string) []string {
	var flags []string
	for _, a := range args {
		flags = append(flags, a)
	}
	return flags
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

