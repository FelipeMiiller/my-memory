package store

import (
	"fmt"
	"sort"
)

// DefaultRRFK define a constante padrão de suavização para RRF (k = 60).
const DefaultRRFK = 60

// RankedSource representa uma lista ordenada de itens gerada por uma modalidade de busca.
type RankedSource[T comparable] struct {
	Name  string // Nome da fonte (ex: "fts", "vector", "graph")
	Items []T    // Itens ordenados por relevância decrescente
}

// FusedItem representa um item resultante da fusão RRF com seu score e lista de fontes.
type FusedItem[T comparable] struct {
	Item    T
	Score   float64
	Sources []string // Detalhe de onde veio e o rank na fonte (ex: ["fts:1", "vector:3"])
}

// FuseRRF calcula a fusão Reciprocal Rank Fusion para múltiplas listas ranqueadas.
// Fórmula: Score(d) = Σ 1 / (k + rank(d)), onde rank é 1-indexed.
// Se k <= 0, k assume DefaultRRFK (60).
func FuseRRF[T comparable](sources []RankedSource[T], k int) []FusedItem[T] {
	if k <= 0 {
		k = DefaultRRFK
	}

	scores := make(map[T]float64)
	sourceMap := make(map[T][]string)
	seenInSource := make(map[T]map[string]bool)

	for _, src := range sources {
		for rankIdx, item := range src.Items {
			rank := rankIdx + 1 // 1-based rank

			// Evita duplicatas do mesmo item na mesma lista fonte
			if seenInSource[item] == nil {
				seenInSource[item] = make(map[string]bool)
			}
			if seenInSource[item][src.Name] {
				continue
			}
			seenInSource[item][src.Name] = true

			reciprocal := 1.0 / float64(k+rank)
			scores[item] += reciprocal
			sourceMap[item] = append(sourceMap[item], fmt.Sprintf("%s:%d", src.Name, rank))
		}
	}

	fused := make([]FusedItem[T], 0, len(scores))
	for item, score := range scores {
		fused = append(fused, FusedItem[T]{
			Item:    item,
			Score:   score,
			Sources: sourceMap[item],
		})
	}

	// Ordena por score decrescente
	sort.Slice(fused, func(i, j int) bool {
		if fused[i].Score != fused[j].Score {
			return fused[i].Score > fused[j].Score
		}
		// Desempate determinístico pela quantidade de fontes e representação do item
		if len(fused[i].Sources) != len(fused[j].Sources) {
			return len(fused[i].Sources) > len(fused[j].Sources)
		}
		return fmt.Sprintf("%v", fused[i].Item) < fmt.Sprintf("%v", fused[j].Item)
	})

	return fused
}

// RankedResultSource agrupa resultados de SearchResult provenientes de uma fonte de busca.
type RankedResultSource struct {
	Name    string
	Results []SearchResult
}

// FuseSearchResults funde múltiplos slices de SearchResult aplicando RRF baseado no ChunkID ou DocumentID.
func FuseSearchResults(sources []RankedResultSource, k int, limit int) []SearchResult {
	if k <= 0 {
		k = DefaultRRFK
	}

	type chunkMeta struct {
		result   SearchResult
		score    float64
		sources  []string
		seenSrcs map[string]bool
	}

	metaMap := make(map[string]*chunkMeta)
	var orderedKeys []string

	for _, src := range sources {
		for rankIdx, res := range src.Results {
			key := res.ChunkID
			if key == "" {
				key = res.DocumentID
			}
			if key == "" {
				continue
			}

			rank := rankIdx + 1
			reciprocal := 1.0 / float64(k+rank)

			meta, exists := metaMap[key]
			if !exists {
				meta = &chunkMeta{
					result:   res,
					seenSrcs: make(map[string]bool),
				}
				metaMap[key] = meta
				orderedKeys = append(orderedKeys, key)
			} else {
				// Combina conexões de grafo se presentes
				if len(res.Neighbors) > 0 {
					meta.result.Neighbors = mergeNeighbors(meta.result.Neighbors, res.Neighbors)
				}
				if meta.result.Content == "" && res.Content != "" {
					meta.result.Content = res.Content
				}
				if meta.result.Distance == 0 && res.Distance > 0 {
					meta.result.Distance = res.Distance
				}
			}

			if !meta.seenSrcs[src.Name] {
				meta.seenSrcs[src.Name] = true
				meta.score += reciprocal
				meta.sources = append(meta.sources, fmt.Sprintf("%s:%d", src.Name, rank))
			}
		}
	}

	fused := make([]SearchResult, 0, len(metaMap))
	for _, meta := range metaMap {
		res := meta.result
		res.Score = meta.score
		res.Sources = meta.sources
		fused = append(fused, res)
	}

	sort.Slice(fused, func(i, j int) bool {
		if fused[i].Score != fused[j].Score {
			return fused[i].Score > fused[j].Score
		}
		if len(fused[i].Sources) != len(fused[j].Sources) {
			return len(fused[i].Sources) > len(fused[j].Sources)
		}
		return fused[i].ChunkID < fused[j].ChunkID
	})

	if limit > 0 && len(fused) > limit {
		fused = fused[:limit]
	}

	return fused
}

func mergeNeighbors(existing, incoming []string) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, n := range existing {
		if !seen[n] {
			seen[n] = true
			merged = append(merged, n)
		}
	}
	for _, n := range incoming {
		if !seen[n] {
			seen[n] = true
			merged = append(merged, n)
		}
	}
	return merged
}
