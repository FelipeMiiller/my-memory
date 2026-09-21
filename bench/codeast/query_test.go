package codeast_bench

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/mcp"
)

// BenchmarkHybridQuery mede latência p99 de query híbrida (code_symbols LIKE +
// boost CA-05). Alvo CA-13: ≤ 50ms. Indicativo para mock (sem embeddings reais).
func BenchmarkHybridQuery(b *testing.B) {
	const Target = 0.050 // 50ms
	database, _ := newBenchDB(b)
	prepopulateSymbolsDB(b, database, 5000)
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		elapsed := measureSeconds(func() {
			_, err := mcp.SQLCodeSearch(context.Background(), database, "Helper", "go", "", 10, true)
			if err != nil {
				b.Fatalf("SQLCodeSearch: %v", err)
			}
		})
		checkAlvo(b, "BenchmarkHybridQuery", elapsed, Target, IndicativeAtivado)
	}
}

// TestHybridQuery_Alvo é o harness 1x com assert programático.
func TestHybridQuery_Alvo(t *testing.T) {
	if testing.Short() {
		t.Skip("skip em -short")
	}
	const Target = 0.050

	database, _ := newBenchDB(t)
	prepopulateSymbolsDB(t, database, 5000)

	// Warm-up (cache effects).
	_, _ = mcp.SQLCodeSearch(context.Background(), database, "Helper", "go", "", 10, true)

	elapsed := measureSeconds(func() {
		_, err := mcp.SQLCodeSearch(context.Background(), database, "Helper", "go", "", 10, true)
		if err != nil {
			t.Fatalf("SQLCodeSearch: %v", err)
		}
	})
	t.Logf("hybrid query in %.6fs (target=%.6fs indicative=%v)", elapsed, Target, IndicativeAtivado)
	checkAlvo(t, "TestHybridQuery_Alvo", elapsed, Target, IndicativeAtivado)
}

// prepopulateSymbolsDB insere N rows em code_symbols para medir latência de query.
func prepopulateSymbolsDB(tb testing.TB, database *sql.DB, n int) {
	tb.Helper()
	ctx := context.Background()
	fileID, err := db.UpsertCodeFile(ctx, database, "bench/gen.go", "go",
		"hash-bench", "", int64(n*100), 1700000000)
	if err != nil {
		tb.Fatalf("UpsertCodeFile: %v", err)
	}
	kinds := []string{"function", "method", "class", "struct", "interface", "variable", "constant"}
	for i := 0; i < n; i++ {
		_, err := db.UpsertCodeSymbol(ctx, database, fileID,
			kinds[i%len(kinds)], "Helper",
			"pkg"+strconv.Itoa(i)+".Helper", "", "", i+1, i+1, 0, 0)
		if err != nil {
			tb.Fatalf("UpsertCodeSymbol i=%d: %v", i, err)
		}
	}
	_ = fmt.Sprintf
}
