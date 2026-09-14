package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/graph"
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
	Abstract   string   `json:"abstract,omitempty"`
	Category   string   `json:"category,omitempty"`
}

// SearchKNN busca os pedaços mais próximos usando sqlite-vec nativo
func SearchKNN(ctx context.Context, db *sql.DB, queryVec []float32, limit int) ([]SearchResult, error) {
	return SearchKNNWithOptions(ctx, db, queryVec, limit, store.SearchOptions{Level: "l1"})
}

// SearchKNNWithOptions busca os pedaços mais próximos usando sqlite-vec nativo com opções de busca
func SearchKNNWithOptions(ctx context.Context, db *sql.DB, queryVec []float32, limit int, searchOpts store.SearchOptions) ([]SearchResult, error) {
	if !HasSqliteVec {
		return nil, fmt.Errorf("sqlite-vec não está disponível nesta plataforma (use TurboQuant)")
	}

	vecBlob, err := serializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar vetor de busca: %w", err)
	}

	cat := strings.TrimSpace(searchOpts.Category)
	query := `
	SELECT c.id, c.document_id, c.content, v.distance, COALESCE(d.updated_at, 0),
	       COALESCE(d.abstract, ''), COALESCE(d.category, 'resource')
	FROM chunks_vec v
	JOIN chunks c ON c.id = v.chunk_id
	LEFT JOIN documents d ON d.id = c.document_id
	WHERE v.embedding MATCH ? AND k = ?
	  AND (? = '' OR LOWER(d.category) = LOWER(?))
	ORDER BY v.distance
	`

	rows, err := db.QueryContext(ctx, query, vecBlob, limit, cat, cat)
	if err != nil {
		return nil, fmt.Errorf("erro na busca vetorial: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &r.Distance, &r.UpdatedAt, &r.Abstract, &r.Category); err != nil {
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
	return SearchTurboQuantWithOptions(ctx, db, q, queryVec, limit, store.SearchOptions{Level: "l1"})
}

// SearchTurboQuantWithOptions busca usando a projeção ortogonal e os vetores 4-bit comprimidos com opções de busca
func SearchTurboQuantWithOptions(ctx context.Context, db *sql.DB, q *turboquant.Quantizer, queryVec []float32, limit int, searchOpts store.SearchOptions) ([]SearchResult, error) {
	rotatedQuery := q.RotateQuery(queryVec)
	cat := strings.TrimSpace(searchOpts.Category)

	rows, err := db.QueryContext(ctx, `
		SELECT tq.chunk_id, c.document_id, c.content, tq.scale, tq.data, COALESCE(d.updated_at, 0),
		       COALESCE(d.abstract, ''), COALESCE(d.category, 'resource')
		FROM chunks_turboquant tq
		JOIN chunks c ON c.id = tq.chunk_id
		LEFT JOIN documents d ON d.id = c.document_id
		WHERE (? = '' OR LOWER(d.category) = LOWER(?))
	`, cat, cat)
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
		abstract   string
		category   string
	}

	var items []scoredItem
	for rows.Next() {
		var chunkID, docID, content string
		var scale float32
		var data []byte
		var updatedAt int64
		var abstract, category string

		if err := rows.Scan(&chunkID, &docID, &content, &scale, &data, &updatedAt, &abstract, &category); err != nil {
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
			abstract:   abstract,
			category:   category,
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
			Abstract:   it.abstract,
			Category:   it.category,
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

// ResolveNodeCanonicalID resolve um identificador informal, slug, título ou caminho relativo para o ID canônico no grafo
func ResolveNodeCanonicalID(ctx context.Context, db *sql.DB, query string) (string, error) {
	clean := strings.TrimSpace(query)
	clean = strings.TrimPrefix(clean, "[[")
	clean = strings.TrimSuffix(clean, "]]")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "", fmt.Errorf("identificador de busca não pode ser vazio")
	}

	cleanSlash := strings.ReplaceAll(clean, "\\", "/")
	cleanBack := strings.ReplaceAll(clean, "/", "\\")

	// 1. Checagem exata em documents por id ou path (suportando / e \)
	var canonicalID string
	err := db.QueryRowContext(ctx, `
		SELECT id FROM documents 
		WHERE id = ? OR path = ? OR path = ? OR path = ? OR path = ?
		LIMIT 1
	`, clean, cleanSlash, cleanBack, cleanSlash+".md", cleanBack+".md").Scan(&canonicalID)
	if err == nil && canonicalID != "" {
		return canonicalID, nil
	}

	// 2. Checagem em graph_nodes por id ou name
	err = db.QueryRowContext(ctx, `
		SELECT id FROM graph_nodes 
		WHERE id = ? OR name = ? OR id = ? OR name = ?
		LIMIT 1
	`, clean, clean, cleanSlash, cleanBack).Scan(&canonicalID)
	if err == nil && canonicalID != "" {
		return canonicalID, nil
	}

	// 3. Checagem exata por título
	err = db.QueryRowContext(ctx, `
		SELECT id FROM documents 
		WHERE LOWER(title) = LOWER(?)
		LIMIT 1
	`, clean).Scan(&canonicalID)
	if err == nil && canonicalID != "" {
		return canonicalID, nil
	}

	// 4. Checagem difusa (LIKE) em id, path ou title
	likeSlash := "%" + cleanSlash + "%"
	likeBack := "%" + cleanBack + "%"
	err = db.QueryRowContext(ctx, `
		SELECT id FROM documents 
		WHERE id LIKE ? OR path LIKE ? OR path LIKE ? OR title LIKE ?
		ORDER BY 
			CASE 
				WHEN id LIKE ? THEN 1
				WHEN path LIKE ? OR path LIKE ? THEN 2
				ELSE 3
			END
		LIMIT 1
	`, likeSlash, likeSlash, likeBack, "%"+clean+"%", clean+"%", cleanSlash+"%", cleanBack+"%").Scan(&canonicalID)
	if err == nil && canonicalID != "" {
		return canonicalID, nil
	}

	// 5. Se existe nas arestas do grafo como source ou target direto
	err = db.QueryRowContext(ctx, `
		SELECT target_id FROM graph_edges WHERE target_id = ? OR target_id = ? OR target_id = ?
		UNION
		SELECT source_id FROM graph_edges WHERE source_id = ? OR source_id = ? OR source_id = ?
		LIMIT 1
	`, clean, cleanSlash, cleanBack, clean, cleanSlash, cleanBack).Scan(&canonicalID)
	if err == nil && canonicalID != "" {
		return canonicalID, nil
	}

	return "", fmt.Errorf("nó '%s' não encontrado no grafo", query)
}

// CalculateImpactForTarget resolve o nó alvo e calcula a análise de impacto (blast radius) completa no SQLite
func CalculateImpactForTarget(ctx context.Context, db *sql.DB, targetQuery string, maxDepth int) (*graph.ImpactResult, error) {
	canonicalID, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, err
	}

	// 1. Carregar todos os nós e seus tipos
	nodeTypes := make(map[string]string)
	nodesSet := make(map[string]bool)

	docRows, err := db.QueryContext(ctx, "SELECT id FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id string
			if err := docRows.Scan(&id); err == nil {
				nodesSet[id] = true
				nodeTypes[id] = "note"
			}
		}
	}

	nodeRows, err := db.QueryContext(ctx, "SELECT id, type FROM graph_nodes")
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var id, nType string
			if err := nodeRows.Scan(&id, &nType); err == nil {
				nodesSet[id] = true
				nodeTypes[id] = nType
			}
		}
	}

	// 2. Carregar arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0) FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para análise de impacto: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source: src,
			Target: tgt,
			Type:   rel,
			Weight: weight,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	// 3. Calcular PageRanks e Comunidades para enriquecer a análise
	prScores := graph.ComputePageRank(allNodes, edges, graph.DefaultPageRankOptions())

	commRes := graph.DetectCommunities(allNodes, edges, graph.DefaultCommunityOptions())
	commMap := make(map[string]int)
	for _, c := range commRes.Communities {
		for _, m := range c.Members {
			commMap[m] = c.ID
		}
	}

	opts := graph.ImpactOptions{
		MaxDepth:    maxDepth,
		NodeTypes:   nodeTypes,
		Communities: commMap,
		PageRanks:   prScores,
	}

	return graph.CalculateImpact(allNodes, edges, canonicalID, opts)
}

