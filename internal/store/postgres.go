package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
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

func (s *PostgresStore) PruneDeletedDocuments(ctx context.Context, repo, rootDir string, activeDocIDs []string) ([]string, error) {
	if repo == "" {
		repo = "default"
	}

	activeMap := make(map[string]bool, len(activeDocIDs))
	for _, id := range activeDocIDs {
		activeMap[id] = true
		activeMap[filepath.Clean(id)] = true
		if abs, err := filepath.Abs(id); err == nil {
			activeMap[abs] = true
			activeMap[filepath.Clean(abs)] = true
		}
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path FROM documents WHERE repository = $1
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar documentos para pruning: %w", err)
	}
	defer rows.Close()

	var toPrune []string
	cleanRoot := ""
	if rootDir != "" {
		if abs, err := filepath.Abs(rootDir); err == nil {
			cleanRoot = strings.ToLower(filepath.Clean(abs))
		} else {
			cleanRoot = strings.ToLower(filepath.Clean(rootDir))
		}
	}

	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}

		cleanPath := path
		if abs, err := filepath.Abs(path); err == nil {
			cleanPath = abs
		}
		cleanPathLower := strings.ToLower(filepath.Clean(cleanPath))

		if cleanRoot != "" {
			if !strings.HasPrefix(cleanPathLower, cleanRoot) {
				continue
			}
		}

		cleanID := id
		if abs, err := filepath.Abs(id); err == nil {
			cleanID = abs
		}

		if !activeMap[id] && !activeMap[path] && !activeMap[cleanPath] && !activeMap[cleanID] {
			toPrune = append(toPrune, id)
		}
	}

	var pruned []string
	for _, id := range toPrune {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			continue
		}

		_, _ = tx.ExecContext(ctx, `DELETE FROM chunks WHERE repository = $1 AND document_id = $2`, repo, id)
		_, _ = tx.ExecContext(ctx, `DELETE FROM graph_edges WHERE repository = $1 AND (source_id = $2 OR target_id = $2)`, repo, id)
		_, _ = tx.ExecContext(ctx, `DELETE FROM documents WHERE repository = $1 AND id = $2`, repo, id)
		_, _ = tx.ExecContext(ctx, `DELETE FROM graph_nodes WHERE repository = $1 AND id = $2`, repo, id)

		if err := tx.Commit(); err == nil {
			pruned = append(pruned, id)
		}
	}

	return pruned, nil
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

	// Garante que nós de origem e destino existam para satisfazer integridade referencial
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name)
		VALUES ($1, $2, 'note', $1)
		ON CONFLICT (repository, id) DO NOTHING
	`, sourceID, repo)
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, repository, type, name)
		VALUES ($1, $2, 'note', $1)
		ON CONFLICT (repository, id) DO NOTHING
	`, targetID, repo)

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

