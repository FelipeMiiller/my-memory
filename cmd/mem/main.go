package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/canvas"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/repo"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

const EmbeddingDim = 768

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	ctx := context.Background()
	emb := embedder.NewOllamaClient("", "nomic-embed-text")
	tq := turboquant.NewQuantizer(EmbeddingDim)
	defaultRepo := repo.DetectRepository(".")

	switch os.Args[1] {
	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		dirPath := indexCmd.String("dir", "", "Caminho da pasta com arquivos Markdown")
		force := indexCmd.Bool("force", false, "Força a reindexação completa ignorando o cache SHA-256")
		dbPath := indexCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := indexCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := indexCmd.String("repo", defaultRepo, "Identificador/slug do repositório")
		indexCmd.Parse(os.Args[2:])

		target := *dirPath
		if target == "" && indexCmd.NArg() > 0 {
			target = indexCmd.Arg(0)
		}
		if target == "" {
			fmt.Println("Uso: mem index [--force] [--db <caminho>] [--postgres <url>] [--repo <nome>] <pasta_com_markdown>")
			return
		}

		if *pgURL != "" {
			pgStore, err := store.NewPostgresStore(*pgURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runIndexPostgres(ctx, pgStore, emb, *targetRepo, target, *force)
		} else {
			database, err := db.InitDB(*dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runIndexSQLite(ctx, database, emb, tq, target, *force)
		}

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		useTurbo := searchCmd.Bool("tq", false, "Usar busca via TurboQuant (4-bits, SQLite)")
		mode := searchCmd.String("mode", "hybrid", "Modo de busca: 'hybrid' (FTS+vetor+grafo via RRF), 'vector' (apenas k-NN), 'fts' (apenas léxico)")
		k := searchCmd.Int("k", 60, "Constante de suavização do algoritmo RRF (padrão: 60)")
		limit := searchCmd.Int("limit", 5, "Número máximo de resultados (padrão: 5)")
		dbPath := searchCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := searchCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := searchCmd.String("repo", defaultRepo, "Identificador/slug do repositório para filtrar")
		searchCmd.Parse(os.Args[2:])

		query := strings.Join(searchCmd.Args(), " ")
		if query == "" {
			fmt.Println("Uso: mem search [--mode hybrid|vector|fts] [-tq] [--k 60] [--limit 5] [--db <caminho>] [--postgres <url>] [--repo <nome>] \"sua pergunta aqui\"")
			return
		}

		if *pgURL != "" {
			pgStore, err := store.NewPostgresStore(*pgURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runSearchPostgres(ctx, pgStore, emb, query, *targetRepo, *mode, *limit, *k)
		} else {
			database, err := db.InitDB(*dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runSearchSQLite(ctx, database, emb, tq, query, *mode, *useTurbo, *limit, *k)
		}

	case "mcp":
		mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
		dbPath := mcpCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := mcpCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := mcpCmd.String("repo", defaultRepo, "Identificador padrão do repositório")
		mcpCmd.Parse(os.Args[2:])

		var pgStore *store.PostgresStore
		var database *sql.DB
		var err error

		if *pgURL != "" {
			pgStore, err = store.NewPostgresStore(*pgURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] Erro conectando ao PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			fmt.Fprintf(os.Stderr, "[mcp] Conectado ao PostgreSQL com pgvector (repo padrão: %s)\n", *targetRepo)
		} else {
			if _, statErr := os.Stat(*dbPath); os.IsNotExist(statErr) {
				fmt.Fprintf(os.Stderr, "[mcp] Aviso: Banco de dados '%s' não encontrado. Um novo banco será criado na primeira gravação.\n", *dbPath)
			}
			database, err = db.InitDB(*dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[mcp] Erro inicializando banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			fmt.Fprintf(os.Stderr, "[mcp] Conectado ao SQLite: %s\n", *dbPath)
		}

		runMCPServer(ctx, pgStore, database, emb, *targetRepo)

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
		canvasNode := exportCmd.String("canvas", "", "Nome ou identificador da nota raiz para exportar subgrafo para JSON Canvas (.canvas)")
		depth := exportCmd.Int("depth", 1, "Profundidade máxima de vizinhos no grafo (padrão: 1)")
		outFile := exportCmd.String("out", "", "Caminho do arquivo .canvas de saída (padrão: <nota>.canvas)")
		dbPath := exportCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := exportCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := exportCmd.String("repo", defaultRepo, "Identificador/slug do repositório")
		exportCmd.Parse(os.Args[2:])

		node := *canvasNode
		if node == "" && exportCmd.NArg() > 0 {
			node = exportCmd.Arg(0)
		}
		if node == "" {
			fmt.Println("Uso: mem export --canvas <nota> [--depth 1] [--out <saida.canvas>] [--db <caminho>] [--postgres <url>] [--repo <slug>]")
			return
		}

		if *outFile == "" {
			safeName := strings.ReplaceAll(node, "/", "_")
			safeName = strings.ReplaceAll(safeName, "\\", "_")
			safeName = strings.TrimSuffix(safeName, ".md")
			*outFile = safeName + ".canvas"
		}

		runExportCanvas(ctx, *pgURL, *dbPath, *targetRepo, node, *depth, *outFile)

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("=== My-Memory CLI (SQLite / PostgreSQL com pgvector / TurboQuant / MCP) ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  mem index [--force] [--db <arq>] [--postgres <url>] [--repo <slug>] <pasta>")
	fmt.Println("      Indexa notas Markdown com cache incremental SHA-256 (use --force para reconstruir)")
	fmt.Println("  mem search [--mode hybrid|vector|fts] [-tq] [--k 60] [--limit 5] [--db <arq>] [--postgres <url>] [--repo <slug>] \"<pergunta>\"")
	fmt.Println("      Busca híbrida com Reciprocal Rank Fusion (RRF), FTS5/tsvector, vetores e grafo")
	fmt.Println("  mem export --canvas <nota> [--depth 1] [--out <arquivo.canvas>] [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Exporta um subgrafo em torno de uma nota no formato aberto JSON Canvas (.canvas) do Obsidian")
	fmt.Println("  mem mcp [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Inicia servidor Model Context Protocol via stdio para agentes de IA (Claude, Cursor, etc)")
	fmt.Println()
	fmt.Println("Variáveis de ambiente:")
	fmt.Println("  MY_MEMORY_PG_URL - URL de conexão padrão para o PostgreSQL (ex: postgres://user:pass@localhost:5432/memory?sslmode=disable)")
	fmt.Println("  MY_MEMORY_REPO   - Força o slug do repositório atual (sobrescreve auto-detecção git)")
}

func runIndexPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, targetRepo, rootDir string, force bool) {
	fmt.Printf("🔍 Indexando notas no PostgreSQL (pgvector) para repo [%s] em: %s\n", targetRepo, rootDir)
	totalCount := 0
	indexedCount := 0
	cachedCount := 0

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		totalCount++
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), ".md")
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

		// 1. Salva documento com content_hash
		if err := s.InsertDocument(ctx, targetRepo, docID, path, title, time.Now().Unix(), currentHash); err != nil {
			return err
		}

		// 2. Extrai e salva conexões do grafo ([[wikilinks]])
		connections := parser.ExtractConnections(content)
		for _, link := range connections.OutgoingLinks {
			_ = s.InsertEdge(ctx, targetRepo, docID, link, "links_to")
		}

		// 3. Divide em chunks e gera embeddings
		chunks := parser.ChunkText(content, 200, 30)
		for i, c := range chunks {
			chunkID := fmt.Sprintf("%s#%d", docID, i)
			vec, err := emb.GenerateEmbedding(c)
			if err != nil {
				fmt.Printf("Aviso: falha ao gerar embedding para %s (Ollama está rodando?)\n", chunkID)
				continue
			}

			_ = s.InsertChunk(ctx, targetRepo, chunkID, docID, c, i, vec)
		}

		indexedCount++
		fmt.Printf("✔ Indexado no Postgres: %s (%d links, %d chunks)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados no PostgreSQL (%d indexados, %d em cache).\n", totalCount, indexedCount, cachedCount)
	}
}

func runIndexSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, rootDir string, force bool) {
	fmt.Printf("🔍 Indexando notas no SQLite em: %s (com TurboQuant 4-bit ativado)\n", rootDir)
	totalCount := 0
	indexedCount := 0
	cachedCount := 0

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		totalCount++
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), ".md")
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

		// 1. Salva documento com content_hash
		if err := db.InsertDocument(ctx, database, docID, path, title, time.Now().Unix(), currentHash); err != nil {
			return err
		}

		// 2. Extrai e salva conexões do grafo ([[wikilinks]])
		connections := parser.ExtractConnections(content)
		for _, link := range connections.OutgoingLinks {
			_ = db.InsertEdge(ctx, database, docID, link, "links_to")
		}

		// 3. Divide em chunks e gera embeddings
		chunks := parser.ChunkText(content, 200, 30)
		for i, c := range chunks {
			chunkID := fmt.Sprintf("%s#%d", docID, i)
			vec, err := emb.GenerateEmbedding(c)
			if err != nil {
				fmt.Printf("Aviso: falha ao gerar embedding para %s (Ollama está rodando?)\n", chunkID)
				continue
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
		fmt.Printf("✔ Indexado no SQLite: %s (%d links, %d chunks comprimidos)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados no SQLite (%d indexados, %d em cache com TurboQuant).\n", totalCount, indexedCount, cachedCount)
	}
}

func runSearchPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, query, targetRepo, mode string, limit, k int) {
	fmt.Printf("🔎 Buscando no PostgreSQL (pgvector) [Repo: %s, Modo: %s] por: \"%s\"\n\n", targetRepo, mode, query)

	var results []store.SearchResult
	var err error

	switch strings.ToLower(mode) {
	case "fts":
		results, err = s.SearchFTS(ctx, targetRepo, query, limit)
	case "vector":
		queryVec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", embErr)
			return
		}
		results, err = s.SearchKNN(ctx, targetRepo, queryVec, limit)
	default: // hybrid
		var queryVec []float32
		vec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Aviso: Ollama offline ou falha ao gerar embedding (%v). Executando fallback para busca textual FTS.\n\n", embErr)
		} else {
			queryVec = vec
		}
		results, err = s.SearchHybridRRF(ctx, targetRepo, query, queryVec, limit, k)
	}

	if err != nil {
		fmt.Printf("Erro na busca: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("Nenhum resultado encontrado.")
		return
	}

	for i, res := range results {
		var headers []string
		if res.Score > 0 {
			headers = append(headers, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headers = append(headers, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		if res.Repository != "" {
			headers = append(headers, fmt.Sprintf("Repo: %s", res.Repository))
		}
		headers = append(headers, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Printf("--- [%d] %s ---\n", i+1, strings.Join(headers, " | "))
		if len(res.Sources) > 0 {
			fmt.Printf("📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
		}
		fmt.Println(res.Content)
		if len(res.Neighbors) > 0 {
			fmt.Printf("🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		fmt.Println()
	}
}

func runSearchSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, query, mode string, useTurbo bool, limit, k int) {
	vecSubmode := "sqlite-vec"
	if useTurbo {
		vecSubmode = "TurboQuant"
	}
	fmt.Printf("🔎 Buscando no SQLite por: \"%s\" [Modo: %s (%s)]\n\n", query, mode, vecSubmode)

	var results []db.SearchResult
	var err error

	switch strings.ToLower(mode) {
	case "fts":
		results, err = db.SearchFTS(ctx, database, query, limit)
	case "vector":
		queryVec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", embErr)
			return
		}
		if useTurbo {
			results, err = db.SearchTurboQuant(ctx, database, tq, queryVec, limit)
		} else {
			results, err = db.SearchKNN(ctx, database, queryVec, limit)
		}
	default: // hybrid
		var queryVec []float32
		vec, embErr := emb.GenerateEmbedding(query)
		if embErr != nil {
			fmt.Printf("Aviso: Ollama offline ou falha ao gerar embedding (%v). Executando fallback para busca textual FTS.\n\n", embErr)
		} else {
			queryVec = vec
		}
		results, err = db.SearchHybridRRF(ctx, database, tq, query, queryVec, limit, k, useTurbo)
	}

	if err != nil {
		fmt.Printf("Erro na busca: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("Nenhum resultado encontrado.")
		return
	}

	for i, res := range results {
		var headers []string
		if res.Score > 0 {
			headers = append(headers, fmt.Sprintf("Score RRF: %.4f", res.Score))
		}
		if res.Distance > 0 {
			headers = append(headers, fmt.Sprintf("Distância: %.4f", res.Distance))
		}
		headers = append(headers, fmt.Sprintf("Documento: %s", res.DocumentID))

		fmt.Printf("--- [%d] %s ---\n", i+1, strings.Join(headers, " | "))
		if len(res.Sources) > 0 {
			fmt.Printf("📊 Fontes RRF: [%s]\n", strings.Join(res.Sources, ", "))
		}
		fmt.Println(res.Content)
		if len(res.Neighbors) > 0 {
			fmt.Printf("🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		fmt.Println()
	}
}

func runMCPServer(ctx context.Context, pgStore *store.PostgresStore, database *sql.DB, emb *embedder.OllamaClient, defaultRepo string) {
	srv := mcp.NewServer("my-memory", "1.0.0", os.Stdin, os.Stdout, os.Stderr)

	if pgStore != nil {
		srv.SetAdvancedSearchHandler(func(ctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
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

			var pgResults []store.SearchResult
			var err error

			switch params.Mode {
			case "fts":
				pgResults, err = pgStore.SearchFTS(ctx, repo, params.Query, limit)
			case "vector":
				queryVec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr != nil {
					return nil, fmt.Errorf("falha ao gerar embedding: %w", embErr)
				}
				pgResults, err = pgStore.SearchKNN(ctx, repo, queryVec, limit)
			default: // hybrid
				var queryVec []float32
				vec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr == nil {
					queryVec = vec
				}
				pgResults, err = pgStore.SearchHybridRRF(ctx, repo, params.Query, queryVec, limit, k)
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
				}
			}
			return mcpResults, nil
		})

		srv.SetNeighborsHandler(func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
			if repo == "" {
				repo = defaultRepo
			}
			return pgStore.GetNodeNeighbors(ctx, repo, nodeID, maxDepth)
		})
	} else if database != nil {
		tq := turboquant.NewQuantizer(EmbeddingDim)
		srv.SetAdvancedSearchHandler(func(ctx context.Context, params mcp.SearchParams) ([]mcp.SearchResult, error) {
			limit := params.Limit
			if limit <= 0 {
				limit = 5
			}
			k := params.K
			if k <= 0 {
				k = 60
			}

			var dbResults []db.SearchResult
			var err error

			switch params.Mode {
			case "fts":
				dbResults, err = db.SearchFTS(ctx, database, params.Query, limit)
			case "vector":
				queryVec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr != nil {
					return nil, fmt.Errorf("falha ao gerar embedding: %w", embErr)
				}
				dbResults, err = db.SearchKNN(ctx, database, queryVec, limit)
			default: // hybrid
				var queryVec []float32
				vec, embErr := emb.GenerateEmbedding(params.Query)
				if embErr == nil {
					queryVec = vec
				}
				dbResults, err = db.SearchHybridRRF(ctx, database, tq, params.Query, queryVec, limit, k, false)
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
				}
			}
			return mcpResults, nil
		})

		srv.SetNeighborsHandler(func(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
			return db.GetNodeNeighbors(ctx, database, nodeID, maxDepth)
		})
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

