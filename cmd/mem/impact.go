package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runImpactCLI executa o subcomando mem impact direcionando para stdout
func runImpactCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runImpactCommand(ctx, defaultRepo, args, os.Stdout)
}

// runImpactCommand executa a análise de impacto permitindo injeção de io.Writer para testes
func runImpactCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	impactCmd := flag.NewFlagSet("impact", flag.ContinueOnError)
	impactCmd.SetOutput(out)

	depth := impactCmd.Int("depth", 2, "Profundidade máxima de dependências reversas (padrão: 2)")
	jsonOutput := impactCmd.Bool("json", false, "Exibe o resultado em formato JSON estruturado")
	dbPath := impactCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := impactCmd.String("postgres", "", "URL de conexão PostgreSQL")
	storage := impactCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
	targetRepo := impactCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := impactCmd.Parse(rearrangeImpactArgs(args)); err != nil {
		return err
	}

	remaining := impactCmd.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("identificador do nó alvo é obrigatório (ex: mem impact <node_id>)")
	}
	targetNode := remaining[0]

	resolvedDepth := *depth
	if resolvedDepth <= 0 {
		resolvedDepth = 2
	}

	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
	var errOverride error
	resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
	if errOverride != nil {
		return errOverride
	}

	var impactResult *graph.ImpactResult
	var impactErr error

	if resolvedPG != "" && (*pgURL != "" || (cfg != nil && cfg.Storage.Engine == "postgres")) {
		pgStore, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()

		impactResult, impactErr = pgStore.CalculateImpact(ctx, resolvedRepo, targetNode, resolvedDepth)
	} else {
		database, err := db.InitDB(resolvedDB)
		if err != nil {
			return fmt.Errorf("falha ao abrir banco SQLite '%s': %w", resolvedDB, err)
		}
		defer database.Close()

		impactResult, impactErr = db.CalculateImpactForTarget(ctx, database, targetNode, resolvedDepth)
	}

	if impactErr != nil {
		return fmt.Errorf("falha na análise de impacto: %w", impactErr)
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(impactResult)
	}

	// Renderização Tabular para terminal
	fmt.Fprintf(out, "\n=== Análise de Impacto (Blast Radius) ===\n")
	fmt.Fprintf(out, "Nó Alvo:            %s\n", impactResult.TargetNode)
	fmt.Fprintf(out, "Total Impactado:    %d nós (Diretos: %d, Indiretos: %d)\n",
		impactResult.TotalImpacted, impactResult.DirectDependents, impactResult.IndirectDependents)
	fmt.Fprintf(out, "Profundidade Máx:   %d\n", impactResult.MaxDepthReached)
	fmt.Fprintf(out, "Score de Risco:     %.1f / 100 [%s]\n", impactResult.RiskScore, impactResult.RiskLevel)

	if len(impactResult.AffectedClusters) > 0 {
		clustersStr := make([]string, len(impactResult.AffectedClusters))
		for i, c := range impactResult.AffectedClusters {
			clustersStr[i] = fmt.Sprintf("Cluster %d", c)
		}
		fmt.Fprintf(out, "Clusters Afetados:  %s\n", strings.Join(clustersStr, ", "))
	}

	fmt.Fprintf(out, "Severidades:        CRÍTICO: %d | ALTO: %d | MÉDIO: %d | BAIXO: %d\n\n",
		impactResult.SeverityCounts[graph.SeverityCritical],
		impactResult.SeverityCounts[graph.SeverityHigh],
		impactResult.SeverityCounts[graph.SeverityMedium],
		impactResult.SeverityCounts[graph.SeverityLow],
	)

	if len(impactResult.Nodes) == 0 {
		fmt.Fprintf(out, "Nenhum nó dependente identificado dentro da profundidade %d.\n", resolvedDepth)
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PROFUNDIDADE\tSEVERIDADE\tRELAÇÃO\tVIA NÓ\tTIPO\tID AFETADO")
	fmt.Fprintln(w, "------------\t----------\t-------\t------\t----\t----------")

	for _, n := range impactResult.Nodes {
		sevBadge := fmt.Sprintf("[%s]", n.Severity)
		via := n.ViaNode
		if via == impactResult.TargetNode {
			via = "(direto)"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			n.Depth,
			sevBadge,
			n.Relation,
			via,
			n.NodeType,
			n.ID,
		)
	}
	w.Flush()
	fmt.Fprintln(out)

	return nil
}

func rearrangeImpactArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "--depth" || arg == "-depth" ||
				arg == "--db" || arg == "-db" || arg == "--postgres" || arg == "-postgres" ||
				arg == "--repo" || arg == "-repo") {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			nonFlags = append(nonFlags, arg)
		}
	}
	return append(flags, nonFlags...)
}
