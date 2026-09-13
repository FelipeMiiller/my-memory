package graph

import (
	"fmt"
	"math/rand"
	"testing"
)

func generateBenchmarkGraph(numNodes, avgOutDegree int, rng *rand.Rand) ([]string, []WeightedEdge) {
	nodes := make([]string, numNodes)
	for i := 0; i < numNodes; i++ {
		nodes[i] = fmt.Sprintf("node_%d", i)
	}

	var edges []WeightedEdge
	for i := 0; i < numNodes; i++ {
		outCount := rng.Intn(avgOutDegree*2 + 1)
		for c := 0; c < outCount; c++ {
			targetIdx := rng.Intn(numNodes)
			if targetIdx != i {
				edgeType := "EXTRACTED"
				if rng.Float64() < 0.3 {
					edgeType = "INFERRED"
				}
				edges = append(edges, WeightedEdge{
					Source: nodes[i],
					Target: nodes[targetIdx],
					Type:   edgeType,
					Weight: 1.0,
				})
			}
		}
	}
	return nodes, edges
}

func BenchmarkComputePageRank_100Nodes(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	nodes, edges := generateBenchmarkGraph(100, 5, rng)
	opts := DefaultPageRankOptions()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputePageRank(nodes, edges, opts)
	}
}

func BenchmarkComputePageRank_500Nodes(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	nodes, edges := generateBenchmarkGraph(500, 5, rng)
	opts := DefaultPageRankOptions()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputePageRank(nodes, edges, opts)
	}
}

func BenchmarkComputePageRank_1000Nodes(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	nodes, edges := generateBenchmarkGraph(1000, 5, rng)
	opts := DefaultPageRankOptions()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ComputePageRank(nodes, edges, opts)
	}
}
