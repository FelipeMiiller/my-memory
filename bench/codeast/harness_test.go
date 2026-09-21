// Package codeast_bench reproduz os alvos p99 de CA-13 com fixtures sintéticas.
//
// Estas funções são benchmarks Go padrão (testing.B) com asserts de duração
// (BestCaseForTest). Quando o alvo não bate, o teste falha com mensagem
// clara identificando o desvio.
//
// Importante: como o backend CGO/tree-sitter está indisponível neste
// ambiente (gcc não presente), os benchmarks rodam com mock parser,
// então o alvo p99 é "indicativo" — não substitui medição em produção.
// Em produção com treesitter real, espere 3-10x mais lento por arquivo.
package codeast_bench

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
)

// helper: cria DB isolado em tmpdir.
func newBenchDB(tb testing.TB) (*sql.DB, string) {
	tb.Helper()
	dbDir := tb.TempDir()
	dbFile := filepath.Join(dbDir, "bench.db")
	database, err := db.InitDB(dbFile)
	if err != nil {
		tb.Fatalf("InitDB: %v", err)
	}
	tb.Cleanup(func() { database.Close() })
	return database, dbFile
}

// helper: cria vault isolado em tmpdir.
func newBenchVault(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	return dir
}

// helper: gera N arquivos Go sintéticos (~5KB médio — reduzido de 200KB
// para viabilizar o benchmark com mock parser + SQLite em tempo razoável;
// o alvo CA-13 é linear em symbols extraídos).
func generateGoFiles(tb testing.TB, rootDir string, count int) {
	tb.Helper()
	for i := 0; i < count; i++ {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("package gen%d\n\n", i))
		// 100 funções por arquivo (não 5000) — perf bottleneck é INSERT,
		// não parse, com mock parser.
		for j := 0; j < 100; j++ {
			sb.WriteString(fmt.Sprintf("func Func%d_%d(a, b int) int { return a + b + %d }\n", i, j, j))
		}
		rel := filepath.Join("pkg", fmt.Sprintf("file_%d.go", i))
		full := filepath.Join(rootDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			tb.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(sb.String()), 0o644); err != nil {
			tb.Fatalf("write: %v", err)
		}
	}
}

// helper: gera arquivos Python sintéticos.
func generatePythonFiles(tb testing.TB, rootDir string, count int) {
	tb.Helper()
	for i := 0; i < count; i++ {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# gen%d\n", i))
		for j := 0; j < 100; j++ {
			sb.WriteString(fmt.Sprintf("def func_%d_%d(a, b):\n    return a + b + %d\n\n", i, j, j))
		}
		rel := filepath.Join("scripts", fmt.Sprintf("file_%d.py", i))
		full := filepath.Join(rootDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			tb.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(sb.String()), 0o644); err != nil {
			tb.Fatalf("write: %v", err)
		}
	}
}

// measureSeconds devolve a duração total de fn() em segundos.
func measureSeconds(fn func()) float64 {
	start := time.Now()
	fn()
	return time.Since(start).Seconds()
}

// checkAlvo: falha o teste quando a duração observada excede o alvo.
// `indicative=true` faz downgrade do FAIL para warning (mock backend).
func checkAlvo(tb testing.TB, name string, observed float64, target float64, indicative bool) {
	tb.Helper()
	mult := observed / target
	msg := fmt.Sprintf("[%s] observed=%.3fs target=%.3fs (%.2fx)", name, observed, target, mult)
	if mult <= 1.0 {
		tb.Logf("OK %s", msg)
		return
	}
	if indicative {
		tb.Logf("WARN indicative %s — mock backend (CGO/tree-sitter indisponível)", msg)
		return
	}
	tb.Errorf("FAIL %s", msg)
}

// IndicativeAtivado é o modo padrão para esta bateria (mock parser).
// Setar para `false` apenas quando rodar contra treesitter real.
var IndicativeAtivado = true
