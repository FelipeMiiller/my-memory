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
	"github.com/FelipeMiiller/my-memory/internal/graphview"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// ClustersResponse estrutura a resposta serializada em JSON para o subcomando mem clusters
type ClustersResponse struct {
	Modularity  float64           `json:"modularity"`
	TotalNodes  int               `json:"total_nodes"`
	TotalEdges  int               `json:"total_edges"`
	MinSize     int               `json:"min_size"`
	Communities []graph.Community `json:"communities"`
}

// runClustersCLI executa o subcomando mem clusters direcionando para stdout
func runClustersCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runClustersCommand(ctx, defaultRepo, args, os.Stdout)
}

// runClustersCommand executa a detecção de clusters permitindo injeção de io.Writer para testes
func runClustersCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	clustersCmd := flag.NewFlagSet("clusters", flag.ContinueOnError)
	clustersCmd.SetOutput(out)

	minSize := clustersCmd.Int("min-size", 2, "Tamanho mínimo de nós para exibir um cluster (padrão: 2)")
	jsonOutput := clustersCmd.Bool("json", false, "Exibe o resultado em formato JSON estruturado")
	dbPath := clustersCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := clustersCmd.String("postgres", "", "URL de conexão PostgreSQL")
	targetRepo := clustersCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := clustersCmd.Parse(args); err != nil {
		return err
	}

	threshold := *minSize
	if threshold <= 0 {
		threshold = 1
	}

	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)

	var gv *graphview.GraphView
	var err error

	if resolvedPG != "" {
		pgStore, pgErr := store.NewPostgresStore(resolvedPG)
		if pgErr != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", pgErr)
		}
		defer pgStore.Close()

		gv, err = graphview.BuildFromPostgres(ctx, pgStore, resolvedRepo, "", 0)
		if err != nil {
			return fmt.Errorf("falha ao extrair clusters do PostgreSQL: %w", err)
		}
	} else {
		database, dbErr := db.InitDB(resolvedDB)
		if dbErr != nil {
			// Degrada graciosamente para grafo vazio em ambientes sem SQLite configurado
			gv = graphview.BuildGraphView(nil, nil, "", 0, resolvedRepo)
		} else {
			defer database.Close()
			gv, err = graphview.BuildFromSQLite(ctx, database, "", 0, resolvedRepo)
			if err != nil {
				return fmt.Errorf("falha ao extrair clusters do SQLite: %w", err)
			}
		}
	}

	var filtered []graph.Community
	for _, c := range gv.Communities {
		if c.Size >= threshold {
			filtered = append(filtered, c)
		}
	}
	if filtered == nil {
		filtered = []graph.Community{}
	}

	if *jsonOutput {
		return formatClustersJSON(filtered, gv.Stats, threshold, out)
	}

	return formatClustersTable(filtered, gv.Stats, threshold, out)
}

// formatClustersJSON serializa os clusters em JSON indentado
func formatClustersJSON(filtered []graph.Community, stats graphview.GraphStats, threshold int, out io.Writer) error {
	resp := ClustersResponse{
		Modularity:  stats.Modularity,
		TotalNodes:  stats.TotalNodes,
		TotalEdges:  stats.TotalEdges,
		MinSize:     threshold,
		Communities: filtered,
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// formatClustersTable renderiza a tabela alinhada para o terminal
func formatClustersTable(filtered []graph.Community, stats graphview.GraphStats, threshold int, out io.Writer) error {
	fmt.Fprintf(out, "=== Clusters e Comunidades no Grafo (Newman-Girvan Q: %.3f) ===\n", stats.Modularity)
	fmt.Fprintf(out, "Total de nós: %d | Total de arestas: %d | Clusters (tamanho >= %d): %d\n\n",
		stats.TotalNodes, stats.TotalEdges, threshold, len(filtered))

	if len(filtered) == 0 {
		fmt.Fprintf(out, "Nenhum cluster com tamanho >= %d encontrado. (Dica: tente executar com --min-size 1)\n", threshold)
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tLÍDER\tTAMANHO\tTIPO DOMINANTE\tMEMBROS")
	for _, c := range filtered {
		membersStr := strings.Join(c.Members, ", ")
		if len(membersStr) > 50 {
			membersStr = membersStr[:47] + "..."
		}
		domType := c.DominantType
		if domType == "" {
			domType = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\n", c.ID, c.LeadNode, c.Size, domType, membersStr)
	}
	return w.Flush()
}
