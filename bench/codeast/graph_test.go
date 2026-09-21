package codeast_bench

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
)

// BenchmarkCodeGraphDepth2 mede latência p99 de code_graph depth=2 sobre
// um grafo pré-populado. Alvo CA-13: ≤ 100ms. Indicativo para mock.
func BenchmarkCodeGraphDepth2(b *testing.B) {
	const Target = 0.100 // 100ms
	database, _ := newBenchDB(b)
	prepopulateGraphDB(b, database, 200)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		elapsed := measureSeconds(func() {
			_, _, err := mcp.SQLCodeNeighbors(context.Background(), database,
				"pkg0.Func0", 2, "both")
			if err != nil {
				b.Fatalf("SQLCodeNeighbors: %v", err)
			}
		})
		checkAlvo(b, "BenchmarkCodeGraphDepth2", elapsed, Target, IndicativeAtivado)
	}
}

// TestCodeGraphDepth2_Alvo é o harness 1x com assert programático.
func TestCodeGraphDepth2_Alvo(t *testing.T) {
	if testing.Short() {
		t.Skip("skip em -short")
	}
	const Target = 0.100

	database, _ := newBenchDB(t)
	prepopulateGraphDB(t, database, 200)

	// Warm-up.
	_, _, _ = mcp.SQLCodeNeighbors(context.Background(), database, "pkg0.Func0", 2, "both")

	elapsed := measureSeconds(func() {
		_, _, err := mcp.SQLCodeNeighbors(context.Background(), database, "pkg0.Func0", 2, "both")
		if err != nil {
			t.Fatalf("SQLCodeNeighbors: %v", err)
		}
	})
	t.Logf("code_graph depth=2 in %.6fs (target=%.6fs indicative=%v)", elapsed, Target, IndicativeAtivado)
	checkAlvo(t, "TestCodeGraphDepth2_Alvo", elapsed, Target, IndicativeAtivado)
}

// prepopulateGraphDB cria uma cadeia linear de N nodes com edges entre consecutivos.
func prepopulateGraphDB(tb testing.TB, database *sql.DB, n int) int64 {
	tb.Helper()
	ctx := context.Background()
	fileID, err := db.UpsertCodeFile(ctx, database, "bench/graph.go", "go",
		"hash-graph", "", int64(n*100), 1700000000)
	if err != nil {
		tb.Fatalf("UpsertCodeFile: %v", err)
	}
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		name := "Func" + strconv.Itoa(i)
		qn := "pkg0." + name
		id, err := db.UpsertCodeSymbol(ctx, database, fileID, "function", name, qn, "", "", i+1, i+1, 0, 0)
		if err != nil {
			tb.Fatalf("UpsertCodeSymbol i=%d: %v", i, err)
		}
		ids[i] = id
	}
	// Edge chain: 0 → 1 → 2 → ... → n-1
	for i := 0; i < n-1; i++ {
		if err := db.UpsertCodeEdge(ctx, database, ids[i], ids[i+1], fileID, "calls", i+1, i+1, 1.0); err != nil {
			tb.Fatalf("UpsertCodeEdge i=%d: %v", i, err)
		}
	}
	return ids[0]
}
