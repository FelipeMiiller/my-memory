package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/FelipeMiiller/my-memory/internal/turboquant"
	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance,omitempty"`
	Score      float64  `json:"score,omitempty"`
	Sources    []string `json:"sources,omitempty"`
	Neighbors  []string `json:"neighbors,omitempty"` // Conexões descobertas no grafo
}

// SearchKNN busca os pedaços mais próximos usando sqlite-vec nativo
func SearchKNN(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	vecBlob, err := sqlite_vec.SerializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar vetor de busca: %w", err)
	}

	query := `
	SELECT c.id, c.document_id, c.content, v.distance
	FROM chunks_vec v
	JOIN chunks c ON c.id = v.chunk_id
	WHERE v.embedding MATCH ? AND k = ?
	ORDER BY v.distance
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

		neighbors, err := GetNodeNeighbors(ctx, db, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

// SearchTurboQuant busca usando a projeção ortogonal e os vetores 4-bit comprimidos
func SearchTurboQuant(ctx context.Context, db *sql.DB, q *turboquant.Quantizer, queryVec []float32, limit int) ([]SearchResult, error) {
	rotatedQuery := q.RotateQuery(queryVec)

	rows, err := db.QueryContext(ctx, `
		SELECT tq.chunk_id, c.document_id, c.content, tq.scale, tq.data
		FROM chunks_turboquant tq
		JOIN chunks c ON c.id = tq.chunk_id
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar chunks turboquant: %w", err)
	}
	defer rows.Close()

	type scoredItem struct {
		chunkID    string
		documentID string
		content    string
		score      float32
	}

	var items []scoredItem
	for rows.Next() {
		var chunkID, docID, content string
		var scale float32
		var data []byte

		if err := rows.Scan(&chunkID, &docID, &content, &scale, &data); err != nil {
			continue
		}

		cv := &turboquant.CompressedVector{
			Dim:   len(queryVec),
			Scale: scale,
			Data:  data,
		}

		dot := q.DotProduct(rotatedQuery, cv)
		items = append(items, scoredItem{
			chunkID:    chunkID,
			documentID: docID,
			content:    content,
			score:      dot,
		})
	}

	// Ordena por maior similaridade (produto escalar)
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	if len(items) > limit {
		items = items[:limit]
	}

	var results []SearchResult
	for _, it := range items {
		// Distância invertida para visualização consistente (1 - cos_sim normalizado)
		dist := float64(1.0 - it.score)
		neighbors, _ := GetNodeNeighbors(ctx, db, it.documentID, 1)

		results = append(results, SearchResult{
			ChunkID:    it.chunkID,
			DocumentID: it.documentID,
			Content:    it.content,
			Distance:   dist,
			Neighbors:  neighbors,
		})
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
