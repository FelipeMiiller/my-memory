package store

import (
	"fmt"
	"testing"
)

func generateBenchmarkResults(count int, prefix string) []SearchResult {
	results := make([]SearchResult, count)
	for i := 0; i < count; i++ {
		docID := fmt.Sprintf("doc-%d", (i*7)%count)
		results[i] = SearchResult{
			ChunkID:    fmt.Sprintf("%s#0", docID),
			DocumentID: docID,
			Content:    fmt.Sprintf("Benchmark content for %s from source %s", docID, prefix),
			Distance:   float64(i) * 0.05,
		}
	}
	return results
}

func BenchmarkFuseRRF_100Items(b *testing.B) {
	sources := []RankedResultSource{
		{Name: "fts", Results: generateBenchmarkResults(100, "fts")},
		{Name: "vector", Results: generateBenchmarkResults(100, "vec")},
		{Name: "graph", Results: generateBenchmarkResults(100, "graph")},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = FuseSearchResults(sources, 60, 10)
	}
}

func BenchmarkFuseRRF_1000Items(b *testing.B) {
	sources := []RankedResultSource{
		{Name: "fts", Results: generateBenchmarkResults(1000, "fts")},
		{Name: "vector", Results: generateBenchmarkResults(1000, "vec")},
		{Name: "graph", Results: generateBenchmarkResults(1000, "graph")},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = FuseSearchResults(sources, 60, 20)
	}
}
