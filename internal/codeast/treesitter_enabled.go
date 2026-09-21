//go:build treesitter

// Stub do backend tree-sitter compilado COM a build tag `treesitter`.
//
// IMPORTANTE: este arquivo é um placeholder consciente. O binding real
// (`github.com/tree-sitter/go-tree-sitter`) requer CGO e gcc —
// quando o toolchain estiver disponível, basta substituir o body de
// newBackendParser() e IsTreesitterEnabled() pela implementação real.
//
// Documentação de ativação (CA-14):
//   1. go get github.com/tree-sitter/go-tree-sitter
//   2. CGO_ENABLED=1 (Linux/macOS default; Windows requer gcc-mingw)
//   3. go build -tags treesitter ./cmd/mem
//
// Enquanto esta stub existir, ParseFile retorna ErrTreesitterDisabled,
// IsTreesitterEnabled() devolve false e LogTreesitterBootStatus registra
// "tree-sitter: enabled (stub)" para deixar claro que o backend real ainda
// não foi injetado neste binário.
//
// O code_pipeline trata ErrTreesitterDisabled como no-op silencioso (CA-03).

package codeast

import (
	"fmt"
	"sync"
)

var (
	treesitterLogOnce sync.Once
)

// treesitterBackend é o placeholder real para quando gcc + dependência
// forem adicionados. Por ora, devolve ErrTreesitterDisabled em ParseFile.
type treesitterBackend struct {
	languages []string
}

// newBackendParser constrói o backend tree-sitter.
//
// Stub: enquanto o binding real (github.com/tree-sitter/go-tree-sitter)
// não for injetado aqui, devolve um backend funcional apenas com metadata
// (Languages, Backend) e ParseFile devolve ErrTreesitterDisabled.
func newBackendParser() (Parser, error) {
	return &treesitterBackend{
		languages: languagesToNames(DefaultLanguageTable()),
	}, nil
}

// ParseFile é stub. Substituir pela chamada real
//
//	parser := tree_sitter.NewParser(tree_sitter.NewLanguage(tree_sitter.Go))
//	tree := parser.Parse(content, nil)
//	// ... walk tree, extract symbols por kind ...
//
// quando o binding CGO estiver disponível.
func (t *treesitterBackend) ParseFile(path string, content []byte) (*FileResult, error) {
	return nil, fmt.Errorf("%w (stub treesitter_enabled.go; real backend requires CGO toolchain)", ErrTreesitterDisabled)
}

// Languages devolve a lista de linguagens do top-10 (ADR-047).
func (t *treesitterBackend) Languages() []string {
	out := make([]string, len(t.languages))
	copy(out, t.languages)
	return out
}

// Backend devolve "treesitter" para indicar qual implementação atenderia.
func (t *treesitterBackend) Backend() string {
	return "treesitter"
}

// IsTreesitterEnabled devolve true no build com a tag `treesitter` mesmo
// que o backend real ainda não tenha sido injetado (stub retorna erro em
// ParseFile). Marca a intenção de build e isola o branch no-op (CA-03).
func IsTreesitterEnabled() bool {
	return true
}

// LogTreesitterBootStatus imprime a linha de boot exatamente uma vez.
func LogTreesitterBootStatus() {
	treesitterLogOnce.Do(func() {
		fmt.Println("tree-sitter: enabled (stub backend — bind to github.com/tree-sitter/go-tree-sitter when CGO toolchain available)")
	})
}

// languagesToNames devolve a lista de nomes canônicos (lower-case).
func languagesToNames(defs []LanguageDef) []string {
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Name)
	}
	return out
}
