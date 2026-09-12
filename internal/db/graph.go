package db

import (
	"context"
	"database/sql"
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance"`
	Neighbors  []string `json:"neighbors"` // Conexões descobertas no grafo
}

// SearchKNN busca os pedaços mais próximos usando sqlite-vec
func SearchKNN(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	vecBlob, err := sqlite_vec.SerializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar vetor de busca: %w", err)
	}

	query := `
	SELECT c.id, c.document_id, c.content, v.distance
	FROM chunks_vec v
	JOIN chunks c ON c.id = v.chunk_id
	WHERE v.embedding MATCH ?
	ORDER BY v.distance
	LIMIT ?
	`

	rows, err := db.QueryContext(ctx, query, vecBlob, limit)
	if err != nil {
		return nil, fmt.Errorf("erro na busca vetorial: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Distance); err != nil {
			return nil, err
		}

		// Expande 1 salto de vizinhos no grafo
		neighbors, err := GetNodeNeighbors(ctx, db, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

// GetNodeNeighbors realiza travessia de grafo em SQL usando Recursive CTE
func GetNodeNeighbors(ctx context.Context, db *sql.DB, nodeID string, maxDepth int) ([]string, error) {
	cteQuery := `
	WITH RECURSIVE traversal AS (
		SELECT target_id, 1 AS depth
		FROM graph_edges
		WHERE source_id = ?
		
		UNION
		
		SELECT e.target_id, t.depth + 1
		FROM graph_edges e
		JOIN traversal t ON e.source_id = t.target_id
		WHERE t.depth < ?
	)
	SELECT DISTINCT target_id FROM traversal;
	`

	rows, err := db.QueryContext(ctx, cteQuery, nodeID, maxDepth)
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
