package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runPackCLI executa o subcomando mem pack direcionando a saída para stdout
func runPackCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runPackCommand(ctx, defaultRepo, args, os.Stdout)
}

// runPackCommand executa o empacotamento de subgrafo permitindo injeção de io.Writer para testes
func runPackCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	packCmd := flag.NewFlagSet("pack", flag.ContinueOnError)
	packCmd.SetOutput(out)

	depth := packCmd.Int("depth", 2, "Profundidade máxima de saltos na expansão BFS (padrão: 2)")
	maxDepthAlias := packCmd.Int("max-depth", 2, "Alias para --depth")
	maxTokens := packCmd.Int("max-tokens", 4000, "Orçamento máximo de tokens para o pacote de contexto (padrão: 4000)")
	direction := packCmd.String("direction", "both", "Direção da navegação ('both', 'outbound', 'inbound')")
	outFile := packCmd.String("out", "", "Caminho do arquivo Markdown para gravar o pacote gerado")
	noAbstracts := packCmd.Bool("no-abstracts", false, "Desativa a inclusão de micro-abstracts L0/L1 para nós periféricos")
	jsonOutput := packCmd.Bool("json", false, "Exibe o resultado em formato JSON estruturado")
	dbPath := packCmd.String("db", "", "Caminho do arquivo SQLite")
	pgURL := packCmd.String("postgres", "", "URL de conexão PostgreSQL")
	storage := packCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
	targetRepo := packCmd.String("repo", "", "Slug ou identificador do repositório")

	if err := packCmd.Parse(rearrangePackArgs(args)); err != nil {
		return err
	}

	remaining := packCmd.Args()
	if len(remaining) < 1 {
		return fmt.Errorf("nó raiz obrigatório (ex: mem pack <nota_ou_id>)")
	}
	rootQuery := remaining[0]

	resolvedDepth := *depth
	if packCmd.Lookup("max-depth").Value.String() != "2" && *maxDepthAlias != 2 {
		resolvedDepth = *maxDepthAlias
	}
	if resolvedDepth <= 0 {
		resolvedDepth = 2
	}

	opts := graph.PackOptions{
		MaxDepth:               resolvedDepth,
		MaxTokens:              *maxTokens,
		Direction:              *direction,
		IncludeFringeAbstracts: !*noAbstracts,
	}

	// 1. Resolução de configuração e repositório
	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
	var errOverride error
	resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
	if errOverride != nil {
		return errOverride
	}

	var res *graph.PackResult
	var err error

	if resolvedPG != "" && (*pgURL != "" || (cfg != nil && cfg.Storage.Engine == "postgres")) {
		pgStore, pgErr := store.NewPostgresStore(resolvedPG)
		if pgErr != nil {
			return fmt.Errorf("erro ao conectar no PostgreSQL: %w", pgErr)
		}
		defer pgStore.Close()

		res, err = pgStore.PackContext(ctx, resolvedRepo, rootQuery, opts)
		if err != nil {
			return fmt.Errorf("falha no empacotamento postgres: %w", err)
		}
	} else {
		database, dbErr := db.InitDB(resolvedDB)
		if dbErr != nil {
			return fmt.Errorf("erro ao abrir SQLite (%s): %w", resolvedDB, dbErr)
		}
		defer database.Close()

		res, err = db.PackContext(ctx, database, rootQuery, opts)
		if err != nil {
			return fmt.Errorf("falha no empacotamento sqlite: %w", err)
		}
	}

	// 2. Saída JSON
	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	// 3. Gravação em arquivo (se fornecido --out)
	if *outFile != "" {
		if writeErr := os.WriteFile(*outFile, []byte(res.Markdown), 0644); writeErr != nil {
			return fmt.Errorf("falha ao salvar arquivo '%s': %w", *outFile, writeErr)
		}
		fmt.Fprintf(out, "\n=== 📦 Pacote de Subgrafo Gerado com Sucesso ===\n")
		fmt.Fprintf(out, "Nó Raiz:         %s\n", res.RootTitle)
		fmt.Fprintf(out, "ID Canônico:     %s\n", res.RootID)
		fmt.Fprintf(out, "Profundidade:    %d saltos\n", opts.MaxDepth)
		fmt.Fprintf(out, "Orçamento:       ~%d / %d tokens\n", res.TotalTokens, res.MaxTokens)
		fmt.Fprintf(out, "Nós Centrais:    %d\n", res.CoreCount)
		fmt.Fprintf(out, "Nós Resumidos:   %d\n", res.FringeCount)
		fmt.Fprintf(out, "Nós Omitidos:    %d\n", res.OmittedCount)
		fmt.Fprintf(out, "Arquivo Salvo:   %s\n\n", *outFile)
		return nil
	}

	// 4. Emissão no terminal
	fmt.Fprintln(out, res.Markdown)
	return nil
}

// rearrangePackArgs reorganiza argumentos permitindo flags posicionadas após o nó raiz
func rearrangePackArgs(args []string) []string {
	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Flags com parâmetros obrigatórios
			if arg == "--depth" || arg == "-depth" ||
				arg == "--max-depth" || arg == "-max-depth" ||
				arg == "--max-tokens" || arg == "-max-tokens" ||
				arg == "--direction" || arg == "-direction" ||
				arg == "--out" || arg == "-out" ||
				arg == "--db" || arg == "-db" ||
				arg == "--postgres" || arg == "-postgres" ||
				arg == "--repo" || arg == "-repo" {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flags = append(flags, args[i+1])
					i++
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}

	return append(flags, positional...)
}
