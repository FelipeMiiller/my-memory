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
		dbPath := indexCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := indexCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := indexCmd.String("repo", defaultRepo, "Identificador/slug do repositório")
		indexCmd.Parse(os.Args[2:])

		target := *dirPath
		if target == "" && indexCmd.NArg() > 0 {
			target = indexCmd.Arg(0)
		}
		if target == "" {
			fmt.Println("Uso: mem index [--db <caminho>] [--postgres <url>] [--repo <nome>] <pasta_com_markdown>")
			return
		}

		if *pgURL != "" {
			pgStore, err := store.NewPostgresStore(*pgURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runIndexPostgres(ctx, pgStore, emb, *targetRepo, target)
		} else {
			database, err := db.InitDB(*dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runIndexSQLite(ctx, database, emb, tq, target)
		}

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		useTurbo := searchCmd.Bool("tq", false, "Usar busca via TurboQuant (4-bits, SQLite)")
		dbPath := searchCmd.String("db", "memory.db", "Caminho do arquivo SQLite")
		pgURL := searchCmd.String("postgres", os.Getenv("MY_MEMORY_PG_URL"), "URL de conexão PostgreSQL (com pgvector)")
		targetRepo := searchCmd.String("repo", defaultRepo, "Identificador/slug do repositório para filtrar")
		searchCmd.Parse(os.Args[2:])

		query := strings.Join(searchCmd.Args(), " ")
		if query == "" {
			fmt.Println("Uso: mem search [-tq] [--db <caminho>] [--postgres <url>] [--repo <nome>] \"sua pergunta aqui\"")
			return
		}

		if *pgURL != "" {
			pgStore, err := store.NewPostgresStore(*pgURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no PostgreSQL: %v\n", err)
				os.Exit(1)
			}
			defer pgStore.Close()
			runSearchPostgres(ctx, pgStore, emb, query, *targetRepo)
		} else {
			database, err := db.InitDB(*dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao inicializar banco SQLite: %v\n", err)
				os.Exit(1)
			}
			defer database.Close()
			runSearchSQLite(ctx, database, emb, tq, query, *useTurbo)
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

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("=== My-Memory CLI (SQLite / PostgreSQL com pgvector / TurboQuant / MCP) ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  mem index [--db <arq>] [--postgres <url>] [--repo <slug>] <pasta>")
	fmt.Println("      Indexa notas Markdown, links [[wikilinks]], FTS5 e vetores")
	fmt.Println("  mem search [-tq] [--db <arq>] [--postgres <url>] [--repo <slug>] \"<pergunta>\"")
	fmt.Println("      Busca semântica k-NN com expansão de grafo")
	fmt.Println("  mem mcp [--db <arq>] [--postgres <url>] [--repo <slug>]")
	fmt.Println("      Inicia servidor Model Context Protocol via stdio para agentes de IA (Claude, Cursor, etc)")
	fmt.Println()
	fmt.Println("Variáveis de ambiente:")
	fmt.Println("  MY_MEMORY_PG_URL - URL de conexão padrão para o PostgreSQL (ex: postgres://user:pass@localhost:5432/memory?sslmode=disable)")
	fmt.Println("  MY_MEMORY_REPO   - Força o slug do repositório atual (sobrescreve auto-detecção git)")
}

func runIndexPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, targetRepo, rootDir string) {
	fmt.Printf("🔍 Indexando notas no PostgreSQL (pgvector) para repo [%s] em: %s\n", targetRepo, rootDir)
	count := 0

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), ".md")
		docID := path

		// 1. Salva documento
		if err := s.InsertDocument(ctx, targetRepo, docID, path, title, time.Now().Unix()); err != nil {
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

		count++
		fmt.Printf("✔ Indexado no Postgres: %s (%d links, %d chunks)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados no PostgreSQL.\n", count)
	}
}

func runIndexSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, rootDir string) {
	fmt.Printf("🔍 Indexando notas no SQLite em: %s (com TurboQuant 4-bit ativado)\n", rootDir)
	count := 0

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(contentBytes)
		title := strings.TrimSuffix(d.Name(), ".md")
		docID := path

		// 1. Salva documento
		if err := db.InsertDocument(ctx, database, docID, path, title, time.Now().Unix()); err != nil {
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

		count++
		fmt.Printf("✔ Indexado no SQLite: %s (%d links, %d chunks comprimidos)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados com TurboQuant.\n", count)
	}
}

func runSearchPostgres(ctx context.Context, s *store.PostgresStore, emb *embedder.OllamaClient, query, targetRepo string) {
	fmt.Printf("🔎 Buscando no PostgreSQL (pgvector) [Repo: %s] por: \"%s\"\n\n", targetRepo, query)

	queryVec, err := emb.GenerateEmbedding(query)
	if err != nil {
		fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", err)
		return
	}

	results, err := s.SearchKNN(ctx, targetRepo, queryVec, 5)
	if err != nil {
		fmt.Printf("Erro na busca: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("Nenhum resultado encontrado.")
		return
	}

	for i, res := range results {
		fmt.Printf("--- [%d] Distância: %.4f | Repo: %s | Documento: %s ---\n", i+1, res.Distance, res.Repository, res.DocumentID)
		fmt.Println(res.Content)
		if len(res.Neighbors) > 0 {
			fmt.Printf("🕸 Conexões no Grafo: %s\n", strings.Join(res.Neighbors, ", "))
		}
		fmt.Println()
	}
}

func runSearchSQLite(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, query string, useTurbo bool) {
	mode := "sqlite-vec (float32)"
	if useTurbo {
		mode = "TurboQuant (4-bit, 384 bytes)"
	}
	fmt.Printf("🔎 Buscando no SQLite por: \"%s\" [Modo: %s]\n\n", query, mode)

	queryVec, err := emb.GenerateEmbedding(query)
	if err != nil {
		fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", err)
		return
	}

	var results []db.SearchResult
	if useTurbo {
		results, err = db.SearchTurboQuant(ctx, database, tq, queryVec, 5)
	} else {
		results, err = db.SearchKNN(ctx, database, queryVec, 5)
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
		fmt.Printf("--- [%d] Distância: %.4f | Documento: %s ---\n", i+1, res.Distance, res.DocumentID)
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
		srv.SetSearchHandler(func(ctx context.Context, repo string, query string, limit int) ([]mcp.SearchResult, error) {
			if repo == "" {
				repo = defaultRepo
			}
			queryVec, err := emb.GenerateEmbedding(query)
			if err != nil {
				return nil, fmt.Errorf("falha ao gerar embedding: %w", err)
			}

			pgResults, err := pgStore.SearchKNN(ctx, repo, queryVec, limit)
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
		srv.SetSearchHandler(func(ctx context.Context, repo string, query string, limit int) ([]mcp.SearchResult, error) {
			queryVec, err := emb.GenerateEmbedding(query)
			if err != nil {
				return nil, fmt.Errorf("falha ao gerar embedding: %w", err)
			}

			dbResults, err := db.SearchKNN(ctx, database, queryVec, limit)
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
