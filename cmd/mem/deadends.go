package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

// runDeadendsCLI executa o subcomando mem deadends direcionando para stdout
func runDeadendsCLI(ctx context.Context, defaultRepo string, args []string) error {
	return runDeadendsCommand(ctx, defaultRepo, args, os.Stdout)
}

// runDeadendsCommand lista as folhas do grafo: notas que apontam para alguma
// coisa e não são apontadas por ninguém.
//
// A consulta mora no engine (db.ListVaultDeadends e
// PostgresStore.ListVaultDeadends) para que a definição de deadend seja uma só.
func runDeadendsCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error {
	deadendsCmd := flag.NewFlagSet("deadends", flag.ContinueOnError)
	deadendsCmd.SetOutput(out)

	flags := registerVaultFlags(deadendsCmd)

	if err := deadendsCmd.Parse(args); err != nil {
		return err
	}

	reader, err := flags.open(defaultRepo)
	if err != nil {
		return err
	}
	defer reader.Close()

	docs, err := reader.ListDeadends(ctx)
	if err != nil {
		return fmt.Errorf("falha ao listar deadends: %w", err)
	}

	for _, doc := range docs {
		fmt.Fprintln(out, doc.Path)
	}
	return nil
}
