//go:build onnx

package embedder

import (
	"math"
	"math/rand"
)

// schemaDim é a dimensão que o schema do banco exige (embedding float[768]).
const schemaDim = 768

// projectKind identifica a estratégia de projeção 384d → 768d.
type projectKind int

const (
	projectJL      projectKind = iota // Johnson-Lindenstrauss
	projectPadZero                    // padding com zeros
)

// ProjectJL projeta um vetor nativo para a dimensão do schema usando um
// projetao aleatoria de Johnson-Lindenstrauss.
//
// A ideia: um mapa linear aleatório R preserva distâncias até distorção
// O(log n / sqrt(k)) — ou seja, o cosseno é praticamente mantido. E
// determinístico quando a seed é fixa, o que mantém o cache SHA-256 do
// índice consistente.
func ProjectJL(v []float32, outDim int, seed int64) []float32 {
	in := len(v)
	if in == 0 || outDim <= in {
		cp := make([]float32, in)
		copy(cp, v)
		return cp
	}
	rng := rand.New(rand.NewSource(seed))
	scale := float32(1.0 / math.Sqrt(float64(in)))

	// R é in×out com N(0,1). Gerado e aplicado em bloco para não segurar
	// in*out floats na memória de uma vez.
	out := make([]float32, outDim)
	col := make([]float32, in)
	for j := 0; j < outDim; j++ {
		for i := range col {
			col[i] = float32(rng.NormFloat64())
		}
		var acc float64
		for i := 0; i < in; i++ {
			acc += float64(v[i]) * float64(col[i])
		}
		out[j] = float32(acc) * scale
	}
	l2Normalize(out)
	return out
}

// ProjectPadZero preenche com zeros até a dimensão do schema.
//
// Só existe para a medição: não preserva similaridade alguma, já que metade
// das dimensões ficam zeradas e o cosseno só enxerga o subspace original
// (com norma reduzida). Serve de contraponto ao JL.
func ProjectPadZero(v []float32, outDim int) []float32 {
	out := make([]float32, outDim)
	copy(out, v)
	return out
}

// LexicalSimilarity é um proxy de similaridade puramente léxica (Jaccard
// sobre tokens), que é o que um FTS5/BM25 consegue enxergar: sobreposição
// de termos, zero semântica.
//
// Existe para dar um piso honesto ao benchmark — um provider vetorial só
// vale se o gap sin-vs-dis for maior que o que a sobreposição de palavras já
// entrega sozinha.
func LexicalSimilarity(a, b string) float64 {
	ta := tokenSet(a)
	tb := tokenSet(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for w := range ta {
		if tb[w] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
