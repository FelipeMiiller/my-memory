package codeast

import "testing"

// TestHashSymbolsStructural_IgnoraPosicaoEVeAssinatura fixa o contrato do
// hash ESTRUTURAL (defeito D6):
//
//   - insensível à POSIÇÃO (comentário/whitespace que desloca linhas);
//   - sensível à ASSINATURA (parâmetros, tipo de retorno).
//
// E o hash de CACHE (HashSymbols) continua sensível à posição — são DOIS
// hashes diferentes (o de cache decide reindexação; o estrutural decide
// drift).
func TestHashSymbolsStructural_IgnoraPosicaoEVeAssinatura(t *testing.T) {
	base := []Symbol{
		{Kind: KindFunction, Name: "Run", QualifiedName: "pkg.Run", Signature: "func Run(a int) error", StartLine: 10, EndLine: 12},
		{Kind: KindStruct, Name: "Server", QualifiedName: "pkg.Server", Signature: "type Server struct", StartLine: 14, EndLine: 16},
	}
	baseHash := HashSymbolsStructural(base)
	if baseHash == "" {
		t.Fatal("o hash estrutural não pode ser vazio")
	}

	// 1. Deslocar as linhas (comentário/whitespace) NÃO muda o hash
	//    estrutural.
	shifted := []Symbol{
		{Kind: KindFunction, Name: "Run", QualifiedName: "pkg.Run", Signature: "func Run(a int) error", StartLine: 25, EndLine: 27},
		{Kind: KindStruct, Name: "Server", QualifiedName: "pkg.Server", Signature: "type Server struct", StartLine: 29, EndLine: 31},
	}
	if got := HashSymbolsStructural(shifted); got != baseHash {
		t.Errorf("o hash estrutural mudou com o deslocamento de linhas: %s vs %s", got, baseHash)
	}

	// 2. Mudar a ASSINATURA muda o hash estrutural.
	changed := []Symbol{
		{Kind: KindFunction, Name: "Run", QualifiedName: "pkg.Run", Signature: "func Run(a int, b string) error", StartLine: 10, EndLine: 12},
		{Kind: KindStruct, Name: "Server", QualifiedName: "pkg.Server", Signature: "type Server struct", StartLine: 14, EndLine: 16},
	}
	if got := HashSymbolsStructural(changed); got == baseHash {
		t.Error("o hash estrutural não mudou com a assinatura — o drift ficaria cego")
	}

	// 3. O hash de CACHE é sensível à posição (são hashes diferentes).
	if HashSymbols(base) == HashSymbols(shifted) {
		t.Error("o hash de cache (HashSymbols) deve incluir a posição")
	}
	if HashSymbols(base) == HashSymbolsStructural(base) {
		t.Error("o hash de cache e o estrutural não podem ser o mesmo valor")
	}

	// 4. A ordem de entrada não importa.
	reversed := []Symbol{base[1], base[0]}
	if HashSymbolsStructural(reversed) != baseHash {
		t.Error("o hash estrutural deve ser independente da ordem de entrada")
	}
}
