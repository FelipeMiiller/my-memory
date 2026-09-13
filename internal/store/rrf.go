package store

import (
	"fmt"
	"math"
	"sort"
	"time"
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

// DecayOptions configura os parâmetros de decaimento temporal exponencial para a busca híbrida.
type DecayOptions struct {
	Enabled  bool    `json:"enabled"`
	HalfLife float64 `json:"half_life"` // Meia-vida em dias (padrão: 30)
	Weight   float64 `json:"weight"`    // Peso do decaimento w in [0.0, 1.0] (padrão: 0.3)
	RefTime  int64   `json:"ref_time"`  // Timestamp de referência Unix em segundos (0 = time.Now().Unix())
}

// DefaultDecayOptions retorna as opções padrão recomendadas (30 dias de meia-vida, peso 0.3).
func DefaultDecayOptions() DecayOptions {
	return DecayOptions{
		Enabled:  false,
		HalfLife: 30.0,
		Weight:   0.3,
		RefTime:  0,
	}
}

// CalculateTimeDecay calcula o fator multiplicador temporal com base na idade do documento,
// tempo de meia-vida e peso de decaimento com piso assintótico:
// Multiplier = (1 - w) + w * 2^(-deltaT / Thalf)
// Se updatedAt <= 0, retorna 1.0 (neutro).
func CalculateTimeDecay(updatedAt, refTime int64, halfLifeDays, weight float64) float64 {
	if updatedAt <= 0 {
		return 1.0
	}
	if refTime <= 0 {
		refTime = time.Now().Unix()
	}
	deltaT := refTime - updatedAt
	if deltaT < 0 {
		deltaT = 0
	}

	if halfLifeDays <= 0 {
		halfLifeDays = 30.0
	}
	if weight < 0.0 {
		weight = 0.0
	} else if weight > 1.0 {
		weight = 1.0
	}

	tHalfSeconds := halfLifeDays * 86400.0
	decay := math.Pow(2.0, -float64(deltaT)/tHalfSeconds)
	multiplier := (1.0 - weight) + (weight * decay)
	return multiplier
}

// FuseSearchResultsWithDecay funde múltiplos slices de SearchResult aplicando RRF ponderado por decaimento temporal.
func FuseSearchResultsWithDecay(sources []RankedResultSource, k int, limit int, opts DecayOptions) []SearchResult {
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
				if res.UpdatedAt > meta.result.UpdatedAt {
					meta.result.UpdatedAt = res.UpdatedAt
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
		decayMult := 1.0
		if opts.Enabled {
			decayMult = CalculateTimeDecay(res.UpdatedAt, opts.RefTime, opts.HalfLife, opts.Weight)
		}
		res.Score = meta.score * decayMult
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

// FuseSearchResults funde múltiplos slices de SearchResult aplicando RRF baseado no ChunkID ou DocumentID.
func FuseSearchResults(sources []RankedResultSource, k int, limit int) []SearchResult {
	return FuseSearchResultsWithDecay(sources, k, limit, DecayOptions{Enabled: false})
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
