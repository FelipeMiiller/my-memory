//go:build onnx

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
	"github.com/FelipeMiiller/my-memory/internal/store"
	"github.com/FelipeMiiller/my-memory/internal/turboquant"
)

// TestBuscaComProviderRealDeclaraFonteVector prova o AC4 de ponta a ponta
// com o provider EMBUTIDO DE VERDADE: o binário com a tag `onnx` indexa um
// vault, o provider real gera o vetor de cada chunk, e a busca reporta a
// fonte `vector` nas fontes RRF.
//
// O contrato da AC4 é esse: "`mem search` contra um vault vetorizado reporta
// a fonte `vector`". Sem provider a resposta é `[fts:1]` — é a diferença
// entre a busca que funciona e a que finge que funciona.
//
// O arquivo só compila na build com a tag `onnx`. Quando os artefatos do
// provider não estão no disco (a CI não baixa 134 MB), o teste PULA com
// motivo explícito: não falha e não finge que provou nada.
func TestBuscaComProviderRealDeclaraFonteVector(t *testing.T) {
	cfg := config.DefaultConfig()

	emb, err := newBuiltinEmbedder(&cfg)
	if err != nil {
		t.Skipf("provider embutido indisponível neste ambiente (%v) — o teste precisa do modelo e da biblioteca do ONNX Runtime", err)
	}
	if closer, ok := emb.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closer.Close() })
	}

	vault := t.TempDir()
	for _, nome := range []string{"arquitetura-do-sistema.md", "operacoes-do-cofre.md"} {
		conteudo := "# " + nome + "\n\nA arquitetura do sistema e as operacoes do cofre central de memoria.\n"
		if err := os.WriteFile(filepath.Join(vault, nome), []byte(conteudo), 0o644); err != nil {
			t.Fatalf("criando nota %s: %v", nome, err)
		}
	}
	dbPath := filepath.Join(t.TempDir(), "memory.db")

	// Indexa pelo comando de verdade: o mesmo caminho do `mem index`, com o
	// provider resolvido pelo próprio binário.
	if err := runIndexCLI(context.Background(), "repo_teste", []string{
		"--storage=sqlite",
		"--db", dbPath,
		"--repo", "repo_teste",
		vault,
	}); err != nil {
		t.Fatalf("runIndexCLI: %v", err)
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB falhou: %v", err)
	}
	defer database.Close()

	tq := turboquant.NewQuantizer(embedder.EmbeddingDim)

	var out string
	codigo := 0
	out = captureStdout(t, func() {
		codigo = runSearchSQLite(context.Background(), database, emb, tq, "arquitetura", "hybrid", true, 10, 60,
			store.DefaultDecayOptions(), store.SearchOptions{})
	})

	if codigo != 0 {
		t.Fatalf("a busca híbrida com o provider real deveria responder (código %d):\n%s", codigo, out)
	}
	if !strings.Contains(out, "vector:") {
		t.Errorf("a saída não declara a fonte vetorial do RRF com o provider real:\n%s", out)
	}
	if strings.Contains(out, "LEXICAL") {
		t.Errorf("a saída declarou fallback léxico com o provider real disponível:\n%s", out)
	}
	// Não-vacuous: o documento tem de estar no resultado.
	if !strings.Contains(out, "arquitetura-do-sistema") {
		t.Errorf("a busca com o provider real não trouxe o documento indexado:\n%s", out)
	}
}
