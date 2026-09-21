//go:build !treesitter

// Stub do backend tree-sitter para builds SEM a build tag `treesitter`.
// Garante que `go build` (padrão) e ambientes sem gcc/CGO continuem compilando.
// O code_pipeline trata ErrTreesitterDisabled como no-op silencioso (CA-03).

package codeast

import (
	"sync"
)

var (
	treesitterLogOnce sync.Once
	treesitterEnabled = false
)

// newBackendParser devolve ErrTreesitterDisabled para qualquer chamada
// (não há backend real implementado neste arquivo).
func newBackendParser() (Parser, error) {
	return nil, ErrTreesitterDisabled
}

// IsTreesitterEnabled devolve false no build sem a tag `treesitter`.
// Usado pelo log de boot (CA-14) e pelo pipeline (CA-03) para skip automático.
func IsTreesitterEnabled() bool {
	return false
}

// LogTreesitterBootStatus imprime "tree-sitter: disabled (code pipeline skipped)"
// exatamente uma vez durante o boot do processo.
func LogTreesitterBootStatus() {
	treesitterLogOnce.Do(func() {
		// Saída controlada pelo chamador via fmt; aqui só sinalizamos.
		treesitterEnabled = false
	})
}
