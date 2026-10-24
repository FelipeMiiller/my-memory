//go:build !treesitter

// Backend de code AST do build PADRÃO (sem a build tag `treesitter`).
//
// O build default NÃO é mais um no-op: o motor é o backend Go em Go puro
// (go/parser + go/ast + go/token da stdlib — sem CGO, sem build tag e sem
// dependência nova). O caminho tree-sitter continua isolado atrás da tag
// `treesitter` e tem precedência quando compilado com ela.
//
// A ADR-047 NÃO é tocada aqui: o plano de binding tree-sitter segue deferred
// (0/3 gatilhos) e o MockParser permanece como fallback determinístico para as
// linguagens que ainda não têm backend AST.

package codeast

import (
	"fmt"
	"sync"
)

var (
	treesitterLogOnce sync.Once
	treesitterEnabled = false
)

// newBackendParser devolve o backend de code AST do build padrão: o backend
// Go (stdlib). O MockParser deixa de ser o caminho de produção.
//
// Este é o ponto de extensão real do motor (ADR-061 §Fase 2): quando o
// binding tree-sitter existir, a substituição acontece em
// treesitter_enabled.go, sem tocar no pipeline.
func newBackendParser() (Parser, error) {
	return NewGoParser(), nil
}

// IsTreesitterEnabled devolve false no build sem a tag `treesitter`.
// Usado pelo log de boot (CA-14) e pelo pipeline (CA-03) para skip automático.
func IsTreesitterEnabled() bool {
	return false
}

// LogTreesitterBootStatus imprime o status do motor de code AST exatamente uma
// vez durante o boot do processo (CA-14).
//
// A mensagem é honesta: o code pipeline NÃO é mais skipped no build default —
// o motor ativo é o backend Go da stdlib. O tree-sitter continua desligado
// (build sem a tag), mas isso já não significa "code pipeline skipped".
func LogTreesitterBootStatus() {
	treesitterLogOnce.Do(func() {
		fmt.Println("code AST: backend go (stdlib); tree-sitter: disabled")
		treesitterEnabled = false
	})
}
