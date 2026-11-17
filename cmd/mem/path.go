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
	"text/tabwriter"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runPathCLI executa o subcomando mem path direcionando para stdout
func runPathCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runPathCommand(ctx, defaultRepo, args, os.Stdout)
}

// runPathCommand executa a descoberta de rota permitindo injeção de io.Writer para testes
func runPathCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	defaultOpts := graph.DefaultPathOptions()

	pathCmd := flag.NewFlagSet("path", flag.ContinueOnError)
	pathCmd.SetOutput(out)

	maxDepth := pathCmd.Int("max-depth", defaultOpts.MaxDepth, fmt.Sprintf("Profundidade máxima de saltos a explorar (padrão: %d)", defaultOpts.MaxDepth))
	depthAlias := pathCmd.Int("depth", defaultOpts.MaxDepth, "Alias para --max-depth")
	directed := pathCmd.Bool("directed", defaultOpts.Directed, fmt.Sprintf("Navegação respeitando o sentido das arestas (padrão: %t)", defaultOpts.Directed))
	undirected := pathCmd.Bool("undirected", false, "Navegação bidirecional ignorando o sentido das arestas")
	modeStr := pathCmd.String("mode", string(defaultOpts.CostMode), "Modo de custo ('epistemic' ponderado ou 'hops' uniforme)")
	jsonOutput := pathCmd.Bool("json", false, "Exibe o resultado em formato JSON estruturado")
	dbPath := pathCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := pathCmd.String("postgres", "", "URL de conexão PostgreSQL")
	storage := pathCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
	targetRepo := pathCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := pathCmd.Parse(rearrangePathArgs(args)); err != nil {
		return err
	}

	remaining := pathCmd.Args()
	if len(remaining) < 2 {
		return fmt.Errorf("origem e destino são obrigatórios (ex: mem path <source> <target>)")
	}
	sourceNode := remaining[0]
	targetNode := remaining[1]

	defaultDepthStr := fmt.Sprintf("%d", defaultOpts.MaxDepth)
	resolvedDepth := *maxDepth
	if pathCmd.Lookup("depth").Value.String() != defaultDepthStr && *depthAlias != defaultOpts.MaxDepth {
		resolvedDepth = *depthAlias
	}
	if resolvedDepth <= 0 {
		resolvedDepth = defaultOpts.MaxDepth
	}

	isDirected := *directed
	if *undirected {
		isDirected = false
	}

	costMode := defaultOpts.CostMode
	if strings.ToLower(strings.TrimSpace(*modeStr)) == "hops" {
		costMode = graph.CostModeHops
	}

	opts := graph.PathOptions{
		MaxDepth: resolvedDepth,
		Directed: isDirected,
		CostMode: costMode,
	}

	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
	var errOverride error
	resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
	if errOverride != nil {
		return errOverride
	}

	var pathResult *graph.PathResult
	var pathErr error
	// A conexão fica em escopo de função porque o aviso de staleness (ADR-028)
	// reaproveita a mesma conexão que a consulta abriu.
	var database *sql.DB
	var pgStore *store.PostgresStore

	if resolvedPG != "" && (*pgURL != "" || (cfg != nil && cfg.Storage.Engine == "postgres")) {
		pg, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		defer pg.Close()
		pgStore = pg

		pathResult, pathErr = pgStore.FindPath(ctx, resolvedRepo, sourceNode, targetNode, opts)
	} else {
		sqliteDB, err := db.InitDB(resolvedDB)
		if err != nil {
			return fmt.Errorf("falha ao abrir banco SQLite '%s': %w", resolvedDB, err)
		}
		defer sqliteDB.Close()
		database = sqliteDB

		pathResult, pathErr = db.FindPath(ctx, database, sourceNode, targetNode, opts)
	}

	if pathErr != nil {
		return fmt.Errorf("falha na descoberta de rota: %w", pathErr)
	}

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(pathResult)
	}

	// Aviso de desatualização (ADR-028): sai antes do resultado e é
	// suprimido no --json (ali o stdout tem que continuar sendo só o
	// documento JSON).
	printStalenessBanner(ctx, out, stalenessBannerOptions{
		Config:   cfg,
		Database: database,
		PgStore:  pgStore,
		Repo:     resolvedRepo,
	})

	// Renderização amigável em terminal
	fmt.Fprintf(out, "\n=== Descoberta de Rotas & Menor Caminho ===\n")
	fmt.Fprintf(out, "Origem:          %s\n", pathResult.Source)
	fmt.Fprintf(out, "Destino:         %s\n", pathResult.Target)
	fmt.Fprintf(out, "Modo:            %s\n", pathResult.CostMode)
	dirStr := "Direcionado (A -> B)"
	if !pathResult.Directed {
		dirStr = "Bidirecional / Não-direcionado (A <-> B)"
	}
	fmt.Fprintf(out, "Direcionamento:  %s\n", dirStr)
	fmt.Fprintf(out, "Profundidade:    %d saltos máx\n", resolvedDepth)

	// Relatório de ambiguidade (GPR-11 a GPR-14): impresso antes de Status:
	if pathResult.Ambiguity != nil && len(pathResult.Ambiguity.Candidates) > 1 {
		fmt.Fprintf(out, "\n⚠️  Aviso de Ambiguidade:\n%s\n", pathResult.Ambiguity.FormatCLI())
	}

	if !pathResult.Found {
		fmt.Fprintf(out, "\nStatus:          ❌ Não encontrado\n")
		fmt.Fprintf(out, "Detalhes:        %s\n\n", pathResult.Summary)
		return nil
	}

	fmt.Fprintf(out, "Status:          ✅ Encontrado\n")
	fmt.Fprintf(out, "Saltos:          %d\n", pathResult.Hops)
	fmt.Fprintf(out, "Custo Total:     %.3f\n\n", pathResult.TotalCost)

	if pathResult.Hops == 0 {
		fmt.Fprintf(out, "Origem e destino são o mesmo nó ('%s').\n\n", pathResult.Source)
		return nil
	}

	// Cadeia de Nós Visual
	fmt.Fprintf(out, "Rota Conectada:\n  [%s]\n", pathResult.Nodes[0])
	for i, edge := range pathResult.Edges {
		arrow := "──>"
		dirLabel := "forward"
		if edge.Direction == "reverse" {
			arrow = "<──"
			dirLabel = "reverse"
		}
		fmt.Fprintf(out, "   └──(%s [%s | %s] | custo: %.3f)%s \n  [%s]\n",
			edge.Relation, edge.EpistemicStatus, dirLabel, edge.Cost, arrow, pathResult.Nodes[i+1])
	}
	fmt.Fprintln(out)

	// Tabela Detalhada de Segmentos
	fmt.Fprintf(out, "Segmentos Percorridos:\n")
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  Salto\tDe\tPara\tRelação\tStatus\tDireção\tCusto")
	fmt.Fprintln(w, "  -----\t--\t----\t-------\t------\t-------\t-----")
	for i, edge := range pathResult.Edges {
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\t%.3f\n",
			i+1,
			edge.From,
			edge.To,
			edge.Relation,
			edge.EpistemicStatus,
			edge.Direction,
			edge.Cost,
		)
	}
	w.Flush()
	fmt.Fprintln(out)

	return nil
}

func rearrangePathArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "--max-depth" || arg == "-max-depth" ||
				arg == "--depth" || arg == "-depth" || arg == "--mode" || arg == "-mode" ||
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
