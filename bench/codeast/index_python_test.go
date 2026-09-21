package codeast_bench

import (
	"context"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/codeast"
)

// BenchmarkIndexPython100Files mede o tempo de indexação de N arquivos Python.
// N reduzido para viabilidade do mock backend; alvo escalado linearmente.
func BenchmarkIndexPython100Files(b *testing.B) {
	const N = 25
	const Target = 0.500 // 25/100 * 2s
	for n := 0; n < b.N; n++ {
		vault := newBenchVault(b)
		generatePythonFiles(b, vault, N)
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
		checkAlvo(b, "BenchmarkIndexPython100Files", elapsed, Target, IndicativeAtivado)
	}
}

// TestIndexPython100Files_Alvo é o harness 1x com assert programático.
// N reduzido para viabilidade do mock backend.
func TestIndexPython100Files_Alvo(t *testing.T) {
	if testing.Short() {
		t.Skip("skip em -short")
	}
	const N = 25
	const Target = 0.500 // 25/100 * 2s

	vault := newBenchVault(t)
	generatePythonFiles(t, vault, N)
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
	t.Logf("indexed %d Python files in %.3fs (target=%.3fs indicative=%v — escalado de 2s para 100 files)", N, elapsed, Target, IndicativeAtivado)
	checkAlvo(t, "TestIndexPython100Files_Alvo", elapsed, Target, IndicativeAtivado)
}