// InspectNode constrói a visualização cirúrgica em 3 colunas (Triptych) de um nó no banco SQLite
func InspectNode(ctx context.Context, db *sql.DB, targetQuery string, maxContentLen int) (*graph.TriptychView, error) {
	canonicalID, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, err
	}

	// 1. Carregar metadados do documento alvo
	var targetTitle, targetPath string
	var targetUpdatedAt int64
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(title, ''), COALESCE(path, ''), COALESCE(updated_at, 0) FROM documents WHERE id = ?", canonicalID).Scan(&targetTitle, &targetPath, &targetUpdatedAt)

	var targetType string
	_ = db.QueryRowContext(ctx, "SELECT COALESCE(type, 'note') FROM graph_nodes WHERE id = ?", canonicalID).Scan(&targetType)
	if targetType == "" {
		targetType = "note"
	}
	if targetTitle == "" {
		targetTitle = canonicalID
	}

	// 2. Carregar preview do conteúdo do nó alvo
	var contentBuilder strings.Builder
	chunkRows, err := db.QueryContext(ctx, "SELECT content FROM chunks WHERE document_id = ? ORDER BY chunk_index ASC", canonicalID)
	if err == nil {
		defer chunkRows.Close()
		for chunkRows.Next() {
			var chunkContent string
			if err := chunkRows.Scan(&chunkContent); err == nil {
				if contentBuilder.Len() > 0 {
					contentBuilder.WriteString("\n\n")
				}
				contentBuilder.WriteString(chunkContent)
				if maxContentLen > 0 && contentBuilder.Len() > maxContentLen*3 {
					break
				}
			}
		}
	}

	// 3. Tags associadas ao nó
	var tags []string
	tagRows, err := db.QueryContext(ctx, "SELECT target_id FROM graph_edges WHERE source_id = ? AND (relation = 'tagged_as' OR target_id LIKE '#%')", canonicalID)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var t string
			if err := tagRows.Scan(&t); err == nil {
				tags = append(tags, t)
			}
		}
	}

	// 4. Carregar todos os nós, tipos, títulos e status de existência
	nodeTypes := make(map[string]string)
	titles := make(map[string]string)
	existingNodes := make(map[string]bool)
	nodesSet := make(map[string]bool)

	docRows, err := db.QueryContext(ctx, "SELECT id, COALESCE(title, id) FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id, t string
			if err := docRows.Scan(&id, &t); err == nil {
				nodesSet[id] = true
				existingNodes[id] = true
				if t != "" {
					existingNodes[t] = true
				}
				titles[id] = t
				nodeTypes[id] = "note"
			}
		}
	}

	gnRows, err := db.QueryContext(ctx, "SELECT id, COALESCE(name, id), COALESCE(type, 'other') FROM graph_nodes")
	if err == nil {
		defer gnRows.Close()
		for gnRows.Next() {
			var id, n, t string
			if err := gnRows.Scan(&id, &n, &t); err == nil {
				nodesSet[id] = true
				if titles[id] == "" || titles[id] == id {
					titles[id] = n
				}
				if nodeTypes[id] == "" {
					nodeTypes[id] = t
				}
				if strings.HasPrefix(id, "#") || (t != "note" && t != "other") {
					existingNodes[id] = true
				}
			}
		}
	}

	// 5. Carregar todas as arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0) FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para inspeção: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source: src,
			Target: tgt,
			Type:   rel,
			Weight: weight,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	// 6. Calcular PageRank e Comunidades
	prScores := graph.ComputePageRank(allNodes, edges, graph.DefaultPageRankOptions())
	commRes := graph.DetectCommunities(allNodes, edges, graph.DefaultCommunityOptions())
	commMap := make(map[string]int)
	commLabels := make(map[int]string)
	for _, c := range commRes.Communities {
		commLabels[c.ID] = c.DominantType
		for _, m := range c.Members {
			commMap[m] = c.ID
		}
	}

	// 7. Calcular Impacto / Risco para o alvo
	impactRes, _ := graph.CalculateImpact(allNodes, edges, canonicalID, graph.ImpactOptions{
		MaxDepth:    2,
		NodeTypes:   nodeTypes,
		Communities: commMap,
		PageRanks:   prScores,
	})

	var updatedTime time.Time
	if targetUpdatedAt > 0 {
		updatedTime = time.Unix(targetUpdatedAt, 0)
	}

	targetSummary := graph.NodeSummary{
		ID:             canonicalID,
		Title:          targetTitle,
		Path:           targetPath,
		Type:           targetType,
		Tags:           tags,
		PageRank:       prScores[canonicalID],
		CommunityID:    commMap[canonicalID],
		CommunityLabel: commLabels[commMap[canonicalID]],
		ContentPreview: contentBuilder.String(),
		UpdatedAt:      updatedTime,
	}

	opts := graph.InspectorOptions{
		MaxContentLength: maxContentLen,
		PageRanks:        prScores,
		Communities:      commMap,
		CommunityLabels:  commLabels,
		NodeTypes:        nodeTypes,
		Titles:           titles,
		ExistingNodes:    existingNodes,
		RiskResult:       impactRes,
	}

	return graph.BuildTriptychView(targetSummary, edges, opts)
}

