//go:build onnx

package main

import (
	"path/filepath"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// defaultModelsDir é onde `mem embed download` deixa os artefatos.
func defaultModelsDir() string {
	return embedder.DefaultModelsDir()
}

// defaultORTLibPath é onde `mem embed download` deixa a biblioteca do
// ONNX Runtime.
//
// O nome do arquivo não é o mesmo em todo sistema (onnxruntime.dll no
// Windows, libonnxruntime.so no Linux): o runtime é carregado por dlopen no
// caminho exato.
func defaultORTLibPath() string {
	return embedder.DefaultORTLibPath()
}

// newBuiltinEmbedder constrói o provider embutido (ADR-035). Só existe na
// build com a tag `onnx`.
func newBuiltinEmbedder(cfg *config.Config) (embedder.Embedder, error) {
	bc := embedder.BuiltinConfig{
		ModelPath:    filepath.Join(defaultModelsDir(), "model_int8.onnx"),
		TokenizerDir: defaultModelsDir(),
		ORTLibPath:   defaultORTLibPath(),
	}
	if cfg != nil {
		// Campos vêm do config quando declarados; senão os defaults acima.
		_ = cfg // reservado para wiring da T10 (BuiltinConfig no EmbeddingConfig)
	}
	return embedder.NewBuiltinClient(bc)
}