// FindSurprisingConnections descobre conexões latentes entre documentos conceitualmente similares sem arestas no grafo
func (s *PostgresStore) FindSurprisingConnections(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error) {
	if limit <= 0 {
		limit = 10
	}
	if minSimilarity <= 0 {
		minSimilarity = 0.70
	}

	// 1. Verifica se existem embeddings calculados no repositório
	var vectorCount int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM chunks
		WHERE ($1 = '' OR repository = $1) AND embedding IS NOT NULL
	`, repo).Scan(&vectorCount)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar vetores postgres: %w", err)
	}

	// Se houver vetores, executa busca via pgvector cosine distance (<=>)
	if vectorCount > 0 {
		query := `
			SELECT 
				c1.document_id AS source_id,
				COALESCE(d1.title, c1.document_id) AS source_name,
				c2.document_id AS target_id,
				COALESCE(d2.title, c2.document_id) AS target_name,
				MAX(1.0 - (c1.embedding <=> c2.embedding)) AS similarity
			FROM chunks c1
			JOIN chunks c2 ON c1.document_id < c2.document_id AND c1.repository = c2.repository
			JOIN documents d1 ON d1.repository = c1.repository AND d1.id = c1.document_id
			JOIN documents d2 ON d2.repository = c2.repository AND d2.id = c2.document_id
			WHERE ($1 = '' OR c1.repository = $1)
			  AND c1.embedding IS NOT NULL
			  AND c2.embedding IS NOT NULL
			  AND NOT EXISTS (
				  SELECT 1 FROM graph_edges e
				  WHERE e.repository = c1.repository
					AND ((e.source_id = c1.document_id AND e.target_id = c2.document_id)
					  OR (e.source_id = c2.document_id AND e.target_id = c1.document_id))
			  )
			GROUP BY c1.document_id, d1.title, c2.document_id, d2.title
			HAVING MAX(1.0 - (c1.embedding <=> c2.embedding)) >= $2
			ORDER BY similarity DESC
			LIMIT $3;
		`
		rows, err := s.db.QueryContext(ctx, query, repo, minSimilarity, limit)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar conexoes inesperadas vetoriais postgres: %w", err)
		}
		defer rows.Close()

		var results []SurprisingConnection
		for rows.Next() {
			var conn SurprisingConnection
			if err := rows.Scan(&conn.SourceID, &conn.SourceName, &conn.TargetID, &conn.TargetName, &conn.Similarity); err != nil {
				return nil, err
			}
			conn.Reason = fmt.Sprintf("Alta proximidade semântica (%.0f%%) sem conexão direta no grafo", conn.Similarity*100)
			results = append(results, conn)
		}
		if results == nil {
			results = []SurprisingConnection{}
		}
		return results, nil
	}

	// 2. Fallback Léxico (Jaccard) caso embeddings não estejam disponíveis
	return s.findSurprisingConnectionsLexical(ctx, repo, limit, minSimilarity)
}

func (s *PostgresStore) findSurprisingConnectionsLexical(ctx context.Context, repo string, limit int, minSimilarity float64) ([]SurprisingConnection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, COALESCE(d.title, d.id), COALESCE(string_agg(c.content, ' '), '')
		FROM documents d
		LEFT JOIN chunks c ON c.repository = d.repository AND c.document_id = d.id
		WHERE ($1 = '' OR d.repository = $1)
		GROUP BY d.id, d.title
		ORDER BY d.id
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar documentos para analise lexica: %w", err)
	}
	defer rows.Close()

	type docEntry struct {
		id      string
		title   string
		content string
	}
	var docs []docEntry
	for rows.Next() {
		var de docEntry
		if err := rows.Scan(&de.id, &de.title, &de.content); err == nil {
			docs = append(docs, de)
		}
	}

	edgeRows, err := s.db.QueryContext(ctx, `
		SELECT source_id, target_id FROM graph_edges
		WHERE ($1 = '' OR repository = $1)
	`, repo)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar arestas: %w", err)
	}
	defer edgeRows.Close()

	linked := make(map[string]bool)
	for edgeRows.Next() {
		var src, tgt string
		if err := edgeRows.Scan(&src, &tgt); err == nil {
			linked[src+"->"+tgt] = true
			linked[tgt+"->"+src] = true
		}
	}

	var results []SurprisingConnection
	for i := 0; i < len(docs); i++ {
		for j := i + 1; j < len(docs); j++ {
			d1 := docs[i]
			d2 := docs[j]

			if linked[d1.id+"->"+d2.id] {
				continue
			}

			sim := CalculateJaccardSimilarity(d1.content, d2.content)
			if sim >= minSimilarity {
				results = append(results, SurprisingConnection{
					SourceID:   d1.id,
					SourceName: d1.title,
					TargetID:   d2.id,
					TargetName: d2.title,
					Similarity: sim,
					Reason:     fmt.Sprintf("Alta sobreposição léxica (%.0f%%) sem conexão direta no grafo", sim*100),
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}
	if results == nil {
		results = []SurprisingConnection{}
	}
	return results, nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}
