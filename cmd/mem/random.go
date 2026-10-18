package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
)

// RandomResponse é o JSON de `mem random --json`.
type RandomResponse struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	MTime int64  `json:"mtime"`
	Size  int64  `json:"size"`
}

// runRandomCLI executa o subcomando mem random direcionando para stdout
func runRandomCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runRandomCommand(ctx, defaultRepo, args, os.Stdout)
}

// runRandomCommand sorteia um documento entre os indexados.
//
// Reaproveita a consulta de `mem files` em vez de abrir uma tabela nova: a
// lista completa é o conjunto sorteável, e o caminho de leitura é o mesmo que
// o usuário já conhece.
//
// Índice vazio é erro, não resultado. Sortear "nenhuma nota" e sair zero
// deixaria `mem random` indistinguível de um vault que existe e está vazio —
// e é exatamente esse estado que o usuário precisa ver.
func runRandomCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	randomCmd := flag.NewFlagSet("random", flag.ContinueOnError)
	randomCmd.SetOutput(out)

	jsonOutput := randomCmd.Bool("json", false, "Exibe a nota sorteada em JSON estruturado")
	flags := registerVaultFlags(randomCmd)

	if err := randomCmd.Parse(args); err != nil {
		return err
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	docs, err := reader.ListDocuments(ctx)
	if err != nil {
		return fmt.Errorf("falha ao listar documentos do vault: %w", err)
	}
	if len(docs) == 0 {
		return fmt.Errorf("o índice está vazio: nenhum documento indexado para sortear (execute 'mem index')")
	}

	doc := docs[rand.Intn(len(docs))]

	if *jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(RandomResponse{
			Path:  doc.Path,
			Title: doc.Title,
			MTime: doc.MTime,
			Size:  doc.Size,
		})
	}

	fmt.Fprintf(out, "%s\n", doc.Path)
	if doc.Title != "" {
		fmt.Fprintf(out, "  %s\n", doc.Title)
	}
	return nil
}
