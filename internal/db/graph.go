package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

type SearchResult struct {
	ChunkID    string   `json:"chunk_id"`
	DocumentID string   `json:"document_id"`
	Content    string   `json:"content"`
	Distance   float64  `json:"distance,omitempty"`
	Score      float64  `json:"score,omitempty"`
	Sources    []string `json:"sources,omitempty"`
	Neighbors  []string `json:"neighbors,omitempty"` // Conexões descobertas no grafo
	UpdatedAt  int64    `json:"updated_at,omitempty"`
}

// SearchKNN busca os pedaços mais próximos usando sqlite-vec nativo
func SearchKNN(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	vecBlob, err := serializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar vetor de busca: %w", err)
	}

	query := `
	SELECT c.id, c.document_id, c.content, v.distance, COALESCE(d.updated_at, 0)
	FROM chunks_vec v
	JOIN chunks c ON c.id = v.chunk_id
	LEFT JOIN documents d ON d.id = c.document_id
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
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Distance, &r.UpdatedAt); err != nil {
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
		SELECT tq.chunk_id, c.document_id, c.content, tq.scale, tq.data, COALESCE(d.updated_at, 0)
		FROM chunks_turboquant tq
		JOIN chunks c ON c.id = tq.chunk_id
		LEFT JOIN documents d ON d.id = c.document_id
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
		updatedAt  int64
	}

	var items []scoredItem
	for rows.Next() {
		var chunkID, docID, content string
		var scale float32
		var data []byte
		var updatedAt int64

		if err := rows.Scan(&chunkID, &docID, &content, &scale, &data, &updatedAt); err != nil {
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
			updatedAt:  updatedAt,
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
			UpdatedAt:  it.updatedAt,
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

// DiagnoseHealth audita a integridade topológica do grafo e tabelas relacionais do SQLite
func DiagnoseHealth(ctx context.Context, db *sql.DB) (*store.DoctorReport, error) {
	report := &store.DoctorReport{
		DeadLinks:      []store.DeadLink{},
		OrphanNotes:    []store.OrphanNote{},
		SelfLoops:      []store.SelfLoop{},
		DesyncedChunks: []store.DesyncedChunk{},
	}

	// 1. Contagens gerais
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&report.TotalDocuments)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks`).Scan(&report.TotalChunks)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges`).Scan(&report.TotalEdges)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes`).Scan(&report.TotalNodes)

	// 2. Dead Links (arestas que apontam para notas inexistentes, ignorando tags)
	deadRows, err := db.QueryContext(ctx, `
		SELECT e.source_id, e.target_id, e.relation
		FROM graph_edges e
		WHERE e.target_id NOT LIKE '#%'
		  AND e.target_id NOT IN (SELECT id FROM documents)
		  AND e.target_id NOT IN (SELECT title FROM documents)
		ORDER BY e.source_id, e.target_id
	`)
	if err == nil {
		defer deadRows.Close()
		for deadRows.Next() {
			var dl store.DeadLink
			if err := deadRows.Scan(&dl.SourceID, &dl.TargetID, &dl.Relation); err == nil {
				report.DeadLinks = append(report.DeadLinks, dl)
			}
		}
	}

	// 3. Orphan Notes (documentos sem conexões de entrada ou saída)
	orphanRows, err := db.QueryContext(ctx, `
		SELECT id, title
		FROM documents
		WHERE id NOT IN (SELECT source_id FROM graph_edges)
		  AND id NOT IN (SELECT target_id FROM graph_edges)
		  AND title NOT IN (SELECT target_id FROM graph_edges)
		ORDER BY title
	`)
	if err == nil {
		defer orphanRows.Close()
		for orphanRows.Next() {
			var on store.OrphanNote
			if err := orphanRows.Scan(&on.ID, &on.Title); err == nil {
				report.OrphanNotes = append(report.OrphanNotes, on)
			}
		}
	}

	// 4. Self-Loops (arestas onde source_id = target_id)
	loopRows, err := db.QueryContext(ctx, `
		SELECT source_id, relation
		FROM graph_edges
		WHERE source_id = target_id
		ORDER BY source_id
	`)
	if err == nil {
		defer loopRows.Close()
		for loopRows.Next() {
			var sl store.SelfLoop
			if err := loopRows.Scan(&sl.NodeID, &sl.Relation); err == nil {
				report.SelfLoops = append(report.SelfLoops, sl)
			}
		}
	}

	// 5. Chunks dessincronizados (chunks sem correspondência em chunks_turboquant)
	desyncRows, err := db.QueryContext(ctx, `
		SELECT c.id, c.document_id
		FROM chunks c
		WHERE c.id NOT IN (SELECT chunk_id FROM chunks_turboquant)
		ORDER BY c.id
	`)
	if err == nil {
		defer desyncRows.Close()
		for desyncRows.Next() {
			var dc store.DesyncedChunk
			if err := desyncRows.Scan(&dc.ChunkID, &dc.DocumentID); err == nil {
				dc.Issue = "Missing TurboQuant 4-bit compression"
				report.DesyncedChunks = append(report.DesyncedChunks, dc)
			}
		}
	}

	// 6. Cálculo de Health Score (0 a 100)
	score := 100
	if report.TotalDocuments > 0 {
		// Penalidade por Dead Links: até 40 pontos (5 pts cada)
		deadPenalty := len(report.DeadLinks) * 5
		if deadPenalty > 40 {
			deadPenalty = 40
		}
		score -= deadPenalty

		// Penalidade por Orphan Notes: até 30 pontos proporcional
		orphanRatio := float64(len(report.OrphanNotes)) / float64(report.TotalDocuments)
		orphanPenalty := int(orphanRatio * 40.0)
		if orphanPenalty > 30 {
			orphanPenalty = 30
		}
		score -= orphanPenalty

		// Penalidade por Self-Loops: até 15 pontos (5 pts cada)
		loopPenalty := len(report.SelfLoops) * 5
		if loopPenalty > 15 {
			loopPenalty = 15
		}
		score -= loopPenalty

		// Penalidade por Chunks dessincronizados: até 15 pontos (2 pts cada)
		desyncPenalty := len(report.DesyncedChunks) * 2
		if desyncPenalty > 15 {
			desyncPenalty = 15
		}
		score -= desyncPenalty

		if score < 0 {
			score = 0
		}
	}
	report.HealthScore = score

	return report, nil
}

// FixHealthIssues repara anomalias conhecidas do grafo (remove self-loops e links mortos)
func FixHealthIssues(ctx context.Context, db *sql.DB) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Remove self-loops
	resLoops, err := tx.ExecContext(ctx, `DELETE FROM graph_edges WHERE source_id = target_id`)
	if err != nil {
		return 0, err
	}
	loopsFixed, _ := resLoops.RowsAffected()

	// 2. Remove dead links
	resDead, err := tx.ExecContext(ctx, `
		DELETE FROM graph_edges
		WHERE target_id NOT LIKE '#%'
		  AND target_id NOT IN (SELECT id FROM documents)
		  AND target_id NOT IN (SELECT title FROM documents)
	`)
	if err != nil {
		return 0, err
	}
	deadFixed, _ := resDead.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(loopsFixed + deadFixed), nil
}
