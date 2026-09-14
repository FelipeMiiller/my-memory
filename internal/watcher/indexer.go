package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/parser"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// IndexResult reporta o resultado da indexação ou purga cirúrgica de uma nota
type IndexResult struct {
	Action      string        // "indexed", "cached", "purged", "skipped"
	DocID       string        // Identificador do documento (caminho relativo ou absoluto)
	Title       string        // Título extraído do arquivo
	EdgesCount  int           // Quantidade de arestas de grafo extraídas
	ChunksCount int           // Quantidade de chunks gerados
	Duration    time.Duration // Tempo de execução da operação
}

// IndexSingleFileSQLite reindexa cirurgicamente uma única nota no SQLite
func IndexSingleFileSQLite(
	ctx context.Context,
	database *sql.DB,
	emb *embedder.OllamaClient,
	tq *turboquant.Quantizer,
	filePath string,
	force bool,
) (*IndexResult, error) {
	start := time.Now()

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo para indexação cirúrgica: %w", err)
	}

	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	currentHash := store.CalculateContentHash(contentBytes)

	// Verificação de cache incremental via SHA-256
	if !force {
		storedHash, err := db.GetDocumentHash(ctx, database, docID)
		if err == nil && storedHash != "" && storedHash == currentHash {
			return &IndexResult{
				Action:   "cached",
				DocID:    docID,
				Title:    title,
				Duration: time.Since(start),
			}, nil
		}
	}

	content := string(contentBytes)
	fm, body := parser.ExtractFrontmatter(content)
	category := "resource"
	if fm != nil && fm.Category != "" {
		category = fm.Category
	}
	if category != "resource" && category != "memory" && category != "skill" {
		category = "resource"
	}
	var abstract string
	if fm != nil && fm.Summary != "" {
		abstract = fm.Summary
	} else if fm != nil && fm.Abstract != "" {
		abstract = fm.Abstract
	} else {
		abstract = parser.ExtractMicroAbstract(body, 160)
	}

	// 1. Limpa dados anteriores do documento para reindexação limpa
	_ = db.DeleteDocumentData(ctx, database, docID)

	// 2. Salva documento com content_hash, abstract e category
	if err := db.InsertDocumentWithMeta(ctx, database, docID, filePath, title, time.Now().Unix(), currentHash, abstract, category); err != nil {
		return nil, fmt.Errorf("erro ao inserir documento SQLite: %w", err)
	}

	// 3. Extrai e salva conexões do grafo
	connections := parser.ExtractConnections(content)
	for _, edge := range connections.Edges {
		_ = db.InsertEdgeWithProps(ctx, database, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
	}

	// 4. Divide em chunks e indexa no FTS e vetorial (com fallback para FTS se offline)
	chunks := parser.ChunkText(content, 200, 30)
	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s#%d", docID, i)
		var vec []float32
		if emb != nil {
			v, err := emb.GenerateEmbedding(c)
			if err == nil {
				vec = v
			}
		}
		if vec == nil {
			vec = make([]float32, 768)
		}

		_ = db.InsertChunk(ctx, database, chunkID, docID, c, i, vec)

		if tq != nil {
			cv, err := tq.Quantize(vec)
			if err == nil {
				_ = db.InsertTurboQuantChunk(ctx, database, chunkID, cv.Scale, cv.Data)
			}
		}
	}

	return &IndexResult{
		Action:      "indexed",
		DocID:       docID,
		Title:       title,
		EdgesCount:  len(connections.Edges),
		ChunksCount: len(chunks),
		Duration:    time.Since(start),
	}, nil
}

// PurgeSingleFileSQLite remove completamente uma nota, seus chunks e arestas do SQLite
func PurgeSingleFileSQLite(ctx context.Context, database *sql.DB, filePath string) (*IndexResult, error) {
	start := time.Now()
	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	if err := db.DeleteDocumentComplete(ctx, database, docID); err != nil {
		return nil, fmt.Errorf("erro ao purgar documento SQLite: %w", err)
	}

	return &IndexResult{
		Action:   "purged",
		DocID:    docID,
		Title:    title,
		Duration: time.Since(start),
	}, nil
}

// IndexSingleFilePostgres reindexa cirurgicamente uma única nota no PostgreSQL com pgvector
func IndexSingleFilePostgres(
	ctx context.Context,
	s *store.PostgresStore,
	emb *embedder.OllamaClient,
	targetRepo string,
	filePath string,
	force bool,
) (*IndexResult, error) {
	start := time.Now()

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo para indexação cirúrgica: %w", err)
	}

	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	currentHash := store.CalculateContentHash(contentBytes)

	if !force {
		storedHash, err := s.GetDocumentHash(ctx, targetRepo, docID)
		if err == nil && storedHash != "" && storedHash == currentHash {
			return &IndexResult{
				Action:   "cached",
				DocID:    docID,
				Title:    title,
				Duration: time.Since(start),
			}, nil
		}
	}

	content := string(contentBytes)
	fm, body := parser.ExtractFrontmatter(content)
	category := "resource"
	if fm != nil && fm.Category != "" {
		category = fm.Category
	}
	if category != "resource" && category != "memory" && category != "skill" {
		category = "resource"
	}
	var abstract string
	if fm != nil && fm.Summary != "" {
		abstract = fm.Summary
	} else if fm != nil && fm.Abstract != "" {
		abstract = fm.Abstract
	} else {
		abstract = parser.ExtractMicroAbstract(body, 160)
	}

	_ = s.DeleteDocumentData(ctx, targetRepo, docID)

	if err := s.InsertDocumentWithMeta(ctx, targetRepo, docID, filePath, title, time.Now().Unix(), currentHash, abstract, category); err != nil {
		return nil, fmt.Errorf("erro ao inserir documento Postgres: %w", err)
	}
	connections := parser.ExtractConnections(content)
	for _, edge := range connections.Edges {
		_ = s.InsertEdgeWithProps(ctx, targetRepo, docID, edge.Target, edge.Relation, edge.EpistemicStatus, edge.Weight)
	}

	chunks := parser.ChunkText(content, 200, 30)
	for i, c := range chunks {
		chunkID := fmt.Sprintf("%s#%d", docID, i)
		var vec []float32
		if emb != nil {
			v, err := emb.GenerateEmbedding(c)
			if err == nil {
				vec = v
			}
		}
		if vec == nil {
			vec = make([]float32, 768)
		}

		_ = s.InsertChunk(ctx, targetRepo, chunkID, docID, c, i, vec)
	}

	return &IndexResult{
		Action:      "indexed",
		DocID:       docID,
		Title:       title,
		EdgesCount:  len(connections.Edges),
		ChunksCount: len(chunks),
		Duration:    time.Since(start),
	}, nil
}

// PurgeSingleFilePostgres remove chunks e arestas de uma nota no PostgreSQL
func PurgeSingleFilePostgres(
	ctx context.Context,
	s *store.PostgresStore,
	targetRepo string,
	filePath string,
) (*IndexResult, error) {
	start := time.Now()
	docID := filePath
	title := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	_ = s.DeleteDocumentData(ctx, targetRepo, docID)

	return &IndexResult{
		Action:   "purged",
		DocID:    docID,
		Title:    title,
		Duration: time.Since(start),
	}, nil
}
