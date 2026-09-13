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
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

const EmbeddingDim = 768

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	dbPath := "memory.db"
	database, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Printf("Erro ao inicializar banco: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	ctx := context.Background()
	emb := embedder.NewOllamaClient("", "nomic-embed-text")
	tq := turboquant.NewQuantizer(EmbeddingDim)

	switch os.Args[1] {
	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		dirPath := indexCmd.String("dir", "", "Caminho da pasta com arquivos Markdown")
		indexCmd.Parse(os.Args[2:])

		target := *dirPath
		if target == "" && indexCmd.NArg() > 0 {
			target = indexCmd.Arg(0)
		}
		if target == "" {
			fmt.Println("Uso: mem index <pasta_com_markdown>")
			return
		}
		runIndex(ctx, database, emb, tq, target)

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		useTurbo := searchCmd.Bool("tq", false, "Usar busca via TurboQuant (4-bits)")
		searchCmd.Parse(os.Args[2:])

		query := strings.Join(searchCmd.Args(), " ")
		if query == "" {
			fmt.Println("Uso: mem search [-tq] \"sua pergunta aqui\"")
			return
		}
		runSearch(ctx, database, emb, tq, query, *useTurbo)

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("=== My-Memory CLI (com TurboQuant) ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  mem index <pasta>          - Indexa notas Markdown, links [[wikilinks]], FTS5 e TurboQuant 4-bit")
	fmt.Println("  mem search \"<pergunta>\"    - Busca semântica k-NN via sqlite-vec padrão com expansão de grafo")
	fmt.Println("  mem search -tq \"<pergunta>\" - Busca semântica ultrarrápida via TurboQuant (4-bits)")
}

func runIndex(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, rootDir string) {
	fmt.Printf("🔍 Indexando notas em: %s (com TurboQuant 4-bit ativado)\n", rootDir)
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
		fmt.Printf("✔ Indexado: %s (%d links, %d chunks comprimidos)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados com TurboQuant.\n", count)
	}
}

func runSearch(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, tq *turboquant.Quantizer, query string, useTurbo bool) {
	mode := "sqlite-vec (float32)"
	if useTurbo {
		mode = "TurboQuant (4-bit, 384 bytes)"
	}
	fmt.Printf("🔎 Buscando por: \"%s\" [Modo: %s]\n\n", query, mode)

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