// FindPath resolve os nós de origem e destino e calcula o menor caminho entre eles no SQLite
func FindPath(ctx context.Context, db *sql.DB, sourceQuery, targetQuery string, opts graph.PathOptions) (*graph.PathResult, error) {
	canonicalSource, err := ResolveNodeCanonicalID(ctx, db, sourceQuery)
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver nó de origem: %w", err)
	}

	canonicalTarget, err := ResolveNodeCanonicalID(ctx, db, targetQuery)
	if err != nil {
		return nil, fmt.Errorf("falha ao resolver nó de destino: %w", err)
	}

	// 1. Carregar todos os nós
	nodesSet := make(map[string]bool)
	docRows, err := db.QueryContext(ctx, "SELECT id FROM documents")
	if err == nil {
		defer docRows.Close()
		for docRows.Next() {
			var id string
			if err := docRows.Scan(&id); err == nil {
				nodesSet[id] = true
			}
		}
	}

	nodeRows, err := db.QueryContext(ctx, "SELECT id FROM graph_nodes")
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var id string
			if err := nodeRows.Scan(&id); err == nil {
				nodesSet[id] = true
			}
		}
	}

	// 2. Carregar arestas
	edgeRows, err := db.QueryContext(ctx, "SELECT source_id, target_id, COALESCE(relation, 'links_to'), COALESCE(weight, 1.0), COALESCE(epistemic_status, 'EXTRACTED') FROM graph_edges")
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arestas para busca de caminho: %w", err)
	}
	defer edgeRows.Close()

	var edges []graph.WeightedEdge
	for edgeRows.Next() {
		var src, tgt, rel, epStatus string
		var weight float64
		if err := edgeRows.Scan(&src, &tgt, &rel, &weight, &epStatus); err != nil {
			return nil, err
		}
		nodesSet[src] = true
		nodesSet[tgt] = true
		edges = append(edges, graph.WeightedEdge{
			Source:          src,
			Target:          tgt,
			Type:            rel,
			Weight:          weight,
			EpistemicStatus: epStatus,
		})
	}

	allNodes := make([]string, 0, len(nodesSet))
	for n := range nodesSet {
		allNodes = append(allNodes, n)
	}

	return graph.FindPath(allNodes, edges, canonicalSource, canonicalTarget, opts)
}


