package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

// PostgresStore implementa a interface Store utilizando PostgreSQL e extensão pgvector
type PostgresStore struct {
	db *sql.DB
}

const PostgresSchema = `
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS documents (
    id TEXT NOT NULL,
    repository TEXT NOT NULL,
    path TEXT NOT NULL,
    title TEXT,
    updated_at BIGINT NOT NULL,
    PRIMARY KEY (repository, id)
);

CREATE TABLE IF NOT EXISTS chunks (
    id TEXT NOT NULL,
    repository TEXT NOT NULL,
    document_id TEXT NOT NULL,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(768),
    PRIMARY KEY (repository, id),
    FOREIGN KEY (repository, document_id) REFERENCES documents(repository, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS graph_nodes (
    id TEXT NOT NULL,
    repository TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    PRIMARY KEY (repository, id)
);

CREATE TABLE IF NOT EXISTS graph_edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    repository TEXT NOT NULL,
    relation TEXT NOT NULL,
    PRIMARY KEY (repository, source_id, target_id, relation),
    FOREIGN KEY (repository, source_id) REFERENCES graph_nodes(repository, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_target ON graph_edges(repository, target_id);
CREATE INDEX IF NOT EXISTS idx_chunks_fts ON chunks USING gin(to_tsvector('simple', content));
`

var _ Store = (*PostgresStore)(nil)

