package store

import (
	"context"
)

// DiagnoseHealth audita a integridade topológica do grafo e tabelas relacionais no PostgreSQL
func (s *PostgresStore) DiagnoseHealth(ctx context.Context, repo string) (*DoctorReport, error) {
	if repo == "" {
		repo = "default"
	}

	report := &DoctorReport{
		DeadLinks:      []DeadLink{},
		OrphanNotes:    []OrphanNote{},
		SelfLoops:      []SelfLoop{},
		DesyncedChunks: []DesyncedChunk{},
	}

	// 1. Contagens gerais
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE repository = $1`, repo).Scan(&report.TotalDocuments)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks WHERE repository = $1`, repo).Scan(&report.TotalChunks)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges WHERE repository = $1`, repo).Scan(&report.TotalEdges)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes WHERE repository = $1`, repo).Scan(&report.TotalNodes)

	// 2. Dead Links (arestas que apontam para notas inexistentes, ignorando tags)
	deadRows, err := s.db.QueryContext(ctx, `
		SELECT e.source_id, e.target_id, e.relation
		FROM graph_edges e
		WHERE e.repository = $1
		  AND e.target_id NOT LIKE '#%'
		  AND e.target_id NOT IN (SELECT id FROM documents WHERE repository = $1)
		  AND e.target_id NOT IN (SELECT title FROM documents WHERE repository = $1)
		ORDER BY e.source_id, e.target_id
	`, repo)
	if err == nil {
		defer deadRows.Close()
		for deadRows.Next() {
			var dl DeadLink
			if err := deadRows.Scan(&dl.SourceID, &dl.TargetID, &dl.Relation); err == nil {
				report.DeadLinks = append(report.DeadLinks, dl)
			}
		}
	}

	// 3. Orphan Notes (documentos sem conexões de entrada ou saída)
	orphanRows, err := s.db.QueryContext(ctx, `
		SELECT id, title
		FROM documents
		WHERE repository = $1
		  AND id NOT IN (SELECT source_id FROM graph_edges WHERE repository = $1)
		  AND id NOT IN (SELECT target_id FROM graph_edges WHERE repository = $1)
		  AND title NOT IN (SELECT target_id FROM graph_edges WHERE repository = $1)
		ORDER BY title
	`, repo)
	if err == nil {
		defer orphanRows.Close()
		for orphanRows.Next() {
			var on OrphanNote
			if err := orphanRows.Scan(&on.ID, &on.Title); err == nil {
				report.OrphanNotes = append(report.OrphanNotes, on)
			}
		}
	}

	// 4. Self-Loops (arestas onde source_id = target_id)
	loopRows, err := s.db.QueryContext(ctx, `
		SELECT source_id, relation
		FROM graph_edges
		WHERE repository = $1 AND source_id = target_id
		ORDER BY source_id
	`, repo)
	if err == nil {
		defer loopRows.Close()
		for loopRows.Next() {
			var sl SelfLoop
			if err := loopRows.Scan(&sl.NodeID, &sl.Relation); err == nil {
				report.SelfLoops = append(report.SelfLoops, sl)
			}
		}
	}

	// 5. Chunks dessincronizados (sem embedding)
	desyncRows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.document_id
		FROM chunks c
		WHERE c.repository = $1 AND c.embedding IS NULL
		ORDER BY c.id
	`, repo)
	if err == nil {
		defer desyncRows.Close()
		for desyncRows.Next() {
			var dc DesyncedChunk
			if err := desyncRows.Scan(&dc.ChunkID, &dc.DocumentID); err == nil {
				dc.Issue = "Missing vector embedding"
				report.DesyncedChunks = append(report.DesyncedChunks, dc)
			}
		}
	}

	// 6. Cálculo de Health Score (0 a 100)
	score := 100
	if report.TotalDocuments > 0 {
		deadPenalty := len(report.DeadLinks) * 5
		if deadPenalty > 40 {
			deadPenalty = 40
		}
		score -= deadPenalty

		orphanRatio := float64(len(report.OrphanNotes)) / float64(report.TotalDocuments)
		orphanPenalty := int(orphanRatio * 40.0)
		if orphanPenalty > 30 {
			orphanPenalty = 30
		}
		score -= orphanPenalty

		loopPenalty := len(report.SelfLoops) * 5
		if loopPenalty > 15 {
			loopPenalty = 15
		}
		score -= loopPenalty

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

// FixHealthIssues repara anomalias conhecidas do grafo no PostgreSQL (remove self-loops e links mortos)
func (s *PostgresStore) FixHealthIssues(ctx context.Context, repo string) (int, error) {
	if repo == "" {
		repo = "default"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	resLoops, err := tx.ExecContext(ctx, `
		DELETE FROM graph_edges WHERE repository = $1 AND source_id = target_id
	`, repo)
	if err != nil {
		return 0, err
	}
	loopsFixed, _ := resLoops.RowsAffected()

	resDead, err := tx.ExecContext(ctx, `
		DELETE FROM graph_edges
		WHERE repository = $1
		  AND target_id NOT LIKE '#%'
		  AND target_id NOT IN (SELECT id FROM documents WHERE repository = $1)
		  AND target_id NOT IN (SELECT title FROM documents WHERE repository = $1)
	`, repo)
	if err != nil {
		return 0, err
	}
	deadFixed, _ := resDead.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(loopsFixed + deadFixed), nil
}
