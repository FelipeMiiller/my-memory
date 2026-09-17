package graphview

import (
	"fmt"
	"testing"
)

// generateDocs cria N docs com IDs/títulos realistas pra benchmarks
func generateDocs(n int) []RawDoc {
	docs := make([]RawDoc, n)
	for i := 0; i < n; i++ {
		docs[i] = RawDoc{
		ID:        fmt.Sprintf("doc-%d.md", i),
		Title:     fmt.Sprintf("Document %d", i),
		UpdatedAt: 1700000000 + int64(i),
	}
	}
	return docs
}

// generateEdges cria ~3 edges por doc (similar à densidade típica de wikilinks)
func generateEdges(n int) []RawEdge {
	edges := make([]RawEdge, 0, n*3)
	for i := 0; i < n; i++ {
		for j := 0; j < 3 && i+j+1 < n; j++ {
			edges = append(edges, RawEdge{
			Source:   fmt.Sprintf("doc-%d.md", i),
			Target:   fmt.Sprintf("doc-%d.md", i+j+1),
			Relation: "links_to",
			})
		}
	}
	return edges
}

// BenchmarkBuildGraphView_Small mede BuildGraphView com 50 docs (~150 edges)
func BenchmarkBuildGraphView_Small(b *testing.B) {
	docs := generateDocs(50)
	edges := generateEdges(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraphView(docs, edges, "", 5, "bench-repo")
	}
}

// BenchmarkBuildGraphView_Medium mede BuildGraphView com 500 docs (~1500 edges)
func BenchmarkBuildGraphView_Medium(b *testing.B) {
	docs := generateDocs(500)
	edges := generateEdges(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraphView(docs, edges, "", 5, "bench-repo")
	}
}

// BenchmarkBuildGraphView_Large mede BuildGraphView com 2000 docs (~6000 edges)
func BenchmarkBuildGraphView_Large(b *testing.B) {
	docs := generateDocs(2000)
	edges := generateEdges(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraphView(docs, edges, "", 5, "bench-repo")
	}
}

// BenchmarkRenderHTML mede o custo de renderizar o template standalone (cobre A1 — encoding correctness)
func BenchmarkRenderHTML(b *testing.B) {
	gv := &GraphView{
		Title:      "Grafo de Memória [bench] — Português: Memória, Síntese, Usuário",
		Repository: "bench-repo",
	}
	for i := 0; i < 50; i++ {
		gv.Nodes = append(gv.Nodes, Node{
		ID:    fmt.Sprintf("n-%d", i),
		Title: fmt.Sprintf("Nó %d — acentuação: ação, opção, fenômeno", i),
		Type:  "guide",
		})
	}
	for i := 0; i < 100; i++ {
		gv.Edges = append(gv.Edges, Edge{
		Source: fmt.Sprintf("n-%d", i%50),
		Target: fmt.Sprintf("n-%d", (i+1)%50),
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RenderHTML(gv)
		if err != nil {
			b.Fatalf("RenderHTML: %v", err)
		}
	}
}