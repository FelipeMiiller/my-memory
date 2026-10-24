//go:build treesitter

// Backend de code AST do build COM a build tag `treesitter`.
//
// O binding tree-sitter (`github.com/tree-sitter/go-tree-sitter`) exige CGO e
// gcc — e ele NÃO foi injetado: a ADR-047 segue deferred (0 de 3 gatilhos).
//
// Até o binding existir, este arquivo NÃO pode SOMBREAD o motor real. A
// versão anterior devolvia um stub cujo ParseFile devolvia
// ErrTreesitterDisabled — e o pipeline tratava isso como no-op silencioso:
// o build com `-tags treesitter` produzia um code graph VAZIO (1 símbolo,
// 0 arestas) sem nenhum erro, enquanto o build padrão produzia 8 símbolos e
// 4 arestas (defeito D3).
//
// A política adotada (coerente com o deferral da ADR-047):
//
//  1. o binding tree-sitter NÃO está implementado — o log de boot avisa;
//  2. o motor ativo é o backend Go (stdlib), o MESMO do build padrão.
//
// Consequência: o build com `-tags treesitter` produz o mesmo code graph do
// build padrão (mesmos símbolos e arestas) — a tag nunca degrada o grafo.
// Quando o binding existir, este arquivo passa a devolver o parser
// tree-sitter de verdade e IsTreesitterEnabled() volta a true.
//
// Documentação de ativação do binding real (ainda não feita):
//
//  1. go get github.com/tree-sitter/go-tree-sitter
//  2. CGO_ENABLED=1 (Linux/macOS default; Windows requer gcc-mingw)
//  3. go build -tags treesitter ./cmd/mem
//  4. substituir newBackendParser() e IsTreesitterEnabled() abaixo

package codeast

import (
	"fmt"
	"sync"
)

var (
	treesitterLogOnce sync.Once
)

// newBackendParser devolve o backend de code AST do build com a tag
// `treesitter`.
//
// Enquanto o binding real (github.com/tree-sitter/go-tree-sitter) não for
// injetado, devolve o backend Go (stdlib) — o mesmo do build padrão. É o que
// garante que o code graph NUNCA degrade para "vazio sem erro" (D3).
func newBackendParser() (Parser, error) {
	return NewGoParser(), nil
}

// IsTreesitterEnabled devolve false enquanto o binding tree-sitter não
// estiver implementado.
//
// A pergunta é "o backend tree-sitter está ATIVO?" — e a resposta honesta é
// não: o binário foi compilado com a tag, mas o motor ativo é o backend Go
// (stdlib). Devolver true aqui seria mentir: o ParseFile do stub devolvia
// ErrTreesitterDisabled e esvaziava o code graph.
func IsTreesitterEnabled() bool {
	return false
}

// LogTreesitterBootStatus avisa, uma vez por boot (CA-14), que a tag
// `treesitter` foi compilada mas o binding NÃO está implementado, e que o
// motor ativo é o backend Go (stdlib).
//
// A mensagem é honesta e explícita: quem seguir os passos de ativação
// documentados no arquivo tem que saber, na hora, que o tree-sitter não está
// em uso — e que o grafo de código NÃO degrada por causa disso.
func LogTreesitterBootStatus() {
	treesitterLogOnce.Do(func() {
		fmt.Println("code AST: backend go (stdlib); tree-sitter: build tag ativa, mas o binding NAO esta implementado — o motor ativo e o backend Go (stdlib)")
	})
}
