package store

import (
	"context"
)

// codeSymbolNodeType é o `type` do graph_nodes que representa um símbolo de
// código (ADR-061, CG-17/CG-18).
//
// O literal é repetido aqui porque `internal/store` não pode importar
// `internal/db` (o `internal/db` importa o `internal/store` — o ciclo
// quebraria o build). O valor é o mesmo de `db.CodeSymbolNodeType`.
const codeSymbolNodeType = "code_symbol"

// HealthScoreApplicable reports whether the report's Health Score can be read
// as a verdict.
//
// The metric is a ratio over the indexed documents: without documents there is
// no denominator, so the score does not apply (ISSUE-019). A vault that was
// never indexed is not a sick vault — there is nothing to penalize. This is the
// single rule of applicability: the calculation and every consumer ask it.
func (r DoctorReport) HealthScoreApplicable() bool {
	return r.TotalDocuments > 0
}

// CalculateHealthScore computes the Health Score (0-100) from the anomalies
// found in the report.
//
// The penalties are the ones established in ADR-013: 5 points per dead link
// (max 40), the orphan ratio times 40 (max 30), 5 points per self-loop
// (max 15) and 2 points per desynchronized chunk (max 15).
//
// A report with no indexed documents returns 0: the score has no denominator,
// so it does not apply, and 0 here is an inert value — never a verdict. The
// caller must consult HealthScoreApplicable before presenting it (ISSUE-019).
func CalculateHealthScore(report DoctorReport) int {
	if !report.HealthScoreApplicable() {
		return 0
	}

	score := 100

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
	return score
}

