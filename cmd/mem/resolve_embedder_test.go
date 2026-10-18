package main

import (
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// TestResolveEmbedder_ConfigDimensionNaoVenceSchema trava que a dimensão do
// schema é a única que vale.
//
// Um config legado — ou o template antigo do `mem init` — declarando
// `dimension: 768` não pode virar o `dim` devolvido: a coluna é `vector(384)`
// e o provider embutido devolve 384. Honrar o config produzia um quantizador
// TurboQuant do tamanho errado, que quebra na hora de ler o vetor de volta.
//
// O provider em si é opcional (build sem a tag `onnx` cai para FTS5 léxico),
// então o teste fixa a DIMENSÃO e não a existência do embedder.
func TestResolveEmbedder_ConfigDimensionNaoVenceSchema(t *testing.T) {
	// O env tem precedência no caminho antigo; zera para isolar o efeito do
	// config e para o teste não depender do ambiente de quem roda `go test`.
	t.Setenv("MY_MEMORY_EMBED_DIM", "")

	cfg := &config.Config{}
	cfg.Embedding.Provider = "ollama"
	cfg.Embedding.Model = "nomic-embed-text"
	cfg.Embedding.URL = "http://localhost:11434"
	cfg.Embedding.Dimension = 768

	_, tq, dim := resolveEmbedder(cfg)

	if dim != embedder.EmbeddingDim {
		t.Errorf("esperava dim %d (EmbeddingDim), obteve %d", embedder.EmbeddingDim, dim)
	}
	if tq == nil {
		t.Fatalf("resolveEmbedder devolveu quantizador nil")
	}
}

// TestResolveEmbedder_ConfigVazioUsaEmbeddingDim cobre o caso sem config:
// nil não pode estourar e o dim tem de ser o do schema.
func TestResolveEmbedder_ConfigVazioUsaEmbeddingDim(t *testing.T) {
	t.Setenv("MY_MEMORY_EMBED_DIM", "")

	_, tq, dim := resolveEmbedder(nil)

	if dim != embedder.EmbeddingDim {
		t.Errorf("esperava dim %d (EmbeddingDim), obteve %d", embedder.EmbeddingDim, dim)
	}
	if tq == nil {
		t.Fatalf("resolveEmbedder devolveu quantizador nil")
	}
}
