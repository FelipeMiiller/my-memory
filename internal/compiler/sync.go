package compiler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
	"github.com/FelipeMiiller/my-memory/internal/watcher"
)

// SyncEngine gerencia a sincronização cirúrgica de notas com a camada de persistência
type SyncEngine struct {
	SQLiteDB *sql.DB
	PGStore  *store.PostgresStore
	Emb      *embedder.OllamaClient
	TQ       *turboquant.Quantizer
	RepoSlug string
}

// NewSyncEngine cria uma nova instância de SyncEngine
func NewSyncEngine(database *sql.DB, pgStore *store.PostgresStore, emb *embedder.OllamaClient, tq *turboquant.Quantizer, repoSlug string) *SyncEngine {
	return &SyncEngine{
		SQLiteDB: database,
		PGStore:  pgStore,
		Emb:      emb,
		TQ:       tq,
		RepoSlug: repoSlug,
	}
}

// SyncFile executa a indexação cirúrgica síncrona de um arquivo no storage ativo
func (s *SyncEngine) SyncFile(ctx context.Context, absPath string) (*watcher.IndexResult, error) {
	if s == nil {
		return &watcher.IndexResult{Action: "skipped", DocID: absPath}, nil
	}

	if s.PGStore != nil {
		repo := s.RepoSlug
		if repo == "" {
			repo = "default"
		}
		res, err := watcher.IndexSingleFilePostgres(ctx, s.PGStore, s.Emb, repo, absPath, true)
		if err != nil {
			return nil, fmt.Errorf("falha ao sincronizar cirurgicamente no PostgreSQL: %w", err)
		}
		return res, nil
	}

	if s.SQLiteDB != nil {
		res, err := watcher.IndexSingleFileSQLite(ctx, s.SQLiteDB, s.Emb, s.TQ, absPath, true)
		if err != nil {
			return nil, fmt.Errorf("falha ao sincronizar cirurgicamente no SQLite: %w", err)
		}
		return res, nil
	}

	return &watcher.IndexResult{Action: "skipped", DocID: absPath}, nil
}

// WriteAndSyncNote grava a nota e dispara a sincronização cirúrgica imediata
func WriteAndSyncNote(
	ctx context.Context,
	sync *SyncEngine,
	vaultRoot, requestedPath, title, content string,
	tags, aliases []string,
	noteType string,
	relations []parser.EdgeConnection,
	overwrite bool,
) (*NoteResult, *watcher.IndexResult, error) {
	noteRes, err := WriteAtomicNote(vaultRoot, requestedPath, title, content, tags, aliases, noteType, relations, overwrite)
	if err != nil {
		return nil, nil, err
	}

	indexRes, err := sync.SyncFile(ctx, noteRes.AbsPath)
	if err != nil {
		return noteRes, nil, fmt.Errorf("nota gravada em '%s', mas falhou na sincronização: %w", noteRes.Path, err)
	}

	return noteRes, indexRes, nil
}

// AppendAndSyncSection anexa conteúdo sob uma seção e sincroniza cirurgicamente
func AppendAndSyncSection(
	ctx context.Context,
	sync *SyncEngine,
	vaultRoot, requestedPath, heading, content string,
	createIfMissing bool,
) (*NoteResult, *watcher.IndexResult, error) {
	noteRes, err := AppendSection(vaultRoot, requestedPath, heading, content, createIfMissing)
	if err != nil {
		return nil, nil, err
	}

	indexRes, err := sync.SyncFile(ctx, noteRes.AbsPath)
	if err != nil {
		return noteRes, nil, fmt.Errorf("seção anexada em '%s', mas falhou na sincronização: %w", noteRes.Path, err)
	}

	return noteRes, indexRes, nil
}

// CompileAndSyncTopicNote sintetiza resultados de busca, grava a nota e sincroniza com o banco
func CompileAndSyncTopicNote(
	ctx context.Context,
	sync *SyncEngine,
	vaultRoot, requestedPath, title, topic string,
	sources []CompiledSource,
	tags []string,
	relations []parser.EdgeConnection,
	overwrite bool,
) (*NoteResult, *watcher.IndexResult, error) {
	noteRes, err := CompileTopicNote(vaultRoot, requestedPath, title, topic, sources, tags, relations, overwrite)
	if err != nil {
		return nil, nil, err
	}

	indexRes, err := sync.SyncFile(ctx, noteRes.AbsPath)
	if err != nil {
		return noteRes, nil, fmt.Errorf("nota compilada gravada em '%s', mas falhou na sincronização: %w", noteRes.Path, err)
	}

	return noteRes, indexRes, nil
}
