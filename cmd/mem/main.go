package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/canvas"
	"github.com/FelipeMiiller/my-memory/internal/compiler"
	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/deeplink"
	"github.com/FelipeMiiller/my-memory/internal/drift"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/federation"
	"github.com/FelipeMiiller/my-memory/internal/graph"
	"github.com/FelipeMiiller/my-memory/internal/graphview"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/repo"
	"github.com/FelipeMiiller/my-memory/internal/staleness"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
	"github.com/FelipeMiiller/my-memory/internal/watcher"
)

const EmbeddingDim = 768

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	ctx := context.Background()

	// Carrega automaticamente variáveis de ambiente de .memory/.env ou .env se presente
	_, _ = config.FindAndLoadDotEnv(".")

	emb, tq, _ := resolveEmbedder(nil)
	defaultRepo := repo.DetectRepository(".")

	switch os.Args[1] {
	case "init":
		initCmd := flag.NewFlagSet("init", flag.ExitOnError)
		repoSlug := initCmd.String("repo", defaultRepo, "Identificador/slug do repositório para o arquivo de configuração")
		dbPath := initCmd.String("db", "memory.db", "Caminho padrão do arquivo de banco SQLite")
		force := initCmd.Bool("force", false, "Sobrescreve o arquivo de configuração existente se já existir")
		vscode := initCmd.Bool("vscode", false, "Gera configuração .vscode/mcp.json para VS Code Copilot")
		copilot := initCmd.Bool("copilot", false, "Gera instruções .github/copilot-instructions.md para o GitHub Copilot")
		cursor := initCmd.Bool("cursor", false, "Gera configuração .cursor/mcp.json para o Cursor IDE")
		all := initCmd.Bool("all", false, "Gera integrações para todas as IDEs suportadas (VS Code Copilot, Cursor)")
		initCmd.Parse(os.Args[2:])

		targetDir := "."
		if initCmd.NArg() > 0 {
			targetDir = initCmd.Arg(0)
		}

		genVSCode := *vscode || *all
		genCopilot := *copilot || *all
		genCursor := *cursor || *all

		if err := runInit(targetDir, *repoSlug, *dbPath, *force, genVSCode, genCopilot, genCursor); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar vault: %v\n", err)
			os.Exit(1)
		}

	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		dirPath := indexCmd.String("dir", "", "Caminho da pasta com arquivos Markdown")
		force := indexCmd.Bool("force", false, "Força a reindexação completa ignorando o cache SHA-256")
		noPrune := indexCmd.Bool("no-prune", false, "Desativa a exclusão de notas que foram removidas do disco")
		dbPath := indexCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := indexCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := indexCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := indexCmd.String("repo", "", "Identificador/slug do repositório")
		indexCmd.Parse(os.Args[2:])

		target := *dirPath
		if target == "" && indexCmd.NArg() > 0 {
			target = indexCmd.Arg(0)
		}

		// Descoberta e resolução automática de configuração do vault em cascata
		searchDir := target
		if searchDir == "" {
			searchDir = "."
		}
		_, _ = config.FindAndLoadDotEnv(searchDir)
		cfg, cfgPath, _ := config.LoadCascadingConfig(searchDir)
		if cfgPath != "" && target == "" {
			dirOfCfg := filepath.Dir(cfgPath)
			if filepath.Base(dirOfCfg) == ".memory" {
				target = filepath.Dir(dirOfCfg)
			} else {
				target = dirOfCfg
			}
		}

		if cfg == nil {
			def := config.DefaultConfig()
			cfg = &def
		}

		if target == "" {
			fmt.Println("Uso: mem index [--force] [--no-prune] [--db <caminho>] [--postgres <url>] [--repo <nome>] [<pasta_com_markdown>]")
			fmt.Println("     Dica: execute 'mem init' para criar um arquivo de configuração declarativa.")
			return
		}

		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}
		emb, tq, _ = resolveEmbedder(cfg)

		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runIndexPostgres(ctx, pgStore, emb, cfg, resolvedRepo, target, *force, !*noPrune)
		} else {
			database, err := db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runIndexSQLite(ctx, database, emb, tq, cfg, target, *force, !*noPrune)
		}

	case "watch":
		watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
		dirPath := watchCmd.String("dir", "", "Caminho da pasta com notas Markdown (padrão: raiz do vault configurado)")
		debounceMs := watchCmd.Int("debounce", 500, "Janela de debounce em milissegundos (padrão: 500)")
		intervalMs := watchCmd.Int("interval", 1000, "Intervalo de polling em milissegundos (padrão: 1000)")
		dbPath := watchCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := watchCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := watchCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := watchCmd.String("repo", "", "Identificador/slug do repositório")
		watchCmd.Parse(os.Args[2:])

		target := *dirPath
		if target == "" && watchCmd.NArg() > 0 {
			target = watchCmd.Arg(0)
		}

		// Descoberta e resolução automática de configuração do vault em cascata
		searchDir := target
		if searchDir == "" {
			searchDir = "."
		}
		_, _ = config.FindAndLoadDotEnv(searchDir)
		cfg, cfgPath, _ := config.LoadCascadingConfig(searchDir)
		if cfgPath != "" && target == "" {
			dirOfCfg := filepath.Dir(cfgPath)
			if filepath.Base(dirOfCfg) == ".memory" {
				target = filepath.Dir(dirOfCfg)
			} else {
				target = dirOfCfg
			}
		}

		if cfg == nil {
			def := config.DefaultConfig()
			cfg = &def
		}

		if target == "" {
			target = "."
		}

		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}
		emb, tq, _ = resolveEmbedder(cfg)

		setFlags := make(map[string]bool)
		watchCmd.Visit(func(f *flag.Flag) {
			setFlags[f.Name] = true
		})

		debounceVal := *debounceMs
		if !setFlags["debounce"] && cfg.Watcher.DebounceMs > 0 {
			debounceVal = cfg.Watcher.DebounceMs
		}

		intervalVal := *intervalMs
		if !setFlags["interval"] && cfg.Watcher.IntervalMs > 0 {
			intervalVal = cfg.Watcher.IntervalMs
		}

		debounceDur := time.Duration(debounceVal) * time.Millisecond
		intervalDur := time.Duration(intervalVal) * time.Millisecond

		runWatch(ctx, emb, tq, cfg, resolvedRepo, resolvedDB, resolvedPG, target, debounceDur, intervalDur)

	case "hook":
		if len(os.Args) < 3 {
			fmt.Println("Uso: mem hook <install|uninstall> [--force] [<pasta>]")
			os.Exit(1)
		}
		subAction := os.Args[2]
		hookCmd := flag.NewFlagSet("hook", flag.ExitOnError)
		force := hookCmd.Bool("force", false, "Sobrescreve hooks existentes que não pertençam ao My-Memory")
		hookCmd.Parse(os.Args[3:])

		targetDir := "."
		if hookCmd.NArg() > 0 {
			targetDir = hookCmd.Arg(0)
		}

		switch subAction {
		case "install":
			path, err := InstallGitHook(targetDir, *force)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao instalar pre-commit hook: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ Git pre-commit hook instalado com sucesso em: %s\n", path)
		case "uninstall":
			err := UninstallGitHook(targetDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao desinstalar pre-commit hook: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Git pre-commit hook removido com sucesso.")
		default:
			fmt.Printf("Ação desconhecida: '%s'. Use 'install' ou 'uninstall'.\n", subAction)
			os.Exit(1)
		}

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		useTurbo := searchCmd.Bool("tq", false, "Usar busca via TurboQuant (4-bits, SQLite)")
		mode := searchCmd.String("mode", "", "Modo de busca: 'hybrid' (FTS+vetor+grafo via RRF), 'vector' (apenas k-NN), 'fts' (apenas léxico)")
		level := searchCmd.String("level", "", "Nível de densidade de contexto: 'l0' (micro-abstract cirúrgico), 'l1' (overview/tríptico padrão), 'l2' (detalhes completos)")
		category := searchCmd.String("category", "", "Filtra por taxonomia de memória: 'resource', 'memory', 'skill' (padrão: todas)")
		k := searchCmd.Int("k", 0, "Constante de suavização do algoritmo RRF (padrão: 60)")
		limit := searchCmd.Int("limit", 0, "Número máximo de resultados (padrão: 5)")
		decay := searchCmd.Bool("decay", false, "Ativa decaimento temporal exponencial para priorizar notas mais recentes")
		halfLife := searchCmd.Float64("half-life", 0.0, "Tempo de meia-vida em dias para decaimento temporal (padrão: 30.0)")
		decayWeight := searchCmd.Float64("decay-weight", -1.0, "Peso do fator temporal entre 0.0 e 1.0 (padrão: 0.3)")
		dbPath := searchCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := searchCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := searchCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := searchCmd.String("repo", "", "Identificador/slug do repositório para filtrar")
		showLinks := searchCmd.Bool("links", false, "Exibe deep links (Obsidian e VS Code) abaixo de cada resultado")
		searchCmd.Parse(rearrangeSearchArgs(os.Args[2:]))

		query := strings.Join(searchCmd.Args(), " ")
		if query == "" {
			fmt.Println("Uso: mem search [--mode hybrid|vector|fts] [--level l0|l1|l2] [--category resource|memory|skill] [-tq] [--decay] [--half-life 30] [--decay-weight 0.3] [--k 60] [--limit 5] [--links] [--db <caminho>] [--postgres <url>] [--repo <nome>] \"sua pergunta aqui\"")
			return
		}

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}
		emb, tq, _ = resolveEmbedder(cfg)

		setFlags := make(map[string]bool)
		searchCmd.Visit(func(f *flag.Flag) {
			setFlags[f.Name] = true
		})

		resolvedMode := *mode
		if !setFlags["mode"] {
			if cfg.Search.Mode != "" {
				resolvedMode = cfg.Search.Mode
			} else {
				resolvedMode = "hybrid"
			}
		}

		resolvedLevel := *level
		if !setFlags["level"] {
			if cfg.Search.Level != "" {
				resolvedLevel = cfg.Search.Level
			} else {
				resolvedLevel = "l1"
			}
		}
		resolvedLevel = strings.ToLower(strings.TrimSpace(resolvedLevel))
		if resolvedLevel != "l0" && resolvedLevel != "l1" && resolvedLevel != "l2" {
			resolvedLevel = "l1"
		}

		resolvedCategory := *category
		if !setFlags["category"] {
			resolvedCategory = cfg.Search.Category
		}
		resolvedCategory = strings.ToLower(strings.TrimSpace(resolvedCategory))

		searchOpts := store.SearchOptions{
			Level:     resolvedLevel,
			Category:  resolvedCategory,
			ShowLinks: *showLinks,
		}

		resolvedLimit := *limit
		if !setFlags["limit"] {
			if cfg.Search.Limit > 0 {
				resolvedLimit = cfg.Search.Limit
			} else {
				resolvedLimit = 5
			}
		}

		resolvedK := *k
		if !setFlags["k"] {
			if cfg.Search.K > 0 {
				resolvedK = cfg.Search.K
			} else {
				resolvedK = 60
			}
		}

		resolvedDecay := *decay
		if !setFlags["decay"] {
			resolvedDecay = cfg.Search.Decay
		}

		resolvedHalfLife := *halfLife
		if !setFlags["half-life"] {
			if cfg.Search.HalfLife > 0 {
				resolvedHalfLife = cfg.Search.HalfLife
			} else {
				resolvedHalfLife = 30.0
			}
		}

		resolvedDecayWeight := *decayWeight
		if !setFlags["decay-weight"] {
			if cfg.Search.DecayWeight >= 0 {
				resolvedDecayWeight = cfg.Search.DecayWeight
			} else {
				resolvedDecayWeight = 0.3
			}
		}

		resolvedUseTurbo := *useTurbo
		if !setFlags["tq"] {
			resolvedUseTurbo = cfg.Search.UseTurbo
		}

		decayOpts := store.DefaultDecayOptions()
		if resolvedDecay {
			decayOpts.Enabled = true
			decayOpts.HalfLife = resolvedHalfLife
			decayOpts.Weight = resolvedDecayWeight
		}

		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runSearchPostgres(ctx, pgStore, emb, query, resolvedRepo, resolvedMode, resolvedLimit, resolvedK, decayOpts, searchOpts)
		} else {
			database, err := db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runSearchSQLite(ctx, database, emb, tq, query, resolvedMode, resolvedUseTurbo, resolvedLimit, resolvedK, decayOpts, searchOpts)
		}

	case "mcp":
		mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
		dbPath := mcpCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := mcpCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := mcpCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := mcpCmd.String("repo", "", "Identificador padrão do repositório")
		port := mcpCmd.Int("port", 0, "Porta para iniciar o servidor MCP via HTTP/SSE (ex: 38400)")
		host := mcpCmd.String("host", "127.0.0.1", "Host/interface de rede para o servidor HTTP")
		httpAddr := mcpCmd.String("http", "", "Endereço completo para o servidor HTTP (ex: :38400 ou 0.0.0.0:38400)")
		cors := mcpCmd.Bool("cors", true, "Habilita suporte a CORS para conexões de navegadores")
		mcpCmd.Parse(os.Args[2:])

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}
		emb, tq, _ := resolveEmbedder(cfg)

		targetHTTP := *httpAddr
		if targetHTTP == "" && *port > 0 {
			targetHTTP = fmt.Sprintf("%s:%d", *host, *port)
		}

		var pgStore *store.PostgresStore
		var database *sql.DB
		var err error

		if resolvedPG != "" {
			pgStore, err = store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] Erro conectando ao PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			fmt.Fprintf(os.Stderr, "[mcp] Conectado ao PostgreSQL com pgvector (repo padrão: %s)\n", resolvedRepo)
		} else {
			if _, statErr := os.Stat(resolvedDB); os.IsNotExist(statErr) {
				fmt.Fprintf(os.Stderr, "[mcp] Aviso: Banco de dados '%s' não encontrado. Um novo banco será criado na primeira gravação.\n", resolvedDB)
			}
			database, err = db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] Erro inicializando banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			fmt.Fprintf(os.Stderr, "[mcp] Conectado ao SQLite: %s\n", resolvedDB)
		}

		runMCPServer(ctx, pgStore, database, emb, tq, resolvedRepo, targetHTTP, *cors, cfg)

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
		canvasNode := exportCmd.String("canvas", "", "Nome ou identificador da nota raiz para exportar subgrafo para JSON Canvas (.canvas)")
		htmlOut := exportCmd.String("html", "", "Exporta visualização interativa do grafo para arquivo HTML standalone")
		depth := exportCmd.Int("depth", 1, "Profundidade máxima de vizinhos no grafo (padrão: 1)")
		outFile := exportCmd.String("out", "", "Caminho do arquivo .canvas de saída (padrão: <nota>.canvas)")
		openHTML := exportCmd.Bool("open", false, "Abre automaticamente o arquivo no navegador (apenas para exportação HTML)")
		dbPath := exportCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := exportCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := exportCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := exportCmd.String("repo", "", "Identificador/slug do repositório")
		exportCmd.Parse(os.Args[2:])

		if *htmlOut != "" || (exportCmd.NArg() > 0 && strings.HasSuffix(exportCmd.Arg(0), ".html")) {
			targetOut := *htmlOut
			if targetOut == "" && exportCmd.NArg() > 0 {
				targetOut = exportCmd.Arg(0)
			}
			var gArgs []string
			if *openHTML {
				gArgs = append(gArgs, "view")
			} else {
				gArgs = append(gArgs, "export")
			}
			gArgs = append(gArgs, "--out", targetOut, "--depth", fmt.Sprintf("%d", *depth))
			if *canvasNode != "" {
				gArgs = append(gArgs, "--root", *canvasNode)
			}
			if *dbPath != "" {
				gArgs = append(gArgs, "--db", *dbPath)
			}
			if *pgURL != "" {
				gArgs = append(gArgs, "--postgres", *pgURL)
			}
			if *targetRepo != "" {
				gArgs = append(gArgs, "--repo", *targetRepo)
			}
			if err := runGraphCLI(ctx, defaultRepo, gArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
				os.Exit(1)
			}
			return
		}

		node := *canvasNode
		if node == "" && exportCmd.NArg() > 0 {
			node = exportCmd.Arg(0)
		}
		if node == "" {
			fmt.Println("Uso: mem export --canvas <nota> [--depth 1] [--out <saida.canvas>] ou mem export --html <saida.html>")
			return
		}

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}

		if *outFile == "" {
			safeName := strings.ReplaceAll(node, "/", "_")
			safeName = strings.ReplaceAll(safeName, "\\", "_")
			safeName = strings.TrimSuffix(safeName, ".md")
			*outFile = safeName + ".canvas"
		}

		runExportCanvas(ctx, resolvedPG, resolvedDB, resolvedRepo, node, *depth, *outFile)

	case "graph":
		if err := runGraphCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "hubs":
		hubsCmd := flag.NewFlagSet("hubs", flag.ExitOnError)
		top := hubsCmd.Int("top", 10, "Número máximo de nós centrais a exibir (padrão: 10)")
		algorithm := hubsCmd.String("algorithm", "degree", "Algoritmo de centralidade: 'degree' (grau total) ou 'pagerank' (autoridade iterativa ponderada)")
		damping := hubsCmd.Float64("damping", 0.85, "Fator de amortecimento para PageRank (padrão: 0.85)")
		maxIter := hubsCmd.Int("iter", 30, "Número máximo de iterações para PageRank (padrão: 30)")
		dbPath := hubsCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := hubsCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := hubsCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := hubsCmd.String("repo", "", "Identificador/slug do repositório para filtrar")
		hubsCmd.Parse(os.Args[2:])

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}

		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			if strings.ToLower(*algorithm) == "pagerank" {
				runPageRankPostgres(ctx, pgStore, resolvedRepo, *top, *damping, *maxIter)
			} else {
				runHubsPostgres(ctx, pgStore, resolvedRepo, *top)
			}
		} else {
			database, err := db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			if strings.ToLower(*algorithm) == "pagerank" {
				runPageRankSQLite(ctx, database, *top, *damping, *maxIter)
			} else {
				runHubsSQLite(ctx, database, *top)
			}
		}

	case "insights":
		insightsCmd := flag.NewFlagSet("insights", flag.ExitOnError)
		limit := insightsCmd.Int("limit", 10, "Número máximo de conexões inesperadas a exibir (padrão: 10)")
		minSim := insightsCmd.Float64("min-similarity", 0.70, "Limiar mínimo de similaridade semântica (padrão: 0.70)")
		dbPath := insightsCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := insightsCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := insightsCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := insightsCmd.String("repo", "", "Identificador/slug do repositório para filtrar")
		insightsCmd.Parse(os.Args[2:])

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}
		emb, tq, _ = resolveEmbedder(cfg)

		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runInsightsPostgres(ctx, pgStore, resolvedRepo, *limit, *minSim)
		} else {
			database, err := db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runInsightsSQLite(ctx, database, *limit, *minSim)
		}

	case "bench":
		benchCmd := flag.NewFlagSet("bench", flag.ExitOnError)
		benchCmd.Parse(os.Args[2:])
		runBenchmarks()

	case "doctor":
		docCmd := flag.NewFlagSet("doctor", flag.ExitOnError)
		fix := docCmd.Bool("fix", false, "Repara automaticamente anomalias conhecidas (self-loops e dead links)")
		showEvents := docCmd.Bool("events", false, "Inclui seção 'Events' com saúde do event_runtime (ADR-043)")
		dbPath := docCmd.String("db", "", "Caminho do arquivo SQLite")
		pgURL := docCmd.String("postgres", "", "URL de conexão PostgreSQL (com pgvector)")
		storage := docCmd.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)")
		targetRepo := docCmd.String("repo", "", "Identificador/slug do repositório para filtrar")
		docCmd.Parse(os.Args[2:])

		cfg := resolveConfig()
		resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
		var errOverride error
		resolvedDB, resolvedPG, errOverride = applyStorageOverride(*storage, resolvedDB, resolvedPG)
		if errOverride != nil {
			log.Println(errOverride)
			os.Exit(1)
		}

		if resolvedPG != "" {
			pgStore, err := store.NewPostgresStore(resolvedPG)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runDoctorPostgres(ctx, pgStore, resolvedRepo, *fix)
			if *showEvents {
				// Postgres event_log integration is out of scope for T12.
				fmt.Fprintln(os.Stderr, "mem doctor --events: Postgres backend não suportado nesta versão")
			}
		} else {
			database, err := db.InitDB(resolvedDB)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runDoctorSQLite(ctx, database, *fix)
			if *showEvents {
				if err := runDoctorEventsSection(ctx, database); err != nil {
					fmt.Fprintf(os.Stderr, "mem doctor --events: %v\n", err)
				}
			}
		}

	case "note":
		if err := runNoteCLI(ctx, emb, tq, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "compile":
		if err := runCompileCLI(ctx, emb, tq, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "clusters":
		if err := runClustersCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "impact":
		if err := runImpactCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "inspect":
		if err := runInspectCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "path":
		if err := runPathCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "pack":
		if err := runPackCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "open":
		if err := runOpenCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "drift":
		if err := runDriftCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "up":
		code := runUpCommand(ctx, os.Args[2:], ".memory", "", os.Stdout, os.Stderr)
		os.Exit(code)

	case "down":
		code := runDownCommand(ctx, os.Args[2:], ".memory", os.Stdout, os.Stderr)
		os.Exit(code)

	case "profiles":
		code := runProfilesCommand(ctx, os.Args[2:], ".memory", os.Stdout, os.Stderr)
		os.Exit(code)

	case "logs":
		code := runLogsCommand(ctx, os.Args[2:], ".memory", os.Stdout, os.Stderr)
		os.Exit(code)

	case "status":
		if err := runStatusCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "events":
		if err := runEventsCLI(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "setup":
		if err := runSetupCLI(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "central":
		if err := runCentralCLI(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "repos":
		if err := runReposCLI(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "code-index":
		if err := runCodeIndexCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "code-search":
		if err := runCodeSearchCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "install":
		if err := runInstallCLI(ctx, defaultRepo, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	case "version", "--version", "-v":
		if err := runVersionCLI(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
			os.Exit(1)
		}

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("=== My-Memory CLI (SQLite / PostgreSQL com pgvector / TurboQuant / MCP) ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  mem init [--repo <slug>] [--db <arq>] [--force] [--vscode] [--copilot] [--cursor] [--all] [<pasta>]")
	fmt.Println("      Inicializa novo vault (.memory/config.yaml) e opcionalmente gera integrações para VS Code Copilot e Cursor")
	fmt.Println("  mem index [--force] [--no-prune] [--db <arq>] [--postgres <url>] [--repo <slug>] [<pasta>]")
	fmt.Println("      Indexa notas Markdown com cache incremental SHA-256 e pruning de arquivos deletados")
	fmt.Println("  mem status [--json] [--db <arq>] [--postgres <url>] [--repo <slug>] [<pasta>]")
	fmt.Println("      Exibe o status de sincronização e detecção de desatualização do vault em tempo real")
	fmt.Println("  mem events <tail|inspect|replay|trace|last-sequence|stats> [opções]")
	fmt.Println("      Inspeciona o event_runtime event_log (JSON Lines) e re-emite eventos para replay")
	fmt.Println("  mem watch [--debounce <ms>] [--interval <ms>] [--db <arq>] [--postgres <url>] [--repo <slug>] [<pasta>]")
	fmt.Println("      Monitora continuamente o vault em segundo plano e reindexa notas em tempo real")
	fmt.Println("  mem hook <install|uninstall> [--force] [<pasta>]")
	fmt.Println("      Instala ou remove o Git pre-commit hook para indexação automática pré-commit")
	fmt.Println("  mem doctor [--fix] [--events] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Audita a saúde do grafo (dead links, notas órfãs, self-loops e Health Score). Use --events para inspecionar o event_runtime")
	fmt.Println("  mem search [--mode hybrid|vector|fts] [--level l0|l1|l2] [--category resource|memory|skill] [-tq] [--decay] [--half-life 30] [--decay-weight 0.3] [--k 60] [--limit 5] [--db <arq>] [--postgres <url>] [--repo <slug>] \"<pergunta>\"")
	fmt.Println("      Busca com Progressive Context Loading (L0/L1/L2), filtro de categoria, RRF, decaimento temporal e grafo")

	fmt.Println("  mem hubs [--algorithm degree|pagerank] [--damping 0.85] [--iter 30] [--top 10] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Exibe os nós centrais por grau (God Nodes) ou por autoridade estrutural (PageRank ponderado)")
	fmt.Println("  mem clusters [--min-size 2] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Detecta clusters e módulos conceituais no grafo via LPA ponderado e Modularidade Newman-Girvan Q")
	fmt.Println("  mem impact <nota_ou_id> [--depth 2] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Analisa o raio de destruição (Blast Radius) e dependentes reversos com score de risco")
	fmt.Println("  mem inspect <nota_ou_id> [--json] [--full] [--max-len 500] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Visualização cirúrgica em 3 colunas (in-links, nó central e out-links) com risco e preview")
	fmt.Println("  mem path <origem> <destino> [--undirected] [--max-depth 6] [--mode epistemic|hops] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Descoberta de rotas e menor caminho ponderado por custos epistêmicos entre dois nós do grafo")
	fmt.Println("  mem pack <nota_ou_id> [--depth 2] [--max-tokens 4000] [--direction both] [--out <bundle.md>] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Empacota um subgrafo de contexto coerente centrado em uma nota raiz com controle rígido de tokens")
	fmt.Println("  mem open <nota_ou_caminho> [--app obsidian|vscode|system] [--line <n>] [--dry-run] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Abre diretamente a nota ou nó no editor configurado (Obsidian, VS Code) ou exibe deep links acionáveis")
	fmt.Println("  mem drift [--since <faixa>] [--threshold <0.0-1.0>] [--uncovered] [--strict] [--json] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Analisa desvio entre commits de código e notas de memória (Semantic Drift) e detecta código órfão")

	fmt.Println("  mem insights [--limit 10] [--min-similarity 0.70] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Descobre conexões conceituais inesperadas (Surprising Connections) sem links diretos no grafo")
	fmt.Println("  mem graph [view|export] [--root <nota>] [--depth 2] [--out <saida.html>] [--open] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Visualizador interativo de grafo em HTML/SVG standalone com física de forças, busca e PageRank")
	fmt.Println("  mem export [--canvas <nota>] [--html <saida.html>] [--depth 1] [--out <arquivo>] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Exporta o grafo para JSON Canvas 1.0 (.canvas) do Obsidian ou visualizador HTML standalone interativo")
	fmt.Println("  mem bench")
	fmt.Println("      Executa micro-benchmarks de performance (TurboQuant, RRF, SHA-256, Parsing) com resumo tabular")
	fmt.Println("  mem note <create|append> [opções] <caminho>")
	fmt.Println("      Cria ou anexa seções em notas Markdown atômicas com frontmatter e sincronização imediata")
	fmt.Println("  mem compile --topic \"<termo>\" --out \"<caminho.md>\" [--limit 5] [--mode hybrid|vector|fts]")
	fmt.Println("      Compila e sintetiza fragmentos de busca em uma nota atômica com backlinks (Compile-not-Retrieve)")
	fmt.Println("  mem mcp [--port <porta>] [--host <ip>] [--http <addr>] [--cors] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Inicia servidor Model Context Protocol via stdio (padrão) ou via HTTP/SSE na porta indicada")
	fmt.Println("  mem install [--target <ferramenta>] [--dry-run] [--workspace] [--global] [--force]")
	fmt.Println("      Auto-wiring zero-touch: registra o servidor MCP em Claude Desktop, Cursor, VS Code e Windsurf")
	fmt.Println("  mem setup [--central <pasta>] [--engine sqlite|postgres] [--postgres-url <url>] [--mcp-port <porta>] [--yes]")
	fmt.Println("      Assistente interativo de configuração global (~/.memory/config.yaml) e vinculação de cofre central")
	fmt.Println("  mem central <status|bootstrap> [opções]")
	fmt.Println("      Gerencia e audita o Cofre Central de Conhecimento (Global Brain no Google Drive / OneDrive)")
	fmt.Println("  mem repos")
	fmt.Println("      Lista todos os repositórios federados registrados no catálogo global")
	fmt.Println("  mem code-index [--lang=go,py,...] [--include=glob] [--exclude=glob] [--ast-hash] [--no-embed] [--storage=sqlite|postgres] [--db=<arq>] [--repo=<slug>] [<dir>]")
	fmt.Println("      Indexa arquivos de código-fonte via tree-sitter (ADR-047) — grava symbols/edges em code_* tabelas")
	fmt.Println("  mem code-search [--lang=go,py,...] [--kind=function|class|...] [--limit=10] [--no-code-boost] [--links] [--db=<arq>] [--repo=<slug>] <query>")
	fmt.Println("      Busca em code_symbols com boost RRF 2x por qualified_name match (CA-05/CA-07)")
	fmt.Println("  mem version [--json]")
	fmt.Println("      Exibe metadados de versão, commit, data de compilação e arquitetura (ou via -v, --version)")
	fmt.Println()
	fmt.Println("Variáveis de ambiente:")
	fmt.Println("  MY_MEMORY_PG_URL - URL de conexão padrão para o PostgreSQL (ex: postgres://user:pass@localhost:5432/memory?sslmode=disable)")
	fmt.Println("  MY_MEMORY_REPO   - Força o slug do repositório atual (sobrescreve auto-detecção git)")
}

func runInit(targetDir, repoSlug, dbPath string, force bool, vscode, copilot, cursor bool) error {
	if targetDir == "" {
		targetDir = "."
	}
	memDir := filepath.Join(targetDir, ".memory")
	cfgPath := filepath.Join(memDir, "config.yaml")

	configCreated := false
	if _, err := os.Stat(cfgPath); err == nil && !force {
		fmt.Printf("⚠️ Arquivo de configuração já existe em %s. Use --force para sobrescrever.\n", cfgPath)
	} else {
		if err := os.MkdirAll(memDir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório .memory: %w", err)
		}

		if repoSlug == "" {
			repoSlug = "local/vault"
		}
		if dbPath == "" {
			dbPath = "memory.db"
		}

		// Preserva repo_id caso já exista no config.yaml anterior
		repoID := ""
		if existing, err := config.LoadConfig(cfgPath); err == nil && existing.RepoID != "" {
			repoID = existing.RepoID
		}
		if repoID == "" {
			repoID = config.GenerateRepoID()
		}

		template := fmt.Sprintf(`# ==============================================================================
# My-Memory Vault Configuration
# Documentação: docs/REPOSITORY_BRAIN.md e docs/CLI_GUIDE.md
# ==============================================================================

version: 1

# Identificador criptográfico imutável do repositório no catálogo global
repo_id: %q

# Identificador / slug do repositório ou vault para escopo multi-tenant
repository: %q

# Nome amigável do vault de conhecimento
vault_name: "Knowledge Vault"

# Padrões glob de arquivos a serem indexados
# Exemplos: pastas específicas (docs, specs), extensões ou arquivos únicos
include:
  - "docs/**/*.md"      # Exemplo: Documentações em docs/
  - "specs/**/*.md"     # Exemplo: Especificações em specs/
  - ".specs/**/*.md"    # Exemplo: Especificações técnicas em .specs/
  - "**/*.md"           # Padrão: Todos os arquivos Markdown

# Padrões glob e diretórios ignorados durante a varredura
exclude:
  - ".git/**"
  - "node_modules/**"
  - "vendor/**"
  - ".obsidian/**"
  - ".trash/**"
  - ".memory/**"

# Configurações do modelo de embeddings
embedding:
  provider: "ollama"
  model: "nomic-embed-text"
  url: "http://localhost:11434"
  dimension: 768

# Preferências padrão de busca e recuperação
search:
  mode: "hybrid"            # "hybrid", "vector" ou "fts"
  limit: 5                  # Número padrão de resultados
  k: 60                     # Constante RRF (Reciprocal Rank Fusion)
  decay: false              # Priorizar notas mais recentes por data
  half_life: 30.0           # Meia-vida em dias para decaimento temporal
  decay_weight: 0.3         # Peso do decaimento temporal (0.0 a 1.0)
  use_turbo: false          # Busca quantizada 4-bit TurboQuant no SQLite

# Configurações do monitoramento contínuo em tempo real (mem watch)
watcher:
  debounce_ms: 500          # Janela de debounce para agrupar rajadas de gravação
  interval_ms: 1000         # Intervalo de polling periódico
`, repoID, repoSlug)

		if err := os.WriteFile(cfgPath, []byte(template), 0644); err != nil {
			return fmt.Errorf("erro ao salvar arquivo de configuração: %w", err)
		}

		// Registra o repositório no catálogo global (~/.memory/config.yaml)
		absTarget, err := filepath.Abs(targetDir)
		if err == nil {
			_ = config.RegisterRepositoryInGlobalConfig(config.RepositoryCatalogEntry{
				ID:   repoID,
				Path: absTarget,
				Name: repoSlug,
			})
		}

		// Criação padrão de .memory/.gitignore para proteger segredos e bancos locais
		gitIgnorePath := filepath.Join(memDir, ".gitignore")
		if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) || force {
			gitIgnoreContent := `# Variáveis de ambiente locais com credenciais reais (não commitar no Git)
.env
*.env
*.local
*.env.local

# Bancos de dados SQLite locais
*.db
*.db-journal
*.db-wal
*.db-shm
`
			_ = os.WriteFile(gitIgnorePath, []byte(gitIgnoreContent), 0644)
		}

		// Criação padrão de .memory/.env.example para template de PostgreSQL e IA de indexação
		envExamplePath := filepath.Join(memDir, ".env.example")
		if _, err := os.Stat(envExamplePath); os.IsNotExist(err) || force {
			envExampleContent := `# ==============================================================================
# My-Memory - Variáveis de Ambiente (.memory/.env)
# ==============================================================================

# --- Banco de Dados PostgreSQL (pgvector) ---
# Ao definir MY_MEMORY_PG_URL aqui, o My-Memory conecta automaticamente no PostgreSQL:
MY_MEMORY_PG_URL=postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable
# MY_MEMORY_REPO=FelipeMiiller/my-memory

# --- Modelo de IA de Indexação / Embeddings (Opcional) ---
# Por padrão, o My-Memory já utiliza o modelo embutido (nomic-embed-text na porta 11434, dim 768).
# Para substituir por outro modelo (ex: bge-m3, mxbai-embed-large, all-minilm), descomente abaixo:
# MY_MEMORY_EMBED_PROVIDER=ollama
# MY_MEMORY_EMBED_URL=http://localhost:11434
# MY_MEMORY_EMBED_MODEL=nomic-embed-text
# MY_MEMORY_EMBED_DIM=768
`
			_ = os.WriteFile(envExamplePath, []byte(envExampleContent), 0644)
		}

		// Criação padrão de .memory/AGENTS.md como modelo para o usuário colocar na raiz do repo
		agentsTemplatePath := filepath.Join(memDir, "AGENTS.md")
		if _, err := os.Stat(agentsTemplatePath); os.IsNotExist(err) || force {
			agentsContent := `# AGENTS.md

Instruções operacionais e diretrizes para **Agentes de IA** (Antigravity, Claude Code, Cursor, Windsurf, Copilot) neste repositório.

> [!TIP]
> **Dica de Instalação:** Mova ou copie este arquivo para a raiz do seu repositório:
> ` + "`" + `cp .memory/AGENTS.md ./AGENTS.md` + "`" + `

---

## 🧠 Memória do Repositório (My-Memory)

Este repositório utiliza o **My-Memory** como motor de contexto semântico, relacional e busca híbrida.

### 1. Verificação de Inicialização Obrigatória
Ao iniciar qualquer tarefa neste repositório:
1. Verifique se o diretório ` + "`.memory/`" + ` ou o arquivo ` + "`.memory/config.yaml`" + ` existe.
2. Se **NÃO existir**:
   - Execute no terminal: ` + "`mem init`" + ` (ou ` + "`go run ./cmd/mem init`" + `).
   - O comando gerará de forma segura:
     - ` + "`.memory/config.yaml`" + `: Escopo de pastas e banco de dados.
     - ` + "`.memory/.gitignore`" + `: Protege credenciais e bancos locais.
     - ` + "`.memory/.env.example`" + `: Modelo de variáveis de ambiente.
     - ` + "`.memory/AGENTS.md`" + `: Este modelo de instruções.
   - Em seguida, execute ` + "`mem index`" + ` para alimentar a base.

### 2. Diretrizes de Uso da Memória pelo Agente
1. **Consulte antes de modificar**: Antes de refatorar código ou criar novas funcionalidades, execute ` + "`memory_search`" + ` (ou ` + "`mem search \"<tema>\"`" + `) para verificar decisões de arquitetura e notas existentes.
2. **Avalie o Raio de Impacto (Blast Radius)**: Ao modificar ou renomear arquivos e conceitos críticos, use ` + "`memory_get_impact`" + ` (ou ` + "`mem impact <id>`" + `) para analisar dependentes diretos e reversos.
3. **Persista Conhecimento Atômico**: Após tomar decisões ou implementar novas features, utilize ` + "`memory_write_note`" + ` para salvar a síntese no vault com ` + "`[[wikilinks]]`" + `.
4. **Higiene e Integridade**: Use ` + "`memory_doctor`" + ` para auditar a saúde do grafo e detectar links quebrados.

---

## 🛠 Comandos Operacionais para o Agente

` + "```bash" + `
# Indexar alterações no vault de notas
mem index

# Busca híbrida no contexto
mem search "<pergunta ou conceito>"

# Inspecionar nó cirúrgico em 3 colunas (in-links, nó central, out-links)
mem inspect "<caminho ou id>"

# Avaliar raio de impacto de alterações
mem impact "<caminho ou id>" --depth 2

# Iniciar servidor Model Context Protocol (MCP)
mem mcp
` + "```" + `

---

## 📌 Regras de Conduta para Agentes de IA
1. **Codificação:** Arquivos em **UTF-8 sem BOM**.
2. **Commits:** Padrão *Conventional Commits* (` + "`feat:`" + `, ` + "`fix:`" + `, ` + "`docs:`" + `, ` + "`chore:`" + `, ` + "`refactor:`" + `).
3. **Testes:** Sempre execute a suíte de testes antes de concluir tarefas.
4. **Memória Atualizada:** Mantenha notas de documentação sincronizadas ao alterar componentes críticos.
`
			_ = os.WriteFile(agentsTemplatePath, []byte(agentsContent), 0644)
		}

		if _, err := config.LoadConfig(cfgPath); err != nil {
			return fmt.Errorf("erro de validação do arquivo de configuração gerado: %w", err)
		}
		configCreated = true
		fmt.Printf("✅ Configuração inicializada com sucesso em %s\n", cfgPath)
		fmt.Printf("   Repo ID: %s\n", repoID)
		fmt.Printf("   Repositório: %s\n", repoSlug)
		fmt.Printf("   Catálogo Global: Registrado em ~/.memory/config.yaml\n")
		fmt.Printf("   Modelos: %s, %s e %s gerados por padrão\n", gitIgnorePath, envExamplePath, agentsTemplatePath)
	}

	if vscode {
		vscodeDir := filepath.Join(targetDir, ".vscode")
		if err := os.MkdirAll(vscodeDir, 0755); err == nil {
			mcpFile := filepath.Join(vscodeDir, "mcp.json")
			if _, err := os.Stat(mcpFile); os.IsNotExist(err) || force {
				vscodeContent := `{
  "servers": {
    "my-memory": {
      "type": "stdio",
      "command": "mem",
      "args": ["mcp"]
    }
  }
}
`
				if err := os.WriteFile(mcpFile, []byte(vscodeContent), 0644); err == nil {
					fmt.Printf("   VS Code Copilot: %s gerado com sucesso\n", mcpFile)
				}
			}
		}
	}

	if copilot {
		ghDir := filepath.Join(targetDir, ".github")
		if err := os.MkdirAll(ghDir, 0755); err == nil {
			instructionsFile := filepath.Join(ghDir, "copilot-instructions.md")
			if _, err := os.Stat(instructionsFile); os.IsNotExist(err) || force {
				copilotContent := `# Instruções para o GitHub Copilot (My-Memory)

Este repositório está integrado com o **My-Memory** como motor de memória semântica e relacional de contexto via Model Context Protocol (MCP).

## Ferramentas MCP Disponíveis para o Copilot
- ` + "`memory_search`" + `: Recupera notas e trechos cirúrgicos via busca híbrida (BM25 + embeddings + grafo).
- ` + "`memory_get_neighbors`" + `: Retorna dependências, chamadores e notas conectadas no grafo.
- ` + "`memory_get_clusters`" + `: Lista os clusters temáticos e comunidades conceituais do repositório.
- ` + "`memory_write_note`" + `: Cria ou atualiza notas atômicas no vault de conhecimento com frontmatter e conexões tipadas.
- ` + "`memory_compile_note`" + `: Sintetiza e compila fragmentos em uma nova nota consolidada (Compile-not-Retrieve).
- ` + "`memory_visualize_graph`" + `: Exporta visualização interativa do grafo em HTML/SVG.

## Diretrizes de Uso Obrigatórias
1. **Consulte a Memória Antes de Sugerir Mudanças Estruturais**: Sempre utilize ` + "`memory_search`" + ` para verificar decisões de arquitetura e precedentes documentados em notas ou ADRs antes de propor novos padrões.
2. **Respeite o Grafo de Dependências**: Consulte ` + "`memory_get_neighbors`" + ` para analisar o raio de impacto (*blast radius*) antes de renomear ou modificar módulos críticos.
3. **Padrão Compile-not-Retrieve**: Quando o usuário solicitar documentar um novo tema, utilize ` + "`memory_compile_note`" + ` ou ` + "`memory_write_note`" + ` para persistir o conhecimento diretamente no vault.
`
				if err := os.WriteFile(instructionsFile, []byte(copilotContent), 0644); err == nil {
					fmt.Printf("   GitHub Copilot: %s gerado com sucesso\n", instructionsFile)
				}
			}
		}
	}

	if cursor {
		cursorDir := filepath.Join(targetDir, ".cursor")
		if err := os.MkdirAll(cursorDir, 0755); err == nil {
			mcpFile := filepath.Join(cursorDir, "mcp.json")
			if _, err := os.Stat(mcpFile); os.IsNotExist(err) || force {
				cursorContent := `{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp"]
    }
  }
}
`
				if err := os.WriteFile(mcpFile, []byte(cursorContent), 0644); err == nil {
					fmt.Printf("   Cursor IDE: %s gerado com sucesso\n", mcpFile)
				}
			}
		}
	}

	if configCreated {
		fmt.Println("   Dica: execute 'mem index' para iniciar a indexação automática.")
	}
	return nil
}

func runWatch(
	ctx context.Context,
	emb *embedder.OllamaClient,
	tq *turboquant.Quantizer,
	cfg *config.Config,
	targetRepo, dbPath, pgURL, rootDir string,
	debounce, interval time.Duration,
) {
	absRoot, err := filepath.Abs(rootDir)
	if err == nil {
		rootDir = absRoot
	}

	var pgStore *store.PostgresStore
	var database *sql.DB

	if pgURL != "" {
		s, err := store.NewPostgresStore(pgURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()
		pgStore = s
		fmt.Printf("👀 [live-watch] Conectado ao PostgreSQL (pgvector) para repo [%s]\n", targetRepo)
	} else {
		d, err := db.InitDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
			os.Exit(1)
		}
		defer d.Close()
		database = d
		fmt.Printf("👀 [live-watch] Conectado ao SQLite (%s)\n", dbPath)
	}

	w := watcher.NewWatcher(rootDir, cfg, interval, debounce)

	sigCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("🔍 Monitorando alterações em: %s\n", rootDir)
	fmt.Printf("   Debounce: %v | Polling: %v\n", debounce, interval)
	fmt.Println("   Pressione Ctrl+C para encerrar o monitoramento.")

	events := w.Start(sigCtx)

	for {
		select {
		case <-sigCtx.Done():
			fmt.Println("\n🛑 Encerrando monitoramento contínuo...")
			w.Stop()
			fmt.Println("👋 Monitoramento finalizado com sucesso.")
			return

		case ev, ok := <-events:
			if !ok {
				return
			}
			switch ev.Type {
			case watcher.EventCreate, watcher.EventModify:
				var res *watcher.IndexResult
				var idxErr error

				if pgStore != nil {
					res, idxErr = watcher.IndexSingleFilePostgres(sigCtx, pgStore, emb, targetRepo, ev.Path, false)
				} else {
					res, idxErr = watcher.IndexSingleFileSQLite(sigCtx, database, emb, tq, ev.Path, false)
				}

				if idxErr != nil {
					fmt.Printf("❌ Erro ao indexar %s: %v\n", ev.RelPath, idxErr)
				} else if res != nil {
					if res.Action == "cached" {
						fmt.Printf("⏩ [live-cached] %s (sem alterações de conteúdo)\n", ev.RelPath)
					} else {
						fmt.Printf("✔ [live-indexed] %s (%d arestas, %d chunks em %v)\n", ev.RelPath, res.EdgesCount, res.ChunksCount, res.Duration.Round(time.Millisecond))
					}
				}

			case watcher.EventDelete:
				var res *watcher.IndexResult
				var purgeErr error

				if pgStore != nil {
					res, purgeErr = watcher.PurgeSingleFilePostgres(sigCtx, pgStore, targetRepo, ev.Path)
				} else {
					res, purgeErr = watcher.PurgeSingleFileSQLite(sigCtx, database, ev.Path)
				}

				if purgeErr != nil {
					fmt.Printf("❌ Erro ao purgar %s: %v\n", ev.RelPath, purgeErr)
				} else if res != nil {
					fmt.Printf("🗑  [live-purged] %s (removido do índice)\n", ev.RelPath)
				}
			}
		}
	}
}

func resolveConfig() *config.Config {
	_, _ = config.FindAndLoadDotEnv(".")
	if loaded, _, err := config.LoadCascadingConfig("."); err == nil && loaded != nil {
		return loaded
	}
	def := config.DefaultConfig()
	if stat, err := os.Stat(".memory"); err == nil && stat.IsDir() {
		def.Storage.SQLitePath = filepath.Join(".memory", "memory.db")
	}
	return &def
}

func resolveStorageAndRepo(cfg *config.Config, targetRepo, dbPath, pgURL, defaultRepo string) (repo string, db string, pg string) {
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}

	repo = targetRepo
	if repo == "" {
		if dbPath != "" && filepath.IsAbs(dbPath) {
			// Override explícito via --db com path absoluto: derivar slug do diretório do DB.
			// Necessário para cofres não registrados no catálogo global (ex: central-memory).
			derived := filepath.Base(filepath.Dir(dbPath))
			if derived != "" && derived != "." && derived != string(filepath.Separator) {
				repo = derived
			} else if cfg.RepoID != "" {
				repo = cfg.RepoID
			} else if cfg.Repository != "" {
				repo = cfg.Repository
			} else {
				repo = defaultRepo
			}
		} else if cfg.RepoID != "" {
			repo = cfg.RepoID
		} else if cfg.Repository != "" {
			repo = cfg.Repository
		} else if envRepo := os.Getenv("MY_MEMORY_REPO"); envRepo != "" {
			repo = envRepo
		} else {
			repo = defaultRepo
		}
	}

	db = dbPath
	if db == "" {
		if cfg.Storage.SQLitePath != "" {
			db = cfg.Storage.SQLitePath
		} else if stat, err := os.Stat(".memory"); err == nil && stat.IsDir() {
			// .memory/ existe → DB canônico vive lá (ADR-016: Auto-Scoping de Vault).
			// db.InitDB cria o arquivo automaticamente; não usar fallback em CWD.
			db = filepath.Join(".memory", "memory.db")
		} else {
			db = "memory.db"
		}
	} else if db == "memory.db" {
		if _, err := os.Stat("memory.db"); os.IsNotExist(err) {
			if stat, err := os.Stat(".memory/memory.db"); err == nil && !stat.IsDir() {
				db = ".memory/memory.db"
			}
		}
	}

	pg = pgURL
	if pg == "" {
		if envPG := os.Getenv("MY_MEMORY_PG_URL"); envPG != "" {
			pg = envPG
		} else if envPG := os.Getenv("POSTGRES_URL"); envPG != "" {
			pg = envPG
		} else if envPG := os.Getenv("DATABASE_URL"); envPG != "" {
			pg = envPG
		} else if cfg.Storage.Engine == "postgres" && cfg.Storage.PostgresURL != "" {
			pg = cfg.Storage.PostgresURL
		}
	}

	// ADR-040 EARS-7: log anti-split-brain quando Postgres é detectado de fonte
	// não-flag (env var ou config global) e SQLite local já existe em disco.
	// Avisa o usuário que o vault remoto vai ser usado e ignora o SQLite local.
	if pg != "" && pgURL == "" {
		if db != "" {
			if _, err := os.Stat(db); err == nil {
				log.Printf("ℹ️  Postgres detectado no config/env. Central vault será usado.\nℹ️  SQLite local em %s será ignorado nesta sessão.\n", db)
			}
		}
	}

	return repo, db, pg
}

func resolveEmbedder(cfg *config.Config) (*embedder.OllamaClient, *turboquant.Quantizer, int) {
	embedURL := ""
	embedModel := "nomic-embed-text"
	dim := EmbeddingDim

	if cfg != nil {
		if cfg.Embedding.URL != "" {
			embedURL = cfg.Embedding.URL
		}
		if cfg.Embedding.Model != "" {
			embedModel = cfg.Embedding.Model
		}
		if cfg.Embedding.Dimension > 0 {
			dim = cfg.Embedding.Dimension
		}
	}

	if envURL := os.Getenv("MY_MEMORY_EMBED_URL"); envURL != "" {
		embedURL = envURL
	}
	if envModel := os.Getenv("MY_MEMORY_EMBED_MODEL"); envModel != "" {
		embedModel = envModel
	}
	if envDim := os.Getenv("MY_MEMORY_EMBED_DIM"); envDim != "" {
		if d, err := strconv.Atoi(envDim); err == nil && d > 0 {
			dim = d
		}
	}

	emb := embedder.NewOllamaClient(embedURL, embedModel)
	tq := turboquant.NewQuantizer(dim)
	return emb, tq, dim
}

// preCollectDocTitles faz um walk rápido do rootDir e retorna os títulos
// (do frontmatter `title:` se presente, senão basename) de todos os arquivos
// .md, exceto os em dirs excluídos por padrão. Esse é o mesmo critério usado
// por db.ListDocumentTitles (que retorna COALESCE(frontmatter.title, id)),
// então a lista fica consistente com o que está no banco.
func preCollectDocTitles(rootDir string) []string {
	var titles []string
	seen := make(map[string]bool)
	skippedDirs := 0
	_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			for _, defEx := range config.DefaultExcludedDirs {
				if base == defEx {
					skippedDirs++
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		// Extrai o title do frontmatter (mesmo critério que ListDocumentTitles usa).
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		if content, err := os.ReadFile(path); err == nil {
			if fm, _ := parser.ExtractFrontmatter(string(content)); fm != nil && fm.Title != "" {
				title = fm.Title
			}
		}
		if !seen[title] {
			seen[title] = true
			titles = append(titles, title)
		}
		return nil
	})
	return titles
}

// mergeStringLists combina duas listas removendo duplicatas, preservando
// ordem da primeira (dbTitles tem precedência).
func mergeStringLists(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func runIndexPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, cfg *config.Config, targetRepo, rootDir string, force, prune bool) {
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}
	fmt.Printf("🔍 Indexando notas no PostgreSQL (pgvector) para repo [%s] em: %s\n", targetRepo, rootDir)
	totalCount := 0
	indexedCount := 0
	cachedCount := 0
	var activeDocIDs []string

	// Pré-coleta títulos: banco + filesystem. Ver runIndexSQLite para rationale.
	dbTitles, _ := s.ListDocumentTitles(ctx, targetRepo)
	fsTitles := preCollectDocTitles(rootDir)
	availableTitles := mergeStringLists(dbTitles, fsTitles)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			relPath = path
		}

		if d.IsDir() {
			if relPath != "." {
				base := d.Name()
				for _, defEx := range config.DefaultExcludedDirs {
					if base == defEx {
						return filepath.SkipDir
					}
				}
				for _, ex := range cfg.Exclude {
					cleanEx := strings.TrimSuffix(ex, "/**")
					if relPath == cleanEx || strings.HasSuffix(relPath, "/"+cleanEx) {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		if !cfg.ShouldIndex(relPath) {
			return nil
		}

		activeDocIDs = append(activeDocIDs, path)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		totalCount++
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		// ISSUE-010 (2026-09-18): usar frontmatter.title quando existir, alinhando
		// com preCollectDocTitles e ListDocumentTitles. Caso contrário, o fuzzy
		// substring match (ex: tag "spec" → "Spec 040: Implementação..." via
		// availableTitles que JÁ contém frontmatter.title) grava edges tagged_as
		// com target longo enquanto documents.title no DB é o basename. Resultado:
		// mem doctor reporta dead links sistemicamente que só doctor --fix remove.
		if content != "" {
			if fmProbe, _ := parser.ExtractFrontmatter(content); fmProbe != nil && fmProbe.Title != "" {
				title = fmProbe.Title
			}
		}
		docID := path
		currentHash := store.CalculateContentHash(contentBytes)

		// Verificação de cache incremental via SHA-256
		if !force {
			storedHash, err := s.GetDocumentHash(ctx, targetRepo, docID)
			if err == nil && storedHash != "" && storedHash == currentHash {
				cachedCount++
				fmt.Printf("⏩ [cached] %s (inalterado)\n", title)
				return nil
			}
		}

		// Limpa chunks e arestas antigas antes da reindexação limpa
		_ = s.DeleteDocumentData(ctx, targetRepo, docID)

		// Extrai frontmatter, categoria e micro-abstract (L0)
		fm, body := parser.ExtractFrontmatter(content)
		category := "resource"
		if fm != nil && fm.Category != "" {
			category = fm.Category
		}
		if category != "resource" && category != "memory" && category != "skill" {
			category = "resource"
		}
		var abstract string
		if fm != nil && fm.Summary != "" {
			abstract = fm.Summary
		} else if fm != nil && fm.Abstract != "" {
			abstract = fm.Abstract
		} else {
			abstract = parser.ExtractMicroAbstract(body, 160)
		}

		// 1. Salva documento com content_hash, abstract e category
		if err := s.InsertDocumentWithMeta(ctx, targetRepo, docID, path, title, time.Now().Unix(), currentHash, abstract, category); err != nil {
			return err
		}

		// 2. Extrai e salva conexões do grafo ([[wikilinks]], tags e relações tipadas)
		connections := parser.ExtractConnections(content)
		parser.ResolveTagConnections(&connections, availableTitles)
		for _, edge := range connections.Edges {
			_ = s.InsertEdgeWithProps(ctx, targetRepo, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
		}

		// 3. Divide em chunks e gera embeddings (com fallback FTS se offline)
		chunks := parser.ChunkText(content, 200, 30)
		for i, c := range chunks {
			chunkID := fmt.Sprintf("%s#%d", docID, i)
			var vec []float32
			if emb != nil {
				v, err := emb.GenerateEmbedding(c)
				if err == nil {
					vec = v
				} else if i == 0 {
					fmt.Printf("Aviso: Ollama indisponível (%s); indexando em modo léxico FTS\n", chunkID)
				}
			}
			if vec == nil {
				vec = make([]float32, 768)
			}

			_ = s.InsertChunk(ctx, targetRepo, chunkID, docID, c, i, vec)
		}

		indexedCount++
		fmt.Printf("✔ Indexado no Postgres: %s (%d arestas, %d chunks)\n", title, len(connections.Edges), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
		return
	}

	prunedCount := 0
	if prune {
		pruned, err := s.PruneDeletedDocuments(ctx, targetRepo, rootDir, activeDocIDs)
		if err == nil && len(pruned) > 0 {
			prunedCount = len(pruned)
			for _, p := range pruned {
				fmt.Printf("🗑  [pruned] %s (removido do disco)\n", p)
			}
		}
	}

	fmt.Printf("🎉 Concluído! %d documentos processados no PostgreSQL (%d indexados, %d em cache, %d podados).\n", totalCount, indexedCount, cachedCount, prunedCount)
}

func runIndexSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, cfg *config.Config, rootDir string, force, prune bool) {
	if cfg == nil {
		def := config.DefaultConfig()
		cfg = &def
	}
	fmt.Printf("🔍 Indexando notas no SQLite em: %s (com TurboQuant 4-bit ativado)\n", rootDir)
	totalCount := 0
	indexedCount := 0
	cachedCount := 0
	var activeDocIDs []string

	// Pré-coleta títulos dos arquivos .md a serem indexados para o fuzzy resolve.
	// Combinado com ListDocumentTitles do banco, forma a lista completa antes
	// do walk — evita race onde um doc referencia outro que será indexado depois
	// (ex: AGENTS.md menciona [[Wikilinks]] antes do ADR-005 ser processado).
	dbTitles, _ := db.ListDocumentTitles(ctx, database, "")
	fsTitles := preCollectDocTitles(rootDir)
	availableTitles := mergeStringLists(dbTitles, fsTitles)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			relPath = path
		}

		if d.IsDir() {
			if relPath != "." {
				base := d.Name()
				for _, defEx := range config.DefaultExcludedDirs {
					if base == defEx {
						return filepath.SkipDir
					}
				}
				for _, ex := range cfg.Exclude {
					cleanEx := strings.TrimSuffix(ex, "/**")
					if relPath == cleanEx || strings.HasSuffix(relPath, "/"+cleanEx) {
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		if !cfg.ShouldIndex(relPath) {
			return nil
		}

		activeDocIDs = append(activeDocIDs, path)
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		totalCount++
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		// ISSUE-010 (2026-09-18): usar frontmatter.title quando existir, alinhando
		// com preCollectDocTitles e ListDocumentTitles. Caso contrário, o fuzzy
		// substring match (ex: tag "spec" → "Spec 040: Implementação..." via
		// availableTitles que JÁ contém frontmatter.title) grava edges tagged_as
		// com target longo enquanto documents.title no DB é o basename. Resultado:
		// mem doctor reporta dead links sistemicamente que só doctor --fix remove.
		if content != "" {
			if fmProbe, _ := parser.ExtractFrontmatter(content); fmProbe != nil && fmProbe.Title != "" {
				title = fmProbe.Title
			}
		}
		docID := path
		currentHash := store.CalculateContentHash(contentBytes)

		// Verificação de cache incremental via SHA-256
		if !force {
			storedHash, err := db.GetDocumentHash(ctx, database, docID)
			if err == nil && storedHash != "" && storedHash == currentHash {
				cachedCount++
				fmt.Printf("⏩ [cached] %s (inalterado)\n", title)
				return nil
			}
		}

		// Limpa chunks e arestas antigas antes da reindexação limpa
		_ = db.DeleteDocumentData(ctx, database, docID)

		// Extrai frontmatter, categoria e micro-abstract (L0)
		fm, body := parser.ExtractFrontmatter(content)
		category := "resource"
		if fm != nil && fm.Category != "" {
			category = fm.Category
		}
		if category != "resource" && category != "memory" && category != "skill" {
			category = "resource"
		}
		var abstract string
		if fm != nil && fm.Summary != "" {
			abstract = fm.Summary
		} else if fm != nil && fm.Abstract != "" {
			abstract = fm.Abstract
		} else {
			abstract = parser.ExtractMicroAbstract(body, 160)
		}

		// 1. Salva documento com content_hash, abstract e category
		if err := db.InsertDocumentWithMeta(ctx, database, docID, path, title, time.Now().Unix(), currentHash, abstract, category); err != nil {
			return err
		}

		// 2. Extrai e salva conexões do grafo ([[wikilinks]], tags e relações tipadas)
		connections := parser.ExtractConnections(content)
		parser.ResolveTagConnections(&connections, availableTitles)
		for _, edge := range connections.Edges {
			_ = db.InsertEdgeWithProps(ctx, database, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
		}

		// 3. Divide em chunks e gera embeddings (com fallback FTS se offline)
		chunks := parser.ChunkText(content, 200, 30)
		for i, c := range chunks {
			chunkID := fmt.Sprintf("%s#%d", docID, i)
			var vec []float32
			if emb != nil {
				v, err := emb.GenerateEmbedding(c)
				if err == nil {
					vec = v
				} else if i == 0 {
					fmt.Printf("Aviso: Ollama indisponível (%s); indexando em modo léxico FTS\n", chunkID)
				}
			}
			if vec == nil {
				vec = make([]float32, 768)
			}

			// Inserção padrão (sqlite-vec + FTS5)
			_ = db.InsertChunk(ctx, database, chunkID, docID, c, i, vec)

			// Inserção TurboQuant (4-bit comprimido: 384 bytes)
			cv, err := tq.Quantize(vec)
			if err == nil {
				_ = db.InsertTurboQuantChunk(ctx, database, chunkID, cv.Scale, cv.Data)
			}
		}

		indexedCount++
		fmt.Printf("✔ Indexado no SQLite: %s (%d arestas, %d chunks comprimidos)\n", title, len(connections.Edges), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
		return
	}

	prunedCount := 0
	if prune {
		pruned, err := db.PruneDeletedDocuments(ctx, database, rootDir, activeDocIDs)
		if err == nil && len(pruned) > 0 {
			prunedCount = len(pruned)
			for _, p := range pruned {
				fmt.Printf("🗑  [pruned] %s (removido do disco)\n", p)
			}
		}
	}

	fmt.Printf("🎉 Concluído! %d documentos processados no SQLite (%d indexados, %d em cache, %d podados).\n", totalCount, indexedCount, cachedCount, prunedCount)
}

func runSearchPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, query, targetRepo, mode string, limit, k int, decayOpts store.DecayOptions, searchOpts store.SearchOptions) {
	decayInfo := ""
	if decayOpts.Enabled {
		decayInfo = fmt.Sprintf(" [Decay: ativo (meia-vida: %.1fd, peso: %.2f)]", decayOpts.HalfLife, decayOpts.Weight)
	}
	catInfo := ""
	if searchOpts.Category != "" {
		catInfo = fmt.Sprintf(" [Categoria: %s]", strings.ToUpper(searchOpts.Category))
	}
	levelInfo := fmt.Sprintf(" [Nível: %s]", strings.ToUpper(searchOpts.Level))

	fmt.Printf("🔎 Buscando no PostgreSQL (pgvector) [Repo: %s, Modo: %s%s%s%s] por: \"%s\"\n\n", targetRepo, mode, levelInfo, catInfo, decayInfo, query)

	var results []store.SearchResult
	var err error

	switch strings.ToLower(mode) {
	case "fts":
		results, err = s.SearchFTSWithOptions(ctx, targetRepo, query, limit, searchOpts)
	case "vector":
		queryVec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", embErr)
			return
		}
		results, err = s.SearchKNNWithOptions(ctx, targetRepo, queryVec, limit, searchOpts)
	default: // hybrid
		var queryVec []float32
		vec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Aviso: Ollama offline ou falha ao gerar embedding (%v). Executando fallback para busca textual FTS.\n\n", embErr)
		} else {
			queryVec = vec
		}
		results, err = s.SearchHybridWithOptions(ctx, targetRepo, query, queryVec, limit, k, decayOpts, searchOpts)
	}

	if err != nil {
		fmt.Printf("Erro na busca: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("Nenhum resultado encontrado.")
		return
	}

	if strings.EqualFold(searchOpts.Level, "l0") {
		for i, res := range results {
			catBadge := "RESOURCE"
			if res.Category != "" {
				catBadge = strings.ToUpper(res.Category)
			}
			dateStr := ""
			if res.UpdatedAt > 0 {
				dateStr = fmt.Sprintf(" | %s", time.Unix(res.UpdatedAt, 0).UTC().Format("2006-01-02"))
			}
			scoreStr := ""
			if res.Score > 0 {
				scoreStr = fmt.Sprintf(" (Score: %.4f%s)", res.Score, dateStr)
			} else if res.Distance > 0 {
				scoreStr = fmt.Sprintf(" (Dist: %.4f%s)", res.Distance, dateStr)
			}

			fmt.Printf("[%d] \033[1;34m[%s]\033[0m \033[1m%s\033[0m%s\n", i+1, catBadge, res.DocumentID, scoreStr)
			abstract := res.Abstract
			if abstract == "" {
				abstract = "(sem abstract disponível)"
			}
			fmt.Printf("    💡 %s\n", abstract)
			if len(res.Neighbors) > 0 {
				fmt.Printf("    🕸  Vizinhos: [%s]\n", strings.Join(res.Neighbors, ", "))
			}
			if searchOpts.ShowLinks {
				links := deeplink.GenerateLinks("", "", res.DocumentID, 0)
				fmt.Printf("    🔗 Obsidian: %s\n", links.Obsidian)
				fmt.Printf("    🔗 VS Code:  %s\n", links.VSCode)
			}
			fmt.Println()
		}
		return
	}

	for i, res := range results {
		var headers []string
		if res.Category != "" {
			headers = append(headers, fmt.Sprintf("[%s]", strings.ToUpper(res.Category)))
		} else {
			headers = append(headers, "[RESOURCE]")
		}
		if res.Score > 0 {
			headers = append(headers, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headers = append(headers, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		if res.Repository != "" {
			headers = append(headers, fmt.Sprintf("Repo: %s", res.Repository))
		}
		if res.UpdatedAt > 0 {
			headers = append(headers, fmt.Sprintf("Atualizado: %s", time.Unix(res.UpdatedAt, 0).UTC().Format("2006-01-02 15:04:05")))
		}
		headers = append(headers, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Printf("--- [%d] %s ---\n", i+1, strings.Join(headers, " | "))
		if res.Abstract != "" {
			fmt.Printf("💡 Resumo (L0): %s\n", res.Abstract)
		}
		if len(res.Sources) > 0 {
			fmt.Printf("📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
		}
		if res.Content != "" {
			fmt.Println(res.Content)
		}
		if len(res.Neighbors) > 0 {
			fmt.Printf("🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		if searchOpts.ShowLinks {
			links := deeplink.GenerateLinks("", "", res.DocumentID, 0)
			fmt.Printf("🔗 Obsidian: %s\n", links.Obsidian)
			fmt.Printf("🔗 VS Code:  %s\n", links.VSCode)
		}
		fmt.Println()
	}
}

func runSearchSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, query, mode string, useTurbo bool, limit, k int, decayOpts store.DecayOptions, searchOpts store.SearchOptions) {
	vecSubmode := "sqlite-vec"
	if useTurbo || !db.HasSqliteVec {
		vecSubmode = "TurboQuant"
	}
	decayInfo := ""
	if decayOpts.Enabled {
		decayInfo = fmt.Sprintf(" [Decay: ativo (meia-vida: %.1fd, peso: %.2f)]", decayOpts.HalfLife, decayOpts.Weight)
	}
	catInfo := ""
	if searchOpts.Category != "" {
		catInfo = fmt.Sprintf(" [Categoria: %s]", strings.ToUpper(searchOpts.Category))
	}
	levelInfo := fmt.Sprintf(" [Nível: %s]", strings.ToUpper(searchOpts.Level))

	fmt.Printf("🔎 Buscando no SQLite por: \"%s\" [Modo: %s (%s)%s%s%s]\n\n", query, mode, vecSubmode, levelInfo, catInfo, decayInfo)

	var results []db.SearchResult
	var err error

	switch strings.ToLower(mode) {
	case "fts":
		results, err = db.SearchFTSWithOptions(ctx, database, query, limit, searchOpts)
	case "vector":
		queryVec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", embErr)
			return
		}
		if useTurbo || !db.HasSqliteVec {
			results, err = db.SearchTurboQuantWithOptions(ctx, database, tq, queryVec, limit, searchOpts)
		} else {
			results, err = db.SearchKNNWithOptions(ctx, database, queryVec, limit, searchOpts)
			if err != nil {
				results, err = db.SearchTurboQuantWithOptions(ctx, database, tq, queryVec, limit, searchOpts)
			}
		}
	default: // hybrid
		var queryVec []float32
		vec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Aviso: Ollama offline ou falha ao gerar embedding (%v). Executando fallback para busca textual FTS.\n\n", embErr)
		} else {
			queryVec = vec
		}
		results, err = db.SearchHybridRRFWithOptions(ctx, database, tq, query, queryVec, limit, k, useTurbo, decayOpts, searchOpts)
	}

	if err != nil {
		fmt.Printf("Erro na busca: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("Nenhum resultado encontrado.")
		return
	}

	if strings.EqualFold(searchOpts.Level, "l0") {
		for i, res := range results {
			catBadge := "RESOURCE"
			if res.Category != "" {
				catBadge = strings.ToUpper(res.Category)
			}
			dateStr := ""
			if res.UpdatedAt > 0 {
				dateStr = fmt.Sprintf(" | %s", time.Unix(res.UpdatedAt, 0).UTC().Format("2006-01-02"))
			}
			scoreStr := ""
			if res.Score > 0 {
				scoreStr = fmt.Sprintf(" (Score: %.4f%s)", res.Score, dateStr)
			} else if res.Distance > 0 {
				scoreStr = fmt.Sprintf(" (Dist: %.4f%s)", res.Distance, dateStr)
			}

			fmt.Printf("[%d] \033[1;34m[%s]\033[0m \033[1m%s\033[0m%s\n", i+1, catBadge, res.DocumentID, scoreStr)
			abstract := res.Abstract
			if abstract == "" {
				abstract = "(sem abstract disponível)"
			}
			fmt.Printf("    💡 %s\n", abstract)
			if len(res.Neighbors) > 0 {
				fmt.Printf("    🕸  Vizinhos: [%s]\n", strings.Join(res.Neighbors, ", "))
			}
			if searchOpts.ShowLinks {
				links := deeplink.GenerateLinks("", "", res.DocumentID, 0)
				fmt.Printf("    🔗 Obsidian: %s\n", links.Obsidian)
				fmt.Printf("    🔗 VS Code:  %s\n", links.VSCode)
			}
			fmt.Println()
		}
		return
	}

	for i, res := range results {
		var headers []string
		if res.Category != "" {
			headers = append(headers, fmt.Sprintf("[%s]", strings.ToUpper(res.Category)))
		} else {
			headers = append(headers, "[RESOURCE]")
		}
		if res.Score > 0 {
			headers = append(headers, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headers = append(headers, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		if res.UpdatedAt > 0 {
			headers = append(headers, fmt.Sprintf("Atualizado: %s", time.Unix(res.UpdatedAt, 0).UTC().Format("2006-01-02 15:04:05")))
		}
		headers = append(headers, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Printf("--- [%d] %s ---\n", i+1, strings.Join(headers, " | "))
		if res.Abstract != "" {
			fmt.Printf("💡 Resumo (L0): %s\n", res.Abstract)
		}
		if len(res.Sources) > 0 {
			fmt.Printf("📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
		}
		if res.Content != "" {
			fmt.Println(res.Content)
		}
		if len(res.Neighbors) > 0 {
			fmt.Printf("🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		if searchOpts.ShowLinks {
			links := deeplink.GenerateLinks("", "", res.DocumentID, 0)
			fmt.Printf("🔗 Obsidian: %s\n", links.Obsidian)
			fmt.Printf("🔗 VS Code:  %s\n", links.VSCode)
		}
		fmt.Println()
	}
}

func rearrangeSearchArgs(args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Se a flag espera um argumento posicional separado por espaço
			if !strings.Contains(arg, "=") && (arg == "--mode" || arg == "-mode" ||
				arg == "--level" || arg == "-level" ||
				arg == "--category" || arg == "-category" ||
				arg == "--k" || arg == "-k" ||
				arg == "--limit" || arg == "-limit" ||
				arg == "--half-life" || arg == "-half-life" ||
				arg == "--decay-weight" || arg == "-decay-weight" ||
				arg == "--db" || arg == "-db" ||
				arg == "--postgres" || arg == "-postgres" ||
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

func buildSQLiteSearchFunc(database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer) func(ctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
	return func(ctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
		limit := params.Limit
		if limit <= 0 {
			limit = 5
		}
		k := params.K
		if k <= 0 {
			k = 60
		}

		searchOpts := store.SearchOptions{
			Level:    params.DetailLevel,
			Category: params.Category,
		}
		if searchOpts.Level == "" {
			searchOpts.Level = "l1"
		}

		var dbResults []db.SearchResult
		var err error

		switch params.Mode {
		case "fts":
			dbResults, err = db.SearchFTSWithOptions(ctx, database, params.Query, limit, searchOpts)
		case "vector":
			queryVec, embErr := emb.GenerateEmbedding(params.Query)
			if embErr != nil {
				return nil, fmt.Errorf("falha ao gerar embedding: %w", embErr)
			}
			if !db.HasSqliteVec {
				dbResults, err = db.SearchTurboQuantWithOptions(ctx, database, tq, queryVec, limit, searchOpts)
			} else {
				dbResults, err = db.SearchKNNWithOptions(ctx, database, queryVec, limit, searchOpts)
				if err != nil {
					dbResults, err = db.SearchTurboQuantWithOptions(ctx, database, tq, queryVec, limit, searchOpts)
				}
			}
		default: // hybrid
			var queryVec []float32
			vec, embErr := emb.GenerateEmbedding(params.Query)
			if embErr == nil {
				queryVec = vec
			}
			decayOpts := store.DefaultDecayOptions()
			if params.Decay {
				decayOpts.Enabled = true
				if params.HalfLife > 0 {
					decayOpts.HalfLife = params.HalfLife
				}
				if params.DecayWeight >= 0 {
					decayOpts.Weight = params.DecayWeight
				}
			}
			dbResults, err = db.SearchHybridRRFWithOptions(ctx, database, tq, params.Query, queryVec, limit, k, false, decayOpts, searchOpts)
		}

		if err != nil {
			return nil, err
		}

		mcpResults := make([]mcp.SearchResult, len(dbResults))
		for i, r := range dbResults {
			mcpResults[i] = mcp.SearchResult{
				ChunkID:    r.ChunkID,
				DocumentID: r.DocumentID,
				Content:    r.Content,
				Distance:   r.Distance,
				Score:      r.Score,
				Sources:    r.Sources,
				Neighbors:  r.Neighbors,
				UpdatedAt:  r.UpdatedAt,
				Abstract:   r.Abstract,
				Category:   r.Category,
			}
		}
		return mcpResults, nil
	}
}

func runMCPServer(ctx context.Context, pgStore *store.PostgresStore, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, defaultRepo string, httpAddr string, corsEnabled bool, cfg ...*config.Config) {
	srv := mcp.NewServer("my-memory", "1.0.0", os.Stdin, os.Stdout, os.Stderr)

	var mcpCfg *config.Config
	if len(cfg) > 0 && cfg[0] != nil {
		mcpCfg = cfg[0]
	} else {
		mcpCfg = resolveConfig()
	}
	vaultRoot := "."
	if cfgPath, err := config.FindConfigFile("."); err == nil {
		dirOfCfg := filepath.Dir(cfgPath)
		if filepath.Base(dirOfCfg) == ".memory" {
			vaultRoot = filepath.Dir(dirOfCfg)
		} else {
			vaultRoot = dirOfCfg
		}
	}
	stalenessDetector := staleness.NewDetector(vaultRoot, mcpCfg, database, pgStore, defaultRepo)
	srv.SetStalenessDetector(stalenessDetector)

	if pgStore != nil {
		pgSearchFunc := func(ctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
			repo := params.Repo
			if repo == "" {
				repo = defaultRepo
			}
			limit := params.Limit
			if limit <= 0 {
				limit = 5
			}
			k := params.K
			if k <= 0 {
				k = 60
			}

			searchOpts := store.SearchOptions{
				Level:    params.DetailLevel,
				Category: params.Category,
			}
			if searchOpts.Level == "" {
				searchOpts.Level = "l1"
			}

			var pgResults []store.SearchResult
			var err error

			switch params.Mode {
			case "fts":
				pgResults, err = pgStore.SearchFTSWithOptions(ctx, repo, params.Query, limit, searchOpts)
			case "vector":
				queryVec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr != nil {
					return nil, fmt.Errorf("falha ao gerar embedding: %w", embErr)
				}
				pgResults, err = pgStore.SearchKNNWithOptions(ctx, repo, queryVec, limit, searchOpts)
			default: // hybrid
				var queryVec []float32
				vec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr == nil {
					queryVec = vec
				}
				decayOpts := store.DefaultDecayOptions()
				if params.Decay {
					decayOpts.Enabled = true
					if params.HalfLife > 0 {
						decayOpts.HalfLife = params.HalfLife
					}
					if params.DecayWeight >= 0 {
						decayOpts.Weight = params.DecayWeight
					}
				}
				pgResults, err = pgStore.SearchHybridWithOptions(ctx, repo, params.Query, queryVec, limit, k, decayOpts, searchOpts)
			}

			if err != nil {
				return nil, err
			}

			mcpResults := make([]mcp.SearchResult, len(pgResults))
			for i, r := range pgResults {
				mcpResults[i] = mcp.SearchResult{
					ChunkID:    r.ChunkID,
					DocumentID: r.DocumentID,
					Repository: r.Repository,
					Content:    r.Content,
					Distance:   r.Distance,
					Score:      r.Score,
					Sources:    r.Sources,
					Neighbors:  r.Neighbors,
					UpdatedAt:  r.UpdatedAt,
					Abstract:   r.Abstract,
					Category:   r.Category,
				}
			}
			return mcpResults, nil
		}

		var effectiveSearchFunc mcp.AdvancedSearchFunc = pgSearchFunc
		if mcpCfg != nil && mcpCfg.CentralVault.Path != "" {
			expandedCentral := config.ExpandPath(mcpCfg.CentralVault.Path)
			centralSearchFunc := func(cctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
				cp := params
				cp.Repo = "repo_central"
				return pgSearchFunc(cctx, cp)
			}
			fedSearcher := federation.NewFederatedSearcher(pgSearchFunc, centralSearchFunc, expandedCentral, mcpCfg.CentralVault.ReadOnly)
			effectiveSearchFunc = fedSearcher.Search
		}

		srv.SetAdvancedSearchHandler(effectiveSearchFunc)
		syncEngine := compiler.NewSyncEngine(nil, pgStore, emb, nil, defaultRepo)
		srv.SetCompilerEngine(syncEngine, effectiveSearchFunc, ".")

		srv.SetNeighborsHandler(func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.GetNodeNeighbors(ctx, repo, nodeID, maxDepth)
		})

		srv.SetAdvancedHubsHandler(func(ctx context.Context, repo string, limit int) ([]mcp.GodNode, error) {
			if repo == "" {
				repo = defaultRepo
			}
			nodes, err := pgStore.GetGodNodes(ctx, repo, limit)
			if err != nil {
				return nil, err
			}
			mcpNodes := make([]mcp.GodNode, len(nodes))
			for i, n := range nodes {
				mcpNodes[i] = mcp.GodNode{
					ID:          n.ID,
					Name:        n.Name,
					InDegree:    n.InDegree,
					OutDegree:   n.OutDegree,
					TotalDegree: n.TotalDegree,
				}
			}
			return mcpNodes, nil
		}, func(ctx context.Context, repo string, limit int) ([]mcp.PageRankNode, error) {
			if repo == "" {
				repo = defaultRepo
			}
			nodes, err := pgStore.ComputePageRank(ctx, repo, 0.85, 30)
			if err != nil {
				return nil, err
			}
			if limit > 0 && len(nodes) > limit {
				nodes = nodes[:limit]
			}
			mcpNodes := make([]mcp.PageRankNode, len(nodes))
			for i, n := range nodes {
				mcpNodes[i] = mcp.PageRankNode{
					ID:        n.ID,
					Name:      n.Name,
					Score:     n.Score,
					Rank:      n.Rank,
					InDegree:  n.InDegree,
					OutDegree: n.OutDegree,
				}
			}
			return mcpNodes, nil
		})

		srv.SetInsightsHandler(func(ctx context.Context, repo string, limit int, minSimilarity float64) ([]mcp.SurprisingConnection, error) {
			if repo == "" {
				repo = defaultRepo
			}
			conns, err := pgStore.FindSurprisingConnections(ctx, repo, limit, minSimilarity)
			if err != nil {
				return nil, err
			}
			mcpConns := make([]mcp.SurprisingConnection, len(conns))
			for i, c := range conns {
				mcpConns[i] = mcp.SurprisingConnection{
					SourceID:   c.SourceID,
					SourceName: c.SourceName,
					TargetID:   c.TargetID,
					TargetName: c.TargetName,
					Similarity: c.Similarity,
					Reason:     c.Reason,
				}
			}
			return mcpConns, nil
		})

		srv.SetDoctorHandler(
			func(ctx context.Context, repo string) (*mcp.DoctorReport, error) {
				if repo == "" {
					repo = defaultRepo
				}
				rep, err := pgStore.DiagnoseHealth(ctx, repo)
				if err != nil {
					return nil, err
				}
				return convertStoreDoctorReportToMCP(rep), nil
			},
			func(ctx context.Context, repo string) (int, error) {
				if repo == "" {
					repo = defaultRepo
				}
				return pgStore.FixHealthIssues(ctx, repo)
			},
		)

		srv.SetGraphViewHandler(func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return graphview.BuildFromPostgres(ctx, pgStore, repo, rootNode, maxDepth)
		}, ".")

		srv.SetClustersHandler(func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			gv, err := graphview.BuildFromPostgres(ctx, pgStore, repo, "", 0)
			if err != nil {
				return graph.CommunityResult{}, err
			}
			return graph.CommunityResult{
				Communities: gv.Communities,
				Modularity:  gv.Stats.Modularity,
				TotalNodes:  gv.Stats.TotalNodes,
				TotalEdges:  gv.Stats.TotalEdges,
			}, nil
		})

		srv.SetImpactHandler(func(ctx context.Context, repo, nodeID string, maxDepth int) (*graph.ImpactResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.CalculateImpact(ctx, repo, nodeID, maxDepth)
		})

		srv.SetInspectHandler(func(ctx context.Context, repo, nodeID string, maxContentLen int) (*graph.TriptychView, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.InspectNode(ctx, repo, nodeID, maxContentLen)
		})

		srv.SetPathHandler(func(ctx context.Context, repo, source, target string, opts graph.PathOptions) (*graph.PathResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.FindPath(ctx, repo, source, target, opts)
		})

		srv.SetPackHandler(func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.PackContext(ctx, repo, rootQuery, opts)
		})

		cwd, _ := os.Getwd()
		srv.SetOpenHandler(func(ctx context.Context, repo, nodeID string) (string, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.ResolveNodeCanonicalID(ctx, repo, nodeID)
		}, cwd, mcpCfg.ResolveObsidianVault(cwd), deeplink.DefaultLauncher)

		srv.SetDriftHandler(func(ctx context.Context, rangeStr string, threshold float64, includeUncovered bool, repoSlug string) (*drift.DriftReport, error) {
			repoRoot, _ := os.Getwd()
			vaultName := mcpCfg.ResolveObsidianVault(repoRoot)
			return drift.AnalyzeDrift(ctx, drift.NewOSGitRunner(), pgStore.DB(), repoRoot, vaultName, rangeStr, threshold, includeUncovered)
		})
	} else if database != nil {

		if tq == nil {
			tq = turboquant.NewQuantizer(EmbeddingDim)
		}
		dbSearchFunc := buildSQLiteSearchFunc(database, emb, tq)
		var effectiveSearchFunc mcp.AdvancedSearchFunc = dbSearchFunc

		if mcpCfg != nil && mcpCfg.CentralVault.Path != "" {
			expandedCentral := config.ExpandPath(mcpCfg.CentralVault.Path)
			centralDBPath := federation.ResolveCentralSQLitePath(expandedCentral)
			var centralSearchFunc federation.SearchFunc
			if _, statErr := os.Stat(centralDBPath); statErr == nil {
				if centralDB, dbErr := db.InitDB(centralDBPath); dbErr == nil {
					defer centralDB.Close()
					centralSearchFunc = buildSQLiteSearchFunc(centralDB, emb, tq)
				}
			}
			fedSearcher := federation.NewFederatedSearcher(dbSearchFunc, centralSearchFunc, expandedCentral, mcpCfg.CentralVault.ReadOnly)
			effectiveSearchFunc = fedSearcher.Search
		}

		srv.SetAdvancedSearchHandler(effectiveSearchFunc)
		syncEngine := compiler.NewSyncEngine(database, nil, emb, tq, defaultRepo)
		srv.SetCompilerEngine(syncEngine, effectiveSearchFunc, ".")

		srv.SetNeighborsHandler(func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
			return db.GetNodeNeighbors(ctx, database, nodeID, maxDepth)
		})

		srv.SetAdvancedHubsHandler(func(ctx context.Context, repo string, limit int) ([]mcp.GodNode, error) {
			nodes, err := db.GetGodNodes(ctx, database, limit)
			if err != nil {
				return nil, err
			}
			mcpNodes := make([]mcp.GodNode, len(nodes))
			for i, n := range nodes {
				mcpNodes[i] = mcp.GodNode{
					ID:          n.ID,
					Name:        n.Name,
					InDegree:    n.InDegree,
					OutDegree:   n.OutDegree,
					TotalDegree: n.TotalDegree,
				}
			}
			return mcpNodes, nil
		}, func(ctx context.Context, repo string, limit int) ([]mcp.PageRankNode, error) {
			nodes, err := db.ComputePageRank(ctx, database, 0.85, 30)
			if err != nil {
				return nil, err
			}
			if limit > 0 && len(nodes) > limit {
				nodes = nodes[:limit]
			}
			mcpNodes := make([]mcp.PageRankNode, len(nodes))
			for i, n := range nodes {
				mcpNodes[i] = mcp.PageRankNode{
					ID:        n.ID,
					Name:      n.Name,
					Score:     n.Score,
					Rank:      n.Rank,
					InDegree:  n.InDegree,
					OutDegree: n.OutDegree,
				}
			}
			return mcpNodes, nil
		})

		srv.SetInsightsHandler(func(ctx context.Context, repo string, limit int, minSimilarity float64) ([]mcp.SurprisingConnection, error) {
			conns, err := db.FindSurprisingConnections(ctx, database, limit, minSimilarity)
			if err != nil {
				return nil, err
			}
			mcpConns := make([]mcp.SurprisingConnection, len(conns))
			for i, c := range conns {
				mcpConns[i] = mcp.SurprisingConnection{
					SourceID:   c.SourceID,
					SourceName: c.SourceName,
					TargetID:   c.TargetID,
					TargetName: c.TargetName,
					Similarity: c.Similarity,
					Reason:     c.Reason,
				}
			}
			return mcpConns, nil
		})

		srv.SetDoctorHandler(
			func(ctx context.Context, repo string) (*mcp.DoctorReport, error) {
				rep, err := db.DiagnoseHealth(ctx, database)
				if err != nil {
					return nil, err
				}
				return convertStoreDoctorReportToMCP(rep), nil
			},
			func(ctx context.Context, repo string) (int, error) {
				return db.FixHealthIssues(ctx, database)
			},
		)

		srv.SetGraphViewHandler(func(ctx context.Context, repo, rootNode string, maxDepth int) (*graphview.GraphView, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return graphview.BuildFromSQLite(ctx, database, rootNode, maxDepth, repo)
		}, ".")

		srv.SetClustersHandler(func(ctx context.Context, repo string, minSize int) (graph.CommunityResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			gv, err := graphview.BuildFromSQLite(ctx, database, "", 0, repo)
			if err != nil {
				return graph.CommunityResult{}, err
			}
			return graph.CommunityResult{
				Communities: gv.Communities,
				Modularity:  gv.Stats.Modularity,
				TotalNodes:  gv.Stats.TotalNodes,
				TotalEdges:  gv.Stats.TotalEdges,
			}, nil
		})

		srv.SetImpactHandler(func(ctx context.Context, repo, nodeID string, maxDepth int) (*graph.ImpactResult, error) {
			return db.CalculateImpactForTarget(ctx, database, nodeID, maxDepth)
		})

		srv.SetInspectHandler(func(ctx context.Context, repo, nodeID string, maxContentLen int) (*graph.TriptychView, error) {
			return db.InspectNode(ctx, database, nodeID, maxContentLen)
		})

		srv.SetPathHandler(func(ctx context.Context, repo, source, target string, opts graph.PathOptions) (*graph.PathResult, error) {
			return db.FindPath(ctx, database, source, target, opts)
		})

		srv.SetPackHandler(func(ctx context.Context, repo, rootQuery string, opts graph.PackOptions) (*graph.PackResult, error) {
			return db.PackContext(ctx, database, rootQuery, opts)
		})

		cwd, _ := os.Getwd()
		srv.SetOpenHandler(func(ctx context.Context, repo, nodeID string) (string, error) {
			return db.ResolveNodeCanonicalID(ctx, database, nodeID)
		}, cwd, mcpCfg.ResolveObsidianVault(cwd), deeplink.DefaultLauncher)

		srv.SetDriftHandler(func(ctx context.Context, rangeStr string, threshold float64, includeUncovered bool, repoSlug string) (*drift.DriftReport, error) {
			repoRoot, _ := os.Getwd()
			vaultName := mcpCfg.ResolveObsidianVault(repoRoot)
			return drift.AnalyzeDrift(ctx, drift.NewOSGitRunner(), database, repoRoot, vaultName, rangeStr, threshold, includeUncovered)
		})
	}

	if httpAddr != "" {

		httpSrv := mcp.NewHTTPServer(srv, mcp.HTTPServerOptions{
			Addr:        httpAddr,
			CORSEnabled: corsEnabled,
			DefaultRepo: defaultRepo,
			Logger:      log.New(os.Stderr, "[mcp-http] ", log.LstdFlags),
		})

		ctxSig, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer cancel()

		serverErr := make(chan error, 1)
		go func() {
			fmt.Fprintf(os.Stderr, "🚀 Servidor MCP HTTP/SSE ativo em http://%s\n", httpAddr)
			fmt.Fprintf(os.Stderr, "   - Endpoint SSE:      http://%s/sse\n", httpAddr)
			fmt.Fprintf(os.Stderr, "   - Endpoint Mensagem: http://%s/message?sessionId=<uuid>\n", httpAddr)
			fmt.Fprintf(os.Stderr, "   - Endpoint Direto:   http://%s/mcp\n", httpAddr)
			fmt.Fprintf(os.Stderr, "   - Diagnóstico:       http://%s/health\n", httpAddr)
			serverErr <- httpSrv.ListenAndServe()
		}()

		select {
		case <-ctxSig.Done():
			fmt.Fprintf(os.Stderr, "\n[mcp] Encerrando servidor HTTP graciosamente...\n")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = httpSrv.Shutdown(shutdownCtx)
		case err := <-serverErr:
			if err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "[mcp] Erro fatal no servidor HTTP: %v\n", err)
			}
		}
		return
	}

	if err := srv.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "[mcp] servidor encerrado com erro: %v\n", err)
	}
}

func runExportCanvas(ctx context.Context, pgURL, dbPath, repo, nodeID string, maxDepth int, outputPath string) {
	fmt.Printf("🎨 Exportando subgrafo centrado em '%s' (profundidade: %d) para JSON Canvas...\n", nodeID, maxDepth)

	var neighbors []string
	var err error

	if pgURL != "" {
		pgStore, errConn := store.NewPostgresStore(pgURL)
		if errConn != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", errConn)
			os.Exit(1)
		}
		defer pgStore.Close()
		neighbors, err = pgStore.GetNodeNeighbors(ctx, repo, nodeID, maxDepth)
	} else {
		database, errConn := db.InitDB(dbPath)
		if errConn != nil {
			fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", errConn)
			os.Exit(1)
		}
		defer database.Close()
		neighbors, err = db.GetNodeNeighbors(ctx, database, nodeID, maxDepth)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao recuperar vizinhos do grafo: %v\n", err)
		os.Exit(1)
	}

	c := canvas.FromNeighbors(nodeID, neighbors)
	if err := c.SaveToFile(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao salvar arquivo JSON Canvas: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✔ Sucesso! Arquivo gerado em: %s (%d nós, %d arestas)\n", outputPath, len(c.Nodes), len(c.Edges))
	if len(neighbors) > 0 {
		fmt.Printf("Conexões mapeadas: %s\n", strings.Join(neighbors, ", "))
	}
}

func runHubsPostgres(ctx context.Context, s *store.PostgresStore, targetRepo string, top int) {
	fmt.Printf("🌟 Calculando nós centrais (God Nodes / Hubs) no PostgreSQL para repo [%s] (top %d)...\n", targetRepo, top)
	hubs, err := s.GetGodNodes(ctx, targetRepo, top)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar nós centrais: %v\n", err)
		os.Exit(1)
	}
	displayHubsTable(hubs)
}

func runHubsSQLite(ctx context.Context, database *sql.DB, top int) {
	fmt.Printf("🌟 Calculando nós centrais (God Nodes / Hubs) no SQLite (top %d)...\n", top)
	hubs, err := db.GetGodNodes(ctx, database, top)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar nós centrais: %v\n", err)
		os.Exit(1)
	}
	displayHubsTable(hubs)
}

func displayHubsTable(hubs []store.GodNode) {
	if len(hubs) == 0 {
		fmt.Println("Nenhum nó ou aresta encontrado no grafo.")
		return
	}

	fmt.Printf("\n%-4s | %-40s | %-10s | %-10s | %-12s\n", "Rank", "Nó / Documento", "Entradas", "Saídas", "Total Grau")
	fmt.Println(strings.Repeat("-", 85))
	for i, h := range hubs {
		displayName := h.Name
		if displayName == "" {
			displayName = h.ID
		}
		if len(displayName) > 38 {
			displayName = displayName[:35] + "..."
		}
		fmt.Printf("%-4d | %-40s | %-10d | %-10d | %-12d\n", i+1, displayName, h.InDegree, h.OutDegree, h.TotalDegree)
	}
	fmt.Println()
}

func runPageRankPostgres(ctx context.Context, s *store.PostgresStore, targetRepo string, top int, damping float64, maxIter int) {
	fmt.Printf("🌐 Calculando autoridade de nós via PageRank no PostgreSQL para repo [%s] (top %d, d=%.2f, maxIter=%d)...\n", targetRepo, top, damping, maxIter)
	nodes, err := s.ComputePageRank(ctx, targetRepo, damping, maxIter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao calcular PageRank: %v\n", err)
		os.Exit(1)
	}
	if top > 0 && len(nodes) > top {
		nodes = nodes[:top]
	}
	displayPageRankTable(nodes)
}

func runPageRankSQLite(ctx context.Context, database *sql.DB, top int, damping float64, maxIter int) {
	fmt.Printf("🌐 Calculando autoridade de nós via PageRank no SQLite (top %d, d=%.2f, maxIter=%d)...\n", top, damping, maxIter)
	nodes, err := db.ComputePageRank(ctx, database, damping, maxIter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao calcular PageRank: %v\n", err)
		os.Exit(1)
	}
	if top > 0 && len(nodes) > top {
		nodes = nodes[:top]
	}
	displayPageRankTable(nodes)
}

func displayPageRankTable(nodes []store.PageRankNode) {
	if len(nodes) == 0 {
		fmt.Println("Nenhum nó ou aresta encontrado no grafo.")
		return
	}

	fmt.Printf("\n%-4s | %-40s | %-12s | %-10s | %-10s\n", "Rank", "Nó / Documento", "Score (PR)", "Entradas", "Saídas")
	fmt.Println(strings.Repeat("-", 85))
	for _, n := range nodes {
		displayName := n.Name
		if displayName == "" {
			displayName = n.ID
		}
		if len(displayName) > 38 {
			displayName = displayName[:35] + "..."
		}
		fmt.Printf("%-4d | %-40s | %-12.4f | %-10d | %-10d\n", n.Rank, displayName, n.Score, n.InDegree, n.OutDegree)
	}
	fmt.Println()
}

func runInsightsPostgres(ctx context.Context, s *store.PostgresStore, targetRepo string, limit int, minSim float64) {
	fmt.Printf("💡 Descobrindo conexões inesperadas no PostgreSQL para repo [%s] (mín. %.0f%%, limite: %d)...\n", targetRepo, minSim*100, limit)
	conns, err := s.FindSurprisingConnections(ctx, targetRepo, limit, minSim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar conexões inesperadas: %v\n", err)
		os.Exit(1)
	}
	displayInsightsTable(conns)
}

func runInsightsSQLite(ctx context.Context, database *sql.DB, limit int, minSim float64) {
	fmt.Printf("💡 Descobrindo conexões inesperadas no SQLite (mín. %.0f%%, limite: %d)...\n", minSim*100, limit)
	conns, err := db.FindSurprisingConnections(ctx, database, limit, minSim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao buscar conexões inesperadas: %v\n", err)
		os.Exit(1)
	}
	displayInsightsTable(conns)
}

func displayInsightsTable(conns []store.SurprisingConnection) {
	if len(conns) == 0 {
		fmt.Println("Nenhuma conexão inesperada encontrada com os critérios especificados.")
		return
	}

	fmt.Printf("\n%-4s | %-30s | %-30s | %-12s | %s\n", "Rank", "Origem", "Destino", "Similaridade", "Razão / Detalhes")
	fmt.Println(strings.Repeat("-", 115))
	for i, c := range conns {
		src := c.SourceName
		if src == "" {
			src = c.SourceID
		}
		if len(src) > 28 {
			src = src[:25] + "..."
		}
		tgt := c.TargetName
		if tgt == "" {
			tgt = c.TargetID
		}
		if len(tgt) > 28 {
			tgt = tgt[:25] + "..."
		}
		simStr := fmt.Sprintf("%.1f%%", c.Similarity*100)
		fmt.Printf("%-4d | %-30s | %-30s | %-12s | %s\n", i+1, src, tgt, simStr, c.Reason)
	}
	fmt.Println()
}

func runDoctorPostgres(ctx context.Context, s *store.PostgresStore, targetRepo string, fix bool) {
	fmt.Printf("🩺 Auditando integridade do grafo no PostgreSQL [%s]...\n", targetRepo)
	fixedCount := 0
	if fix {
		n, err := s.FixHealthIssues(ctx, targetRepo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao aplicar correções: %v\n", err)
		} else {
			fixedCount = n
		}
	}

	report, err := s.DiagnoseHealth(ctx, targetRepo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao diagnosticar banco: %v\n", err)
		os.Exit(1)
	}
	displayDoctorReport(report, fixedCount)
}

func runDoctorSQLite(ctx context.Context, database *sql.DB, fix bool) {
	fmt.Printf("🩺 Auditando integridade do grafo no SQLite...\n")
	fixedCount := 0
	if fix {
		n, err := db.FixHealthIssues(ctx, database)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao aplicar correções: %v\n", err)
		} else {
			fixedCount = n
		}
	}

	report, err := db.DiagnoseHealth(ctx, database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao diagnosticar banco: %v\n", err)
		os.Exit(1)
	}
	displayDoctorReport(report, fixedCount)
}

func displayDoctorReport(report *store.DoctorReport, fixedCount int) {
	if report == nil {
		return
	}

	scoreColor := "🟢"
	if report.HealthScore < 70 {
		scoreColor = "🔴"
	} else if report.HealthScore < 90 {
		scoreColor = "🟡"
	}

	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Printf(" %s RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: %d/100)\n", scoreColor, report.HealthScore)
	fmt.Println(strings.Repeat("=", 90))
	fmt.Printf(" Documentos: %d | Chunks: %d | Arestas: %d | Nós: %d\n",
		report.TotalDocuments, report.TotalChunks, report.TotalEdges, report.TotalNodes)
	if fixedCount > 0 {
		fmt.Printf(" 🛠  Reparos aplicados (--fix): %d arestas problemáticas removidas\n", fixedCount)
	}
	fmt.Println(strings.Repeat("-", 90))

	// Dead Links
	fmt.Printf(" [🔗 Links Quebrados / Dead Links] (%d)\n", len(report.DeadLinks))
	if len(report.DeadLinks) == 0 {
		fmt.Println("  ✔ Nenhum link quebrado detectado. Todas as conexões apontam para notas existentes.")
	} else {
		for i, dl := range report.DeadLinks {
			if i >= 15 {
				fmt.Printf("  ... e mais %d links quebrados ocultados\n", len(report.DeadLinks)-15)
				break
			}
			fmt.Printf("  - [[%s]] -> [[%s]] (relação: %s)\n", dl.SourceID, dl.TargetID, dl.Relation)
		}
	}
	fmt.Println(strings.Repeat("-", 90))

	// Orphan Notes
	fmt.Printf(" [🏝️  Notas Órfãs / Sem Conexões] (%d)\n", len(report.OrphanNotes))
	if len(report.OrphanNotes) == 0 {
		fmt.Println("  ✔ Nenhuma nota isolada. Todos os documentos possuem ao menos 1 conexão.")
	} else {
		for i, on := range report.OrphanNotes {
			if i >= 15 {
				fmt.Printf("  ... e mais %d notas órfãs ocultadas\n", len(report.OrphanNotes)-15)
				break
			}
			displayName := on.Title
			if displayName == "" {
				displayName = on.ID
			}
			fmt.Printf("  - %s (%s)\n", displayName, on.ID)
		}
	}
	fmt.Println(strings.Repeat("-", 90))

	// Self-Loops
	if len(report.SelfLoops) > 0 {
		fmt.Printf(" [🔄 Self-Loops / Conexões Reflexivas] (%d)\n", len(report.SelfLoops))
		for _, sl := range report.SelfLoops {
			fmt.Printf("  - %s (relação: %s)\n", sl.NodeID, sl.Relation)
		}
		fmt.Println(strings.Repeat("-", 90))
	}

	// Desynced Chunks
	if len(report.DesyncedChunks) > 0 {
		fmt.Printf(" [⚡ Chunks Sem Vetores] (%d)\n", len(report.DesyncedChunks))
		for _, dc := range report.DesyncedChunks {
			fmt.Printf("  - Chunk %s: %s\n", dc.ChunkID, dc.Issue)
		}
		fmt.Println(strings.Repeat("-", 90))
	}

	fmt.Println()
}

func convertStoreDoctorReportToMCP(rep *store.DoctorReport) *mcp.DoctorReport {
	if rep == nil {
		return nil
	}
	mcpReport := &mcp.DoctorReport{
		TotalDocuments: rep.TotalDocuments,
		TotalChunks:    rep.TotalChunks,
		TotalEdges:     rep.TotalEdges,
		TotalNodes:     rep.TotalNodes,
		HealthScore:    rep.HealthScore,
		DeadLinks:      make([]mcp.DeadLink, len(rep.DeadLinks)),
		OrphanNotes:    make([]mcp.OrphanNote, len(rep.OrphanNotes)),
		SelfLoops:      make([]mcp.SelfLoop, len(rep.SelfLoops)),
		DesyncedChunks: make([]mcp.DesyncedChunk, len(rep.DesyncedChunks)),
	}
	for i, dl := range rep.DeadLinks {
		mcpReport.DeadLinks[i] = mcp.DeadLink{SourceID: dl.SourceID, TargetID: dl.TargetID, Relation: dl.Relation}
	}
	for i, on := range rep.OrphanNotes {
		mcpReport.OrphanNotes[i] = mcp.OrphanNote{ID: on.ID, Title: on.Title}
	}
	for i, sl := range rep.SelfLoops {
		mcpReport.SelfLoops[i] = mcp.SelfLoop{NodeID: sl.NodeID, Relation: sl.Relation}
	}
	for i, dc := range rep.DesyncedChunks {
		mcpReport.DesyncedChunks[i] = mcp.DesyncedChunk{ChunkID: dc.ChunkID, DocumentID: dc.DocumentID, Issue: dc.Issue}
	}
	return mcpReport
}

type BenchMetric struct {
	Name        string
	Iterations  int
	Duration    time.Duration
	NsPerOp     int64
	BytesPerOp  int64
	AllocsPerOp int64
	Throughput  string
}

func runBenchLoop(name string, duration time.Duration, bytesProcessed int64, fn func()) BenchMetric {
	for i := 0; i < 5; i++ {
		fn()
	}

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)

	start := time.Now()
	iters := 0
	for time.Since(start) < duration {
		fn()
		iters++
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&endMem)

	if iters == 0 {
		iters = 1
	}

	allocBytes := int64(endMem.TotalAlloc - startMem.TotalAlloc)
	allocs := int64(endMem.Mallocs - startMem.Mallocs)

	nsPerOp := elapsed.Nanoseconds() / int64(iters)
	bPerOp := allocBytes / int64(iters)
	aPerOp := allocs / int64(iters)

	var tp string
	if bytesProcessed > 0 {
		mb := float64(bytesProcessed*int64(iters)) / (1024 * 1024)
		sec := elapsed.Seconds()
		if sec > 0 {
			tp = fmt.Sprintf("%.2f MB/s", mb/sec)
		}
	} else {
		tp = "-"
	}

	return BenchMetric{
		Name:        name,
		Iterations:  iters,
		Duration:    elapsed,
		NsPerOp:     nsPerOp,
		BytesPerOp:  bPerOp,
		AllocsPerOp: aPerOp,
		Throughput:  tp,
	}
}

func formatLatency(ns int64) string {
	if ns < 1000 {
		return fmt.Sprintf("%d ns/op", ns)
	} else if ns < 1000000 {
		return fmt.Sprintf("%.2f µs/op", float64(ns)/1000.0)
	} else if ns < 1000000000 {
		return fmt.Sprintf("%.2f ms/op", float64(ns)/1000000.0)
	}
	return fmt.Sprintf("%.2f s/op", float64(ns)/1000000000.0)
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B/op", b)
	} else if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB/op", float64(b)/1024.0)
	}
	return fmt.Sprintf("%.1f MB/op", float64(b)/(1024.0*1024.0))
}

func displayBenchTable(metrics []BenchMetric) {
	fmt.Println("\n" + strings.Repeat("=", 98))
	fmt.Printf(" %-30s | %-10s | %-13s | %-11s | %-9s | %s\n",
		"Micro-Benchmark", "Iterações", "Latência", "Memória", "Alocações", "Throughput")
	fmt.Println(strings.Repeat("-", 98))
	for _, m := range metrics {
		fmt.Printf(" %-30s | %-10d | %-13s | %-11s | %-9s | %s\n",
			m.Name, m.Iterations, formatLatency(m.NsPerOp), formatBytes(m.BytesPerOp), fmt.Sprintf("%d/op", m.AllocsPerOp), m.Throughput)
	}
	fmt.Println(strings.Repeat("=", 98) + "\n")
}

func runBenchmarks() {
	fmt.Println("🚀 Executando suíte de micro-benchmarks do My-Memory...")
	fmt.Println("Avaliando: TurboQuant 4-bit, Reciprocal Rank Fusion (RRF), SHA-256 Hashing e Markdown Parsing")

	dim := EmbeddingDim
	tq := turboquant.NewQuantizer(dim)

	v1 := make([]float32, dim)
	v2 := make([]float32, dim)
	for i := range v1 {
		v1[i] = float32((i*7+13)%100) / 100.0
		v2[i] = float32((i*11+17)%100) / 100.0
	}
	compressed, _ := tq.Quantize(v1)
	rotatedQuery := tq.RotateQuery(v2)

	buf64K := make([]byte, 64*1024)
	for i := range buf64K {
		buf64K[i] = byte(i % 256)
	}
	buf1M := make([]byte, 1024*1024)
	for i := range buf1M {
		buf1M[i] = byte(i % 256)
	}

	generateSources := func(n int) []store.RankedResultSource {
		fts := make([]store.SearchResult, n)
		vec := make([]store.SearchResult, n)
		for i := 0; i < n; i++ {
			fts[i] = store.SearchResult{ChunkID: fmt.Sprintf("doc-%d#0", i), DocumentID: fmt.Sprintf("doc-%d", i)}
			vec[i] = store.SearchResult{ChunkID: fmt.Sprintf("doc-%d#0", (i+n/2)%n), DocumentID: fmt.Sprintf("doc-%d", (i+n/2)%n)}
		}
		return []store.RankedResultSource{
			{Name: "fts", Results: fts},
			{Name: "vector", Results: vec},
		}
	}
	sources100 := generateSources(100)
	sources1000 := generateSources(1000)

	sampleMarkdown := `# Arquitetura do Sistema My-Memory
Este documento descreve a integração com [[sqlite-vec]] e [[pgvector]].
Conexão epistêmica: [[ADR-007]] e [[turboquant]].
#architecture #ai-memory #go #database

## Chunks e Embeddings
Usamos representações vetoriais de 768 dimensões comprimidas com [[TurboQuant 4-bit]].
A velocidade de busca é maximizada com [[RRF]] e CTEs recursivas.
`

	dur := 150 * time.Millisecond
	var metrics []BenchMetric

	metrics = append(metrics, runBenchLoop("TurboQuant Quantize (4-bit)", dur, 0, func() {
		_, _ = tq.Quantize(v1)
	}))

	metrics = append(metrics, runBenchLoop("TurboQuant Dequantize (4-bit)", dur, 0, func() {
		_ = tq.Dequantize(compressed)
	}))

	metrics = append(metrics, runBenchLoop("TurboQuant Dot Product (4-bit)", dur, 0, func() {
		_ = tq.DotProduct(rotatedQuery, compressed)
	}))

	metrics = append(metrics, runBenchLoop("Float32 Dot Product (768-dim)", dur, 0, func() {
		var sum float32
		for i := 0; i < dim; i++ {
			sum += v1[i] * v2[i]
		}
		_ = sum
	}))

	metrics = append(metrics, runBenchLoop("RRF Fusão (100 itens)", dur, 0, func() {
		_ = store.FuseSearchResults(sources100, 60, 10)
	}))

	metrics = append(metrics, runBenchLoop("RRF Fusão (1.000 itens)", dur, 0, func() {
		_ = store.FuseSearchResults(sources1000, 60, 20)
	}))

	metrics = append(metrics, runBenchLoop("SHA-256 Hashing (64 KB)", dur, int64(len(buf64K)), func() {
		_ = store.CalculateContentHash(buf64K)
	}))

	metrics = append(metrics, runBenchLoop("SHA-256 Hashing (1 MB)", dur, int64(len(buf1M)), func() {
		_ = store.CalculateContentHash(buf1M)
	}))

	metrics = append(metrics, runBenchLoop("Markdown Conexões (Wikilinks)", dur, int64(len(sampleMarkdown)), func() {
		_ = parser.ExtractConnections(sampleMarkdown)
	}))

	metrics = append(metrics, runBenchLoop("Markdown Chunking (200w/30o)", dur, int64(len(sampleMarkdown)), func() {
		_ = parser.ChunkText(sampleMarkdown, 200, 30)
	}))

	displayBenchTable(metrics)
}
