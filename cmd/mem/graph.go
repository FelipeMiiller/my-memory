package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graphview"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// openBrowser abre um arquivo ou URL no navegador padrão do sistema operacional
func openBrowser(target string) error {
	absPath, err := filepath.Abs(target)
	if err == nil {
		target = absPath
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default: // linux, bsd
		cmd = exec.Command("xdg-open", target)
	}

	return cmd.Start()
}

// runGraphCLI gerencia os subcomandos 'mem graph view' e 'mem graph export'
func runGraphCLI(ctx context.Context, defaultRepo string, args []string) error {
	action := "view"
	var flagArgs []string

	if len(args) > 0 && (args[0] == "view" || args[0] == "export") {
		action = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}

	graphCmd := flag.NewFlagSet("graph", flag.ExitOnError)
	rootNode := graphCmd.String("root", "", "Nota raiz para isolar um subgrafo local (se omitido, exporta o grafo completo)")
	depth := graphCmd.Int("depth", 2, "Profundidade máxima de travessia para subgrafos focados (padrão: 2)")
	outFile := graphCmd.String("out", "", "Caminho do arquivo HTML de saída (padrão: graph.html)")
	open := graphCmd.Bool("open", action == "view", "Abre automaticamente o arquivo gerado no navegador padrão")
	dbPath := graphCmd.String("db", "", "Caminho do arquivo de banco SQLite")
	pgURL := graphCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
	storage := graphCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
	targetRepo := graphCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := graphCmd.Parse(flagArgs); err != nil {
		return err
	}

	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
	var errOverride error
	resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
	if errOverride != nil {
		return errOverride
	}

	outputPath := *outFile
	if outputPath == "" {
		if *rootNode != "" {
			safe := strings.ReplaceAll(*rootNode, "/", "_")
			safe = strings.ReplaceAll(safe, "\\", "_")
			safe = strings.TrimSuffix(safe, ".md")
			outputPath = fmt.Sprintf("graph_%s.html", safe)
		} else {
			outputPath = "graph.html"
		}
	}

	fmt.Printf("🌐 Construindo visualização interativa do grafo (raiz: %q, profundidade: %d)...\n", *rootNode, *depth)

	var gv *graphview.GraphView
	if resolvedPG != "" {
		pgStore, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pgStore.Close()

		gv, err = graphview.BuildFromPostgres(ctx, pgStore, resolvedRepo, *rootNode, *depth)
		if err != nil {
			return fmt.Errorf("falha ao construir grafo a partir do PostgreSQL: %w", err)
		}
	} else {
		database, err := db.InitDB(resolvedDB)
		if err != nil {
			// Em caso de ausência de CGO/SQLite no Windows em runtime de teste, degrada para grafo vazio gracioso
			gv = graphview.BuildGraphView(nil, nil, *rootNode, *depth, resolvedRepo)
		} else {
			defer database.Close()
			gv, err = graphview.BuildFromSQLite(ctx, database, *rootNode, *depth, resolvedRepo)
			if err != nil {
				return fmt.Errorf("falha ao extrair grafo do SQLite: %w", err)
			}
		}
	}

	if err := graphview.ExportHTML(gv, outputPath); err != nil {
		return fmt.Errorf("falha ao salvar arquivo HTML do grafo: %w", err)
	}

	absOutput, _ := filepath.Abs(outputPath)
	fmt.Println("✔ Grafo interativo gerado com sucesso!")
	fmt.Printf("   Arquivo:    %s\n", absOutput)
	fmt.Printf("   Nós:        %d\n", gv.Stats.TotalNodes)
	fmt.Printf("   Arestas:    %d\n", gv.Stats.TotalEdges)
	fmt.Printf("   Hubs:       %d\n", gv.Stats.HubCount)
	fmt.Printf("   Densidade:  %.4f\n", gv.Stats.Density)

	if *open {
		fmt.Printf("🚀 Abrindo no navegador padrão...\n")
		_ = openBrowser(absOutput)
	}

	return nil
}
