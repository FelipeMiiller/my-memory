//go:build onnx

package main

import (
	"os"
	"path/filepath"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// defaultModelsDir é onde `mem embed download` deixa os artefatos.
func defaultModelsDir() string {
	if d := os.Getenv("MY_MEMORY_MINILM_MODEL_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".memory/models/paraphrase-multilingual-MiniLM-L12-v2"
	}
	return filepath.Join(home, ".memory", "models", "paraphrase-multilingual-MiniLM-L12-v2")
}

func defaultORTLibPath() string {
	if p := os.Getenv("MY_MEMORY_ORT_LIB"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Linux/macOS ficam a cargo de LD_LIBRARY_PATH / DYLD_LIBRARY_PATH.
	return filepath.Join(home, ".memory", "models", "onnxruntime", "onnxruntime.dll")
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
