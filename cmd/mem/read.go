package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/store"
)

// runReadCLI executa o subcomando mem read direcionando para stdout
func runReadCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runReadCommand(ctx, defaultRepo, args, os.Stdout)
}

// runReadCommand imprime o Markdown bruto da nota.
//
// O texto vem de chunks.content, que é o arquivo como o indexador o leu —
// nunca do disco. É o que faz `mem read` responder num vault que só existe no
// Postgres, e o que evita que a saída divirja do que foi indexado.
func runReadCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	readCmd := flag.NewFlagSet("read", flag.ContinueOnError)
	readCmd.SetOutput(out)

	flags := registerVaultFlags(readCmd)

	if err := readCmd.Parse(args); err != nil {
		return err
	}

	if readCmd.NArg() == 0 {
		return fmt.Errorf("mem read exige o caminho da nota (ex: mem read docs/adr/001.md)")
	}

	query := vaultSlash(readCmd.Arg(0))

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	doc, err := resolveVaultDocument(ctx, reader, query)
	if err != nil {
		return err
	}

	content, err := reader.ReadContent(ctx, doc.ID)
	if err != nil {
		return fmt.Errorf("falha ao ler o conteúdo de '%s': %w", doc.Path, err)
	}

	fmt.Fprint(out, content)
	return nil
}

// resolveVaultDocument escolhe o documento que o usuário pediu.
//
// Path exato vence sempre. Sem match exato, o basename casa — e aí a
// ambiguidade é um erro reportado, não uma escolha silenciosa: com `a/x.md` e
// `b/x.md` indexados, `mem read x.md` não pode devolver um dos dois sem dizer
// qual escolheu.
func resolveVaultDocument(ctx context.Context, reader vaultReader, query string) (store.VaultDocument, error) {
	candidates, err := reader.FindByPath(ctx, query)
	if err != nil {
		return store.VaultDocument{}, fmt.Errorf("falha ao buscar '%s' no índice: %w", query, err)
	}
	return resolveVaultDocumentFrom(candidates, query)
}

// resolveVaultDocumentFrom aplica a regra de escolha sobre candidatos já
// buscado. Existe separado da consulta porque o servidor MCP tem os candidatos
// na mão e precisa da mesma decisão — escolher entre duas notas com o mesmo
// basename é regra de produto, não detalhe de SQL.
func resolveVaultDocumentFrom(candidates []store.VaultDocument, query string) (store.VaultDocument, error) {
	exact := make([]store.VaultDocument, 0, 1)
	for _, doc := range candidates {
		if vaultSlash(doc.Path) == query {
			exact = append(exact, doc)
		}
	}
	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
	default:
		return store.VaultDocument{}, ambiguousVaultPathError(query, exact)
	}

	switch len(candidates) {
	case 0:
		return store.VaultDocument{}, fmt.Errorf("nota '%s' não está no índice (use 'mem files' para ver o que existe)", query)
	case 1:
		return candidates[0], nil
	default:
		return store.VaultDocument{}, ambiguousVaultPathError(query, candidates)
	}
}

// ambiguousVaultPathError lista os candidatos para que o usuário-desambigue
// possa repetir o comando com o caminho completo.
func ambiguousVaultPathError(query string, candidates []store.VaultDocument) error {
	paths := make([]string, 0, len(candidates))
	for _, doc := range candidates {
		paths = append(paths, doc.Path)
	}
	return fmt.Errorf("'%s' é ambíguo: %d notas têm esse nome (%s). Use o caminho completo",
		filepath.Base(query), len(paths), strings.Join(paths, ", "))
}
