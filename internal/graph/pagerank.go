package graph

import (
	"math"
	"strings"
)

// Pesos padrão para arestas epistêmicas
const (
	WeightExtracted  = 1.0
	WeightInferred   = 0.6
	WeightTag        = 0.3
	DefaultDamping   = 0.85
	DefaultMaxIter   = 30
	DefaultTolerance = 1e-6
)

// WeightedEdge representa uma aresta direcionada ponderada entre dois nós do grafo
type WeightedEdge struct {
	Source string
	Target string
	Weight float64
	Type   string
}

// PageRankOptions define os hiperparâmetros de execução do algoritmo PageRank
type PageRankOptions struct {
	DampingFactor float64
	MaxIterations int
	Tolerance     float64
}

// DefaultPageRankOptions retorna os parâmetros canônicos recomendados
func DefaultPageRankOptions() PageRankOptions {
	return PageRankOptions{
		DampingFactor: DefaultDamping,
		MaxIterations: DefaultMaxIter,
		Tolerance:     DefaultTolerance,
	}
}

// ResolveEdgeWeight determina o peso numérico da aresta com base em seu tipo epistêmico
func ResolveEdgeWeight(edgeType string, customWeight float64) float64 {
	if customWeight > 0 {
		return customWeight
	}
	switch strings.ToUpper(strings.TrimSpace(edgeType)) {
	case "EXTRACTED":
		return WeightExtracted
	case "INFERRED":
		return WeightInferred
	case "TAG":
		return WeightTag
	default:
		return 1.0
	}
}

type inLink struct {
	source    string
	outWeight float64
	weight    float64
}

// ComputePageRank calcula o PageRank ponderado iterativo sobre nós e arestas fornecidos.
// Trata dangling nodes via redistribuição uniforme e converge quando delta < tol ou maxIter atingido.
// Retorna um mapa contendo o score de cada nó, cuja soma é estritamente 1.0 (se N > 0).
func ComputePageRank(nodes []string, edges []WeightedEdge, opts PageRankOptions) map[string]float64 {
	if opts.DampingFactor <= 0 || opts.DampingFactor >= 1.0 {
		opts.DampingFactor = DefaultDamping
	}
	if opts.MaxIterations <= 0 {
		opts.MaxIterations = DefaultMaxIter
	}
	if opts.Tolerance <= 0 {
		opts.Tolerance = DefaultTolerance
	}

	// 1. Identificar todos os nós únicos
	nodeSet := make(map[string]bool)
	for _, n := range nodes {
		if strings.TrimSpace(n) != "" {
			nodeSet[n] = true
		}
	}
	for _, e := range edges {
		if strings.TrimSpace(e.Source) != "" {
			nodeSet[e.Source] = true
		}
		if strings.TrimSpace(e.Target) != "" {
			nodeSet[e.Target] = true
		}
	}

	totalNodes := len(nodeSet)
	if totalNodes == 0 {
		return make(map[string]float64)
	}

	nodeList := make([]string, 0, totalNodes)
	for n := range nodeSet {
		nodeList = append(nodeList, n)
	}

	if totalNodes == 1 {
		return map[string]float64{nodeList[0]: 1.0}
	}

	// 2. Calcular pesos de saída (out-weight) e registrar conexões de entrada (in-links)
	outWeights := make(map[string]float64)
	for _, e := range edges {
		if !nodeSet[e.Source] || !nodeSet[e.Target] {
			continue
		}
		w := ResolveEdgeWeight(e.Type, e.Weight)
		if w <= 0 {
			w = 1.0
		}
		outWeights[e.Source] += w
	}

	inLinksMap := make(map[string][]inLink)
	for _, e := range edges {
		if !nodeSet[e.Source] || !nodeSet[e.Target] {
			continue
		}
		w := ResolveEdgeWeight(e.Type, e.Weight)
		if w <= 0 {
			w = 1.0
		}
		ow := outWeights[e.Source]
		inLinksMap[e.Target] = append(inLinksMap[e.Target], inLink{
			source:    e.Source,
			outWeight: ow,
			weight:    w,
		})
	}

	// 3. Inicializar scores uniformemente: PR_0(u) = 1 / N
	initialScore := 1.0 / float64(totalNodes)
	currentScores := make(map[string]float64, totalNodes)
	for _, n := range nodeList {
		currentScores[n] = initialScore
	}

	d := opts.DampingFactor
	nFloat := float64(totalNodes)

	// 4. Iterações com redistribuição de dangling nodes
	for iter := 0; iter < opts.MaxIterations; iter++ {
		// Soma das massas de nós sem saída (dangling)
		danglingSum := 0.0
		for _, n := range nodeList {
			if outWeights[n] == 0 {
				danglingSum += currentScores[n]
			}
		}

		// Contribuição base para cada nó: teletransporte aleatório + dangling redistribuído
		baseScore := ((1.0 - d) / nFloat) + (d * (danglingSum / nFloat))

		nextScores := make(map[string]float64, totalNodes)
		delta := 0.0

		for _, n := range nodeList {
			incomingFlow := 0.0
			for _, in := range inLinksMap[n] {
				if in.outWeight > 0 {
					incomingFlow += currentScores[in.source] * (in.weight / in.outWeight)
				}
			}
			newScore := baseScore + (d * incomingFlow)
			nextScores[n] = newScore
			delta += math.Abs(newScore - currentScores[n])
		}

		currentScores = nextScores
		if delta < opts.Tolerance {
			break
		}
	}

	// 5. Normalização final garantindo que a soma seja exatamente 1.0
	sum := 0.0
	for _, score := range currentScores {
		sum += score
	}
	if sum > 0 {
		for n := range currentScores {
			currentScores[n] /= sum
		}
	}

	return currentScores
}
