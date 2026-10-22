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
	// FindByPath resolve um caminho (exato ou basename) em candidatos, para
	// que a desambiguação seja responsabilidade do comando.
	FindByPath(ctx context.Context, query string) ([]store.VaultDocument, error)
	// ReadContent devolve o Markdown bruto do documento remontando os chunks.
	ReadContent(ctx context.Context, docID string) (string, error)
	// ListDeadends devolve as notas com aresta de saída e sem aresta de entrada.
	ListDeadends(ctx context.Context) ([]store.VaultDocument, error)
	// ListRecent devolve os documentos por mtime decrescente, com mtime = 0 no fim.
	ListRecent(ctx context.Context, limit int) ([]store.VaultDocument, error)
	// ListTags devolve o inventário de tags do vault.
	ListTags(ctx context.Context, filter store.TagFilter) ([]store.VaultTag, error)
	// FindDocumentsByTag devolve os documentos que carregam a tag e suas descendentes.
	FindDocumentsByTag(ctx context.Context, tag string) ([]store.VaultDocument, error)
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

func (r *sqliteVaultReader) FindByPath(ctx context.Context, query string) ([]store.VaultDocument, error) {
	return db.FindVaultDocumentsByPath(ctx, r.db, query)
}

func (r *sqliteVaultReader) ReadContent(ctx context.Context, docID string) (string, error) {
	return db.ReadVaultDocumentContent(ctx, r.db, docID)
}

func (r *sqliteVaultReader) ListDeadends(ctx context.Context) ([]store.VaultDocument, error) {
	return db.ListVaultDeadends(ctx, r.db)
}

func (r *sqliteVaultReader) ListRecent(ctx context.Context, limit int) ([]store.VaultDocument, error) {
	return db.ListVaultDocumentsByMTime(ctx, r.db, limit)
}

func (r *sqliteVaultReader) ListTags(ctx context.Context, filter store.TagFilter) ([]store.VaultTag, error) {
	return db.ListVaultTags(ctx, r.db, filter)
}

func (r *sqliteVaultReader) FindDocumentsByTag(ctx context.Context, tag string) ([]store.VaultDocument, error) {
	return db.FindVaultDocumentsByTag(ctx, r.db, tag)
}

func (r *postgresVaultReader) Close() error { return r.store.Close() }

func (r *postgresVaultReader) ListDocuments(ctx context.Context) ([]store.VaultDocument, error) {
	return r.store.ListVaultDocuments(ctx, r.repo)
}

func (r *postgresVaultReader) FindByPath(ctx context.Context, query string) ([]store.VaultDocument, error) {
	return r.store.FindVaultDocumentsByPath(ctx, r.repo, query)
}

func (r *postgresVaultReader) ReadContent(ctx context.Context, docID string) (string, error) {
	return r.store.ReadVaultDocumentContent(ctx, r.repo, docID)
}

func (r *postgresVaultReader) ListDeadends(ctx context.Context) ([]store.VaultDocument, error) {
	return r.store.ListVaultDeadends(ctx, r.repo)
}

func (r *postgresVaultReader) ListRecent(ctx context.Context, limit int) ([]store.VaultDocument, error) {
	return r.store.ListVaultDocumentsByMTime(ctx, r.repo, limit)
}

func (r *postgresVaultReader) ListTags(ctx context.Context, filter store.TagFilter) ([]store.VaultTag, error) {
	return r.store.ListVaultTags(ctx, r.repo, filter)
}

func (r *postgresVaultReader) FindDocumentsByTag(ctx context.Context, tag string) ([]store.VaultDocument, error) {
	return r.store.FindVaultDocumentsByTag(ctx, r.repo, tag)
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