// NewPostgresStore conecta e inicializa o schema do PostgreSQL
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexao postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar no postgres: %w", err)
	}

	if _, err := db.Exec(PostgresSchema); err != nil {
		return nil, fmt.Errorf("erro ao aplicar schema postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// FormatVector converte slice de float32 no formato textual aceito pelo pgvector "[x,y,z]"
func FormatVector(vec []float32) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	sb.WriteString("]")
	return sb.String()
}

func (s *PostgresStore) InsertDocument(ctx context.Context, repo, id, path, title string, updatedAt int64) error {
	if repo == "" {
		repo = "default"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO documents (id, repository, path, title, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (repository, id) DO UPDATE SET
			title = EXCLUDED.title,
			updated_at = EXCLUDED.updated_at
	`, id, repo, path, title, updatedAt)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name)
		VALUES ($1, $2, 'note', $3)
		ON CONFLICT (repository, id) DO UPDATE SET
			name = EXCLUDED.name
	`, id, repo, title)
	return err
}

func (s *PostgresStore) InsertChunk(ctx context.Context, repo, chunkID, docID, content string, index int, vec []float32) error {
	if repo == "" {
		repo = "default"
	}

	vecStr := FormatVector(vec)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chunks (id, repository, document_id, chunk_index, content, embedding)
		VALUES ($1, $2, $3, $4, $5, $6::vector)
		ON CONFLICT (repository, id) DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding
	`, chunkID, repo, docID, index, content, vecStr)
	return err
}

func (s *PostgresStore) InsertEdge(ctx context.Context, repo, sourceID, targetID, relation string) error {
	if repo == "" {
		repo = "default"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (repository, source_id, target_id, relation) DO NOTHING
	`, sourceID, targetID, repo, relation)
	return err
}

func (s *PostgresStore) SearchKNN(ctx context.Context, repo string, queryVec []float32, limit int) ([]SearchResult, error) {
	vecStr := FormatVector(queryVec)

	query := `
		SELECT c.id, c.document_id, c.repository, c.content, (c.embedding <=> $1::vector) AS distance
		FROM chunks c
		WHERE ($2 = '' OR c.repository = $2)
		ORDER BY c.embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, vecStr, repo, limit)
	if err != nil {
		return nil, fmt.Errorf("erro na busca vetorial postgres: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Repository, &r.Content, &r.Distance); err != nil {
			return nil, err
		}

		neighbors, err := s.GetNodeNeighbors(ctx, r.Repository, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

func (s *PostgresStore) GetNodeNeighbors(ctx context.Context, repo string, nodeID string, maxDepth int) ([]string, error) {
	cteQuery := `
		WITH RECURSIVE traversal AS (
			SELECT target_id, 1 AS depth
			FROM graph_edges
			WHERE source_id = $1 AND ($2 = '' OR repository = $2)
			
			UNION
			
			SELECT e.target_id, t.depth + 1
			FROM graph_edges e
			JOIN traversal t ON e.source_id = t.target_id
			WHERE t.depth < $3 AND ($2 = '' OR e.repository = $2)
		)
		SELECT DISTINCT target_id FROM traversal;
	`

	rows, err := s.db.QueryContext(ctx, cteQuery, nodeID, repo, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var neighbors []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err == nil {
			neighbors = append(neighbors, target)
		}
	}
	return neighbors, nil
}

func (s *PostgresStore) SearchFTS(ctx context.Context, repo string, query string, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	q := `
		SELECT c.id, c.document_id, c.repository, c.content,
		       ts_rank(to_tsvector('simple', c.content), plainto_tsquery('simple', $1)) AS rank
		FROM chunks c
		WHERE ($2 = '' OR c.repository = $2)
		  AND to_tsvector('simple', c.content) @@ plainto_tsquery('simple', $1)
		ORDER BY rank DESC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, q, query, repo, limit)
	if err != nil {
		return nil, fmt.Errorf("erro na busca textual postgres: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var rank float64
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Repository, &r.Content, &rank); err != nil {
			return nil, err
		}
		r.Distance = rank

		neighbors, err := s.GetNodeNeighbors(ctx, r.Repository, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

func (s *PostgresStore) SearchHybridRRF(ctx context.Context, repo string, query string, queryVec []float32, limit int, k int) ([]SearchResult, error) {
	candidateLimit := limit * 2
	if candidateLimit < 10 {
		candidateLimit = 10
	}

	// 1. Busca textual via FTS
	var ftsResults []SearchResult
	if strings.TrimSpace(query) != "" {
		ftsResults, _ = s.SearchFTS(ctx, repo, query, candidateLimit)
	}

	// 2. Busca vetorial via pgvector
	var vecResults []SearchResult
	if len(queryVec) > 0 {
		vecResults, _ = s.SearchKNN(ctx, repo, queryVec, candidateLimit)
	}

	// 3. Expansão de vizinhos estruturais no grafo a partir das sementes mais relevantes
	seedDocs := make(map[string]bool)
	for _, r := range ftsResults {
		if len(seedDocs) >= 3 {
			break
		}
		seedDocs[r.DocumentID] = true
	}
	for _, r := range vecResults {
		if len(seedDocs) >= 6 {
			break
		}
		seedDocs[r.DocumentID] = true
	}

	var graphResults []SearchResult
	seenChunks := make(map[string]bool)
	for seed := range seedDocs {
		neighbors, err := s.GetNodeNeighbors(ctx, repo, seed, 1)
		if err != nil {
			continue
		}

		for _, n := range neighbors {
			rows, err := s.db.QueryContext(ctx, `
				SELECT id, document_id, repository, content
				FROM chunks
				WHERE document_id = $1 AND ($2 = '' OR repository = $2)
				ORDER BY chunk_index
				LIMIT 2
			`, n, repo)
			if err != nil {
				continue
			}

			for rows.Next() {
				var gr SearchResult
				if err := rows.Scan(&gr.ChunkID, &gr.DocumentID, &gr.Repository, &gr.Content); err == nil {
					if !seenChunks[gr.ChunkID] {
						seenChunks[gr.ChunkID] = true
						graphResults = append(graphResults, gr)
					}
				}
			}
			rows.Close()
		}
	}

	// 4. Fusão RRF através de FuseSearchResults
	sources := []RankedResultSource{
		{Name: "fts", Results: ftsResults},
		{Name: "vector", Results: vecResults},
		{Name: "graph", Results: graphResults},
	}

	return FuseSearchResults(sources, k, limit), nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
