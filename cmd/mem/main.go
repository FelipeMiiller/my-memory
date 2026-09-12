package main

import (
	"database/sql"
	"context"
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
)

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
		runIndex(ctx, database, emb, target)

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("Uso: mem search \"sua pergunta aqui\"")
			return
		}
		query := strings.Join(os.Args[2:], " ")
		runSearch(ctx, database, emb, query)

	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("=== My-Memory CLI ===")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  mem index <pasta>   - Indexa notas Markdown, extrai [[wikilinks]], gera vetores")
	fmt.Println("  mem search <query>  - Faz busca semântica k-NN e expande o grafo de relações")
}

func runIndex(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, rootDir string) {
	fmt.Printf("🔍 Indexando notas em: %s\n", rootDir)
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
			_ = db.InsertChunk(ctx, database, chunkID, docID, c, i, vec)
		}

		count++
		fmt.Printf("✔ Indexado: %s (%d links, %d chunks)\n", title, len(connections.OutgoingLinks), len(chunks))
		return nil
	})

	if err != nil {
		fmt.Printf("Erro durante a indexação: %v\n", err)
	} else {
		fmt.Printf("🎉 Concluído! %d documentos processados.\n", count)
	}
}

func runSearch(ctx context.Context, database *sql.DB, emb *embedder.OllamaClient, query string) {
	fmt.Printf("🔎 Buscando por: \"%s\"\n\n", query)

	queryVec, err := emb.GenerateEmbedding(query)
	if err != nil {
		fmt.Printf("Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v\n", err)
		return
	}

	results, err := db.SearchKNN(ctx, database, queryVec, 5)
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
