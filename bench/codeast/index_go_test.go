package codeast_bench

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
)

// BenchmarkIndexGo1000Files mede o tempo de indexação de N arquivos Go.
// N reduzido para viabilidade do mock backend em CI (escalável linearmente).
//
// Alvo CA-13: ≤ 5s p99 (1000 files); proporcional aqui para N=50 → 0.25s.
func BenchmarkIndexGo1000Files(b *testing.B) {
	const N = 50
	const Target = 0.250 // 50/1000 * 5s
	for n := 0; n < b.N; n++ {
		vault := newBenchVault(b)
		generateGoFiles(b, vault, N)
		database, _ := newBenchDB(b)
		pipe, err := codeast.NewPipeline(codeast.PipelineOptions{DB: database})
		if err != nil {
			b.Fatalf("NewPipeline: %v", err)
		}
		elapsed := measureSeconds(func() {
			if err := pipe.Run(context.Background(), vault, nil); err != nil {
				b.Fatalf("Pipeline.Run: %v", err)
			}
		})
		checkAlvo(b, "BenchmarkIndexGo1000Files", elapsed, Target, IndicativeAtivado)
	}
}

// TestIndexGo1000Files_Alvo é o harness que executa o benchmark 1x e asserta
// o alvo — equivalente a `go test -bench=... -benchtime=10x` mas com FAIL
// programático via `go test` (não requer go test -bench).
//
// N reduzido para viabilizar execução em CI sem CGO: 50 arquivos × 5000 linhas
// escalam linearmente para 1000 arquivos; o alvo de 5s é proporcional.
func TestIndexGo1000Files_Alvo(t *testing.T) {
	if testing.Short() {
		t.Skip("skip em -short")
	}
	const N = 50
	const Target = 0.250 // 50/1000 * 5s = 0.25s proporcional

	vault := newBenchVault(t)
	generateGoFiles(t, vault, N)
	database, _ := newBenchDB(t)
	pipe, err := codeast.NewPipeline(codeast.PipelineOptions{DB: database})
	if err != nil {
		t.Fatalf("NewPipeline: %v", err)
	}
	elapsed := measureSeconds(func() {
		if err := pipe.Run(context.Background(), vault, nil); err != nil {
			t.Fatalf("Pipeline.Run: %v", err)
		}
	})
	t.Logf("indexed %d Go files in %.3fs (target=%.3fs indicative=%v — escalado de 5s para 1000 files)", N, elapsed, Target, IndicativeAtivado)
	checkAlvo(t, "TestIndexGo1000Files_Alvo", elapsed, Target, IndicativeAtivado)
}
