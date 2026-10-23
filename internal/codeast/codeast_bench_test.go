package codeast

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
)

// BenchmarkListCodeFiles mede o caminho quente do code stage: a seleção de
// arquivos pelo escopo de código. O escopo é avaliado por arquivo (glob por
// arquivo), então o custo é proporcional ao número de arquivos do repositório
// — e é o que roda a cada `mem index` com o code stage ligado.
func BenchmarkListCodeFiles(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 200; i++ {
		dir := filepath.Join(root, "pkg", "sub")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		name := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		if err := os.WriteFile(name, []byte("package x\nfunc F(){}\n"), 0o644); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "nota.md"), []byte("# Nota\n"), 0o644); err != nil {
			b.Fatal(err)
		}
	}

	cfg := config.DefaultConfig()
	cfg.Code = config.CodeScopeConfig{Include: []string{"**/*.go"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ListCodeFiles(root, &cfg, nil)
		if err != nil {
			b.Fatalf("ListCodeFiles falhou: %v", err)
		}
		if len(files) != 200 {
			b.Fatalf("esperava 200 arquivos, obteve %d", len(files))
		}
	}
}
