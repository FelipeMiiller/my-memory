package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// formatVaultMTime renderiza o instante de edição de um documento.
//
// mtime = 0 não é formatado como 1970: é "não sabemos quando essa nota foi
// editada", e fingir que foi em 1970 mentiria sobre a idade do conhecimento.
func formatVaultMTime(mtime int64) string {
	if mtime <= 0 {
		return "desconhecido (documento indexado antes do ADR-057)"
	}
	return time.Unix(mtime, 0).UTC().Format(time.RFC3339)
}

// runFileCLI executa o subcomando mem file direcionando para stdout
func runFileCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runFileCommand(ctx, defaultRepo, args, os.Stdout)
}

// runFileCommand reporta os metadados de um documento: path, título, tamanho e
// instante de edição.
//
// Tudo vem do índice. O mtime gravado na indexação é o que responde "quando
// essa nota mudou", que é a pergunta que updated_at (quando nós indexamos) não
// consegue responder — é o motivo de a ADR-057 existir.
func runFileCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	fileCmd := flag.NewFlagSet("file", flag.ContinueOnError)
	fileCmd.SetOutput(out)

	flags := registerVaultFlags(fileCmd)

	if err := fileCmd.Parse(args); err != nil {
		return err
	}

	if fileCmd.NArg() == 0 {
		return fmt.Errorf("mem file exige o caminho da nota (ex: mem file docs/adr/001.md)")
	}

	query := vaultSlash(fileCmd.Arg(0))

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	doc, err := resolveVaultDocument(ctx, reader, query)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Path:    %s\n", doc.Path)
	fmt.Fprintf(out, "Title:   %s\n", doc.Title)
	fmt.Fprintf(out, "Size:    %d bytes\n", doc.Size)
	fmt.Fprintf(out, "MTime:   %s\n", formatVaultMTime(doc.MTime))
	return nil
}

// runRecentsCLI executa o subcomando mem recents direcionando para stdout
func runRecentsCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runRecentsCommand(ctx, defaultRepo, args, os.Stdout)
}

// runRecentsCommand lista os documentos por ordem de edição decrescente.
func runRecentsCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	recentsCmd := flag.NewFlagSet("recents", flag.ContinueOnError)
	recentsCmd.SetOutput(out)

	limit := recentsCmd.Int("limit", 10, "Número máximo de documentos a exibir (padrão: 10)")
	flags := registerVaultFlags(recentsCmd)

	if err := recentsCmd.Parse(args); err != nil {
		return err
	}

	if *limit <= 0 {
		return fmt.Errorf("--limit precisa ser maior que zero (recebi %d)", *limit)
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	docs, err := reader.ListRecent(ctx, *limit)
	if err != nil {
		return fmt.Errorf("falha ao listar documentos recentes: %w", err)
	}

	for _, doc := range docs {
		fmt.Fprintf(out, "%s  %s\n", formatVaultMTime(doc.MTime), doc.Path)
	}
	return nil
}
