//go:build !onnx

package main

import (
	"errors"

	"github.com/FelipeMiiller/my-memory/internal/config"
	"github.com/FelipeMiiller/my-memory/internal/embedder"
)

// newBuiltinEmbedder só existe quando o binário foi compilado com a tag
// `onnx`. Sem ela o provider embutido não está disponível e a indexação cai
// para FTS5 léxico puro.
//
// Compile com `go build -tags onnx` para ter busca semântica.
func newBuiltinEmbedder(cfg *config.Config) (embedder.Embedder, error) {
	return nil, errors.New("provider builtin indisponível: este binário foi compilado sem a tag `onnx` (go build -tags onnx)")
}
