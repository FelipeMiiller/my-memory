package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// SearchFTS busca trechos via texto completo na tabela chunks_fts utilizando ordenação nativa BM25 do FTS5
func SearchFTS(ctx context.Context, database *sql.DB, query string, limit int) ([]SearchResult, error) {
	sanitized := store.SanitizeFTS5Query(query)
	if sanitized == "" {
		return nil, nil
	}

	q := `
	SELECT c.id, c.document_id, c.content, fts.rank
	FROM chunks_fts fts
	JOIN chunks c ON c.id = fts.chunk_id
	WHERE chunks_fts MATCH ?
	ORDER BY fts.rank
	LIMIT ?
	`

	rows, err := database.QueryContext(ctx, q, sanitized, limit)
	if err != nil {
		return nil, fmt.Errorf("erro na busca textual FTS5: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var rank float64
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Content, &rank); err != nil {
			return nil, err
		}
		// Distância invertida/normalizada baseada no rank BM25
		r.Distance = rank

		neighbors, err := GetNodeNeighbors(ctx, database, r.DocumentID, 1)
		if err == nil {
			r.Neighbors = neighbors
		}

		results = append(results, r)
	}

	return results, nil
}

// SearchHybridRRF funde busca textual FTS5, busca vetorial (sqlite-vec ou TurboQuant) e vizinhos no grafo via RRF
func SearchHybridRRF(ctx context.Context, database *sql.DB, q *turboquant.Quantizer, query string, queryVec []float32, limit, k int, useTurbo bool) ([]SearchResult, error) {
	candidateLimit := limit * 2
	if candidateLimit < 10 {
		candidateLimit = 10
	}

	// 1. Busca Léxica (FTS5)
	var ftsResults []SearchResult
	if strings.TrimSpace(query) != "" {
		ftsResults, _ = SearchFTS(ctx, database, query, candidateLimit)
	}

	// 2. Busca Vetorial (sqlite-vec ou TurboQuant)
	var vecResults []SearchResult
	if len(queryVec) > 0 {
		if useTurbo && q != nil {
			vecResults, _ = SearchTurboQuant(ctx, database, q, queryVec, candidateLimit)
		} else {
			vecResults, _ = SearchKNN(ctx, database, queryVec, candidateLimit)
		}
	}

	// 3. Expansão de vizinhos estruturais no grafo a partir dos documentos mais relevantes
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
		neighbors, err := GetNodeNeighbors(ctx, database, seed, 1)
		if err != nil {
			continue
		}

		for _, n := range neighbors {
			rows, err := database.QueryContext(ctx, `
				SELECT id, document_id, content
				FROM chunks
				WHERE document_id = ?
				ORDER BY chunk_index
				LIMIT 2
			`, n)
			if err != nil {
				continue
			}

			for rows.Next() {
				var gr SearchResult
				if err := rows.Scan(&gr.ChunkID, &gr.DocumentID, &gr.Content); err == nil {
					if !seenChunks[gr.ChunkID] {
						seenChunks[gr.ChunkID] = true
						graphResults = append(graphResults, gr)
					}
				}
			}
			rows.Close()
		}
	}

	// 4. Fusão RRF através de store.FuseSearchResults
	sources := []store.RankedResultSource{
		{Name: "fts", Results: toStoreResults(ftsResults)},
		{Name: "vector", Results: toStoreResults(vecResults)},
		{Name: "graph", Results: toStoreResults(graphResults)},
	}

	fusedStoreResults := store.FuseSearchResults(sources, k, limit)
	return fromStoreResults(fusedStoreResults), nil
}

func toStoreResults(items []SearchResult) []store.SearchResult {
	res := make([]store.SearchResult, len(items))
	for i, it := range items {
		res[i] = store.SearchResult{
			ChunkID:    it.ChunkID,
			DocumentID: it.DocumentID,
			Content:    it.Content,
			Distance:   it.Distance,
			Score:      it.Score,
			Sources:    it.Sources,
			Neighbors:  it.Neighbors,
		}
	}
	return res
}

func fromStoreResults(items []store.SearchResult) []SearchResult {
	res := make([]SearchResult, len(items))
	for i, it := range items {
		res[i] = SearchResult{
			ChunkID:    it.ChunkID,
			DocumentID: it.DocumentID,
			Content:    it.Content,
			Distance:   it.Distance,
			Score:      it.Score,
			Sources:    it.Sources,
			Neighbors:  it.Neighbors,
		}
	}
	return res
}
