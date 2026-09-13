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
    content_hash TEXT,
    PRIMARY KEY (repository, id)
);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS content_hash TEXT;

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
    epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED',
    weight REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (repository, source_id, target_id, relation),
    FOREIGN KEY (repository, source_id) REFERENCES graph_nodes(repository, id) ON DELETE CASCADE
);
ALTER TABLE graph_edges ADD COLUMN IF NOT EXISTS epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED';
ALTER TABLE graph_edges ADD COLUMN IF NOT EXISTS weight REAL NOT NULL DEFAULT 1.0;

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

func (s *PostgresStore) InsertDocument(ctx context.Context, repo, id, path, title string, updatedAt int64, contentHash string) error {
	if repo == "" {
		repo = "default"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO documents (id, repository, path, title, updated_at, content_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository, id) DO UPDATE SET
			title = EXCLUDED.title,
			updated_at = EXCLUDED.updated_at,
			content_hash = EXCLUDED.content_hash
	`, id, repo, path, title, updatedAt, contentHash)
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

func (s *PostgresStore) GetDocumentHash(ctx context.Context, repo, id string) (string, error) {
	if repo == "" {
		repo = "default"
	}

	var hash sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT content_hash FROM documents
		WHERE repository = $1 AND id = $2
	`, repo, id).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash.String, nil
}

func (s *PostgresStore) DeleteDocumentData(ctx context.Context, repo, id string) error {
	if repo == "" {
		repo = "default"
	}

	_, err := s.db.ExecContext(ctx, `
		DELETE FROM chunks WHERE repository = $1 AND document_id = $2
	`, repo, id)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM graph_edges WHERE repository = $1 AND source_id = $2
	`, repo, id)
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

func (s *PostgresStore) InsertEdgeWithProps(ctx context.Context, repo, sourceID, targetID, relation, epistemicStatus string, weight float64) error {
	if repo == "" {
		repo = "default"
	}
	if epistemicStatus == "" {
		epistemicStatus = "EXTRACTED"
	}
	if weight <= 0 {
		weight = 1.0
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO graph_edges (source_id, target_id, repository, relation, epistemic_status, weight)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository, source_id, target_id, relation) DO UPDATE SET
			epistemic_status = EXCLUDED.epistemic_status,
			weight = EXCLUDED.weight
	`, sourceID, targetID, repo, relation, epistemicStatus, weight)
	return err
}

func (s *PostgresStore) InsertEdge(ctx context.Context, repo, sourceID, targetID, relation string) error {
	return s.InsertEdgeWithProps(ctx, repo, sourceID, targetID, relation, "EXTRACTED", 1.0)
}

func (s *PostgresStore) GetGodNodes(ctx context.Context, repo string, limit int) ([]GodNode, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		WITH degrees AS (
			SELECT source_id AS node_id, 0 AS in_cnt, 1 AS out_cnt FROM graph_edges WHERE ($1 = '' OR repository = $1)
			UNION ALL
			SELECT target_id AS node_id, 1 AS in_cnt, 0 AS out_cnt FROM graph_edges WHERE ($1 = '' OR repository = $1)
		)
		SELECT d.node_id, COALESCE(n.name, d.node_id) AS name,
		       SUM(d.in_cnt) AS in_degree,
		       SUM(d.out_cnt) AS out_degree,
		       COUNT(*) AS total_degree
		FROM degrees d
		LEFT JOIN graph_nodes n ON n.id = d.node_id AND ($1 = '' OR n.repository = $1)
		GROUP BY d.node_id, n.name
		ORDER BY total_degree DESC, in_degree DESC
		LIMIT $2;
	`

	rows, err := s.db.QueryContext(ctx, query, repo, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar god nodes postgres: %w", err)
	}
	defer rows.Close()

	var hubs []GodNode
	for rows.Next() {
		var h GodNode
		if err := rows.Scan(&h.ID, &h.Name, &h.InDegree, &h.OutDegree, &h.TotalDegree); err != nil {
			return nil, err
		}
		hubs = append(hubs, h)
	}
	return hubs, nil
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
