package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/store"
)

// vaultReader é a superfície de leitura que os comandos de paridade do Obsidian
// (files, folders, read, outline, wordcount, random, deadends, file, recents)
// consomem. Ela existe para que o comando seja escrito uma vez e funcione nos
// dois engines — inclusive num vault que só existe no Postgres, sem cópia local.
//
// Nenhum método toca o filesystem: todo comando lê o índice, que é o requisito
// de paridade para vault federado.
type vaultReader interface {
	// Close encerra a conexão aberta por vaultFlags.open.
	Close() error
	// ListDocuments devolve todos os documentos do vault, ordenados por path.
	ListDocuments(ctx context.Context) ([]store.VaultDocument, error)
}

type sqliteVaultReader struct {
	db *sql.DB
}

type postgresVaultReader struct {
	store *store.PostgresStore
	repo  string
}

func (r *sqliteVaultReader) Close() error { return r.db.Close() }

func (r *sqliteVaultReader) ListDocuments(ctx context.Context) ([]store.VaultDocument, error) {
	return db.ListVaultDocuments(ctx, r.db)
}

func (r *postgresVaultReader) Close() error { return r.store.Close() }

func (r *postgresVaultReader) ListDocuments(ctx context.Context) ([]store.VaultDocument, error) {
	return r.store.ListVaultDocuments(ctx, r.repo)
}

// vaultFlags são as flags de storage compartilhadas por todos os comandos de
// leitura do vault, para que a resolução de engine (ADR-040) fique escrita uma
// vez só.
type vaultFlags struct {
	dbPath     *string
	pgURL      *string
	storage    *string
	targetRepo *string
}

func registerVaultFlags(fs *flag.FlagSet) vaultFlags {
	return vaultFlags{
		dbPath:     fs.String("db", "", "Caminho do arquivo SQLite"),
		pgURL:      fs.String("postgres", "", "URL de conexão PostgreSQL"),
		storage:    fs.String("storage", "", "Força engine: 'sqlite' ou 'postgres'. Default = auto-detect (ADR-040)"),
		targetRepo: fs.String("repo", "", "Slug ou identificador do repositório"),
	}
}

// open resolve o engine (ADR-040) e devolve a superfície de leitura do vault.
func (f vaultFlags) open(defaultRepo string) (vaultReader, error) {
	cfg := resolveConfig()
	resolvedRepo, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *f.targetRepo, *f.dbPath, *f.pgURL, defaultRepo)

	var errOverride error
	resolvedDB, resolvedPG, errOverride = applyStorageOverride(*f.storage, resolvedDB, resolvedPG)
	if errOverride != nil {
		return nil, errOverride
	}

	if resolvedPG != "" {
		pgStore, err := store.NewPostgresStore(resolvedPG)
		if err != nil {
			return nil, fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
		}
		return &postgresVaultReader{store: pgStore, repo: resolvedRepo}, nil
	}

	database, err := db.InitDB(resolvedDB)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco SQLite '%s': %w", resolvedDB, err)
	}
	return &sqliteVaultReader{db: database}, nil
}