// DiagnoseHealth audita a integridade topológica do grafo e tabelas relacionais no PostgreSQL
func (s *PostgresStore) DiagnoseHealth(ctx context.Context, repo string) (*DoctorReport, error) {
	if repo == "" {
		repo = "default"
	}

	report := &DoctorReport{
		DeadLinks:         []DeadLink{},
		OrphanNotes:       []OrphanNote{},
		SelfLoops:         []SelfLoop{},
		DesyncedChunks:    []DesyncedChunk{},
		OrthographyIssues: []PortugueseOrthographyIssue{},
	}

	// 1. Contagens gerais
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE repository = $1`, repo).Scan(&report.TotalDocuments)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks WHERE repository = $1`, repo).Scan(&report.TotalChunks)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges WHERE repository = $1`, repo).Scan(&report.TotalEdges)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes WHERE repository = $1`, repo).Scan(&report.TotalNodes)
	// ADR-057: documentos indexados antes de mtime existir ficam com 0. Reportar
	// a contagem torna a lacuna visível em vez de silenciosa.
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents WHERE repository = $1 AND COALESCE(mtime, 0) = 0`, repo).Scan(&report.MissingMTimeCount)

	// 2. Dead Links (arestas que apontam para notas inexistentes, ignorando tags)
	//
	// O alvo de uma aresta `references_code` (ADR-061) é um NÓ de código
	// (`type='code_symbol'`, id = "code:<repository>:<qualified_name>") — ele
	// não é documento, mas também não está quebrado: o nó existe no grafo.
	// Sem esta cláusula, todas as arestas da ponte contavam como link morto
	// (ISSUE-036). O escopo de repositório é o mesmo da aresta: no PostgreSQL
	// a identidade do nó é (repository, id).
	deadRows, err := s.db.QueryContext(ctx, `
		SELECT e.source_id, e.target_id, e.relation
		FROM graph_edges e
		WHERE e.repository = $1
		  AND e.target_id NOT LIKE '#%'
		  AND e.target_id NOT IN (SELECT id FROM documents WHERE repository = $1)
		  AND e.target_id NOT IN (SELECT title FROM documents WHERE repository = $1)
		  AND e.target_id NOT IN (SELECT id FROM graph_nodes WHERE repository = $1 AND type = $2)
		ORDER BY e.source_id, e.target_id
	`, repo, codeSymbolNodeType)
	if err == nil {
		defer deadRows.Close()
		for deadRows.Next() {
			var dl DeadLink
			if err := deadRows.Scan(&dl.SourceID, &dl.TargetID, &dl.Relation); err == nil {
				report.DeadLinks = append(report.DeadLinks, dl)
			}
		}
	}

	// 3. Orphan Notes (documentos sem conexões de entrada ou saída).
	// ISSUE-006 (2026-09-18): docs by-design (ADRs, specs, skills, configs, .github)
	// são excluídos da contagem de órfãos — eles existem para leitura isolada, não
	// como hubs do grafo. Sem isso, o Health Score fica saturado em ~74/100 só
	// porque a maioria dos docs do repo é by-design.
	orphanRows, err := s.db.QueryContext(ctx, `
		SELECT id, title
		FROM documents
		WHERE repository = $1
		  AND id NOT IN (SELECT source_id FROM graph_edges WHERE repository = $1)
		  AND id NOT IN (SELECT target_id FROM graph_edges WHERE repository = $1)
		  AND title NOT IN (SELECT target_id FROM graph_edges WHERE repository = $1)
		  AND (path NOT LIKE '%docs\adr\%' AND path NOT LIKE '%docs/adr/%')
		  AND (path NOT LIKE '%.specs\%'   AND path NOT LIKE '%.specs/%')
		  AND (path NOT LIKE '%.agents\skills\%' AND path NOT LIKE '%.agents/skills/%')
		  AND (path NOT LIKE '%.memory\%'  AND path NOT LIKE '%.memory/%')
		  AND (path NOT LIKE '%.github\%'  AND path NOT LIKE '%.github/%')
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

	// 6. Cálculo de Health Score (0 a 100). ISSUE-019: a guarda que pulava o
	// cálculo inteiro quando o vault não tinha documentos deixava o score no
	// valor inicial (100) — nota máxima para um vault que não existe. Agora a
	// matemática vive em CalculateHealthScore, compartilhada com o SQLite, e o
	// score só existe quando há denominador.
	report.HealthScore = CalculateHealthScore(*report)

	// 7. Aviso de ortografia (ISSUE-017). Fica DEPOIS do Health Score de propósito:
	// "sao" e "são" tokenizam diferente e a busca falha, mas isso não é corrupção
	// de dado — não pode virar penalidade sem decisão de produto.
	acc := &OrthographyAccumulator{}
	titleRows, err := s.db.QueryContext(ctx, `
		SELECT id, title FROM documents
		WHERE repository = $1 AND title IS NOT NULL
	`, repo)
	if err == nil {
		for titleRows.Next() {
			var id, title string
			if err := titleRows.Scan(&id, &title); err == nil {
				acc.Add(id, title)
			}
		}
		titleRows.Close()
	}
	abstractRows, err := s.db.QueryContext(ctx, `
		SELECT id, abstract FROM documents
		WHERE repository = $1 AND abstract IS NOT NULL
	`, repo)
	if err == nil {
		for abstractRows.Next() {
			var id, abstract string
			if err := abstractRows.Scan(&id, &abstract); err == nil {
				acc.Add(id, abstract)
			}
		}
		abstractRows.Close()
	}
	chunkRows, err := s.db.QueryContext(ctx, `
		SELECT c.document_id, c.content FROM chunks c
		WHERE c.repository = $1
	`, repo)
	if err == nil {
		for chunkRows.Next() {
			var documentID, content string
			if err := chunkRows.Scan(&documentID, &content); err == nil {
				acc.Add(documentID, content)
			}
		}
		chunkRows.Close()
	}
	report.OrthographyDocuments, report.OrthographyIssues = acc.Report(OrthographyIssueLimit)

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
		  AND target_id NOT IN (SELECT id FROM graph_nodes WHERE repository = $1 AND type = $2)
	`, repo, codeSymbolNodeType)
	if err != nil {
		return 0, err
	}
	deadFixed, _ := resDead.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(loopsFixed + deadFixed), nil
}
